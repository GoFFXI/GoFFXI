package data

import (
	"bytes"
	"crypto/md5" //nolint:gosec // we are using MD5 for compatibility with FFXI protocol, not for security
	"encoding/binary"
	"fmt"

	"github.com/GoFFXI/GoFFXI/internal/constants"
	"github.com/GoFFXI/GoFFXI/internal/lobby/packets"
	"github.com/GoFFXI/GoFFXI/internal/tools"
)

const (
	CommandRequestSelectCharacter  = 0x00A2
	CommandResponseSelectCharacter = 0x000B
)

type ResponseSelectCharacter struct {
	Header packets.PacketHeader

	FFXIID           uint32
	FFXIDWorld       uint32
	CharacterName    [16]byte
	ServerID         uint32
	MapServerIP      uint32
	MapServerPort    uint32
	SearchServerIP   uint32
	SearchServerPort uint32
}

func NewResponseSelectCharacter() *ResponseSelectCharacter {
	return &ResponseSelectCharacter{
		Header: packets.PacketHeader{
			PacketSize: 0x0048,
			Command:    CommandResponseSelectCharacter,
			Terminator: constants.ResponsePacketTerminator,
		},
	}
}

func (p *ResponseSelectCharacter) Serialize() ([]byte, error) { //nolint:gocyclo // it's acceptable because of the multiple writes in specific orders
	buf := new(bytes.Buffer)

	// Write header fields individually
	if err := binary.Write(buf, binary.LittleEndian, p.Header.PacketSize); err != nil {
		return nil, fmt.Errorf("failed to write packet size: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, p.Header.Terminator); err != nil {
		return nil, fmt.Errorf("failed to write terminator: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, p.Header.Command); err != nil {
		return nil, fmt.Errorf("failed to write command: %w", err)
	}

	// Write identifier as bytes (must be exactly 16 bytes)
	// If empty, write zeros (will be filled with hash later)
	if len(p.Header.Identifier[:]) == 0 {
		if _, err := buf.Write(make([]byte, 16)); err != nil {
			return nil, fmt.Errorf("failed to write empty identifier: %w", err)
		}
	} else {
		if len(p.Header.Identifier) != 16 {
			return nil, fmt.Errorf("identifier must be exactly 16 bytes, got %d", len(p.Header.Identifier))
		}
		if _, err := buf.Write(p.Header.Identifier[:]); err != nil {
			return nil, fmt.Errorf("failed to write identifier: %w", err)
		}
	}

	// now write the rest of the fields
	if err := binary.Write(buf, binary.LittleEndian, p.FFXIID); err != nil {
		return nil, fmt.Errorf("failed to write FFXIID: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, p.FFXIDWorld); err != nil {
		return nil, fmt.Errorf("failed to write FFXIDWorld: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, p.CharacterName); err != nil {
		return nil, fmt.Errorf("failed to write CharacterName: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, p.ServerID); err != nil {
		return nil, fmt.Errorf("failed to write ServerID: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, p.MapServerIP); err != nil {
		return nil, fmt.Errorf("failed to write MapServerIP: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, p.MapServerPort); err != nil {
		return nil, fmt.Errorf("failed to write MapServerPort: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, p.SearchServerIP); err != nil {
		return nil, fmt.Errorf("failed to write SearchServerIP: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, p.SearchServerPort); err != nil {
		return nil, fmt.Errorf("failed to write SearchServerPort: %w", err)
	}

	return buf.Bytes(), nil
}

func (p *ResponseSelectCharacter) CalculateAndSetHash() error {
	p.Header.Identifier = [16]byte{}

	// serialize to calculate hash
	data, err := p.Serialize()
	if err != nil {
		return err
	}

	// calculate and set MD5 hash
	hash := md5.Sum(data) //nolint:gosec // game has to have this
	p.Header.Identifier = hash

	return nil
}

func (s *DataServer) handleRequestSelectCharacter(sessionCtx *sessionContext, data []byte) bool {
	logger := s.Logger().With("request", "select-character")
	logger.Info("handling request")

	// extract the magic blowfish key from the request
	var magicKey [20]uint8
	copy(magicKey[:], data[1:21])
	logger.Info("extracted magic key", "magicKey", magicKey)

	// make sure there is a session account ID
	if sessionCtx.accountID == 0 {
		logger.Error("no valid session account ID")
		s.sendErrorResponse(sessionCtx)

		return true
	}

	// check if it's the first time logging in after character creation
	if sessionCtx.freshCharacterLogin {
		logger.Info("first time logging in after character creation")
		magicKey[16] += 6
	}

	// ??? always increment the magic key's 16th byte by the increment value (1)
	// todo: investigate this value further:
	// https://github.com/LandSandBoat/server/blob/b3cb68560fb055b5696b0399d28e2b8972282338/src/login/data_session.cpp#L393
	// magicKey[16] += 0

	// todo: make sure there are no other active sessions for this account by checking their ip address
	// (unless they have an ip exception which allows multiple logins per ip address)

	// fetch the character
	character, err := s.DB().GetCharacterByID(sessionCtx.ctx, sessionCtx.selectedCharacterID)
	if err != nil {
		logger.Error("failed to fetch character", "error", err)
		s.sendErrorResponse(sessionCtx)

		return true
	}

	// make sure the character belongs to the account
	if character.AccountID != sessionCtx.accountID {
		logger.Error("character does not belong to account", "characterID", character.ID, "accountID", sessionCtx.accountID)
		s.sendErrorResponse(sessionCtx)

		return true
	}

	// update the character's new previous zone to be their current zone
	logger.Info("updating character previous zone", "characterID", character.ID, "from", character.PosPrevZone, "to", character.PosZone)
	character.PosPrevZone = character.PosZone
	_, err = s.DB().UpdateCharacter(sessionCtx.ctx, &character)
	if err != nil {
		logger.Error("failed to update character", "error", err)
		s.sendErrorResponse(sessionCtx)

		return true
	}

	// build a response packet
	response := NewResponseSelectCharacter()
	response.FFXIID = character.ID
	response.FFXIDWorld = character.ID & 0xFFFF
	response.ServerID = (character.ID >> 16) & 0xFF
	response.MapServerIP = tools.StringToIP(s.Config().MapServerIP)
	response.MapServerPort = s.Config().MapServerPort
	response.SearchServerIP = tools.StringToIP(s.Config().SearchServerIP)
	response.SearchServerPort = s.Config().SearchServerPort

	// copy character name into response
	response.CharacterName = [16]byte{}
	copy(response.CharacterName[:], character.Name)

	// calculate and set the response packet hash
	err = response.CalculateAndSetHash()
	if err != nil {
		logger.Error("failed to calculate response packet hash", "error", err)
		s.sendErrorResponse(sessionCtx)

		return true
	}

	// serialize the response packet
	responseData, err := response.Serialize()
	if err != nil {
		logger.Error("failed to serialize response packet", "error", err)
		s.sendErrorResponse(sessionCtx)

		return true
	}

	// instruct the view server to send the response packet to the client
	logger.Info("instructing view server to send response packet to client")
	_ = s.NATS().Publish(fmt.Sprintf("session.%s.view.send", sessionCtx.sessionKey), responseData)

	// todo: update character flags for online status
	// todo: update character stats for zoning = 2
	// todo: log the user's ip address in account ip records

	return false
}
