package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestMergeLevelModWritesAComputedZero pins CalcActiveSkill.lua:22 — `if not value`, a nil test.
func TestMergeLevelModWritesAComputedZero(t *testing.T) {
	for _, c := range []struct {
		name  string
		value float64
	}{
		{"a computed zero overwrites the placeholder", 0},
		{"the nonzero control still injects", 7},
	} {
		t.Run(c.name, func(t *testing.T) {
			list := moddb.NewModList()
			mergeLevelMod(list, mod.NewFloat("Damage", mod.TypeIncrease, 999).Source("Skill"), c.value)

			mods := list.Mods()
			if len(mods) != 1 {
				t.Fatalf("list holds %d mods, want 1", len(mods))
			}
			if got := mods[0].Value().Float(); got != c.value {
				t.Errorf("mod value = %v, want %v — mergeLevelMod must WRITE the computed value, "+
					"and 999 is the placeholder it was leaving in place", got, c.value)
			}
		})
	}

	t.Run("the source mod is not mutated", func(t *testing.T) {
		src := mod.NewFloat("Damage", mod.TypeIncrease, 999).Source("Skill")
		mergeLevelMod(moddb.NewModList(), src, 0)
		if got := src.Value().Float(); got != 999 {
			t.Errorf("source mod value = %v, want 999 — the statMap entry is shared", got)
		}
	})
}
