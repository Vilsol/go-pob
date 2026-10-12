package calculator

import (
	"os"
	"testing"
)

// TestEffectiveSourceRateIsStarvedNotMisFactored: EffectiveSourceRate is 3.29's name for the SOURCE skill's own rate.
func TestEffectiveSourceRateIsStarvedNotMisFactored(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	_, names := corpusNamesForLoadedData(t)

	var graded, agree, starved, misFactored, capExplains int
	for _, b := range names {
		rec, ok := oracle[b]
		if !ok {
			continue
		}
		want, has := rec.Output["EffectiveSourceRate"]
		if !has || want == 0 {
			continue
		}
		env := corpusEnvForLoadedData(t, b)
		if env == nil || env.Player == nil {
			continue
		}
		graded++
		got := env.Player.Output["EffectiveSourceRate"]
		if oracleClose(got, want, oracleDumpTolerance) {
			agree++
			continue
		}
		if got == 0 {
			starved++
		} else {
			misFactored++
		}
		if oracleClose(env.Player.Output["TriggerRateCap"], want, oracleDumpTolerance) {
			capExplains++
		}
	}

	if graded == 0 {
		t.Fatal("REFUSING: no build has a nonzero EffectiveSourceRate in the oracle, so this " +
			"sweep is not reaching the trigger calculation and its counts mean nothing")
	}
	if agree == 0 {
		t.Fatal("REFUSING: EffectiveSourceRate agrees on NO build. The three ported trigger " +
			"blocks should still land it on the builds they cover, so this reads as the key " +
			"having stopped being published at all rather than as a worse port")
	}

	t.Logf("EffectiveSourceRate: %d graded, %d agree, %d starved (ours exactly 0), %d mis-factored; "+
		"TriggerRateCap explains %d", graded, agree, starved, misFactored, capExplains)

	if wrong := starved + misFactored; wrong > 0 && starved*2 < wrong {
		t.Fatalf("EffectiveSourceRate's defect shape has INVERTED: %d of %d disagreements are now "+
			"mis-factored rather than starved (was 37 of 774 on 2026-09-02). Either the "+
			"CalcTriggers.lua port landed — re-derive this test against it — or a nonzero value "+
			"is now being published on builds that previously wrote none",
			misFactored, wrong)
	}

	const agreeFloor = 534
	if agree < agreeFloor {
		t.Errorf("EffectiveSourceRate agreement REGRESSED to %d of %d builds, floor is %d -- a "+
			"configTable entry has stopped dispatching", agree, graded, agreeFloor)
	}
}
