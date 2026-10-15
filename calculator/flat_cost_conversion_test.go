package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestPositiveFlatManaCostConvertsWithItsResource pins CalcOffence.lua:1743-1747.
func TestPositiveFlatManaCostConvertsWithItsResource(t *testing.T) {
	const file = "0046-elementalist-arc.xml"
	output := func(lines string) map[string]float64 {
		build := parseCorpusBuild(t, file)
		build.Config.CustomModifierBlocks = append(build.Config.CustomModifierBlocks, pob.CustomModifierBlock{Text: lines})
		return GetMiscCalculator(build).Base().Player.Output
	}
	cases := []struct {
		convert, cost string
		flat, want    float64
		hybrid        bool
	}{
		{"Skills Cost Life instead of Mana", "LifeCost", 4, 4, false},
		{"Skills Cost Life instead of Mana", "LifeCost", -7, 0, false},
		{"Skills Cost Energy Shield instead of Mana or Life", "ESCost", 4, 4, false},
		{"Skills Cost Energy Shield instead of Mana or Life", "ESCost", -7, 0, false},
		{"Skills Cost Life instead of 15% of Mana Cost", "LifeCost", 4, 1, true},
		{"Skills Cost Life instead of 15% of Mana Cost", "LifeCost", -7, 0, true},
	}
	for _, c := range cases {
		base := output(c.convert)
		if base[c.cost] <= 0 {
			t.Fatalf("REFUSING: %s gives %s %v on %s", c.convert, c.cost, base[c.cost], file)
		}
		flat := "+4 to Total Mana Cost"
		if c.flat < 0 {
			flat = "Non-Channelling Skills have -7 to Total Mana Cost"
		}
		got := output(c.convert + "\n" + flat)
		if delta := got[c.cost] - base[c.cost]; delta != c.want {
			t.Errorf("%s, %s: %s moved by %v, want %v", c.convert, flat, c.cost, delta, c.want)
		}
		if !c.hybrid && got["ManaCost"] != 0 {
			t.Errorf("%s, %s: ManaCost %v, want 0", c.convert, flat, got["ManaCost"])
		}
	}
}
