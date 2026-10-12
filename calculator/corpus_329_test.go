package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestCorpus3_29 is the CRASH GATE for the 3.20 corpus: every build in testdata/corpus-3.20 must parse and calculate without panicking.
func TestCorpus3_29(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}

	requireGameData(t, "3.29")

	corpusDir, names := corpusNamesForLoadedData(t)

	if len(names) < 1400 {
		t.Fatalf("REFUSING: found %d builds in %s, expected >=1400. An empty or short corpus "+
			"passes this gate trivially. Check `git lfs pull`.", len(names), corpusDir)
	}

	type result struct {
		parseFailure string
		panicked     string
		built        bool
	}
	results := make([]result, len(names))

	workers := evalWorkers()
	t.Logf("evaluating %d builds across %d workers on %s game data", len(names), workers, raw.LatestVersion)

	runConcurrently(len(names), workers, func(i int, cache *EnvironmentCache) {
		res := &results[i]

		file, err := os.ReadFile(filepath.Join(corpusDir, names[i]))
		if err != nil {
			res.parseFailure = fmt.Sprintf("%s: %s", names[i], err)
			return
		}

		build, err := builds.ParseBuild(file)
		if err != nil {
			res.parseFailure = fmt.Sprintf("%s: %s", names[i], err)
			return
		}

		var env *Environment
		func() {
			defer func() {
				if r := recover(); r != nil {
					res.panicked = fmt.Sprintf("%s: %v", names[i], r)
				}
			}()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		res.built = env != nil
	})

	var (
		built         int
		parseFailures []string
		panicked      []string
	)
	for _, res := range results {
		if res.parseFailure != "" {
			parseFailures = append(parseFailures, res.parseFailure)
		}
		if res.panicked != "" {
			panicked = append(panicked, res.panicked)
		}
		if res.built {
			built++
		}
	}

	fmt.Printf("\n=== CORPUS 3.20 (game data %s) ===\n%d/%d built, %d parse failure(s), %d panic(s)\n",
		raw.LatestVersion, built, len(names), len(parseFailures), len(panicked))

	for _, f := range firstN(parseFailures, 20) {
		t.Errorf("PARSE: %s", f)
	}
	for _, p := range firstN(panicked, 20) {
		t.Errorf("PANIC: %s", p)
	}
	if len(parseFailures) > 20 {
		t.Errorf("... and %d more parse failure(s)", len(parseFailures)-20)
	}
	if len(panicked) > 20 {
		t.Errorf("... and %d more panic(s)", len(panicked)-20)
	}
}

func firstN(s []string, n int) []string {
	if len(s) < n {
		return s
	}
	return s[:n]
}
