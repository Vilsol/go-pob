package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestTempestShieldPublishesTriggerRates pins the one configTable entry calc_triggers.go ports.
func TestTempestShieldPublishesTriggerRates(t *testing.T) {
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

	const stat = "TriggerRateCap"
	const tol = 0.005

	checked, agreed, distinct := 0, 0, map[float64]bool{}
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
		if !strings.Contains(string(raw), "Tempest Shield") {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			continue
		}
		ae := env.Player.MainSkill.ActiveEffect
		if ae == nil || ae.GrantedEffect == nil || ae.GrantedEffect.Raw == nil {
			continue
		}
		as := ae.GrantedEffect.Raw.GetActiveSkill()
		if as == nil || as.DisplayedName != "Tempest Shield" {
			continue
		}

		checked++
		want := rec.Output[stat]
		distinct[math.Round(want*1e6)/1e6] = true

		agrees := true
		for _, key := range []string{"TriggerRateCap", "EffectiveSourceRate", "SkillTriggerRate"} {
			got := env.Player.Output[key]
			exp := rec.Output[key]
			if exp == 0 {
				continue
			}
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("%s: %s is %v -- a NaN/Inf compares equal to nothing and cannot be graded", name, key, got)
			}
			if math.Abs(got-exp) > tol*math.Abs(exp) {
				agrees = false
				if got > exp {
					t.Errorf("%s: %s = %v EXCEEDS PoB %v -- the remainder below is a LOW-side story and this is not it", name, key, got, exp)
				}
			}
		}
		if agrees {
			agreed++
		} else {
			diverged = append(diverged, name)
		}
	}

	if checked < 50 {
		t.Fatalf("REFUSING: only %d Tempest Shield carriers found; the fixture population has moved", checked)
	}
	if len(distinct) < 2 {
		t.Fatalf("REFUSING: every carrier published the same cap (%d distinct); the sweep no longer exercises the cooldown expression", len(distinct))
	}

	const floor = 64
	if agreed < floor {
		t.Errorf("agreement REGRESSED: %d of %d carriers agree, floor is %d", agreed, checked, floor)
	}

	if len(diverged) > 0 {
		t.Logf("known remainder (%d, all ours-LOW, conditional cooldown recovery): %v", len(diverged), diverged)
	}
	t.Logf("checked %d Tempest Shield carriers, %d distinct published caps", checked, len(distinct))
}
