package calculator

import (
	"math"
	"testing"
)

// TestAilmentCfgCarriesSummonSkillName: every ailment / dot cfg upstream copies skillCfg.summonSkillName (CalcOffence.lua:4294, :4567, :4872, :5581, :5652).
func TestAilmentCfgCarriesSummonSkillName(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0956-gladiator-herald-of-agony-4.xml")
	if env.Minion == nil {
		t.Fatalf("fixture premise moved: no minion")
	}
	const want = 690931.0
	if got := env.Minion.Output["PoisonDPS"]; math.Abs(got-want)/want > 0.001 {
		t.Errorf("minion PoisonDPS = %v, want %v (PoB)", got, want)
	}
}
