package calculator

import (
	"strings"
	"testing"
)

// TestDealNoDamageWithSkillsYourself pins ModParser.lua:5251.
//
//pob:block:src/Modules/ModParser.lua:5251-5253
func TestDealNoDamageWithSkillsYourself(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("Deal no Chaos Damage", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: an already-ported DealNo* row no longer parses")
	}

	for _, tc := range []struct {
		name string
		line string
		want []string
	}{
		{"ancestralBond", "You can't deal Damage with Skills yourself", []string{
			"SkillType/neg=true/skillTypes=26,36,33",
			"Condition/neg=true/vars=usedByMirage",
		}},

		{"nb_spellSkills", "Spell Skills deal no Damage", []string{"SkillType/skillTypes=2"}},
		{"nb_notLowLife", "Deal no Damage when not on Low Life", []string{"Condition/neg=true/vars=LowLife"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) == 0 {
				t.Fatalf("no mods at all (the line is DROPPED WHOLE): %s", tc.line)
			}
			if extra := strings.TrimSpace(modExtra(e)); extra != "" {
				t.Fatalf("leftover %q — an unparsed remainder discards the WHOLE line: %s", extra, tc.line)
			}
			got := collectCanonTags(e.ModList)
			for _, w := range tc.want {
				if !contains(got, w) {
					t.Fatalf("tags %v, want %q: %s", got, w, tc.line)
				}
			}
		})
	}
}

// TestDealNoDamageYourselfEmitsTheFlag is the name half, kept separate from the tag half above so that a failure says WHICH of the two properties broke.
func TestDealNoDamageYourselfEmitsTheFlag(t *testing.T) {
	requireGameData(t, "3.29")

	e := ParseMod("You can't deal Damage with Skills yourself", false)
	if e == nil || len(e.ModList) == 0 {
		t.Fatalf("no mods at all")
	}
	got := make([]string, 0, len(e.ModList))
	for _, m := range e.ModList {
		got = append(got, m.Name())
	}
	if !contains(got, "DealNoDamage") {
		t.Fatalf("produced %v, want DealNoDamage — the name calc_offence.go:2494 reads", got)
	}
}
