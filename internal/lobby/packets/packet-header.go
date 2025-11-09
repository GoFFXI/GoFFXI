package packets

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// https://github.com/atom0s/XiPackets/blob/main/lobby/Header.md
type PacketHeader struct {
	PacketSize uint32   // Total size of the packet
	Terminator uint32   // Always 0x46465849 ("IXFF")
	Command    uint32   // OpCode
	Identifier [16]byte // Identifier - must be exactly 16 bytes
}

func NewPacketHeader(request []byte) (*PacketHeader, error) {
	// Check minimum size
	if len(request) < 28 {
		return nil, fmt.Errorf("insufficient data: need 28 bytes, got %d", len(request))
	}

	header := &PacketHeader{}
	buf := bytes.NewReader(request)

	// Read the entire struct at once (works because all fields are fixed-size)
	err := binary.Read(buf, binary.LittleEndian, header)
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	return header, nil
}
