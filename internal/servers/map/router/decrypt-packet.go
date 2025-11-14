package router

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/GoFFXI/GoFFXI/internal/database"
	"github.com/GoFFXI/GoFFXI/internal/ffxizlib"
	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
	clientPackets "github.com/GoFFXI/GoFFXI/internal/packets/map/client"
)

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
			if !sessionExists {
				session.lastServerPacketID = 0
			}

			session.lastClientPacketID = loginPacket.Header.Sync

			// todo: clear any pending packets in the session buffer
			if s.packetsToSend == nil {
				s.packetsToSend = make(map[string][]*mapPackets.RoutedPacket)
			}
			s.packetsToSend[clientAddr.String()] = nil
		}

		// at this point, we have a valid login packet and session
		return 0, data
	}

	// if we reach here, the packet is not a login packet so we must have an existing session to proceed
	if !sessionExists {
		s.Logger().Warn("no existing session for non-login packet", "clientAddr", clientAddr.String())
		return -1, []byte{}
	}

	packet := data[:length]

	decryptCount := int32(0)
	needsBackup := session.currentBlowfish != nil && session.currentBlowfish.status == BlowfishPendingZone && session.previousBlowfish != nil
	var backup []byte

	if needsBackup {
		backup = make([]byte, len(packet))
		copy(backup, packet)
	}

	if !s.tryDecryptPacket(packet, session.currentBlowfish) {
		if needsBackup && session.previousBlowfish != nil && s.tryDecryptPacket(backup, session.previousBlowfish) {
			copy(packet, backup)
			decryptCount = 1
		} else {
			s.Logger().Warn("failed to decrypt packet with available keys", "client", clientAddr.String())
			return -1, []byte{}
		}
	}

	if session.currentBlowfish != nil && session.currentBlowfish.status == BlowfishWaiting {
		session.currentBlowfish.status = BlowfishAccepted
	}

	decompressed, err := s.decompressPacket(packet)
	if err != nil {
		s.Logger().Warn("failed to decompress packet", "client", clientAddr.String(), "error", err)
		return -1, []byte{}
	}

	return decryptCount, decompressed
}

func (s *MapRouterServer) tryDecryptPacket(packet []byte, bf *Blowfish) bool {
	if bf == nil {
		return false
	}

	bf.DecryptPacket(packet, mapPackets.HeaderSize)
	return mapPackets.PerformPacketChecksum(packet)
}

func (s *MapRouterServer) decompressPacket(packet []byte) ([]byte, error) {
	if len(packet) < mapPackets.HeaderSize+4+mapPackets.MD5ChecksumSize {
		return nil, fmt.Errorf("packet too small for decompression (%d bytes)", len(packet))
	}

	if s.codec == nil {
		return nil, fmt.Errorf("ffxizlib codec not initialized")
	}

	sizeOffset := len(packet) - mapPackets.MD5ChecksumSize - 4
	bitCount := binary.LittleEndian.Uint32(packet[sizeOffset : sizeOffset+4])
	compressedBytes := ffxizlib.CompressedSize(bitCount)

	expectedCompressedEnd := mapPackets.HeaderSize + int(compressedBytes)
	if expectedCompressedEnd != sizeOffset {
		return nil, fmt.Errorf("compressed data length mismatch (expected %d, got %d)", expectedCompressedEnd, sizeOffset)
	}

	compressedData := packet[mapPackets.HeaderSize:expectedCompressedEnd]
	scratch := make([]byte, kMaxBufferSize)

	decompressedLen, err := s.codec.Decompress(compressedData, bitCount, scratch)
	if err != nil {
		return nil, err
	}

	finalSize := mapPackets.HeaderSize + decompressedLen
	if finalSize > len(packet) {
		return nil, fmt.Errorf("decompressed data exceeds packet buffer (%d > %d)", finalSize, len(packet))
	}

	copy(packet[mapPackets.HeaderSize:], scratch[:decompressedLen])
	return packet[:finalSize], nil
}
