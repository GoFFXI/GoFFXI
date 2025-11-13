package router

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/GoFFXI/GoFFXI/internal/database"
	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
	clientPackets "github.com/GoFFXI/GoFFXI/internal/packets/map/client"
	"github.com/GoFFXI/GoFFXI/internal/servers/base/udp"
)

type MapRouterServer struct {
	*udp.UDPServer

	sessions map[string]*Session
}

func NewMapRouterServer(baseServer *udp.UDPServer) *MapRouterServer {
	return &MapRouterServer{
		UDPServer: baseServer,
		sessions:  make(map[string]*Session),
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
		// Ensure the packet is large enough to have an MD5 checksum
		if len(decryptedPacket) <= mapPackets.HeaderSize+mapPackets.MD5ChecksumSize {
			// Handle packets too small to have MD5 (shouldn't happen in normal operation)
			s.Logger().Warn("decrypted packet too small", "size", len(decryptedPacket))
			return
		}

		// remember: the last 16 bytes are the MD5 checksum (which we don't need for parsing)
		packets, err := s.parsePackets(decryptedPacket[:len(decryptedPacket)-mapPackets.MD5ChecksumSize], session)
		if err != nil {
			s.Logger().Warn("failed to parse packet", "clientAddr", clientAddrStr, "error", err)
			return
		}

		// relay the parsed packets to the appropriate map server via NATS
		s.relayPackets(packets, session)
	case 2:
		// client failed to rotate keys
	}
}

// decryptPacket tries to decrypt the incoming packet.
// Returns -1 if decryption failed, 0 if unencrypted, 1 if decrypted successfully, 2 for successful decryption with previous key.
// It also returns the decrypted packet data.
func (s *MapRouterServer) decryptPacket(ctx context.Context, length int, data []byte, clientAddr *net.UDPAddr) (numAttempts int32, decryptedPacket []byte) {
	// check for underflow or no-data packet
	if length <= (mapPackets.HeaderSize + 16) {
		return -1, []byte{}
	}

	// attempt to lookup the session by client address
	session, sessionExists := s.sessions[clientAddr.String()]

	// the only unencrypted packet we expect is the login request packet
	// this is the only packet whose checksum we can validate at this stage
	loginPacket, err := clientPackets.ParseLoginPacket(data[:length])
	if err != nil {
		// if the parse failed due to invalid checksum, it's ok - assume the packet is encrypted
		// otherwise, we need to log the error
		if err.Error() != "invalid packet checksum" {
			s.Logger().Warn("failed to parse login packet, assuming encrypted", "error", err)
			return -1, []byte{}
		}
	}

	// check if the login packet was successfully parsed
	if loginPacket != nil {
		// if we don't have a session yet, create one
		if !sessionExists {
			session, err = s.createSession(ctx, loginPacket.UniqueNo, clientAddr)
			if err != nil {
				s.Logger().Warn("failed to create new session", "error", err)
				return -1, []byte{}
			}
		}

		// check if the session has already populated the character
		if session.character == nil {
			var character database.Character

			character, err = s.DB().GetCharacterByID(ctx, loginPacket.UniqueNo)
			if err != nil {
				s.Logger().Warn("failed to lookup character for session", "characterID", loginPacket.UniqueNo, "error", err)
				return -1, []byte{}
			}

			session.character = &character
		}

		// handle out of sync zone correction
		if loginPacket.Header.Sync > 1 {
			// This incoming login packet from the client wants us to set the starting sync count for
			// all new packets to the sync value in the login packet header.
			//
			// If we don't do this, all further packets may be ignored by the client and will result
			// in a disconnection from the server.
			session.lastServerPacketID = loginPacket.Header.Sync

			// todo: clear any pending packets in the session buffer
		}

		// at this point, we have a valid login packet and session
		return 0, data
	}

	// if we reach here, the packet is not a login packet so we must have an existing session to proceed
	if !sessionExists {
		s.Logger().Warn("no existing session for non-login packet", "clientAddr", clientAddr.String())
		return -1, []byte{}
	}

	// treat the packet as encrypted and try to decrypt it using the session data
	spew.Dump("encrypted packet found", data[:length])

	// attempt to decrypt the packet using the current blowfish key
	// if that fails, try the previous key (in case of key rotation)
	// https://github.com/LandSandBoat/server/blob/base/src/map/map_networking.cpp#L401

	// if everything succeeds, we can decompress the data using zlib

	return -1, []byte{}
}

