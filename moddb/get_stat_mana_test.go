package moddb

import (
	"math"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

type manaActor map[string]float64

func (a manaActor) GetOutput(key string) (float64, bool) {
	v, ok := a[key]
	return v, ok
}

// TestGetStatSubstitutesManaForNaNManaUnreserved: ModStore.lua:234-236.
func TestGetStatSubstitutesManaForNaNManaUnreserved(t *testing.T) {
	db := NewModDB()
	db.Actor = manaActor{"Mana": 1234, "ManaUnreserved": math.NaN()}
	db.AddMod(mod.NewFloat("Multiplier:Consumed", mod.TypeBase, 1).Tag(mod.PerStat(1, "ManaUnreserved")))

	testza.AssertEqual(t, float64(1234), db.getStat("ManaUnreserved", nil))

	got := db.Sum(mod.TypeBase, nil, "Multiplier:Consumed")
	testza.AssertFalse(t, math.IsNaN(got))
	testza.AssertEqual(t, float64(1234), got)
}

// TestGetStatLeavesOtherNaNStatsAlone: The substitution is keyed on the STAT NAME as well as on NaN, exactly as the Lua is.
func TestGetStatLeavesOtherNaNStatsAlone(t *testing.T) {
	db := NewModDB()
	db.Actor = manaActor{"Mana": 1234, "EnergyShield": math.NaN()}
	testza.AssertTrue(t, math.IsNaN(db.getStat("EnergyShield", nil)))

	db2 := NewModDB()
	db2.Actor = manaActor{"Mana": 1234, "ManaUnreserved": 500}
	testza.AssertEqual(t, float64(500), db2.getStat("ManaUnreserved", nil))
}

// TestGetStatReverseEngineersNegativeManaUnreserved: ModStore.lua:311-314, the arm below the NaN one.
func TestGetStatReverseEngineersNegativeManaUnreserved(t *testing.T) {
	db := NewModDB()
	db.Actor = manaActor{
		"Mana":                  6628,
		"ManaUnreserved":        -10113,
		"ManaUnreservedPercent": -152.57996378998,
		"ManaEfficiency":        27,
	}

	got := db.getStat("ManaUnreserved", nil)
	testza.AssertTrue(t, math.Abs(got-21275.88) < 1e-6,
		"ModStore.lua:311-314 reverse-engineers the pre-efficiency reservation; got", got)

	db.AddMod(mod.NewFloat("Multiplier:Consumed", mod.TypeBase, 1).Tag(mod.PerStat(50, "ManaUnreserved")))
	testza.AssertEqual(t, float64(425), db.Sum(mod.TypeBase, nil, "Multiplier:Consumed"))
}

// TestGetStatLeavesNegativeLifeUnreservedAlone: The arm is keyed on the stat NAME.
func TestGetStatLeavesNegativeLifeUnreservedAlone(t *testing.T) {
	db := NewModDB()
	db.Actor = manaActor{"Life": 100, "LifeUnreserved": -50, "LifeUnreservedPercent": -150, "LifeEfficiency": 0}
	testza.AssertEqual(t, float64(-50), db.getStat("LifeUnreserved", nil))
}
