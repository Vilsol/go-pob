package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestDefianceOfDestinyParses: Defiance of Destiny's two mod lines.
func TestDefianceOfDestinyParses(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
	}{
		{"Gain 30% of Missing Unreserved Life before being Hit by an Enemy", "MissingLifeBeforeEnemyHit"},
		{"Gain 20% of Missing Unreserved Mana before being Hit by an Enemy", "MissingManaBeforeEnemyHit"},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || len(e.ModList) == 0 {
			t.Fatalf("ParseMod(%q) returned NOTHING — the whole line is discarded, not just a term", tc.line)
		}
		found := false
		for _, m := range e.ModList {
			if m.Name() == tc.name {
				found = true
				if m.Type() != mod.TypeBase {
					t.Errorf("%s: type %v, want BASE", tc.name, m.Type())
				}
			}
		}
		if !found {
			t.Errorf("ParseMod(%q) produced %d mod(s) but none named %s", tc.line, len(e.ModList), tc.name)
		}
	}
}
