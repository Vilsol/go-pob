package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestShockImmuneStillAddsZeroShockMod pins CalcPerform.lua:891-892: a shock-immune player still gets the Shock
// DamageTaken mod, with effect 0.
func TestShockImmuneStillAddsZeroShockMod(t *testing.T) {
	env := corpusOnlyEnv(t, "0024-trickster-frostbite")
	found := false
	for _, m := range env.ModDB.Mods["DamageTaken"] {
		if string(m.GetSource()) == "Shock" && m.Type() == mod.TypeIncrease {
			found = true
			if v := ourModValue(m); v != 0 {
				t.Errorf("Shock DamageTaken = %v, PoB 0", v)
			}
		}
	}
	// PoB's modDB dump carries the row.
	if !found {
		t.Errorf("no Shock DamageTaken mod")
	}
}
