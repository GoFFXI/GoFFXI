package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type LoginPacketState uint32

const (
	LoginPacketType = 0x000A
	LoginPacketSize = 0x0104
)

const (
	// None.
	LoginPacketStateNone LoginPacketState = iota

	// The local client is within their residence.
	LoginPacketStateMyRoom

	// The local client is within the normal game areas.
	LoginPacketStateGame

	// The local client is performing a POL exit.
	//
	// This feature has been deprecated and no longer functions.
	LoginPacketStatePOLExit

	// The local client is exiting the job menu.
	//
	// This state is no longer used as the old job menu has been removed.
	LoginPacketStateJobExit

	// The local client is performing a POL exit from within their residence.
	//
	// This feature has been deprecated and no longer functions.
	LoginPacketStatePOLExitMyRoom

	// N/A.
	LoginPacketStateEnd
)

type LoginPacketPosHead struct {
	// The local players server id.
	UniqueNo uint32

	// The local players target index.
	ActIndex uint16

	// Padding; unused.
	//
	// This value originally was used as the SendFlg. However, this packet no longer
	// uses this value and is now considered padding in this case.
	Padding06 uint8

	// The local players rotation direction.
	Direction int8

	// The local players X position.
	PosX float32

	// The local players Y position.
	PosZ float32

	// The local players Y position.
	PosY float32

	// Bit flags holding different purposes about the local player.
	Flags1 uint32

	// The local players speed.
	Speed uint8

	// The local players speed base.
	SpeedBase uint8

	// The local players health percentage.
	HPMax uint8

	// The local players server status.
	ServerStatus uint8

	// Bit flags holding different purposes about the local player.
	Flags2 uint32

	// Bit flags holding different purposes about the local player.
	Flags3 uint32

	// Bit flags holding different purposes about the local player.
	Flags4 uint32

	// The local players battle target information.
	BtTargetID uint32
}

type LoginPacketMyRoomDancer struct {
	// The local players race id.
	RaceID uint16

	// The local players face id.
	FaceID uint16

	// The local players main job id.
	MainJobID uint8

	// The local players hair id.
	HairID uint8

	// The local players model size.
	CharacterSize uint8

	// The local players sub job id.
	SubJobID uint8

	// The local players unlocked job flags.
	GetJobFlag uint32

	// The array of the local players job levels.
	JobLevel [16]uint8

	// The local players base stats.
	BPBase [7]uint16

	// The local players stat adjustments.
	BPAdj [7]int16

	// The local players max health.
	HPMax int32

	// The local players max mana.
	MPMax int32

	// The local players sub job flag.
	//
	// This value is used to determine if the player has unlocked and can change
	// their sub job.
	SubJobFlag uint8

	// Unknown.
	//
	// It is assumed this array is padding and unused.
	Unknown41 [3]uint8
}

type LoginPacketSaveConf struct {
	// Unknown.
	//
	// This array of data holds the clients server-side savable configurations.
	//
	// Check the documentation for more details.
	Unknown00 [3]uint32
}

