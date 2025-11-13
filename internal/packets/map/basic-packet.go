package mappackets

import "encoding/json"

type BasicPacket struct {
	Type     uint16
	Size     uint16
	Sequence uint16
	Data     []byte
}

type RoutedPacket struct {
	ClientAddr  string
	CharacterID uint32
	Packet      BasicPacket
}

func (rp *RoutedPacket) ToJSON() []byte {
	bytes, _ := json.Marshal(rp)
	return bytes
}
