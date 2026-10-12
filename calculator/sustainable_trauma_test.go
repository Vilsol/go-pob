package calculator

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

// TestSustainableTraumaAgainstOracle grades the closed-form trauma solver.
func TestSustainableTraumaAgainstOracle(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}

	carriers := make([]string, 0, 128)
	for name, rec := range oracle {
		if rec.TreeSources < oracleMinTreeSources {
			continue
		}
		if _, ok := rec.Output["SustainableTrauma"]; ok {
			carriers = append(carriers, name)
		}
	}
	if len(carriers) == 0 {
		t.Fatalf("REFUSING: the oracle reports SustainableTrauma on 0 gradeable builds — this " +
			"fixture has no population and a green result would mean nothing was checked")
	}
	sort.Strings(carriers)

	var agree, disagree int
	var firstMismatch string
	var ratios []string
	for _, name := range carriers {
		want := oracle[name].Output["SustainableTrauma"]
		env := corpusEnvForLoadedData(t, name)
		got := env.Player.Output["SustainableTrauma"]
		if oracleClose(got, want, oracleDumpTolerance) {
			agree++
			continue
		}
		disagree++
		if got != 0 {
			rung := ""
			for _, k := range []string{"HitChance", "Speed", "Repeats", "ActionSpeedMod", "Duration"} {
				ours := env.Player.Output[k]
				pob := oracle[name].Output[k]
				if !oracleClose(ours, pob, oracleDumpTolerance) {
					rung += fmt.Sprintf(" %s(%.3f vs %.3f)", k, ours, pob)
				}
			}
			if rung == "" {
				rung = " [all rungs agree]"
			}
			ratios = append(ratios, fmt.Sprintf("%.4f  %s%s", want/got, name, rung))
		} else {
			ratios = append(ratios, fmt.Sprintf("ZERO    %s (PoB %.3f)", name, want))
		}
		if firstMismatch == "" {
			firstMismatch = fmt.Sprintf("%s: got %.4f, PoB %.4f", name, got, want)
		}
	}

	fmt.Printf("\nSustainableTrauma: %d/%d builds agree with the pinned PoB\n", agree, len(carriers))
	sort.Strings(ratios)
	for _, r := range ratios {
		fmt.Println("  " + r)
	}
	const agreementFloor = 61
	if agree < agreementFloor {
		t.Errorf("SustainableTrauma agreement REGRESSED to %d of %d (floor %d); first mismatch: %s",
			agree, len(carriers), agreementFloor, firstMismatch)
	}
}
