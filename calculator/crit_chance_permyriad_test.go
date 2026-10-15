package calculator

import "testing"

// TestCritChanceStoredToPermyriad pins CalcOffence.lua:3106-3107 (#10264): crit chance is rounded to 0.01% before the cap.
func TestCritChanceStoredToPermyriad(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0001-necromancer-unknown.xml")
	if got := env.Player.Output["CritChance"]; got != 13.27 {
		t.Errorf("CritChance = %v, want 13.27", got)
	}
	if env.Minion == nil {
		t.Fatal("REFUSING: no minion")
	}
	if got := env.Minion.Output["CritChance"]; got != 97.3 {
		t.Errorf("Minion CritChance = %v, want 97.3", got)
	}
	env = corpusEnvForLoadedData(t, "0005-occultist-vaal-double-strike.xml")
	if got := env.Player.Output["CritChance"]; got != 87.86 {
		t.Errorf("CritChance = %v, want 87.86", got)
	}
}
