package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestPercentStatReadsTheNamedActorsStat pins ModStore.lua:492-495 — the actor redirect that decides WHOSE stat the percentage is taken of.
func TestPercentStatReadsTheNamedActorsStat(t *testing.T) {
	minionDB, parent := minionAndParent()
	parent.outputs["Life"] = 4000

	cfg := &ListCfg{SkillStats: map[string]float64{"Life": 100}}

	minionDB.AddMod(mod.NewFloat("FireMin", mod.TypeBase, 1).
		Tag(mod.PercentStat("Life", 1).Actor("parent")))
	testza.AssertEqual(t, 40.0, minionDB.Sum(mod.TypeBase, cfg, "FireMin"))

	selfDB, _ := minionAndParent()
	selfDB.AddMod(mod.NewFloat("FireMin", mod.TypeBase, 1).Tag(mod.PercentStat("Life", 1)))
	testza.AssertEqual(t, 1.0, selfDB.Sum(mod.TypeBase, cfg, "FireMin"))

	unresolved, _ := minionAndParent()
	unresolved.AddMod(mod.NewFloat("FireMin", mod.TypeBase, 1).
		Tag(mod.PercentStat("Life", 1).Actor("nobody")))
	testza.AssertEqual(t, 1.0, unresolved.Sum(mod.TypeBase, cfg, "FireMin"))
}

// TestPercentStatFloorsItsProduct pins ModStore.lua:506-508.
func TestPercentStatFloorsItsProduct(t *testing.T) {
	eval := func(t *testing.T, life, pct, value float64, floor bool) float64 {
		t.Helper()
		store := NewModDB()
		cfg := &ListCfg{SkillStats: map[string]float64{"Life": life}}
		tag := mod.PercentStat("Life", pct)
		if floor {
			tag = tag.Floor()
		}
		return store.evalPercentStatTag(mod.NewModValueFloat(value), cfg, tag).ValueFloat
	}

	t.Run("the product is floored before the value is scaled", func(t *testing.T) {
		testza.AssertEqual(t, 216.0, eval(t, 4321, 5, 1, true))
	})
	t.Run("CONTROL: without floor the ceil still rounds up", func(t *testing.T) {
		testza.AssertEqual(t, 217.0, eval(t, 4321, 5, 1, false))
	})
	t.Run("CONTROL: an exact product is untouched", func(t *testing.T) {
		testza.AssertEqual(t, 200.0, eval(t, 4000, 5, 1, true))
	})
	t.Run("the floor lands on the product, not the result", func(t *testing.T) {
		testza.AssertEqual(t, 86.0, eval(t, 4321, 1, 2, true))
		testza.AssertEqual(t, 87.0, eval(t, 4321, 1, 2, false))
	})
}
