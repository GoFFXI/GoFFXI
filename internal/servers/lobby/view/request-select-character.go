package view

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/GoFFXI/GoFFXI/internal/database"
)

const (
	CommandRequestSelectCharacter  = 0x0007
	CommandResponseSelectCharacter = 0x0002
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
	err := binary.Read(buf, binary.LittleEndian, request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (s *ViewServer) handleRequestSelectCharacter(sessionCtx *sessionContext, sessionKey string, _ *database.AccountSession, data []byte) bool {
	logger := s.Logger().With("request", "select-character")
	logger.Info("handling request")

	req, err := NewRequestSelectCharacter(data)
	if err != nil {
		logger.Error("failed to parse request", "error", err)

		return true
	}

	// lookup the character
	character, err := s.DB().GetCharacterByID(sessionCtx.ctx, req.FFXIID)
	if err != nil {
		logger.Error("failed to get character by ID", "error", err)

		return true
	}

	// make sure the character belongs to the account
	if character.AccountID != sessionCtx.accountID {
		logger.Warn("character does not belong to account", "characterAccountID", character.AccountID, "sessionAccountID", sessionCtx.accountID)

		return true
	}

	// now, attempt to see if the account is banned
	isBanned, err := s.DB().IsAccountBanned(sessionCtx.ctx, sessionCtx.accountID)
	if err != nil {
		logger.Error("failed to check if account is banned", "error", err)

		return true
	}

	if isBanned {
		logger.Info("account is banned, disconnecting")

		return true
	}

	// set the selected character in the session context
	sessionCtx.selectedCharacterID = req.FFXIID

	// the data server needs this context as well for when the character selection is confirmed
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, req.FFXIID)
	_ = s.NATS().Publish(fmt.Sprintf("session.%s.data.character.selectID", sessionKey), buf.Bytes())

	// the view server does not send a response for this request; it's
	// actually handled by the data server after the character selection is confirmed.
	var dataResponse [5]byte
	dataResponse[0] = CommandResponseSelectCharacter
	_ = s.NATS().Publish(fmt.Sprintf("session.%s.data.send", sessionKey), dataResponse[:])
	logger.Info("instructing data server to proceed with character selection")

	return false
}
