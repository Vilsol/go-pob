package calculator

import (
	"math"
	"testing"
)

// TestAttackCooldownClampsPassSpeed: CalcOffence.lua:2357-2360 reads `globalOutput.Cooldown`, COPIES it into the pass table and only then clamps the pass's Speed.
//
//	if globalOutput.Cooldown then
//		output.Speed = m_min(output.Speed, 1 / output.Cooldown * output.Repeats)
//	end
func TestAttackCooldownClampsPassSpeed(t *testing.T) {
	build := loadCorpusBuild(t, "2909-ladder-Correction_2725.SussyHeraldEnjoyer.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	if env.Minion == nil || env.Minion.Actor == nil {
		t.Fatal("REFUSING: build has no minion actor, so this fixture proves nothing")
	}
	main := env.Minion.OutputTable[OutTableMainHand]

	if got, want := main["Cooldown"], 7.524; math.Abs(got-want) > 1e-6 {
		t.Errorf("Minion MainHand Cooldown = %v, want %v (globalOutput.Cooldown copied into the pass)", got, want)
	}
	if got, want := main["Speed"], 1.0/7.524; math.Abs(got-want) > 1e-9 {
		t.Errorf("Minion MainHand Speed = %v, want %v (clamped to 1/cooldown)", got, want)
	}
	if got, want := env.Minion.Output["Speed"], 1.0/7.524; math.Abs(got-want) > 1e-9 {
		t.Errorf("Minion Speed = %v, want %v", got, want)
	}
}
