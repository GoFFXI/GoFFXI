package view

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/GoFFXI/GoFFXI/internal/database"
	"github.com/GoFFXI/GoFFXI/internal/packets/lobby"
)

const (
	CommandRequestDeleteCharacter = 0x0014
)

type RequestDeleteCharacter struct {
	FFXIID     uint32
	FFXIDWorld uint32
	Password   [16]byte
}

func NewRequestDeleteCharacter(data []byte) (*RequestDeleteCharacter, error) {
	if len(data) != 0x0034 {
		return nil, fmt.Errorf("insufficient data for RequestDeleteCharacter: need 52 bytes, got %d", len(data))
	}

	data = data[28:]
	request := &RequestDeleteCharacter{}
	buf := bytes.NewReader(data)

	// Read the entire struct at once (works because all fields are fixed-size)
	err := binary.Read(buf, binary.LittleEndian, request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (s *ViewServer) handleRequestDeleteCharacter(sessionCtx *sessionContext, accountSession *database.AccountSession, request []byte) bool {
	logger := sessionCtx.logger.With("request", "delete-character")
	logger.Info("handling request")

	req, err := NewRequestDeleteCharacter(request)
	if err != nil {
		logger.Error("failed to parse request", "error", err)
		s.sendErrorResponse(sessionCtx, lobby.ErrorCodeUnableToConnectToLobbyServer)

		return true
	}

	// get the account ID associated with this character
	character, err := s.DB().GetCharacterByID(sessionCtx.ctx, req.FFXIID)
	if err != nil {
		logger.Error("failed to lookup character", "characterID", req.FFXIID, "error", err)
		s.sendErrorResponse(sessionCtx, lobby.ErrorCodeUnableToConnectToLobbyServer)

		return true
	}

	// make sure this character belongs to the account
	if character.AccountID != accountSession.AccountID {
		logger.Error("character does not belong to account", "characterID", req.FFXIID, "accountID", accountSession.AccountID)

		return true
	}

	// next, replace the account ID to mark the character as deleted
	character.OriginalAccountID = character.AccountID
	character.AccountID = 0

	_, err = s.DB().UpdateCharacter(sessionCtx.ctx, &character)
	if err != nil {
		logger.Error("failed to delete character", "characterID", req.FFXIID, "error", err)
		s.sendErrorResponse(sessionCtx, lobby.ErrorCodeUnableToConnectToLobbyServer)

		return true
	}

	// finally, send a success response
	s.sendOKResponse(sessionCtx)

	return false
}
