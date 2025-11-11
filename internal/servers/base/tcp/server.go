package tcp

import (
	"context"
	"crypto/tls"
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

// ConnectionHandler defines a function type for handling incoming connections.
type ConnectionHandler func(ctx context.Context, conn net.Conn)

// TCPServer represents a TCP server with optional TLS support.
type TCPServer struct {
	socket      net.Listener
	log         *slog.Logger
	connections chan net.Conn
	cfg         *config.Config
	natsConn    *nats.Conn
	db          *database.DBImpl
}

// NewTCPServer creates and configures a new TCPServer instance.
func NewTCPServer(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*TCPServer, error) {
	var socket net.Listener
	var cert tls.Certificate
	var err error

	// Check for TLS configuration
	if cfg.ServerTLSCertPath != "" && cfg.ServerTLSKeyPath != "" {
		// start with TLS
		cert, err = tls.LoadX509KeyPair(cfg.ServerTLSCertPath, cfg.ServerTLSKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
		}

		logger.Info("TLS certificates loaded", "certPath", cfg.ServerTLSCertPath, "keyPath", cfg.ServerTLSKeyPath)

		tlsConfig := &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}

		socket, err = tls.Listen("tcp4", fmt.Sprintf(":%d", cfg.ServerPort), tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to start TLS listener: %w", err)
		}
	} else {
		netConfig := &net.ListenConfig{}

		// start without TLS
		socket, err = netConfig.Listen(ctx, "tcp4", fmt.Sprintf(":%d", cfg.ServerPort))
		if err != nil {
			return nil, fmt.Errorf("failed to start TCP listener: %w", err)
		}
	}

	logger.Info("server listening", "address", socket.Addr().String())
	srv := TCPServer{
		socket:      socket,
		log:         logger,
		cfg:         cfg,
		connections: make(chan net.Conn, cfg.MaxServerConnections),
	}

	return &srv, nil
}

// Config returns the server's configuration.
func (s *TCPServer) Config() *config.Config {
	return s.cfg
}

// Logger returns the server's logger.
func (s *TCPServer) Logger() *slog.Logger {
	return s.log
}

// Socket returns the server's network listener.
func (s *TCPServer) Socket() net.Listener {
	return s.socket
}

// NATS returns the server's NATS connection.
func (s *TCPServer) NATS() *nats.Conn {
	return s.natsConn
}

// DB returns the server's database instance.
func (s *TCPServer) DB() *database.DBImpl {
	return s.db
}

// AcceptConnections starts accepting incoming TCP connections.
func (s *TCPServer) AcceptConnections(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		// accept will block until a new connection arrives or the listener is closed
		conn, err := s.Socket().Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				s.Logger().Info("stopping connection acceptor")
				return
			default:
				s.Logger().Error("failed to accept connection", "error", err)
				continue
			}
		}

		remoteAddr := conn.RemoteAddr().String()
		// todo: check if this IP is banned before proceeding
		s.Logger().Info("new connection accepted", "client", remoteAddr)

		// add connection to channel for processing
		select {
		case s.connections <- conn:
			s.Logger().Debug("connection queued for processing", "client", remoteAddr)
		case <-ctx.Done():
			// server is shutting down, close the connection
			if conn != nil {
				_ = conn.Close()
			}
			return
		default:
			// connection channel is full, reject the connection
			s.Logger().Warn("connection queue full, rejecting connection", "client", remoteAddr)

			if conn != nil {
				_ = conn.Close()
			}
		}
	}
}

// ProcessConnections processes incoming connections using the provided handler function.
func (s *TCPServer) ProcessConnections(ctx context.Context, wg *sync.WaitGroup, handler ConnectionHandler) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			s.Logger().Info("stopping connection processor")

			// drain remaining connections
			for {
				select {
				case conn := <-s.connections:
					if conn != nil {
						_ = conn.Close()
					}
				default:
					return
				}
			}
		case conn := <-s.connections:
			// handle each connection in a separate goroutine
			// this ensures we can process multiple connections concurrently
			// while still maintaining the order through the channel
			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				handler(ctx, c)
			}(conn)
		}
	}
}

func (s *TCPServer) WaitForShutdown(cancelCtx context.CancelFunc, wg *sync.WaitGroup) error {
	// setup signal handling
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// block until signal received
	sig := <-signalChannel
	s.Logger().Info("shutdown signal received", "signal", sig.String())

	// cancel context to signal all gouroutines to stop
	cancelCtx()

	// close listener to stop accepting new connections
	if s.Socket() != nil {
		if err := s.Socket().Close(); err != nil {
			s.log.Error("failed to close socket", "error", err)
		}
	}

	// close connection channel to stop processing new connections
	close(s.connections)

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
