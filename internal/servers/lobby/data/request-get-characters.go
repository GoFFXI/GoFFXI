package data

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"

	"github.com/GoFFXI/GoFFXI/internal/constants"
	"github.com/GoFFXI/GoFFXI/internal/database"
	"github.com/GoFFXI/GoFFXI/internal/packets/lobby"
)

const (
	// https://github.com/atom0s/XiPackets/blob/main/lobby/C2S_0x001F_RequestGetChr.md
	// for Some reason, the command request is 0xA1 but the docs say 0x001F
	CommandRequestGetCharacters   = 0x00A1
	CommandResponseChrInfo2       = 0x0020
	CommandResponseListCharacters = 0x0003

	CharacterStatusInvalid        uint16 = 0
	CharacterStatusAvailable      uint16 = 1
	CharacterStatusDisabledUnpaid uint16 = 2

	MaxCharacterSlots = 16
)

// https://github.com/atom0s/XiPackets/blob/main/lobby/C2S_0x001F_RequestGetChr.md
type RequestGetCharacters struct {
	_         byte   // 1 byte
	AccountID uint32 // 4 bytes

	// This packet should apparently contain a Password field but I could not
	// parse it properly from the client data. Leaving it out for now as it's not used.
}

func NewRequestGetCharacters(data []byte) (*RequestGetCharacters, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("invalid RequestGetCharacters packet size: expected at least 5, got %d", len(data))
	}

	request := &RequestGetCharacters{}
	buf := bytes.NewReader(data)

	// Read the entire struct at once (works because all fields are fixed-size)
	err := binary.Read(buf, binary.LittleEndian, request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

// ResponseCharacterList represents the data socket packet sent to xiloader
// Total size: 0x148 (328) bytes
type ResponseCharacterList struct {
	Command        uint8                          // Offset 0x00: 0x03 = Send character list command
	CharacterCount uint8                          // Offset 0x01: Number of characters in list
	_              [14]byte                       // Offset 0x02-0x0F: Padding
	Characters     [20]ResponseCharacterListEntry // Offset 0x10: Character entries (max 20 chars)
	_              [8]byte                        // Offset 0x150-0x147: Padding to reach 0x148 total
}

// ResponseCharacterListEntry represents a single character entry in the xiloader packet
// Each entry is 16 bytes, starting at offset 16
type ResponseCharacterListEntry struct {
	ContentID   uint32  // Character/Content ID (same value)
	CharIDMain  uint16  // Lower 16 bits of character ID
	WorldID     uint8   // World ID (ignored by xiloader)
	CharIDExtra uint8   // Upper 8 bits of character ID (ignored by xiloader)
	_           [8]byte // Padding to make 16 bytes total
}

// NewResponseCharacterList creates a new character list packet for xiloader
func NewResponseCharacterList() *ResponseCharacterList {
	return &ResponseCharacterList{
		Command: CommandResponseListCharacters,
	}
}

// AddCharacter adds a character to the list
func (p *ResponseCharacterList) AddCharacter(contentID uint32) error {
	if p.CharacterCount >= 20 {
		return fmt.Errorf("maximum character limit (20) reached")
	}

	p.Characters[p.CharacterCount] = ResponseCharacterListEntry{
		ContentID:   contentID,
		CharIDMain:  uint16(contentID & 0xFFFF), //nolint:gosec // lower 16 bits
		WorldID:     1,
		CharIDExtra: uint8((contentID >> 16) & 0xFF), //nolint:gosec // upper 8 bits
	}

	p.CharacterCount++
	return nil
}

// Serialize converts the packet to bytes for transmission
func (p *ResponseCharacterList) Serialize() ([]byte, error) {
	// Create a buffer of exactly 0x148 bytes
	buffer := make([]byte, 0x148)

	// Set command and character count
	buffer[0] = p.Command
	buffer[1] = p.CharacterCount

	// Write each character entry starting at offset 16
	for i := uint8(0); i < p.CharacterCount && i < 20; i++ {
		offset := 16 * (i + 1) // Matches C++ formula: uint32 uListOffset = 16 * (i + 1)

		char := p.Characters[i]

		// Write ContentID (4 bytes)
		binary.LittleEndian.PutUint32(buffer[offset:], char.ContentID)

		// Write CharIDMain (2 bytes)
		binary.LittleEndian.PutUint16(buffer[offset+4:], char.CharIDMain)

		// Write WorldID (1 byte)
		buffer[offset+6] = char.WorldID

		// Write CharIDExtra (1 byte)
		buffer[offset+7] = char.CharIDExtra

		// Remaining 8 bytes stay as zeros (padding)
	}

	return buffer, nil
}

type ResponseChrInfo2Sub struct {
	FFXIID         uint32              // Unique character ID
	FFXIIDWorld    uint16              // Character's in-game server ID
	WorldID        uint16              // Server world ID
	Status         uint16              // 1=Available, 2=Disabled (unpaid)
	Flags          uint8               // Bit 0: RenameFlag, Bit 1: RaceChangeFlag
	FFXIIDWorldTbl uint8               // Character's in-game server ID (hi-byte)
	CharacterName  [16]byte            // Character name (null-terminated)
	WorldName      [16]byte            // World name (null-terminated)
	CharacterInfo  lobby.CharacterInfo // Character creation/appearance data
}

// https://github.com/atom0s/XiPackets/blob/main/lobby/S2C_0x0020_ResponseChrInfo2.md
type ResponseChrInfo2 struct {
	Header lobby.PacketHeader

	Characters uint32                // Number of character entries
	CharInfo   []ResponseChrInfo2Sub // Array of character information
}

func NewResponseChrInfo2(characters []ResponseChrInfo2Sub) (*ResponseChrInfo2, error) {
	packet := &ResponseChrInfo2{
		Header: lobby.PacketHeader{
			Terminator: constants.ResponsePacketTerminator,
			Command:    CommandResponseChrInfo2,
		},
		Characters: uint32(len(characters)), //nolint:gosec // we trust the length of our own slice
		CharInfo:   characters,
	}

	// Calculate packet size
	// header + character count + character data
	packet.Header.PacketSize = uint32(28 + 4 + len(characters)*140) //nolint:gosec // we trust the length of our own slice

	return packet, nil
}

func (p ResponseChrInfo2) serialize() ([]byte, error) { //nolint:gocyclo // it's acceptable because of the multiple writes in specific orders
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

	// Write character count
	if err := binary.Write(buf, binary.LittleEndian, p.Characters); err != nil {
		return nil, fmt.Errorf("failed to write character count: %w", err)
	}

	// Write each character info
	for i, char := range p.CharInfo { //nolint:gocritic // it's intentional to copy 140 bytes at a time
		// Write the fixed-size parts of CharacterInfoSub
		if err := binary.Write(buf, binary.LittleEndian, char.FFXIID); err != nil {
			return nil, fmt.Errorf("failed to write character %d FFXI ID: %w", i, err)
		}
		if err := binary.Write(buf, binary.LittleEndian, char.FFXIIDWorld); err != nil {
			return nil, fmt.Errorf("failed to write character %d FFXI ID World: %w", i, err)
		}
		if err := binary.Write(buf, binary.LittleEndian, char.WorldID); err != nil {
			return nil, fmt.Errorf("failed to write character %d World ID: %w", i, err)
		}
		if err := binary.Write(buf, binary.LittleEndian, char.Status); err != nil {
			return nil, fmt.Errorf("failed to write character %d status: %w", i, err)
		}
		if err := binary.Write(buf, binary.LittleEndian, char.Flags); err != nil {
			return nil, fmt.Errorf("failed to write character %d flags: %w", i, err)
		}
		if err := binary.Write(buf, binary.LittleEndian, char.FFXIIDWorldTbl); err != nil {
			return nil, fmt.Errorf("failed to write character %d FFXI ID World Tbl: %w", i, err)
		}
		if err := binary.Write(buf, binary.LittleEndian, char.CharacterName); err != nil {
			return nil, fmt.Errorf("failed to write character %d name: %w", i, err)
		}
		if err := binary.Write(buf, binary.LittleEndian, char.WorldName); err != nil {
			return nil, fmt.Errorf("failed to write character %d world name: %w", i, err)
		}

		// Write the TCOperationMake structure (this is fixed-size, so we can use binary.Write)
		if err := binary.Write(buf, binary.LittleEndian, char.CharacterInfo); err != nil {
			return nil, fmt.Errorf("failed to write character %d info: %w", i, err)
		}
	}

	return buf.Bytes(), nil
}

func (p *ResponseChrInfo2) SerializeWithHash() ([]byte, error) {
	// clear identifier to write zeros for hash calculation
	p.Header.Identifier = [16]byte{}

	// serialize packet without hash
	packet, err := p.serialize()
	if err != nil {
		return nil, err
	}

	// calculate the MD5 hash over the entire packet
	p.Header.Identifier = md5.Sum(packet)

	// re-serialize packet with correct hash
	packet, err = p.serialize()
	if err != nil {
		return nil, err
	}

	return packet, nil
}

func (s DataServer) handleRequestGetCharacters(sessionCtx *sessionContext, accountSession *database.AccountSession, data []byte) bool {
	logger := sessionCtx.logger.With("request", "get-characters")
	logger.Info("handling request")

	// make sure the account session is valid
	if accountSession == nil {
		logger.Warn("no valid account session for this request")
		return true
	}

	// parse the request packet
	request, err := NewRequestGetCharacters(data)
	if err != nil {
		logger.Error("failed to parse RequestGetCharacters packet", "error", err)
		return true
	}

	// make sure the session's account ID is correct
	if request.AccountID != accountSession.AccountID {
		logger.Warn("account ID mismatch", "expected", accountSession.AccountID, "got", request.AccountID)
		_ = s.NATS().Publish(fmt.Sprintf("session.%s.view.close", accountSession.SessionKey), nil)
		return true
	}

	// the view server never receives the account ID in any of it's requests; so, to get around this
	// we are going to send over the account ID via NATS so it can associate future requests via the session key
	accountIDBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(accountIDBytes, request.AccountID)
	_ = s.NATS().Publish(fmt.Sprintf("session.%s.view.account.id", accountSession.SessionKey), accountIDBytes)

	// also, store the account ID in the session context for future use
	sessionCtx.accountID = request.AccountID

	// in order to process this request, we need to do 2 things:
	// 1. send a response over the data connection to instruct xiloader to list characters
	// 2. send a response over the view connection with the actual character data

	// fetch the characters from the database
	var characters []database.Character
	err = s.DB().BunDB().NewSelect().Model(&characters).Where("account_id = ?", accountSession.AccountID).
		Relation("Jobs").
		Relation("Stats").
		Relation("Looks").
		Scan(sessionCtx.ctx)
	if err != nil {
		logger.Error("failed to fetch characters", "error", err)
		return true
	}

	// limit to max character slots to it's maximum of 16
	// note how this is not the server configurable max, but the protocol max
	// the server config will be applied when creating new characters
	totalCharacters := uint8(len(characters)) //nolint:gosec // we trust the length will not exceed uint8
	if totalCharacters > MaxCharacterSlots {
		totalCharacters = MaxCharacterSlots
	}

	// setup our data response (so we can add characters to it later)
	dataResponse := NewResponseCharacterList()

	// now, let's prepare the character data for the view connection
	characterSlots := make([]ResponseChrInfo2Sub, 0, totalCharacters)

	// loop through existing characters and add them to the response
	for _, character := range characters {
		_ = dataResponse.AddCharacter(character.ID)
		characterSlots = append(characterSlots, ConvertDBCharacterToResponseCharInfo2Sub(&character, s.Config().WorldName))
	}

	// finally, fill remaining slots with empty slots
	emptySlots := s.Config().MaxContentIDsPerAccount - len(characters)
	for range emptySlots {
		characterSlots = append(characterSlots, CreateEmptySlot())
	}

	// send the data response to xiloader
	dataPacket, err := dataResponse.Serialize()
	if err != nil {
		logger.Error("failed to serialize ResponseCharacterList packet", "error", err)
		return true
	}

	_, err = sessionCtx.conn.Write(dataPacket)
	if err != nil {
		logger.Error("failed to send ResponseCharacterList packet", "error", err)
		return true
	}

	// finally, send the view response to the view server
	viewResponse, err := NewResponseChrInfo2(characterSlots)
	if err != nil {
		logger.Error("failed to create ResponseChrInfo2 packet", "error", err)
		return true
	}

	viewPacket, err := viewResponse.SerializeWithHash()
	if err != nil {
		logger.Error("failed to serialize ResponseChrInfo2 packet", "error", err)
		return true
	}

	logger.Info("instructing view server to send character data")
	_ = s.NATS().Publish(fmt.Sprintf("session.%s.view.send", accountSession.SessionKey), viewPacket)
	return false
}

func ConvertDBCharacterToResponseCharInfo2Sub(character *database.Character, worldName string) ResponseChrInfo2Sub {
	char := ResponseChrInfo2Sub{
		FFXIID:         character.ID,
		FFXIIDWorld:    uint16(character.ID & 0xFFFF),      //nolint:gosec // lower 16 bits
		FFXIIDWorldTbl: uint8((character.ID >> 16) & 0xFF), //nolint:gosec // upper 8 bits
		WorldID:        1,
		Status:         CharacterStatusAvailable,
	}

	// copy character name
	nameBytes := make([]byte, 16)
	copy(nameBytes, character.Name)
	copy(char.CharacterName[:], nameBytes)

	// copy world name
	worldBytes := make([]byte, 16)
	copy(worldBytes, worldName)
	copy(char.WorldName[:], worldBytes)

	// now, fill in character info
	char.CharacterInfo = lobby.CharacterInfo{
		RaceID:       uint16(character.Looks.Race),
		MainJobID:    character.Stats.MainJob,
		SubJobID:     character.Stats.SubJob,
		FaceModelID:  uint16(character.Looks.Face),
		TownNumber:   character.Nation,
		GenFlag:      0,
		HairModelID:  character.Looks.Face,
		WorldNumber:  1,
		ZoneNumber:   uint8(character.PosZone & 0xFF),     //nolint:gosec // lower 16 bits
		ZoneNum2:     uint8((character.PosZone >> 8) & 1), //nolint:gosec // upper 8 bits
		MainJobLevel: character.GetMainJobLevel(),
		GrapIDTbl: [8]uint16{
			uint16(character.Looks.Face),
			character.Looks.Head,
			character.Looks.Body,
			character.Looks.Hands,
			character.Looks.Legs,
			character.Looks.Feet,
			character.Looks.Main,
			character.Looks.Sub,
		},
	}

	return char
}

func CreateEmptySlot() ResponseChrInfo2Sub {
	char := ResponseChrInfo2Sub{
		Status:        CharacterStatusAvailable, // Available slot for creation
		CharacterName: [16]byte{0x20},
	}

	// Empty character info structure - most fields stay at zero
	char.CharacterInfo = lobby.CharacterInfo{
		JobLevels: [16]uint8{1}, // First slot always 1
	}

	return char
}
