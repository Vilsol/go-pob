package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestMoltenStrikeOverlapChance pins act_str.lua:7776-7818.
func TestMoltenStrikeOverlapChance(t *testing.T) {
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

	carriers, agreed, written := 0, 0, 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output["OverlapChance"] == 0 {
			continue
		}
		if rec.Output["AreaOfEffectRadiusTertiary"] == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
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
		got := env.Player.Output["OverlapChance"]
		if math.IsNaN(got) || math.IsInf(got, 0) {
			t.Fatalf("%s: OverlapChance is %v -- it cannot be graded and its ceiling only moves down", name, got)
		}
		if got != 0 {
			written++
		}
		if math.Abs(got-rec.Output["OverlapChance"]) <= 0.005*math.Abs(rec.Output["OverlapChance"]) {
			agreed++
		}
	}

	if carriers < 10 {
		t.Fatalf("REFUSING: only %d act_str OverlapChance carriers; the population has moved", carriers)
	}
	const writtenFloor, agreeFloor = 17, 11
	if written < writtenFloor {
		t.Errorf("OverlapChance written on only %d of %d carriers, floor %d -- the block is not running", written, carriers, writtenFloor)
	}
	if agreed < agreeFloor {
		t.Errorf("OverlapChance agreement REGRESSED to %d of %d, floor %d", agreed, carriers, agreeFloor)
	}
	t.Logf("act_str carriers %d, written %d, agreed %d", carriers, written, agreed)
}
