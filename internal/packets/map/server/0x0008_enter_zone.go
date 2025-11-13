package server

import mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"

const (
	PacketTypeEnterZone = 0x0008
	PacketSizeEnterZone = 0x0034
)

// https://github.com/atom0s/XiPackets/tree/main/world/server/0x0008
type EnterZonePacket struct {
	Header mapPackets.PacketHeader

	// Array holding the previous entered zone information, in bits.
	EnterZoneTbl [48]uint8
}
