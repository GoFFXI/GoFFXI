package view

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// https://github.com/atom0s/XiPackets/blob/main/lobby/Header.md
type RequestHeader struct {
	// Header (28 bytes)
	PacketSize uint32
	Terminator uint32 // Always 0x46465849 ("IXFF")
	Command    uint32
	Identifier [16]byte // MD5 hash of the packet (calculated with this field as zeros)
}

func NewRequestHeader(request []byte) (*RequestHeader, error) {
	// Check minimum size
	if len(request) < 28 {
		return nil, fmt.Errorf("insufficient data: need 28 bytes, got %d", len(request))
	}

	header := &RequestHeader{}
	buf := bytes.NewReader(request)

	// Read the entire struct at once (works because all fields are fixed-size)
	err := binary.Read(buf, binary.LittleEndian, header)
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	return header, nil
}
