package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestOverrideFormIsParsed covers ModParser.lua:154's `is (%-?%d+)%%?
func TestOverrideFormIsParsed(t *testing.T) {
	requireGameData(t, "3.29")

	only := func(line string) (mod.Mod, string) {
		t.Helper()
		e := ParseMod(line, false)
		if len(e.ModList) != 1 {
			t.Errorf("%q gave %d mods, want 1 (extra %q)", line, len(e.ModList), e.Extra)
			return nil, e.Extra
		}
		return e.ModList[0], e.Extra
	}

	if m, extra := only("Your Movement Speed is 150% of its base value"); m != nil {
		if m.Name() != "MovementSpeed" || m.Type() != mod.TypeOverride || m.Value().Float() != 1.5 || extra != "" {
			t.Errorf("control: movement speed parsed as %s %s %v (extra %q); the form is shadowing specialModList",
				m.Name(), m.Type(), m.Value().Float(), extra)
		}
	}

	if m, extra := only("10% increased Movement Speed"); m != nil {
		if m.Name() != "MovementSpeed" || m.Type() != mod.TypeIncrease || m.Value().Float() != 10 || extra != "" {
			t.Errorf("control: %s %s %v (extra %q), want MovementSpeed INC 10", m.Name(), m.Type(), m.Value().Float(), extra)
		}
	}

	if m, extra := only("Chaos Resistance is 0"); m != nil {
		if m.Name() != "ChaosResist" || m.Type() != mod.TypeOverride || m.Value().Float() != 0 || extra != "" {
			t.Errorf("%q gave %s %s %v (extra %q), want ChaosResist OVERRIDE 0",
				"Chaos Resistance is 0", m.Name(), m.Type(), m.Value().Float(), extra)
		}
	}
}
