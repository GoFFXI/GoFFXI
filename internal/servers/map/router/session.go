package router

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/GoFFXI/GoFFXI/internal/database"
	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
	"github.com/nats-io/nats.go"
)

type Session struct {
	clientAddr *net.UDPAddr
	character  *database.Character
	lastUpdate time.Time

	lastClientPacketID uint16
	lastServerPacketID uint16

	sessionKey       string
	currentBlowfish  *Blowfish
	previousBlowfish *Blowfish

	server        *MapRouterServer
	subscriptions []*nats.Subscription

	lastServerPacket     []byte
	lastServerPacketSize int
}

func NewSession(clientAddr *net.UDPAddr, sessionKey string, server *MapRouterServer) (*Session, error) {
	blowfish, err := NewBlowfish(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create blowfish for session: %w", err)
	}

	session := &Session{
		clientAddr:      clientAddr,
		character:       nil,
		lastUpdate:      time.Now(),
		sessionKey:      sessionKey,
		currentBlowfish: blowfish,
		server:          server,
		subscriptions:   []*nats.Subscription{},
	}

	// setup the NATS subscriptions for the session
	subject := fmt.Sprintf("map.router.%s.send", clientAddr.String())
	if err := session.addNATSSubscription(subject, session.processNATSSendRequest); err != nil {
		return nil, fmt.Errorf("failed to subscribe to NATS for session send: %w", err)
	}

	return session, nil
}

func (s *Session) addNATSSubscription(subject string, handler nats.MsgHandler) error {
	subscription, err := s.server.NATS().Subscribe(subject, handler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to NATS subject %s: %w", subject, err)
	}

	s.subscriptions = append(s.subscriptions, subscription)
	return nil
}

func (s *Session) processNATSSendRequest(msg *nats.Msg) {
	s.server.Logger().Info("processing NATS send request", "clientAddr", s.clientAddr.String())
	_ = msg.Ack()

	var routedPacket mapPackets.RoutedPacket
	if err := json.Unmarshal(msg.Data, &routedPacket); err != nil {
		s.server.Logger().Warn("failed to unmarshal routed packet from NATS", "clientAddr", s.clientAddr.String(), "error", err)
		return
	}

	// queue the packet to be sent to the client
	s.server.Logger().Info("queuing packet to send to client", "clientAddr", s.clientAddr.String(), "packetType", routedPacket.Packet.Type, "packetSize", routedPacket.Packet.Size)
	if s.server.packetsToSend == nil {
		s.server.packetsToSend = make(map[string][]*mapPackets.RoutedPacket)
	}

	if s.server.packetsToSend[routedPacket.ClientAddr] == nil {
		s.server.packetsToSend[routedPacket.ClientAddr] = []*mapPackets.RoutedPacket{}
	}

	s.server.packetsToSend[routedPacket.ClientAddr] = append(s.server.packetsToSend[routedPacket.ClientAddr], &routedPacket)
}

func (s *Session) IncrementBlowfish() error {
	// Save the current key as previous
	s.previousBlowfish = s.currentBlowfish

	// Create new Blowfish with incremented key
	newBF := &Blowfish{
		key:    s.currentBlowfish.key,
		status: BlowfishPendingZone,
	}

	// Increment the key
	if err := newBF.IncrementKey(); err != nil {
		return err
	}

	s.currentBlowfish = newBF
	return nil
}

func (s *Session) Close() {
	for _, sub := range s.subscriptions {
		_ = sub.Unsubscribe()
	}

	s.subscriptions = nil
}
