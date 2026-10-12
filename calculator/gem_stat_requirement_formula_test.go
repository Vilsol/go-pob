package calculator

import "testing"

// TestGemStatRequirementUsesThePinnedFormula pins calcLib.getGemStatRequirement, CalcTools.lua:170-180.
//
//	local statType = 0.7
//	if isSupport then statType = 0.5 end
//	local req = round( ( 20 + ( level - 3 ) * 3 ) * ( multi / 100 ) ^ 0.9 * statType )
func TestGemStatRequirementUsesThePinnedFormula(t *testing.T) {
	cases := []struct {
		name      string
		level     float64
		isSupport bool
		multi     float64
		want      float64
	}{
		{"Molten Shell Str", 70, false, 100, 155},
		{"Automation Dex", 66, false, 40, 64},
		{"Automation Int", 66, false, 60, 92},
		{"Storm Burst Str", 68, false, 40, 66},
		{"Storm Burst Int", 68, false, 60, 95},
		{"Infused Channelling Dex", 49, false, 40, 48},
		{"Infused Channelling Int", 49, false, 60, 70},
		{"Herald of Ash Str", 70, false, 60, 98},
		{"Herald of Ash Int", 70, false, 40, 68},
		{"below the 14 floor", 1, false, 40, 0},
		{"no requirement", 70, false, 0, 0},

		{"active 60 at 36", 36, false, 60, 53},
		{"active 60 at 45", 45, false, 60, 65},
		{"active 60 at 88", 88, false, 60, 122},
		{"active 40 at 11", 11, false, 40, 14},
		{"active 40 at 23", 23, false, 40, 25},
		{"support 60 at 19", 19, true, 60, 21},
		{"support 40 at 26", 26, true, 40, 20},
		{"support 75 at 40", 40, true, 75, 51},
		{"support 75 at 60", 60, true, 75, 74},
	}
	for _, c := range cases {
		if got := getGemStatRequirement(c.level, c.isSupport, c.multi); got != c.want {
			t.Errorf("%s: getGemStatRequirement(%v, %v, %v) = %v, PoB %v",
				c.name, c.level, c.isSupport, c.multi, got, c.want)
		}
	}
}
