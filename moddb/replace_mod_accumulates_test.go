package moddb

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestReplaceModReplacesOnceThenAccumulates pins ModDB.lua:38-60's `replaced` semantics.
//
//pob:src/Classes/ModDB.lua:42-69
func TestReplaceModReplacesOnceThenAccumulates(t *testing.T) {
	newOverride := func(v float64) mod.Mod {
		return mod.NewFloat("ChargesMax", mod.TypeOverride, v).Source(mod.Source("Item:1:Badge"))
	}

	db := NewModDB()
	db.ReplaceMod(newOverride(3))
	if got := len(db.Mods["ChargesMax"]); got != 1 {
		t.Fatalf("after the first ReplaceMod: %d mods, want 1", got)
	}

	db.ReplaceMod(newOverride(5))
	if got := len(db.Mods["ChargesMax"]); got != 1 {
		t.Fatalf("after the second ReplaceMod: %d mods, want 1 (it must REPLACE)", got)
	}
	if !db.Mods["ChargesMax"][0].Replaced() {
		t.Fatal("the stored mod is not marked replaced — ModDB.lua:51 is not firing, and every " +
			"assertion below would then pass for the wrong reason")
	}

	db.ReplaceMod(newOverride(9))
	if got := len(db.Mods["ChargesMax"]); got != 2 {
		t.Fatalf("after the third ReplaceMod: %d mods, want 2 — a replaced shape ACCUMULATES", got)
	}

	got := db.Override(nil, "ChargesMax")
	if got == nil {
		t.Fatal("Override returned nil for a name carrying two OVERRIDE mods")
	}
	if f := got.Float(); f != 5 {
		t.Errorf("Override = %v, want 5 — the first surviving mod. 9 means the third call "+
			"overwrote, i.e. the replaced conjunct is absent", f)
	}

	db.ReplaceMod(newOverride(11))
	if got := len(db.Mods["ChargesMax"]); got != 2 {
		t.Fatalf("after the fourth ReplaceMod: %d mods, want 2 — it must replace the unmarked one", got)
	}
	if f := db.Override(nil, "ChargesMax").Float(); f != 5 {
		t.Errorf("Override = %v after the fourth call, want 5 — the first mod is untouched", f)
	}
}
