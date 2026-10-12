package calculator

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestTriggerDispatchAgreesWithOracle sweeps every corpus build whose MAIN SKILL our trigger dispatcher answers for.
func TestTriggerDispatchAgreesWithOracle(t *testing.T) {
	t.Parallel()

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

	type tally struct {
		agreed, total, high int
		diverged            []string
	}
	byEntry := map[string]*tally{}

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok {
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
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			continue
		}
		if env.Player.Output["TriggerRateCap"] == 0 {
			continue
		}

		key := triggerDispatchKeyForTest(env, env.Player.MainSkill)
		if byEntry[key] == nil {
			byEntry[key] = &tally{}
		}
		byEntry[key].total++

		agrees, high := true, false
		for _, stat := range []string{"TriggerRateCap", "EffectiveSourceRate", "SkillTriggerRate"} {
			got := env.Player.Output[stat]
			exp := rec.Output[stat]
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("%s: %s is %v -- a NaN/Inf cannot be graded and its ceiling only moves down", name, stat, got)
			}
			if exp == 0 {
				if got != 0 {
					agrees = false
				}
				continue
			}
			if math.Abs(got-exp) > tol*math.Abs(exp) {
				agrees = false
				if got > exp {
					high = true
				}
			}
		}
		if agrees {
			byEntry[key].agreed++
		} else {
			if high {
				byEntry[key].high++
			}
			if len(byEntry[key].diverged) < 6 {
				byEntry[key].diverged = append(byEntry[key].diverged, name)
			}
		}
	}

	floors := map[string]int{
		"tempest shield":         64,
		"cast when stunned":      160,
		"cast when damage taken": 36,
		"cast on ward break":     5,
		"automation":             199,
		"autoexertion":           42,
		"cast while channelling": 76,
		"<none>":                 8,
	}

	keys := make([]string, 0, len(byEntry))
	for k := range byEntry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := byEntry[k]
		t.Logf("%-26s agreed %d of %d  (%d of the %d divergences are ours-HIGH) %v",
			k, v.agreed, v.total, v.high, v.total-v.agreed, v.diverged)
	}
	for k, floor := range floors {
		v := byEntry[k]
		if v == nil {
			t.Errorf("REFUSING: entry %q dispatched on NO build; it has left the table or its key changed", k)
			continue
		}
		if v.agreed < floor {
			t.Errorf("%s: agreement REGRESSED to %d of %d, floor is %d", k, v.agreed, v.total, floor)
		}
	}
}

func triggerDispatchKeyForTest(env *Environment, mainSkill *ActiveSkill) string {
	grantedEffect := mainSkill.ActiveEffect.GrantedEffect
	if triggerConfigFor(env, grantedEffect.ID(), mainSkill) != nil {
		return grantedEffect.ID()
	}
	if n := strings.ToLower(grantedEffect.Name()); triggerConfigFor(env, n, mainSkill) != nil {
		return n
	}
	if mainSkill.TriggeredBy == nil || mainSkill.TriggeredBy.GrantedEffect == nil {
		return "<none>"
	}
	n := strings.ToLower(mainSkill.TriggeredBy.GrantedEffect.Name())
	if triggerConfigFor(env, n, mainSkill) != nil {
		return n
	}
	if d := strings.TrimPrefix(n, "awakened "); triggerConfigFor(env, d, mainSkill) != nil {
		return d
	}
	return "<none>"
}
