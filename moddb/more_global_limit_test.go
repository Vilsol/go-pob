package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestMoreAppliesGlobalLimit: two doubling-shaped MORE mods sharing one key double once.
func TestMoreAppliesGlobalLimit(t *testing.T) {
	db := NewModDB()
	for i := 0; i < 2; i++ {
		db.AddMod(mod.NewFloat("Defences", mod.TypeMore, 100).
			Tag(mod.Multiplier("OathOfTheMajiDoubled").GlobalLimit(100).GlobalLimitKey("OathOfTheMajiLimit")))
	}
	db.Multipliers["OathOfTheMajiDoubled"] = 1

	testza.AssertEqual(t, 2.0, db.More(nil, "Defences"))
}

// TestMoreGlobalLimitOnNonFirstTag.
func TestMoreGlobalLimitOnNonFirstTag(t *testing.T) {
	db := NewModDB()
	for i := 0; i < 2; i++ {
		db.AddMod(mod.NewFloat("Defences", mod.TypeMore, 100).
			Tag(mod.Condition("UsingBody Armour")).
			Tag(mod.Multiplier("OathOfTheMajiDoubled").GlobalLimit(100).GlobalLimitKey("OathOfTheMajiLimit")))
	}
	db.Conditions["UsingBody Armour"] = true
	db.Multipliers["OathOfTheMajiDoubled"] = 1

	testza.AssertEqual(t, 2.0, db.More(nil, "Defences"))
}

// TestMoreGlobalLimitFrameIsPerCall.
func TestMoreGlobalLimitFrameIsPerCall(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Defences", mod.TypeMore, 100).
		Tag(mod.Multiplier("OathOfTheMajiDoubled").GlobalLimit(100).GlobalLimitKey("OathOfTheMajiLimit")))
	db.Multipliers["OathOfTheMajiDoubled"] = 1

	testza.AssertEqual(t, 2.0, db.More(nil, "Defences"))
	testza.AssertEqual(t, 2.0, db.More(nil, "Defences"))
}

// TestModListMoreHasNoGlobalLimit is the CONSTRUCTED NEGATIVE.
func TestModListMoreHasNoGlobalLimit(t *testing.T) {
	list := NewModList()
	for i := 0; i < 2; i++ {
		list.AddMod(mod.NewFloat("Defences", mod.TypeMore, 100).
			Tag(mod.Multiplier("OathOfTheMajiDoubled").GlobalLimit(100).GlobalLimitKey("OathOfTheMajiLimit")))
	}
	list.Multipliers["OathOfTheMajiDoubled"] = 1

	testza.AssertEqual(t, 4.0, list.More(nil, "Defences"))
}
