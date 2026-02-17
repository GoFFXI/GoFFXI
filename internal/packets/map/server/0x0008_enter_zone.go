package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	EnterZonePacketType = 0x0008
	// Payload size only; the sub-packet header (4 bytes) is added by the router.
	EnterZonePacketSize = 0x0030
)

// https://github.com/atom0s/XiPackets/tree/main/world/server/0x0008
type EnterZonePacket struct {
	// Array holding the previous entered zone information, in bits.
	EnterZoneTbl [48]uint8
}

func (p *EnterZonePacket) Type() uint16 {
	return EnterZonePacketType
}

func (p *EnterZonePacket) Size() uint16 {
	return EnterZonePacketSize
}

func (p *EnterZonePacket) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write all fields in order
	if err := binary.Write(buf, binary.LittleEndian, p); err != nil {
		return nil, fmt.Errorf("failed to write packet: %w", err)
	}

	return buf.Bytes(), nil
}
