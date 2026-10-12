package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestDotMitigationUsesResistOverTime pins CalcDefence.lua:1925.
//
//	local resist = modDB:Flag(nil, "SelfIgnore"..damageType.."Resistance") and 0
//	               or output[damageType.."ResistOverTime"]
//	               or output[damageType.."Resist"]
func TestDotMitigationUsesResistOverTime(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, names := corpusNamesForLoadedData(t)

	checked, wrong := 0, 0
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok {
			continue
		}
		want, ok := rec.Output["ChaosTakenDotMult"]
		if !ok || want == 0 {
			continue
		}
		file, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			continue
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		}()
		if env == nil {
			continue
		}
		checked++
		if got := env.Player.Output["ChaosTakenDotMult"]; math.Abs(got-want) > 0.005*math.Abs(want) {
			wrong++
			if wrong <= 3 {
				t.Logf("still wrong: %s: ChaosTakenDotMult = %v, want %v", name, got, want)
			}
		}
		if checked >= 150 {
			break
		}
	}
	if checked < 50 {
		t.Fatalf("REFUSING: only %d builds publish ChaosTakenDotMult; this test would pass "+
			"vacuously", checked)
	}
	const ceiling = 13
	if wrong > ceiling {
		t.Errorf("ChaosTakenDotMult wrong on %d of %d builds, ceiling %d — this is a DOWN-ONLY "+
			"counter and a rise means the DoT mitigation path regressed", wrong, checked, ceiling)
	}
}
