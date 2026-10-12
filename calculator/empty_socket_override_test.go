package calculator

import (
	"testing"
)

// TestEmptySocketConfigOverride: overrideEmpty<Colour>Sockets replaces the counted empty sockets (CalcSetup.lua:1261-1265).
func TestEmptySocketConfigOverride(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "15345-inquisitor-flameblast-of-celerity.xml")
	for k, want := range map[string]float64{"FireResistTotal": 100, "Life": 8906} {
		if got := env.Player.Output[k]; got != want {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}
