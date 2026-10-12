package calculator

import (
	"math"
	"os"
	"testing"
)

// TestCritRollsCountsExtremeLuck covers CalcOffence.lua:3027-3042.
func TestCritRollsCountsExtremeLuck(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range []string{
		"1114-gladiator-tempest-shield.xml",
		"2155-ladder-Hikkaru01_7581.NieWspieram.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok || rec.TreeSources < oracleMinTreeSources {
				t.Fatalf("REFUSING: %s is not graded", build)
			}
			pre := rec.Output["PreEffectiveCritChance"]
			post := rec.Output["PreBifurcateCritChance"]
			if pre <= 0 {
				t.Fatalf("REFUSING: PoB reports PreEffectiveCritChance %v for %s", pre, build)
			}
			p := pre / 100
			twoRolls := (1 - math.Pow(1-p, 3)) * 100
			oneRoll := (1 - math.Pow(1-p, 2)) * 100
			if math.Abs(post-twoRolls) > 1e-6 {
				t.Fatalf("REFUSING: PoB's PreBifurcateCritChance %v is not the two-roll value %v "+
					"for %s; this fixture does not exercise ExtremeLuck", post, twoRolls, build)
			}
			if math.Abs(twoRolls-oneRoll) < 1e-6 {
				t.Fatalf("REFUSING: one roll and two rolls agree to %v on %s; the doubling is "+
					"unobservable there", twoRolls, build)
			}

			env := corpusEnvForLoadedData(t, build)
			ourPre := env.Player.Output["PreEffectiveCritChance"]
			got := env.Player.Output["PreBifurcateCritChance"]
			q := ourPre / 100
			want := (1 - math.Pow(1-q, 3)) * 100
			if math.Abs(got-want) > 1e-9 {
				t.Errorf("PreBifurcateCritChance = %v, want %v = 1-(1-p)^3 of our own "+
					"PreEffectiveCritChance %v (CalcOffence.lua:3040 with critRolls=2)",
					got, want, ourPre)
			}
		})
	}
}
