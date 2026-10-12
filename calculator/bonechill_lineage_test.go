package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestBonechillCarriersAgreeOnCurrentChill pins CalcPerform.lua:3444 and :3540 against the v2.18.1 Bonechill lineage this tree carried.
func TestBonechillCarriersAgreeOnCurrentChill(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	const agreeFloor = 310

	carriers, agreed := 0, 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok {
			continue
		}
		want, published := rec.Output["CurrentChill"]
		if !published || want == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), "Bonechill") {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil {
			continue
		}
		carriers++
		got := env.Player.Output["CurrentChill"]
		if math.Abs(got-want) <= math.Abs(want)*oracleDumpTolerance {
			agreed++
		}
	}

	if carriers < 100 {
		t.Fatalf("REFUSING: only %d Bonechill carriers publish CurrentChill; the population has moved", carriers)
	}
	if agreed < agreeFloor {
		t.Errorf("CurrentChill agreement on Bonechill carriers is %d of %d, floor %d", agreed, carriers, agreeFloor)
	}
	t.Logf("CurrentChill on Bonechill carriers: agreed %d of %d", agreed, carriers)
}
