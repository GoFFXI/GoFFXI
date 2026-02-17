package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	PacketTypeGrapList = 0x0051
	// Payload size only; the sub-packet header (4 bytes) is added by the router.
	PacketSizeGrapList = 0x0014
)

// https://github.com/atom0s/XiPackets/tree/main/world/server/0x0051
type GrapListPacket struct {
	// The clients equipment model visual ids.
	//
	// 0 = race/hair, 1 = head, 2 = body, 3 = hands, 4 = legs,
	// 5 = feet, 6 = main hand, 7 = sub hand, 8 = ranged.
	GrapIDTbl [9]uint16

	// Padding; unused.
	Padding16 uint16
}

func (p *GrapListPacket) Type() uint16 {
	return PacketTypeGrapList
}

func (p *GrapListPacket) Size() uint16 {
	return PacketSizeGrapList
}

func (p *GrapListPacket) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write all fields in order
	if err := binary.Write(buf, binary.LittleEndian, p); err != nil {
		return nil, fmt.Errorf("failed to write packet: %w", err)
	}

	return buf.Bytes(), nil
}
