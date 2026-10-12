package calculator

import (
	"os"
	"testing"
)

// TestSpectreAuraReachesMinionConditions pins CalcPerform.lua:2818-2831.
func TestSpectreAuraReachesMinionConditions(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0031-guardian-raise-spectre.xml")
	if env.Minion == nil {
		t.Fatal("0031 has no minion")
	}
	for _, cond := range []string{"AffectedByWrath", "AffectedByHatred", "AffectedByAnger", "AffectedByAura"} {
		if !env.Minion.ModDB.Conditions[cond] {
			t.Errorf("minion condition %s is false; PoB's dump has it true", cond)
		}
	}
}
