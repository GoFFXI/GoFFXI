package server

type ServerPacket interface {
	Type() uint16
	Size() uint16
	Serialize() ([]byte, error)
}
