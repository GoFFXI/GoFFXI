package data

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/GoFFXI/login-server/internal/config"
	"github.com/GoFFXI/login-server/internal/database/migrations"
	"github.com/GoFFXI/login-server/internal/server"
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

func (s *DataServer) handleConnection(_ context.Context, conn net.Conn) {
	//nolint:errcheck // closing connection
	defer conn.Close()

	logger := s.Logger().With("client", conn.RemoteAddr().String())
	logger.Info("processing connection")
}
