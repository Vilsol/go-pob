package calculator

import (
	"os"
	"testing"
)

// TestAilmentCritGateReadsTheActorModDB pins the STORE and the CFG of the ailment crit gate.
//
//	modDB:Flag(nil, "AilmentsAreNeverFromCrit")
func TestAilmentCritGateReadsTheActorModDB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	carriers := []string{
		"0006-elementalist-storm-burst.xml",
		"0007-elementalist-storm-burst-of-repulsion.xml",
		"0010-elementalist-storm-burst-2.xml",
		"0011-saboteur-cyclone.xml",
	}

	set := 0
	for _, build := range carriers {
		env := corpusEnvForLoadedData(t, build)
		if env.Player.ModDB.Flag(nil, "AilmentsAreNeverFromCrit") {
			set++
		} else {
			t.Errorf("%s: PoB's dump carries AilmentsAreNeverFromCrit and our actor modDB does "+
				"not have it. The three ailment crit gates read this store; if it is empty they "+
				"are gating on nothing.", build)
		}
	}

	if set == 0 {
		t.Fatal("REFUSING: not one carrier has the flag in our actor modDB — the population " +
			"emptied, so this test would otherwise report a correct port and a broken one " +
			"identically")
	}
}
