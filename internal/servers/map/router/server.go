package router

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/GoFFXI/GoFFXI/internal/ffxizlib"
	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
	"github.com/GoFFXI/GoFFXI/internal/servers/base/udp"
)

type MapRouterServer struct {
	*udp.UDPServer

	sessions      map[string]*Session
	packetsToSend map[string][]*mapPackets.RoutedPacket
	codec         *ffxizlib.Codec
}

func NewMapRouterServer(baseServer *udp.UDPServer) *MapRouterServer {
	var codec *ffxizlib.Codec
	if baseServer != nil && baseServer.Config() != nil {
		codec = ffxizlib.NewCodec(baseServer.Config().FFXIResourcePath)
	} else {
		codec = ffxizlib.NewCodec("")
	}

	return &MapRouterServer{
		UDPServer:     baseServer,
		sessions:      make(map[string]*Session),
		packetsToSend: make(map[string][]*mapPackets.RoutedPacket),
		codec:         codec,
	}
}

func (s *MapRouterServer) HandleIncomingPacket(ctx context.Context, length int, data []byte, clientAddr *net.UDPAddr) {
	clientAddrStr := clientAddr.String()
	s.Logger().Info("handling incoming packet", "dataLength", length, "clientAddr", clientAddrStr)

	// attempt to decrypt the packet
	decryptAttempts, decryptedPacket := s.decryptPacket(ctx, length, data, clientAddr)

	// make sure the decryptAttempts is not negative
	if decryptAttempts < 0 {
		s.Logger().Warn("failed to decrypt packet", "clientAddr", clientAddrStr)
		return
	}

	// attempt to lookup the session by client address
	session, sessionExists := s.sessions[clientAddrStr]
	if !sessionExists {
		s.Logger().Warn("no session found for client address", "clientAddr", clientAddrStr)
		return
	}

	// if the packet was successfully decrypted (or already unencrypted), it's time to parse it
	switch decryptAttempts {
	case 0, 1:
		if decryptedPacket == nil {
			// nothing to parse (e.g., login handshake)
			return
		}
		if len(decryptedPacket) <= mapPackets.HeaderSize {
			s.Logger().Warn("decrypted packet too small", "size", len(decryptedPacket))
			return
		}

		packets, err := s.parsePackets(decryptedPacket, session)
		if err != nil {
			s.Logger().Warn("failed to parse packet", "clientAddr", clientAddrStr, "error", err)
			return
		}

		// relay the parsed packets to the appropriate map server via NATS
		s.relayPackets(packets, session)
	case 2:
		s.Logger().Warn("client failed to rotate blowfish key", "clientAddr", clientAddrStr)
	}
}

// createSession creates a new session for the given character ID and client address.
func (s *MapRouterServer) createSession(ctx context.Context, characterID uint32, clientAddr *net.UDPAddr) (*Session, error) {
	accountSession, err := s.DB().GetAccountSessionByCharacterID(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup account session for character ID %d: %w", characterID, err)
	}

	if accountSession.ClientIP != clientAddr.IP.String() {
		return nil, fmt.Errorf("client IP mismatch for character ID %d: expected %s, got %s", characterID, accountSession.ClientIP, clientAddr.IP.String())
	}

	session, err := NewSession(clientAddr, string(accountSession.SessionKey), s)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// In session creation
	s.Logger().Info("BLOWFISH DEBUG",
		"sessionKeyHex", hex.EncodeToString(accountSession.SessionKey),
		"blowfishKeyHex", hex.EncodeToString(session.currentBlowfish.GetKeyBytes()),
		"md5HashHex", session.currentBlowfish.HashHex(),
		"match", bytes.Equal(accountSession.SessionKey, session.currentBlowfish.GetKeyBytes()))

	s.sessions[clientAddr.String()] = session
	return session, nil
}

// relayPackets sends the given packets to the appropriate map server via NATS.
func (s *MapRouterServer) relayPackets(packets []mapPackets.BasicPacket, session *Session) {
	var err error

	// relay packets to the appropriate map server via NATS
	// build the NATS subject based on the character's map server ID
	subject := fmt.Sprintf("map.instance.%d", session.character.PosZone)

	for _, packet := range packets {
		// create a new routed packet
		routedPacket := mapPackets.RoutedPacket{
			ClientAddr:  session.clientAddr.String(),
			CharacterID: session.character.ID,
			Packet:      packet,
		}

		err = s.NATS().Publish(subject, routedPacket.ToJSON())
		if err != nil {
			s.Logger().Warn("failed to publish routed packet to NATS", "subject", subject, "error", err)
		}
	}
}

func (s *MapRouterServer) DeliverPacketsToClients(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(time.Millisecond * 100)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for clientAddr, packets := range s.packetsToSend {
				if len(packets) == 0 {
					continue
				}

				if err := s.sendPacketsToClient(packets, clientAddr); err != nil {
					s.Logger().Warn("failed to send packets to client", "clientAddr", clientAddr, "error", err)
					continue
				}

				s.packetsToSend[clientAddr] = nil
			}
		}
	}
}
