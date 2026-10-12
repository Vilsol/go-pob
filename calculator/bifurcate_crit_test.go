package calculator

import (
	"math"
	"os"
	"testing"
)

// TestPreBifurcateCritChanceMatchesPoB covers CalcOffence.lua:3043-3047 and :3136-3170.
func TestPreBifurcateCritChanceMatchesPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var seen, agreed, bifurcating int
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		want, ok := rec.Output["PreBifurcateCritChance"]
		if !ok || want == 0 {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		got := env.Player.Output["PreBifurcateCritChance"]
		seen++
		if oracleClose(got, want, oracleDumpTolerance) {
			agreed++
		}
		if env.Player.MainSkill != nil && env.Player.MainSkill.SkillModList.Flag(nil, "BifurcateCrit") {
			bifurcating++
		}
		if seen >= 150 {
			break
		}
	}

	if seen == 0 {
		t.Fatal("REFUSING: no graded build reports a nonzero PreBifurcateCritChance; this test no " +
			"longer exercises the mechanism it was written for")
	}
	t.Logf("PreBifurcateCritChance: %d of %d sampled builds agree; %d of them carry BifurcateCrit",
		agreed, seen, bifurcating)

	const agreementFloor = 44
	if agreed < agreementFloor {
		t.Errorf("PreBifurcateCritChance agrees with PoB on %d of %d sampled builds, below the "+
			"floor of %d measured when the bifurcation landed", agreed, seen, agreementFloor)
	}

	for _, build := range []string{
		"0020-assassin-storm-burst.xml",
		"0060-assassin-icicle-mine.xml",
		"0105-assassin-spellslinger.xml",
		"0122-assassin-bladefall.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no oracle record", build)
			}
			if cb := rec.Output["CritBifurcates"]; cb <= 1 {
				t.Fatalf("REFUSING: PoB reports CritBifurcates %v for %s, so it does not bifurcate "+
					"and cannot exercise the transform this fixture was chosen for", cb, build)
			}

			env := corpusEnvForLoadedData(t, build)
			pre := env.Player.Output["PreBifurcateCritChance"]
			crit := env.Player.Output["CritChance"]
			if pre <= 0 {
				t.Fatalf("PreBifurcateCritChance is %v; the output is not being written", pre)
			}
			want := (1 - math.Pow(1-pre/100, 2)) * 100
			if math.Abs(crit-want) > 1e-9 {
				t.Errorf("CritChance = %v, want %v = bifurcation of PreBifurcateCritChance %v "+
					"(CalcOffence.lua:3046)", crit, want, pre)
			}
			if cb := env.Player.Output["CritBifurcates"]; cb <= 1 {
				t.Errorf("CritBifurcates = %v, want > 1 on a bifurcating build "+
					"(CalcOffence.lua:3152)", cb)
			}
		})
	}
}
