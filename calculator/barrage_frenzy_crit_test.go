package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestBarrageFrenzyCritPerEnduranceParses pins ModParser.lua:3528.
func TestBarrageFrenzyCritPerEnduranceParses(t *testing.T) {
	e := ParseMod("Barrage and Frenzy have 40% increased Critical Strike Chance per Endurance Charge", false)
	if e == nil || strings.TrimSpace(modExtra(e)) != "" || len(e.ModList) != 1 {
		t.Fatalf("want one CritChance mod with no Extra, got %+v", e)
	}
	m := e.ModList[0]
	if m.Name() != "CritChance" || m.Type() != mod.TypeIncrease || m.Value().ValueFloat != 40 {
		t.Fatalf("got %s %s %v", m.Name(), m.Type(), m.Value().ValueFloat)
	}

	env := (&Calculator{PoB: loadCorpusBuild(t, "15625-slayer-frenzy.xml")}).BuildOutput(OutputModeMain)
	if got := env.Player.Output["CritChance"]; got != 100 {
		t.Errorf("CritChance = %v, want 100", got)
	}
}
