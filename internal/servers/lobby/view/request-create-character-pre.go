package view

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode"

	"github.com/GoFFXI/GoFFXI/internal/lobby/packets"
)

const (
	CommandRequestCreateCharacterPre  = 0x0022
	CommandResponseCharacterCreatePre = 0x0003
)

type RequestCreateCharacterPre struct {
	FFXIID         uint32
	CharacterName  [16]byte
	Password       [16]byte
	WorldName      [16]byte
	FriendPassword [16]byte
}

func NewRequestCreateCharacterPre(data []byte) (*RequestCreateCharacterPre, error) {
	if len(data) < 80 {
		return nil, fmt.Errorf("insufficient data for RequestCreateCharacterPre: need at least 80 bytes, got %d", len(data))
	}

	data = data[28:]
	request := &RequestCreateCharacterPre{}
	buf := bytes.NewReader(data)

	// Read the entire struct at once (works because all fields are fixed-size)
	err := binary.Read(buf, binary.BigEndian, request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (s *ViewServer) handleRequestCreateCharacterPre(sessionCtx *sessionContext, request []byte) bool {
	logger := sessionCtx.logger.With("request", "create-character-pre")
	logger.Info("handling request")

	req, err := NewRequestCreateCharacterPre(request)
	if err != nil {
		logger.Error("failed to parse request", "error", err)
		s.sendErrorResponse(sessionCtx, packets.ErrorCodeIncorrectCharacterParameters)

		return true
	}

	// make sure the character name only contains alphabetic characters
	characterName := string(bytes.TrimRight(req.CharacterName[:], "\x00"))
	for _, char := range characterName {
		if !unicode.IsLetter(char) {
			logger.Warn("character name contains non-alphabetic characters")
			s.sendErrorResponse(sessionCtx, packets.ErrorCodeCharacterNameInvalid)

			return false
		}
	}

	// make sure the character name length is valid
	if len(characterName) < 3 || len(characterName) > 15 {
		logger.Warn("character name length invalid")
		s.sendErrorResponse(sessionCtx, packets.ErrorCodeCharacterNameInvalid)

		return false
	}

	// name must be available
	exists, err := s.DB().CharacterNameExists(sessionCtx.ctx, characterName)
	if err != nil {
		logger.Error("failed to check if character name exists", "error", err)
		s.sendErrorResponse(sessionCtx, packets.ErrorCodeFailedToRegisterWithNameServer)

		return true
	}

	if exists {
		logger.Warn("character name already in use", "name", characterName)
		s.sendErrorResponse(sessionCtx, packets.ErrorCodeCharacterNameInvalid)

		return true
	}

	// make sure the world name matches
	worldName := string(bytes.TrimRight(req.WorldName[:], "\x00"))
	if worldName != s.Config().WorldName {
		logger.Warn("invalid world name", "worldName", worldName)
		s.sendErrorResponse(sessionCtx, packets.ErrorCodeFailedToRegisterWithNameServer)

		return false
	}

	// todo: name must not be an npc name
	// todo: name must not be on the bad word list

	// all checks passed, send OK response
	logger.Info("character name is valid and available", "name", characterName)
	s.sendOKResponse(sessionCtx)

	// for whatever reason, the game will not send this information again, so we need
	// to store it in the session context for later use
	sessionCtx.requestedCharacterName = characterName
	return false
}
