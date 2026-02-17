package instance

import (
	"github.com/GoFFXI/GoFFXI/internal/database"
	serverPackets "github.com/GoFFXI/GoFFXI/internal/packets/map/server"
)

type equipmentStub struct {
	Slot        serverPackets.EquipKind
	ItemIndex   uint8
	ContainerID uint8
}

var stubbedCharacterEquipment = []equipmentStub{
	{Slot: serverPackets.EquipKindMain, ItemIndex: 5, ContainerID: 0},
	{Slot: serverPackets.EquipKindSub, ItemIndex: 6, ContainerID: 0},
	{Slot: serverPackets.EquipKindRanged, ItemIndex: 7, ContainerID: 0},
	{Slot: serverPackets.EquipKindAmmo, ItemIndex: 8, ContainerID: 0},
	{Slot: serverPackets.EquipKindHead, ItemIndex: 9, ContainerID: 0},
	{Slot: serverPackets.EquipKindBody, ItemIndex: 10, ContainerID: 0},
	{Slot: serverPackets.EquipKindHands, ItemIndex: 11, ContainerID: 0},
	{Slot: serverPackets.EquipKindLegs, ItemIndex: 12, ContainerID: 0},
	{Slot: serverPackets.EquipKindFeet, ItemIndex: 13, ContainerID: 0},
	{Slot: serverPackets.EquipKindNeck, ItemIndex: 14, ContainerID: 0},
	{Slot: serverPackets.EquipKindWaist, ItemIndex: 15, ContainerID: 0},
	{Slot: serverPackets.EquipKindRightEar, ItemIndex: 16, ContainerID: 0},
	{Slot: serverPackets.EquipKindLeftEar, ItemIndex: 17, ContainerID: 0},
	{Slot: serverPackets.EquipKindRightRing, ItemIndex: 18, ContainerID: 0},
	{Slot: serverPackets.EquipKindLeftRing, ItemIndex: 19, ContainerID: 0},
	{Slot: serverPackets.EquipKindBack, ItemIndex: 20, ContainerID: 0},
}

func stubbedEquipmentForCharacter(character *database.Character) []equipmentStub {
	return stubbedCharacterEquipment
}

func defaultEquipmentStubs() []equipmentStub {
	order := []serverPackets.EquipKind{
		serverPackets.EquipKindMain,
		serverPackets.EquipKindSub,
		serverPackets.EquipKindRanged,
		serverPackets.EquipKindAmmo,
		serverPackets.EquipKindHead,
		serverPackets.EquipKindBody,
		serverPackets.EquipKindHands,
		serverPackets.EquipKindLegs,
		serverPackets.EquipKindFeet,
		serverPackets.EquipKindNeck,
		serverPackets.EquipKindWaist,
		serverPackets.EquipKindRightEar,
		serverPackets.EquipKindLeftEar,
		serverPackets.EquipKindRightRing,
		serverPackets.EquipKindLeftRing,
		serverPackets.EquipKindBack,
	}

	stubs := make([]equipmentStub, 0, len(order))
	for idx, slot := range order {
		stubs = append(stubs, equipmentStub{
			Slot:        slot,
			ItemIndex:   uint8(idx),
			ContainerID: 0,
		})
	}

	return stubs
}
