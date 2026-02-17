package instance

import (
	"math"
	"time"

	"github.com/GoFFXI/GoFFXI/internal/database"
	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
	serverPackets "github.com/GoFFXI/GoFFXI/internal/packets/map/server"
)

func (s *InstanceWorker) processLoginPacket(routedPacket mapPackets.RoutedPacket) {
	s.Logger().Info("processing login packet", "clientAddr", routedPacket.ClientAddr)

	character, err := s.DB().GetCharacterByID(s.ctx, routedPacket.CharacterID)
	if err != nil {
		s.Logger().Warn("failed to load character for login", "characterID", routedPacket.CharacterID, "error", err)
	}

	var looks *database.CharacterLooks
	if cl, err := s.DB().GetCharacterLooksByID(s.ctx, routedPacket.CharacterID); err == nil {
		looks = &cl
	}

	var stats *database.CharacterStats
	if cs, err := s.DB().GetCharacterStatsByID(s.ctx, routedPacket.CharacterID); err == nil {
		stats = &cs
	}

	// send a character update packet first so the client has entity context
	charUpdatePacket := CreateCharacterUpdatePacket(&character, looks, stats)
	if err := s.sendPacket(routedPacket.ClientAddr, charUpdatePacket); err != nil {
		s.Logger().Warn("failed to send char update", "clientAddr", routedPacket.ClientAddr, "error", err)
	}

	equipClearPacket := serverPackets.EquipClearPacket{}
	equipListPackets := CreateEquipListPackets(&character)
	graphListPacket := CreateGrapListPacket(looks, &character)
	itemMaxPacket := CreateItemMaxPacket(&character)
	loginPacket := CreateLoginPacketFromCharacter(&character, looks, stats)
	enterZonePacket := CreateEnterZonePacket()

	clientAddr := routedPacket.ClientAddr
	if err := s.sendPacket(clientAddr, &equipClearPacket); err != nil {
		s.Logger().Warn("failed to send equip clear", "clientAddr", clientAddr, "error", err)
	}
	for _, equipPacket := range equipListPackets {
		if err := s.sendPacket(clientAddr, equipPacket); err != nil {
			s.Logger().Warn("failed to send equip list", "clientAddr", clientAddr, "slot", equipPacket.EquipKind, "error", err)
		}
	}
	if err := s.sendPacket(clientAddr, graphListPacket); err != nil {
		s.Logger().Warn("failed to send grap list", "clientAddr", clientAddr, "error", err)
	}
	if err := s.sendPacket(clientAddr, itemMaxPacket); err != nil {
		s.Logger().Warn("failed to send item max", "clientAddr", clientAddr, "error", err)
	}
	if err := s.sendPacket(clientAddr, loginPacket); err != nil {
		s.Logger().Warn("failed to send login", "clientAddr", clientAddr, "error", err)
	}
	if err := s.sendPacket(clientAddr, enterZonePacket); err != nil {
		s.Logger().Warn("failed to send enter zone", "clientAddr", clientAddr, "error", err)
	}
}

func CreateFakeGrapIDTbl() [9]uint16 {
	return [9]uint16{0x0100, 0, 8, 8, 8, 8, 0, 0, 0}
}

func CreateEquipListPackets(character *database.Character) []*serverPackets.EquipListPacket {
	stubs := stubbedEquipmentForCharacter(character)
	if len(stubs) == 0 {
		stubs = defaultEquipmentStubs()
	}

	packets := make([]*serverPackets.EquipListPacket, 0, len(stubs))
	for _, entry := range stubs {
		packets = append(packets, &serverPackets.EquipListPacket{
			PropertyItemIndex: entry.ItemIndex,
			EquipKind:         entry.Slot,
			Category:          entry.ContainerID,
		})
	}

	return packets
}

func CreateGrapListPacket(looks *database.CharacterLooks, character *database.Character) *serverPackets.GrapListPacket {
	packet := &serverPackets.GrapListPacket{GrapIDTbl: buildCharacterGrapIDs(looks)}
	if len(packet.GrapIDTbl) == 0 {
		packet.GrapIDTbl = CreateFakeGrapIDTbl()
	}
	return packet
}

func CreateEnterZonePacket() *serverPackets.EnterZonePacket {
	packet := &serverPackets.EnterZonePacket{}
	copy(packet.EnterZoneTbl[:], stubbedEnterZoneHistory())
	return packet
}

