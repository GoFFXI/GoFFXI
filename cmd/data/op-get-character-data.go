package data

// import (
// 	"context"
// 	"fmt"
// 	"net"

// 	"github.com/GoFFXI/login-server/internal/constants"
// 	"github.com/GoFFXI/login-server/internal/database"
// 	responsecharacterinfo "github.com/GoFFXI/login-server/internal/packets/response-character-info"
// 	"github.com/GoFFXI/login-server/internal/tools"
// )

// const (
// 	MaxCharacterSlots           = 16
// 	StatusInvalid        uint16 = 0
// 	StatusAvailable      uint16 = 1
// 	StatusDisabledUnpaid uint16 = 2
// )

// func (s *DataServer) opGetCharacterData(_ context.Context, conn net.Conn, accountSession *database.AccountSession, request []byte) {
// 	logger := s.Logger().With("client", conn.RemoteAddr().String())
// 	logger.Info("handling character data request")

// 	accountID := tools.GetUint32FromByteBuffer(request, 1)
// 	logger.Info("detected account ID", "accountID", accountID)

// 	if accountID != accountSession.AccountID {
// 		logger.Warn("account ID mismatch", "expected", accountSession.AccountID, "got", accountID)
// 		_ = s.NATS().Publish(fmt.Sprintf("session.%s.view.close", accountSession.SessionKey), nil)
// 		_ = conn.Close()

// 		return
// 	}

// 	serverIP := tools.GetIPAddressFromBuffer(request, 5)
// 	logger.Info("detected server IP", "serverIP", serverIP)

// 	// here is where it gets interesting; we need to send 2 packets:
// 	// 1. a response to the data connection which contains a command for xiloader to list characters and the number of characters to list
// 	// 2. a response to the view connection which contains the actual character data

// 	// first, let's handle the response over this data connection
// 	dataPacket := make([]byte, 0x148)
// 	dataPacket[0] = 0x03 // instruct xiloader to list characters
// 	dataPacket[1] = 3    // character count (max 16)
// 	_, _ = conn.Write(dataPacket)

// 	viewPacket, err := getExampleCharacterData(accountSession.SessionKey)
// 	if err != nil {
// 		logger.Error("error getting example character data", "err", err)
// 		return
// 	}

// 	// inform the view server it needs to send this packet
// 	logger.Info("sending character data to view server", "sessionKey", accountSession.SessionKey)
// 	_ = s.NATS().Publish(fmt.Sprintf("session.%s.view.send", accountSession.SessionKey), viewPacket)
// }

// func getExampleCharacterData(sessionKey string) ([]byte, error) {
// 	characters := []responsecharacterinfo.CharacterInfoSub{
// 		// Active character using database-like creation
// 		CreateCharacterFromDB(
// 			0x00010001,             // Character ID
// 			"Naga",                 // Character name
// 			"YoyoIsABitch",         // World/Server name
// 			constants.RaceHumeMale, // Race
// 			6,                      // Face
// 			constants.JobWarrior,   // Main job
// 			constants.JobMonk,      // Sub job
// 			75,                     // Main job level
// 			[16]uint8{1, 75, 37, 20, 15, 10, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // Job levels
// 			constants.TownSandoria, // Nation
// 			230,                    // Zone (Southern San d'Oria)
// 			constants.SizeMedium,   // Size
// 			[8]uint16{0, 0x11D6, 0x21D6, 0x31D6, 0x41D6, 0x51D6, 0x636D, 0x71CB}, // Equipment
// 		),
// 		// Empty slot for new character
// 		CreateEmptySlot(0x00010002, 0, "YoyoIsABitch"),
// 	}

// 	// Create packet with MD5 hash
// 	packet1, err := responsecharacterinfo.NewResponseChrInfo2WithHash(characters)
// 	if err != nil {
// 		return nil, fmt.Errorf("error creating packet with hash: %w", err)
// 	}

// 	// Serialize the packet
// 	data1, err := packet1.Serialize()
// 	if err != nil {
// 		return nil, fmt.Errorf("error serializing packet with hash: %w", err)
// 	}

// 	return data1, nil
// }

// // CreateCharacterFromDB creates a CharacterInfoSub from database-like data
// // This mimics the LandSandBoat server's database query structure
// func CreateCharacterFromDB(
// 	charID uint32,
// 	charName string,
// 	worldName string,
// 	race uint16,
// 	face uint16,
// 	mjob uint8,
// 	sjob uint8,
// 	mjobLevel uint8,
// 	jobLevels [16]uint8,
// 	nation uint8,
// 	zone uint16,
// 	size uint8,
// 	equipment [8]uint16,
// ) responsecharacterinfo.CharacterInfoSub {
// 	// The character ID is made up of two parts (following C++ pattern)
// 	charIDMain := uint16(charID & 0xFFFF)
// 	charIDExtra := uint8((charID >> 16) & 0xFF)

// 	char := responsecharacterinfo.CharacterInfoSub{
// 		FFXIID:         charID, // Content ID = Character ID in private servers
// 		FFXIIDWorld:    charIDMain,
// 		WorldID:        0, // 0 for single world servers
// 		Status:         StatusAvailable,
// 		Flags:          0, // No rename or race change
// 		FFXIIDWorldTbl: charIDExtra,
// 	}

// 	// Copy character name (max 15 chars + null terminator)
// 	nameBytes := make([]byte, 16)
// 	copy(nameBytes, charName)
// 	copy(char.CharacterName[:], nameBytes)

// 	// Copy world name
// 	worldBytes := make([]byte, 16)
// 	copy(worldBytes, worldName)
// 	copy(char.WorldName[:], worldBytes)

// 	// Fill character info
// 	char.CharacterInfo = responsecharacterinfo.TCOperationMake{
// 		MonNo:     race,
// 		MJobNo:    mjob,
// 		SJobNo:    sjob,
// 		FaceNo:    face,
// 		TownNo:    nation,
// 		GenFlag:   0,
// 		HairNo:    uint8(face), // Note: C++ code uses face for hair (may need adjustment)
// 		Size:      size,
// 		WorldNo:   0,
// 		ZoneNo:    uint8(zone & 0xFF),
// 		ZoneNo2:   uint8((zone >> 8) & 1),
// 		MJobLevel: mjobLevel,
// 		OpenFlag:  1, // Not anonymous
// 		Version:   2,
// 		JobLev:    jobLevels,
// 	}

// 	// Copy equipment IDs
// 	// Note: GrapIDTbl[0] should be calculated, but C++ code shows it using face
// 	char.CharacterInfo.GrapIDTbl[0] = face // This may need the proper calculation
// 	for i := 1; i < 8; i++ {
// 		char.CharacterInfo.GrapIDTbl[i] = equipment[i]
// 	}

// 	return char
// }

// // CreateEmptySlot creates an empty character slot (for character creation)
// func CreateEmptySlot(ffxiID uint32, worldID uint16, worldName string) responsecharacterinfo.CharacterInfoSub {
// 	char := responsecharacterinfo.CharacterInfoSub{
// 		Status:        StatusAvailable, // Available slot for creation
// 		CharacterName: [16]byte{0x20},
// 	}

// 	// Empty character info structure - most fields stay at zero
// 	char.CharacterInfo = responsecharacterinfo.TCOperationMake{
// 		JobLev: [16]uint8{1}, // First slot always 1
// 	}

// 	return char
// }
