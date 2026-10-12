package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestImpaleStacksCapScalesByAdditionalDuration pins CalcPerform.lua:3321's second term.
func TestImpaleStacksCapScalesByAdditionalDuration(t *testing.T) {
	db := moddb.NewModDB()
	db.AddMod(mod.NewFloat("ImpaleStacksMax", mod.TypeBase, 5).Source("Base"))
	if got := impaleStacksCap(db); got != 5 {
		t.Fatalf("REFUSING: bare ImpaleStacksMax = %v, want 5 — the fixture reaches nothing", got)
	}
	db.AddMod(mod.NewFloat("ImpaleAdditionalDurationChance", mod.TypeBase, 10).Source("Test"))
	if got := impaleStacksCap(db); got != 5.5 {
		t.Errorf("cap with a 10%% additional-duration chance = %v, want 5.5 — the term is dropped", got)
	}
}
