package data

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/GoFFXI/login-server/internal/config"
	"github.com/GoFFXI/login-server/internal/database"
	"github.com/GoFFXI/login-server/internal/database/migrations"
	"github.com/GoFFXI/login-server/internal/server"
	"github.com/GoFFXI/login-server/internal/tools"
	"github.com/nats-io/nats.go"
)

const (
	RequestKeepXILoaderSpinning           = 0xFE
	RequestNotifyLobbyOfCurrentSelections = 0xA2
	RequestGetCharacterData               = 0xA1
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

func (s *DataServer) handleConnection(ctx context.Context, conn net.Conn) {
	logger := s.Logger().With("client", conn.RemoteAddr().String())
	logger.Info("processing connection")

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
		opCode := tools.GetIntFromByteBuffer(request, 0)
		sessionKey := tools.BytesToString(request, 12, 16)

		// attempt to lookup the account session
		logger.Info("looking up session", "sessionKey", sessionKey, "opCode", opCode)
		accountSession, err := s.DB().GetAccountSessionBySessionKey(ctx, sessionKey)
		if err != nil {
			// this shouldn't happen normally, log and close the connection
			logger.Error("failed to lookup account session", "session_key", sessionKey, "error", err)
			_ = conn.Close()

			return
		}

		// subscribe to session-related NATS messages
		callback := (func(conn net.Conn, accountSession *database.AccountSession) func(msg *nats.Msg) {
			// all of this hoopla is just to bind the conn & accountSession variables into the closure
			return func(msg *nats.Msg) {
				s.handleNATSSendRequest(conn, msg, accountSession)
			}
		})(conn, &accountSession)

		_, err = s.NATS().Subscribe(fmt.Sprintf("session.%s.data.send", accountSession.SessionKey), callback)
		if err != nil {
			logger.Error("failed to subscribe to NATS", "error", err)
			_ = conn.Close()
			return
		}

		switch opCode {
		case RequestKeepXILoaderSpinning:
			logger.Debug("keeping XILoader spinning")
			_, _ = conn.Write([]byte{})
		case RequestNotifyLobbyOfCurrentSelections:
			s.opNotifyLobbyOfCurrentSelection(ctx, conn, &accountSession, request)
		case RequestGetCharacterData:
			s.opGetCharacterData(ctx, conn, &accountSession, request)
		}
	}
}

func (s *DataServer) handleNATSSendRequest(conn net.Conn, msg *nats.Msg, _ *database.AccountSession) {
	s.Logger().Info("received NATS message to send data to client", "length", len(msg.Data))
	_ = msg.Ack()

	// so far, we're only sending raw data back to the client
	_, err := conn.Write(msg.Data)
	if err != nil {
		s.Logger().Error("failed to write NATS data to connection", "error", err)
		_ = conn.Close()
	}
}
