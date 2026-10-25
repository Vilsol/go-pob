package calculator2

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob2"
)

// itemOutputStats are graded on the oracle's output; itemBaseStats on the BASE sums PoB's final
// modDB holds, both in total and from item sources alone.
var (
	itemOutputStats = []string{"Str", "Dex", "Int", "Life", "Mana", "Spirit"}
	itemBaseStats   = []string{"Str", "Dex", "Int", "Life", "Mana", "Spirit", "Armour", "Evasion", "EnergyShield", "FireResist", "ColdResist", "LightningResist", "ChaosResist"}
)

type dumpMod struct {
	Source string `json:"source"`
	Type   string `json:"type"`
	Value  any    `json:"value"`
}

func isItemSource(s string) bool { return strings.HasPrefix(s, "Item:") }

func baseSums(get func(name string) [][3]any) (map[string]float64, map[string]float64) {
	total, items := map[string]float64{}, map[string]float64{}
	for _, name := range itemBaseStats {
		for _, m := range get(name) {
			if m[0].(string) != "BASE" {
				continue
			}
			v, ok := m[2].(float64)
			if !ok {
				continue
			}
			total[name] += v
			if isItemSource(m[1].(string)) {
				items[name] += v
			}
		}
	}
	return total, items
}

// TestCorpusPoE2ItemsVsOracle reports how many of the local PoE2 builds agree with PoB-PoE2 on the
// attribute and pool outputs and on the BASE sums items put into the player modDB.
// POE2_ITEMS=off measures the same with no items; POE2_ITEMS_WORKLIST=<file> writes the disagreements,
// and <file>.sources each side's BASE mods behind a disagreeing total.
func TestCorpusPoE2ItemsVsOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 corpus")
	}
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5-skillsoff")
	oracleDir := filepath.Join("..", "testdata", "oracle-poe2-0.5")
	oraclePath := filepath.Join(oracleDir, "output-poe2-0.5-skillsoff.jsonl.gz")
	if _, err := os.Stat(oraclePath); err != nil {
		t.Skipf("no PoE2 skills-off oracle at %s", oraclePath)
	}
	realOracle, err := filepath.EvalSymlinks(oracleDir)
	if err != nil {
		t.Fatal(err)
	}
	dumpDir := filepath.Join(filepath.Dir(realOracle), "dumps-skillsoff")
	if _, err := os.Stat(dumpDir); err != nil {
		t.Skipf("no PoE2 skills-off dumps beside the oracle at %s", dumpDir)
	}
	tree := loadTestTree(t)
	itemsOff := os.Getenv("POE2_ITEMS") == "off"
	limit, _ := strconv.Atoi(os.Getenv("LIMIT"))

	f, err := os.Open(oraclePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		Build string         `json:"build"`
		Raw   map[string]any `json:"output"`
	}
	var rows []row
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 64<<20), 64<<20)
	for sc.Scan() {
		var r row
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
		if limit > 0 && len(rows) >= limit {
			break
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	type result struct {
		err                 error
		diffs, sources      []string
		agreeOut, agreeBase map[string]bool
		agreeItem           map[string]bool
	}
	results := make([]result, len(rows))
	near := func(a, b float64) bool { return math.Abs(a-b) <= 1e-6*math.Max(1, math.Abs(a)) }
	work := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < min(8, runtime.GOMAXPROCS(0)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				r := rows[i]
				res := result{agreeOut: map[string]bool{}, agreeBase: map[string]bool{}, agreeItem: map[string]bool{}}
				raw, err := os.ReadFile(filepath.Join(corpusDir, r.Build))
				var build *pob2.PathOfBuilding2
				if err == nil {
					build, err = pob2.ParseBuild(raw)
				}
				var env *Environment
				if err == nil {
					opts := Options{Tree: tree}
					if !itemsOff {
						opts.Items = NewItemsTab(build, tree, nil)
					}
					env, err = BuildOutput(build, opts)
				}
				var dump struct {
					ModDB      map[string][]dumpMod `json:"modDB"`
					EnemyModDB map[string][]dumpMod `json:"enemyModDB"`
				}
				if err == nil {
					var dr []byte
					dr, err = os.ReadFile(filepath.Join(dumpDir, strings.TrimSuffix(r.Build, ".xml")+".json"))
					if err == nil {
						err = json.Unmarshal(dr, &dump)
					}
				}
				if err != nil {
					res.err = err
					results[i] = res
					continue
				}
				pobAcc, ourAcc := 0.0, 0.0
				for _, m := range dump.EnemyModDB["Accuracy"] {
					if v, ok := m.Value.(float64); ok && m.Type == "BASE" && m.Source == "Base" {
						pobAcc += v
					}
				}
				for _, m := range env.EnemyDB.Mods["Accuracy"] {
					if m.Type() == mod.TypeBase && m.GetSource() == "Base" {
						ourAcc += m.Value().Float()
					}
				}
				res.agreeOut["EnemyAccuracy"] = near(pobAcc, ourAcc)
				for _, s := range itemOutputStats {
					pv, _ := r.Raw[s].(float64)
					ov := env.Player.Output[s]
					res.agreeOut[s] = near(pv, ov)
					if !res.agreeOut[s] {
						res.diffs = append(res.diffs, fmt.Sprintf("%s\toutput\t%s\t%g\t%g", r.Build, s, pv, ov))
					}
				}
				pobTotal, pobItems := baseSums(func(name string) [][3]any {
					var out [][3]any
					for _, m := range dump.ModDB[name] {
						out = append(out, [3]any{m.Type, m.Source, m.Value})
					}
					return out
				})
				ourTotal, ourItems := baseSums(func(name string) [][3]any {
					var out [][3]any
					for _, m := range env.ModDB.Mods[name] {
						var v any
						if mv := m.Value(); mv != nil && mv.Type() == mod.ModValueMultiTypeFloat {
							v = mv.Float()
						}
						out = append(out, [3]any{string(m.Type()), string(m.GetSource()), v})
					}
					return out
				})
				for _, s := range itemBaseStats {
					res.agreeBase[s] = near(pobTotal[s], ourTotal[s])
					res.agreeItem[s] = near(pobItems[s], ourItems[s])
					if !res.agreeItem[s] {
						res.diffs = append(res.diffs, fmt.Sprintf("%s\titemBASE\t%s\t%g\t%g", r.Build, s, pobItems[s], ourItems[s]))
					} else if !res.agreeBase[s] {
						res.diffs = append(res.diffs, fmt.Sprintf("%s\tBASE\t%s\t%g\t%g", r.Build, s, pobTotal[s], ourTotal[s]))
					}
					if !res.agreeBase[s] {
						for _, m := range dump.ModDB[s] {
							if v, ok := m.Value.(float64); ok && m.Type == "BASE" {
								res.sources = append(res.sources, fmt.Sprintf("%s\t%s\tpob\t%s\t%g", r.Build, s, m.Source, v))
							}
						}
						for _, m := range env.ModDB.Mods[s] {
							if v := m.Value(); v != nil && v.Type() == mod.ModValueMultiTypeFloat && m.Type() == mod.TypeBase {
								res.sources = append(res.sources, fmt.Sprintf("%s\t%s\tours\t%s\t%g", r.Build, s, m.GetSource(), v.Float()))
							}
						}
					}
				}
				results[i] = res
			}
		}()
	}
	for i := range rows {
		work <- i
	}
	close(work)
	wg.Wait()

	failed := 0
	agreeOut, agreeBase, agreeItem := map[string]int{}, map[string]int{}, map[string]int{}
	var diffs, sources []string
	for _, r := range results {
		if r.err != nil {
			failed++
			continue
		}
		for s, ok := range r.agreeOut {
			if ok {
				agreeOut[s]++
			}
		}
		for s, ok := range r.agreeBase {
			if ok {
				agreeBase[s]++
			}
		}
		for s, ok := range r.agreeItem {
			if ok {
				agreeItem[s]++
			}
		}
		diffs = append(diffs, r.diffs...)
		sources = append(sources, r.sources...)
	}
	n := len(results) - failed
	t.Logf("enemy base Accuracy (enemy level) agree %d / %d", agreeOut["EnemyAccuracy"], n)
	for _, s := range itemOutputStats {
		t.Logf("output %-16s agree %d / %d", s, agreeOut[s], n)
	}
	for _, s := range itemBaseStats {
		t.Logf("BASE   %-16s total agree %d / %d, item-sourced agree %d / %d", s, agreeBase[s], n, agreeItem[s], n)
	}
	t.Logf("builds %d, failed %d", len(results), failed)
	for i, r := range results {
		if r.err != nil {
			t.Logf("first failure: %s: %v", rows[i].Build, r.err)
			break
		}
	}
	if p := os.Getenv("POE2_ITEMS_WORKLIST"); p != "" {
		sort.Strings(diffs)
		if err := os.WriteFile(p, []byte("build\tkind\tstat\tpob\tours\n"+strings.Join(diffs, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p+".sources", []byte("build\tstat\tside\tsource\tvalue\n"+strings.Join(sources, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