type LoginPacket struct {
	// Sub-structure containing general update information about the local client.
	//
	// Check the documentation for more details.
	PosHead LoginPacketPosHead

	// The zone number.
	ZoneNo uint32

	// The time system time value in seconds.
	NTTime uint32

	// The time system count value.
	NTTimeSec uint32

	// The game time value.
	GameTime uint32

	// The event no value.
	//
	// This value is used with EventNum to determine which event data to be loaded.
	EventNo uint16

	// The map number.
	//
	// This value will generally match the ZoneNo value for most zones.
	MapNumber uint16

	// The clients equipment model visual ids.
	//
	// 0 = race/hair, 1 = head, 2 = body, 3 = hands, 4 = legs,
	// 5 = feet, 6 = main hand, 7 = sub hand, 8 = ranged.
	GrapIDTbl [9]uint16

	// The array of music values to be used for the various purposes within the zone.
	//
	// 0 = Day, 1 = Night, 2 = Battle Solo, 3 = Battle Party, 4 = Mount
	MusicNum [5]uint16

	// The sub map number.
	//
	// This value represents the inner-region within a zone that the player is located
	// within. If the zone does not have sub-regions, then this value will generally
	// be 0. These regions can be used to separate the client from within common reused
	// events such as airships. (For example, the airship counters/rooms/docks in Port
	// Jeuno are each separate regions.)
	SubMapNumber uint16

	// The event num value.
	//
	// This value is used with EvetNo to determine which event data to be loaded.
	EventNum uint16

	// The event para value.
	//
	// This value is used as the event id to determine which event opcode block to execute.
	EventPara uint16

	// The event mode.
	//
	// This value is used as a set of flags for the event system.
	EventMode uint16

	// The zone weather values.
	//
	// Check the documentation for more details.
	WeatherNumber uint16

	// The zone weather values.
	//
	// Check the documentation for more details.
	WeatherNumber2 uint16

	// The zone weather values.
	//
	// Check the documentation for more details.
	WeatherTime uint32

	// The zone weather values.
	//
	// Check the documentation for more details.
	WeatherTime2 uint32

	// The zone weather values.
	//
	// Check the documentation for more details.
	WeatherOffsetTime uint32

	// The ship start time value to initialize the ship system with.
	ShipStart uint32

	// The ship end time value to initialize the ship system with.
	ShipEnd uint16

	// Flag that states if Monstrosity is active.
	IsMonstrosity uint16

	// The clients login state.
	LoginState LoginPacketState

	// The clients character name.
	Name [16]byte

	// The clients certificate values.
	//
	// These values are unique to the current client. The client sends these values
	// back to the server with every packet. (Part of the 0x001C bytes of header data
	// with each packet.) These values MUST match what the client was last sent from
	// the server, otherwise it will be disconnected from the server. The server checks
	// this value constantly and will begin to R0 the client as soon as its invalid.
	Certificate [2]int32

	// The purpose of this value is unknown.
	//
	// The client stores this value in the main GC_ZONE instance, but never uses it
	// afterward. It has been observed to match the value sent as ZoneSubNo or usually
	// set to 0.
	Unknown9C uint16

	// The zone sub number value.
	//
	// This value is used for zones that are instanced or have multiple inner maps in
	// the same area. For example, the battle content Sortie makes use of several zones
	// that contain sub-areas using this value. To load the client into the area Outer
	// Ra'Kaznar [U2] for Sortie, the client would set:
	//
	// ZoneNo: 133 / MapNumber: 133 / ZoneSubNo: 1031
	//
	// The client uses this value when calculating what DAT files to be loaded for the
	// zone.
	ZoneSubNo uint16

	// The clients play time offset value.
	PlayTime uint32

	// The client death counter value.
	DeadCounter uint32

	// The clients mog house sub map number.
	//
	// The client uses this value in various checks to determine if you are allowed to
	// place/move furniture, which kind of exit menu is displayed, etc.
	MyRoomSubMapNumber uint8

	// The purpose of this value is unknown.
	//
	// The client stores this value in the main GC_ZONE instance, but never uses it
	// afterward.
	UnknownA9 uint8

	// The clients mog house map number.
	//
	// This value represents the model id of the area to be loaded for the players
	// mog house. The value will vary based on some additional circumstances. For
	// example, if the player has their mog house registered to one of the main three
	// nations, then the value will change based on if they are aligned to that nation
	// or if they are a guest.
	//
	// Check the documentation for more details.
	MyRoomMapNumber uint16

	// Entity (NPC) load count limiter value.
	//
	// The server can use this value to lock the client in place and prevent movement
	// upon first entering a zone. This value represents the number of flagged NPC
	// entities (referred to as king) that must load (via 0x000E packets sent from the
	// server) before the client is allowed to move. When the client is put into this
	// state, rendering additional entities is paused and allows for all needed entities
	// to populate. (This is generally used to allow on-zone-in cutscenes to load their
	// needed entities data first before playing the cutscene.)
	//
	// In the event the client fails to load the expected SendCount number of entities,
	// the client has a built-in timeout as a failsafe when put into this mode. If the
	// client does not reach the expected count within 6 seconds, it will be released
	// and the event will play regardless.
	SendCount uint16

	// Value that controls the exit menu type when leaving the mog house.
	//
	// The naming of this variable has likely changed since PS2 beta as it no longer is
	// treated as a single bit flag. Instead, it is now a value (ranging from 0 to 9)
	// which determines the kind of menu that will be displayed when leaving each of
	// the various residential areas of the game.
	//
	// Check the documentation for more details.
	MyRoomExitBit uint8

	// Flag that states if the current zone has access to the mog menu.
	//
	// This flag is set in areas that have nomad Moogles.
	MogZoneFlag uint8

	// Sub-structure containing general update information about the local clients character.
	//
	// Check the documentation for more details.
	Dancer LoginPacketMyRoomDancer

	// Sub-structure containing client configuration data.
	//
	// Check the documentation for more details.
	ConfData LoginPacketSaveConf

	// The purpose of this value is unknown.
	//
	// This is a debug related value that is only used in a disabled/hidden debug
	// function in the client. It is simply printed to the screen and is not used
	// otherwise.
	Ex uint32
}

func (p *LoginPacket) Type() uint16 {
	return LoginPacketType
}

func (p *LoginPacket) Size() uint16 {
	return LoginPacketSize
}

func (p *LoginPacket) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write all fields in order
	if err := binary.Write(buf, binary.LittleEndian, p); err != nil {
		return nil, fmt.Errorf("failed to write packet: %w", err)
	}

	return buf.Bytes(), nil
}
