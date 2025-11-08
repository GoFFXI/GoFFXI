package view

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/davecgh/go-spew/spew"

	"github.com/GoFFXI/GoFFXI/internal/database"
)

const (
	CommandRequestSelectCharacter = 0x0007
)

type RequestSelectCharacter struct {
	FFXIID           uint32
	FFXIIDWorld      uint32
	CharacterName    [16]byte
	Password         [16]byte
	Unknown0000      uint32
	AuthCodeChecksum [16]byte
}

func NewRequestSelectCharacter(data []byte) (*RequestSelectCharacter, error) {
	if len(data) != 0x0058 {
		return nil, fmt.Errorf("invalid data length for RequestSelectCharacter: expected 88 bytes, got %d", len(data))
	}

	data = data[28:]
	request := &RequestSelectCharacter{}
	buf := bytes.NewReader(data)

	// Read the entire struct at once (works because all fields are fixed-size)
	err := binary.Read(buf, binary.BigEndian, request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (s *ViewServer) handleRequestSelectCharacter(_ *sessionContext, _ *database.AccountSession, data []byte) bool {
	logger := s.Logger().With("request", "select-character")
	logger.Info("handling request")

	req, err := NewRequestSelectCharacter(data)
	if err != nil {
		logger.Error("failed to parse request", "error", err)

		return true
	}

	spew.Dump(req)
	return false
}
