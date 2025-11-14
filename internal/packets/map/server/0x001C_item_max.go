package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type ContainerKind uint8

const (
	ItemMaxPacketType = 0x001C
	ItemMaxPacketSize = 0x0064
)

// Container indices for FFXI inventory system
const (
	ContainerKindInventory ContainerKind = iota // Main inventory
	ContainerKindMogSafe
	ContainerKindStorage
	ContainerKindTempItems
	ContainerKindMogLocker
	ContainerKindMogSatchel
	ContainerKindMogSack
	ContainerKindMogCase
	ContainerKindMogWardrobe
	ContainerKindMogSafe2
	ContainerKindMogWardrobe2
	ContainerKindMogWardrobe3
	ContainerKindMogWardrobe4
	ContainerKindMogWardrobe5
	ContainerKindMogWardrobe6
	ContainerKindMogWardrobe7
	ContainerKindMogWardrobe8
	ContainerKindRecycleBin
)

// https://github.com/atom0s/XiPackets/tree/main/world/server/0x001C
type ItemMaxPacket struct {
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

func (p *ItemMaxPacket) Type() uint16 {
	return ItemMaxPacketType
}

func (p *ItemMaxPacket) Size() uint16 {
	return ItemMaxPacketSize
}

func (p *ItemMaxPacket) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write all fields in order
	if err := binary.Write(buf, binary.LittleEndian, p); err != nil {
		return nil, fmt.Errorf("failed to write packet: %w", err)
	}

	return buf.Bytes(), nil
}
