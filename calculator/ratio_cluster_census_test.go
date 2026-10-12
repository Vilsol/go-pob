package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestRatioClusterCensus ranks every graded stat by its LARGEST EXACT CLUSTER rather than by its disagreement count.
func TestRatioClusterCensus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, names := corpusNamesForLoadedData(t)

	type obs struct {
		build string
		ratio float64
		delta float64
	}

	var mu sync.Mutex
	obsByStat := map[string][]obs{}
	built := 0

	runConcurrently(len(names), 12, func(i int, cache *EnvironmentCache) {
		name := names[i]
		rec, ok := oracle[name]
		if !ok {
			return
		}
		file, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			return
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			return
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil {
			return
		}
		local := map[string]obs{}
		for _, pair := range oracleTablePairs(rec, env) {
			for k, want := range pair.exp {
				got, ok := pair.got[k]
				if !ok {
					continue
				}
				if math.IsNaN(got) || math.IsInf(got, 0) {
					continue
				}
				if math.Abs(got-want) <= 0.005*math.Abs(want) {
					continue
				}
				r := math.NaN()
				if want != 0 {
					r = got / want
				}
				local[pair.prefix+k] = obs{build: name, ratio: r, delta: got - want}
			}
		}
		mu.Lock()
		built++
		for k, v := range local {
			obsByStat[k] = append(obsByStat[k], v)
		}
		mu.Unlock()
	})

	if built < len(names)/2 {
		t.Fatalf("REFUSING: built %d of %d builds; a census over half a corpus is not a census",
			built, len(names))
	}

	round := func(v float64) (float64, bool) {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return math.Round(v*10000) / 10000, true
	}

	type row struct {
		stat         string
		wrong        int
		mBest, aBest int
		mKey, aKey   float64
		mExample     string
		aExample     string
	}
	rows := make([]row, 0, len(obsByStat))
	for stat, os_ := range obsByStat {
		mc := map[float64][]string{}
		ac := map[float64][]string{}
		for _, o := range os_ {
			if k, ok := round(o.ratio); ok {
				mc[k] = append(mc[k], o.build)
			}
			if k, ok := round(o.delta); ok {
				ac[k] = append(ac[k], o.build)
			}
		}
		r := row{stat: stat, wrong: len(os_)}
		for k, v := range mc {
			if len(v) > r.mBest {
				r.mBest, r.mKey, r.mExample = len(v), k, v[0]
			}
		}
		for k, v := range ac {
			if len(v) > r.aBest {
				r.aBest, r.aKey, r.aExample = len(v), k, v[0]
			}
		}
		rows = append(rows, r)
	}

	emit := func(title string, best func(row) int, key func(row) float64, ex func(row) string) {
		sorted := make([]row, len(rows))
		copy(sorted, rows)
		sort.Slice(sorted, func(a, b int) bool {
			if best(sorted[a]) != best(sorted[b]) {
				return best(sorted[a]) > best(sorted[b])
			}
			return sorted[a].stat < sorted[b].stat
		})
		fmt.Printf("\n=== %s ===\n", title)
		fmt.Printf("%-46s %6s %6s %5s %12s  %s\n", "STAT", "WRONG", "CLUST", "CONC", "KEY", "EXAMPLE BUILD")
		n := 0
		for _, r := range sorted {
			if best(r) < 3 {
				break
			}
			fmt.Printf("%-46s %6d %6d %4.0f%% %12.4f  %s\n",
				r.stat, r.wrong, best(r), 100*float64(best(r))/float64(r.wrong), key(r), ex(r))
			if n++; n >= 60 {
				break
			}
		}
	}

	fmt.Printf("\n=== RATIO CLUSTER CENSUS over %d builds, %d stats with a disagreement ===\n",
		built, len(rows))
	emit("MULTIPLICATIVE — largest exact got/PoB cluster", func(r row) int { return r.mBest },
		func(r row) float64 { return r.mKey }, func(r row) string { return r.mExample })
	emit("ADDITIVE — largest exact got-PoB cluster", func(r row) int { return r.aBest },
		func(r row) float64 { return r.aKey }, func(r row) string { return r.aExample })
}
