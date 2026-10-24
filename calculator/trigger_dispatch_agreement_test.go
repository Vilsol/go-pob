package calculator

import (
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
)

var _ = registerCorpusCheck("TestTriggerDispatchAgreesWithOracle", func() corpusCheck { return &triggerDispatchCheck{} })

type triggerDispatchResult struct {
	name, key    string
	agrees, high bool
}

type triggerDispatchCheck struct {
	mu      sync.Mutex
	results []triggerDispatchResult
	log     corpusCheckLog
}

func (c *triggerDispatchCheck) visit(name string, rec oracleRecord, _ bool, env *Environment) {
	const tol = 0.005
	if rec.Output == nil {
		return
	}
	if env == nil || env.Player == nil || env.Player.MainSkill == nil {
		return
	}
	if env.Player.Output["TriggerRateCap"] == 0 {
		return
	}
	r := triggerDispatchResult{name: name, key: triggerDispatchKeyForTest(env, env.Player.MainSkill), agrees: true}
	for _, stat := range []string{"TriggerRateCap", "EffectiveSourceRate", "SkillTriggerRate"} {
		got := env.Player.Output[stat]
		exp := rec.Output[stat]
		if math.IsNaN(got) || math.IsInf(got, 0) {
			c.log.fatalf(name, "%s: %s is %v -- a NaN/Inf cannot be graded and its ceiling only moves down", name, stat, got)
			return
		}
		if exp == 0 {
			if got != 0 {
				r.agrees = false
			}
			continue
		}
		if math.Abs(got-exp) > tol*math.Abs(exp) {
			r.agrees = false
			if got > exp {
				r.high = true
			}
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = append(c.results, r)
}

// TestTriggerDispatchAgreesWithOracle sweeps every corpus build whose MAIN SKILL our trigger dispatcher answers for.
func TestTriggerDispatchAgreesWithOracle(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	c := corpusWalkCheck[*triggerDispatchCheck](t)
	c.log.report(t)
	sort.Slice(c.results, func(i, j int) bool { return c.results[i].name < c.results[j].name })

	type tally struct {
		agreed, total, high int
		diverged            []string
	}
	byEntry := map[string]*tally{}
	for _, r := range c.results {
		if byEntry[r.key] == nil {
			byEntry[r.key] = &tally{}
		}
		byEntry[r.key].total++
		if r.agrees {
			byEntry[r.key].agreed++
		} else {
			if r.high {
				byEntry[r.key].high++
			}
			if len(byEntry[r.key].diverged) < 6 {
				byEntry[r.key].diverged = append(byEntry[r.key].diverged, r.name)
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
