package calculator

import (
	"math"
	"os"
	"testing"
)

// TestGhostReaverLeechUsesItsOwnAccumulator pins CalcOffence.lua:3300 / :3594-3598 / :3646-3650.
//
//	:3300       local ghostReaverLifeLeech = 0
//	:3594-3598  if ghostReaver then ghostReaverLifeLeech = ghostReaverLifeLeech + lifeLeechTotal * portion
//	:3646-3650  if ghostReaver and ghostReaverLifeLeech > 0 then
func TestGhostReaverLeechUsesItsOwnAccumulator(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"0060-assassin-icicle-mine.xml",
		"0062-occultist-crackling-lance.xml",
		"0066-hierophant-forbidden-rite-of-soul-sacrifice.xml",
	}

	graded := 0
	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		if rec.Output["LifeLeechInstantProportion"] == 0 {
			t.Fatalf("REFUSING: %s has LifeLeechInstantProportion 0, so both orderings agree "+
				"on it", build)
		}
		want := rec.Output["EnergyShieldLeechInstant"]
		if want == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero EnergyShieldLeechInstant for %s", build)
		}
		graded++

		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			got := env.Player.Output["EnergyShieldLeechInstant"]
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: EnergyShieldLeechInstant = %v, PoB reports %v (ratio %.4f)",
					build, got, want, got/want)
			}
		})
	}

	if graded == 0 {
		t.Fatal("REFUSING: no fixture graded the ghost-reaver leech split — the population " +
			"emptied rather than this test passing")
	}
}
