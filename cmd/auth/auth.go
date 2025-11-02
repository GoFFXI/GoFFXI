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
	"github.com/GoFFXI/login-server/internal/database/migrations"
	"github.com/GoFFXI/login-server/internal/server"
	"github.com/GoFFXI/login-server/internal/tools"
)

const (
	ResponseFail          = 0x00
	ResponseSuccess       = 0x01
	ResponseErrorOccurred = 0x02

	RequestAttemptLogin   = 0x10
	RequestCreateAccount  = 0x20
	RequestChangePassword = 0x30

	ErrorInvalidClientVersion = 0x0B
)

type AuthServer struct {
	*server.Server
}

func Run(cfg *config.Config, logger *slog.Logger) error {
	// setup wait group for goroutines
	var wg sync.WaitGroup

	// create a context for graceful shutdown
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	// setup new AuthServer
	baseServer, err := server.NewServer(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create base server: %w", err)
	}

	authServer := &AuthServer{
		Server: baseServer,
	}

	// connect to NATS server
	if err = authServer.CreateNATSConnection(); err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// connect to database
	if err = authServer.CreateDBConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// run database migrations
	if err = migrations.Migrate(ctx, authServer.DB().BunDB()); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
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

	// buffer for reading data
	buffer := make([]byte, 4096)

	// connection handling loop
	for {
		// make sure we exit if the server is shutting down
		select {
		case <-ctx.Done():
			return
		default:
		}

		// read data from client
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
			_, _ = conn.Write([]byte{ErrorInvalidClientVersion})
			_ = conn.Close()
			return
		}

		switch opCode {
		case RequestAttemptLogin:
			s.opAttemptLogin(ctx, conn, username, password)
		case RequestCreateAccount:
			s.opCreateAccount(ctx, conn, username, password)
		case RequestChangePassword:
			s.opChangePassword(ctx, conn, username, password, buffer)
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
