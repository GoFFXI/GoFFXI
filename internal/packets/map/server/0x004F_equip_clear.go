package server

import mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"

const (
	PacketTypeEquipClear = 0x004F
	PacketSizeEquipClear = 0x0008
)

// https://github.com/atom0s/XiPackets/tree/main/world/server/0x004F
type PacketEquipClear struct {
	Header mapPackets.PacketHeader

	// Padding; unused.
	Padding04 uint32
}
