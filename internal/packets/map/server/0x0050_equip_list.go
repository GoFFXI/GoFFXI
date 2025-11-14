package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type EquipKind uint8

const (
	EquipListPacketType = 0x0050
	EquipListPacketSize = 0x0008
)

const (
	EquipKindMain EquipKind = iota
	EquipKindSub
	EquipKindRanged
	EquipKindAmmo
	EquipKindHead
	EquipKindBody
	EquipKindHands
	EquipKindLegs
	EquipKindFeet
	EquipKindNeck
	EquipKindWaist
	EquipKindRightEar
	EquipKindLeftEar
	EquipKindRightRing
	EquipKindLeftRing
	EquipKindBack
	EquipKindEnd
)

// https://github.com/atom0s/XiPackets/tree/main/world/server/0x0050
type EquipListPacket struct {
	// The index of the item within the container.
	PropertyItemIndex uint8

	// The equipment slot enumeration id.
	//
	// Note: This is aligned as a single byte!
	EquipKind EquipKind

	// The container holding the item being equipped.
	Category uint8

	// Padding; unused.
	Padding07 uint8
}

func (p *EquipListPacket) Type() uint16 {
	return EquipListPacketType
}

func (p *EquipListPacket) Size() uint16 {
	return EquipListPacketSize
}

func (p *EquipListPacket) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write all fields in order
	if err := binary.Write(buf, binary.LittleEndian, p); err != nil {
		return nil, fmt.Errorf("failed to write packet: %w", err)
	}

	return buf.Bytes(), nil
}
