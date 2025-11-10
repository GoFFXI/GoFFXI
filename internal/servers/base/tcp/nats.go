package tcp

import (
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

func (s *TCPServer) CreateNATSConnection() error {
	hostname, _ := os.Hostname()

	// create a new NATS connection
	options := []nats.Option{
		nats.Name(fmt.Sprintf("%s%s", s.Config().NATSClientPrefix, hostname)),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2 * time.Second),
		nats.ReconnectBufSize(s.Config().NATSOutgoingBufferSize),
		nats.DisconnectErrHandler(s.OnNATSDisconnected),
		nats.ReconnectHandler(s.OnNATSReconnected),
		nats.ClosedHandler(s.OnNATSClosed),
	}

	// connect to NATS server
	nc, err := nats.Connect(s.Config().NATSURL, options...)
	if err != nil {
		return err
	}

	s.natsConn = nc
	return nil
}

func (s *TCPServer) OnNATSDisconnected(_ *nats.Conn, err error) {
	s.Logger().Warn("NATS disconnected", "error", err)
	s.natsConn = nil
}

func (s *TCPServer) OnNATSReconnected(nc *nats.Conn) {
	s.Logger().Info("NATS reconnected")
	s.natsConn = nc
}

func (s *TCPServer) OnNATSClosed(_ *nats.Conn) {
	s.Logger().Info("NATS connection permanently closed")
	s.natsConn = nil
}
