package server

import mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"

type EquipKind uint8

const (
	PacketTypeEquipList = 0x0050
	PacketSizeEquipList = 0x0008

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
	Header mapPackets.PacketHeader

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
