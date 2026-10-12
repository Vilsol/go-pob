package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestChargesMaxReadsTheOverrideArm pins CalcPerform.lua:953/:960/:967's `modDB:Override(nil, "<X>ChargesMax") or m_max(fallback, 0)`.
func TestChargesMaxReadsTheOverrideArm(t *testing.T) {
	db := moddb.NewModDB()
	db.AddMod(mod.NewFloat("PowerChargesMax", mod.TypeBase, 3).Source("Base"))
	if got := chargesMax(db, "PowerChargesMax", db.Sum(mod.TypeBase, nil, "PowerChargesMax")); got != 3 {
		t.Fatalf("REFUSING: with no override PowerChargesMax = %v, want 3 — the fixture reaches no arm", got)
	}
	if got := chargesMax(db, "PowerChargesMax", -2); got != 0 {
		t.Errorf("a negative fallback = %v, want 0 — the m_max(..., 0) clamp is absent", got)
	}

	db.AddMod(mod.NewFloat("PowerChargesMax", mod.TypeOverride, 0).Source("Minion:Chilling Skitterbot"))
	if got := chargesMax(db, "PowerChargesMax", db.Sum(mod.TypeBase, nil, "PowerChargesMax")); got != 0 {
		t.Errorf("with OVERRIDE 0 PowerChargesMax = %v, want 0 — the override arm is not read", got)
	}
}

// TestReplaceChargesMaxFromFlagWritesTheOverride pins CalcPerform.lua:955-958.
func TestReplaceChargesMaxFromFlagWritesTheOverride(t *testing.T) {
	db := moddb.NewModDB()
	db.AddMod(mod.NewFloat("FrenzyChargesMax", mod.TypeBase, 3).Source("Base"))
	replaceChargesMaxFromFlag(db, "MaximumFrenzyChargesIsMaximumPowerCharges", "FrenzyChargesMax", 7)
	if got := db.Override(nil, "FrenzyChargesMax"); got != nil {
		t.Fatalf("REFUSING: an override was written with the flag absent (%v)", got.Float())
	}

	db.AddMod(mod.NewFlag("MaximumFrenzyChargesIsMaximumPowerCharges", true).Source("Voll's Devotion"))
	replaceChargesMaxFromFlag(db, "MaximumFrenzyChargesIsMaximumPowerCharges", "FrenzyChargesMax", 7)
	got := db.Override(nil, "FrenzyChargesMax")
	if got == nil {
		t.Fatal("no FrenzyChargesMax OVERRIDE was written — the flag is resolved only at the read site")
	}
	if got.Float() != 7 {
		t.Errorf("FrenzyChargesMax OVERRIDE = %v, want 7", got.Float())
	}
	if src := db.Mods["FrenzyChargesMax"][len(db.Mods["FrenzyChargesMax"])-1].GetSource(); src != "Voll's Devotion" {
		t.Errorf("override source = %q, want the flag mod's own source", src)
	}
}
