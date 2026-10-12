package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestOverAppliedCensus ranks the stats where our value is systematically LARGER than PoB's.
func TestOverAppliedCensus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, names := corpusNamesForLoadedData(t)

	var mu sync.Mutex
	ratios := map[string][]float64{}
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
		local := map[string]float64{}
		for _, pair := range oracleTablePairs(rec, env) {
			for k, want := range pair.exp {
				if want == 0 {
					continue
				}
				got, ok := pair.got[k]
				if !ok || got == 0 {
					continue
				}
				if math.IsNaN(got) || math.IsInf(got, 0) {
					continue
				}
				if math.Abs(got-want) <= 0.005*math.Abs(want) {
					continue
				}
				local[pair.prefix+k] = got / want
			}
		}
		mu.Lock()
		built++
		for k, v := range local {
			ratios[k] = append(ratios[k], v)
		}
		mu.Unlock()
	})

	if built < len(names)/2 {
		t.Fatalf("REFUSING: built %d of %d builds; a census over half a corpus is not a census",
			built, len(names))
	}

	type row struct {
		stat   string
		n      int
		median float64
	}
	var rows []row
	for k, vs := range ratios {
		if len(vs) < 25 {
			continue
		}
		sort.Float64s(vs)
		rows = append(rows, row{k, len(vs), vs[len(vs)/2]})
	}
	sort.Slice(rows, func(a, b int) bool {
		if rows[a].median != rows[b].median {
			return rows[a].median > rows[b].median
		}
		return rows[a].n > rows[b].n
	})

	fmt.Printf("\n=== OVER-APPLIED CENSUS over %d builds ===\n", built)
	fmt.Printf("stats with >=25 disagreeing builds, ranked by MEDIAN ratio (ours / PoB)\n")
	fmt.Printf("%-42s %7s %9s\n", "STAT", "BUILDS", "MEDIAN")
	shown := 0
	for _, r := range rows {
		if r.median < 1.05 {
			break
		}
		fmt.Printf("%-42s %7d %9.4f\n", r.stat, r.n, r.median)
		if shown++; shown >= 40 {
			break
		}
	}
	if shown == 0 {
		fmt.Println("  (none above 1.05 — no stat is systematically too large)")
	}

	fmt.Printf("\nCONTROL — most systematically SMALL (expect many; the residual is mostly low)\n")
	low := 0
	for i := len(rows) - 1; i >= 0 && low < 5; i-- {
		if rows[i].median >= 1 {
			break
		}
		fmt.Printf("%-42s %7d %9.4f\n", rows[i].stat, rows[i].n, rows[i].median)
		low++
	}
	if low == 0 {
		t.Fatalf("REFUSING: no stat is systematically SMALL, which contradicts every other " +
			"measurement of this residual; the oracle join is broken")
	}
	_ = strings.TrimSpace
}
