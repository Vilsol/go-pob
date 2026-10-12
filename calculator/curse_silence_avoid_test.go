package calculator

import (
	"os"
	"testing"
)

// TestCurseAndSilenceAvoidChanceArePublished pins CalcDefence.lua:1588-1589, two keys that had NO PRODUCER anywhere in this tree.
//
//	output.CurseAvoidChance = modDB:Flag(nil, "CurseImmune") and 100
//	    or m_min(modDB:Sum("BASE", nil, "AvoidCurse"), 100)
//	output.SilenceAvoidChance = modDB:Flag(nil, "SilenceImmune") and 100 or output.CurseAvoidChance
func TestCurseAndSilenceAvoidChanceArePublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var builds, missingKey, mirrorSeen, mirrorWrong int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		if _, has := rec.Output["CurseAvoidChance"]; !has {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		builds++

		for _, key := range []string{"CurseAvoidChance", "SilenceAvoidChance"} {
			if _, published := env.Player.Output[key]; !published {
				missingKey++
			}
		}
		if !env.Player.ModDB.Flag(nil, "SilenceImmune") {
			mirrorSeen++
			if env.Player.Output["SilenceAvoidChance"] != env.Player.Output["CurseAvoidChance"] {
				mirrorWrong++
			}
		}
		if builds >= 200 {
			break
		}
	}

	if builds == 0 {
		t.Fatal("REFUSING: the blob publishes CurseAvoidChance on no graded build, so this test " +
			"measured nothing")
	}
	if mirrorSeen == 0 {
		t.Fatal("REFUSING: every graded build carries SilenceImmune, so the :1589 mirror is never " +
			"exercised and this test no longer measures it")
	}
	if missingKey > 0 {
		t.Errorf("%d of %d key writes are absent — the producer is starved again", missingKey, builds*2)
	}
	if mirrorWrong > 0 {
		t.Errorf("SilenceAvoidChance does not mirror CurseAvoidChance on %d of %d builds without "+
			"SilenceImmune", mirrorWrong, mirrorSeen)
	}
}
