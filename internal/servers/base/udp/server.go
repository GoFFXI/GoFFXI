package udp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/GoFFXI/GoFFXI/internal/config"
	"github.com/GoFFXI/GoFFXI/internal/database"
)

const (
	MaxBufferSize = 4096
)

// ConnectionHandler defines a function type for handling incoming UDP connections.
type ConnectionHandler func(ctx context.Context, length int, data []byte, clientAddr *net.UDPAddr)

// UDPServer represents a UDP server.
type UDPServer struct {
	socket   *net.UDPConn
	log      *slog.Logger
	cfg      *config.Config
	natsConn *nats.Conn
	db       *database.DBImpl
}

// NewUDPServer creates and configures a new UDPServer instance.
func NewUDPServer(cfg *config.Config, logger *slog.Logger) (*UDPServer, error) {
	// resolve udp address to host on
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", cfg.ServerPort))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	// create the UDP listener
	socket, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start UDP listener: %w", err)
	}

	logger.Info("server listening", "address", socket.LocalAddr().String())
	srv := UDPServer{
		socket: socket,
		log:    logger,
		cfg:    cfg,
	}

	return &srv, nil
}

// Config returns the server's configuration.
func (s *UDPServer) Config() *config.Config {
	return s.cfg
}

// Logger returns the server's logger.
func (s *UDPServer) Logger() *slog.Logger {
	return s.log
}

// Socket returns the server's UDP socket.
func (s *UDPServer) Socket() *net.UDPConn {
	return s.socket
}

// NATS returns the server's NATS connection.
func (s *UDPServer) NATS() *nats.Conn {
	return s.natsConn
}

// DB returns the server's database instance.
func (s *UDPServer) DB() *database.DBImpl {
	return s.db
}

// ProcessConnections processes incoming UDP connections using the provided handler function.
func (s *UDPServer) ProcessConnections(ctx context.Context, wg *sync.WaitGroup, handler ConnectionHandler) {
	defer wg.Done()

	buffer := make([]byte, MaxBufferSize)

	for {
		select {
		case <-ctx.Done():
			s.Logger().Info("stopping connection processor")
			return
		default:
			n, clientAddr, err := s.Socket().ReadFromUDP(buffer)
			if err != nil {
				s.Logger().Error("error reading from UDP socket", "error", err)
				continue
			}

			// Make a copy of the data for processing
			data := make([]byte, n)
			copy(data, buffer[:n])

			// handle the incoming data
			handler(ctx, n, data, clientAddr)
		}
	}
}

// WaitForShutdown waits for shutdown signals and triggers the provided cancel function.
func (s *UDPServer) WaitForShutdown(cancelCtx context.CancelFunc, wg *sync.WaitGroup) error {
	// setup signal handling
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// block until signal received
	sig := <-signalChannel
	s.Logger().Info("shutdown signal received", "signal", sig.String())

	// cancel context to signal all gouroutines to stop
	cancelCtx()

	// wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.Logger().Info("all goroutines have finished")
		return nil
	case <-time.After(time.Duration(s.Config().ShutdownTimeoutSeconds) * time.Second):
		s.Logger().Warn("shutdown timeout reached, forcing exit")
		return fmt.Errorf("shutdown timeout reached")
	}
}
