package moddb

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestGetMultiplierHonoursAnOverride pins the FIRST term of ModStore.lua:266, which this port omitted entirely.
//
//	return (not noMod and self:Override(cfg, multiplierName[var]))
//	    or (self.multipliers[var] or 0) + (parent...) + (not noMod and self:Sum("BASE", cfg, ...) or 0)
func TestGetMultiplierHonoursAnOverride(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Multiplier:ThingDoubled", mod.TypeOverride, 1))

	if got := db.GetMultiplier("ThingDoubled", nil, false); got != 1 {
		t.Errorf("GetMultiplier = %v, want 1 — an OVERRIDE on Multiplier:<var> is ignored, so "+
			"every DOUBLED-form mod scales to nothing", got)
	}

	if got := db.GetMultiplier("ThingDoubled", nil, true); got != 0 {
		t.Errorf("GetMultiplier(noMod=true) = %v, want 0 — the Override arm is gated on `not noMod`", got)
	}
}

// TestGetMultiplierOverrideOfZeroWins pins the arm a `!= 0` guard would silently lose.
func TestGetMultiplierOverrideOfZeroWins(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Multiplier:Suppressed", mod.TypeBase, 7))
	db.AddMod(mod.NewFloat("Multiplier:Suppressed", mod.TypeOverride, 0))

	if got := db.GetMultiplier("Suppressed", nil, false); got != 0 {
		t.Errorf("GetMultiplier = %v, want 0 — an OVERRIDE of zero must WIN over the BASE sum, "+
			"not fall through to it", got)
	}
}

// TestGetMultiplierWithoutAnOverrideStillSums is the control: it must pass in BOTH states, so the short-circuit cannot be added by breaking the ordinary path.
func TestGetMultiplierWithoutAnOverrideStillSums(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Multiplier:Plain", mod.TypeBase, 3))
	db.AddMod(mod.NewFloat("Multiplier:Plain", mod.TypeBase, 4))

	if got := db.GetMultiplier("Plain", nil, false); got != 7 {
		t.Errorf("CONTROL: GetMultiplier = %v, want 7 — the BASE sum path has been disturbed", got)
	}
}
