package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestBareRecoveryKeyDoesNotShadow pins ModParser.lua:861's bare `recovery` name key together with every longer `...
func TestBareRecoveryKeyDoesNotShadow(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
		typ  mod.Type
		val  float64
	}{
		{"101% more Recovery if used while on Low Life", "FlaskLifeRecoveryLowLife", mod.TypeMore, 101},
		{"66% increased Recovery rate", "FlaskRecoveryRate", mod.TypeIncrease, 66},
		{"25% increased Life Recovery from Flasks", "FlaskLifeRecovery", mod.TypeIncrease, 25},
		{"Instant Recovery", "FlaskInstantRecovery", mod.TypeBase, 100},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v, want 1 mod", tc.line, e)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			m := e.ModList[0]
			if m.Name() != tc.name || m.Type() != tc.typ || m.Value().Float() != tc.val {
				t.Errorf("got %s %v %v, want %s %v %v",
					m.Name(), m.Type(), m.Value().Float(), tc.name, tc.typ, tc.val)
			}
		})
	}
}
