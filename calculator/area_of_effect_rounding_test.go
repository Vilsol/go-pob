package calculator

import "testing"

// TestAreaOfEffectIntegerStages pins CalcOffence.lua:160-178 (#10268).
func TestAreaOfEffectIntegerStages(t *testing.T) {
	if got := calcAreaMod(1.01, 1.5); got != 1.51 {
		t.Errorf("calcAreaMod(1.01, 1.5) = %v, want 1.51", got)
	}
	if got := calcAreaMod(0.5, 1.5); got != 0.75 {
		t.Errorf("calcAreaMod(0.5, 1.5) = %v, want 0.75", got)
	}
	if got := calcRadius(15, 1, 1.3); got != 19 {
		t.Errorf("calcRadius(15, 1, 1.3) = %v, want 19", got)
	}
}

func TestAreaOfEffectIntegerStagesOnCarriers(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0052-juggernaut-cleave.xml")
	if got := env.Player.Output["AreaOfEffectRadius"]; got != 34 {
		t.Errorf("Cleave AreaOfEffectRadius = %v, want 34", got)
	}
	env = corpusEnvForLoadedData(t, "0869-gladiator-animate-guardian-2.xml")
	if env.Minion == nil {
		t.Fatal("REFUSING: no minion")
	}
	if got := env.Minion.Output["AreaOfEffectRadius"]; got != 60 {
		t.Errorf("Minion AreaOfEffectRadius = %v, want 60", got)
	}
}
