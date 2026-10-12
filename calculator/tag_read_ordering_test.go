package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/moddb"
)

// TestTagReadOrderingCensus reports output keys that a TAG resolved BEFORE anything wrote them, and output keys a tag read at a value the build later changed.
func TestTagReadOrderingCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	stride := 22
	if s := os.Getenv("STRIDE"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			t.Fatalf("REFUSING: STRIDE=%q is not a positive integer — a defaulted sampling knob "+
				"that happens to be the maximum is the worst failure mode this census has", s)
		}
		stride = n
	}
	stacksFor := os.Getenv("STACKS")

	corpus := corpusDirForLoadedData(t)
	files, err := filepath.Glob(filepath.Join(corpus, "*.xml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: no corpus XMLs under %s (%v)", corpus, err)
	}
	sort.Strings(files)
	if only := os.Getenv("BUILD"); only != "" {
		want := filepath.Join(corpus, only+".xml")
		if _, err := os.Stat(want); err != nil {
			t.Fatalf("REFUSING: BUILD=%s is not in %s (%v)", only, corpus, err)
		}
		files = []string{want}
		stride = 1
	}

	type key struct{ label, stat string }
	absentOn := map[key]int{}
	staleOn := map[key]int{}
	presentOn := map[key]int{}
	stacks := map[string]int{}
	staleDetail := map[key]string{}
	absentBuilds := map[key][]string{}
	staleBuilds := map[key][]string{}
	builds_ := 0

	for i := 0; i < len(files); i += stride {
		name := strings.TrimSuffix(filepath.Base(files[i]), ".xml")
		raw, err := os.ReadFile(files[i])
		if err != nil {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}

		type read struct {
			actor any
			stat  string
			value float64
			found bool
			chain string
			out   map[string]float64
		}
		var mu sync.Mutex
		var reads []read
		seen := map[[2]any]bool{}
		moddb.StatReadProbe = func(actor any, stat string, value float64, found bool) {
			mu.Lock()
			defer mu.Unlock()
			k := [2]any{actor, stat}
			if seen[k] {
				return
			}
			seen[k] = true
			r := read{actor: actor, stat: stat, value: value, found: found}
			if a, ok := actor.(*Actor); ok {
				r.out = a.Output
			}
			if stacksFor != "" && (stacksFor == "*" || stat == stacksFor) {
				r.chain = callerChain()
				if !found {
					stacks[stat+"  "+r.chain]++
				}
			}
			reads = append(reads, r)
		}
		env := func() *Environment {
			defer func() { moddb.StatReadProbe = nil }()
			return BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		}()
		builds_++

		label := func(a any) string {
			switch a {
			case any(env.Player):
				return "player"
			case any(env.Enemy):
				return "enemy"
			}
			if env.Minion != nil && a == any(env.Minion.Actor) {
				return "minion"
			}
			return "other"
		}
		perBuild := map[key]string{}
		for _, r := range reads {
			k := key{label(r.actor), r.stat}
			switch {
			case !r.found:
				perBuild[k] = "absent"
			default:
				if r.out == nil {
					if perBuild[k] == "" {
						perBuild[k] = "present"
					}
					continue
				}
				final := r.out[r.stat]
				if final != r.value && perBuild[k] != "absent" {
					perBuild[k] = "stale"
					if r.chain != "" {
						staleDetail[k] = fmt.Sprintf("read=%g final=%g  %s", r.value, final, r.chain)
					}
				} else if perBuild[k] == "" {
					perBuild[k] = "present"
				}
			}
		}
		for k, v := range perBuild {
			switch v {
			case "absent":
				absentOn[k]++
				absentBuilds[k] = append(absentBuilds[k], name)
			case "stale":
				staleOn[k]++
				staleBuilds[k] = append(staleBuilds[k], name)
			default:
				presentOn[k]++
			}
		}
	}

	if builds_ == 0 {
		t.Fatalf("REFUSING: %d corpus files matched and none built — an empty census reads exactly "+
			"like a clean one", len(files))
	}

	rows := map[key]bool{}
	for k := range absentOn {
		rows[k] = true
	}
	for k := range staleOn {
		rows[k] = true
	}
	ordered := make([]key, 0, len(rows))
	for k := range rows {
		ordered = append(ordered, k)
	}
	sort.Slice(ordered, func(a, b int) bool {
		if absentOn[ordered[a]]+staleOn[ordered[a]] != absentOn[ordered[b]]+staleOn[ordered[b]] {
			return absentOn[ordered[a]]+staleOn[ordered[a]] > absentOn[ordered[b]]+staleOn[ordered[b]]
		}
		return ordered[a].stat < ordered[b].stat
	})

	t.Logf("tag-read ordering census: %d builds sampled (stride %d of %d files)", builds_, stride, len(files))
	t.Logf("%-8s %-34s %6s %6s %8s", "actor", "stat", "absent", "stale", "present")
	for _, k := range ordered {
		t.Logf("%-8s %-34s %6d %6d %8d", k.label, k.stat, absentOn[k], staleOn[k], presentOn[k])
	}
	if stacksFor != "" {
		for _, k := range ordered {
			if (stacksFor == "*" || k.stat == stacksFor) && len(absentBuilds[k]) > 0 {
				t.Logf("--- %s %s ABSENT on: %s", k.label, k.stat, strings.Join(absentBuilds[k], " "))
			}
			if (stacksFor == "*" || k.stat == stacksFor) && len(staleBuilds[k]) > 0 {
				t.Logf("--- %s %s STALE on: %s", k.label, k.stat, strings.Join(staleBuilds[k], " "))
				if d := staleDetail[k]; d != "" {
					t.Logf("    last stale read: %s", d)
				}
			}
		}
		t.Logf("--- absent-read call chains for %q ---", stacksFor)
		chains := make([]string, 0, len(stacks))
		for c := range stacks {
			chains = append(chains, c)
		}
		sort.Slice(chains, func(a, b int) bool { return stacks[chains[a]] > stacks[chains[b]] })
		for _, c := range chains {
			t.Logf("%4d  %s", stacks[c], c)
		}
	}
}

func callerChain() string {
	pcs := make([]uintptr, 40)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var out []string
	for {
		f, more := frames.Next()
		if strings.Contains(f.File, "/calculator/") && !strings.Contains(f.File, "_test.go") {
			out = append(out, fmt.Sprintf("%s:%d", filepath.Base(f.File), f.Line))
			if len(out) >= 4 {
				break
			}
		}
		if !more {
			break
		}
	}
	return strings.Join(out, " <- ")
}
