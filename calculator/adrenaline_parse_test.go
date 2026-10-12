package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAdrenalineAndLinkedMinionLinesParse pins the producers for consumers that were already complete here.
func TestAdrenalineAndLinkedMinionLinesParse(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
	}{
		{"Gain Adrenaline for 3 seconds on Kill", "Condition:Adrenaline"},
		{"Gain Adrenaline when you become Flame-Touched", "Condition:Adrenaline"},
		{"Your Linked Minions take 66% less Damage", "MinionModifier"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v mods, want 1", tc.line, e)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			if got := e.ModList[0].Name(); got != tc.name {
				t.Errorf("mod name = %q, want %q", got, tc.name)
			}
		})
	}

	if e := ParseMod("Lose Adrenaline when you cease to be Flame-Touched", false); e != nil && len(e.ModList) > 0 {
		t.Errorf("the pin produces no mods for the 'lose adrenaline' line; we produced %d", len(e.ModList))
	}

	_ = mod.TypeBase
}
