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

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

func evalWorkers() int {
	n := runtime.NumCPU()
	if v := os.Getenv("WORKERS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			n = parsed
		}
	}
	return n
}

func runConcurrently(n int, workers int, work func(i int, cache *EnvironmentCache)) {
	if workers > n {
		workers = n
	}

	tasks := make(chan int, n)
	for i := 0; i < n; i++ {
		tasks <- i
	}
	close(tasks)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache := NewEnvironmentCache()
			for i := range tasks {
				work(i, cache)
			}
		}()
	}
	wg.Wait()
}

func envSnapshot(env *Environment) string {
	lines := make([]string, 0, 1024)
	for k, v := range env.Player.Output {
		lines = append(lines, fmt.Sprintf("O:%s=%.12g", k, v))
	}
	for table, m := range env.Player.OutputTable {
		for k, v := range m {
			lines = append(lines, fmt.Sprintf("T%v:%s=%.12g", table, k, v))
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// TestConcurrentEvaluationMatchesSerial is the regression test for the process-global maps that made any concurrent evaluation abort with "concurrent map read and map write".
func TestConcurrentEvaluationMatchesSerial(t *testing.T) {
	dir, names := corpusNamesForLoadedData(t)
	testza.AssertNotZero(t, len(names))

	if len(names) > 200 {
		names = names[:200]
	}

	parsed := make([]*pob.PathOfBuilding, len(names))
	for i, name := range names {
		file, err := os.ReadFile(filepath.Join(dir, name))
		testza.AssertNoError(t, err)
		build, err := builds.ParseBuild(file)
		testza.AssertNoError(t, err)
		parsed[i] = build
	}

	const rounds = 3
	workers := evalWorkers()
	if workers < 4 {
		workers = 4
	}

	tasks := len(names) * rounds
	got := make([]string, tasks)
	panics := make([]string, tasks)

	runConcurrently(tasks, workers, func(i int, cache *EnvironmentCache) {
		defer func() {
			if r := recover(); r != nil {
				panics[i] = fmt.Sprintf("%s: %v", names[i%len(names)], r)
			}
		}()
		env := BuildOutputWithCache(&Calculator{PoB: parsed[i%len(names)]}, OutputModeMain, cache)
		got[i] = envSnapshot(env)
	})

	for _, p := range panics {
		if p != "" {
			t.Errorf("panic during concurrent evaluation: %s", p)
		}
	}

	for i, name := range names {
		want := envSnapshot(BuildOutputWithCache(&Calculator{PoB: parsed[i]}, OutputModeMain, NewEnvironmentCache()))
		for r := 0; r < rounds; r++ {
			testza.AssertEqual(t, want, got[r*len(names)+i], fmt.Sprintf("%s (round %d)", name, r))
		}
	}
}
