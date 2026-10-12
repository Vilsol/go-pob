package calculator

import (
	"os"
	"testing"
)

// TestMinionFlaggedMainSkillStillDispatchesTriggers pins CalcTriggers.lua:1593's guard.
func TestMinionFlaggedMainSkillStillDispatchesTriggers(t *testing.T) {
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

	carriers, published, agreed := 0, 0, 0
	for _, b := range names {
		rec, ok := oracle[b]
		if !ok {
			continue
		}
		want, has := rec.Output["TriggerRateCap"]
		if !has || want == 0 {
			continue
		}
		env := corpusEnvForLoadedData(t, b)
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			continue
		}
		if !env.Player.MainSkill.SkillFlags[SkillFlagMinion] {
			continue
		}
		carriers++
		got := env.Player.Output["TriggerRateCap"]
		if got != 0 {
			published++
		}
		if oracleClose(got, want, oracleDumpTolerance) {
			agreed++
		}
	}

	if carriers == 0 {
		t.Fatalf("REFUSING: no corpus build has a minion-flagged main skill that PoB gives a TriggerRateCap; this test cannot see the guard it exists for")
	}
	if published == 0 {
		t.Errorf("the minion conjunct is back: %d minion-flagged main skills PoB publishes TriggerRateCap for, and we publish it for 0", carriers)
	}
	t.Logf("minion-flagged main skills with a PoB TriggerRateCap: %d, we publish %d, agree %d", carriers, published, agreed)
}
