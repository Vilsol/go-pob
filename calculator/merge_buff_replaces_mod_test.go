package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestMergeBuffReplacesTheWholeMod: CalcPerform.lua:354-355 replaces the whole MOD — `dest[index] = mod` — when the incoming value is larger.
func TestMergeBuffReplacesTheWholeMod(t *testing.T) {
	src := moddb.NewModList()
	src.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).Source("Winner"))

	dest := map[string]*moddb.ModList{}
	dest["buff"] = moddb.NewModList()
	dest["buff"].AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 10).Source("Loser"))

	mergeBuff(src, dest, "buff")

	mods := dest["buff"].Mods()
	testza.AssertLen(t, mods, 1)
	testza.AssertEqual(t, float64(40), mods[0].Value().Float())
	testza.AssertEqual(t, mod.Source("Winner"), mods[0].GetSource(),
		"the surviving mod must be the INCOMING one, not the old one with a new number")
}

// TestMergeBuffKeepsTheWinnerWhenIncomingIsSmaller: The converse: a SMALLER incoming value leaves the destination mod untouched, source included.
func TestMergeBuffKeepsTheWinnerWhenIncomingIsSmaller(t *testing.T) {
	src := moddb.NewModList()
	src.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 10).Source("Loser"))

	dest := map[string]*moddb.ModList{}
	dest["buff"] = moddb.NewModList()
	dest["buff"].AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).Source("Winner"))

	mergeBuff(src, dest, "buff")

	mods := dest["buff"].Mods()
	testza.AssertLen(t, mods, 1)
	testza.AssertEqual(t, float64(40), mods[0].Value().Float())
	testza.AssertEqual(t, mod.Source("Winner"), mods[0].GetSource())
}
