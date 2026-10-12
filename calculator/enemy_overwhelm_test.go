package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestEnemyOverwhelmRowParses pins ModParser.lua:5517-5519, the producer for CalcDefence.lua:1763's `EnemyPhysicalOverwhelm` term.
func TestEnemyOverwhelmRowParses(t *testing.T) {
	entry := ParseMod("Hits against you Overwhelm 15% of Physical Damage Reduction", false)
	if strings.Trim(entry.Extra, " ") != "" {
		t.Fatalf("unparsed remainder %q -- the whole line is discarded", entry.Extra)
	}
	if len(entry.ModList) != 1 {
		t.Fatalf("got %d mods, want 1", len(entry.ModList))
	}
	m := entry.ModList[0]
	if m.Name() != "EnemyPhysicalOverwhelm" || m.Type() != mod.TypeBase {
		t.Fatalf("got %s %s, want EnemyPhysicalOverwhelm BASE", m.Name(), m.Type())
	}
	if v := m.Value(); v == nil || v.Float() != 15 {
		t.Fatalf("got value %v, want 15", m.Value())
	}

	sibling := ParseMod("Overwhelm 15% Physical Damage Reduction", false)
	if len(sibling.ModList) != 1 || sibling.ModList[0].Name() == "EnemyPhysicalOverwhelm" {
		t.Fatalf("the player-side Overwhelm row produced %v", sibling.ModList)
	}
}
