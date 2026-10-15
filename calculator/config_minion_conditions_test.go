package calculator

import (
	"testing"
)

// TestConfigMinionsConditionLowLife: minionsConditionLowLife reaches the minion as Condition:LowLife (ConfigOptions.lua:201-203).
func TestConfigMinionsConditionLowLife(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0789-necromancer-animate-guardians-weapon.xml")
	if want, got := 12727142.524895, env.Minion.Output["TotalDPS"]; !oracleClose(got, want, oracleDumpTolerance) {
		t.Errorf("Minion TotalDPS = %v, want %v (PoB)", got, want)
	}
}

// TestConfigMultiplierBarnacleStacks: the enemy's Barnacle stacks lengthen chill (ConfigOptions.lua:1892-1897).
func TestConfigMultiplierBarnacleStacks(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1055-saboteur-blade-blast-of-unloading-2.xml")
	for k, want := range map[string]float64{"FreezeDurationMod": 1.55, "ChillDuration": 3.1} {
		if got := env.Player.Output[k]; !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}

// TestConfigMultiplierUsedVaalSkillInPast8Seconds: the Vaal-skill count raises the soul cost (ConfigOptions.lua:1549-1551).
func TestConfigMultiplierUsedVaalSkillInPast8Seconds(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "15376-trickster-vaal-blade-vortex.xml")
	if want, got := 17.0, env.Player.Output["SoulCost"]; !oracleClose(got, want, oracleDumpTolerance) {
		t.Errorf("SoulCost = %v, want %v (PoB)", got, want)
	}
}
