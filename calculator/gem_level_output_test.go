package calculator

import (
	"os"
	"testing"
)

// TestGemLevelSumsItemAndSupportLevels pins CalcActiveSkill.lua:542-564 and CalcPerform.lua:3870-3875.
func TestGemLevelSumsItemAndSupportLevels(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	build := loadCorpusBuild(t, "4623-ladder-Englian_6826.End_of_Luminary.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

	for _, tc := range []struct {
		key  string
		want float64
	}{
		{"GemLevel", 41},
		{"GemQuality", 48},
	} {
		if got := env.Player.Output[tc.key]; got != tc.want {
			t.Errorf("%s = %v, want %v (PoB's value)", tc.key, got, tc.want)
		}
	}
}
