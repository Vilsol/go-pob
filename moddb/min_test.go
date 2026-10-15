package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestMin pins ModStore.lua:231-240: the smallest live MIN mod, nil when none.
func TestMin(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("PoisonStackLimit", mod.TypeMIN, 8))
	db.AddMod(mod.NewFloat("PoisonStackLimit", mod.TypeMIN, 1).Tag(mod.Condition("Gate")))
	db.AddMod(mod.NewFloat("PoisonStackLimit", mod.TypeMIN, 3))
	db.AddMod(mod.NewFloat("PoisonStackLimit", mod.TypeBase, 0))

	v, ok := db.Min(nil, "PoisonStackLimit")
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, 3.0, v)

	db.Conditions["Gate"] = true
	v, _ = db.Min(nil, "PoisonStackLimit")
	testza.AssertEqual(t, 1.0, v)

	list := NewModList()
	list.AddMod(mod.NewFloat("PoisonStackLimit", mod.TypeMIN, 5))
	list.AddMod(mod.NewFloat("PoisonStackLimit", mod.TypeMIN, 2))
	v, ok = list.Min(nil, "PoisonStackLimit")
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, 2.0, v)

	_, ok = db.Min(nil, "NoSuchMod")
	testza.AssertFalse(t, ok)
}
