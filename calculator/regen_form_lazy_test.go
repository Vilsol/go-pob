package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestRegenFormLazyCapture: the regenerate/lose forms capture (.-) (ModParser.lua:109-118), so a trailing "per second" tag is left for modTagList.
func TestRegenFormLazyCapture(t *testing.T) {
	e := ParseMod("Regenerate 80 Energy Shield per Second per Poison on you, up to 400 per second", false)
	if e.Extra != "" || len(e.ModList) != 1 {
		t.Fatalf("got %d mods, extra %q", len(e.ModList), e.Extra)
	}
	m := e.ModList[0]
	if m.Name() != "EnergyShieldRegen" || m.Type() != mod.TypeBase || m.Value().Float() != 80 {
		t.Fatalf("got %s %s %v", m.Name(), m.Type(), m.Value().Float())
	}
	tags := m.Tags()
	mt, ok := tags[0].(*mod.MultiplierTag)
	if len(tags) != 1 || !ok || mt.TagLimit == nil || *mt.TagLimit != 400 || !mt.TagLimitTotal {
		t.Fatalf("tags %+v", tags)
	}
}

func TestApepsSlumberESRegenCorpus(t *testing.T) {
	requireGameData(t, "3.29")
	for _, b := range []string{"0678-assassin-cremation-of-the-volcano.xml", "1403-pathfinder-eye-of-winter.xml"} {
		env := corpusEnvForLoadedData(t, b)
		if got := env.Player.Output["EnergyShieldRegen"]; got != 400 {
			t.Errorf("%s EnergyShieldRegen = %v, want 400 (PoB)", b, got)
		}
	}
}
