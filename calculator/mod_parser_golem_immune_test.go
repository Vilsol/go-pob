package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestGolemElementalImmunityMatchesThePin pins ModParser.lua:2763-2766.
func TestGolemElementalImmunityMatchesThePin(t *testing.T) {
	requireGameData(t, "3.29")

	e := ParseMod("Summoned Golems are Immune to Elemental Damage", false)
	if e == nil || len(e.ModList) == 0 {
		t.Fatalf("REFUSING: the line does not parse at all; this test cannot check its shape")
	}

	var inner []string
	for _, m := range e.ModList {
		if m.Name() != "MinionModifier" {
			t.Errorf("got a %q mod, want only MinionModifier", m.Name())
			continue
		}
		mm, ok := m.Value().ValueList.(mod.MinionModifier)
		if !ok || mm.Mod == nil {
			t.Errorf("MinionModifier carries no inner mod")
			continue
		}
		inner = append(inner, mm.Mod.Name())
	}

	if len(inner) != 2 {
		t.Fatalf("emitted %d minion modifiers (%v), want exactly 2 — the pin emits Elemancer "+
			"and ElementalDamageTaken, NOT six resistance overrides", len(inner), inner)
	}
	if !contains(inner, "Elemancer") {
		t.Errorf("missing the Elemancer flag: %v", inner)
	}
	if !contains(inner, "ElementalDamageTaken") {
		t.Errorf("missing ElementalDamageTaken: %v", inner)
	}
	for _, n := range inner {
		if n == "ColdResist" || n == "FireResist" || n == "LightningResist" ||
			n == "ColdResistMax" || n == "FireResistMax" || n == "LightningResistMax" {
			t.Errorf("still emits the invented resistance override %q", n)
		}
	}
}
