package router

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/GoFFXI/GoFFXI/internal/ffxizlib"
	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
)

const (
	kMaxBufferSize           = 4096
	kMaxPacketPerCompression = 10
	kMaxPacketBacklogSize    = 100
)

func (s *MapRouterServer) sendPacketsToClient(packets []*mapPackets.RoutedPacket, clientAddr string) error {
	// lookup the session
	session, sessionExists := s.sessions[clientAddr]
	if !sessionExists {
		return fmt.Errorf("no session found for client address %s", clientAddr)
	}

	if len(packets) == 0 {
		s.Logger().Debug("no packets to send",
			"characterID", session.character.ID,
			"clientAddr", session.clientAddr.String())
		return fmt.Errorf("no packets to send")
	}

	s.Logger().Debug("preparing packets for client",
		"characterID", session.character.ID,
		"packetCount", len(packets),
		"serverPacketID", session.lastServerPacketID,
		"clientPacketID", session.lastClientPacketID)

	// Create the output buffer
	outBuffer := make([]byte, kMaxBufferSize)

	// === CRITICAL FIX: Properly format the 28-byte FFXI header ===
	// The header structure is very specific:
	// Bytes 0-1: Server packet ID (sequence we're sending)
	// Bytes 2-3: Client packet ID (last sequence we received)
	// Bytes 4-7: Reserved/padding (should be 0)
	// Bytes 8-11: Timestamp (32-bit Unix timestamp)
	// Bytes 12-27: More padding/reserved (should be 0)

	// Clear the header first (important!)
	for i := 0; i < mapPackets.HeaderSize; i++ {
		outBuffer[i] = 0
	}

	binary.LittleEndian.PutUint16(outBuffer[0:2], session.lastServerPacketID)
	binary.LittleEndian.PutUint16(outBuffer[2:4], session.lastClientPacketID)
	timestamp := uint32(time.Now().Unix())
	binary.LittleEndian.PutUint32(outBuffer[8:12], timestamp)

	s.Logger().Debug("packet header set",
		"serverPacketID", session.lastServerPacketID,
		"clientPacketID", session.lastClientPacketID,
		"timestamp", timestamp)

	// Build the packet data section (after header)
	dataOffset := mapPackets.HeaderSize
	packetCount := 0
	incrementKeyAfterEncrypt := false
	packetTypes := make([]string, 0, len(packets))

	// Combine multiple small packets into the data section
	for i, rPacket := range packets {
		packet := rPacket.Packet

		// === Build packet with header ===
		// packet.Data contains ONLY the payload (e.g., 248 bytes for login packet)
		// We ALWAYS need to add the 4-byte internal packet header

		// Calculate total packet size (header + data)
		totalPacketSize := 4 + len(packet.Data)

		// FFXI requires even-sized packets
		if totalPacketSize%2 != 0 {
			totalPacketSize++
		}

		packetBytes := make([]byte, totalPacketSize)

		// Pack the packet type and size
		// Bits 0-8: Packet type (9 bits)
		// Bits 9-15: Size in 2-byte units (7 bits)
		packetType := packet.Type & 0x1FF
		sizeUnits := uint16(totalPacketSize / 2)

		// First byte: low 8 bits of type
		packetBytes[0] = byte(packetType & 0xFF)
		// Second byte: high bit of type + size
		packetBytes[1] = byte(((packetType >> 8) & 0x01) | (sizeUnits << 1))

		// Sequence number
		binary.LittleEndian.PutUint16(packetBytes[2:4], session.lastServerPacketID)

		// Copy the actual packet data
		if len(packet.Data) > 0 {
			copy(packetBytes[4:], packet.Data)
		}

		// Debug log for login packet
		if packet.Type == 0x00A {
			s.Logger().Info("LOGIN PACKET CONSTRUCTION",
				"dataSize", len(packet.Data),
				"totalSize", totalPacketSize,
				"sizeUnits", sizeUnits,
				"header", fmt.Sprintf("%02X %02X %02X %02X",
					packetBytes[0], packetBytes[1], packetBytes[2], packetBytes[3]))
		}

		// Check if we have room for this packet
		if dataOffset+len(packetBytes) >= kMaxBufferSize {
			s.Logger().Warn("buffer size limit reached",
				"characterID", session.character.ID,
				"processedPackets", packetCount,
				"totalPackets", len(packets),
				"bufferUsed", dataOffset,
				"nextPacketSize", len(packetBytes))
			break
		}

		packetTypes = append(packetTypes, fmt.Sprintf("0x%03X", packet.Type))

		// Check for special packet types
		if packet.Type == 0x00B { // Logout/Zone packet
			incrementKeyAfterEncrypt = true
			s.Logger().Info("zone/logout packet detected",
				"characterID", session.character.ID,
				"packetType", fmt.Sprintf("0x%03X", packet.Type))
		}

		// Log packet details
		s.Logger().Debug("adding packet to buffer",
			"index", i,
			"type", fmt.Sprintf("0x%03X", packet.Type),
			"size", len(packetBytes),
			"sequence", session.lastServerPacketID,
			"offset", dataOffset)

		// Copy packet to buffer
		copy(outBuffer[dataOffset:], packetBytes)
		dataOffset += len(packetBytes)
		packetCount++

		// Limit packets per compression
		if packetCount >= kMaxPacketPerCompression {
			s.Logger().Debug("max packets per compression reached",
				"limit", kMaxPacketPerCompression,
				"processed", packetCount)
			break
		}
	}

	// === IMPORTANT: Add padding/terminator ===
	// The client expects either more packets or a terminator
	// A size byte of 0x00 indicates end of packets
	if dataOffset < kMaxBufferSize {
		outBuffer[dataOffset] = 0x00 // Terminator
		dataOffset++
	}

	s.Logger().Info("packets combined",
		"characterID", session.character.ID,
		"packetsCombined", packetCount,
		"totalDataSize", dataOffset-mapPackets.HeaderSize,
		"packetTypes", packetTypes)

	// Compress the data section (everything after the header)
	if s.codec == nil {
		return fmt.Errorf("ffxizlib codec not initialized")
	}

	dataToCompress := outBuffer[mapPackets.HeaderSize:dataOffset]
	originalSize := len(dataToCompress)
	compressedBuffer := make([]byte, kMaxBufferSize)
	bitCount, err := s.codec.Compress(dataToCompress, compressedBuffer)
	if err != nil {
		s.Logger().Error("compression failed",
			"characterID", session.character.ID,
			"originalSize", originalSize,
			"error", err)
		return fmt.Errorf("failed to compress packet data: %w", err)
	}

	compressedBytes := int(ffxizlib.CompressedSize(bitCount))
	if compressedBytes <= 0 || compressedBytes > len(compressedBuffer) {
		return fmt.Errorf("invalid compressed size %d", compressedBytes)
	}

	compressionRatio := float64(compressedBytes) / float64(originalSize) * 100
	s.Logger().Debug("data compressed",
		"originalSize", originalSize,
		"compressedSize", compressedBytes,
		"compressionRatio", fmt.Sprintf("%.1f%%", compressionRatio))

	// Build the compressed packet structure
	// Format: [compressed_data:N][compressed_size:4][md5:16]
	scratchBuffer := make([]byte, compressedBytes+4+mapPackets.MD5ChecksumSize)
	copy(scratchBuffer[:compressedBytes], compressedBuffer[:compressedBytes])
	binary.LittleEndian.PutUint32(scratchBuffer[compressedBytes:compressedBytes+4], bitCount)

	md5Data := scratchBuffer[:compressedBytes+4]
	hash := md5.Sum(md5Data)
	copy(scratchBuffer[compressedBytes+4:], hash[:])

	s.Logger().Debug("MD5 hash calculated",
		"hashHex", fmt.Sprintf("%x", hash))

	// Copy compressed + hashed data back to output buffer
	finalDataSize := compressedBytes + 4 + mapPackets.MD5ChecksumSize
	copy(outBuffer[mapPackets.HeaderSize:], scratchBuffer[:finalDataSize])

	// Encrypt in 8-byte blocks (pairs of uint32s)
	// CRITICAL FIX: Match C++ encryption exactly
	// C++ code: for (uint32 j = 0; j < CypherSize; j += 2)
	//           blowfish_encipher((uint32*)(buff) + j + 7, (uint32*)(buff) + j + 8, ...)
	// The +7 means starting at uint32 position 7, which is byte position 28 (after header)

	s.Logger().Info("ENCRYPTION TEST",
		"dataAtPos28", hex.EncodeToString(outBuffer[28:36]),
		"dataAtPos32", hex.EncodeToString(outBuffer[32:40]),
		"dataAtPos36", hex.EncodeToString(outBuffer[36:44]))
	originalData := make([]byte, 8)
	copy(originalData, outBuffer[28:36])

	// Calculate the number of uint32 pairs to encrypt
	cypherSize := (finalDataSize / 4) & ^1 // Round down to even number of uint32s

	if session.currentBlowfish != nil && session.currentBlowfish.cipher != nil {
		encryptedBlocks := 0

		// The C++ loop processes uint32 pairs, incrementing by 2 each time
		for j := uint32(0); j < uint32(cypherSize); j += 2 {
			// In C++: (uint32*)(buff) + j + 7 points to uint32 at index (j+7)
			// This translates to byte position (j+7)*4
			// Since j starts at 0, first position is 7*4 = 28 (right after header)
			bytePos := (j + 7) * 4

			// Make sure we don't go out of bounds
			if bytePos+8 <= uint32(len(outBuffer)) {
				// Encrypt 8 bytes (2 uint32s) at this position
				session.currentBlowfish.EncryptECB(outBuffer[bytePos : bytePos+8])
				encryptedBlocks++
			}
		}

		s.Logger().Debug("data encrypted",
			"encryptedBlocks", encryptedBlocks,
			"cypherSize", cypherSize,
			"totalEncryptedBytes", encryptedBlocks*8)
	} else {
		s.Logger().Warn("no blowfish cipher available",
			"characterID", session.character.ID)
	}

	s.Logger().Info("AFTER ENCRYPTION",
		"dataAtPos28", hex.EncodeToString(outBuffer[28:36]),
		"dataAtPos32", hex.EncodeToString(outBuffer[32:40]),
		"dataAtPos36", hex.EncodeToString(outBuffer[36:44]),
		"changed", !bytes.Equal(originalData, outBuffer[28:36]))

	// Calculate final buffer size
	totalSize := mapPackets.HeaderSize + finalDataSize

	// === FIX: Save packet for potential resend ===
	// The client might not ACK this packet and we'll need to resend it
	session.lastServerPacket = make([]byte, totalSize)
	copy(session.lastServerPacket, outBuffer[:totalSize])
	session.lastServerPacketSize = totalSize

	// Handle key increment for zone transitions
	if incrementKeyAfterEncrypt {
		s.Logger().Info("incrementing blowfish key for zone transition",
			"characterID", session.character.ID,
			"oldStatus", session.currentBlowfish.status)

		if err := session.IncrementBlowfish(); err != nil {
			s.Logger().Error("failed to increment blowfish key",
				"characterID", session.character.ID,
				"error", err)
		} else {
			session.currentBlowfish.status = BlowfishPendingZone
			s.Logger().Debug("blowfish key incremented and saved",
				"characterID", session.character.ID,
				"newStatus", BlowfishPendingZone)
		}
	}

	// Increment server packet ID for next packet
	oldPacketID := session.lastServerPacketID
	session.lastServerPacketID++

	s.Logger().Info("packet prepared successfully",
		"characterID", session.character.ID,
		"totalSize", totalSize,
		"oldServerPacketID", oldPacketID,
		"newServerPacketID", session.lastServerPacketID,
		"packetsIncluded", packetCount,
		"keyIncremented", incrementKeyAfterEncrypt)

	// Send the packet via UDP
	_, err = s.Socket().WriteToUDP(outBuffer[:totalSize], session.clientAddr)
	if err != nil {
		s.Logger().Error("failed to send packet",
			"clientAddr", clientAddr,
			"error", err)
		return fmt.Errorf("failed to send packet to client %s: %w", clientAddr, err)
	}

	s.Logger().Debug("packet sent successfully",
		"clientAddr", clientAddr,
		"bytesSent", totalSize)

	return nil
}

// SerializePacketForSending properly formats a packet with its internal header
func SerializePacketForSending(packet *mapPackets.BasicPacket, sequence uint16) []byte {
	// Determine total size (must be even for FFXI)
	dataSize := len(packet.Data)
	totalSize := 4 + dataSize // 4 bytes header + data
	if totalSize%2 != 0 {
		totalSize++ // Pad to even size
	}

	result := make([]byte, totalSize)

	// Pack the packet type and size
	packetType := packet.Type & 0x1FF
	sizeUnits := uint16(totalSize / 2)

	// First byte: low 8 bits of type
	result[0] = byte(packetType & 0xFF)
	// Second byte: high bit of type + size
	result[1] = byte(((packetType >> 8) & 0x01) | (sizeUnits << 1))

	// Sequence number
	binary.LittleEndian.PutUint16(result[2:4], sequence)

	// Copy data
	copy(result[4:], packet.Data)

	return result
}
