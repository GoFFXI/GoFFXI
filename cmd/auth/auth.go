package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/GoFFXI/login-server/internal/config"
	"github.com/GoFFXI/login-server/internal/server"
	"github.com/GoFFXI/login-server/internal/tools"
	"github.com/nats-io/nats.go"
)

const (
	OpResponseFail          = 0x00
	OpResponseSuccess       = 0x01
	OpErrorOccurred         = 0x02
	OpRequestAttemptLogin   = 0x10
	OpRequestCreateAccount  = 0x20
	OpRequestChangePassword = 0x30

	OpErrorInvalidClientVersion = 0x0B
)

type AuthServer struct {
	*server.Server
}

func Run(cfg *config.Config, logger *slog.Logger, nc *nats.Conn) error {
	// setup wait group for goroutines
	var wg sync.WaitGroup

	// create a context for graceful shutdown
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	baseServer, err := server.NewServer(ctx, cfg, logger, nc)
	if err != nil {
		return fmt.Errorf("failed to create base server: %w", err)
	}

	authServer := &AuthServer{
		Server: baseServer,
	}

	//nolint:errcheck // socket will be closed on shutdown
	defer authServer.Socket().Close()

	// start connection processor goroutine
	wg.Add(1)
	go authServer.ProcessConnections(ctx, &wg, authServer.handleConnection)

	// start accepting connections
	wg.Add(1)
	go authServer.AcceptConnections(ctx, &wg)

	// wait for shutdown signal
	return authServer.WaitForShutdown(cancelCtx, &wg)
}

func (s *AuthServer) handleConnection(ctx context.Context, conn net.Conn) {
	//nolint:errcheck // closing connection
	defer conn.Close()

	logger := s.Logger().With("client", conn.RemoteAddr().String())
	logger.Info("processing connection")

	// set read/write timeout for the connection
	_ = conn.SetDeadline(time.Now().Add(time.Duration(s.Config().ServerReadTimeoutSeconds) * time.Second))

	// connection handling loop
	for {
		// make sure we exit if the server is shutting down
		select {
		case <-ctx.Done():
			return
		default:
		}

		// read data from client
		buffer := make([]byte, 1024)
		length, err := conn.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.Info("client disconnected")
				break
			} else if errors.Is(err, net.ErrClosed) {
				break
			}

			logger.Error("error reading from connection", "error", err)
			break
		}

		// parse the expected received data
		request := buffer[:length]
		username := tools.BytesToString(request, 0x09, 16)
		password := tools.BytesToString(request, 0x19, 32)
		clientVersion := tools.BytesToString(request, 0x61, 5)
		opCode := tools.GetIntFromByteBuffer(request, 0x39)

		// handle client version enforcement
		logger.Debug("detected client version", "version", clientVersion)
		if !s.clientVersionIsValid(clientVersion) {
			logger.Error("client version mismatch - disconnecting", "got", clientVersion)
			_, _ = conn.Write([]byte{OpErrorInvalidClientVersion})
			_ = conn.Close()
			return
		}

		switch opCode {
		case OpRequestAttemptLogin:
			s.opAttemptLogin(logger, username, password)
		case OpRequestCreateAccount:
			s.opCreateAccount(logger, username, password)
		case OpRequestChangePassword:
			s.opChangePassword(logger, username, password, buffer)
		}
	}
}

func (s *AuthServer) clientVersionIsValid(clientVersion string) bool {
	if s.Config().XIClientEnforceVersion == 1 {
		// enforce exact version match
		if !tools.VersionIsEqualTo(clientVersion, s.Config().XILoaderVersion) {
			return false
		}
	} else if s.Config().XIClientEnforceVersion == 2 {
		// enforce minimum version
		clientMajor, clientMinor, clientPatch := tools.GetVersionsFromString(clientVersion)
		minMajor, minMinor, minPatch := tools.GetVersionsFromString(s.Config().XILoaderVersion)

		if (clientMajor < minMajor) ||
			(clientMajor == minMajor && clientMinor < minMinor) ||
			(clientMajor == minMajor && clientMinor == minMinor && clientPatch < minPatch) {
			return false
		}
	}

	return true
}
