package calculator2

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// TestCorpusPoE2AllStatsTSV writes, per oracle output key, seen / ok / nonzero / nonzeroOK over the
// skills-off corpus to ALL_STATS_TSV, and with ALL_STATS_PAIRS=<stat,...> the per-build pairs of those stats.
func TestCorpusPoE2AllStatsTSV(t *testing.T) {
	out := os.Getenv("ALL_STATS_TSV")
	if os.Getenv("CORPUS") != "true" || out == "" {
		t.Skip("CORPUS=true ALL_STATS_TSV=<file> reads the local PoE2 corpus")
	}
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5-skillsoff")
	oraclePath := filepath.Join("..", "testdata", "oracle-poe2-0.5", "output-poe2-0.5-skillsoff.jsonl.gz")
	if _, err := os.Stat(oraclePath); err != nil {
		t.Skipf("no PoE2 skills-off oracle at %s", oraclePath)
	}
	tree := loadTestTree(t)
	recs, err := loadPoE2Oracle(oraclePath)
	if err != nil {
		t.Fatal(err)
	}
	pairStats := map[string]bool{}
	for _, s := range strings.Split(os.Getenv("ALL_STATS_PAIRS"), ",") {
		if s != "" {
			pairStats[s] = true
		}
	}

	var mu sync.Mutex
	all := map[string]*poe2Tally{}
	var pairs []string
	failed := 0
	tasks := make(chan int, len(recs))
	for i := range recs {
		tasks <- i
	}
	close(tasks)
	var wg sync.WaitGroup
	for range min(poe2Workers(), len(recs)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range tasks {
				raw, err := os.ReadFile(filepath.Join(corpusDir, recs[i].Build))
				var env *Environment
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
					tl := all[key]
					if tl == nil {
						tl = &poe2Tally{}
						all[key] = tl
					}
					tl.seen++
					hit := !math.IsNaN(got) && !math.IsInf(got, 0) && poe2OracleClose(got, want, poe2OracleTolerance)
					if hit {
						tl.ok++
					}
					if want != 0 {
						tl.nonzero++
						if hit {
							tl.nonzeroOK++
						}
					}
					if pairStats[key] && !hit {
						pairs = append(pairs, fmt.Sprintf("%s\t%s\t%.17g\t%.17g", recs[i].Build, key, want, got))
					}
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("stat\tseen\tok\tnonzero\tnonzeroOK\n")
	for _, k := range keys {
		a := all[k]
		fmt.Fprintf(&b, "%s\t%d\t%d\t%d\t%d\n", k, a.seen, a.ok, a.nonzero, a.nonzeroOK)
	}
	if err := os.WriteFile(out, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if len(pairs) > 0 {
		sort.Strings(pairs)
		if err := os.WriteFile(out+".pairs", []byte("build\tstat\texp\tgot\n"+strings.Join(pairs, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d stats written to %s, %d builds failed", len(keys), out, failed)
}