func CreateLoginPacketFromCharacter(character *database.Character, looks *database.CharacterLooks, stats *database.CharacterStats) *serverPackets.LoginPacket {
	stub := stubbedLoginData(character)

	name := "Adventurer"
	zone := uint32(231)
	posX, posY, posZ := float32(0), float32(0), float32(0)
	uniqueID := uint32(1)

	if character != nil && character.ID != 0 {
		uniqueID = character.ID
		zone = uint32(character.PosZone)
		posX = character.PosX
		posY = character.PosY
		posZ = character.PosZ
		if character.Name != "" {
			name = character.Name
		}
	}

	grapIDs := buildCharacterGrapIDs(looks)

	gender := uint32(0)
	sizeIdx := uint8(0)
	raceID := uint16(0)
	faceID := uint16(0)
	if looks != nil {
		gender = getGenderFlag(looks.Race)
		sizeIdx = looks.Size & 0x3
		raceID = uint16(looks.Race)
		faceID = uint16(looks.Face)
	}

	sizeBit := uint32(1 << sizeIdx)
	flags2 := ((gender * 128) + sizeBit) << 8

	hpp := uint8(100)
	hpMax := int32(100)
	mpMax := int32(50)
	mainJob := uint8(1)
	subJob := uint8(0)
	if stats != nil {
		if stats.HP == 0 {
			hpp = 0
		} else if stats.HP < 100 {
			hpp = uint8(stats.HP)
		}
		hpMax = int32(stats.HP)
		mpMax = int32(stats.MP)
		if stats.MainJob != 0 {
			mainJob = stats.MainJob
		}
		subJob = stats.SubJob
	}

	now := uint32(time.Now().Unix())

	packet := &serverPackets.LoginPacket{
		PosHead: serverPackets.LoginPacketPosHead{
			UniqueNo:     uniqueID,
			ActIndex:     0x0400,
			Padding06:    0,
			Direction:    0,
			PosX:         posX,
			PosZ:         posY,
			PosY:         posZ,
			Flags1:       0,
			Speed:        40,
			SpeedBase:    40,
			HPMax:        hpp,
			ServerStatus: 0,
			Flags2:       flags2,
			Flags3:       0,
			Flags4:       0x0100,
			BtTargetID:   0,
		},
		ZoneNo:            zone,
		MapNumber:         uint16(zone),
		NTTime:            now,
		NTTimeSec:         now,
		GameTime:          getVanadielTime(),
		EventNo:           0,
		EventNum:          0,
		EventPara:         0,
		EventMode:         0,
		GrapIDTbl:         grapIDs,
		MusicNum:          stub.Music,
		SubMapNumber:      0,
		WeatherNumber:     0,
		WeatherNumber2:    0,
		WeatherTime:       0,
		WeatherTime2:      0,
		WeatherOffsetTime: 0,
		ShipStart:         0,
		ShipEnd:           0,
		IsMonstrosity:     0,
		LoginState:        serverPackets.LoginPacketStateGame,
		Name:              nameToBytes(name),
		Certificate: [2]int32{
			int32(uniqueID * 12345),
			int32(uniqueID * 67890),
		},
		Unknown9C:          0,
		ZoneSubNo:          0,
		PlayTime:           stub.PlayTime,
		DeadCounter:        0,
		MyRoomSubMapNumber: 0,
		UnknownA9:          0,
		MyRoomMapNumber:    stub.MyRoomMapID,
		SendCount:          0,
		MyRoomExitBit:      stub.MyRoomExitBit,
		MogZoneFlag:        stub.MogZoneFlag,
		Dancer: serverPackets.LoginPacketMyRoomDancer{
			RaceID:        raceID,
			FaceID:        faceID,
			MainJobID:     mainJob,
			HairID:        0,
			CharacterSize: sizeIdx,
			SubJobID:      subJob,
			GetJobFlag:    stub.UnlockedJobs,
			JobLevel:      stub.JobLevels,
			BPBase:        [7]uint16{50, 50, 50, 50, 50, 50, 50},
			BPAdj:         [7]int16{0, 0, 0, 0, 0, 0, 0},
			HPMax:         hpMax,
			MPMax:         mpMax,
			SubJobFlag:    boolToUint8(stub.SubJobUnlocked),
			Unknown41:     [3]uint8{0, 0, 0},
		},
		ConfData: serverPackets.LoginPacketSaveConf{
			Unknown00: [3]uint32{0, 0, 0},
		},
		Ex: 1,
	}

	return packet
}

