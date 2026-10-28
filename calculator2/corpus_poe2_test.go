package calculator2

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob2"
	"github.com/Vilsol/go-pob/storage"
)

var (
	testTreeOnce sync.Once
	testTree     *tree2.PassiveTree
	errTestTree  error
)

func loadTestTree(t *testing.T) *tree2.PassiveTree {
	t.Helper()
	testTreeOnce.Do(func() {
		testTree, errTestTree = tree2.LoadData(context.Background(), "0_5", raw.AssetLoaderWrapper{Storage: storage.Get()})
	})
	if errTestTree != nil {
		t.Skipf("go-pob-data's PoE2 tree not loadable: %v", errTestTree)
	}
	return testTree
}

// sliceStats are the stats this package computes so far, graded against the skills-off oracle.
var sliceStats = []string{"Str", "Dex", "Int", "Life", "Mana", "Spirit"}

// TestCorpusPoE2SetupVsOracle reports, per stat, how many builds agree with PoB-PoE2's skills-off
// oracle while items contribute nothing to the calculation. It grades nothing yet: a build whose
// items carry the stat disagrees by construction. POE2_SETUP_WORKLIST=<file> writes per-build rows.
func TestCorpusPoE2SetupVsOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 corpus")
	}
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5")
	oraclePath := filepath.Join("..", "testdata", "oracle-poe2-0.5", "output-poe2-0.5-skillsoff.jsonl.gz")
	if _, err := os.Stat(oraclePath); err != nil {
		t.Skipf("no PoE2 skills-off oracle at %s", oraclePath)
	}
	tree := loadTestTree(t)

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
		Build  string             `json:"build"`
		Output map[string]float64 `json:"-"`
		Raw    map[string]any     `json:"output"`
	}
	type result struct {
		build   string
		ours    map[string]float64
		sources []string
		pob     map[string]float64
		err     error
	}
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 64<<20), 64<<20)
	var results []result
	for sc.Scan() {
		var r row
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		pobOut := map[string]float64{}
		for _, s := range sliceStats {
			if v, ok := r.Raw[s].(float64); ok {
				pobOut[s] = v
			}
		}
		res := result{build: r.Build, pob: pobOut}
		raw, err := os.ReadFile(filepath.Join(corpusDir, r.Build))
		if err == nil {
			var build *pob2.PathOfBuilding2
			if build, err = pob2.ParseBuild(raw); err == nil {
				var env *Environment
				if env, err = BuildOutput(build, Options{Tree: tree}); err == nil {
					res.ours = env.Player.Output
					if os.Getenv("POE2_SETUP_WORKLIST") != "" {
						for _, stat := range sliceStats {
							for _, m := range env.ModDB.Mods[stat] {
								if v := m.Value(); v != nil && v.Type() == mod.ModValueMultiTypeFloat {
									res.sources = append(res.sources, fmt.Sprintf("%s\t%s\t%s\t%s\t%g", r.Build, stat, m.GetSource(), m.Type(), v.Float()))
								}
							}
						}
					}
				}
			}
		}
		res.err = err
		results = append(results, res)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	var w *bufio.Writer
	if p := os.Getenv("POE2_SETUP_WORKLIST"); p != "" {
		out, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		defer out.Close()
		w = bufio.NewWriter(out)
		defer w.Flush()
		fmt.Fprintf(w, "build\tstat\tpob\tours\n")
		srcOut, err := os.Create(p + ".sources")
		if err != nil {
			t.Fatal(err)
		}
		defer srcOut.Close()
		sw := bufio.NewWriter(srcOut)
		defer sw.Flush()
		fmt.Fprintf(sw, "build\tstat\tsource\ttype\tvalue\n")
		for _, r := range results {
			for _, l := range r.sources {
				fmt.Fprintln(sw, l)
			}
		}
	}
	failed := 0
	agree := map[string]int{}
	for _, r := range results {
		if r.err != nil {
			failed++
			continue
		}
		for _, s := range sliceStats {
			pv, ov := r.pob[s], r.ours[s]
			if math.Abs(pv-ov) <= 1e-9*math.Max(1, math.Abs(pv)) {
				agree[s]++
			}
			if w != nil {
				fmt.Fprintf(w, "%s\t%s\t%g\t%g\n", r.build, s, pv, ov)
			}
		}
	}
	keys := append([]string(nil), sliceStats...)
	sort.Strings(keys)
	for _, s := range keys {
		t.Logf("%-8s agree %d / %d", s, agree[s], len(results)-failed)
	}
	t.Logf("builds %d, failed to calculate %d", len(results), failed)
	if failed > 0 {
		for _, r := range results {
			if r.err != nil {
				t.Logf("first failure: %s: %v", r.build, r.err)
				break
			}
		}
	}
}
