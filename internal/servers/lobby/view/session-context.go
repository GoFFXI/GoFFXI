package view

import (
	"context"
	"encoding/binary"
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

	accountID              uint32
	requestedCharacterName string
	selectedCharacterID    uint32
}

func (s *sessionContext) SetupSubscriptions(sessionKey string) error {
	if len(s.subscriptions) > 0 {
		return nil
	}

	// add the close request subscription
	subject := fmt.Sprintf("session.%s.view.close", sessionKey)
	if err := s.addNATSSubscription(subject, s.processNATSCloseRequest); err != nil {
		return fmt.Errorf("failed to subscribe to NATS for session close: %w", err)
	}

	// add the send request subscription
	subject = fmt.Sprintf("session.%s.view.send", sessionKey)
	if err := s.addNATSSubscription(subject, s.processNATSSendRequest); err != nil {
		return fmt.Errorf("failed to subscribe to NATS for session send: %w", err)
	}

	// add the account ID subscription
	subject = fmt.Sprintf("session.%s.view.account.id", sessionKey)
	if err := s.addNATSSubscription(subject, s.processNATSAccountID); err != nil {
		return fmt.Errorf("failed to subscribe to NATS for account ID: %w", err)
	}

	return nil
}

func (s *sessionContext) addNATSSubscription(subject string, handler nats.MsgHandler) error {
	subscription, err := s.server.NATS().Subscribe(subject, handler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to NATS subject %s: %w", subject, err)
	}

	s.subscriptions = append(s.subscriptions, subscription)
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

	// check for shutdown command
	if len(msg.Data) >= 12 {
		command := binary.LittleEndian.Uint32(msg.Data[8:12])

		if command == 0x000B {
			s.logger.Info("received shutdown command from NATS data")
			s.Close()
		}
	}
}

func (s *sessionContext) processNATSAccountID(msg *nats.Msg) {
	s.logger.Info("received NATS message with account ID")
	_ = msg.Ack()

	// Process the account ID
	if len(msg.Data) != 4 {
		s.logger.Error("invalid account ID length", "length", len(msg.Data))
		return
	}

	s.accountID = binary.LittleEndian.Uint32(msg.Data)
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
