package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestAilmentTypeListMatchesPoB pins data.Ailment("").Values() to PoB v2.18.1 src/Modules/Data.lua:341.
func TestAilmentTypeListMatchesPoB(t *testing.T) {
	want := []data.Ailment{
		data.AilmentBleed,
		data.AilmentPoison,
		data.AilmentIgnite,
		data.AilmentChill,
		data.AilmentFreeze,
		data.AilmentShock,
		data.AilmentScorch,
		data.AilmentBrittle,
		data.AilmentSap,
	}

	got := data.Ailment("").Values()
	if len(got) != len(want) {
		t.Fatalf("ailmentTypeList: got %d entries %v, PoB Data.lua:341 has %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ailmentTypeList[%d]: got %q, PoB Data.lua:341 has %q", i, got[i], want[i])
		}
	}
}

// TestElementalAilmentTypeListMatchesPoB pins data.ElementalAilment("").Values() to PoB v2.18.1 src/Modules/Data.lua:342.
func TestElementalAilmentTypeListMatchesPoB(t *testing.T) {
	want := []data.Ailment{
		data.AilmentIgnite,
		data.AilmentChill,
		data.AilmentFreeze,
		data.AilmentShock,
		data.AilmentScorch,
		data.AilmentBrittle,
		data.AilmentSap,
	}

	got := data.ElementalAilment("").Values()
	if len(got) != len(want) {
		t.Fatalf("elementalAilmentTypeList: got %d entries %v, PoB Data.lua:342 has %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("elementalAilmentTypeList[%d]: got %q, PoB Data.lua:342 has %q", i, got[i], want[i])
		}
	}
}

// TestElementalAilmentImmunityModsStayElementalOnly guards the "immune to elemental ailments while X" handlers.
func TestElementalAilmentImmunityModsStayElementalOnly(t *testing.T) {
	lines := []string{
		"Immune to Elemental Ailments while on Consecrated Ground",
		"Immune to Elemental Ailments while you have Arcane Surge",
		"Consecrated Ground you create grants Immunity to Elemental Ailments to you and Allies",
		"Immune to Elemental Ailments during any Flask Effect",
		"Immune to Elemental Ailments while Phasing",
		"Immune to Elemental Ailments while on Consecrated Ground if you have at least 100 Devotion",
	}

	forbidden := map[string]bool{
		"Avoid" + string(data.AilmentBleed):  true,
		"Avoid" + string(data.AilmentPoison): true,
	}

	for _, line := range lines {
		mods, extra := parseMod(line, 1)
		if mods == nil {
			t.Errorf("%q: did not parse (extra %q)", line, extra)
			continue
		}
		if len(mods) != 1 {
			t.Errorf("%q: emitted %d mods, pin-3.29's ModParser emits one ElementalAilmentImmune flag", line, len(mods))
			continue
		}
		m := mods[0]
		if m.Name() == "ExtraAura" {
			if aura, ok := m.Value().List().(mod.ExtraAura); ok {
				m = aura.Mod
			}
		}
		if m.Name() != "ElementalAilmentImmune" {
			t.Errorf("%q: emitted %s, want ElementalAilmentImmune", line, m.Name())
		}
		if forbidden[m.Name()] {
			t.Errorf("%q: emitted %s, which is not an elemental ailment", line, m.Name())
		}
	}
}

// TestBleedAndPoisonHaveNoNonDamagingAilmentData pins the absence at PoB src/Modules/Data.lua:344-351.
func TestBleedAndPoisonHaveNoNonDamagingAilmentData(t *testing.T) {
	for _, ailment := range []data.Ailment{data.AilmentBleed, data.AilmentPoison} {
		if _, ok := data.NonDamagingAilments[ailment]; ok {
			t.Errorf("%s is in NonDamagingAilments, but PoB Data.lua:344-351 has no entry for it", ailment)
		}
		if ailment.IsNonDamaging() {
			t.Errorf("%s.IsNonDamaging() is true, but PoB's data.nonDamagingAilment[%q] is nil", ailment, ailment)
		}
	}
}
