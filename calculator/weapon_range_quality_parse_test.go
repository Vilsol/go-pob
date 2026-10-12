package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestWeaponRangePer10QualityLinesParse pins ModParser.lua:5757-5758, whose METRES arm was absent while its bare-units sibling was carried.
func TestWeaponRangePer10QualityLinesParse(t *testing.T) {
	for _, tc := range []struct {
		line string
		want float64
	}{
		{"+0.1 metres to Weapon Range per 10% Quality", 1},
		{"+0.2 metre to Weapon Range per 10% Quality", 2},
		{"+2 Weapon Range per 10% Quality", 2},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v mods, want 1", tc.line, e)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			m := e.ModList[0]
			if got := m.Name(); got != "AlternateQualityLocalWeaponRangePer10Quality" {
				t.Fatalf("mod name = %q, want AlternateQualityLocalWeaponRangePer10Quality", got)
			}
			if m.Type() != mod.TypeBase {
				t.Errorf("mod type = %v, want BASE", m.Type())
			}
			if got := m.Value().Float(); got != tc.want {
				t.Errorf("value = %v, want %v", got, tc.want)
			}
		})
	}
}
