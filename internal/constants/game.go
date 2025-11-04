package constants

// Response Packets
const (
	ResponsePacketTerminator = 0x46465849 // "IXFF"
)

// Race IDs
const (
	RaceInvalid        uint16 = 0
	RaceHumeMale       uint16 = 1
	RaceHumeFemale     uint16 = 2
	RaceElvaanMale     uint16 = 3
	RaceElvaanFemale   uint16 = 4
	RaceTarutaruMale   uint16 = 5
	RaceTarutaruFemale uint16 = 6
	RaceMithra         uint16 = 7
	RaceGalka          uint16 = 8
)

// Job IDs
const (
	JobNone         uint8 = 0
	JobWarrior      uint8 = 1
	JobMonk         uint8 = 2
	JobWhiteMage    uint8 = 3
	JobBlackMage    uint8 = 4
	JobRedMage      uint8 = 5
	JobThief        uint8 = 6
	JobPaladin      uint8 = 7
	JobDarkKnight   uint8 = 8
	JobBeastmaster  uint8 = 9
	JobBard         uint8 = 10
	JobRanger       uint8 = 11
	JobSamurai      uint8 = 12
	JobNinja        uint8 = 13
	JobDragoon      uint8 = 14
	JobSummoner     uint8 = 15
	JobBlueMage     uint8 = 16
	JobCorsair      uint8 = 17
	JobPuppetmaster uint8 = 18
	JobDancer       uint8 = 19
	JobScholar      uint8 = 20
	JobGeomancer    uint8 = 21
	JobRuneFencer   uint8 = 22
)

// Town/Nation IDs
const (
	TownSandoria uint8 = 0
	TownBastok   uint8 = 1
	TownWindurst uint8 = 2
)

// Size values
const (
	SizeSmall  uint8 = 0
	SizeMedium uint8 = 1
	SizeLarge  uint8 = 2
)
