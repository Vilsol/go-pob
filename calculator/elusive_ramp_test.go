package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestElusiveRampMatchesOracle pins CalcPerform.lua:787-800 — the elusive RAMP.
func TestElusiveRampMatchesOracle(t *testing.T) {
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

	const stat = "ElusiveEffectMod"
	const tol = 0.005
	const rampPhrase = "Effect on you is increased instead for the first"

	rampSeen, rampOK, plainSeen, plainOK := 0, 0, 0, 0
	var diverged []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output[stat] == 0 {
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
		exp := rec.Output[stat]
		got := env.Player.Output[stat]
		if math.IsNaN(got) || math.IsInf(got, 0) {
			t.Fatalf("%s: %s is %v -- a NaN/Inf compares equal to nothing and cannot be graded", name, stat, got)
		}
		agrees := math.Abs(got-exp) <= tol*math.Abs(exp)
		if strings.Contains(string(raw), rampPhrase) {
			rampSeen++
			if agrees {
				rampOK++
			} else {
				diverged = append(diverged, name)
			}
		} else {
			plainSeen++
			if agrees {
				plainOK++
			}
		}
	}

	if rampSeen < 30 || plainSeen < 30 {
		t.Fatalf("REFUSING: ramp carriers %d, non-carriers %d; the fixture population has moved", rampSeen, plainSeen)
	}

	const rampFloor = 36
	const plainFloor = 79
	if rampOK < rampFloor {
		shown := diverged
		if len(shown) > 5 {
			shown = shown[:5]
		}
		t.Errorf(":797 REGRESSED: %d of %d ramp carriers agree, floor is %d; first diverged: %v", rampOK, rampSeen, rampFloor, shown)
	}
	if plainOK < plainFloor {
		t.Errorf(":799 REGRESSED: %d of %d non-carriers agree, floor is %d", plainOK, plainSeen, plainFloor)
	}
	t.Logf("ramp %d/%d, plain %d/%d", rampOK, rampSeen, plainOK, plainSeen)
}
