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
	logger := s.Logger().With("client", conn.RemoteAddr().String())
	logger.Info("processing connection")

	//nolint:errcheck // connection will be closed at the end of this function
	defer conn.Close()

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

		if shouldExit := s.parseIncomingRequest(ctx, logger, conn, buffer[:length]); shouldExit {
			break
		}
	}
}

func (s *AuthServer) parseIncomingRequest(ctx context.Context, logger *slog.Logger, conn net.Conn, request []byte) bool {
	// parse the auth request header
	header, err := NewRequestHeader(request)
	if err != nil {
		logger.Error("failed to parse auth request header", "error", err)
		return true
	}

	// handle client version enforcement
	logger.Debug("detected client version", "version", header.ClientVersion)
	if !s.clientVersionIsValid(header.ClientVersion) {
		logger.Error("client version mismatch - disconnecting", "got", header.ClientVersion)
		_, _ = conn.Write([]byte{ErrorInvalidClientVersion})
		return true
	}

	switch header.Command {
	case RequestAttemptLogin:
		return s.handleRequestAttemptLogin(ctx, conn, header.Username, header.Password)
	case RequestCreateAccount:
		s.handleRequestCreateAccount(ctx, conn, header.Username, header.Password)
	case RequestChangePassword:
		s.handleRequestChangePassword(ctx, conn, header.Username, header.Password, request)
	}

	return false
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
