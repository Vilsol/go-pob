package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
)

// TestHitResInversionStatMapRowIsEmitted covers the SKILL-GRANTED half of HitsInvertEleResChance.
func TestHitResInversionStatMapRowIsEmitted(t *testing.T) {
	const stat = "treat_enemy_resistances_as_negated_on_elemental_damage_hit_%_chance"
	entry := raw.SkillStatMap[stat]
	if entry == nil {
		t.Fatalf("SkillStatMap has no %q row at all", stat)
	}
	if entry.Div == nil || *entry.Div != 100 {
		t.Fatalf("%q must divide by 100 — CHANCE is read as a fraction, got %v", stat, entry.Div)
	}
	if len(entry.Mods) != 1 {
		t.Fatalf("%q carries %d mods, want 1 (HitsInvertEleResChance) — an empty list makes the "+
			"row consume the stat and produce nothing", stat, len(entry.Mods))
	}
	testza.AssertEqual(t, "HitsInvertEleResChance", entry.Mods[0].Name())
	testza.AssertEqual(t, mod.TypeChance, entry.Mods[0].Type())
}
