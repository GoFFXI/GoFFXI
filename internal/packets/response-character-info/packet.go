package responsecharacterinfo

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
)

/*
ResponseChrInfo2 Packet Implementation
=======================================

This implements the FFXI lobby server's ResponseChrInfo2 packet (OpCode 0x0020)
sent to clients containing character list information.

Packet Structure:
- PacketHeader: 28 bytes
  - PacketSize (4), Terminator (4), Command (4), Identifier (16)
- Character Count: 4 bytes (uint32)
- Character Info Array: n * 140 bytes each
  - IDs and flags: 12 bytes
  - Character name: 16 bytes
  - World name: 16 bytes
  - TC_OPERATION_MAKE: 96 bytes (character creation/status data)

Total packet sizes:
- 0 characters: 32 bytes (0x20)
- 1 character: 172 bytes (0xAC)
- 3 characters: 452 bytes (0x1C4)
- 16 characters (max): 2272 bytes (0x8E0)

Based on XiPackets documentation:
https://github.com/atom0s/XiPackets/blob/main/lobby/S2C_0x0020_ResponseChrInfo2.md
https://github.com/atom0s/XiPackets/blob/main/lobby/CharacterInfo.md
*/

// PacketHeader represents the standard packet header for lobby packets
type PacketHeader struct {
	PacketSize uint32 // Total size of the packet
	Terminator uint32 // Always 0x46465849 ("IXFF")
	Command    uint32 // OpCode - 0x0020 for ResponseChrInfo2
	Identifier string // Identifier - must be exactly 16 bytes
}

// TCOperationMake represents character creation/appearance data
// This is the TC_OPERATION_MAKE structure from the PS2 beta
// Total size: 96 bytes (fits within the 140-byte CharacterInfoSub)
type TCOperationMake struct {
	MonNo         uint16    // Race ID (mon_no)
	MJobNo        uint8     // Main job ID (mjob_no)
	SJobNo        uint8     // Sub job ID (sjob_no)
	FaceNo        uint16    // Face model ID (face_no)
	TownNo        uint8     // Town/Nation ID (town_no)
	GenFlag       uint8     // Unknown flag, always 0 (gen_flag)
	HairNo        uint8     // Hair model ID (hair_no)
	Size          uint8     // Character size: 0=Small, 1=Medium, 2=Large (size)
	WorldNo       uint16    // Unknown world number (world_no)
	GrapIDTbl     [8]uint16 // Model IDs: [0]=calculated, [1]=head, [2]=body, etc (GrapIDTbl)
	ZoneNo        uint8     // Current zone ID (low byte) (zone_no)
	MJobLevel     uint8     // Main job level (mjob_level)
	OpenFlag      uint8     // Online status: 0=hidden(/anon), 1=visible (open_flag)
	GMCallCounter uint8     // GM call counter (unused by client) (GMCallCounter)
	Version       uint16    // Version, usually 2 (version)
	Skill1        uint8     // Fishing skill level (skill1)
	ZoneNo2       uint8     // Zone ID high byte (zone_no2)
	RankSD        uint8     // San d'Oria rank (TC_OPERATION_WORK_USER_RANK_LEVEL_SD_)
	RankBS        uint8     // Bastok rank (TC_OPERATION_WORK_USER_RANK_LEVEL_BS_)
	RankWS        uint8     // Windurst rank (TC_OPERATION_WORK_USER_RANK_LEVEL_WS_)
	ErrCounter    uint8     // Error counter, always 0 (ErrCounter)
	FameSD        uint16    // San d'Oria fame (TC_OPERATION_WORK_USER_FAME_SD_COMMON_)
	FameBS        uint16    // Bastok fame (TC_OPERATION_WORK_USER_FAME_BS_COMMON_)
	FameWS        uint16    // Windurst fame (TC_OPERATION_WORK_USER_FAME_WS_COMMON_)
	FameDark      uint16    // Norg fame (TC_OPERATION_WORK_USER_FAME_DARK_GUILD_)
	PlayTime      uint32    // Play time in seconds (PlayTime)
	GetJobFlag    uint32    // Unlocked jobs flag mask (get_job_flag)
	JobLev        [16]uint8 // Job levels array (first slot unused) (job_lev)
	FirstLogin    uint32    // First login timestamp (FirstLoginDate)
	Gold          uint32    // Current gil amount (Gold)
	Skill2        uint8     // Woodworking skill level (skill2)
	Skill3        uint8     // Smithing skill level (skill3)
	Skill4        uint8     // Goldsmithing skill level (skill4)
	Skill5        uint8     // Clothcraft skill level (skill5)
	ChatCounter   uint32    // Chat counter (unused by client) (ChatCounter)
	PartyCounter  uint32    // Party counter (unused by client) (PartyCounter)
	Skill6        uint8     // Leathercraft skill level (skill6)
	Skill7        uint8     // Bonecraft skill level (skill7)
	Skill8        uint8     // Alchemy skill level (skill8)
	Skill9        uint8     // Cooking skill level (skill9)
}

