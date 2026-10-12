package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

var knownArtifactStats = map[string]bool{
	"BrittleAvoidChance": true,
	"SapAvoidChance":     true,
	"ScorchAvoidChance":  true,
}

// TestBuildRanking ranks GRADED builds by how many oracle pairs they get wrong, and reports how concentrated the remaining error is.
func TestBuildRanking(t *testing.T) {
	requireOperatorRun(t, "report-only worklist; `task build-ranking`")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatal(err)
	}
	countArtifacts := os.Getenv("ARTIFACTS") == "1"

	corpusDir, names := corpusNamesForLoadedData(t)

	type buildRow struct {
		name      string
		wrong     int
		artifacts int
		stats     []string
	}

	var (
		mu       sync.Mutex
		rows     []buildRow
		excluded int
		total    int
		totalArt int
	)

	runConcurrently(len(names), evalWorkers(), func(i int, cache *EnvironmentCache) {
		name := names[i]
		rec, ok := oracle[name]
		if !ok {
			return
		}
		if rec.TreeSources < oracleMinTreeSources {
			mu.Lock()
			excluded++
			mu.Unlock()
			return
		}
		xml, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			return
		}
		build, err := builds.ParseBuild(xml)
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

		var wrong, art int
		per := map[string]int{}
		for _, p := range oracleTablePairs(rec, env) {
			if p.exp == nil {
				continue
			}
			for k, exp := range p.exp {
				if oracleClose(p.got[k], exp, oracleDumpTolerance) {
					continue
				}
				if knownArtifactStats[k] {
					art++
					if !countArtifacts {
						continue
					}
				}
				wrong++
				per[p.prefix+k]++
			}
		}
		if wrong == 0 && art == 0 {
			return
		}
		keys := make([]string, 0, len(per))
		for k := range per {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		mu.Lock()
		total += wrong
		totalArt += art
		if wrong > 0 {
			rows = append(rows, buildRow{strings.TrimSuffix(name, ".xml"), wrong, art, keys})
		}
		mu.Unlock()
	})

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].wrong != rows[j].wrong {
			return rows[i].wrong > rows[j].wrong
		}
		return rows[i].name < rows[j].name
	})

	top := 40
	if v := os.Getenv("TOP"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &top)
	}
	nStats := 6
	if v := os.Getenv("STATS"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &nStats)
	}

	fmt.Printf("\n=== PER-BUILD ORACLE MISMATCH RANKING ===\n")
	fmt.Printf("CORPUS %s (game data %s)\n", corpusDir, raw.LatestVersion)
	fmt.Printf("%d graded builds have at least one wrong pair; %d excluded (treeSources < %d).\n",
		len(rows), excluded, oracleMinTreeSources)
	fmt.Printf("TOTAL wrong pairs %d", total)
	if !countArtifacts {
		fmt.Printf("  (+%d in the %d known-artifact stats, EXCLUDED — ARTIFACTS=1 to count them)",
			totalArt, len(knownArtifactStats))
	}
	fmt.Printf("\n\n")

	cum := 0
	fmt.Printf("CONCENTRATION — share of remaining error held by the worst N builds:\n")
	for _, n := range []int{10, 25, 50, 100, 200, 400} {
		if n > len(rows) {
			break
		}
		cum = 0
		for i := 0; i < n; i++ {
			cum += rows[i].wrong
		}
		fmt.Printf("  top %-4d %6d pairs  %5.1f%%\n", n, cum, 100*float64(cum)/float64(total))
	}

	fmt.Printf("\nWORST BUILDS (a build's stats are usually ONE cause smeared across consumers —\n")
	fmt.Printf("read the stat list as a signature, not as a defect count):\n")
	for i, r := range rows {
		if i >= top {
			fmt.Printf("  … and %d more builds holding %d pairs\n", len(rows)-top, total-cum)
			break
		}
		s := r.stats
		if len(s) > nStats {
			s = append(append([]string{}, s[:nStats]...), fmt.Sprintf("+%d", len(r.stats)-nStats))
		}
		fmt.Printf("  %-50s %4d  %s\n", r.name, r.wrong, strings.Join(s, " "))
	}
}
