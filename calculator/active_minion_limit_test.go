package calculator

import (
	"os"
	"testing"
)

// TestActiveMinionLimitPortsBothHalvesOfCalcOffence1023 pins CalcOffence.lua:1023-1025 on two real builds, expected values read from PoB's oracle for each.
func TestActiveMinionLimitPortsBothHalvesOfCalcOffence1023(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	cases := []struct {
		build string
		want  float64
	}{
		{"1047-elementalist-summon-flame-golem-of-hordes-2.xml", 20},
		{"0503-reliquarian-vaal-blade-vortex.xml", 22},
	}
	for _, c := range cases {
		env := corpusEnvForLoadedData(t, c.build)
		if got := env.Player.Output["ActiveMinionLimit"]; got != c.want {
			t.Errorf("%s: ActiveMinionLimit = %v, PoB's oracle has %v", c.build, got, c.want)
		}
	}
}
