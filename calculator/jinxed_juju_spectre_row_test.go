package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestJinxedJujuSpectreRowParses pins ModParser.lua:5036 and Data.lua:761 (upstream #10298).
func TestJinxedJujuSpectreRowParses(t *testing.T) {
	for _, line := range []string{
		"20% of Damage from Hits is taken from your Raised Spectres' Life before you",
		"20% of Damage from Hits is taken from your Spectres' Life before you",
	} {
		entry := ParseMod(line, false)
		if strings.TrimSpace(entry.Extra) != "" || len(entry.ModList) != 1 {
			t.Fatalf("%q: extra %q, %d mods; want one mod", line, entry.Extra, len(entry.ModList))
		}
		if m := entry.ModList[0]; m.Name() != "takenFromSpectresBeforeYou" || m.Type() != mod.TypeBase || m.Value().Float() != 20 {
			t.Errorf("%q: got %s %s %v, want takenFromSpectresBeforeYou BASE 20", line, m.Name(), m.Type(), m.Value().Float())
		}
	}
	found := false
	for _, p := range itemTagSpecialExclusionPattern["life"]["amulet"] {
		if p == "Spectres' Life" {
			found = true
		}
	}
	if !found {
		t.Error(`amulet life exclusions lack "Spectres' Life"`)
	}
}
