package data

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/GoFFXI/GoFFXI/internal/config"
	"github.com/GoFFXI/GoFFXI/internal/database/migrations"
	"github.com/GoFFXI/GoFFXI/internal/lobby/packets"
	"github.com/GoFFXI/GoFFXI/internal/server"
)

const (
	CommandRequestKeepXILoaderSpinning = 0xFE
)

type DataServer struct {
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

	dataServer := &DataServer{
		Server: baseServer,
	}

	// connect to NATS server
	if err = dataServer.CreateNATSConnection(); err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// connect to database
	if err = dataServer.CreateDBConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// run database migrations
	if err = migrations.Migrate(ctx, dataServer.DB().BunDB()); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	//nolint:errcheck // socket will be closed on shutdown
	defer dataServer.Socket().Close()

	// start connection processor goroutine
	wg.Add(1)
	go dataServer.ProcessConnections(ctx, &wg, dataServer.handleConnection)

	// start accepting connections
	wg.Add(1)
	go dataServer.AcceptConnections(ctx, &wg)

	// wait for shutdown signal
	return dataServer.WaitForShutdown(cancelCtx, &wg)
}

func (s DataServer) handleConnection(ctx context.Context, conn net.Conn) {
	logger := s.Logger().With("client", conn.RemoteAddr().String())
	logger.Info("processing connection")

	// create a new session for this connection
	sessionCtx := sessionContext{
		ctx:    ctx,
		conn:   conn,
		server: &s,
		logger: logger,
	}
	defer sessionCtx.Close()

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

		if shouldExit := s.parseIncomingRequest(&sessionCtx, buffer[:length]); shouldExit {
			break
		}
	}
}

func (s DataServer) parseIncomingRequest(sessionCtx *sessionContext, request []byte) bool {
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
		// don't treat missing session as an error, just log and continue
		// the 2nd part of selecting a character won't pass in a valid session key for whatever reason
		sessionCtx.logger.Warn("failed to lookup account session", "sessionKey", sessionKey, "error", err)
	}

	// make sure this session context has subscriptions set up
	// this should only be done once per session
	if err = sessionCtx.SetupSubscriptions(sessionKey); err != nil {
		sessionCtx.logger.Error("failed to setup subscriptions", "error", err)
		return true
	}

	switch header.Command {
	case CommandRequestKeepXILoaderSpinning:
		// this is just a keep-alive, respond with empty payload
		_, _ = sessionCtx.conn.Write([]byte{})
	case CommandRequestGetCharacters:
		return s.handleRequestGetCharacters(sessionCtx, &accountSession, request)
	case CommandRequestSelectCharacter:
		return s.handleRequestSelectCharacter(sessionCtx, request)
	}

	return false
}

func (s *DataServer) sendErrorResponse(sessionCtx *sessionContext) {
	response, err := packets.NewResponseError(packets.ErrorCodeUnableToConnectToLobbyServer)
	if err != nil {
		return
	}

	responsePacket, err := response.Serialize()
	if err != nil {
		return
	}

	// it's okay if this write fails, we're already in an error state
	_, _ = sessionCtx.conn.Write(responsePacket)
}
