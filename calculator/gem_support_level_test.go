package calculator

import (
	"os"
	"testing"
)

// TestGemSupportedPropertyGrantsReachTheActiveSkill pins CalcActiveSkill.lua:549-559.
func TestGemSupportedPropertyGrantsReachTheActiveSkill(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		level float64
		why   string
	}{
		{"0445-elementalist-vaal-reap-of-butchery.xml", 30,
			"largest cluster (92 builds at exactly 0.8000); was 24, six levels short"},
		{"0385-assassin-ice-trap.xml", 31,
			"was a known gap at 27; the socket-colour GemProperty fix closed it to PoB's 31"},
		{"0375-pathfinder-toxic-rain-12.xml", 28,
			"third cluster (75 builds at 0.8214); was 23, five levels short"},

		{"0004-trickster-lightning-tendrils-of-eccentricity.xml", 20,
			"CONTROL: no support grant; must not move"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output["GemLevel"]
			if !ok {
				t.Fatal("GemLevel ABSENT; PoB publishes it on every corpus build with a levelled skill")
			}
			if got != tc.level {
				t.Errorf("GemLevel = %v, want %v (3.29 oracle) — %s", got, tc.level, tc.why)
			}
		})
	}
}
