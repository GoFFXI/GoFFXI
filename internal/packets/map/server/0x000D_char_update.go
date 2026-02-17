package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	CharUpdatePacketType = 0x000D
)

type CharUpdateSendFlags uint8

const (
	CharUpdateFlagPosition    CharUpdateSendFlags = 1 << 0
	CharUpdateFlagClaimStatus CharUpdateSendFlags = 1 << 1
	CharUpdateFlagGeneral     CharUpdateSendFlags = 1 << 2
	CharUpdateFlagName        CharUpdateSendFlags = 1 << 3
	CharUpdateFlagModel       CharUpdateSendFlags = 1 << 4
	CharUpdateFlagDespawn     CharUpdateSendFlags = 1 << 5
)

// CharUpdatePacket represents GP_SERV_COMMAND_CHAR_PC in a reduced form.
// https://github.com/atom0s/XiPackets/tree/main/world/server/0x000D
type CharUpdatePacket struct {
	UniqueID           uint32 // server-side actor id
	ActIndex           uint16 // target index used by client
	SendFlags          CharUpdateSendFlags
	Direction          int8    // facing direction
	PosX               float32 // world X
	PosZ               float32 // world Z
	PosY               float32 // world Y
	Flags0             uint32  // movement/time/target bits
	Speed              uint8   // movement speed
	SpeedBase          uint8   // base speed
	HPPercent          uint8   // current HP percent
	ServerStatus       uint8   // animation/state
	Flags1             uint32  // anonymity/graph size/LS/GM flags
	Flags2             uint32  // color/PvP/named/single flags
	Flags3             uint32  // mentor/trust/geo/various toggle flags
	TargetID           uint32  // claim/target id
	CostumeID          uint16  // costume override
	BallistaInfo       uint8   // ballista info
	Flags4             uint8   // trial/job master flags
	CustomProperties   [2]uint32
	PetActIndex        uint16
	MonstrosityFlags   uint16
	MonstrosityNameID1 uint8
	MonstrosityNameID2 uint8
	Flags5             uint8 // geo/indi flags
	ModelHitboxSize    uint8
	Flags6             uint32 // mount id encoded in upper bits
	GrapIDTbl          [9]uint16
	Name               [16]byte
	NameLength         uint8
}

func (p *CharUpdatePacket) Type() uint16 {
	return CharUpdatePacketType
}

// Size is not relied upon by the sender (it uses the serialized length).
// Keep it zero to avoid misleading callers.
func (p *CharUpdatePacket) Size() uint16 { return 0 }

func (p *CharUpdatePacket) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	writers := []interface{}{
		p.UniqueID,
		p.ActIndex,
	}

	for _, v := range writers {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			return nil, fmt.Errorf("write char update field: %w", err)
		}
	}

	buf.WriteByte(uint8(p.SendFlags))
	buf.WriteByte(uint8(p.Direction))

	more := []interface{}{
		p.PosX,
		p.PosZ,
		p.PosY,
		p.Flags0,
	}

	for _, v := range more {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			return nil, fmt.Errorf("write char update field: %w", err)
		}
	}

	buf.WriteByte(p.Speed)
	buf.WriteByte(p.SpeedBase)
	buf.WriteByte(p.HPPercent)
	buf.WriteByte(p.ServerStatus)

	trailing := []interface{}{
		p.Flags1,
		p.Flags2,
		p.Flags3,
		p.TargetID,
		p.CostumeID,
	}

	for _, v := range trailing {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			return nil, fmt.Errorf("write char update field: %w", err)
		}
	}

	buf.WriteByte(p.BallistaInfo)
	buf.WriteByte(p.Flags4)

	extra := []interface{}{
		p.CustomProperties[0],
		p.CustomProperties[1],
		p.PetActIndex,
		p.MonstrosityFlags,
	}

	for _, v := range extra {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			return nil, fmt.Errorf("write char update field: %w", err)
		}
	}

	buf.WriteByte(p.MonstrosityNameID1)
	buf.WriteByte(p.MonstrosityNameID2)
	buf.WriteByte(p.Flags5)
	buf.WriteByte(p.ModelHitboxSize)

	if err := binary.Write(buf, binary.LittleEndian, p.Flags6); err != nil {
		return nil, fmt.Errorf("write char update field: %w", err)
	}

	for _, value := range p.GrapIDTbl {
		if err := binary.Write(buf, binary.LittleEndian, value); err != nil {
			return nil, fmt.Errorf("write char update grap id: %w", err)
		}
	}

	if _, err := buf.Write(p.Name[:]); err != nil {
		return nil, fmt.Errorf("write char update name: %w", err)
	}

	const nameOffset = 90

	data := make([]byte, buf.Len())
	copy(data, buf.Bytes())

	if nameLen := int(p.NameLength); nameLen > 0 {
		if nameLen > len(p.Name) {
			nameLen = len(p.Name)
		}
		targetLen := nameOffset + nameLen + 4
		if targetLen > len(data) {
			data = append(data, make([]byte, targetLen-len(data))...)
		} else if targetLen < len(data) {
			data = data[:targetLen]
		}
	}

	if rem := len(data) % 4; rem != 0 {
		data = append(data, make([]byte, 4-rem)...)
	}

	return data, nil
}
