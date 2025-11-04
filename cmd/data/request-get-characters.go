package data

import (
	"bytes"
	"crypto/md5" //nolint:gosec // we are using MD5 for compatibility with FFXI protocol, not for security
	"encoding/binary"
	"fmt"

	"github.com/GoFFXI/login-server/internal/constants"
	"github.com/GoFFXI/login-server/internal/database"
)

const (
	// https://github.com/atom0s/XiPackets/blob/main/lobby/C2S_0x001F_RequestGetChr.md
	// for Some reason, the command request is 0xA1 but the docs say 0x001F
	CommandRequestGetCharacters = 0xA1
	CommandResponseChrInfo2     = 0x0020

	CharacterStatusInvalid        uint16 = 0
	CharacterStatusAvailable      uint16 = 1
	CharacterStatusDisabledUnpaid uint16 = 2
)

// https://github.com/atom0s/XiPackets/blob/main/lobby/C2S_0x001F_RequestGetChr.md
type RequestGetCharacters struct {
	Padding1  byte   // 1 byte
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
	err := binary.Read(buf, binary.BigEndian, request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

type ResponseChrInfo2Header struct {
	PacketSize uint32 // Total size of the packet
	Terminator uint32 // Always 0x46465849 ("IXFF")
	Command    uint32 // OpCode - 0x0020 for ResponseChrInfo2
	Identifier string // Identifier - must be exactly 16 bytes
}

type ResponseChrInfo2ChrInfo struct {
	RaceID            uint16    // Race ID (mon_no)
	MainJobID         uint8     // Main job ID (mjob_no)
	SubJobID          uint8     // Sub job ID (sjob_no)
	FaceModelID       uint16    // Face model ID (face_no)
	TownNumber        uint8     // Town/Nation ID (town_no)
	GenFlag           uint8     // Unknown flag, always 0 (gen_flag)
	HairModelID       uint8     // Hair model ID (hair_no)
	CharacterSize     uint8     // Character size: 0=Small, 1=Medium, 2=Large (size)
	WorldNumber       uint16    // Unknown world number (world_no)
	GrapIDTbl         [8]uint16 // Model IDs: [0]=calculated, [1]=head, [2]=body, etc (GrapIDTbl)
	ZoneNumber        uint8     // Current zone ID (low byte) (zone_no)
	MainJobLevel      uint8     // Main job level (mjob_level)
	OnlineStatus      uint8     // Online status: 0=hidden(/anon), 1=visible (open_flag)
	GMCallCounter     uint8     // GM call counter (unused by client) (GMCallCounter)
	Version           uint16    // Version, usually 2 (version)
	FishingSkill      uint8     // Fishing skill level (skill1)
	ZoneNum2          uint8     // Zone ID high byte (zone_no2)
	SandyRank         uint8     // San d'Oria rank (TC_OPERATION_WORK_USER_RANK_LEVEL_SD_)
	BastokRank        uint8     // Bastok rank (TC_OPERATION_WORK_USER_RANK_LEVEL_BS_)
	WindyRank         uint8     // Windurst rank (TC_OPERATION_WORK_USER_RANK_LEVEL_WS_)
	ErrCounter        uint8     // Error counter, always 0 (ErrCounter)
	SandyFame         uint16    // San d'Oria fame (TC_OPERATION_WORK_USER_FAME_SD_COMMON_)
	BastokFame        uint16    // Bastok fame (TC_OPERATION_WORK_USER_FAME_BS_COMMON_)
	WindyFame         uint16    // Windurst fame (TC_OPERATION_WORK_USER_FAME_WS_COMMON_)
	NorgFame          uint16    // Norg fame (TC_OPERATION_WORK_USER_FAME_DARK_GUILD_)
	PlayTimeSeconds   uint32    // Play time in seconds (PlayTime)
	UnlockedJobsFlag  uint32    // Unlocked jobs flag mask (get_job_flag)
	JobLevels         [16]uint8 // Job levels array (first slot unused) (job_lev)
	FirstLoginDate    uint32    // First login timestamp (FirstLoginDate)
	GilAmount         uint32    // Current gil amount (Gold)
	WoodworkingSkill  uint8     // Woodworking skill level (skill2)
	SmithingSkill     uint8     // Smithing skill level (skill3)
	GoldsmithingSkill uint8     // Goldsmithing skill level (skill4)
	ClothcraftSkill   uint8     // Clothcraft skill level (skill5)
	ChatCounter       uint32    // Chat counter (unused by client) (ChatCounter)
	PartyCounter      uint32    // Party counter (unused by client) (PartyCounter)
	LeathercraftSkill uint8     // Leathercraft skill level (skill6)
	BonecraftSkill    uint8     // Bonecraft skill level (skill7)
	AlchemySkill      uint8     // Alchemy skill level (skill8)
	CookingSkill      uint8     // Cooking skill level (skill9)
}

type ResponseChrInfo2Sub struct {
	FFXIID         uint32                  // Unique character ID
	FFXIIDWorld    uint16                  // Character's in-game server ID
	WorldID        uint16                  // Server world ID
	Status         uint16                  // 1=Available, 2=Disabled (unpaid)
	Flags          uint8                   // Bit 0: RenameFlag, Bit 1: RaceChangeFlag
	FFXIIDWorldTbl uint8                   // Character's in-game server ID (hi-byte)
	CharacterName  [16]byte                // Character name (null-terminated)
	WorldName      [16]byte                // World name (null-terminated)
	CharacterInfo  ResponseChrInfo2ChrInfo // Character creation/appearance data
}

// https://github.com/atom0s/XiPackets/blob/main/lobby/S2C_0x0020_ResponseChrInfo2.md
type ResponseChrInfo2 struct {
	Header     ResponseChrInfo2Header
	Characters uint32                // Number of character entries
	CharInfo   []ResponseChrInfo2Sub // Array of character information
}

func NewResponseChrInfo2(characters []ResponseChrInfo2Sub) (*ResponseChrInfo2, error) {
	packet := &ResponseChrInfo2{
		Header: ResponseChrInfo2Header{
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
	if p.Header.Identifier == "" {
		if _, err := buf.Write(make([]byte, 16)); err != nil {
			return nil, fmt.Errorf("failed to write empty identifier: %w", err)
		}
	} else {
		if len(p.Header.Identifier) != 16 {
			return nil, fmt.Errorf("identifier must be exactly 16 bytes, got %d", len(p.Header.Identifier))
		}
		if _, err := buf.WriteString(p.Header.Identifier); err != nil {
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
	p.Header.Identifier = ""

	// serialize packet without hash
	packet, err := p.serialize()
	if err != nil {
		return nil, err
	}

	// calculate the MD5 hash over the entire packet
	hash := md5.Sum(packet) //nolint:gosec // we are using MD5 for compatibility with FFXI protocol, not for security
	p.Header.Identifier = string(hash[:])

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

	// in order to process this request, we need to do 2 things:
	// 1. send a response over the data connection to instruct xiloader to list characters
	// 2. send a response over the view connection with the actual character data

	// first, let's handle the response over this data connection
	dataPacket := make([]byte, 0x148)
	dataPacket[0] = 0x03 // instruct xiloader to list characters
	dataPacket[1] = 3    // character count (max 16)
	_, _ = sessionCtx.conn.Write(dataPacket)

	// now, let's prepare the character data for the view connection
	characters := []ResponseChrInfo2Sub{
		CreateEmptySlot(),
		CreateEmptySlot(),
		CreateEmptySlot(),
	}

	viewResponse, err := NewResponseChrInfo2(characters)
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

func CreateEmptySlot() ResponseChrInfo2Sub {
	char := ResponseChrInfo2Sub{
		Status:        CharacterStatusAvailable, // Available slot for creation
		CharacterName: [16]byte{0x20},
	}

	// Empty character info structure - most fields stay at zero
	char.CharacterInfo = ResponseChrInfo2ChrInfo{
		JobLevels: [16]uint8{1}, // First slot always 1
	}

	return char
}
