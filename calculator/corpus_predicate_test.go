package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

type corpusPredicate func(name string, build *pob.PathOfBuilding, env *Environment) bool

func countCorpusBuilds(t *testing.T, label string, pred corpusPredicate) {
	t.Helper()

	corpusDir, names := corpusNamesForLoadedData(t)

	hits := make([]bool, len(names))
	var nParseErr, nPanic int64

	runConcurrently(len(names), evalWorkers(), func(i int, cache *EnvironmentCache) {
		xml, err := os.ReadFile(filepath.Join(corpusDir, names[i]))
		if err != nil {
			atomic.AddInt64(&nParseErr, 1)
			return
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			atomic.AddInt64(&nParseErr, 1)
			return
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil {
			atomic.AddInt64(&nPanic, 1)
		}
		func() {
			defer func() { _ = recover() }()
			hits[i] = pred(strings.TrimSuffix(names[i], ".xml"), build, env)
		}()
	})

	matched := make([]string, 0, 32)
	for i, h := range hits {
		if h {
			matched = append(matched, strings.TrimSuffix(names[i], ".xml"))
		}
	}

	fmt.Printf("\n=== CORPUS PREDICATE: %s ===\n", label)
	fmt.Printf("CORPUS %s (game data %s)\n", corpusDir, raw.LatestVersion)
	fmt.Printf("MATCHED %d of %d builds (%.1f%%)\n", len(matched), len(names),
		100*float64(len(matched))/float64(len(names)))
	if nParseErr > 0 || nPanic > 0 {
		fmt.Printf("NOT EVALUABLE: %d failed to parse, %d panicked — these count as NO MATCH,\n"+
			"  so a low number here may mean 'could not tell' rather than 'does not happen'.\n",
			nParseErr, nPanic)
	}
	const show = 40
	for i, n := range matched {
		if i >= show {
			fmt.Printf("  … and %d more (all names are above the cut only by sort order)\n", len(matched)-show)
			break
		}
		fmt.Printf("  %s\n", n)
	}
	fmt.Printf("\nNOTE: this counts CORPUS builds. It is NOT the oracle's graded population —\n" +
		"builds whose dump has <5 tree sources are excluded there (184 of the 3.18 corpus). If\n" +
		"your number is going to be compared against an oracle row, exclude those too or say\n" +
		"plainly that you did not.\n")
}

// TestCorpusPredicate is a TEMPLATE.
func TestCorpusPredicate(t *testing.T) {
	if os.Getenv("PREDICATE") != "true" {
		t.Skip("PREDICATE unset — this is a scratch template, not a gate")
	}

	countCorpusBuilds(t, "build grants a minion actor", func(name string, build *pob.PathOfBuilding, env *Environment) bool {
		return env != nil && env.Minion != nil && env.Minion.Actor != nil
	})

	//			if env == nil {
}
