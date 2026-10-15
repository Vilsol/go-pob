package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestRing3SlotModList: Item.lua:2640-2651 builds slotModList[1] and [2] for weapons and rings, and slotModList[3] for RINGS ONLY.
//
//pob:block:src/Classes/Item.lua:2692-2703
func TestRing3SlotModList(t *testing.T) {
	const ring = "Rarity: RARE\nTest Band\nIron Ring\nItem Level: 84\n"
	const amulet = "Rarity: RARE\nTest Pendant\nAmber Amulet\nItem Level: 84\n"

	t.Run("a ring builds all three slot lists", func(t *testing.T) {
		item := ParseItem(ring)
		if item.Base == nil || item.Type != "Ring" {
			t.Fatalf("REFUSING: Iron Ring parsed as type %q, so this case tests nothing", item.Type)
		}
		if item.slotModList3 == nil {
			t.Errorf("slotModList3 is nil; Item.lua:2646-2648 builds it for every Ring")
		}
	})

	t.Run("control: a non-slotted item builds none", func(t *testing.T) {
		item := ParseItem(amulet)
		if item.slotModList2 != nil || item.slotModList3 != nil {
			t.Errorf("an amulet built per-slot lists (2=%v 3=%v); PoB builds them only for "+
				"weapons and rings", item.slotModList2 != nil, item.slotModList3 != nil)
		}
	})

	t.Run("control: a weapon builds two and not three", func(t *testing.T) {
		item := ParseItem("Rarity: RARE\nTest Blade\nImperial Claw\nItem Level: 84\n")
		if item.slotModList2 == nil {
			t.Fatalf("REFUSING: a weapon has no slotModList2, so the guard below proves nothing")
		}
		if item.slotModList3 != nil {
			t.Errorf("a weapon built slotModList3; Item.lua:2646 guards it on `type == \"Ring\"`")
		}
	})

	t.Run("slot 3 resolves {SlotName} to Ring 3", func(t *testing.T) {
		for slot, want := range map[int]string{1: "Ring 1", 2: "Ring 2", 3: "Ring 3"} {
			item := ParseItem(ring)
			built := item.buildModListForSlotNum([]mod.Mod{
				mod.NewFloat("Damage", mod.TypeIncrease, 10).Tag(mod.SocketedIn("{SlotName}")),
			}, slot)
			if len(built) != 1 {
				t.Fatalf("REFUSING: slot %d built %d mods, want 1 — the fixture mod was dropped",
					slot, len(built))
			}
			got := ""
			for _, tag := range built[0].Tags() {
				if st, ok := tag.(*mod.SocketedInTag); ok {
					got = st.SlotName
				}
			}
			if got != want {
				t.Errorf("slot %d resolved {SlotName} to %q, want %q", slot, got, want)
			}
		}
	})
}
