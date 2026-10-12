package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func onlyMod(t *testing.T, line string) mod.Mod {
	t.Helper()
	e := ParseMod(line, false)
	if e.Extra != "" {
		t.Errorf("%q left %q unparsed", line, e.Extra)
		return nil
	}
	if len(e.ModList) != 1 {
		t.Errorf("%q gave %d mods, want 1", line, len(e.ModList))
		return nil
	}
	return e.ModList[0]
}

// TestDegenAndCostFormsAreParsed covers the four modifier forms this port never had.
func TestDegenAndCostFormsAreParsed(t *testing.T) {
	requireGameData(t, "3.29")

	if m := onlyMod(t, "5 Fire Damage taken per second"); m != nil && m.Name() != "FireDegen" {
		t.Fatalf("control: %q parsed as %s, want FireDegen -- the sibling form moved", "5 Fire Damage taken per second", m.Name())
	}

	for _, c := range []struct {
		line string
		name string
		val  float64
	}{
		{"Lose 55.9 Life per second", "LifeDegen", 55.9},
		{"Lose 7% of Mana per Second", "ManaDegenPercent", 7},
		{"You lose 3% of Life per second", "LifeDegenPercent", 3},
		{"Channelling Skills Cost +1 Mana", "ManaCostNoMult", 1},
		{"Non-Channelling Skills Cost +10 Mana", "ManaCostNoMult", 10},
	} {
		m := onlyMod(t, c.line)
		if m == nil {
			continue
		}
		if m.Name() != c.name || m.Value().Float() != c.val {
			t.Errorf("%q gave %s = %v, want %s = %v", c.line, m.Name(), m.Value().Float(), c.name, c.val)
		}
	}
}
