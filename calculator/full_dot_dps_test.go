package calculator

import (
	"fmt"
	"os"
	"testing"
)

const (
	fullDotDPSAgreeFloor       = 96
	fullDotDPSZeroControlFloor = 8242
)

// TestFullDotDPSAgrees pins Calcs.lua:430 — `env.player.output.FullDotDPS = fullDPS.TotalDotDPS`.
func TestFullDotDPSAgrees(t *testing.T) {
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

	var nonzeroSeen, nonzeroOK, zeroSeen, zeroOK int
	var controlLost []string
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		want, has := rec.Output["FullDotDPS"]
		if !has {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		agrees := oracleClose(env.Player.Output["FullDotDPS"], want, oracleDumpTolerance)
		if want != 0 {
			nonzeroSeen++
			if agrees {
				nonzeroOK++
			}
		} else {
			zeroSeen++
			if agrees {
				zeroOK++
			} else if len(controlLost) < 8 {
				controlLost = append(controlLost, fmt.Sprintf("%s (ours %v, PoB 0)",
					name, env.Player.Output["FullDotDPS"]))
			}
		}
	}

	fmt.Printf("=== FullDotDPS === nonzero %d/%d agree, PoB-zero control %d/%d agree\n",
		nonzeroOK, nonzeroSeen, zeroOK, zeroSeen)

	if nonzeroSeen == 0 || zeroSeen == 0 {
		t.Fatalf("REFUSING: nonzero population %d, PoB-zero control %d", nonzeroSeen, zeroSeen)
	}
	if nonzeroOK < fullDotDPSAgreeFloor {
		t.Errorf("FullDotDPS agrees on %d of %d nonzero builds, floor %d",
			nonzeroOK, nonzeroSeen, fullDotDPSAgreeFloor)
	}
	if zeroOK < fullDotDPSZeroControlFloor {
		t.Errorf("the PoB-zero control REGRESSED: %d of %d agree, floor %d — a nonzero gain bought "+
			"out of builds that were already right. Known losers: %v",
			zeroOK, zeroSeen, fullDotDPSZeroControlFloor, controlLost)
	}
}
