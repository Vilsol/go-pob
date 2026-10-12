package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestHitResInversionParse covers ModParser.lua:2462-2464, :3536 and :3537.
func TestHitResInversionParse(t *testing.T) {
	for _, tc := range []struct {
		line  string
		name  string
		value float64
	}{
		{"Hits have 25% chance to treat Enemy Monster Elemental Resistance values as inverted",
			"HitsInvertEleResChance", 0.25},
		{"Hits have 15% chance to treat Enemy Monster Elemental Resistance values as inverted",
			"HitsInvertEleResChance", 0.15},
		{"Treats Enemy Monster Elemental Resistance values as inverted",
			"HitsInvertEleResChance", 100},
		{"Treats Enemy Monster Chaos Resistance values as inverted",
			"HitsInvertChaosResChance", 100},
	} {
		t.Run(tc.line, func(t *testing.T) {
			mods, extra := parseMod(tc.line, 1)
			testza.AssertEqual(t, "", extra, "%q must parse completely", tc.line)
			if len(mods) != 1 {
				t.Fatalf("%q produced %d mods, want 1", tc.line, len(mods))
			}
			testza.AssertEqual(t, tc.name, mods[0].Name())
			testza.AssertEqual(t, mod.TypeChance, mods[0].Type())
			testza.AssertEqual(t, tc.value, mods[0].Value().ValueFloat)
		})
	}
}