// parsePackets extracts individual packets from the decrypted data buffer.
func (s *MapRouterServer) parsePackets(data []byte, session *Session) ([]mapPackets.BasicPacket, error) {
	packetSize := len(data)

	// update session activity time if not pending zone
	if session.currentBlowfish.status != BlowfishPendingZone && session.currentBlowfish.status != BlowfishWaiting {
		session.lastUpdate = time.Now()
	}

	// get the client packet ID from the header
	clientPacketID := binary.LittleEndian.Uint16(data[0:2])

	// start processing packets after the header
	packets := make([]mapPackets.BasicPacket, 0)
	packetDataBegin := mapPackets.HeaderSize
	packetDataEnd := packetSize

	// house-keeping
	s.Logger().Debug("parsing packets", "packetSize", packetSize, "clientPacketID", clientPacketID, "bytes", packetDataEnd-packetDataBegin)

	// loop through all small packets in the buffer
	for ptr := packetDataBegin; ptr < packetDataEnd; {
		// check if we have at least 2 bytes for size/type info
		if ptr+4 > packetDataEnd {
			s.Logger().Debug("not enough data for packet header", "ptr", ptr, "bytesNeeded", 4, "bytesAvailable", packetDataEnd-ptr)
			break
		}

		// read the size field from byte 1
		sizeByte := data[ptr+1]

		// the size is stored in bits 1-7 (masked with 0xFE)
		// this gives us the size in "chunks" of 2 bytes
		sizeInChunks := int(sizeByte & 0xFE)

		// if size is 0, we've hit the terminator
		if sizeInChunks == 0 {
			s.Logger().Debug("found terminator", "ptr", ptr)
			break
		}

		// actual packet size in bytes
		packetSizeBytes := sizeInChunks * 2
		s.Logger().Debug("packet size info", "ptr", ptr, "sizeByte", fmt.Sprintf("0x%02X", sizeByte), "sizeInChunks", sizeInChunks, "packetSizeBytes", packetSizeBytes)

		// try to parse what we have
		packetType := binary.LittleEndian.Uint16(data[ptr:ptr+2]) & 0x1FF

		// check if the full packet fits in the buffer
		if ptr+packetSizeBytes > packetDataEnd {
			// this might be the last packet and it's incomplete or there's corruption in the size field
			remainingBytes := packetDataEnd - ptr
			s.Logger().Debug("packet would extend beyond buffer", "ptr", ptr, "size", packetSizeBytes, "end", packetDataEnd, "remaining", remainingBytes)

			// for login packets, the entire remaining buffer might be the packet
			if remainingBytes >= 4 {
				// special handling for login packet which might not follow size rules
				if packetType == clientPackets.PacketTypeLogin {
					s.Logger().Debug("detected login packet, using remaining buffer", "remainingBytes", remainingBytes)
					packetSizeBytes = remainingBytes
				} else {
					return packets, fmt.Errorf("packet extends beyond buffer")
				}
			} else {
				break
			}
		}

		// read packet sequence number
		var packetSequence uint16
		if ptr+4 <= packetDataEnd {
			packetSequence = binary.LittleEndian.Uint16(data[ptr+2 : ptr+4])
		}

		s.Logger().Debug("parsed packet", "type", fmt.Sprintf("0x%03X", packetType), "sequence", packetSequence, "sizeBytes", packetSizeBytes)

		// validate packet sequence (skip this for login packets as they might not follow the rules)
		if packetType != clientPackets.PacketTypeLogin {
			if (packetSequence <= session.lastClientPacketID) || (packetSequence > clientPacketID) {
				// skip packets with invalid sequence numbers
				s.Logger().Debug("skipping packet with invalid sequence", "packetSequence", packetSequence, "lastClientPacketID", session.lastClientPacketID, "clientPacketID", clientPacketID)
				ptr += packetSizeBytes
				continue
			}
		}

		// create BasicPacket
		packet := mapPackets.BasicPacket{
			Type:     packetType,
			Size:     uint16(sizeInChunks / 2), //nolint:gosec // size in chunks of 2 bytes
			Sequence: packetSequence,
			Data:     make([]byte, packetSizeBytes),
		}

		// copy packet data
		copy(packet.Data, data[ptr:ptr+packetSizeBytes])

		// append to packets list
		packets = append(packets, packet)
		s.Logger().Debug("added packet", "type", fmt.Sprintf("0x%03X", packetType), "size", packetSizeBytes)

		// move to next packet
		ptr += packetSizeBytes
	}

	// Update client packet ID to the overall packet count
	if len(packets) > 0 {
		session.lastClientPacketID = clientPacketID
	}

	s.Logger().Info("finished parsing packets", "totalPackets", len(packets))
	return packets, nil
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

	// https://github.com/LandSandBoat/server/blob/5a2141c6e331733779abd6217b3bab930aee0dc0/src/map/map_session.cpp#L35
	session, err := NewSession(clientAddr, accountSession.SessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

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
