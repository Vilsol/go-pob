package calculator

import (
	"strings"
	"testing"
)

// TestShockMinimumRename pins ModParser.lua:2768, whose output this tree spelled `ShockBase` where upstream spells `ShockMinimum`.
//
//pob:src/Modules/ModParser.lua:2778
func TestShockMinimumRename(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		name string
		line string
		want string
	}{
		{"mastery25", "Shocks from your Hits always increase Damage taken by at least 25%", "ShockMinimum"},
		{"mastery10", "Shocks from your Hits always increase Damage taken by at least 10%", "ShockMinimum"},

		{"nb_chillNotRenamed", "Chills from your Hits always reduce Action Speed by at least 10%", "ChillBase"},
		{"nb_shockBaseKept", "35% chance to Shock Attackers for 4 seconds on Block", "ShockBase"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) == 0 {
				t.Fatalf("no mods at all: %s", tc.line)
			}
			if extra := strings.TrimSpace(modExtra(e)); extra != "" {
				t.Fatalf("leftover %q: %s", extra, tc.line)
			}
			got := make([]string, 0, len(e.ModList))
			for _, m := range e.ModList {
				got = append(got, m.Name())
			}
			if !contains(got, tc.want) {
				t.Fatalf("mods %v, want %q: %s", got, tc.want, tc.line)
			}
		})
	}
}
