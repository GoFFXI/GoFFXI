package instance

import (
	"encoding/json"

	"github.com/davecgh/go-spew/spew"
	"github.com/nats-io/nats.go"

	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
	clientPackets "github.com/GoFFXI/GoFFXI/internal/packets/map/client"
)

func (s *InstanceWorker) ProcessPacket(msg *nats.Msg) {
	s.Logger().Debug("received packet", "natsSubject", msg.Subject)

	// attempt to unmarshal the packet data
	var routedPacket mapPackets.RoutedPacket
	if err := json.Unmarshal(msg.Data, &routedPacket); err != nil {
		s.Logger().Error("failed to unmarshal packet data; discarding", "error", err)
		return
	}

	// begin checking packet type
	switch routedPacket.Packet.Type {
	case clientPackets.PacketTypeLogin:
		s.processLoginPacket(routedPacket)
	default:
		s.Logger().Warn("received unhandled packet type", "packetType", routedPacket.Packet.Type)
		spew.Dump(routedPacket)
	}
}
