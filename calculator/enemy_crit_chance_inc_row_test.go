package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestEnemyCritChanceIncRowParses pins ModParser.lua:3555.
func TestEnemyCritChanceIncRowParses(t *testing.T) {
	entry := ParseMod("Hits have 170% increased Critical Strike Chance against you", false)
	if strings.Trim(entry.Extra, " ") != "" {
		t.Fatalf("unparsed remainder %q -- the whole line is discarded", entry.Extra)
	}
	if len(entry.ModList) != 1 {
		t.Fatalf("got %d mods, want 1", len(entry.ModList))
	}
	m := entry.ModList[0]
	if m.Name() != "EnemyCritChance" || m.Type() != mod.TypeIncrease {
		t.Fatalf("got %s %s, want EnemyCritChance INC", m.Name(), m.Type())
	}
	if got := m.Value().Float(); got != 170 {
		t.Fatalf("got value %v, want 170", got)
	}

	bare := ParseMod("Hits have 170% increased Critical Strike Chance", false)
	for _, bm := range bare.ModList {
		if bm.Name() == "EnemyCritChance" {
			t.Fatalf("the suffix-less line produced an EnemyCritChance mod — the row is matching " +
				"the stripped remainder, not the full line")
		}
	}
}