// CharacterInfoSub represents individual character data in the packet
type CharacterInfoSub struct {
	FFXIID         uint32          // Unique character ID
	FFXIIDWorld    uint16          // Character's in-game server ID
	WorldID        uint16          // Server world ID
	Status         uint16          // 1=Available, 2=Disabled (unpaid)
	Flags          uint8           // Bit 0: RenameFlag, Bit 1: RaceChangeFlag
	FFXIIDWorldTbl uint8           // Character's in-game server ID (hi-byte)
	CharacterName  [16]byte        // Character name (null-terminated)
	WorldName      [16]byte        // World name (null-terminated)
	CharacterInfo  TCOperationMake // Character creation/appearance data
}

// ResponseChrInfo2 represents the complete packet structure
type ResponseChrInfo2 struct {
	Header     PacketHeader
	Characters uint32             // Number of character entries
	CharInfo   []CharacterInfoSub // Array of character information
}

// Constants for packet construction
const (
	OpCodeResponseChrInfo2 = 0x0020
	PacketTerminator       = 0x46465849 // "IXFF" in little-endian
)

// Character status values
const (
	StatusInvalid        uint16 = 0
	StatusAvailable      uint16 = 1
	StatusDisabledUnpaid uint16 = 2
)

// NewResponseChrInfo2 creates a new ResponseChrInfo2 packet
func NewResponseChrInfo2(identifier string, characters []CharacterInfoSub) (*ResponseChrInfo2, error) {
	// Validate identifier length if provided (allow empty for hash calculation)
	if identifier != "" && len(identifier) != 16 {
		return nil, fmt.Errorf("identifier must be exactly 16 characters or empty, got %d", len(identifier))
	}

	packet := &ResponseChrInfo2{
		Header: PacketHeader{
			Terminator: PacketTerminator,
			Command:    OpCodeResponseChrInfo2,
			Identifier: identifier,
		},
		Characters: uint32(len(characters)),
		CharInfo:   characters,
	}

	// Calculate packet size
	packet.Header.PacketSize = uint32(28 + 4 + len(characters)*140) // header + character count + character data

	return packet, nil
}

// FormatIdentifier ensures an identifier string is exactly 16 bytes
// If shorter, it pads with null bytes; if longer, it truncates
func FormatIdentifier(id string) string {
	if len(id) == 16 {
		return id
	}
	if len(id) < 16 {
		// Pad with null bytes
		return id + string(make([]byte, 16-len(id)))
	}
	// Truncate if too long
	return id[:16]
}

// Serialize converts the packet to bytes for transmission
func (p *ResponseChrInfo2) Serialize() ([]byte, error) {
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
	for i, char := range p.CharInfo {
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

// NewResponseChrInfo2WithHash creates a packet and auto-generates MD5 hash as identifier
func NewResponseChrInfo2WithHash(characters []CharacterInfoSub) (*ResponseChrInfo2, error) {
	packet, err := NewResponseChrInfo2("", characters) // Empty identifier for now
	if err != nil {
		return nil, err
	}

	// Serialize to calculate hash
	tempData, err := packet.SerializeForHash()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize for hash: %w", err)
	}

	// Calculate MD5 hash of the packet data
	hash := md5.Sum(tempData)
	packet.Header.Identifier = string(hash[:])

	return packet, nil
}

// SerializeForHash serializes the packet without the identifier for hash calculation
func (p *ResponseChrInfo2) SerializeForHash() ([]byte, error) {
	// Temporarily clear identifier for hash calculation
	originalID := p.Header.Identifier
	p.Header.Identifier = ""

	data, err := p.Serialize()

	// Restore original identifier
	p.Header.Identifier = originalID

	return data, err
}
