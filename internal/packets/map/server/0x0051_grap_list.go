package server

import mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"

const (
	PacketTypeGrapList = 0x0051
	PacketSizeGrapList = 0x0018
)

// https://github.com/atom0s/XiPackets/tree/main/world/server/0x0051
type GrapListPacket struct {
	Header mapPackets.PacketHeader

	// The clients equipment model visual ids.
	//
	// 0 = race/hair, 1 = head, 2 = body, 3 = hands, 4 = legs,
	// 5 = feet, 6 = main hand, 7 = sub hand, 8 = ranged.
	GrapIDTbl [9]uint16

	// Padding; unused.
	Padding16 uint16
}
