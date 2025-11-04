package view

import (
	"bytes"
	"crypto/md5" //nolint:gosec // game has to have this
	"encoding/binary"
	"fmt"

	"github.com/GoFFXI/login-server/internal/constants"
)

const (
	CommandRequestLobbyLogin      = 0x0026
	CommandResponseLobbyLogin     = 0x0005
	ErrorUnsupportedClientVersion = 0x24

	MaskBaseGame             = 0x0001
	MaskRiseOfZilart         = 0x0002
	MaskChainsOfPromathia    = 0x0004
	MaskTreasuresOfAhtUrhgan = 0x0008
	MaskWingsOfTheGoddess    = 0x0010
	MaskACrystallineProphecy = 0x0020
	MaskAMoogleKupoDEtat     = 0x0040
	MaskAShantottoAscension  = 0x0080
	MaskVisionsOfAbyssea     = 0x0100
	MaskScarsOfAbyssea       = 0x0200
	MaskHeroesOfAbyssea      = 0x0400
	MaskSeekersOfAdoulin     = 0x0800

	MaskSecureToken  = 0x0001
	MaskMogWardrobe3 = 0x0004
	MaskMogWardrobe4 = 0x0008
	MaskMogWardrobe5 = 0x0010
	MaskMogWardrobe6 = 0x0020
	MaskMogWardrobe7 = 0x0040
	MaskMogWardrobe8 = 0x0080
)

// https://github.com/atom0s/XiPackets/blob/main/lobby/C2S_0x0026_RequestLobbyLogin.md
type RequestLobbyLogin struct {
	PlayOnlineAccountID [16]byte
	ClientCode          uint8
	AuthCode            uint8
	VersionCode         [16]byte
	ExcodeClient        uint32
	Unknown01           uint32
	Unknown02           uint32
	Unknown03           uint32
	Unknown04           uint32
}

func NewRequestLobbyLogin(data []byte) (*RequestLobbyLogin, error) {
	if len(data) != 0x0098 {
		return nil, fmt.Errorf("invalid data size for RequestLobbyLogin: expected 152 bytes, got %d", len(data))
	}

	data = data[28:] // strip the header (28 bytes)
	request := &RequestLobbyLogin{}
	buf := bytes.NewReader(data)

	// Read the entire struct at once (works because all fields are fixed-size)
	err := binary.Read(buf, binary.LittleEndian, request)
	if err != nil {
		return nil, err
	}

	return request, nil
}

// Total size: 40 bytes
type ResponseLobbyLogin struct {
	// Header (28 bytes)
	PacketSize uint32   // Total packet size (40 bytes)
	Terminator uint32   // Always 0x46465849 ("IXFF")
	Command    uint32   // OpCode 0x0005
	Identifier [16]byte // MD5 hash of the packet

	// Body (12 bytes)
	Key              uint32 // 0x4FE050AD
	ExpansionBitmask uint32 // Bitmask of enabled expansions
	FeaturesBitmask  uint32 // Bitmask of enabled features
}

// NewResponseLobbyLogin creates a new login response packet
func NewResponseLobbyLogin(expansions, features uint32) (*ResponseLobbyLogin, error) {
	response := &ResponseLobbyLogin{
		PacketSize:       40, // Fixed size for this packet
		Terminator:       constants.ResponsePacketTerminator,
		Command:          CommandResponseLobbyLogin,
		Identifier:       [16]byte{}, // Will be filled with MD5 hash
		Key:              0xAD50E04F, // Note: Little-endian representation of 0x4FE050AD
		ExpansionBitmask: expansions,
		FeaturesBitmask:  features,
	}

	// Calculate MD5 hash
	if err := response.CalculateAndSetHash(); err != nil {
		return nil, err
	}

	return response, nil
}

// CalculateAndSetHash calculates the MD5 hash of the packet and sets the identifier
func (r *ResponseLobbyLogin) CalculateAndSetHash() error {
	// Temporarily clear identifier
	r.Identifier = [16]byte{}

	// Serialize to calculate hash
	data, err := r.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize for hash: %w", err)
	}

	// Calculate and set MD5 hash
	hash := md5.Sum(data) //nolint:gosec // game has to have this
	r.Identifier = hash

	return nil
}

