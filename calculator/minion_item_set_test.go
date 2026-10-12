package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/pob"
)

// TestMinionItemSetResolvesByID separates the two arms of CalcPerform.lua:1158-1176, which NO CORPUS BUILD DOES.
func TestMinionItemSetResolvesByID(t *testing.T) {
	items := pob.Items{
		ActiveItemSet: 7,
		ItemSets: []pob.ItemSet{
			{ID: "3"},
			{ID: "7"},
			{ID: "1"},
		},
	}

	for _, tc := range []struct {
		name string
		id   int
		want string
	}{
		{"id 3 is first in document order", 3, "3"},
		{"id 7 is also the active set, and that is a coincidence", 7, "7"},
		{"id 1 is LAST in document order, not first", 1, "1"},
		{"an id naming no set falls back to itemSetOrderList[1]", 99, "3"},
		{"id 0 is the absent attribute and takes the same fallback", 0, "3"},
	} {
		got := minionItemSet(items, tc.id)
		testza.AssertNotNil(t, got, "%s: minionItemSet returned nil", tc.name)
		testza.AssertEqual(t, tc.want, got.ID,
			"%s: minionItemSet(skillMinionItemSet=%d) picked set id %q, want %q "+
				"(CalcActiveSkill.lua:559-562 keys itemsTab.itemSets by id and falls back to "+
				"itemSetOrderList[1], the first set in DOCUMENT order)", tc.name, tc.id, got.ID, tc.want)
	}

	testza.AssertNil(t, minionItemSet(pob.Items{}, 1),
		"a build with no item sets has no set to bind a minion to; PoB indexes an empty table and "+
			"gets nil, and the CalcPerform.lua:1158 guard is what keeps that from being read")
}

// TestMinionItemSetWeaponSwapRename pins the rename at CalcActiveSkill.lua:583/:593 and CalcPerform.lua:1163-1165.
func TestMinionItemSetWeaponSwapRename(t *testing.T) {
	second := true
	first := false

	for _, tc := range []struct {
		name string
		set  *pob.ItemSet
		want string
	}{
		{"a set on the primary weapon set keeps the base name", &pob.ItemSet{UseSecondWeaponSet: &first}, "Weapon 1"},
		{"an absent attribute is the primary set", &pob.ItemSet{}, "Weapon 1"},
		{"a set on the swap weapon set renames", &pob.ItemSet{UseSecondWeaponSet: &second}, "Weapon 1 Swap"},
	} {
		testza.AssertEqual(t, tc.want, weaponSetSlotName(tc.set, "Weapon 1"), tc.name)
	}
}
