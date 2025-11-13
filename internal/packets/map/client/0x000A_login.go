package client

import (
	"bytes"
	"encoding/binary"
	"fmt"

	mapPackets "github.com/GoFFXI/GoFFXI/internal/packets/map"
)

const (
	PacketTypeLogin uint16 = 0x000A
	PacketSizeLogin uint16 = 0x005C
)

// https://github.com/atom0s/XiPackets/blob/main/world/client/0x000A/README.md
type LoginPacket struct {
	Header mapPackets.PacketHeader

	// The packet data checksum.
	//
	// This value is a basic byte-sum value used as a checksum for the packet data to check
	// for tampering. The client adds the values of each byte in the packet (starting at the
	// unknown08 field) and store the total in this value.
	LoginPacketCheck uint8

	// Padding; unused
	Padding05 uint8

	// Unknown.
	//
	// This value is used as a state related value within the client. It will range between 0
	// and 4 depending on what current state the client is in. Check the documentation for
	// more details.
	Unknown06 uint16

	// Unknown.
	//
	// This value is used as an error state related value within the client. It contains two values
	// as the client rotates between values (last-two states) that will be stored. Its values are
	// set during error conditions within the client related to packet traffic. The values that can
	// be set here are between 0 and 32.
	//
	// Check the documentation for more details.
	Unknown08 uint32

	// The server id of the character the client is logging in with.
	UniqueNo uint32

	// This value is no longer used.
	GrapIDTbl [9]uint16

	// Unknown.
	//
	// While the client will set this value into the packet, the value is always null (empty string).
	SName [15]byte

	// Unknown.
	//
	// While the client will set this value into the packet, the value is always null (empty string).
	SAccount [15]byte

	// The unique client ticket checksum value.
	//
	// When the client is building this packet, it will generate a unique ticket value for the client
	// based on several pieces of information. The client will generate an MD5 checksum using the
	// clients account name string and its current blowfish key. The resulting MD5 hash value is stored
	// into this array.
	Ticket [16]uint8

	// Unknown.
	//
	// While the client does use and set this value, it is always set to 0.
	Ver uint32

	// The client platform tag.
	//
	// This value is set to the clients current platform shorthand tag.
	SPlatform [4]uint8

	// The client language id.
	UCliLang uint16

	// Unknown.
	//
	// This value is used as a kind of state related value.
	//
	// The client has an internal system called XiInfo which holds state based or stepping based
	// values throughout different operations within the client. While the client is executing certain
	// functions, it will make use of this system to store information and generally a current 'step'
	// in the process to execute the function. This system allows the client to read/write these values
	// to later determine if it failed to properly execute a given function. The usage of this system
	// is also backed by files on disk, which are the info00.bin and info02.bin files found in the SYS
	// folder of the FINAL FANTASY XI directory.
	//
	// Note: While this parameter has a name similar to how SE names unused padding data, the value
	// is actually used with this name.
	//
	// Check the documentation for more details.
	DammyArea uint16
}

func (p *LoginPacket) validateChecksum(rawData []byte) bool {
	// In Go, we need to calculate the offset manually
	// The checksum starts from the Unknown01 field
	// Offset = ID(2) + Sync(2) + LoginPacketCheck(1) + Padding00(1) + Unknown00(2) = 8 bytes
	const checksumOffset uint16 = 8

	// Calculate checksum starting from Unknown01 to the end
	var calculatedChecksum uint8
	for i := checksumOffset; i < PacketSizeLogin; i++ {
		calculatedChecksum += rawData[i]
	}

	// Compare with the checksum in the packet
	return calculatedChecksum == p.LoginPacketCheck
}

func NewRequestLoginPacket(data []byte) (*LoginPacket, error) {
	// strip the header
	data = data[mapPackets.HeaderSize : mapPackets.HeaderSize+PacketSizeLogin]

	// make sure the data length is correct
	if len(data) != int(PacketSizeLogin) {
		return nil, fmt.Errorf("invalid packet size")
	}

	// parse the packet
	var packet LoginPacket
	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &packet)
	if err != nil {
		return nil, fmt.Errorf("failed to parse login packet: %w", err)
	}

	return &packet, nil
}

func ParseLoginPacket(data []byte) (*LoginPacket, error) {
	// only process packets that have valid checksums
	if !mapPackets.PerformPacketChecksum(data) {
		return nil, fmt.Errorf("invalid packet checksum")
	}

	// extract the packet ID from the header
	packetID := binary.LittleEndian.Uint16(data[mapPackets.HeaderSize:]) & 0x1FF

	// only process login packets for now
	if packetID != PacketTypeLogin {
		return nil, fmt.Errorf("unexpected packet ID: %d", packetID)
	}

	// attempt to parse the login packet
	loginPacket, err := NewRequestLoginPacket(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse login packet: %w", err)
	}

	// we previously validated the overall packet checksum but the FFXI
	// client will also send a checksum within the login packet itself; so
	// let's validate that as well
	if !loginPacket.validateChecksum(data[mapPackets.HeaderSize : mapPackets.HeaderSize+PacketSizeLogin]) {
		return nil, fmt.Errorf("invalid login packet checksum")
	}

	return loginPacket, nil
}
