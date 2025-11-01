package view

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/GoFFXI/login-server/internal/config"
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

func (s *ViewServer) handleConnection(_ context.Context, conn net.Conn) {
	//nolint:errcheck // closing connection
	defer conn.Close()

	logger := s.Logger().With("client", conn.RemoteAddr().String())
	logger.Info("processing connection")
}
