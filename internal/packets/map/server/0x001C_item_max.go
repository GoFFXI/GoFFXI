package server

import mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"

const (
	PacketTypeItemMax = 0x001C
	PacketSizeItemMax = 0x0064
)

// https://github.com/atom0s/XiPackets/tree/main/world/server/0x001C
type ItemMaxPacket struct {
	Header mapPackets.PacketHeader

	// The characters various inventory container sizes.
	//
	// This array holds the maximum space of a container. The client uses the
	// values from this array to determine how large containers are and how large
	// to loop over items within a container. This is handled by the game client
	// function gcItemMaxSpaceGet which will return the value for the given container
	// index that is stored in this array.
	//
	// Containers hold 81 items total, however the first item of every container is
	// hidden and used to represent the characters gil. Because of this, the client
	// will expect a container to have > 1 size in order to be valid to hold items
	// and have space.
	//
	// If a container is set to 0, it will be considered locked.
	ItemNum [18]uint8

	// Padding; unused
	Padding16 [14]uint8

	// The characters various inventory container sizes.
	//
	// This array holds the available space of a container, however the client does
	// not use this array much at all. It is only used to check specific containers
	// and with specific conditions.
	ItemNum2 [18]uint16

	// Padding; unused
	Padding48 [28]uint8
}
