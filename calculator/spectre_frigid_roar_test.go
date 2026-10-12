package calculator

import (
	"testing"
)

// TestSpectreFrigidRoarDebuff: the Primal Crushclaw's hidden Frigid Roar reaches the enemy as ColdDamageTaken (spectre.lua:4837-4857, CalcPerform.lua:2889-2935).
func TestSpectreFrigidRoarDebuff(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0795-necromancer-raise-spectre-5.xml")
	if want, got := 1.1204538300004e+06, env.Minion.Output["TotalDPS"]; !oracleClose(got, want, oracleDumpTolerance) {
		t.Errorf("Minion TotalDPS = %v, want %v (PoB)", got, want)
	}
}
