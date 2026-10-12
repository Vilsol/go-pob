package calculator

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

var lifeLossOverTimeFloors = map[string]int{
	"LifeBelowHalfLossLostOverTime": 456,
	"LifeLossLostAvg":               337,
	"LifeLossLostMax":               438,
}

// TestLifeLossOverTimeTrackersAgree pins the three sites that carry prevented life loss OUT of the EHP walk and into the outputs that read it.
func TestLifeLossOverTimeTrackersAgree(t *testing.T) {
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

	seen := map[string]int{}
	agree := map[string]int{}
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		if env.Player.Output["preventedLifeLossTotal"] <= 0 {
			continue
		}
		for key := range lifeLossOverTimeFloors {
			want, has := rec.Output[key]
			if !has {
				continue
			}
			seen[key]++
			if oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
				agree[key]++
			}
		}
	}

	keys := make([]string, 0, len(lifeLossOverTimeFloors))
	for k := range lifeLossOverTimeFloors {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if seen[k] == 0 {
			t.Fatalf("REFUSING: the oracle carries %s on 0 of the builds with "+
				"preventedLifeLossTotal > 0 — a floor over an empty set asserts nothing", k)
		}
		fmt.Printf("=== LIFE LOSS OVER TIME === %-30s %d / %d agree\n", k, agree[k], seen[k])
	}
	for _, k := range keys {
		if agree[k] < lifeLossOverTimeFloors[k] {
			t.Errorf("%s agrees on %d of %d petrified builds, floor %d", k, agree[k], seen[k],
				lifeLossOverTimeFloors[k])
		}
	}
}
