package mappackets

const (
	HeaderSize = 0x001C
)

type PacketHeader struct {
	ID   uint16 // Contains both id (9 bits) and size (7 bits)
	Sync uint16 // Packet sync count
}

func (p *PacketHeader) GetPacketID() uint16 {
	return p.ID & 0x1FF // First 9 bits
}

func (p *PacketHeader) GetPacketSize() uint16 {
	return (p.ID >> 9) & 0x7F // Next 7 bits
}
