package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSpiritInfusionRow pins ModParser.lua:2955, the ONLY producer of `Condition:CanGainSpiritInfusion` upstream.
func TestSpiritInfusionRow(t *testing.T) {
	e := ParseMod("Gain Spirit Infusion every 0.5 seconds while Channelling a Spell", false)
	if e == nil || e.Extra != "" {
		t.Fatalf("line did not parse whole: %v", e)
	}
	if len(e.ModList) != 1 {
		t.Fatalf("got %d mods, want exactly 1: %v", len(e.ModList), e.ModList)
	}
	m := e.ModList[0]
	if m.Name() != "Condition:CanGainSpiritInfusion" || m.Type() != mod.TypeFlag || !m.Value().Flag() {
		t.Fatalf("got %s %s %v, want Condition:CanGainSpiritInfusion FLAG true",
			m.Name(), m.Type(), m.Value())
	}

	for _, line := range []string{
		"Gain Spirit Infusion every 1 second while Channelling a Spell",
		"Gain Spirit Infusion every 2 seconds while Channelling a Spell",
	} {
		e := ParseMod(line, false)
		if e == nil || e.Extra != "" || len(e.ModList) != 1 ||
			e.ModList[0].Name() != "Condition:CanGainSpiritInfusion" {
			t.Errorf("%q: got %v, want the same single flag", line, e)
		}
	}
}
