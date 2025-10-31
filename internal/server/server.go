package server

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/GoFFXI/login-server/internal/config"
	"github.com/nats-io/nats.go"
)

type Server struct {
	socket      net.Listener
	log         *slog.Logger
	cfg         config.Config
	nc          *nats.Conn
	connections chan net.Conn
}

func NewServer(cfg config.Config, logger *slog.Logger, nc *nats.Conn) (*Server, error) {
	var socket net.Listener
	var err error

	if cfg.AuthServerTLSCertPath != "" && cfg.AuthServerTLSKeyPath != "" {
		// start with TLS
		cert, err := tls.LoadX509KeyPair(cfg.AuthServerTLSCertPath, cfg.AuthServerTLSKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		socket, err = tls.Listen("tcp4", fmt.Sprintf(":%d", cfg.ServerPort), tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to start TLS listener: %w", err)
		}
	} else {
		// start without TLS
		socket, err = net.Listen("tcp4", fmt.Sprintf(":%d", cfg.ServerPort))
		if err != nil {
			return nil, fmt.Errorf("failed to start TCP listener: %w", err)
		}
	}

	logger.Info("server listening", "address", socket.Addr().String())

	srv := &Server{
		socket:      socket,
		log:         logger,
		cfg:         cfg,
		nc:          nc,
		connections: make(chan net.Conn, cfg.MaxServerConnections),
	}

	return srv, nil
}

func (s *Server) SetupSignalHandler() chan os.Signal {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	return signalChannel
}

func (s *Server) Config() config.Config {
	return s.cfg
}

func (s *Server) Logger() *slog.Logger {
	return s.log
}

func (s *Server) NATS() *nats.Conn {
	return s.nc
}

func (s *Server) Socket() net.Listener {
	return s.socket
}

func (s *Server) Connections() chan net.Conn {
	return s.connections
}

func (s *Server) Shutdown() {
	// close listener to stop accepting new connections
	if s.socket != nil {
		if err := s.socket.Close(); err != nil {
			s.log.Error("failed to close socket", "error", err)
		}
	}

	close(s.connections)
}
