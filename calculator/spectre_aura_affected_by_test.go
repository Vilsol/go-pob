package calculator

import (
	"math"
	"os"
	"testing"
)

// TestSpectreAuraSetsAffectedBy pins CalcPerform.lua:2800-2810.
func TestSpectreAuraSetsAffectedBy(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	for _, b := range []string{"0066-hierophant-forbidden-rite-of-soul-sacrifice.xml", "1062-hierophant-forbidden-rite-of-soul-sacrifice-3.xml"} {
		env := corpusEnvForLoadedData(t, b)
		for _, c := range []string{"AffectedByZealotry", "AffectedByDiscipline", "AffectedByAura"} {
			if !env.ModDB.Conditions[c] {
				t.Errorf("%s: condition %s unset", b, c)
			}
		}
		for key, want := range map[string]float64{"Speed": 10.832, "EnergyShieldRegen": 2667.3} {
			if got := env.Player.Output[key]; math.Abs(got-want) > want*1e-4 {
				t.Errorf("%s %s = %v, PoB has %v", b, key, got, want)
			}
		}
	}
}
