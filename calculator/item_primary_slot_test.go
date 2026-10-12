package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestGetPrimarySlotCoversEveryLuaArm: Item.lua:2127-2143 (ItemClass:GetPrimarySlot) has six arms.
//
//	elseif self.type == "Tincture" then return "Flask 1"
//	elseif self.type == "Graft" then return "Graft 1"
func TestGetPrimarySlotCoversEveryLuaArm(t *testing.T) {
	for _, tc := range []struct {
		itemType string
		want     string
	}{
		{"Quiver", "Weapon 2"},
		{"Shield", "Weapon 2"},
		{"Ring", "Ring 1"},
		{"Flask", "Flask 1"},
		{"Tincture", "Flask 1"},
		{"Graft", "Graft 1"},
		{"Helmet", "Helmet"},
		{"Amulet", "Amulet"},
	} {
		t.Run(tc.itemType, func(t *testing.T) {
			i := &Item{Type: tc.itemType}
			testza.AssertEqual(t, tc.want, i.getPrimarySlot())
		})
	}
}

// TestGetPrimarySlotWeaponArmComesFirst: The weapon arm wins over self.type.
func TestGetPrimarySlotWeaponArmComesFirst(t *testing.T) {
	i := &Item{Type: "Graft"}
	testza.AssertEqual(t, "Graft 1", i.getPrimarySlot())
}
