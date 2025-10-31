package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/GoFFXI/login-server/internal/config"
	"github.com/GoFFXI/login-server/internal/server"
	"github.com/nats-io/nats.go"
)

type AuthServer struct {
	*server.Server

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func Run(cfg *config.Config, logger *slog.Logger, nc *nats.Conn) error {
	// create a context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	baseServer, err := server.NewServer(ctx, cfg, logger, nc)
	if err != nil {
		return fmt.Errorf("failed to create auth server: %w", err)
	}

	authServer := &AuthServer{
		Server: baseServer,
		ctx:    ctx,
		cancel: cancel,
	}

	//nolint:errcheck // socket will be closed on shutdown
	defer authServer.Socket().Close()

	// start connection processor goroutine
	authServer.wg.Add(1)
	go authServer.processConnections()

	// start accepting connections
	authServer.wg.Add(1)
	go authServer.acceptConnections()

	// wait for shutdown signal
	return authServer.waitForShutdown()
}

func (s *AuthServer) processConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			s.Logger().Info("stopping connection processor")

			// drain remaining connections
			for {
				select {
				case conn := <-s.Connections():
					_ = conn.Close()
				default:
					return
				}
			}
		case conn := <-s.Connections():
			// handle each connection in a separate goroutine
			// this ensures we can process multiple connections concurrently
			// while still maintaining the order through the channel
			s.wg.Add(1)
			go func(c net.Conn) {
				defer s.wg.Done()
				s.handleConnection(c)
			}(conn)
		}
	}
}

func (s *AuthServer) acceptConnections() {
	defer s.wg.Done()

	for {
		// accept will block until a new connection arrives or the listener is closed
		conn, err := s.Socket().Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				s.Logger().Info("stopping connection acceptor")
				return
			default:
				s.Logger().Error("failed to accept connection", "error", err)
				continue
			}
		}

		remoteAddr := conn.RemoteAddr().String()
		s.Logger().Info("new connection accepted", "client", remoteAddr)

		// add connection to channel for processing
		select {
		case s.Connections() <- conn:
			s.Logger().Debug("connection queued for processing", "client", remoteAddr)
		case <-s.ctx.Done():
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

func (s *AuthServer) handleConnection(conn net.Conn) {
	//nolint:errcheck // closing connection
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	s.Logger().Info("processing connection", "client", clientAddr)

	// set read/write timeout for the connection
	_ = conn.SetDeadline(time.Now().Add(time.Duration(s.Config().ServerReadTimeoutSeconds) * time.Second))

	// // connection handling loop
	// for {
	// 	select {
	// 	case <-s.ctx.Done():
	// 		return
	// 	default:
	// 	}

	// 	// todo: read data from the client
	// }
}

func (s *AuthServer) waitForShutdown() error {
	signalChannel := s.SetupSignalHandler()

	// block until signal received
	sig := <-signalChannel
	s.Logger().Info("shutdown signal received", "signal", sig.String())

	// cancel context to signal all gouroutines to stop
	s.cancel()

	// shutdown the server to stop accepting new connections
	s.Shutdown()

	// wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
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
