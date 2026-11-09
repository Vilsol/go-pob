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

const poe2FlasksMatchedFloor = 738931

const poe2FlasksNonzeroFloor = 386369

type poe2FlaskBuild struct {
	built                bool
	err                  error
	allMatched, allTotal int
	allNonzero, allNzOK  int
	misses               []poe2ConfigMiss
}

// TestCorpusPoE2FlasksVsOracle grades a PoE2 corpus sample with every equipped flask and charm slot
// active (skills on) against PoB-PoE2's oracle for it; the main corpus leaves every slot inactive.
func TestCorpusPoE2FlasksVsOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 flask corpus")
	}
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5-flasks")
	oraclePath := filepath.Join("..", "testdata", "oracle-poe2-0.5-flasks", "output-poe2-0.5-flasks.jsonl.gz")
	for _, p := range []string{oraclePath, corpusDir} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("no PoE2 flask corpus input at %s", p)
		}
	}
	recs, err := loadPoE2Oracle(oraclePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) == 0 {
		t.Fatalf("REFUSING: the flask oracle at %s holds no record", oraclePath)
	}
	tree := loadTestTree(t)

	results := make([]poe2FlaskBuild, len(recs))
	tasks := make(chan int, len(recs))
	for i := range recs {
		tasks <- i
	}
	close(tasks)
	var wg sync.WaitGroup
	workers := min(poe2Workers(), len(recs))
	allPerStat := make([]map[string]*poe2Tally, workers)
	for w := range workers {
		allPerStat[w] = map[string]*poe2Tally{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			mine := allPerStat[w]
			for i := range tasks {
				res := &results[i]
				xml, err := os.ReadFile(filepath.Join(corpusDir, recs[i].Build))
				if err != nil {
					res.err = err
					continue
				}
				build, err := pob2.ParseBuild(xml)
				if err != nil {
					res.err = err
					continue
				}
				var env *Environment
				func() {
					defer func() {
						if r := recover(); r != nil {
							err = fmt.Errorf("panic: %v", r)
						}
					}()
					env, err = BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
				}()
				if err != nil {
					res.err = err
					continue
				}
				res.built = true
				for key, v := range recs[i].Output {
					want, ok := v.(float64)
					if !ok {
						continue
					}
					got := env.Player.Output[key]
					tl := mine[key]
					if tl == nil {
						tl = &poe2Tally{}
						mine[key] = tl
					}
					tl.seen++
					res.allTotal++
					hit := !math.IsNaN(got) && !math.IsInf(got, 0) && poe2OracleClose(got, want, poe2OracleTolerance)
					if hit {
						tl.ok++
						res.allMatched++
					} else {
						res.misses = append(res.misses, poe2ConfigMiss{key, want, got})
					}
					if want != 0 {
						tl.nonzero++
						res.allNonzero++
						if hit {
							tl.nonzeroOK++
							res.allNzOK++
						}
					}
				}
			}
		}()
	}
	wg.Wait()

	var built, failed, allMatched, allTotal, allNonzero, allNzOK int
	var firstErr string
	for i, r := range results {
		if !r.built {
			failed++
			if firstErr == "" {
				firstErr = fmt.Sprintf("%s: %v", recs[i].Build, r.err)
			}
			continue
		}
		built++
		allMatched += r.allMatched
		allTotal += r.allTotal
		allNonzero += r.allNonzero
		allNzOK += r.allNzOK
	}
	all := map[string]*poe2Tally{}
	for _, m := range allPerStat {
		for k, tl := range m {
			a := all[k]
			if a == nil {
				a = &poe2Tally{}
				all[k] = a
			}
			a.seen += tl.seen
			a.ok += tl.ok
			a.nonzero += tl.nonzero
			a.nonzeroOK += tl.nonzeroOK
		}
	}

	pct := func(a, b int) float64 { return 100 * float64(a) / float64(max(1, b)) }
	fmt.Printf("\n=== CORPUS PoE2 0.5 FLASKS vs ORACLE PoE2 0.5 FLASKS (skills on, every flask and charm active) ===\n")
	fmt.Printf("%d graded, %d failed to calculate\n", built, failed)
	if firstErr != "" {
		fmt.Printf("first failure: %s\n", firstErr)
	}
	fmt.Printf("every output:  matched %d / %d  (%.4f%%)   nonzero matched %d / %d  (%.4f%%)   %d distinct stats\n",
		allMatched, allTotal, pct(allMatched, allTotal), allNzOK, allNonzero, pct(allNzOK, allNonzero), len(all))

	keys := make([]string, 0, len(all))
	for k, a := range all {
		if a.ok < a.seen {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := all[keys[i]], all[keys[j]]
		if a.seen-a.ok != b.seen-b.ok {
			return a.seen-a.ok > b.seen-b.ok
		}
		return keys[i] < keys[j]
	})
	fmt.Printf("\n  stats by misses (top 40 of %d with a miss):\n", len(keys))
	for _, k := range keys[:min(40, len(keys))] {
		a := all[k]
		fmt.Printf("  %-44s miss %5d   nonzero matched %5d / %5d\n", k, a.seen-a.ok, a.nonzeroOK, a.nonzero)
	}

	if dir := os.Getenv("WORKLIST"); dir != "" {
		writePoE2FlaskWorklist(t, dir, recs, results, all)
	}
	if allMatched < poe2FlasksMatchedFloor {
		t.Errorf("every-output matched %d, below floor %d — accuracy regressed by %d", allMatched, poe2FlasksMatchedFloor, poe2FlasksMatchedFloor-allMatched)
	} else if allMatched > poe2FlasksMatchedFloor {
		fmt.Printf("every-output matched %d is above the floor %d — raise poe2FlasksMatchedFloor\n", allMatched, poe2FlasksMatchedFloor)
	}
	if allNzOK < poe2FlasksNonzeroFloor {
		t.Errorf("every-output nonzero matched %d, below floor %d — accuracy regressed by %d", allNzOK, poe2FlasksNonzeroFloor, poe2FlasksNonzeroFloor-allNzOK)
	} else if allNzOK > poe2FlasksNonzeroFloor {
		fmt.Printf("every-output nonzero matched %d is above the floor %d — raise poe2FlasksNonzeroFloor\n", allNzOK, poe2FlasksNonzeroFloor)
	}
	if failed > 0 {
		t.Errorf("%d builds failed to calculate; the denominator moved (first: %s)", failed, firstErr)
	}
}

func writePoE2FlaskWorklist(t *testing.T, dir string, recs []poe2OracleRecord, results []poe2FlaskBuild, all map[string]*poe2Tally) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var st, mb strings.Builder
	st.WriteString("stat\tseen\tok\tnonzero\tnonzeroOK\n")
	for _, k := range keys {
		a := all[k]
		fmt.Fprintf(&st, "%s\t%d\t%d\t%d\t%d\n", k, a.seen, a.ok, a.nonzero, a.nonzeroOK)
	}
	mb.WriteString("build\tstat\texp\tgot\n")
	for i, r := range results {
		if !r.built {
			fmt.Fprintf(&mb, "%s\t!failed\t0\t0\n", recs[i].Build)
			continue
		}
		for _, m := range r.misses {
			fmt.Fprintf(&mb, "%s\t%s\t%.17g\t%.17g\n", recs[i].Build, m.stat, m.exp, m.got)
		}
	}
	for name, content := range map[string]string{"all.tsv": st.String(), "misses.tsv": mb.String()} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("\nWORKLIST written to %s (all.tsv, misses.tsv)\n", dir)
}
