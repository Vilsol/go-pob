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
	requireGameData(t, "3.20")

	const name = "5941-champion-summon-skeletons-9.xml"
	want, got := ailmentAvoidance(t, name)
	assertAvoidChances(t, name, []string{
		"IgniteAvoidChance", "ChillAvoidChance", "FreezeAvoidChance", "ShockAvoidChance",
		"ScorchAvoidChance", "BrittleAvoidChance", "SapAvoidChance",
	}, want, got)
}
