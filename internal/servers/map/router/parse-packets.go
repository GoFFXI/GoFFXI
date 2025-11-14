package router

import (
	"encoding/binary"
	"fmt"
	"time"

	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
	clientPackets "github.com/GoFFXI/GoFFXI/internal/packets/map/client"
)

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
			if remainingBytes >= 4 && packetType == clientPackets.PacketTypeLogin {
				s.Logger().Debug("detected login packet, using remaining buffer", "remainingBytes", remainingBytes)
				packetSizeBytes = remainingBytes
			} else {
				s.Logger().Warn("truncated packet detected, stopping parse",
					"type", fmt.Sprintf("0x%03X", packetType),
					"remaining", remainingBytes)
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
