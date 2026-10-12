package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestImmunitySpellingAndSignWidenings pins the second tranche of the pattern-transcription sweep.
func TestImmunitySpellingAndSignWidenings(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse")
	}

	for _, tc := range []struct{ line, want string }{
		{"Immune to Elemental Ailments while on Consecrated Ground", "AvoidIgnite"},
		{"Immunity to Elemental Ailments while on Consecrated Ground", "AvoidIgnite"},
		{"Immune to Elemental Ailments while you have Arcane Surge", "AvoidIgnite"},
		{"Immunity to Elemental Ailments while you have Arcane Surge", "AvoidIgnite"},
		{"Immune to Elemental Ailments while affected by Glorious Madness", "ElementalAilmentImmune"},
		{"Immunity to Elemental Ailments while affected by Glorious Madness", "ElementalAilmentImmune"},
		{"Immune to Elemental Ailments during any Flask Effect", "AvoidIgnite"},
		{"Immunity to Elemental Ailments during any Flask Effect", "AvoidIgnite"},
		{"Consecrated Ground you create grants Immune to Elemental Ailments to you and Allies", "AvoidIgnite"},
		{"Consecrated Ground you create grants Immunity to Elemental Ailments to you and Allies", "AvoidIgnite"},
		{"Immune to Elemental Ailments while on Consecrated Ground if you have at least 100 Devotion", "AvoidIgnite"},
		{"Immunity to Elemental Ailments while on Consecrated Ground if you have at least 100 Devotion", "AvoidIgnite"},
		{"Your Skills have no Mana Cost during Flask Effect", "ManaCost"},
		{"Skills have no Mana Cost during Flask Effect", "ManaCost"},
		{"+1 to Level of all Elemental Skill Gems if the Stars are Aligned", "GemProperty"},
		{"-1 to Level of all Elemental Skill Gems if the Stars are Aligned", "GemProperty"},
		{"+1 to Level of all Elemental Support Gems if the Stars are Aligned", "GemProperty"},
		{"+1 to Level of Socketed Active Skill Gems per 25 Player Levels", "GemProperty"},
		{"-1 to Level of Socketed Active Skill Gems per 25 Player Levels", "GemProperty"},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || len(e.ModList) == 0 {
			t.Errorf("DROPPED WHOLE (no mods): %s", tc.line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q, so the line is discarded: %s", extra, tc.line)
			continue
		}
		var names []string
		for _, m := range e.ModList {
			names = append(names, m.Name())
		}
		if !contains(names, tc.want) {
			t.Errorf("%q -> %v, want %q", tc.line, names, tc.want)
		}
	}

	e := ParseMod("-1 to Level of all Elemental Skill Gems if the Stars are Aligned", false)
	if e != nil && len(e.ModList) == 1 {
		if g := gemPropertyOf(e.ModList[0]); g != nil && g.Value != -1 {
			t.Errorf("sign dropped: -1 parsed to value %v", g.Value)
		}
	}
}

func gemPropertyOf(m mod.Mod) *mod.GemProperty {
	if m.Value().Type() != mod.ModValueMultiTypeList {
		return nil
	}
	g, ok := m.Value().List().(mod.GemProperty)
	if !ok {
		return nil
	}
	return &g
}