func CreateCharacterUpdatePacket(character *database.Character, looks *database.CharacterLooks, stats *database.CharacterStats) *serverPackets.CharUpdatePacket {
	name := "Adventurer"
	posX, posY, posZ := float32(0), float32(0), float32(0)
	uniqueID := uint32(1)

	if character != nil && character.ID != 0 {
		uniqueID = character.ID
		name = character.Name
		posX = character.PosX
		posY = character.PosY
		posZ = character.PosZ
	}

	grapIDs := buildCharacterGrapIDs(looks)
	graphSize := uint32(0)
	gender := uint32(0)
	if looks != nil {
		graphSize = uint32(looks.Size & 0x3)
		gender = getGenderFlag(looks.Race)
	}

	flags0 := uint32(0)

	flags1 := ((graphSize & 0x3) << 9) | ((gender & 0x1) << 15)

	flags2 := uint32(0)

	flags3 := uint32(defaultBallistaTeam()) << 8
	if isNewAdventurer() {
		flags3 |= 1 << 23
	}

	flags4 := uint8(0)
	flags5 := uint8(0x10)
	flags6 := uint32(0)

	hpp := uint8(100)

	nameBytes := nameToBytes(name)
	nameLen := len(name)
	if nameLen > len(nameBytes) {
		nameLen = len(nameBytes)
	}

	packet := &serverPackets.CharUpdatePacket{
		UniqueID:           uniqueID,
		ActIndex:           0x0400,
		SendFlags:          serverPackets.CharUpdateFlagPosition | serverPackets.CharUpdateFlagClaimStatus | serverPackets.CharUpdateFlagGeneral | serverPackets.CharUpdateFlagName | serverPackets.CharUpdateFlagModel,
		Direction:          0,
		PosX:               posX,
		PosZ:               posY,
		PosY:               posZ,
		Flags0:             flags0,
		Speed:              40,
		SpeedBase:          40,
		HPPercent:          hpp,
		ServerStatus:       0,
		Flags1:             flags1,
		Flags2:             flags2,
		Flags3:             flags3,
		TargetID:           0,
		CostumeID:          0,
		BallistaInfo:       0,
		Flags4:             flags4,
		CustomProperties:   [2]uint32{0, 0},
		PetActIndex:        0,
		MonstrosityFlags:   0,
		MonstrosityNameID1: 0,
		MonstrosityNameID2: 0,
		Flags5:             flags5,
		ModelHitboxSize:    4,
		Flags6:             flags6,
		GrapIDTbl:          grapIDs,
		Name:               nameBytes,
	}

	packet.NameLength = uint8(nameLen)
	return packet
}

func nameToBytes(name string) [16]byte {
	var buf [16]byte
	copy(buf[:], name)
	return buf
}

func getVanadielTime() uint32 {
	const vanadielEpoch = 1009810800
	earthTime := time.Now().Unix()
	earthElapsed := earthTime - vanadielEpoch
	vanadielElapsed := earthElapsed * 25
	return uint32(vanadielElapsed % math.MaxUint32)
}

func CreateItemMaxPacket(character *database.Character) *serverPackets.ItemMaxPacket {
	stub := stubbedItemMax(character)
	packet := &serverPackets.ItemMaxPacket{
		ItemNum:   stub.ItemNum,
		ItemNum2:  stub.ItemNum2,
		Padding16: [14]uint8{},
		Padding48: [28]uint8{},
	}
	return packet
}

func buildCharacterGrapIDs(looks *database.CharacterLooks) [9]uint16 {
	grapIDs := CreateFakeGrapIDTbl()
	if looks == nil {
		return grapIDs
	}

	grapIDs[0] = uint16(looks.Race)<<8 | uint16(looks.Face)
	grapIDs[1] = looks.Head + 0x1000
	grapIDs[2] = looks.Body + 0x2000
	grapIDs[3] = looks.Hands + 0x3000
	grapIDs[4] = looks.Legs + 0x4000
	grapIDs[5] = looks.Feet + 0x5000
	grapIDs[6] = looks.Main + 0x6000
	grapIDs[7] = looks.Sub + 0x7000
	grapIDs[8] = looks.Ranged + 0x8000
	return grapIDs
}

type itemMaxStub struct {
	ItemNum  [18]uint8
	ItemNum2 [18]uint16
}

func stubbedItemMax(character *database.Character) itemMaxStub {
	return itemMaxStub{
		ItemNum: [18]uint8{
			80, 80, 80, 30, 80, 80, 80, 80, 80,
			80, 80, 80, 80, 80, 80, 80, 80, 80,
		},
		ItemNum2: [18]uint16{
			80, 80, 80, 30, 80, 80, 80, 80, 80,
			80, 80, 80, 80, 80, 80, 80, 80, 80,
		},
	}
}

func stubbedEnterZoneHistory() []byte {
	return []byte{
		0x10, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00,
		0x40, 0x00, 0x00, 0x00,
		0x08, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
}

func getGenderFlag(race uint8) uint32 {
	if race == 0 {
		return 0
	}

	parity := race % 2
	var galkaToggle uint8
	if race > 6 {
		galkaToggle = 1
	}

	return uint32((parity ^ galkaToggle) & 0x1)
}

func defaultBallistaTeam() uint32 {
	return 1 // ALLEGIANCE_TYPE::PLAYER
}

func isNewAdventurer() bool {
	// TODO: track xi.settings player config once client config packets are implemented.
	return true
}
