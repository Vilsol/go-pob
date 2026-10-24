package calculator

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds/parsecache"
	"github.com/Vilsol/go-pob/data/raw"
)

// corpusCheck is one test's per-build half; visit runs concurrently across builds, so it guards its own state.
// graded is false for a build the oracle lacks or excludes (TreeSources < oracleMinTreeSources); only such a build can arrive with a nil env, when its calculation panicked.
type corpusCheck interface {
	visit(name string, rec oracleRecord, graded bool, env *Environment)
}

var corpusChecks = map[string]func() corpusCheck{}

// registerCorpusCheck puts a test's per-build check into the shared walk; the test itself keeps its name, messages and floors.
func registerCorpusCheck(test string, newCheck func() corpusCheck) bool {
	if _, dup := corpusChecks[test]; dup {
		panic("corpus check registered twice: " + test)
	}
	corpusChecks[test] = newCheck
	return true
}

type corpusWalkResult struct {
	checks map[string]corpusCheck
	err    error
}

// sharedCorpusWalk builds every corpus build once per process and hands it to each selected check.
var sharedCorpusWalk = sync.OnceValue(func() corpusWalkResult {
	checks := map[string]corpusCheck{}
	selected := selectedTestFilter()
	for test, newCheck := range corpusChecks {
		if selected(test) {
			checks[test] = newCheck()
		}
	}
	if len(checks) == 0 {
		return corpusWalkResult{checks: checks}
	}

	oracle, err := oracleForLoadedData()
	if err != nil {
		return corpusWalkResult{err: err}
	}
	dir, err := corpusDirForVersion(raw.LatestVersion)
	if err != nil {
		return corpusWalkResult{err: err}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return corpusWalkResult{err: err}
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return corpusWalkResult{err: fmt.Errorf("REFUSING: %s holds no .xml builds", dir)}
	}

	var mu sync.Mutex
	var failed []string
	runConcurrently(len(names), evalWorkers(), func(i int, cache *EnvironmentCache) {
		name := names[i]
		rawXML, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			mu.Lock()
			failed = append(failed, fmt.Sprintf("%s: %v", name, err))
			mu.Unlock()
			return
		}
		build, err := parsecache.ParseCached(rawXML)
		if err != nil {
			mu.Lock()
			failed = append(failed, fmt.Sprintf("%s: %v", name, err))
			mu.Unlock()
			return
		}
		rec, ok := oracle[name]
		graded := ok && rec.TreeSources >= oracleMinTreeSources
		var env *Environment
		func() {
			defer func() {
				if r := recover(); r != nil && graded {
					mu.Lock()
					failed = append(failed, fmt.Sprintf("%s: panicked: %v", name, r))
					mu.Unlock()
				}
			}()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil && graded {
			return
		}
		for _, c := range checks {
			c.visit(name, rec, graded, env)
		}
	})
	if len(failed) > 0 {
		return corpusWalkResult{err: fmt.Errorf("REFUSING: %d corpus builds did not load: %v", len(failed), failed)}
	}
	return corpusWalkResult{checks: checks}
})

// selectedTestFilter mirrors -test.run's top-level element so a narrowed run walks only the checks it will read.
func selectedTestFilter() func(string) bool {
	all := func(string) bool { return true }
	f := flag.Lookup("test.run")
	if f == nil || f.Value.String() == "" {
		return all
	}
	re, err := regexp.Compile(strings.Split(f.Value.String(), "/")[0])
	if err != nil {
		return all
	}
	return re.MatchString
}

// corpusWalkCheck returns the calling test's check after the shared walk has visited every build.
func corpusWalkCheck[T corpusCheck](t *testing.T) T {
	t.Helper()
	res := sharedCorpusWalk()
	if res.err != nil {
		t.Fatal(res.err)
	}
	c, ok := res.checks[t.Name()].(T)
	if !ok {
		t.Fatalf("REFUSING: %s has no check in the shared corpus walk", t.Name())
	}
	return c
}

type corpusLogEntry struct {
	build, msg string
	fatal      bool
}

// corpusCheckLog holds what a check reports mid-walk; report replays it on the test in corpus order.
type corpusCheckLog struct {
	mu      sync.Mutex
	entries []corpusLogEntry
}

func (l *corpusCheckLog) errorf(build, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, corpusLogEntry{build: build, msg: fmt.Sprintf(format, args...)})
}

func (l *corpusCheckLog) fatalf(build, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, corpusLogEntry{build: build, msg: fmt.Sprintf(format, args...), fatal: true})
}

func (l *corpusCheckLog) report(t *testing.T) {
	t.Helper()
	sort.SliceStable(l.entries, func(i, j int) bool { return l.entries[i].build < l.entries[j].build })
	for _, e := range l.entries {
		if e.fatal {
			t.Fatal(e.msg)
		}
		t.Error(e.msg)
	}
}

// corpusFirst keeps the message of the earliest build in corpus order, as a serial walk's first hit would.
type corpusFirst struct {
	build, msg string
}

func (f *corpusFirst) offer(build, msg string) {
	if f.msg == "" || build < f.build {
		f.build, f.msg = build, msg
	}
}
