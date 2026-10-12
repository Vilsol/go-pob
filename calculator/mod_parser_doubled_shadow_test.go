package calculator

import (
	"strings"
	"testing"
)

// TestDoubledFormIsNotShadowedByLiteralRows pins a class this tree invented.
func TestDoubledFormIsNotShadowedByLiteralRows(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("Maximum Recovery per Energy Shield Leech is Doubled", false); e == nil ||
		len(e.ModList) != 2 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the already-paired doubled row no longer emits two mods")
	}

	for _, tc := range []struct {
		name     string
		line     string
		wantMods []string
		wantTag  string
	}{
		{"esLeechRate", "Maximum total Energy Shield Recovery per second from Leech is doubled",
			[]string{"MaxEnergyShieldLeechRate", "Multiplier:MaxEnergyShieldLeechRateDoubled"},
			"Multiplier/globalLimit=100/globalLimitKey=MaxEnergyShieldLeechRateDoubledLimit/vars=MaxEnergyShieldLeechRateDoubled"},
		{"lifeLeechRate", "Total Recovery per second from Life Leech is Doubled",
			[]string{"LifeLeechRate", "Multiplier:LifeLeechRateDoubled"},
			"Multiplier/globalLimit=100/globalLimitKey=LifeLeechRateDoubledLimit/vars=LifeLeechRateDoubled"},
		{"chaosResist", "Chaos Resistance is doubled",
			[]string{"ChaosResist", "Multiplier:ChaosResistDoubled"},
			"Multiplier/globalLimit=100/globalLimitKey=ChaosResistDoubledLimit/vars=ChaosResistDoubled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) == 0 {
				t.Fatalf("no mods at all (the line is DROPPED WHOLE): %s", tc.line)
			}
			if extra := strings.TrimSpace(modExtra(e)); extra != "" {
				t.Fatalf("leftover %q — an unparsed remainder discards the WHOLE line: %s", extra, tc.line)
			}
			got := make([]string, 0, len(e.ModList))
			for _, m := range e.ModList {
				got = append(got, m.Name())
			}
			for _, w := range tc.wantMods {
				if !contains(got, w) {
					t.Fatalf("mods %v, want %q — a bare MORE with no OVERRIDE leaves the global cap unreachable: %s", got, w, tc.line)
				}
			}
			tags := collectCanonTags(e.ModList)
			if !contains(tags, tc.wantTag) {
				t.Fatalf("tags %v, want %q: %s", tags, tc.wantTag, tc.line)
			}
		})
	}

	t.Run("untagged_lifeLeechedPerSecond", func(t *testing.T) {
		e := ParseMod("Life Leeched per second is Doubled", false)
		if e == nil || len(e.ModList) == 0 {
			t.Fatalf("no mods at all: the pin's own specialModList row stopped matching")
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Fatalf("leftover %q: the specialModList row stopped matching", extra)
		}
		if len(e.ModList) != 1 || e.ModList[0].Name() != "LifeLeechRate" {
			got := make([]string, 0, len(e.ModList))
			for _, m := range e.ModList {
				got = append(got, m.Name())
			}
			t.Fatalf("mods %v, want exactly [LifeLeechRate] — ModParser.lua:2246 is UNTAGGED and "+
				"must not be routed through the DOUBLED form", got)
		}
		if tags := collectCanonTags(e.ModList); len(tags) != 0 {
			t.Fatalf("tags %v, want none — ModParser.lua:2246 carries no tag", tags)
		}
	})
}
