package calculator

import (
	"encoding/json"
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

func knownConfigOption(name string) bool {
	for _, n := range configOrder {
		if n == name {
			return true
		}
	}
	return false
}

func TestConfigCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census; `task census WHICH=config`")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	dumps := diagDumpsDir(t)

	entries, err := os.ReadDir(dumps)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	only := strings.TrimSuffix(os.Getenv("BUILD"), ".xml")
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		n := strings.TrimSuffix(e.Name(), ".json")
		if only != "" && n != only {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Skipf("no dumps under %s", dumps)
	}

	type keyStat struct {
		differ, absent, extra int
		example               string
		exPoB, exOurs         float64
	}

	var (
		mu            sync.Mutex
		phStats       = map[string]*keyStat{}
		inputMismatch = map[string]int{}
		nBuilds       int
		nNoConfig     int
		nUnparsed     int
		buildDiffers  = map[string]int{}
	)

	runConcurrently(len(names), evalWorkers(), func(i int, _ *EnvironmentCache) {
		name := names[i]
		raw, err := os.ReadFile(filepath.Join(dumps, name+".json"))
		if err != nil {
			return
		}
		var d struct {
			ConfigInput       map[string]any     `json:"configInput"`
			ConfigPlaceholder map[string]float64 `json:"configPlaceholder"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			var loose struct {
				ConfigInput       map[string]any `json:"configInput"`
				ConfigPlaceholder map[string]any `json:"configPlaceholder"`
			}
			if json.Unmarshal(raw, &loose) != nil {
				return
			}
			d.ConfigInput = loose.ConfigInput
			d.ConfigPlaceholder = map[string]float64{}
			for k, v := range loose.ConfigPlaceholder {
				if f, ok := v.(float64); ok {
					d.ConfigPlaceholder[k] = f
				}
			}
		}

		mu.Lock()
		if d.ConfigPlaceholder == nil && d.ConfigInput == nil {
			nNoConfig++
			mu.Unlock()
			return
		}
		nBuilds++
		mu.Unlock()

		xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name+".xml"))
		if err != nil {
			mu.Lock()
			nUnparsed++
			mu.Unlock()
			return
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			mu.Lock()
			nUnparsed++
			mu.Unlock()
			return
		}

		ours := enemyConfigPlaceholders(build)

		mu.Lock()
		defer mu.Unlock()

		seen := map[string]bool{}
		for k := range d.ConfigPlaceholder {
			seen[k] = true
		}
		for k := range ours {
			seen[k] = true
		}
		for k := range seen {
			exp, hasPoB := d.ConfigPlaceholder[k]
			got, hasOurs := ours[k]
			st := phStats[k]
			if st == nil {
				st = &keyStat{}
				phStats[k] = st
			}
			switch {
			case hasPoB && !hasOurs:
				st.absent++
			case !hasPoB && hasOurs:
				st.extra++
			case exp != got && math.Abs(exp-got) > 1e-9*math.Max(math.Abs(exp), 1):
				st.differ++
			default:
				continue
			}
			buildDiffers[name]++
			if st.example == "" || name < st.example {
				st.example, st.exPoB, st.exOurs = name, exp, got
			}
		}

		for k := range d.ConfigInput {
			if knownConfigOption(k) {
				continue
			}
			inputMismatch[k]++
		}
	})

	fmt.Printf("\n=== CONFIG CENSUS (%d builds with config data) ===\n", nBuilds)
	if nNoConfig > 0 {
		fmt.Printf("SKIPPED %d dumps carrying no configInput/configPlaceholder — that dump set predates\n"+
			"  oracle/dump_build.lua's 2026-08-04 change. Re-dump, or point DUMPS at v4 or later.\n", nNoConfig)
	}
	if nUnparsed > 0 {
		fmt.Printf("NOT EVALUABLE: %d builds failed to parse on our side.\n", nUnparsed)
	}
	fmt.Printf("A config defect produces NO MOD, so nothing else here can see this surface.\n" +
		"DIFFER = both sides have the key, values disagree — an unported or misordered handler.\n" +
		"ABSENT = PoB computed a placeholder we never write. Only a defect if something reads it;\n" +
		"         our table is deliberately enemy-scoped, so a wider PoB key is expected.\n" +
		"EXTRA  = we invented a placeholder PoB does not have. Always worth explaining.\n\n")

	keys := make([]string, 0, len(phStats))
	for k := range phStats {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := phStats[keys[i]], phStats[keys[j]]
		ta, tb := a.differ+a.absent+a.extra, b.differ+b.absent+b.extra
		if ta != tb {
			return ta > tb
		}
		return keys[i] < keys[j]
	})

	fmt.Printf("%-34s %8s %8s %8s   %s\n", "PLACEHOLDER KEY", "DIFFER", "ABSENT", "EXTRA", "EXAMPLE (PoB -> ours)")
	var tD, tA, tE, shown int
	for _, k := range keys {
		st := phStats[k]
		tD, tA, tE = tD+st.differ, tA+st.absent, tE+st.extra
		if st.differ+st.absent+st.extra == 0 {
			continue
		}
		shown++
		fmt.Printf("%-34s %8d %8d %8d   %g -> %g [%s]\n", k, st.differ, st.absent, st.extra, st.exPoB, st.exOurs, st.example)
	}
	if shown == 0 {
		fmt.Printf("%-34s   every one of the %d compared keys agrees on every build\n", "(none)", len(keys))
	}
	fmt.Printf("%-34s %8d %8d %8d\n", "TOTAL", tD, tA, tE)

	if len(inputMismatch) > 0 {
		fmt.Printf("\nSAVED CONFIG INPUTS OUR CONFIG LAYER DOES NOT MODEL:\n" +
			"  ⚠️ NOT ALL OF THESE ARE GAPS. The corpus XMLs were exported by HETEROGENEOUS, MOSTLY\n" +
			"  NEWER PoB versions (a version confound), so an option here may not\n" +
			"  exist in the pinned v2.18.1 either — `brandAttachedToEnemy` is one such: it is absent\n" +
			"  from our tree AND from ConfigOptions.lua. GREP ConfigOptions.lua BEFORE treating a\n" +
			"  row as a defect; if PoB does not model it, neither should we.\n")
		ik := make([]string, 0, len(inputMismatch))
		for k := range inputMismatch {
			ik = append(ik, k)
		}
		sort.Slice(ik, func(i, j int) bool { return inputMismatch[ik[i]] > inputMismatch[ik[j]] })
		for _, k := range ik {
			fmt.Printf("  %-44s %5d builds\n", k, inputMismatch[k])
		}
	}

	type bd struct {
		name string
		n    int
	}
	worst := make([]bd, 0, len(buildDiffers))
	for n, c := range buildDiffers {
		worst = append(worst, bd{n, c})
	}
	sort.Slice(worst, func(i, j int) bool {
		if worst[i].n != worst[j].n {
			return worst[i].n > worst[j].n
		}
		return worst[i].name < worst[j].name
	})
	fmt.Printf("\nWORST BUILDS (%d of %d have at least one placeholder divergence):\n", len(worst), nBuilds)
	for i, w := range worst {
		if i >= 25 {
			fmt.Printf("  … and %d more\n", len(worst)-25)
			break
		}
		fmt.Printf("  %-52s %3d keys\n", w.name, w.n)
	}
}
