package calculator

import (
	"sort"
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestBareBuffNameBlockParses covers ModParser.lua:3972-3980, the "-- Buffs/debuffs" block of bare buff names.
func TestBareBuffNameBlockParses(t *testing.T) {
	exactly := func(t *testing.T, line string, want ...string) {
		t.Helper()
		mods, extra := parseMod(line, 1)
		testza.AssertEqual(t, "", extra, "the line must parse completely: "+line)
		got := make([]string, 0, len(mods))
		for _, m := range mods {
			got = append(got, m.Name())
		}
		sort.Strings(got)
		sort.Strings(want)
		testza.AssertEqual(t, want, got, "mods for "+line)
	}

	exactly(t, "Rampage", "Condition:Rampage")
	exactly(t, "Soul Eater", "Condition:CanHaveSoulEater")
	exactly(t, "Chaotic Might", "Condition:ChaoticMight")
	exactly(t, "Elusive", "Condition:CanBeElusive")
	exactly(t, "Adrenaline", "Condition:Adrenaline")
	exactly(t, "Arcane Surge", "Condition:ArcaneSurge")
	exactly(t, "Phasing", "Condition:Phasing")
	exactly(t, "Onslaught", "Condition:Onslaught")
	exactly(t, "Unholy Might", "Condition:CanWither", "Condition:UnholyMight")

	t.Run("a bare name must not match inside a longer line", func(t *testing.T) {
		mods, _ := parseMod("Lose Adrenaline when you cease to be Flame-Touched", 1)
		for _, m := range mods {
			if m.Name() == "Condition:Adrenaline" {
				t.Fatalf("bare `adrenaline` matched inside a longer line; PoB emits no mods for it")
			}
		}
	})
}
