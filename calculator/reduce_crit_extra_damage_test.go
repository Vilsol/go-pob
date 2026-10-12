package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestReduceCritExtraDamageFamilyParsesWhole pins ModParser.lua:5846-5861.
func TestReduceCritExtraDamageFamilyParsesWhole(t *testing.T) {
	for _, tc := range []struct {
		line string
		want float64
	}{
		{"Take no Extra Damage from Critical Strikes", 100},
		{"Take no Extra Damage from Critical Strikes if Energy Shield Recharge started Recently", 100},
		{"You take 30% reduced Extra Damage from Critical Strikes by Cursed Enemies", 30},
		{"You take 40% increased Extra Damage from Critical Strikes", -40},
		{"You take 100% reduced Extra Damage from Critical Strikes", 100},
		{"Take no Extra Damage from Critical Strikes if you have a Magic Ring in left slot", 100},
		{"You take no Extra Damage from Critical Strikes while Elusive", 100},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if strings.Trim(e.Extra, " ") != "" {
				t.Fatalf("unparsed remainder %q -- the WHOLE line is discarded", e.Extra)
			}
			if len(e.ModList) != 1 {
				t.Fatalf("got %d mods, want 1", len(e.ModList))
			}
			m := e.ModList[0]
			if m.Name() != "ReduceCritExtraDamage" || m.Type() != mod.TypeBase {
				t.Fatalf("got %s %s, want ReduceCritExtraDamage BASE", m.Name(), m.Type())
			}
			if got := m.Value().Float(); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
