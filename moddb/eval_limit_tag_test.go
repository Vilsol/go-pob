package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestEvalLimitTag_Caps: Limit caps the RUNNING value, not the stored one.
//
//	elseif tag.type == "Limit" then
//		value = m_min(value, tag.limit or self:GetMultiplier(tag.limitVar, cfg))
func TestEvalLimitTag_Caps(t *testing.T) {
	t.Run("literal limit caps a larger value", func(t *testing.T) {
		db := NewModDB()
		m := mod.NewFloat("ColdDamageTaken", mod.TypeIncrease, 45).Tag(mod.Limit(30))
		db.AddMod(m)

		testza.AssertEqual(t, float64(30), db.Sum(mod.TypeIncrease, nil, "ColdDamageTaken"))
		testza.AssertEqual(t, float64(45), m.Value().Float())
	})

	t.Run("literal limit leaves a smaller value alone", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("ColdDamageTaken", mod.TypeIncrease, 12).Tag(mod.Limit(30)))

		testza.AssertEqual(t, float64(12), db.Sum(mod.TypeIncrease, nil, "ColdDamageTaken"))
	})

	t.Run("limitVar reads the ceiling from a multiplier", func(t *testing.T) {
		db := NewModDB()
		db.Multipliers["PyroclastSelfFireMaxLimit"] = 20
		db.AddMod(mod.NewFloat("SelfFireMax", mod.TypeBase, 75).
			Tag(mod.LimitVar("PyroclastSelfFireMaxLimit")))

		testza.AssertEqual(t, float64(20), db.Sum(mod.TypeBase, nil, "SelfFireMax"))
	})

	t.Run("limitVar with no multiplier caps to zero", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("SelfFireMin", mod.TypeBase, 50).Tag(mod.LimitVar("Nonexistent")))

		testza.AssertEqual(t, float64(0), db.Sum(mod.TypeBase, nil, "SelfFireMin"))
	})

	t.Run("caps in tag order, not at the end", func(t *testing.T) {
		db := NewModDB()
		db.Multipliers["Frenzy"] = 3
		db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 45).
			Tag(mod.Limit(30), mod.Multiplier("Frenzy")))

		testza.AssertEqual(t, float64(90), db.Sum(mod.TypeIncrease, nil, "Damage"))
	})
}
