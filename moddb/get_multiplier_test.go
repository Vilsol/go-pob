package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

func TestGetMultiplierBaseMod(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Multiplier:Foo", mod.TypeBase, 5))

	testza.AssertEqual(t, float64(5), db.GetMultiplier("Foo", nil, false))
	testza.AssertEqual(t, float64(0), db.GetMultiplier("Foo", nil, true))
}

func TestGetMultiplierCombined(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Multiplier:Foo", mod.TypeBase, 5))
	db.Multipliers["Foo"] = 2

	testza.AssertEqual(t, float64(7), db.GetMultiplier("Foo", nil, false))
	testza.AssertEqual(t, float64(2), db.GetMultiplier("Foo", nil, true))
}
