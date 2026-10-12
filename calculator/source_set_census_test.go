package calculator

import (
	"encoding/json"
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

func modBagKey(name, typ, value string) string {
	switch typ {
	case "FLAG", "LIST":
		return name + "|" + typ
	}
	return name + "|" + typ + "|" + value
}

func dumpModValueKey(raw json.RawMessage) string {
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	return string(raw)
}

// TestSourceSetCensus lifts `task diag STEP=sources` from ONE build to the WHOLE corpus.
func TestSourceSetCensus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	dumps := diagDumpsDir(t)
	corpusDir, files := corpusNamesForLoadedData(t)

	type agg struct {
		builds  int
		wrong   int
		names   map[string]bool
		example string
	}
	var mu sync.Mutex
	onlyPoB := map[string]*agg{}
	onlyOurs := map[string]*agg{}
	oursOnlyMods := map[string]*agg{}
	pobOnlyMods := map[string]*agg{}
	built, unhealthy, totalWrong, noneMods := 0, 0, 0, 0

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
			for k, want := range pair.exp {
				got, ok := pair.got[k]
				if !ok {
					continue
				}
				if math.IsNaN(got) || math.IsInf(got, 0) {
					wrong++
					continue
				}
				if math.Abs(got-want) > 0.005*math.Abs(want) {
					wrong++
				}
			}
		}

		pobMods, ourDB, _ := diagActorModDB(dump, env, "player")
		pobNames := map[string]string{}
		pob := map[string]bool{}
		pobBag := map[string]int{}
		for name, mods := range pobMods {
			for _, m := range mods {
				key := normalizeSource(m.Source)
				pob[key] = true
				if i := strings.Index(m.Source, "("); i >= 0 {
					pobNames[key] = strings.TrimSuffix(m.Source[i+1:], ")")
				}
				pobBag[modBagKey(name, m.Type, dumpModValueKey(m.Value))]++
			}
		}
		ours := map[string]bool{}
		ourBag := map[string]int{}
		nones := 0
		if ourDB != nil {
			for name, mods := range ourDB.Mods {
				for _, m := range mods {
					k := normalizeSource(string(m.GetSource()))
					ours[k] = true
					if k == "Other:<none>" {
						nones++
					}
					ourBag[modBagKey(name, string(m.Type()),
						strconv.FormatFloat(m.Value().Float(), 'g', -1, 64))]++
				}
			}
		}

		mu.Lock()
		built++
		totalWrong += wrong
		noneMods += nones
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
		for k := range ours {
			if !pob[k] {
				bump(onlyOurs, k, "")
			}
		}
		for k, n := range ourBag {
			if d := n - pobBag[k]; d > 0 {
				bump(oursOnlyMods, k, "")
			}
		}
		for k, n := range pobBag {
			if d := n - ourBag[k]; d > 0 {
				bump(pobOnlyMods, k, "")
			}
		}
		mu.Unlock()
	})

	if built < len(files)/2 {
		t.Fatalf("REFUSING: built %d of %d builds; a census over half a corpus is not a census",
			built, len(files))
	}

	mean := float64(totalWrong) / float64(built)
	fmt.Printf("\n=== SOURCE-SET CENSUS over %d builds (%d unhealthy dumps skipped) ===\n", built, unhealthy)
	fmt.Printf("%d wrong pairs total · mean %.2f wrong/build · %d of our mods carry Other:<none>\n",
		totalWrong, mean, noneMods)
	fmt.Printf("⚠️  RANK ON `wrong` (residual held by the carriers), NOT on `builds`.\n")
	fmt.Printf("⚠️  `n` is DISTINCT PoB NAMES for one id: n=1 a node we never allocate, n>1 a CONQUERED node.\n")

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
		fmt.Printf("\n=== %s (%d distinct sources) ===\n", title, len(rows))
		fmt.Printf("%-28s %7s %8s %7s %4s  %-34s %s\n", "SOURCE", "BUILDS", "WRONG", "W/BLD", "n", "NAME(S)", "EXAMPLE")
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
			if len(joined) > 34 {
				joined = joined[:31] + "..."
			}
			fmt.Printf("%-28s %7d %8d %7.1f %4d  %-34s %s\n",
				r.key, r.a.builds, r.a.wrong, float64(r.a.wrong)/float64(r.a.builds),
				len(r.a.names), joined, r.a.example)
		}
	}

	report("ONLY POB — a source we never build (UPPER BOUND)", onlyPoB, 45)
	report("ONLY OURS — a source PoB never builds (invented)", onlyOurs, 45)

	report("MODS WE EMIT AND PoB DOES NOT — source-blind (OVER-APPLICATION)", oursOnlyMods, 40)
	report("MODS PoB EMITS AND WE DO NOT — source-blind (STARVED)", pobOnlyMods, 40)
}
