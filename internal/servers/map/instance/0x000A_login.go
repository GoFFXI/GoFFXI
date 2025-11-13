package instance

import mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"

func (s *InstanceWorker) processLoginPacket(routedPacket mapPackets.RoutedPacket) {
	s.Logger().Info("processing login packet", "clientAddr", routedPacket.ClientAddr)

	// we need to respond with 5 packets:
	// 0x001C - Item Max
	// 0x004F - Equip Clear
	// 0x0008 - Enter Zone
	// 0x0050 - Equip List
	// 0x0051 - Grap List
}
