package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestSummonedMinionConfigOverrides: ConfigOptions.lua:1128-1133 emits OVERRIDE for both summoned-minion counts.
//
//pob:block:src/Modules/ConfigOptions.lua:1139-1144
func TestSummonedMinionConfigOverrides(t *testing.T) {
	apply := func(name string, val float64) *moddb.ModDB {
		db := moddb.NewModDB()
		db.Conditions["Combat"] = true
		db.AddMod(mod.NewFloat("Multiplier:"+multiplierFor(name), mod.TypeBase, 8))
		handler, ok := configurations[name]
		if !ok {
			t.Fatalf("REFUSING: no handler named %q, so this case cannot test anything", name)
		}
		list := moddb.NewModList()
		handler(val, list, moddb.NewModList())
		db.AddList(list)
		return db
	}

	for _, name := range []string{"multiplierSummonedMinion", "multiplierNonVaalSummonedMinion"} {
		t.Run(name+" overrides rather than adding", func(t *testing.T) {
			db := apply(name, 3)
			if got := db.GetMultiplier(multiplierFor(name), nil, false); got != 3 {
				t.Errorf("Multiplier:%s = %v, want 3 — a BASE would give 11 (8 published + 3)",
					multiplierFor(name), got)
			}
		})
	}

	t.Run("control: the published count survives with no config", func(t *testing.T) {
		db := moddb.NewModDB()
		db.Conditions["Combat"] = true
		db.AddMod(mod.NewFloat("Multiplier:SummonedMinion", mod.TypeBase, 8))
		if got := db.GetMultiplier("SummonedMinion", nil, false); got != 8 {
			t.Fatalf("REFUSING: control = %v, want 8 — the fixture never had a count to override", got)
		}
	})

	t.Run("both are in the dispatch order", func(t *testing.T) {
		inOrder := map[string]bool{}
		for _, n := range configOrder {
			inOrder[n] = true
		}
		for _, name := range []string{"multiplierSummonedMinion", "multiplierNonVaalSummonedMinion"} {
			if _, ok := configurations[name]; !ok {
				t.Errorf("%s has no handler", name)
			}
			if !inOrder[name] {
				t.Errorf("%s is not in configOrder, so env.go never dispatches it", name)
			}
		}
	})
}

func multiplierFor(configName string) string {
	if configName == "multiplierNonVaalSummonedMinion" {
		return "NonVaalSummonedMinion"
	}
	return "SummonedMinion"
}
