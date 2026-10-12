package calculator

import (
	"math"
	"os"
	"testing"
)

// TestRageGateOpensForItsMissingSources pins three `Condition:CanGainRage` producers that had no parser row here, and with them the whole rage block they gate.
func TestRageGateOpensForItsMissingSources(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		stat  string
		want  float64
		why   string
	}{
		{"0007-elementalist-storm-burst-of-repulsion.xml", "MaximumRage", 30,
			"Ravenous Passion's 'Gain N Rage after Spending a total of 200 Mana' opens the gate"},
		{"0007-elementalist-storm-burst-of-repulsion.xml", "Rage", 30, "the gated block publishes it"},

		{"0004-trickster-lightning-tendrils-of-eccentricity.xml", "MaximumRage", 0,
			"CONTROL: no rage source; the gate must stay shut"},
	} {
		t.Run(tc.build+"/"+tc.stat, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got := env.Player.Output[tc.stat]
			if math.Abs(got-tc.want) > math.Max(1, math.Abs(tc.want))*0.005 {
				t.Errorf("%s = %v, want %v — %s", tc.stat, got, tc.want, tc.why)
			}
		})
	}
}
