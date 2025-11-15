package router

import (
	"context"
	"net"

	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
	"github.com/GoFFXI/GoFFXI/internal/servers/base/udp"
	"github.com/GoFFXI/GoFFXI/internal/tools/zlib"
)

type MapRouterServer struct {
	*udp.UDPServer

	sessions      map[string]*Session
	packetsToSend map[string][]*mapPackets.RoutedPacket
	codec         *zlib.FFXICodec
}

func NewMapRouterServer(baseServer *udp.UDPServer) *MapRouterServer {
	var codec *zlib.FFXICodec
	if baseServer != nil && baseServer.Config() != nil {
		codec = zlib.NewCodec(baseServer.Config().FFXIResourcePath)
	} else {
		codec = zlib.NewCodec("")
	}

	return &MapRouterServer{
		UDPServer:     baseServer,
		sessions:      make(map[string]*Session),
		packetsToSend: make(map[string][]*mapPackets.RoutedPacket),
		codec:         codec,
	}
}

func (s *MapRouterServer) HandleIncomingPacket(ctx context.Context, length int, data []byte, clientAddr *net.UDPAddr) {
	clientAddrStr := clientAddr.String()
	s.Logger().Info("handling incoming packet", "dataLength", length, "clientAddr", clientAddrStr)

	//
}
