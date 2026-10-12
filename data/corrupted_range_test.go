package data

import (
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/utils"
)

func baseStr(p *float64) string {
	if p == nil {
		return "<absent>"
	}
	return strconv.FormatFloat(*p, 'g', -1, 64)
}

// TestCorruptedRangeIsABaseValueScalar pins PoB's fifth applyValueScalar parameter.
func TestCorruptedRangeIsABaseValueScalar(t *testing.T) {
	one := 1.0
	for _, c := range []struct {
		line        string
		valueScalar float64
		base        *float64
		precision   *int
		want        string
	}{
		{"16% increased Attack Speed", 1, utils.Ptr(1.21), nil, "19% increased Attack Speed"},
		{"+1000 to maximum Life", 1, utils.Ptr(1.22), nil, "+1220 to maximum Life"},
		{"Penetrate 4% Elemental Resistances per Abyss Jewel affecting you", 1, utils.Ptr(1.22), nil,
			"Penetrate 5% Elemental Resistances per Abyss Jewel affecting you"},
		{"Ignites you inflict with Attacks deal Damage 35% faster", 1, utils.Ptr(1.22), nil,
			"Ignites you inflict with Attacks deal Damage 43% faster"},

		{"25% increased Damage", 1.2, utils.Ptr(1.22), nil, "37% increased Damage"},

		{"7% increased Damage", 1, utils.Ptr(0.78), nil, "5% increased Damage"},

		{"3% increased Damage", 1, utils.Ptr(1.13), nil, "3% increased Damage"},

		{"1.3% increased Damage", 1, utils.Ptr(1.22), utils.Ptr(1), "1.6% increased Damage"},

		{"+25 to Strength", 0.8, utils.Ptr(1.22), nil, "+24 to Strength"},

		{"16% increased Attack Speed", 1, &one, nil, "16% increased Attack Speed"},
		{"16% increased Attack Speed", 1, nil, nil, "16% increased Attack Speed"},
	} {
		got := ApplyValueScalar(c.line, &c.valueScalar, c.base, 1, c.precision)
		if got != c.want {
			t.Errorf("ApplyValueScalar(%q, %v, %v) =\n  %q\nwant\n  %q\n  (ItemTools.lua:37-57, "+
				"driven under the pinned PoB)", c.line, c.valueScalar, baseStr(c.base), got, c.want)
		}
	}
}

// TestCorruptedRangeReachesApplyRange is the same values one level up, through the function the three call sites actually use.
func TestCorruptedRangeReachesApplyRange(t *testing.T) {
	for _, c := range []struct {
		line     string
		rangeVal float64
		base     *float64
		want     string
	}{
		{"(20-30)% increased Armour", 1, utils.Ptr(1.22), "37% increased Armour"},
		{"Adds (8-12) to (14-20) Physical Damage", 1, utils.Ptr(1.22), "Adds 15 to 24 Physical Damage"},
		{"+(15-25) to Strength", 0.817, utils.Ptr(1.22), "+28 to Strength"},
		{"(20-35)% reduced Reservation Efficiency of Skills", 0, utils.Ptr(0.78),
			"16% reduced Reservation Efficiency of Skills"},
		{"(50-100)% increased Spell Critical Strike Chance per Raised Spectre", 1, utils.Ptr(1.15),
			"115% increased Spell Critical Strike Chance per Raised Spectre"},
		{"+(20-30)% to Quality of all Skill Gems", 1, utils.Ptr(1.22), "+37% to Quality of all Skill Gems"},
		{"(3-4)% increased Defences per Minion", 1, utils.Ptr(1.13), "5% increased Defences per Minion"},

		{"(20-30)% increased Armour", 1, nil, "30% increased Armour"},
	} {
		one := 1.0
		got := ApplyRange(c.line, c.rangeVal, &one, c.base)
		if got != c.want {
			t.Errorf("ApplyRange(%q, %v, 1, %v) =\n  %q\nwant\n  %q\n  (ItemTools.lua:62-111, "+
				"driven under the pinned PoB)", c.line, c.rangeVal, baseStr(c.base), got, c.want)
		}
	}
}
