package moddb

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestTabulateReturnsListModsFromModList pins the ONE line that made `ModList.Tabulate("LIST", ...)` return nothing, ever.
func TestTabulateReturnsListModsFromModList(t *testing.T) {
	const name = "ExtraJewelFunc"

	list := NewModList()
	list.AddMod(mod.NewList(name, "payload"))

	if got := len(list.Tabulate(mod.TypeList, nil, name)); got != 1 {
		t.Errorf("ModList.Tabulate returned %d LIST mods, want 1 — a LIST mod's value is not a "+
			"float, so a `value != 0` test drops it and Tabulate(\"LIST\") can never return anything", got)
	}

	db := NewModDB()
	db.AddMod(mod.NewList(name, "payload"))
	if got := len(db.Tabulate(mod.TypeList, nil, name)); got != 1 {
		t.Errorf("CONTROL: ModDB.Tabulate returned %d LIST mods, want 1 — the twin was already "+
			"correct, so this failing means the cause is not the one this test names", got)
	}
}

// TestTabulateStillDropsAZeroValuedNumericMod pins the arm the fix must NOT widen.
func TestTabulateStillDropsAZeroValuedNumericMod(t *testing.T) {
	list := NewModList()
	list.AddMod(mod.NewFloat("Armour", mod.TypeBase, 0))

	if got := len(list.Tabulate(mod.TypeBase, nil, "Armour")); got != 0 {
		t.Errorf("ModList.Tabulate returned %d zero-valued numeric mods, want 0 — the LIST fix has "+
			"widened into the numeric arm and every caller reading ModResult.Value is now affected", got)
	}
}
