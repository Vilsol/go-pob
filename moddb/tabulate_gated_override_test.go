package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestTabulateDropsGatedOutOverride: ModDB.lua:283 is `if value and (value ~= 0 or mod.type == "OVERRIDE") then`.
func TestTabulateDropsGatedOutOverride(t *testing.T) {
	tabulate := func(db *ModDB) []ModResult {
		return db.Tabulate(mod.TypeOverride, nil, "MaxLifeLeechRate")
	}

	t.Run("a gated-out OVERRIDE is dropped", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("MaxLifeLeechRate", mod.TypeOverride, 5).
			Tag(mod.Condition("NeverSet")))

		testza.AssertEqual(t, 0, len(tabulate(db)))
	})

	t.Run("a satisfied OVERRIDE is kept", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("MaxLifeLeechRate", mod.TypeOverride, 5).
			Tag(mod.Condition("IsSet")))
		db.Conditions["IsSet"] = true

		res := tabulate(db)
		testza.AssertEqual(t, 1, len(res))
		testza.AssertEqual(t, 5.0, res[0].Value)
	})

	t.Run("an untagged zero-valued OVERRIDE is kept", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("MaxLifeLeechRate", mod.TypeOverride, 0))

		res := tabulate(db)
		testza.AssertEqual(t, 1, len(res))
		testza.AssertEqual(t, 0.0, res[0].Value)
	})

	t.Run("a gated-out INC is dropped", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("MaxLifeLeechRate", mod.TypeIncrease, 5).
			Tag(mod.Condition("NeverSet")))

		testza.AssertEqual(t, 0, len(db.Tabulate(mod.TypeIncrease, nil, "MaxLifeLeechRate")))
	})
}
