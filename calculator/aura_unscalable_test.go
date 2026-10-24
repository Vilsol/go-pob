package calculator

import (
	"os"
	"testing"
)

// TestUnscalableAuraModsReachTheirTarget pins the second half of the ailment-avoidance defect.
func TestUnscalableAuraModsReachTheirTarget(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	requireGameData(t, "3.29")

	const name = "5830-ladder-Qqz1_1016.TwinRf.xml"
	want, got := ailmentAvoidance(t, name)
	assertAvoidChances(t, name, []string{
		"IgniteAvoidChance", "ChillAvoidChance", "FreezeAvoidChance", "ShockAvoidChance",
		"ScorchAvoidChance", "BrittleAvoidChance", "SapAvoidChance",
	}, want, got)
}
