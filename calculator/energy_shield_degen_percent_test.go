package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestEnergyShieldDegenPercentUsesTheForm pins ModParser.lua:118-121.
func TestEnergyShieldDegenPercentUsesTheForm(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line string
		name string
		val  float64
	}{
		{"Lose 5% of Energy Shield per second", "EnergyShieldDegenPercent", 5},
		{"You lose 5% of Energy Shield per second", "EnergyShieldDegenPercent", 5},
		{"Lose 1% of Life per second", "LifeDegenPercent", 1},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || e.Extra != "" || len(e.ModList) != 1 {
			t.Errorf("%q: want one mod parsed whole, got %+v", tc.line, e)
			continue
		}
		m := e.ModList[0]
		if m.Name() != tc.name || m.Type() != mod.TypeBase || m.Value().Float() != tc.val || len(m.Tags()) != 0 {
			t.Errorf("%q: got %s %s %v tags=%d, want %s BASE %v untagged", tc.line, m.Name(), m.Type(), m.Value().Float(), len(m.Tags()), tc.name, tc.val)
		}
	}

	e := ParseMod("Lose 2% of Life per second if you have been Hit Recently", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 {
		t.Fatalf("hit-recently life degen: want one mod parsed whole, got %+v", e)
	}
	m := e.ModList[0]
	if m.Name() != "LifeDegenPercent" || m.Value().Float() != 2 || len(m.Tags()) != 1 {
		t.Errorf("hit-recently life degen: got %s %v tags=%d, want LifeDegenPercent 2 with one tag", m.Name(), m.Value().Float(), len(m.Tags()))
	} else if c, ok := m.Tags()[0].(*mod.ConditionTag); !ok || len(c.VarList) != 1 || c.VarList[0] != "BeenHitRecently" {
		t.Errorf("hit-recently life degen: tag %#v, want Condition BeenHitRecently", m.Tags()[0])
	}
}
