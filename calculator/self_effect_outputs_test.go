package calculator

import (
	"os"
	"testing"
)

// TestSelfEffectBlockOutputs pins the three unported lines of CalcDefence.lua:1592-1598.
func TestSelfEffectBlockOutputs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build          string
		lightRadiusInc float64
	}{
		{"0001-necromancer-unknown.xml", 9},
		{"0004-trickster-lightning-tendrils-of-eccentricity.xml", 42},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)

			for _, c := range []struct {
				name string
				want float64
			}{
				{"LightRadiusInc", tc.lightRadiusInc},
				{"ExposureEffectOnSelf", 100},
				{"WitherEffectOnSelf", 100},
			} {
				got, ok := env.Player.Output[c.name]
				if !ok {
					t.Errorf("%s ABSENT; PoB publishes it on every corpus build", c.name)
					continue
				}
				if got != c.want {
					t.Errorf("%s = %v, want %v (from the 3.29 oracle blob)", c.name, got, c.want)
				}
			}
		})
	}
}
