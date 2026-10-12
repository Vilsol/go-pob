package calculator

import (
	"os"
	"testing"
)

// TestGemLevelAndQuality pins CalcActiveSkill.lua:542-543 and CalcPerform.lua:3869-3898.
func TestGemLevelAndQuality(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		level float64
	}{
		{"0004-trickster-lightning-tendrils-of-eccentricity.xml", 20},
		{"0005-occultist-vaal-double-strike.xml", 21},
		{"0031-guardian-raise-spectre.xml", 25},
		{"0011-saboteur-cyclone.xml", 28},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output["GemLevel"]
			if !ok {
				t.Fatal("GemLevel ABSENT; PoB publishes it on every corpus build with a levelled skill")
			}
			if got != tc.level {
				t.Errorf("GemLevel = %v, want %v (from the 3.29 oracle blob)", got, tc.level)
			}
		})
	}
}
