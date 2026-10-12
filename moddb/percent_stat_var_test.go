package moddb

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestPercentStatVarReadsTheMultiplier pins ModStore.lua:504's `tag.percent or GetMultiplier(self, tag.percentVar, cfg)`.
func TestPercentStatVarReadsTheMultiplier(t *testing.T) {
	store := NewModDB()
	store.AddMod(mod.NewFloat("Multiplier:ArcaneCloakConsumedMana", mod.TypeBase, 25).Source("Test"))

	cfg := &ListCfg{SkillStats: map[string]float64{"Mana": 1000}}

	tag := mod.PercentStatVar("Mana", "ArcaneCloakConsumedMana")
	if tag.PercentVar != "ArcaneCloakConsumedMana" {
		t.Fatalf("PercentVar not carried: %q", tag.PercentVar)
	}

	val := mod.NewModValueFloat(1)
	got := store.evalPercentStatTag(val, cfg, tag)
	if got.ValueFloat != 250 {
		t.Errorf("percentVar ignored: got %v, want 250 (Mana 1000 * ArcaneCloakConsumedMana 25%%). "+
			"ModStore.lua:504 falls back to GetMultiplier when tag.percent is unset; an ignored "+
			"percentVar yields 1000 (stat used as-is), which looks like a legitimate unset-percent.",
			got.ValueFloat)
	}
}
