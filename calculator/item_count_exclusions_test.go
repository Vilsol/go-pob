package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestUpdateItemCountsExcludesTinctureAndGraft pins the exclusion list at CalcSetup.lua:1173.
//
//	if item.type ~= "Jewel" and item.type ~= "Flask" and item.type ~= "Tincture" and item.type ~= "Graft" then
func TestUpdateItemCountsExcludesTinctureAndGraft(t *testing.T) {
	newEnv := func() *Environment {
		return &Environment{ItemModDB: moddb.NewModDB()}
	}

	for _, excluded := range []string{"Jewel", "Flask", "Tincture", "Graft"} {
		env := newEnv()
		updateItemCounts(env, &Item{Type: excluded, Rarity: RarityUnique}, "Graft 1")
		if n := len(env.ItemModDB.Multipliers); n != 0 {
			t.Errorf("type %q incremented %d multiplier(s) — CalcSetup.lua:1173 excludes it entirely",
				excluded, n)
		}
		if n := len(env.ItemModDB.Conditions); n != 0 {
			t.Errorf("type %q stamped %d condition(s) — CalcSetup.lua:1173 excludes it entirely",
				excluded, n)
		}
	}

	env := newEnv()
	updateItemCounts(env, &Item{Type: "Body Armour", Rarity: RarityUnique}, "Body Armour")
	if env.ItemModDB.Multipliers["UniqueItem"] != 1 {
		t.Errorf("CONTROL: an ordinary unique must still count — UniqueItem = %v, want 1",
			env.ItemModDB.Multipliers["UniqueItem"])
	}
}
