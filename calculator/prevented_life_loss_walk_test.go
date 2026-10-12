package calculator

import (
	"fmt"
	"os"
	"testing"
)

const (
	preventedWalkAgreeFloor   = 500
	unpreventedWalkAgreeFloor = 6202
)

// TestPreventedLifeLossWalkAgrees pins CalcDefence.lua:348-392, the prevented-life-loss arm of calcs.reducePoolsByDamage, as it is threaded through the EHP walk.
func TestPreventedLifeLossWalkAgrees(t *testing.T) {
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

	var preventedSeen, preventedOK, controlSeen, controlOK int
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		want, has := rec.Output["NumberOfDamagingHits"]
		if !has {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		agrees := oracleClose(env.Player.Output["NumberOfDamagingHits"], want, oracleDumpTolerance)
		if env.Player.Output["preventedLifeLossTotal"] > 0 {
			preventedSeen++
			if agrees {
				preventedOK++
			}
		} else {
			controlSeen++
			if agrees {
				controlOK++
			}
		}
	}

	if preventedSeen == 0 || controlSeen == 0 {
		t.Fatalf("REFUSING: prevented population %d, control population %d — a floor over an empty "+
			"set asserts nothing", preventedSeen, controlSeen)
	}
	fmt.Printf("=== PREVENTED LIFE LOSS WALK === petrified %d/%d agree · control %d/%d agree\n",
		preventedOK, preventedSeen, controlOK, controlSeen)

	if preventedOK < preventedWalkAgreeFloor {
		t.Errorf("NumberOfDamagingHits agrees on %d of %d builds with preventedLifeLossTotal > 0, "+
			"floor %d — the CalcDefence.lua:348-392 drain regressed", preventedOK, preventedSeen,
			preventedWalkAgreeFloor)
	}
	if controlOK < unpreventedWalkAgreeFloor {
		t.Errorf("NumberOfDamagingHits agrees on %d of %d builds with NO prevented life loss, floor "+
			"%d — the petrified branch was paid for out of the builds it does not apply to",
			controlOK, controlSeen, unpreventedWalkAgreeFloor)
	}
}
