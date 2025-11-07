package view

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"

	"github.com/GoFFXI/login-server/internal/constants"
	"github.com/GoFFXI/login-server/internal/database"
	"github.com/GoFFXI/login-server/internal/packets"
)

const (
	CommandRequestCreateCharacter = 0x0021
)

type RequestCreateCharacter struct {
	FFXIID        uint32
	Password      [16]byte
	CharacterInfo packets.CharacterInfo
}

func NewRequestCreateCharacter(data []byte) (*RequestCreateCharacter, error) {
	if len(data) != 0x0090 {
		return nil, fmt.Errorf("invalid data length for RequestCreateCharacter: expected 144 bytes, got %d", len(data))
	}

	data = data[28:]
	request := &RequestCreateCharacter{}
	buf := bytes.NewReader(data)

	// Read the entire struct at once (works because all fields are fixed-size)
	err := binary.Read(buf, binary.BigEndian, request)
	if err != nil {
		return nil, err
	}

	// the client sends some fields out of order, so we need to manually set them
	request.CharacterInfo.RaceID = uint16(data[20])
	request.CharacterInfo.FaceModelID = uint16(data[32])
	return request, nil
}

func (s *ViewServer) handleRequestCreateCharacter(sessionCtx *sessionContext, session *database.AccountSession, data []byte) bool {
	logger := s.Logger().With("request", "create-character")
	logger.Info("handling request")

	req, err := NewRequestCreateCharacter(data)
	if err != nil {
		logger.Error("failed to parse request", "error", err)
		return true
	}

	// make sure the race is valid
	if req.CharacterInfo.RaceID < 1 || req.CharacterInfo.RaceID > 8 {
		logger.Warn("invalid race ID", "raceID", req.CharacterInfo.RaceID)
		s.sendErrorResponse(sessionCtx, ErrorCodeIncorrectCharacterParameters)
		return true
	}

	// make sure the size is valid
	if req.CharacterInfo.CharacterSize > 2 {
		logger.Warn("invalid size", "size", req.CharacterInfo.CharacterSize)
		s.sendErrorResponse(sessionCtx, ErrorCodeIncorrectCharacterParameters)
		return true
	}

	// make sure the face is valid
	if req.CharacterInfo.FaceModelID > 15 {
		logger.Warn("invalid face ID", "faceID", req.CharacterInfo.FaceModelID)
		s.sendErrorResponse(sessionCtx, ErrorCodeIncorrectCharacterParameters)
		return true
	}

	// make sure the job is a starting job
	if req.CharacterInfo.MainJobID < 1 || req.CharacterInfo.MainJobID > 6 {
		logger.Warn("invalid main job ID", "mainJobID", req.CharacterInfo.MainJobID)
		s.sendErrorResponse(sessionCtx, ErrorCodeIncorrectCharacterParameters)
		return true
	}

	// make sure the nation is valid
	if req.CharacterInfo.TownNumber > 2 {
		logger.Warn("invalid nation ID", "nationID", req.CharacterInfo.TownNumber)
		s.sendErrorResponse(sessionCtx, ErrorCodeIncorrectCharacterParameters)
		return true
	}

	// check how many characters the account has
	characterCount, err := s.DB().CountCharactersByAccountID(sessionCtx.ctx, session.AccountID)
	if err != nil {
		logger.Error("failed to count characters for account", "error", err)
		s.sendErrorResponse(sessionCtx, ErrorCodeFailedToRegisterWithNameServer)
		return true
	}

	// make sure the account hasn't reached the character limit
	if characterCount >= s.Config().MaxContentIDsPerAccount {
		logger.Warn("account has reached character limit", "accountID", session.AccountID, "characterCount", characterCount)
		s.sendErrorResponse(sessionCtx, ErrorCodeFailedToRegisterWithNameServer)
		return false
	}

	if err = s.saveNewCharacterToDatabase(sessionCtx.ctx, session.AccountID, sessionCtx.requestedCharacterName, &req.CharacterInfo); err != nil {
		logger.Error("failed to save new character to database", "error", err)
		s.sendErrorResponse(sessionCtx, ErrorCodeFailedToRegisterWithNameServer)
		return true
	}

	response, err := NewResponseOK()
	if err != nil {
		logger.Error("failed to create response", "error", err)
		return true
	}

	responsePacket, err := response.Serialize()
	if err != nil {
		logger.Error("failed to serialize response", "error", err)
		return true
	}

	_, _ = sessionCtx.conn.Write(responsePacket)
	return false
}

func (s *ViewServer) saveNewCharacterToDatabase(ctx context.Context, accountID uint32, characterName string, charInfo *packets.CharacterInfo) error {
	// first, create the character record
	character := &database.Character{
		AccountID: accountID,
		Name:      characterName,
		Nation:    charInfo.TownNumber,
		PosZone:   s.getRandomStartingZoneForNation(charInfo.TownNumber),
	}

	savedCharacter, err := s.DB().CreateCharacter(ctx, character)
	if err != nil {
		return fmt.Errorf("failed to create character in database: %w", err)
	}

	// next, create the character appearance record
	characterLooks := &database.CharacterLooks{
		CharacterID: savedCharacter.ID,
		Face:        uint8(charInfo.FaceModelID), //nolint:gosec // faceID is a number between 0 and 15
		Race:        uint8(charInfo.RaceID),      //nolint:gosec // raceID is a number between 1 and 8
		Size:        charInfo.CharacterSize,
	}

	_, err = s.DB().CreateCharacterLooks(ctx, characterLooks)
	if err != nil {
		return fmt.Errorf("failed to create character looks in database: %w", err)
	}

	// next, create the character stats record
	characterStats := &database.CharacterStats{
		CharacterID: savedCharacter.ID,
		MainJob:     charInfo.MainJobID,
	}

	_, err = s.DB().CreateCharacterStats(ctx, characterStats)
	if err != nil {
		return fmt.Errorf("failed to create character stats in database: %w", err)
	}

	// next, create the character jobs record
	characterJobs := &database.CharacterJobs{
		CharacterID: savedCharacter.ID,
	}

	_, err = s.DB().CreateCharacterJobs(ctx, characterJobs)
	if err != nil {
		return fmt.Errorf("failed to create character jobs in database: %w", err)
	}

	return nil

	// todo: extra tables:
	// - character exp
	// - character flags
	// - character jobs
	// - character points
	// - character unlocks
	// - character profile
	// - character storage
	// - character inventory
	// - character variables
}

func (s *ViewServer) getRandomStartingZoneForNation(nationID uint8) uint16 {
	var startingLocations []uint16

	switch nationID {
	case constants.TownSandoria:
		startingLocations = []uint16{0xE6, 0xE7, 0xE8}
	case constants.TownBastok:
		startingLocations = []uint16{0xEA, 0xEB, 0xEC}
	case constants.TownWindurst:
		startingLocations = []uint16{0xEE, 0xF0, 0xF1}
	}

	//nolint:gosec // Starting location selection is not security sensitive
	return startingLocations[rand.Intn(len(startingLocations))]
}
