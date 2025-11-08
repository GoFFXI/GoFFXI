package view

import (
	"bytes"
	"crypto/md5" //nolint:gosec // game has to have this
	"encoding/binary"
	"fmt"

	"github.com/GoFFXI/GoFFXI/internal/constants"
)

const (
	CommandResponseError = 0x0004

	ErrorCodeUnableToConnectToWorldServer   uint32 = 305
	ErrorCodeCharacterNameInvalid           uint32 = 313
	ErrorCodeCharacterAlreadyLoggedIn       uint32 = 201
	ErrorCodeFailedToRegisterWithNameServer uint32 = 314
	ErrorCodeIncorrectCharacterParameters   uint32 = 321
	ErrorCodeGameDataUpdated                uint32 = 331
	ErrorCodeUnableToConnectToLobbyServer   uint32 = 332
)

type ResponseError struct {
	// Header (28 bytes)
	PacketSize uint32   // Total packet size (40 bytes)
	Terminator uint32   // Always 0x46465849 ("IXFF")
	Command    uint32   // OpCode 0x0004
	Identifier [16]byte // MD5 hash of the packet

	Unknown0000 uint32 // Unknown / padding
	ErrorCode   uint32 // Error code
}

func NewResponseError(errorCode uint32) (*ResponseError, error) {
	response := &ResponseError{
		PacketSize: 0x0024, // Fixed size for this packet
		Terminator: constants.ResponsePacketTerminator,
		Command:    CommandResponseError,
		Identifier: [16]byte{}, // Will be filled with MD5 hash
		ErrorCode:  errorCode,
	}

	// Calculate MD5 hash
	if err := response.CalculateAndSetHash(); err != nil {
		return nil, err
	}

	return response, nil
}

// CalculateAndSetHash calculates the MD5 hash of the packet and sets the identifier
func (r *ResponseError) CalculateAndSetHash() error {
	// Temporarily clear identifier
	r.Identifier = [16]byte{}

	// Serialize to calculate hash
	data, err := r.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize for hash: %w", err)
	}

	// Calculate and set MD5 hash
	hash := md5.Sum(data) //nolint:gosec // game has to have this
	r.Identifier = hash

	return nil
}

// Serialize converts the packet to bytes for transmission
func (r *ResponseError) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write all fields in order
	if err := binary.Write(buf, binary.LittleEndian, r); err != nil {
		return nil, fmt.Errorf("failed to write packet: %w", err)
	}

	return buf.Bytes(), nil
}

func (s *ViewServer) sendErrorResponse(sessionCtx *sessionContext, errorCode uint32) {
	response, err := NewResponseError(errorCode)
	if err != nil {
		return
	}

	responsePacket, err := response.Serialize()
	if err != nil {
		return
	}

	// it's okay if this write fails, we're already in an error state
	_, _ = sessionCtx.conn.Write(responsePacket)
}
