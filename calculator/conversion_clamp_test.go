package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestConversionSumsClampAtZero pins PoB's m_max(..., 0) on every conversion sum.
func TestConversionSumsClampAtZero(t *testing.T) {
	for _, tc := range []struct {
		name     string
		modName  string
		value    float64
		wantMult float64
	}{
		{"a negative global conversion cannot raise mult above 1",
			"PhysicalDamageConvertToChaos", -50, 1},
		{"a negative SKILL conversion cannot raise mult above 1",
			"SkillPhysicalDamageConvertToChaos", -50, 1},
		{"an ordinary conversion still reduces mult",
			"PhysicalDamageConvertToChaos", 60, 0.4},
		{"conversion beyond 100% floors mult at 0, never negative",
			"PhysicalDamageConvertToChaos", 150, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			list := moddb.NewModList()
			list.AddMod(mod.NewFloat(tc.modName, mod.TypeBase, tc.value))

			table := buildConversionTable(list, nil)
			got := table[data.DamageTypePhysical].Mult
			if got != tc.wantMult {
				t.Errorf("%s = %v: Physical mult %v, want %v", tc.modName, tc.value, got, tc.wantMult)
			}
			if got > 1 {
				t.Errorf("mult %v exceeds 1 — the hit would keep more of its base than it started "+
					"with, while the destination types still receive their share", got)
			}
		})
	}
}
