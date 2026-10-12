package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestHolyRelicNovaPublishesMinionTriggerRates pins CalcPerform.lua:3736's `calcs.triggers(env, env.minion)` and the ["nova"] configTable entry.
func TestHolyRelicNovaPublishesMinionTriggerRates(t *testing.T) {
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

	const tol = 0.005
	keys := []string{"TriggerRateCap", "EffectiveSourceRate", "SkillTriggerRate", "TriggerTime"}

	checked, agreed := 0, 0
	distinctRate := map[float64]bool{}
	var diverged []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Minion == nil || rec.Minion["TriggerRateCap"] == 0 {
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
		if env == nil || env.Minion == nil || env.Minion.Actor == nil {
			continue
		}

		checked++
		distinctRate[math.Round(rec.Minion["SkillTriggerRate"]*1e6)/1e6] = true

		agrees := true
		for _, key := range keys {
			exp := rec.Minion[key]
			if exp == 0 {
				continue
			}
			got := env.Minion.Output[key]
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("%s: Minion.%s is %v -- a NaN/Inf compares equal to nothing and cannot be graded", name, key, got)
			}
			if math.Abs(got-exp) > tol*math.Abs(exp) {
				agrees = false
			}
		}
		if agrees {
			agreed++
		} else {
			diverged = append(diverged, name)
		}
	}

	if checked < 100 {
		t.Fatalf("REFUSING: only %d minion-trigger carriers found; the fixture population has moved", checked)
	}
	if len(distinctRate) < 20 {
		t.Fatalf("REFUSING: only %d distinct SkillTriggerRate values across %d carriers; the sweep no longer exercises the rotation simulation", len(distinctRate), checked)
	}

	const floor = 116
	if agreed < floor {
		shown := diverged
		if len(shown) > 5 {
			shown = shown[:5]
		}
		t.Errorf("agreement REGRESSED: %d of %d carriers agree on all of %v, floor is %d; %d diverged, first: %v",
			agreed, checked, keys, floor, len(diverged), shown)
	}
}
