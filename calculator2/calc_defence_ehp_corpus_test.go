package calculator2

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// TestCorpusPoE2AllOutputsWorklist writes, for every numeric output of the skills-off oracle, a per-stat
// tally (stats.tsv: seen, ok, nonzero, nonzeroOK) and, for the stats matching ALL_OUTPUTS_STATS (a
// regexp), one row per build (pairs.tsv: build, stat, pob, ours, ok). ALL_OUTPUTS_WORKLIST=<dir> runs it.
func TestCorpusPoE2AllOutputsWorklist(t *testing.T) {
	dir := os.Getenv("ALL_OUTPUTS_WORKLIST")
	if os.Getenv("CORPUS") != "true" || dir == "" {
		t.Skip("CORPUS=true ALL_OUTPUTS_WORKLIST=<dir> reads the local PoE2 corpus")
	}
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5-skillsoff")
	oraclePath := filepath.Join("..", "testdata", "oracle-poe2-0.5", "output-poe2-0.5-skillsoff.jsonl.gz")
	if _, err := os.Stat(oraclePath); err != nil {
		t.Skipf("no PoE2 skills-off oracle at %s", oraclePath)
	}
	statsRE := regexp.MustCompile(os.Getenv("ALL_OUTPUTS_STATS"))
	tree := loadTestTree(t)
	recs, err := loadPoE2Oracle(oraclePath)
	if err != nil {
		t.Fatal(err)
	}

	type pair struct {
		build, stat string
		pob, ours   float64
		ok          bool
	}
	var mu sync.Mutex
	all := map[string]*poe2Tally{}
	var pairs []pair
	failed := 0
	tasks := make(chan int, len(recs))
	for i := range recs {
		tasks <- i
	}
	close(tasks)
	var wg sync.WaitGroup
	for range min(poe2Workers(), len(recs)) {
		wg.Go(func() {
			for i := range tasks {
				var env *Environment
				raw, err := os.ReadFile(filepath.Join(corpusDir, recs[i].Build))
				if err == nil {
					var build *pob2.PathOfBuilding2
					if build, err = pob2.ParseBuild(raw); err == nil {
						func() {
							defer func() {
								if r := recover(); r != nil {
									err = fmt.Errorf("panic: %v", r)
								}
							}()
							env, err = BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
						}()
					}
				}
				mu.Lock()
				if err != nil {
					failed++
					mu.Unlock()
					continue
				}
				for key, v := range recs[i].Output {
					want, ok := v.(float64)
					if !ok {
						continue
					}
					got := env.Player.Output[key]
					hit := !math.IsNaN(got) && !math.IsInf(got, 0) && poe2OracleClose(got, want, poe2OracleTolerance)
					tl := all[key]
					if tl == nil {
						tl = &poe2Tally{}
						all[key] = tl
					}
					tl.seen++
					if hit {
						tl.ok++
					}
					if want != 0 {
						tl.nonzero++
						if hit {
							tl.nonzeroOK++
						}
					}
					if statsRE.MatchString(key) {
						pairs = append(pairs, pair{recs[i].Build, key, want, got, hit})
					}
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	writeTSV := func(name string, fill func(w *bufio.Writer)) {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		w := bufio.NewWriter(f)
		fill(w)
		if err := w.Flush(); err != nil {
			t.Fatal(err)
		}
	}
	writeTSV("stats.tsv", func(w *bufio.Writer) {
		fmt.Fprintf(w, "stat\tseen\tok\tnonzero\tnonzeroOK\n")
		for _, k := range keys {
			a := all[k]
			fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\n", k, a.seen, a.ok, a.nonzero, a.nonzeroOK)
		}
	})
	sort.Slice(pairs, func(a, b int) bool {
		if pairs[a].stat != pairs[b].stat {
			return pairs[a].stat < pairs[b].stat
		}
		return pairs[a].build < pairs[b].build
	})
	writeTSV("pairs.tsv", func(w *bufio.Writer) {
		fmt.Fprintf(w, "build\tstat\tpob\tours\tok\n")
		for _, p := range pairs {
			fmt.Fprintf(w, "%s\t%s\t%v\t%v\t%v\n", p.build, p.stat, p.pob, p.ours, p.ok)
		}
	})
	fmt.Printf("ALL_OUTPUTS_WORKLIST: %d builds, %d failed, %d stats, %d pairs written to %s\n", len(recs), failed, len(keys), len(pairs), dir)
	if failed > 0 {
		t.Errorf("%d builds failed to calculate", failed)
	}
}
