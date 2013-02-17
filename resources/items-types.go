package resources

import (
	"encoding/json"
	"fmt"
)

type ItemType uint16

const (
	AvailableNormally    ItemType = iota // The item is available in survival by crafting or collecting resources
	AvailableInCreative                  // The item is not available in survival, but can be obtained from the creative inventory
	AvailableByTrading                   // The item can only be obtained in survival by trading with villagers
	AvailableBySilkTouch                 // The item can only be obtained in survival with a silk touch pickaxe
	Unavailable                          // The item is not available in either survival or creative without the use of an inventory editor or server commands
)

const (
	IsBlock      ItemType = 1 << (15 - iota) // The item is a block
	IsTileEntity                             // The item is a tile entity
)

type decomposedItemType struct {
	IsBlock      bool   `json:"is_block"`
	IsTileEntity bool   `json:"is_tile_entity"`
	Availability string `json:"availability"`
}

func (t ItemType) MarshalJSON() (data []byte, err error) {
	availabilityStr := ""

	switch t & 0x7 {
	case AvailableNormally:
		availabilityStr = "normal"
	case AvailableInCreative:
		availabilityStr = "creative"
	case AvailableByTrading:
		availabilityStr = "trading"
	case AvailableBySilkTouch:
		availabilityStr = "silktouch"
	case Unavailable:
		availabilityStr = "unavailable"
	}

	dt := &decomposedItemType{
		IsBlock:      t&IsBlock != 0,
		IsTileEntity: t&IsTileEntity != 0,
		Availability: availabilityStr,
	}

	return json.Marshal(dt)
}

type Item struct {
	ID       uint16   `json:"id"`       // The ID number of the item.
	Data     uint16   `json:"data"`     // The data value of the item.
	DataMask uint16   `json:"datamask"` // The mask of bits of the data value that are used to define the item.
	Name     string   `json:"name"`     // The textual name of the item.
	Type     ItemType `json:"type"`     // Item type
	MaxStack uint8    `json:"maxstack"` // The size of a stack of these items.
}

func ItemByIDAndData(id uint16, data uint16) (item Item, ok bool) {
	for _, item := range Items {
		if item.ID == id && item.Data == data&item.DataMask {
			return item, true
		}
	}

	return Item{}, false
}

func (item Item) ImageURL() (url string) {
	return fmt.Sprintf("http://kierdavis.com/mcresources/images/%d-%d.png", item.ID, item.Data)
}