// Serialize converts the packet to bytes for transmission
func (r *ResponseLobbyLogin) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write all fields in order
	if err := binary.Write(buf, binary.LittleEndian, r); err != nil {
		return nil, fmt.Errorf("failed to write packet: %w", err)
	}

	return buf.Bytes(), nil
}

// https://github.com/atom0s/XiPackets/blob/main/lobby/C2S_0x0026_RequestLobbyLogin.md
func (s *ViewServer) handleRequestLobbyLogin(sessionCtx *sessionContext, request []byte) bool {
	logger := sessionCtx.logger.With("request", "lobby-login")
	logger.Info("handling request")

	_, err := NewRequestLobbyLogin(request)
	if err != nil {
		logger.Error("failed to parse request", "error", err)
		return true
	}

	expansions := s.GenerateExpansionBitmask()
	features := s.GenerateFeaturesBitmask()

	response, err := NewResponseLobbyLogin(expansions, features)
	if err != nil {
		logger.Error("failed to create response", "error", err)
		return true
	}

	responsePacket, err := response.Serialize()
	if err != nil {
		logger.Error("failed to serialize response", "error", err)
		return true
	}

	// Send the response packet
	logger.Info("sending response")
	_, err = sessionCtx.conn.Write(responsePacket)
	if err != nil {
		logger.Error("failed to send response", "error", err)
		return true
	}

	return false
}

func (s *ViewServer) GenerateExpansionBitmask() uint32 {
	expansionMap := map[uint32]func() bool{
		MaskRiseOfZilart:         func() bool { return s.Config().RiseOfZilartEnabled },
		MaskChainsOfPromathia:    func() bool { return s.Config().ChainsOfPromathiaEnabled },
		MaskTreasuresOfAhtUrhgan: func() bool { return s.Config().TreasuresOfAhtUrhganEnabled },
		MaskWingsOfTheGoddess:    func() bool { return s.Config().WingsOfTheGoddessEnabled },
		MaskACrystallineProphecy: func() bool { return s.Config().ACrystallineProphecyEnabled },
		MaskAMoogleKupoDEtat:     func() bool { return s.Config().AMoogleKupoDEtatEnabled },
		MaskAShantottoAscension:  func() bool { return s.Config().AShantottoAscensionEnabled },
		MaskVisionsOfAbyssea:     func() bool { return s.Config().VisionsOfAbysseaEnabled },
		MaskScarsOfAbyssea:       func() bool { return s.Config().ScarsOfAbysseaEnabled },
		MaskHeroesOfAbyssea:      func() bool { return s.Config().HeroesOfAbysseaEnabled },
		MaskSeekersOfAdoulin:     func() bool { return s.Config().SeekersOfAdoulinEnabled },
	}

	mask := uint32(MaskBaseGame)
	for maskBit, enabled := range expansionMap {
		if enabled() {
			mask |= maskBit
		}
	}

	return mask
}

func (s *ViewServer) GenerateFeaturesBitmask() uint32 {
	featureMap := map[uint32]func() bool{
		MaskSecureToken:  func() bool { return s.Config().SecureTokenEnabled },
		MaskMogWardrobe3: func() bool { return s.Config().MogWardrobe3Enabled },
		MaskMogWardrobe4: func() bool { return s.Config().MogWardrobe4Enabled },
		MaskMogWardrobe5: func() bool { return s.Config().MogWardrobe5Enabled },
		MaskMogWardrobe6: func() bool { return s.Config().MogWardrobe6Enabled },
		MaskMogWardrobe7: func() bool { return s.Config().MogWardrobe7Enabled },
		MaskMogWardrobe8: func() bool { return s.Config().MogWardrobe8Enabled },
	}

	var mask uint32
	for maskBit, enabled := range featureMap {
		if enabled() {
			mask |= maskBit
		}
	}

	return mask
}
