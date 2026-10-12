package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

func armourMasteryDB() *ModDB {
	db := NewModDB()
	db.AddMod(mod.NewFloat("ArmourDefense", mod.TypeMAX, 100).Tag(mod.Condition("ArmourMax")))
	db.AddMod(mod.NewFloat("ArmourDefense", mod.TypeMAX, 20).Tag(mod.Condition("ArmourAvg")))
	db.AddMod(mod.NewFloat("ArmourDefense", mod.TypeMAX, 5).
		Tag(mod.Condition("ArmourMax").Neg(true)).
		Tag(mod.Condition("ArmourAvg").Neg(true)))
	return db
}

// TestMaxSkipsGatedMods pins that a MAX mod whose tags gate it out contributes nothing instead of nil-dereferencing.
func TestMaxSkipsGatedMods(t *testing.T) {
	db := armourMasteryDB()

	testza.AssertEqual(t, 5.0, db.Max(nil, "ArmourDefense"))

	db.Conditions["ArmourAvg"] = true
	testza.AssertEqual(t, 20.0, db.Max(nil, "ArmourDefense"))

	db.Conditions["ArmourMax"] = true
	testza.AssertEqual(t, 100.0, db.Max(nil, "ArmourDefense"))

	testza.AssertEqual(t, 0.0, db.Max(nil, "NoSuchMod"))
}
