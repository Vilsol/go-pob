package moddb

import (
	"math"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestEvalPerStatTag_ZeroDivIsOne: PoB ModStore.lua:343 reads `local mult = m_floor(base / (tag.div or 1) + 0.0001)` -- `div` is optional and defaults to 1.
func TestEvalPerStatTag_ZeroDivIsOne(t *testing.T) {
	db := NewModDB()
	db.Actor = stubActor{outputs: map[string]float64{"Str": 250}}

	m := mod.NewFloat("Accuracy", mod.TypeBase, 1).Tag(mod.PerStat(0, "Str"))
	db.AddMod(m)

	got := db.Sum(mod.TypeBase, nil, "Accuracy")

	testza.AssertFalse(t, math.IsInf(got, 0), "PerStat with no div must not divide by zero")
	testza.AssertFalse(t, math.IsNaN(got), "PerStat with no div must not produce NaN")
	testza.AssertEqual(t, float64(250), got)
}

// TestEvalPerStatTag_ExplicitDivUnchanged: An explicit divisor still divides, and the floor(+0.0001) rounding is unchanged.
func TestEvalPerStatTag_ExplicitDivUnchanged(t *testing.T) {
	db := NewModDB()
	db.Actor = stubActor{outputs: map[string]float64{"Life": 3500}}

	m := mod.NewFloat("Damage", mod.TypeIncrease, 10).Tag(mod.PerStat(1000, "Life"))
	db.AddMod(m)

	testza.AssertEqual(t, float64(30), db.Sum(mod.TypeIncrease, nil, "Damage"))
}

// TestEvalPerStatTag_ZeroDivZeroStat: A zero divisor combined with a zero stat must yield 0, not NaN (0/0).
func TestEvalPerStatTag_ZeroDivZeroStat(t *testing.T) {
	db := NewModDB()
	db.Actor = stubActor{outputs: map[string]float64{"ChainRemaining": 0}}

	m := mod.NewFloat("Damage", mod.TypeIncrease, 25).Tag(mod.PerStat(0, "ChainRemaining"))
	db.AddMod(m)

	got := db.Sum(mod.TypeIncrease, nil, "Damage")
	testza.AssertFalse(t, math.IsNaN(got), "0 stat / 0 div must not produce NaN")
	testza.AssertEqual(t, float64(0), got)
}

// TestEvalMultiplierTag_ZeroDivIsOne: The Multiplier tag shares the same `(tag.div or 1)` semantics (ModStore.lua:277).
func TestEvalMultiplierTag_ZeroDivIsOne(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Multiplier:Rage", mod.TypeBase, 40))

	m := mod.NewFloat("Damage", mod.TypeIncrease, 1).Tag(&mod.MultiplierTag{
		TagType:      mod.TypeMultiplier,
		VariableList: []string{"Rage"},
	})
	db.AddMod(m)

	got := db.Sum(mod.TypeIncrease, nil, "Damage")
	testza.AssertFalse(t, math.IsInf(got, 0), "Multiplier with no div must not divide by zero")
	testza.AssertEqual(t, float64(40), got)
}
