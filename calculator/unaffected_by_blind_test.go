package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestUnaffectedByBlindParses adds ModParser.lua:5424, `["unaffected by blind"] = { flag("UnaffectedByBlind") }`, which had no counterpart here.
func TestUnaffectedByBlindParses(t *testing.T) {
	for _, line := range []string{
		"unaffected by blind",
		"blind does not affect your chance to hit",
	} {
		entry := ParseMod(line, false)
		if entry == nil || len(entry.ModList) != 1 {
			got := 0
			extra := ""
			if entry != nil {
				got = len(entry.ModList)
				extra = entry.Extra
			}
			t.Errorf("%q: got %d mods (extra %q), want 1 flag", line, got, extra)
			continue
		}
		m := entry.ModList[0]
		if m.Name() != "UnaffectedByBlind" || m.Type() != mod.TypeFlag {
			t.Errorf("%q: got %s/%s, want UnaffectedByBlind/FLAG", line, m.Name(), m.Type())
		}
	}
}
