package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestAilmentStacksMaxPrefersOverride pins the `Override(...) or Sum("BASE", ...)` chain of CalcPerform.lua:3458.
func TestAilmentStacksMaxPrefersOverride(t *testing.T) {
	db := moddb.NewModDB()
	db.AddMod(mod.NewFloat("ShockStacksMax", mod.TypeBase, 1).Source("Base"))
	if got := ailmentStacksMax(db, "ShockStacksMax"); got != 1 {
		t.Fatalf("REFUSING: base-only ShockStacksMax = %v, want 1 — the fixture is not reaching the sum", got)
	}
	db.AddMod(mod.NewFloat("ShockStacksMax", mod.TypeOverride, 50).Source("Test"))
	if got := ailmentStacksMax(db, "ShockStacksMax"); got != 50 {
		t.Errorf("ShockStacksMax with an OVERRIDE = %v, want 50 — the `or` chain reads the sum instead", got)
	}
}
