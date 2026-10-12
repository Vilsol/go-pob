package calculator

import (
	"testing"
)

// TestMinionInstabilityHitTimeFollowsSummonSpeed: Minion Instability hits once per summoned minion (minion.lua:1757-1759, CalcOffence.lua:3977-3979).
func TestMinionInstabilityHitTimeFollowsSummonSpeed(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0256-necromancer-summon-raging-spirit-of-enormity.xml")
	for k, want := range map[string]float64{"HitSpeed": 1.8, "TotalDPS": 9373385.466549} {
		if got := env.Minion.Output[k]; !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("Minion %s = %v, want %v (PoB)", k, got, want)
		}
	}
}
