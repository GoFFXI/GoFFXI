package view

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/GoFFXI/login-server/internal/config"
	"github.com/GoFFXI/login-server/internal/database/migrations"
	"github.com/GoFFXI/login-server/internal/server"
)

type ViewServer struct {
	*server.Server
}

func Run(cfg *config.Config, logger *slog.Logger) error {
	// setup wait group for goroutines
	var wg sync.WaitGroup

	// create a context for graceful shutdown
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	baseServer, err := server.NewServer(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create base server: %w", err)
	}

	viewServer := &ViewServer{
		Server: baseServer,
	}

	// connect to NATS server
	if err = viewServer.CreateNATSConnection(); err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// connect to database
	if err = viewServer.CreateDBConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// run database migrations
	if err = migrations.Migrate(ctx, viewServer.DB().BunDB()); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	//nolint:errcheck // socket will be closed on shutdown
	defer viewServer.Socket().Close()

	// start connection processor goroutine
	wg.Add(1)
	go viewServer.ProcessConnections(ctx, &wg, viewServer.handleConnection)

	// start accepting connections
	wg.Add(1)
	go viewServer.AcceptConnections(ctx, &wg)

	// wait for shutdown signal
	return viewServer.WaitForShutdown(cancelCtx, &wg)
}

func (s ViewServer) handleConnection(ctx context.Context, conn net.Conn) {
	logger := s.Logger().With("client", conn.RemoteAddr().String())
	logger.Info("new client connection established")

	// create a new session for this connection
	sessionCtx := sessionContext{
		ctx:    ctx,
		conn:   conn,
		server: &s,
		logger: logger,
	}
	defer sessionCtx.Close()

	// connection handling loop
	for {
		// make sure we exit if the server is shutting down
		select {
		case <-ctx.Done():
			return
		default:
		}

		// buffer for reading data
		buffer := make([]byte, 4096)

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

		if shouldExit := s.parseIncomingRequest(&sessionCtx, buffer[:length]); shouldExit {
			break
		}
	}
}

func (s ViewServer) parseIncomingRequest(sessionCtx *sessionContext, request []byte) bool {
	header, err := NewRequestHeader(request)
	if err != nil {
		sessionCtx.logger.Error("failed to parse request header", "error", err)
		return true
	}

	// attempt to lookup the account session
	sessionKey := string(header.Identifier[:])
	sessionCtx.logger.Info("looking up session", "sessionKey", sessionKey, "opCode", header.Command)
	accountSession, err := s.DB().GetAccountSessionBySessionKey(sessionCtx.ctx, sessionKey)
	if err != nil {
		// this shouldn't happen normally, log and close the connection
		sessionCtx.logger.Error("failed to lookup account session", "session_key", header.Identifier, "error", err)
		return true
	}

	// make sure this session context has subscriptions set up
	// this should only be done once per session
	if err = sessionCtx.SetupSubscriptions(accountSession.SessionKey); err != nil {
		sessionCtx.logger.Error("failed to setup subscriptions", "error", err)
		return true
	}

	// now, handle the request based on the command
	switch header.Command {
	case CommandRequestLobbyLogin:
		s.handleRequestLobbyLogin(sessionCtx, request)
	case CommandRequestGetCharacter:
		s.handleRequestGetCharacter(sessionCtx, &accountSession, request)
	case CommandRequestQueryWorldList:
		s.handleRequestWorldList(sessionCtx, request)
	}

	return false
}
