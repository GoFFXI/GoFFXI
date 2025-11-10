package view

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/GoFFXI/GoFFXI/internal/database"
)

const (
	CommandRequestGetCharacter = 0x001F
)

// https://github.com/atom0s/XiPackets/blob/main/lobby/C2S_0x001F_RequestGetChr.md
type RequestGetCharacter struct {
	Password [16]byte
}

func NewRequestGetCharacter(data []byte) (*RequestGetCharacter, error) {
	// strip the header (28 bytes)
	if len(data) < 29 {
		return nil, fmt.Errorf("insufficient data for RequestGetCharacter: need at least 29 bytes, got %d", len(data))
	}

	data = data[28:] // strip the header (28 bytes)
	request := &RequestGetCharacter{}
	buf := bytes.NewReader(data)

	// Read the entire struct at once (works because all fields are fixed-size)
	err := binary.Read(buf, binary.BigEndian, request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (s *ViewServer) handleRequestGetCharacter(sessionCtx *sessionContext, accountSession *database.AccountSession, request []byte) bool {
	logger := sessionCtx.logger.With("request", "get-character")
	logger.Info("handling request")

	_, err := NewRequestGetCharacter(request)
	if err != nil {
		logger.Error("failed to parse request", "error", err)
		return true
	}

	// this is a bit of a weird one - the request should actually trigger the data server to send a success response (0x01)
	// no player data is actually sent here, the data server handles that separately
	logger.Info("instructing data server to generate character data")
	_ = s.NATS().Publish(fmt.Sprintf("session.%s.data.send", accountSession.SessionKey), []byte{0x01})

	return false
}
