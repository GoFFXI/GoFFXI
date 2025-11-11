package maprouter

import (
	"context"
	"net"

	"github.com/GoFFXI/GoFFXI/internal/servers/base/udp"
)

type MapRouterServer struct {
	*udp.UDPServer
}

func (s *MapRouterServer) HandlePacket(_ context.Context, length int, _ []byte, clientAddr *net.UDPAddr) {
	s.Logger().Info("received packet", "clientAddr", clientAddr.String(), "dataLength", length)
}
