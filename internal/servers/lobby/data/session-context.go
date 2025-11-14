package data

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
	server        *DataServer
	logger        *slog.Logger

	accountID           uint32
	selectedCharacterID uint32
	freshCharacterLogin bool
	sessionKey          string
}

func (s *sessionContext) SetupSubscriptions(sessionKey string) error {
	if len(s.subscriptions) > 0 {
		return nil
	}

	// store the session key for use later
	s.sessionKey = sessionKey

	// add the close request subscription
	subject := fmt.Sprintf("session.%s.data.close", sessionKey)
	if err := s.addNATSSubscription(subject, s.processNATSCloseRequest); err != nil {
		return fmt.Errorf("failed to subscribe to NATS for session close: %w", err)
	}

	// add the send request subscription
	subject = fmt.Sprintf("session.%s.data.send", sessionKey)
	if err := s.addNATSSubscription(subject, s.processNATSSendRequest); err != nil {
		return fmt.Errorf("failed to subscribe to NATS for session send: %w", err)
	}

	// add the fresh character login subscription
	subject = fmt.Sprintf("session.%s.data.character.freshlogin", sessionKey)
	if err := s.addNATSSubscription(subject, s.processNATSFreshCharacterLogin); err != nil {
		return fmt.Errorf("failed to subscribe to NATS for fresh character login: %w", err)
	}

	// add the selected character ID subscription
	subject = fmt.Sprintf("session.%s.data.character.selectID", sessionKey)
	if err := s.addNATSSubscription(subject, s.processNATSSelectedCharacterID); err != nil {
		return fmt.Errorf("failed to subscribe to NATS for selected character ID: %w", err)
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
}

func (s *sessionContext) processNATSFreshCharacterLogin(msg *nats.Msg) {
	s.logger.Info("received NATS message for fresh character login")
	_ = msg.Ack()

	s.freshCharacterLogin = true
}

func (s *sessionContext) processNATSSelectedCharacterID(msg *nats.Msg) {
	s.logger.Info("received NATS message for selected character ID", "length", len(msg.Data))
	_ = msg.Ack()

	if len(msg.Data) == 4 {
		s.selectedCharacterID = binary.LittleEndian.Uint32(msg.Data)
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
