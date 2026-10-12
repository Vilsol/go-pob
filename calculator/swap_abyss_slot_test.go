package calculator

import (
	"strconv"
	"testing"
)

// TestSwapWeaponAbyssalSockets: 12116 fills all six of Weapon 1 Swap's abyssal sockets; ItemsTab.lua:157 names them "Weapon 1 Swap Abyssal Socket N".
func TestSwapWeaponAbyssalSockets(t *testing.T) {
	initItemsTabData(t)
	const file = "12116-ladder-Spice24_6530.Riku_AllFlame.xml"
	jewel := donorItem(t, file, "Weapon 1 Swap Abyssal Socket 6")

	build := loadCorpusBuild(t, file)
	on := true
	build.Items.ItemSets[activeItemSetIndex(build.Items)].UseSecondWeaponSet = &on
	build.SetSlotItem("Weapon 1 Swap Abyssal Socket 6", 0)
	id, slot := GetMiscCalculator(build).AddItem(jewel)
	set, _ := setItems(build)
	if slot != "Weapon 1 Swap Abyssal Socket 6" || setSlotItemID(set, slot) != id {
		t.Errorf("Save equipped the abyss jewel in %q, want the free Weapon 1 Swap Abyssal Socket 6", slot)
	}

	env := GetMiscCalculator(build).Calc(nil)
	for _, s := range orderedItemSlots {
		if s.parentSlot == "Weapon 1 Swap" {
			if name := "Weapon 1 Abyssal Socket " + strconv.Itoa(s.slotNum); env.Player.ItemList[name] != nil {
				t.Errorf("%s reached itemList as %q; PoB drops swap abyss jewels", s.name, name)
			}
		}
	}
}
