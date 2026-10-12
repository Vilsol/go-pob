package moddb

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestScaleAddListHighPrecision pins ModStore.lua:66-72, the high-precision arm of ScaleAddMod.
//
//	         or (m_floor(value) ~= value and data.defaultHighPrecision)
//	if precision then value = m_floor(value * scale * 10^p) / 10^p
//	else               value = m_modf(round(value * scale, 2))     -- TRUNCATES
func TestScaleAddListHighPrecision(t *testing.T) {
	for _, tc := range []struct {
		name    string
		modName string
		modType mod.Type
		value   float64
		scale   float64
		want    float64
		why     string
	}{
		{
			name: "table hit keeps 2dp", modName: "LifeRegenPercent", modType: mod.TypeBase,
			value: 3, scale: 1.95, want: 5.85,
			why: "highPrecisionMods[LifeRegenPercent][BASE] = 2; truncation gave 5",
		},
		{
			name: "no table entry still truncates", modName: "Damage", modType: mod.TypeBase,
			value: 3, scale: 1.95, want: 5,
			why: "else-branch: m_modf(round(5.85, 2)) truncates to 5",
		},
		{
			name: "non-integer value gets defaultHighPrecision", modName: "Damage", modType: mod.TypeBase,
			value: 2.5, scale: 1.95, want: 4.8,
			why: "floor(2.5*1.95*10)/10 = floor(48.75)/10 = 4.8, NOT 4.875",
		},
		{
			name: "MORE entry keeps 4dp", modName: "SupportManaMultiplier", modType: mod.TypeMore,
			value: 3, scale: 1.5, want: 4.5,
			why: "highPrecisionMods[SupportManaMultiplier][MORE] = 4",
		},
		{
			name: "table entry does not leak across types", modName: "LifeRegenPercent", modType: mod.TypeIncrease,
			value: 3, scale: 1.95, want: 5,
			why: "no INC precision for this name; must truncate",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := NewModList()
			src.AddMod(mod.NewFloat(tc.modName, tc.modType, tc.value))

			dst := NewModList()
			dst.ScaleAddList(src, tc.scale)

			mods := dst.Mods()
			if len(mods) != 1 {
				t.Fatalf("expected exactly 1 scaled mod, got %d", len(mods))
			}
			got := mods[0].Value().Float()
			if got != tc.want {
				t.Errorf("%s %s %v scaled by %v = %v, want %v — %s",
					tc.modName, tc.modType, tc.value, tc.scale, got, tc.want, tc.why)
			}

			if v := src.Mods()[0].Value().Float(); v != tc.value {
				t.Errorf("source mutated: %v, want %v", v, tc.value)
			}
		})
	}
}
