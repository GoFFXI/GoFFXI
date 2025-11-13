package view

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"

	"github.com/GoFFXI/GoFFXI/internal/constants"
	"github.com/GoFFXI/GoFFXI/internal/packets/lobby"
)

const (
	CommandResponseOK = 0x0003
)

type ResponseOK struct {
	Header lobby.PacketHeader

	_ uint32 // Padding to make total size 32 bytes
}

func NewResponseOK() (*ResponseOK, error) {
	response := &ResponseOK{
		Header: lobby.PacketHeader{
			PacketSize: 0x0020, // Fixed size for this packet
			Terminator: constants.ResponsePacketTerminator,
			Command:    CommandResponseOK,
			Identifier: [16]byte{}, // Will be filled with MD5 hash
		},
	}

	// Calculate MD5 hash
	if err := response.CalculateAndSetHash(); err != nil {
		return nil, err
	}

	return response, nil
}

// CalculateAndSetHash calculates the MD5 hash of the packet and sets the identifier
func (r *ResponseOK) CalculateAndSetHash() error {
	// Temporarily clear identifier
	r.Header.Identifier = [16]byte{}

	// Serialize to calculate hash
	data, err := r.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize for hash: %w", err)
	}

	// Calculate and set MD5 hash
	hash := md5.Sum(data)
	r.Header.Identifier = hash

	return nil
}

// Serialize converts the packet to bytes for transmission
func (r *ResponseOK) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write all fields in order
	if err := binary.Write(buf, binary.LittleEndian, r); err != nil {
		return nil, fmt.Errorf("failed to write packet: %w", err)
	}

	return buf.Bytes(), nil
}

func (s *ViewServer) sendOKResponse(sessionCtx *sessionContext) {
	response, err := NewResponseOK()
	if err != nil {
		return
	}

	responsePacket, err := response.Serialize()
	if err != nil {
		return
	}

	// Send the response packet
	_, _ = sessionCtx.conn.Write(responsePacket)
}
