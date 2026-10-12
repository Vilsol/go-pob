package calculator

import (
	"math"
	"os"
	"testing"
)

// TestLifePercentageOutputs pins CalcPerform.lua:74-77.
//
//	local lowLifePerc = modDB:Sum("BASE", nil, "LowLifePercentage")
//	local fullLifePerc = modDB:Sum("BASE", nil, "FullLifePercentage")
func TestLifePercentageOutputs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build     string
		low, full float64
	}{
		{"0001-necromancer-unknown.xml", 50, 100},
		{"0021-assassin-blade-flurry.xml", 55, 100},
		{"0104-gladiator-boneshatter-of-complex-trauma.xml", 55, 90},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			for _, c := range []struct {
				name string
				want float64
			}{
				{"LowLifePercentage", tc.low},
				{"FullLifePercentage", tc.full},
			} {
				got, ok := env.Player.Output[c.name]
				if !ok {
					t.Errorf("%s ABSENT; PoB publishes it on every corpus build", c.name)
					continue
				}
				if math.Abs(got-c.want) > 0.0005 {
					t.Errorf("%s = %v, want %v (from the 3.29 oracle blob)", c.name, got, c.want)
				}
			}
		})
	}
}
