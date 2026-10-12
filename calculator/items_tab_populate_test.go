package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

func addUnequipped(build *pob.PathOfBuilding, text string) int {
	id := 1
	for _, it := range build.Items.Items {
		id = max(id, it.ID+1)
	}
	build.Items.Items = append(build.Items.Items, pob.Item{ID: id, Text: text})
	return id
}

func activeSlotID(build *pob.PathOfBuilding, slotName string) int {
	set, _ := setItems(build)
	return setSlotItemID(set, slotName)
}

// TestPopulateSlots: ItemSlotControl.lua:94-96: Populate empties a slot whose id names no item.
func TestPopulateSlots(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	gloves := activeSlotID(build, "Gloves")
	if gloves == 0 || activeSlotID(build, "Weapon 2") == 0 {
		t.Fatal("REFUSING: 0280 has no gloves or no Weapon 2 to keep or drop")
	}
	staff := addUnequipped(build, donorItem(t, "4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "Weapon 1 Swap"))
	build.SetSlotItem("Weapon 1", staff)
	build.SetSlotItem("Helmet", 9999)

	PopulateSlots(build)
	if got := activeSlotID(build, "Weapon 2"); got != 0 {
		t.Errorf("Weapon 2 = %d beside a staff, want 0", got)
	}
	if got := activeSlotID(build, "Helmet"); got != 0 {
		t.Errorf("Helmet = %d naming no item, want 0", got)
	}
	if got := activeSlotID(build, "Weapon 1"); got != staff {
		t.Errorf("Weapon 1 = %d, want the staff %d", got, staff)
	}
	if got := activeSlotID(build, "Gloves"); got != gloves {
		t.Errorf("Gloves = %d, want %d kept", got, gloves)
	}
}

// TestEquipItemID: ItemSlotControl.lua:82-93: a slot's list is every item of the build that fits it.
func TestEquipItemID(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	ring1, ring2, helmet := activeSlotID(build, "Ring 1"), activeSlotID(build, "Ring 2"), activeSlotID(build, "Helmet")
	if ring1 == 0 || ring2 == 0 || helmet == 0 {
		t.Fatal("REFUSING: 0280 lacks a ring or a helmet")
	}
	ring := addUnequipped(build, donorItem(t, "0008-necromancer-vaal-summon-skeletons.xml", "Ring 1"))

	got := SlotItemIDs(build, "Ring 1")
	for _, want := range []int{ring1, ring2, ring} {
		if !slices.Contains(got, want) {
			t.Errorf("Ring 1's list %v lacks ring %d", got, want)
		}
	}
	if slices.Contains(got, helmet) {
		t.Errorf("Ring 1's list %v offers the helmet %d", got, helmet)
	}
	if !slices.IsSorted(got) {
		t.Errorf("Ring 1's list %v is not in id order", got)
	}

	if EquipItemID(build, "Helmet", ring) || activeSlotID(build, "Helmet") != helmet {
		t.Errorf("a ring was equipped in Helmet (Helmet = %d)", activeSlotID(build, "Helmet"))
	}
	if EquipItemID(build, "Ring 1", ring1) {
		t.Error("choosing the equipped ring reported a change")
	}
	if !EquipItemID(build, "Ring 1", ring) || activeSlotID(build, "Ring 1") != ring {
		t.Errorf("Ring 1 = %d, want the chosen ring %d", activeSlotID(build, "Ring 1"), ring)
	}
	if !EquipItemID(build, "Ring 1", 0) || activeSlotID(build, "Ring 1") != 0 {
		t.Errorf("choosing None left Ring 1 = %d", activeSlotID(build, "Ring 1"))
	}

	staff := addUnequipped(build, donorItem(t, "4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "Weapon 1 Swap"))
	if !EquipItemID(build, "Weapon 1", staff) || activeSlotID(build, "Weapon 2") != 0 {
		t.Errorf("after choosing a staff for Weapon 1, Weapon 2 = %d, want 0", activeSlotID(build, "Weapon 2"))
	}
}

// TestEquipItemIDNoneDropsTheBowsQuiver: ItemSlotControl.lua:12-19: "None" in Weapon 1's dropdown runs PopulateSlots too.
func TestEquipItemIDNoneDropsTheBowsQuiver(t *testing.T) {
	initItemsTabData(t)
	load := func() (*pob.PathOfBuilding, int) {
		build := loadCorpusBuild(t, "0241-warden-galvanic-arrow.xml")
		set, byID := setItems(build)
		bow, quiver := setSlotItem(set, byID, "Weapon 1"), setSlotItem(set, byID, "Weapon 2")
		if bow == nil || bow.Base == nil || bow.Base.Type != "Bow" || quiver == nil || quiver.Type != "Quiver" {
			t.Fatal("REFUSING: 0241 has no bow in Weapon 1 with a quiver in Weapon 2")
		}
		return build, activeSlotID(build, "Weapon 2")
	}

	control, quiver := load()
	control.SetSlotItem("Weapon 1", 0)
	if got := activeSlotID(control, "Weapon 2"); got != quiver {
		t.Fatalf("control: emptying Weapon 1 without PopulateSlots left Weapon 2 = %d, want the quiver %d", got, quiver)
	}

	build, _ := load()
	if !EquipItemID(build, "Weapon 1", 0) {
		t.Fatal("choosing None in Weapon 1 reported no change")
	}
	if got := activeSlotID(build, "Weapon 1"); got != 0 {
		t.Errorf("Weapon 1 = %d after None, want 0", got)
	}
	if got := activeSlotID(build, "Weapon 2"); got != 0 {
		t.Errorf("Weapon 2 = %d after the bow went, want 0 (the quiver %d dropped)", got, quiver)
	}
}
