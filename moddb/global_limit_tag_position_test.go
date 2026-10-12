package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestSumGlobalLimitOnNonFirstTag: the clamp must fire when the pair sits on a later tag.
func TestSumGlobalLimitOnNonFirstTag(t *testing.T) {
	db := NewModDB()
	for i := 0; i < 3; i++ {
		db.AddMod(mod.NewFloat("SkillAuraEffectOnSelf", mod.TypeIncrease, 10).
			Tag(mod.Condition("Effective")).
			Tag(mod.Multiplier("Herald").GlobalLimit(25).GlobalLimitKey("PurposefulHarbinger")))
	}
	db.Conditions["Effective"] = true
	db.Multipliers["Herald"] = 1

	testza.AssertEqual(t, 25.0, db.Sum(mod.TypeIncrease, nil, "SkillAuraEffectOnSelf"))
}

// TestTabulateGlobalLimitOnNonFirstTag: the same, through the other call site that threads a globalLimits frame (ModDB.lua:269).
func TestTabulateGlobalLimitOnNonFirstTag(t *testing.T) {
	db := NewModDB()
	for i := 0; i < 3; i++ {
		db.AddMod(mod.NewFloat("SkillAuraEffectOnSelf", mod.TypeIncrease, 10).
			Tag(mod.Condition("Effective")).
			Tag(mod.Multiplier("Herald").GlobalLimit(25).GlobalLimitKey("PurposefulHarbinger")))
	}
	db.Conditions["Effective"] = true
	db.Multipliers["Herald"] = 1

	total := float64(0)
	for _, r := range db.Tabulate(mod.TypeIncrease, nil, "SkillAuraEffectOnSelf") {
		total += r.Value
	}
	testza.AssertEqual(t, 25.0, total)
}
