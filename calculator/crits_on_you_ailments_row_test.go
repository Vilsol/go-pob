package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestCritsOnYouAilmentRowParses pins ModParser.lua:4839.
func TestCritsOnYouAilmentRowParses(t *testing.T) {
	entry := ParseMod("Critical Strikes against you do not inherently inflict Elemental Ailments", false)
	if strings.Trim(entry.Extra, " ") != "" {
		t.Fatalf("unparsed remainder %q -- the whole line is discarded", entry.Extra)
	}
	if len(entry.ModList) != 1 {
		t.Fatalf("got %d mods, want 1", len(entry.ModList))
	}
	m := entry.ModList[0]
	if m.Name() != "CritsOnYouDontAlwaysApplyElementalAilments" || m.Type() != mod.TypeFlag {
		t.Fatalf("got %s %s, want CritsOnYouDontAlwaysApplyElementalAilments FLAG", m.Name(), m.Type())
	}
	if !m.Value().Flag() {
		t.Fatalf("flag parsed as false, want true")
	}
	if m.Name() == "EnemyModifier" {
		t.Fatalf("row was routed to the ENEMY db -- the consumer at CalcDefence.lua:3494 reads the " +
			"PLAYER db, so this would parse cleanly, satisfy every name-based census, and still " +
			"never be read")
	}
}
