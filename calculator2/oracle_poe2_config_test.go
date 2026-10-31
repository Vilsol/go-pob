package calculator2

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

	"github.com/Vilsol/go-pob/pob2"
)

const poe2ConfigMatchedFloor = 35978

const poe2ConfigNonzeroFloor = 757195

type poe2ConfigMiss struct {
	stat     string
	exp, got float64
}

type poe2ConfigBuild struct {
	built                bool
	err                  error
	matched, total, nan  int
	nonzero, nonzeroOK   int
	allNonzero, allNzOK  int
	allMatched, allTotal int
	misses               []poe2ConfigMiss
}

// TestCorpusPoE2ConfigVsOracle grades the config-randomised PoE2 corpus (skills on, each build with a
// seeded set of config inputs) against PoB-PoE2's oracle for it, then attributes nonzero agreement to
// each config var: builds that set it against builds that do not.
func TestCorpusPoE2ConfigVsOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 config corpus")
	}
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5-config")
	oracleDir := filepath.Join("..", "testdata", "oracle-poe2-0.5-config")
	oraclePath := filepath.Join(oracleDir, "output-poe2-0.5-config.jsonl.gz")
	for _, p := range []string{oraclePath, corpusDir, filepath.Join(oracleDir, "configs.tsv"), filepath.Join(oracleDir, "MANIFEST.json")} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("no PoE2 config corpus input at %s", p)
		}
	}
	var manifest struct {
		Dumps struct{ Usable int } `json:"dumps"`
	}
	raw, err := os.ReadFile(filepath.Join(oracleDir, "MANIFEST.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	recs, err := loadPoE2Oracle(oraclePath)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Dumps.Usable == 0 || len(recs) < manifest.Dumps.Usable {
		t.Fatalf("REFUSING: %d oracle records, the manifest dumped %d usable builds; a truncated oracle gives a meaningless number", len(recs), manifest.Dumps.Usable)
	}
	sets, err := loadPoE2ConfigSets(filepath.Join(oracleDir, "configs.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)

	results := make([]poe2ConfigBuild, len(recs))
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
				for _, stat := range defenceSliceStats {
					want, ok := recs[i].Output[stat].(float64)
					if !ok {
						continue
					}
					got := env.Player.Output[stat]
					res.total++
					switch {
					case math.IsNaN(got) || math.IsInf(got, 0):
						res.nan++
					case poe2OracleClose(got, want, poe2OracleTolerance):
						res.matched++
						if want != 0 {
							res.nonzeroOK++
						}
					}
					if want != 0 {
						res.nonzero++
					}
				}
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
					}
					if want != 0 {
						tl.nonzero++
						res.allNonzero++
						if hit {
							tl.nonzeroOK++
							res.allNzOK++
						} else {
							res.misses = append(res.misses, poe2ConfigMiss{key, want, got})
						}
					}
				}
			}
		}()
	}
	wg.Wait()

	var built, failed, matched, total, nan, nonzero, nonzeroOK, allMatched, allTotal, allNonzero, allNzOK int
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
		matched += r.matched
		total += r.total
		nan += r.nan
		nonzero += r.nonzero
		nonzeroOK += r.nonzeroOK
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
	fmt.Printf("\n=== CORPUS PoE2 0.5 CONFIG vs ORACLE PoE2 0.5 CONFIG (skills on, random config inputs) ===\n")
	fmt.Printf("%d graded, %d failed to calculate\n", built, failed)
	if firstErr != "" {
		fmt.Printf("first failure: %s\n", firstErr)
	}
	fmt.Printf("defence slice: matched %d / %d  (%.4f%%)   nonzero matched %d / %d  (%.4f%%)   NaN/Inf on our side: %d\n",
		matched, total, pct(matched, total), nonzeroOK, nonzero, pct(nonzeroOK, nonzero), nan)
	fmt.Printf("every output:  matched %d / %d  (%.4f%%)   nonzero matched %d / %d  (%.4f%%)   %d distinct stats\n",
		allMatched, allTotal, pct(allMatched, allTotal), allNzOK, allNonzero, pct(allNzOK, allNonzero), len(all))

	attr := poe2ConfigAttribution(recs, results, sets)
	fmt.Printf("\n  config vars by nonzero misses in excess of the builds that leave them unset (top 40 of %d set):\n", len(attr))
	fmt.Printf("  %-44s %6s %10s %8s %10s %8s %8s\n", "var", "builds", "setNZ", "set%", "unsetNZ", "unset%", "excess")
	for _, a := range attr[:min(40, len(attr))] {
		fmt.Printf("  %-44s %6d %10d %8.3f %10d %8.3f %8.0f\n", a.name, a.builds, a.setNZ, pct(a.setOK, a.setNZ), a.unsetNZ, pct(a.unsetOK, a.unsetNZ), a.excess)
	}

	if dir := os.Getenv("WORKLIST"); dir != "" {
		writePoE2ConfigWorklist(t, dir, recs, results, all, attr)
	}

	if matched < poe2ConfigMatchedFloor {
		t.Errorf("defence slice matched %d, below floor %d — accuracy regressed by %d", matched, poe2ConfigMatchedFloor, poe2ConfigMatchedFloor-matched)
	} else if matched > poe2ConfigMatchedFloor {
		fmt.Printf("defence slice matched %d is above the floor %d — raise poe2ConfigMatchedFloor\n", matched, poe2ConfigMatchedFloor)
	}
	if allNzOK < poe2ConfigNonzeroFloor {
		t.Errorf("every-output nonzero matched %d, below floor %d — accuracy regressed by %d", allNzOK, poe2ConfigNonzeroFloor, poe2ConfigNonzeroFloor-allNzOK)
	} else if allNzOK > poe2ConfigNonzeroFloor {
		fmt.Printf("every-output nonzero matched %d is above the floor %d — raise poe2ConfigNonzeroFloor\n", allNzOK, poe2ConfigNonzeroFloor)
	}
	if nan > poe2NaNCeiling {
		t.Errorf("%d NaN/Inf defence stats, ceiling %d", nan, poe2NaNCeiling)
	}
	if failed > 0 {
		t.Errorf("%d builds failed to calculate; the denominator moved (first: %s)", failed, firstErr)
	}
}

