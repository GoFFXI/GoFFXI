package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	EquipClearPacketType = 0x004F
	EquipClearPacketSize = 0x0008
)

// https://github.com/atom0s/XiPackets/tree/main/world/server/0x004F
type EquipClearPacket struct {
	// Padding; unused.
	Padding04 uint32
}

func (p *EquipClearPacket) Type() uint16 {
	return EquipClearPacketType
}

func (p *EquipClearPacket) Size() uint16 {
	return EquipClearPacketSize
}

func (p *EquipClearPacket) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write all fields in order
	if err := binary.Write(buf, binary.LittleEndian, p); err != nil {
		return nil, fmt.Errorf("failed to write packet: %w", err)
	}

	return buf.Bytes(), nil
}
