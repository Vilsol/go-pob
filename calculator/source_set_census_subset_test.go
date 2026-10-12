package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestSourceSetCensusSubset is TestSourceSetCensus over a NAMED SUBSET of the corpus.
func TestSourceSetCensusSubset(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	listPath := os.Getenv("CENSUS_ONLY")
	if listPath == "" {
		t.Skip("set CENSUS_ONLY=<file of build names>")
	}
	raw, err := os.ReadFile(listPath)
	if err != nil {
		t.Fatalf("CENSUS_ONLY: %v", err)
	}
	want := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		want[strings.TrimSuffix(line, ".xml")] = true
	}
	if len(want) == 0 {
		t.Fatalf("REFUSING: CENSUS_ONLY %s named no builds", listPath)
	}

	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	dumps := diagDumpsDir(t)
	corpusDir, all := corpusNamesForLoadedData(t)

	files := make([]string, 0, len(want))
	for _, f := range all {
		if want[strings.TrimSuffix(f, ".xml")] {
			files = append(files, f)
		}
	}
	if len(files) != len(want) {
		t.Fatalf("REFUSING: CENSUS_ONLY named %d builds, %d are in the loaded corpus — a census "+
			"over a silently-shortened list has an unknown denominator", len(want), len(files))
	}

	type agg struct {
		builds  int
		wrong   int
		names   map[string]bool
		example string
	}
	var mu sync.Mutex
	onlyPoB := map[string]*agg{}
	oursOnlyMods := map[string]*agg{}
	pobOnlyMods := map[string]*agg{}
	built, unhealthy, totalWrong := 0, 0, 0

	runConcurrently(len(files), 12, func(i int, cache *EnvironmentCache) {
		rec, ok := oracle[files[i]]
		if !ok {
			return
		}
		name := strings.TrimSuffix(files[i], ".xml")
		dump, healthy := diagLoadDump(dumps, name)
		if dump == nil {
			return
		}
		if !healthy {
			mu.Lock()
			unhealthy++
			mu.Unlock()
			return
		}
		file, err := os.ReadFile(filepath.Join(corpusDir, files[i]))
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

		wrong := 0
		for _, pair := range oracleTablePairs(rec, env) {
			for k, expect := range pair.exp {
				got, ok := pair.got[k]
				if !ok {
					continue
				}
				if math.IsNaN(got) || math.IsInf(got, 0) {
					wrong++
					continue
				}
				if math.Abs(got-expect) > 0.005*math.Abs(expect) {
					wrong++
				}
			}
		}

		pobMods, ourDB, _ := diagActorModDB(dump, env, "player")
		pobNames := map[string]string{}
		pob := map[string]bool{}
		pobBag := map[string]int{}
		for modName, mods := range pobMods {
			for _, m := range mods {
				key := normalizeSource(m.Source)
				pob[key] = true
				if i := strings.Index(m.Source, "("); i >= 0 {
					pobNames[key] = strings.TrimSuffix(m.Source[i+1:], ")")
				}
				pobBag[modBagKey(modName, m.Type, dumpModValueKey(m.Value))]++
			}
		}
		ours := map[string]bool{}
		ourBag := map[string]int{}
		if ourDB != nil {
			for modName, mods := range ourDB.Mods {
				for _, m := range mods {
					ours[normalizeSource(string(m.GetSource()))] = true
					ourBag[modBagKey(modName, string(m.Type()),
						strconv.FormatFloat(m.Value().Float(), 'g', -1, 64))]++
				}
			}
		}

		mu.Lock()
		built++
		totalWrong += wrong
		bump := func(m map[string]*agg, key, nm string) {
			a := m[key]
			if a == nil {
				a = &agg{names: map[string]bool{}, example: name}
				m[key] = a
			}
			a.builds++
			a.wrong += wrong
			if nm != "" {
				a.names[nm] = true
			}
		}
		for k := range pob {
			if !ours[k] {
				bump(onlyPoB, k, pobNames[k])
			}
		}
		for k, n := range ourBag {
			if n > pobBag[k] {
				bump(oursOnlyMods, k, "")
			}
		}
		for k, n := range pobBag {
			if n > ourBag[k] {
				bump(pobOnlyMods, k, "")
			}
		}
		mu.Unlock()
	})

	if built < len(files)/2 {
		t.Fatalf("REFUSING: built %d of %d builds; a census over half a list is not a census",
			built, len(files))
	}

	fmt.Printf("\n=== SUBSET SOURCE-SET CENSUS over %d of %d listed builds (%d unhealthy dumps) ===\n",
		built, len(files), unhealthy)
	fmt.Printf("%d wrong pairs total · mean %.2f wrong/build · list %s\n",
		totalWrong, float64(totalWrong)/float64(built), listPath)
	fmt.Printf("⚠️  RANK ON `wrong` (residual held by the carriers), NOT on `builds`.\n")

	report := func(title string, m map[string]*agg, limit int) {
		type row struct {
			key string
			a   *agg
		}
		rows := make([]row, 0, len(m))
		for k, a := range m {
			rows = append(rows, row{k, a})
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].a.wrong != rows[j].a.wrong {
				return rows[i].a.wrong > rows[j].a.wrong
			}
			return rows[i].key < rows[j].key
		})
		fmt.Printf("\n=== %s (%d distinct) ===\n", title, len(rows))
		fmt.Printf("%-40s %7s %8s %7s %4s  %-30s %s\n", "KEY", "BUILDS", "WRONG", "W/BLD", "n", "NAME(S)", "EXAMPLE")
		for i, r := range rows {
			if i >= limit {
				fmt.Printf("… +%d more\n", len(rows)-limit)
				break
			}
			nm := make([]string, 0, len(r.a.names))
			for n := range r.a.names {
				nm = append(nm, n)
			}
			sort.Strings(nm)
			joined := strings.Join(nm, " | ")
			if len(joined) > 30 {
				joined = joined[:27] + "..."
			}
			fmt.Printf("%-40s %7d %8d %7.1f %4d  %-30s %s\n",
				r.key, r.a.builds, r.a.wrong, float64(r.a.wrong)/float64(r.a.builds),
				len(r.a.names), joined, r.a.example)
		}
	}

	report("ONLY POB — a source we never build (UPPER BOUND)", onlyPoB, 40)
	report("MODS WE EMIT AND PoB DOES NOT — source-blind (OVER-APPLICATION)", oursOnlyMods, 40)
	report("MODS PoB EMITS AND WE DO NOT — source-blind (STARVED)", pobOnlyMods, 40)
}
