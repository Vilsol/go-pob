package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestUsedTablesSkipBaseSourcedMods pins Calcs.lua:502-511.
//
//	local function addTo(out, var, mod)
//		if mod.source == "Base" then
//		end
func TestUsedTablesSkipBaseSourcedMods(t *testing.T) {
	out := map[string][]mod.Mod{}

	addUsedVar(out, "Onslaught", mod.NewFloat("Damage", mod.TypeIncrease, 10).Source("Tree:12345"))
	if len(out["Onslaught"]) != 1 {
		t.Fatalf("REFUSING: a Tree-sourced mod was not recorded (%d entries); the helper is broken "+
			"generally and this test cannot attribute a failure to the Base filter", len(out["Onslaught"]))
	}

	addUsedVar(out, "Fortified", mod.NewFloat("Armour", mod.TypeIncrease, 5).Source("Base"))
	if n := len(out["Fortified"]); n != 0 {
		t.Errorf("a Base-sourced mod was recorded as used (%d entries); Calcs.lua:504-506 drops it", n)
	}

	out2 := map[string][]mod.Mod{}
	addUsedTag(out2, []string{"A", "B"}, mod.NewFloat("Life", mod.TypeBase, 1).Source("Base"))
	if n := len(out2["A"]) + len(out2["B"]); n != 0 {
		t.Errorf("addUsedTag recorded %d Base-sourced entries; it must inherit addUsedVar's filter", n)
	}
}
