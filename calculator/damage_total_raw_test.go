package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestDamageTotalSumsTheRawHit is the red-green for CalcDefence.lua:2643-2645.
//
//	local damage = DamageIn[damageType] or 0
func TestDamageTotalSumsTheRawHit(t *testing.T) {
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

	var shortSeen, shortOK, allSeen, allOK, fractional int
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		want, has := rec.Output["NumberOfDamagingHits"]
		if !has {
			continue
		}
		for _, dt := range data.DamageType("").Values() {
			if v := rec.Output[string(dt)+"TakenHit"]; v > 0 && v != math.Trunc(v) {
				fractional++
				break
			}
		}
		env := corpusEnvForLoadedData(t, name)
		agrees := oracleClose(env.Player.Output["NumberOfDamagingHits"], want, oracleDumpTolerance)
		allSeen++
		if agrees {
			allOK++
		}
		if want <= 1 {
			shortSeen++
			if agrees {
				shortOK++
			}
		}
	}

	t.Logf("=== damageTotal === all: %d of %d agree | single-pass slice: %d of %d | "+
		"builds with fractional incoming damage: %d of %d",
		allOK, allSeen, shortOK, shortSeen, fractional, allSeen)

	if shortSeen == 0 {
		t.Fatal("REFUSING: the single-pass slice is empty, so this fixture cannot reach the " +
			"overkill correction it grades")
	}
	if fractional != allSeen {
		t.Errorf("%d of %d builds carry fractional incoming damage — an integral-damage control "+
			"is now REACHABLE and this fixture should use it as its arithmetic-identity control",
			fractional, allSeen)
	}
	const shortFloor = 840
	if shortOK < shortFloor {
		t.Errorf("single-pass agreement %d of %d, want >= %d — damageTotal is summing the "+
			"ceiled/multiplied hit again (CalcDefence.lua:2645 sums the raw one)",
			shortOK, shortSeen, shortFloor)
	}
	const allFloor = 7940
	if allOK < allFloor {
		t.Errorf("corpus-wide NumberOfDamagingHits agreement %d of %d, want >= %d",
			allOK, allSeen, allFloor)
	}
}
