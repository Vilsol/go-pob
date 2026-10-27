package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// PoB-PoE2 ModStore.lua 543-545: an inverted multiplier scales by 1 / mult, and leaves a zero alone.
func TestMultiplierInvert(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Multiplier:Effect", mod.TypeBase, 4))
	db.AddMod(mod.NewFloat("Damage", mod.TypeBase, 100).Tag(mod.Multiplier("Effect").Invert()))
	db.AddMod(mod.NewFloat("Other", mod.TypeBase, 100).Tag(mod.Multiplier("Missing").Invert()))
	testza.AssertEqual(t, float64(25), db.Sum(mod.TypeBase, nil, "Damage"))
	testza.AssertEqual(t, float64(0), db.Sum(mod.TypeBase, nil, "Other"))
}

// PoB-PoE2 ModStore.lua 577-584, 601: thresholdVar is read from the thresholdActor's modDB.
func TestMultiplierThresholdActor(t *testing.T) {
	minionDB, parent := minionAndParent()
	minionDB.AddMod(mod.NewFloat("Multiplier:Seconds", mod.TypeBase, 3))
	parent.modDB.AddMod(mod.NewFloat("Multiplier:Needed", mod.TypeBase, 5))
	minionDB.AddMod(mod.NewFlag("CanCull", true).Tag(mod.MultiplierThreshold("Seconds").ThresholdVar("Needed").ThresholdActor("parent")))
	testza.AssertFalse(t, minionDB.Flag(nil, "CanCull"))
	minionDB.AddMod(mod.NewFloat("Multiplier:Seconds", mod.TypeBase, 2))
	testza.AssertTrue(t, minionDB.Flag(nil, "CanCull"))
}

// PoB-PoE2 ModStore.lua 622-625: divVar replaces div with a multiplier.
func TestPerStatDivVar(t *testing.T) {
	db := NewModDB()
	db.Actor = stubActor{outputs: map[string]float64{"Armour": 1000}}
	db.AddMod(mod.NewFloat("Multiplier:DamagePerArmour", mod.TypeBase, 250))
	db.AddMod(mod.NewFloat("Damage", mod.TypeMore, 1).Tag(mod.PerStat(1, "Armour").DivVar("DamagePerArmour")))
	testza.AssertEqual(t, float64(4), db.Sum(mod.TypeMore, nil, "Damage"))
}

// PoB-PoE2 ModStore.lua 663-672, 699: a statList sums its stats before the percent.
func TestPercentStatStatList(t *testing.T) {
	db := NewModDB()
	db.Actor = stubActor{outputs: map[string]float64{"Armour": 300, "Evasion": 700}}
	db.AddMod(mod.NewFloat("AegisValue", mod.TypeBase, 1).Tag(mod.PercentStatList(1, "Armour", "Evasion")))
	testza.AssertEqual(t, float64(10), db.Sum(mod.TypeBase, nil, "AegisValue"))
}

// PoB-PoE2 ModStore.lua 706-721: a statList sums its stats before the threshold.
func TestStatThresholdStatList(t *testing.T) {
	db := NewModDB()
	db.Actor = stubActor{outputs: map[string]float64{"LifeCost": 3, "ManaCost": 0}}
	db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).Tag(mod.StatThreshold("", 1).Stats("LifeCost", "ManaCost")))
	testza.AssertEqual(t, float64(40), db.Sum(mod.TypeIncrease, nil, "Damage"))
	db2 := NewModDB()
	db2.Actor = stubActor{outputs: map[string]float64{"LifeCost": 0.5, "ManaCost": 0.4}}
	db2.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).Tag(mod.StatThreshold("", 1).Stats("LifeCost", "ManaCost")))
	testza.AssertEqual(t, float64(0), db2.Sum(mod.TypeIncrease, nil, "Damage"))
}
