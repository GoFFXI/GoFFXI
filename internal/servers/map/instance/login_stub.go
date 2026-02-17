package instance

import "github.com/GoFFXI/GoFFXI/internal/database"

type loginStub struct {
	PlayTime       uint32
	MyRoomMapID    uint16
	MyRoomExitBit  uint8
	MogZoneFlag    uint8
	Music          [5]uint16
	JobLevels      [16]uint8
	UnlockedJobs   uint32
	SubJobUnlocked bool
}

var defaultLoginStub = loginStub{
	PlayTime:      3600,
	MyRoomMapID:   0x0100,
	MyRoomExitBit: 1,
	MogZoneFlag:   1,
	Music:         [5]uint16{152, 153, 111, 112, 105},
	JobLevels: [16]uint8{
		10, 8, 7, 6, 5, 5, 4, 3,
		3, 2, 2, 2, 1, 1, 1, 1,
	},
	UnlockedJobs:   0x7F,
	SubJobUnlocked: true,
}

func stubbedLoginData(character *database.Character) loginStub {
	// Future implementation can branch on character data.
	return defaultLoginStub
}

func boolToUint8(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}
