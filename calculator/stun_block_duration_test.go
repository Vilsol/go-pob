package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestStunAndBlockDurationAreTickQuantised covers CalcDefence.lua:2165-2166.
func TestStunAndBlockDurationAreTickQuantised(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	keys := []string{"StunDuration", "BlockDuration"}
	seen := map[string]int{}
	agreed := map[string]int{}
	var builds, quantised int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		if v, has := rec.Output["BlockDuration"]; !has || v == 0 {
			continue
		}
		builds++
		env := corpusEnvForLoadedData(t, name)
		for _, key := range keys {
			want, has := rec.Output[key]
			if !has || want == 0 {
				continue
			}
			seen[key]++
			if oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
				agreed[key]++
			}
		}
		ticks := rec.Output["BlockDuration"] * data.ServerTickRate
		if math.Abs(ticks-math.Round(ticks)) < 1e-6 {
			quantised++
		}
		if builds >= 200 {
			break
		}
	}

	if builds == 0 {
		t.Fatal("REFUSING: no graded build publishes a nonzero BlockDuration; this test no longer " +
			"measures the mechanism it was written for")
	}
	t.Logf("StunDuration %d of %d · BlockDuration %d of %d · PoB values on a tick boundary: %d of %d",
		agreed["StunDuration"], seen["StunDuration"],
		agreed["BlockDuration"], seen["BlockDuration"], quantised, builds)

	if quantised*100 < builds*90 {
		t.Errorf("REFUSING: only %d of %d PoB BlockDuration values sit on a server-tick boundary; "+
			"the quantisation this test asserts is not what PoB is doing", quantised, builds)
	}
	floors := map[string]int{"StunDuration": 200, "BlockDuration": 200}
	for _, key := range keys {
		if agreed[key] < floors[key] {
			t.Errorf("%s agrees on %d of %d graded builds, below the floor of %d "+
				"(CalcDefence.lua:2165-2166)", key, agreed[key], seen[key], floors[key])
		}
	}
}
