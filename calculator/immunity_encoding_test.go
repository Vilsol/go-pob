package calculator

import (
	"testing"
)

// TestImmunityFamilyUsesThePinEncoding pins the encoding decision for audit row 5c8d446b.
func TestImmunityFamilyUsesThePinEncoding(t *testing.T) {
	for _, tc := range []struct {
		line string
		want []string
		pin  string
	}{
		{"Cannot be Poisoned", []string{"PoisonImmune"}, "ModParser.lua:4872"},
		{"Cannot be Poisoned while Bleeding", []string{"PoisonImmune"}, "ModParser.lua:4873"},
		{"Cannot be Chilled during Onslaught", []string{"ChillImmune"}, "ModParser.lua:4860"},
		{"Cannot be Frozen or Chilled if you've used a Fire Skill Recently",
			[]string{"FreezeImmune", "ChillImmune"}, "ModParser.lua:4861"},
		{"Immunity to Curses while Channelling", []string{"CurseImmune"}, "ModParser.lua:4887"},
		{"Immunity to Poison", []string{"PoisonImmune"}, "ModParser.lua:4871"},
		{"Cannot be Shocked", []string{"ShockImmune"}, "ModParser.lua:4827"},
		{"Cannot be Frozen", []string{"FreezeImmune"}, "ModParser.lua:4829"},
		{"Cannot be Chilled", []string{"ChillImmune"}, "ModParser.lua:4832"},
		{"Cannot be Ignited", []string{"IgniteImmune"}, "ModParser.lua:4834"},
		{"Cannot be Inflicted with Bleeding", []string{"BleedImmune"}, "ModParser.lua:4862"},
		{"Cannot be Stunned", []string{"StunImmune"}, "ModParser.lua:4818"},
		{"Cannot be Knocked Back", []string{"KnockbackImmune"}, "ModParser.lua:4826"},
		{"Cannot be Blinded", []string{"Condition:CannotBeBlinded", "BlindImmune"}, "ModParser.lua:4824"},
		{"You cannot be Impaled", []string{"ImpaleImmune"}, "ModParser.lua:2492"},
		{"You cannot be Maimed", []string{"MaimImmune"}, "ModParser.lua:2491"},
		{"You cannot be Hindered", []string{"HinderImmune"}, "ModParser.lua:2490"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != len(tc.want) {
				t.Fatalf("ParseMod(%q) produced %v mods, want %d (%s)",
					tc.line, e, len(tc.want), tc.pin)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			for i, want := range tc.want {
				if got := e.ModList[i].Name(); got != want {
					t.Errorf("mod %d name = %q, want %q (%s)", i, got, want, tc.pin)
				}
			}
		})
	}
}