// loadPoE2ConfigSets reads the generator's build -> {var: value} record.
func loadPoE2ConfigSets(path string) (map[string]map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config sets: %w", err)
	}
	sets := map[string]map[string]string{}
	for i, line := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") {
		if i == 0 {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 4 {
			return nil, fmt.Errorf("config sets row %d has %d fields, want 4", i+1, len(f))
		}
		if sets[f[0]] == nil {
			sets[f[0]] = map[string]string{}
		}
		sets[f[0]][f[1]] = f[3]
	}
	return sets, nil
}

type poe2ConfigVarAttr struct {
	name                                   string
	builds, setNZ, setOK, unsetNZ, unsetOK int
	excess                                 float64
}

// poe2ConfigAttribution ranks each set var by the nonzero misses its builds carry beyond the rate of
// the graded builds that leave it unset.
func poe2ConfigAttribution(recs []poe2OracleRecord, results []poe2ConfigBuild, sets map[string]map[string]string) []*poe2ConfigVarAttr {
	byVar := map[string]*poe2ConfigVarAttr{}
	var nz, ok int
	for i, r := range results {
		if !r.built {
			continue
		}
		nz += r.allNonzero
		ok += r.allNzOK
		for v := range sets[recs[i].Build] {
			a := byVar[v]
			if a == nil {
				a = &poe2ConfigVarAttr{name: v}
				byVar[v] = a
			}
			a.builds++
			a.setNZ += r.allNonzero
			a.setOK += r.allNzOK
		}
	}
	out := make([]*poe2ConfigVarAttr, 0, len(byVar))
	for _, a := range byVar {
		a.unsetNZ, a.unsetOK = nz-a.setNZ, ok-a.setOK
		unsetRate := 1.0
		if a.unsetNZ > 0 {
			unsetRate = float64(a.unsetOK) / float64(a.unsetNZ)
		}
		a.excess = unsetRate*float64(a.setNZ) - float64(a.setOK)
		out = append(out, a)
	}
	sort.Slice(out, func(x, y int) bool {
		if out[x].excess != out[y].excess {
			return out[x].excess > out[y].excess
		}
		return out[x].name < out[y].name
	})
	return out
}

func writePoE2ConfigWorklist(t *testing.T, dir string, recs []poe2OracleRecord, results []poe2ConfigBuild, all map[string]*poe2Tally, attr []*poe2ConfigVarAttr) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var st, vb, mb strings.Builder
	st.WriteString("stat\tseen\tok\tnonzero\tnonzeroOK\n")
	for _, k := range keys {
		a := all[k]
		fmt.Fprintf(&st, "%s\t%d\t%d\t%d\t%d\n", k, a.seen, a.ok, a.nonzero, a.nonzeroOK)
	}
	vb.WriteString("var\tbuilds\tsetNonzero\tsetNonzeroOK\tunsetNonzero\tunsetNonzeroOK\texcess\n")
	for _, a := range attr {
		fmt.Fprintf(&vb, "%s\t%d\t%d\t%d\t%d\t%d\t%.1f\n", a.name, a.builds, a.setNZ, a.setOK, a.unsetNZ, a.unsetOK, a.excess)
	}
	mb.WriteString("build\tstat\texp\tgot\tgot/exp\n")
	for i, r := range results {
		if !r.built {
			fmt.Fprintf(&mb, "%s\t!failed\t0\t0\t0\n", recs[i].Build)
			continue
		}
		for _, m := range r.misses {
			fmt.Fprintf(&mb, "%s\t%s\t%.17g\t%.17g\t%.17g\n", recs[i].Build, m.stat, m.exp, m.got, m.got/m.exp)
		}
	}
	for name, content := range map[string]string{"all.tsv": st.String(), "vars.tsv": vb.String(), "misses.tsv": mb.String()} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("\nWORKLIST written to %s (all.tsv, vars.tsv, misses.tsv)\n", dir)
}
