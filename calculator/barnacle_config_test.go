package calculator

import "testing"

// TestBarnacleStacksConfig: multiplierBarnacleStacks converts enemy physical to cold and lengthens chill/freeze on it (ConfigOptions.lua:1892-1897).
func TestBarnacleStacksConfig(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1055-saboteur-blade-blast-of-unloading-2.xml")
	for k, want := range map[string]float64{"ChillDuration": 3.1, "FreezeDurationMod": 1.55, "PhysicalRecoupableDamageTaken": 2254, "ColdRecoupableDamageTaken": 802} {
		if got := env.Player.Output[k]; got != want {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}
