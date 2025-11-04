package view

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/nats-io/nats.go"
)

type sessionContext struct {
	ctx           context.Context
	conn          net.Conn
	subscriptions []*nats.Subscription
	server        *ViewServer
	logger        *slog.Logger
}

func (s *sessionContext) SetupSubscriptions(sessionKey string) error {
	if len(s.subscriptions) > 0 {
		return nil
	}

	// add the close request subscription
	subsription, err := s.server.NATS().Subscribe(fmt.Sprintf("session.%s.view.close", sessionKey), s.processNATSCloseRequest)
	if err != nil {
		return fmt.Errorf("failed to subscribe to NATS for session close: %w", err)
	}
	s.subscriptions = append(s.subscriptions, subsription)

	// add the send request subscription
	subsription, err = s.server.NATS().Subscribe(fmt.Sprintf("session.%s.view.send", sessionKey), s.processNATSSendRequest)
	if err != nil {
		return fmt.Errorf("failed to subscribe to NATS for session send: %w", err)
	}
	s.subscriptions = append(s.subscriptions, subsription)

	return nil
}

func (s *sessionContext) processNATSCloseRequest(msg *nats.Msg) {
	s.logger.Info("received NATS message to close session")
	_ = msg.Ack()

	s.Close()
}

func (s *sessionContext) processNATSSendRequest(msg *nats.Msg) {
	s.logger.Info("received NATS message to send data to client", "length", len(msg.Data))
	_ = msg.Ack()

	// so far, we're only sending raw data back to the client
	_, err := s.conn.Write(msg.Data)
	if err != nil {
		s.logger.Error("failed to write NATS data to connection", "error", err)
		s.Close()
	}
}

func (s *sessionContext) Close() {
	for _, sub := range s.subscriptions {
		_ = sub.Unsubscribe()
	}

	s.subscriptions = nil

	if s.conn != nil {
		_ = s.conn.Close()
	}

	s.conn = nil
}
