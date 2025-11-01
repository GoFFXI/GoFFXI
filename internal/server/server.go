package server

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

	"github.com/GoFFXI/login-server/internal/config"
	"github.com/nats-io/nats.go"
)

type ConnectionHandler func(ctx context.Context, conn net.Conn)

type Server struct {
	socket      net.Listener
	log         *slog.Logger
	connections chan net.Conn
	cfg         *config.Config
	natsConn    *nats.Conn
}

func NewServer(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Server, error) {
	var socket net.Listener
	var cert tls.Certificate
	var err error

	if cfg.AuthServerTLSCertPath != "" && cfg.AuthServerTLSKeyPath != "" {
		// start with TLS
		cert, err = tls.LoadX509KeyPair(cfg.AuthServerTLSCertPath, cfg.AuthServerTLSKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
		}

		logger.Debug("TLS certificates loaded", "certPath", cfg.AuthServerTLSCertPath, "keyPath", cfg.AuthServerTLSKeyPath)

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

	srv := &Server{
		socket:      socket,
		log:         logger,
		cfg:         cfg,
		connections: make(chan net.Conn, cfg.MaxServerConnections),
	}

	return srv, nil
}

func (s *Server) Config() *config.Config {
	return s.cfg
}

func (s *Server) Logger() *slog.Logger {
	return s.log
}

func (s *Server) Socket() net.Listener {
	return s.socket
}

func (s *Server) NATS() *nats.Conn {
	return s.natsConn
}

func (s *Server) ProcessConnections(ctx context.Context, wg *sync.WaitGroup, handler ConnectionHandler) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			s.Logger().Info("stopping connection processor")

			// drain remaining connections
			for {
				select {
				case conn := <-s.connections:
					_ = conn.Close()
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

func (s *Server) AcceptConnections(ctx context.Context, wg *sync.WaitGroup) {
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
			_ = conn.Close()
			return
		default:
			// connection channel is full, reject the connection
			s.Logger().Warn("connection queue full, rejecting connection", "client", remoteAddr)
			_ = conn.Close()
		}
	}
}

func (s *Server) WaitForShutdown(cancelCtx context.CancelFunc, wg *sync.WaitGroup) error {
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

func (s *Server) CreateNATSConnection() error {
	hostname, _ := os.Hostname()

	// create a new NATS connection
	options := []nats.Option{
		nats.Name(fmt.Sprintf("%s%s", s.Config().NATSClientPrefix, hostname)),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2 * time.Second),
		nats.ReconnectBufSize(s.Config().NATSOutgoingBufferSize),
		nats.DisconnectErrHandler(s.OnNATSDisconnected),
		nats.ReconnectHandler(s.OnNATSReconnected),
		nats.ClosedHandler(s.OnNATSClosed),
	}

	// connect to NATS server
	nc, err := nats.Connect(s.Config().NATSURL, options...)
	if err != nil {
		return err
	}

	s.natsConn = nc
	return nil
}

func (s *Server) OnNATSDisconnected(_ *nats.Conn, err error) {
	s.Logger().Warn("NATS disconnected", "error", err)
	s.natsConn = nil
}

func (s *Server) OnNATSReconnected(nc *nats.Conn) {
	s.Logger().Info("NATS reconnected")
	s.natsConn = nc
}

func (s *Server) OnNATSClosed(_ *nats.Conn) {
	s.Logger().Info("NATS connection permanently closed")
	s.natsConn = nil
}
