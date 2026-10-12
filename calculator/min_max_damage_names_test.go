package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestMinMaxDamageModNames covers ModParser.lua:690-703 — the ten `minimum`/`maximum <type> damage` names.
func TestMinMaxDamageModNames(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
		typ  mod.Type
		val  float64
	}{
		{"50% less Minimum Lightning Damage", "MinLightningDamage", mod.TypeMore, -50},
		{"25% more Maximum Lightning Damage", "MaxLightningDamage", mod.TypeMore, 25},
		{"20% increased Minimum Cold Damage", "MinColdDamage", mod.TypeIncrease, 20},
		{"20% increased Maximum Cold Damage", "MaxColdDamage", mod.TypeIncrease, 20},
		{"10% increased Minimum Fire Damage", "MinFireDamage", mod.TypeIncrease, 10},
		{"10% increased Maximum Fire Damage", "MaxFireDamage", mod.TypeIncrease, 10},
		{"10% increased Minimum Chaos Damage", "MinChaosDamage", mod.TypeIncrease, 10},
		{"10% increased Maximum Chaos Damage", "MaxChaosDamage", mod.TypeIncrease, 10},
		{"30% more Maximum Physical Damage", "MaxPhysicalDamage", mod.TypeMore, 30},
		{"30% more Minimum Physical Damage", "MinPhysicalDamage", mod.TypeMore, 30},
	} {
		t.Run(tc.line, func(t *testing.T) {
			mods, extra := parseMod(tc.line, 1)
			testza.AssertEqual(t, "", extra, "%q must parse completely — an unmatched remainder "+
				"discards the whole line", tc.line)
			if len(mods) != 1 {
				t.Fatalf("%q produced %d mods, want 1", tc.line, len(mods))
			}
			testza.AssertEqual(t, tc.name, mods[0].Name())
			testza.AssertEqual(t, tc.typ, mods[0].Type())
			testza.AssertEqual(t, tc.val, mods[0].Value().ValueFloat)
		})
	}

	for _, tc := range []struct{ line, name string }{
		{"20% increased Lightning Damage", "LightningDamage"},
		{"20% increased Cold Damage", "ColdDamage"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			mods, extra := parseMod(tc.line, 1)
			testza.AssertEqual(t, "", extra)
			if len(mods) != 1 {
				t.Fatalf("%q produced %d mods, want 1", tc.line, len(mods))
			}
			testza.AssertEqual(t, tc.name, mods[0].Name())
		})
	}
}
