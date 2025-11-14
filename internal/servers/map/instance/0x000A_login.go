package instance

import (
	"math"
	"time"

	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
	serverPackets "github.com/GoFFXI/GoFFXI/internal/packets/map/server"
)

func (s *InstanceWorker) processLoginPacket(routedPacket mapPackets.RoutedPacket) {
	s.Logger().Info("processing login packet", "clientAddr", routedPacket.ClientAddr)

	// todo: fetch all of the relevant character data from the database

	// send an equip clear packet
	equipClearPacket := serverPackets.EquipClearPacket{}

	// send a graph list packet
	graphListPacket := CreateFakeGrapListPacket()

	// send an item max packet
	itemMaxPacket := CreateFakeItemMaxPacket()

	// send a a login response packet
	loginPacket := CreateMockLoginPacketFromSQL()

	// send all of our packets
	clientAddr := routedPacket.ClientAddr
	s.sendPacket(clientAddr, &equipClearPacket)
	s.sendPacket(clientAddr, graphListPacket)
	s.sendPacket(clientAddr, itemMaxPacket)
	s.sendPacket(clientAddr, loginPacket)

	// todo: fetch equipment from database and send equip list packets for each slot
	// if a slot is empty (0), no packet needs to be sent
	// for now, just don't send anything - assume each character is naked
}

func CreateFakeGrapIDTbl() [9]uint16 {
	return [9]uint16{
		0x0100, // Race/Hair: (race << 8) | face = (1 << 8) | 0 = 0x0100
		0,      // Head: 0
		8,      // Body: 8
		8,      // Hands: 8
		8,      // Legs: 8
		8,      // Feet: 8
		0,      // Main: 0
		0,      // Sub: 0
		0,      // Ranged: 0
	}
}

func CreateFakeGrapListPacket() *serverPackets.GrapListPacket {
	packet := &serverPackets.GrapListPacket{
		GrapIDTbl: CreateFakeGrapIDTbl(),
	}

	return packet
}

// CreateMockLoginPacketFromSQL creates a LoginPacket using the exact data from your SQL dumps
func CreateMockLoginPacketFromSQL() *serverPackets.LoginPacket {
	// From characters table:
	// id=1, account_id=1, name='Bit', nation=0, pos_zone=231, pos_prev_zone=231, posx=0, posy=0, posz=0

	// From character_looks:
	// face=0, race=1 (Hume Male), size=0, head=0, body=8, hands=8, legs=8, feet=8, main=0, sub=0, ranged=0

	// From character_stats:
	// hp=50, mp=50, main_job=1 (WAR), sub_job=0

	// From character_jobs:
	// war=1, mnk=1, whm=1, blm=1, rdm=1, thf=1, all others=0

	// Create character name buffer (16 bytes, null-padded)
	nameBytes := [16]byte{}
	copy(nameBytes[:], "Bit")

	packet := &serverPackets.LoginPacket{
		PosHead: serverPackets.LoginPacketPosHead{
			UniqueNo:     1,     // Character ID from DB
			ActIndex:     0x400, // Standard starting index for players
			Padding06:    0,
			Direction:    0,      // Facing south
			PosX:         0,      // From DB: posx=0
			PosZ:         0,      // From DB: posz=0
			PosY:         0,      // From DB: posy=0
			Flags1:       0x0001, // Standard flags
			Speed:        40,     // Default movement speed
			SpeedBase:    40,
			HPMax:        100,    // 100% health
			ServerStatus: 0x0001, // Normal status
			Flags2:       0,
			Flags3:       0,
			Flags4:       0,
			BtTargetID:   0, // No target
		},

		// Zone information from DB
		ZoneNo:    231, // From DB: pos_zone=231
		MapNumber: 231, // Usually matches ZoneNo

		// Time information
		NTTime:    uint32(time.Now().Unix()),
		NTTimeSec: 0,
		GameTime:  getVanadielTime(),

		// Event data (no active event)
		EventNo:   0,
		EventNum:  0,
		EventPara: 0,
		EventMode: 0,

		// Equipment visual IDs from character_looks
		// Race 8 = Galka, Face 0
		GrapIDTbl: CreateFakeGrapIDTbl(),

		// Music for zone 231 (Port Bastok)
		MusicNum: [5]uint16{
			152, // Day music
			153, // Night music
			111, // Battle solo
			112, // Battle party
			105, // Mount music (generic)
		},

		// Sub-map and weather
		SubMapNumber:      0, // Main area
		WeatherNumber:     0, // Clear weather
		WeatherNumber2:    0,
		WeatherTime:       0,
		WeatherTime2:      0,
		WeatherOffsetTime: 0,

		// Ship timers (not used in zone 236)
		ShipStart:     0,
		ShipEnd:       0,
		IsMonstrosity: 0,

		// Login state
		LoginState: serverPackets.LoginPacketStateGame,

		// Character name from DB
		Name: nameBytes,

		// Certificate values (must be consistent per session)
		Certificate: [2]int32{
			int32(1 * 12345), // Based on char ID
			int32(1 * 67890),
		},

		// Zone sub-areas
		Unknown9C: 0,
		ZoneSubNo: 0,

		// Play statistics (would be from DB in real implementation)
		PlayTime:    3600, // 1 hour
		DeadCounter: 0,

		// Mog house settings
		MyRoomSubMapNumber: 0,
		UnknownA9:          0,
		MyRoomMapNumber:    0,

		// Entity loading
		SendCount: 0,

		// Mog house flags
		MyRoomExitBit: 0,
		MogZoneFlag:   0,

		// Character details from DB
		Dancer: serverPackets.LoginPacketMyRoomDancer{
			RaceID:        1,    // From character_looks: race=1 (Hume Male)
			FaceID:        0,    // From character_looks: face=0
			MainJobID:     1,    // From character_stats: main_job=1 (WAR)
			HairID:        0,    // Using face 0 hair
			CharacterSize: 0,    // From character_looks: size=0
			SubJobID:      0,    // From character_stats: sub_job=0
			GetJobFlag:    0x7F, // Jobs unlocked: WAR, MNK, WHM, BLM, RDM, THF (bits 1-6)

			// Job levels from character_jobs
			JobLevel: [16]uint8{
				0, // None
				1, // WAR: 1
				1, // MNK: 1
				1, // WHM: 1
				1, // BLM: 1
				1, // RDM: 1
				1, // THF: 1
				0, // PLD: 0
				0, // DRK: 0
				0, // BST: 0
				0, // BRD: 0
				0, // RNG: 0
				0, // SAM: 0
				0, // NIN: 0
				0, // DRG: 0
				0, // SMN: 0
			},

			// Base stats for Hume Male (approximate starter values)
			BPBase: [7]uint16{
				50, // STR
				50, // DEX
				50, // VIT
				50, // AGI
				50, // INT
				50, // MND
				50, // CHR
			},

			// Stat adjustments for level 1 WAR
			BPAdj: [7]int16{
				2,  // +2 STR (WAR bonus)
				0,  // +0 DEX
				1,  // +1 VIT (WAR bonus)
				0,  // +0 AGI
				-1, // -1 INT
				-1, // -1 MND
				0,  // +0 CHR
			},

			HPMax:      50, // From character_stats: hp=50
			MPMax:      50, // From character_stats: mp=50
			SubJobFlag: 0,  // Subjob not unlocked (sub_job=0)
			Unknown41:  [3]uint8{0, 0, 0},
		},

		// Configuration data
		ConfData: serverPackets.LoginPacketSaveConf{
			Unknown00: [3]uint32{0, 0, 0},
		},

		// Debug value
		Ex: 0,
	}

	return packet
}

