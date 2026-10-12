package calculator

import (
	"os"
	"testing"
)

// TestVaalStoredUsesFeedsStoredUses: `StoredUses` is STARVED, not mis-factored, and that is what names its cause.
//
//	if level.storedUses then     activeSkill.skillData.storedUses = level.storedUses end
//	if level.vaalStoredUses then activeSkill.skillData.storedUses =
func TestVaalStoredUsesFeedsStoredUses(t *testing.T) {
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

	graded, starved, wrongNonzero := 0, 0, 0
	for _, b := range names {
		rec, ok := oracle[b]
		if !ok {
			continue
		}
		want, has := rec.Output["StoredUses"]
		if !has || want == 0 {
			continue
		}
		env := corpusEnvForLoadedData(t, b)
		if env == nil || env.Player == nil {
			continue
		}
		graded++
		got := env.Player.Output["StoredUses"]
		if oracleClose(got, want, oracleDumpTolerance) {
			continue
		}
		if got == 0 {
			starved++
		} else {
			wrongNonzero++
		}
	}

	if graded == 0 {
		t.Fatal("REFUSING: no build carries a nonzero StoredUses in the oracle, so this sweep is " +
			"not reaching the stored-uses block at all")
	}

	t.Logf("StoredUses: %d graded, %d starved to exactly 0, %d wrong-nonzero", graded, starved, wrongNonzero)

	const starvedCeiling = 2
	if starved > starvedCeiling {
		t.Errorf("StoredUses is starved to exactly 0 on %d builds (ceiling %d) — the vaal arm of "+
			"the granted-effect level loop is the only producer for these", starved, starvedCeiling)
	}

	const wrongCeiling = 5
	if wrongNonzero > wrongCeiling {
		t.Errorf("StoredUses is wrong-but-nonzero on %d builds (ceiling %d) — that is a wrong "+
			"factor, not a starved producer", wrongNonzero, wrongCeiling)
	}
}
