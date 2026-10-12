package calculator

import (
	"math"
	"os"
	"testing"
)

// TestEveryNUseCritRaisesCritChance covers CalcOffence.lua:3050-3059.
func TestEveryNUseCritRaisesCritChance(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, tc := range []struct {
		build string
		apply func(float64) float64
		lua   string
	}{
		{"0004-trickster-lightning-tendrils-of-eccentricity.xml", func(c float64) float64 { return (4*c + 100) / 5 }, "CalcOffence.lua:3055"},
		{"0208-hierophant-lightning-tendrils-of-eccentricity.xml", func(c float64) float64 { return (4*c + 100) / 5 }, "CalcOffence.lua:3055"},
		{"0193-occultist-lightning-tendrils.xml", func(c float64) float64 { return (2*c + 100) / 3 }, "CalcOffence.lua:3052"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok || rec.TreeSources < oracleMinTreeSources {
				t.Fatalf("REFUSING: %s is not graded; this fixture no longer exercises the mechanism", tc.build)
			}
			pobPre := rec.Output["PreBifurcateCritChance"]
			pobCrit := rec.Output["CritChance"]
			pobAcc := rec.Output["AccuracyHitChance"]
			if pobPre <= 0 {
				t.Fatalf("REFUSING: PoB reports PreBifurcateCritChance %v for %s", pobPre, tc.build)
			}
			if want := tc.apply(pobPre) * pobAcc / 100; math.Abs(pobCrit-want) > 1e-6 {
				t.Fatalf("REFUSING: PoB's own CritChance %v is not the cadence transform of its "+
					"PreBifurcateCritChance %v (%v); %s does not exercise %s",
					pobCrit, pobPre, want, tc.build, tc.lua)
			}
			if pobPre >= 100 {
				t.Fatalf("REFUSING: PoB's PreBifurcateCritChance is %v; the cadence step is a no-op here", pobPre)
			}

			env := corpusEnvForLoadedData(t, tc.build)
			pre := env.Player.Output["PreBifurcateCritChance"]
			crit := env.Player.Output["CritChance"]
			acc := env.Player.Output["AccuracyHitChance"]
			if pre <= 0 {
				t.Fatalf("PreBifurcateCritChance is %v; the crit chain is not running", pre)
			}
			want := tc.apply(pre) * acc / 100
			if math.Abs(crit-want) > 1e-9 {
				t.Errorf("CritChance = %v, want %v = cadence transform of PreBifurcateCritChance %v "+
					"then x AccuracyHitChance %v (%s)", crit, want, pre, acc, tc.lua)
			}
		})
	}
}