// getVanadielTime calculates the current Vanadiel time
// Vanadiel time runs 25x faster than Earth time
func getVanadielTime() uint32 {
	// Base epoch for Vanadiel time (some arbitrary start point)
	const vanadielEpoch = 1009810800 // Jan 1, 2002 in Unix time

	earthTime := time.Now().Unix()
	earthElapsed := earthTime - vanadielEpoch

	// Vanadiel time moves 25x faster
	vanadielElapsed := earthElapsed * 25

	// Wrap to 32-bit
	return uint32(vanadielElapsed % math.MaxUint32)
}

// CreateFakeItemMaxPacket creates a test ItemMaxPacket with realistic inventory sizes
func CreateFakeItemMaxPacket() *serverPackets.ItemMaxPacket {
	packet := &serverPackets.ItemMaxPacket{
		// ItemNum - maximum container sizes
		ItemNum: [18]uint8{
			30, // Inventory - Starting size (can expand to 80)
			50, // Mog Safe - Default size (can expand to 80)
			0,  // Storage - Locked initially (requires access)
			0,  // Temporary Items - Special container
			0,  // Mog Locker - Locked (requires rental)
			30, // Mog Satchel - Starting size
			0,  // Mog Sack - Locked initially
			0,  // Mog Case - Locked initially
			0,  // Mog Wardrobe - Locked initially
			0,  // Mog Safe 2 - Locked
			0,  // Mog Wardrobe 2 - Locked
			0,  // Mog Wardrobe 3 - Locked
			0,  // Mog Wardrobe 4 - Locked
			0,  // Mog Wardrobe 5 - Not available in era
			0,  // Mog Wardrobe 6 - Not available in era
			0,  // Mog Wardrobe 7 - Not available in era
			0,  // Mog Wardrobe 8 - Not available in era
			0,  // Recycle Bin - Not used
		},

		// Padding
		Padding16: [14]uint8{},

		// ItemNum2 - available space in containers (usually mirrors ItemNum)
		ItemNum2: [18]uint16{
			30, // Inventory
			50, // Mog Safe
			0,  // Storage
			0,  // Temporary Items
			0,  // Mog Locker
			30, // Mog Satchel
			0,  // Mog Sack
			0,  // Mog Case
			0,  // Mog Wardrobe
			0,  // Mog Safe 2
			0,  // Mog Wardrobe 2
			0,  // Mog Wardrobe 3
			0,  // Mog Wardrobe 4
			0,  // Mog Wardrobe 5
			0,  // Mog Wardrobe 6
			0,  // Mog Wardrobe 7
			0,  // Mog Wardrobe 8
			0,  // Recycle Bin
		},

		// Padding
		Padding48: [28]uint8{},
	}

	// Mirror to ItemNum2
	for i := 0; i < 18; i++ {
		packet.ItemNum2[i] = uint16(packet.ItemNum[i])
	}

	return packet
}
