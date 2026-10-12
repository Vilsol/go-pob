package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestEvalMod_NoCompounding: (a) A BASE mod scaled by a single Multiplier must not compound across repeated Sum calls.
func TestEvalMod_NoCompounding(t *testing.T) {
	db := NewModDB()
	db.Multipliers["M"] = 3

	m := mod.NewFloat("Stat", mod.TypeBase, 100).Tag(mod.Multiplier("M"))
	db.AddMod(m)

	first := db.Sum(mod.TypeBase, nil, "Stat")
	second := db.Sum(mod.TypeBase, nil, "Stat")

	testza.AssertEqual(t, float64(300), first)
	testza.AssertEqual(t, second, first)
	testza.AssertEqual(t, float64(100), m.Value().Float())
}

// TestEvalMod_GateShortCircuits: (b) A gating tag that fails must short-circuit: the mod is excluded and a later scaling tag must NOT resurrect it.
func TestEvalMod_GateShortCircuits(t *testing.T) {
	t.Run("gate before scale", func(t *testing.T) {
		db := NewModDB()
		db.Multipliers["M"] = 3
		m := mod.NewFloat("Stat", mod.TypeBase, 100).
			Tag(mod.Condition("Unmet"), mod.Multiplier("M"))
		db.AddMod(m)

		testza.AssertEqual(t, float64(0), db.Sum(mod.TypeBase, nil, "Stat"))
	})

	t.Run("scale before gate", func(t *testing.T) {
		db := NewModDB()
		db.Multipliers["M"] = 3
		m := mod.NewFloat("Stat", mod.TypeBase, 100).
			Tag(mod.Multiplier("M"), mod.Condition("Unmet"))
		db.AddMod(m)

		testza.AssertEqual(t, float64(0), db.Sum(mod.TypeBase, nil, "Stat"))
		testza.AssertEqual(t, float64(100), m.Value().Float())
	})
}

// TestEvalMod_ThreadedScaling: (c) Two scaling tags must compose in sequence (each operates on the previous tag's output) without mutating the stored mod.
func TestEvalMod_ThreadedScaling(t *testing.T) {
	db := NewModDB()
	db.Multipliers["A"] = 3
	db.Multipliers["B"] = 2

	m := mod.NewFloat("Stat", mod.TypeBase, 100).
		Tag(mod.Multiplier("A"), mod.Multiplier("B"))
	db.AddMod(m)

	testza.AssertEqual(t, float64(600), db.Sum(mod.TypeBase, nil, "Stat"))
	testza.AssertEqual(t, float64(100), m.Value().Float())
}

// TestEvalMod_PercentStatZeroMultiplierScalesToZero: THIS TEST PINNED AN INVENTED FALLBACK, AND ITS OWN COMMENT IS HOW.
//
//	local percent = tag.percent or GetMultiplier(self, tag.percentVar, cfg)   -- :504
//	local mult = base * (percent and percent / 100 or 1)                      -- :505
func TestEvalMod_PercentStatZeroMultiplierScalesToZero(t *testing.T) {
	cfg := &ListCfg{SkillStats: map[string]float64{"Strength": 200}}

	t.Run("percentVar resolving to 0 zeroes the mod", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("Stat", mod.TypeBase, 1).Tag(mod.PercentStatVar("Strength", "Consumed")))
		testza.AssertEqual(t, float64(0), db.Sum(mod.TypeBase, cfg, "Stat"))
	})

	t.Run("percentVar resolving nonzero scales by it", func(t *testing.T) {
		db := NewModDB()
		db.Multipliers["Consumed"] = 25
		db.AddMod(mod.NewFloat("Stat", mod.TypeBase, 1).Tag(mod.PercentStatVar("Strength", "Consumed")))
		testza.AssertEqual(t, float64(50), db.Sum(mod.TypeBase, cfg, "Stat"))
	})

	t.Run("a literal percent scales by it", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("Stat", mod.TypeBase, 1).Tag(mod.PercentStat("Strength", 10)))
		testza.AssertEqual(t, float64(20), db.Sum(mod.TypeBase, cfg, "Stat"))
	})
}

// TestEvalMod_PassThrough: (d) A gating tag that passes leaves the value unchanged.
func TestEvalMod_PassThrough(t *testing.T) {
	db := NewModDB()
	db.Conditions["Met"] = true

	m := mod.NewFloat("Stat", mod.TypeBase, 100).Tag(mod.Condition("Met"))
	db.AddMod(m)

	testza.AssertEqual(t, float64(100), db.Sum(mod.TypeBase, nil, "Stat"))
}
