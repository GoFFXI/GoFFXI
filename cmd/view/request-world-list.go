package view

import (
	"bytes"
	"crypto/md5" //nolint:gosec // game has to have this
	"encoding/binary"
	"fmt"

	"github.com/GoFFXI/login-server/internal/constants"
)

const (
	CommandRequestQueryWorldList = 0x0024
	CommandResponseWorldList     = 0x0023
)

// https://github.com/atom0s/XiPackets/blob/main/lobby/C2S_0x0024_RequestQueryWorldList.md
type RequestQueryWorldList struct {
	Password uint8
}

func NewRequestQueryWorldList(data []byte) (*RequestQueryWorldList, error) {
	// strip the header (28 bytes)
	if len(data) < 29 {
		return nil, fmt.Errorf("insufficient data for RequestQueryWorldList: need at least 29 bytes, got %d", len(data))
	}

	data = data[28:] // strip the header (28 bytes)
	request := &RequestQueryWorldList{}
	buf := bytes.NewReader(data)

	// Read the entire struct at once (works because all fields are fixed-size)
	err := binary.Read(buf, binary.BigEndian, request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

// https://github.com/atom0s/XiPackets/blob/main/lobby/S2C_0x0023_ResponseWorldList.md
type ResponseQueryWorldList struct {
	// Header (28 bytes)
	PacketSize uint32 // Variable based on world count
	Terminator uint32 // Always 0x46465849 ("IXFF")
	Command    uint32
	Identifier [16]byte // MD5 hash of the packet (calculated with this field as zeros)

	// Data
	WorldCount uint32      // Number of worlds
	Worlds     []WorldInfo // Array of world information
}

type WorldInfo struct {
	ID   uint32   // World ID (e.g., 127 for test server)
	Name [16]byte // World name (null-terminated)
}

// NewResponseQueryWorldList creates a new world list response packet
func NewResponseQueryWorldList(worlds []WorldInfo) (*ResponseQueryWorldList, error) {
	packet := &ResponseQueryWorldList{
		Terminator: constants.ResponsePacketTerminator,
		Command:    CommandResponseWorldList,
		Identifier: [16]byte{},          // Will be filled with MD5 hash
		WorldCount: uint32(len(worlds)), //nolint:gosec // length is trusted
		Worlds:     worlds,
	}

	// Calculate packet size: header (28) + count (4) + worlds (18 each for minimal structure)
	packet.PacketSize = uint32(28 + 4 + len(worlds)*20) //nolint:gosec // length is trusted

	// Calculate MD5 hash of the packet
	if err := packet.CalculateAndSetHash(); err != nil {
		return nil, err
	}

	return packet, nil
}

// CalculateAndSetHash calculates the MD5 hash of the packet and sets the identifier
func (r *ResponseQueryWorldList) CalculateAndSetHash() error {
	// Temporarily clear identifier
	r.Identifier = [16]byte{}

	// Serialize to calculate hash
	data, err := r.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize for hash: %w", err)
	}

	// Calculate and set MD5 hash
	r.Identifier = md5.Sum(data) //nolint:gosec // game requires this
	return nil
}

// Serialize converts the response packet to bytes
func (r *ResponseQueryWorldList) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write header
	if err := binary.Write(buf, binary.LittleEndian, r.PacketSize); err != nil {
		return nil, fmt.Errorf("failed to write packet size: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, r.Terminator); err != nil {
		return nil, fmt.Errorf("failed to write terminator: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, r.Command); err != nil {
		return nil, fmt.Errorf("failed to write command: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, r.Identifier); err != nil {
		return nil, fmt.Errorf("failed to write identifier: %w", err)
	}

	// Write world count
	if err := binary.Write(buf, binary.LittleEndian, r.WorldCount); err != nil {
		return nil, fmt.Errorf("failed to write world count: %w", err)
	}

	// Write each world info
	for i, world := range r.Worlds {
		if err := binary.Write(buf, binary.LittleEndian, world); err != nil {
			return nil, fmt.Errorf("failed to write world %d: %w", i, err)
		}
	}

	return buf.Bytes(), nil
}

func (s *ViewServer) handleRequestWorldList(sessionCtx *sessionContext, request []byte) bool {
	logger := sessionCtx.logger.With("request", "query-world-list")
	logger.Info("handling request")

	_, err := NewRequestQueryWorldList(request)
	if err != nil {
		logger.Error("failed to parse request", "error", err)
		return true
	}

	worlds := GetExampleWorldList()
	packet, err := NewResponseQueryWorldList(worlds)
	if err != nil {
		logger.Error("failed to create response", "error", err)
		return true
	}

	data, err := packet.Serialize()
	if err != nil {
		logger.Error("failed to serialize response", "error", err)
		return true
	}

	_, err = sessionCtx.conn.Write(data)
	if err != nil {
		logger.Error("failed to send response", "error", err)
		return true
	}

	return false
}

func CreateWorldInfo(id uint32, name string) WorldInfo {
	world := WorldInfo{
		ID: id,
	}

	if len(name) > 15 {
		name = name[:15]
	}

	// Copy name (max 15 chars + null terminator)
	nameBytes := make([]byte, 16)
	copy(nameBytes, name)
	copy(world.Name[:], nameBytes)

	return world
}

func GetExampleWorldList() []WorldInfo {
	return []WorldInfo{
		CreateWorldInfo(1, "YoyoIsABitch"),
		CreateWorldInfo(2, "FuckedUrMom"),
		CreateWorldInfo(3, "EventideSux"),
	}
}
