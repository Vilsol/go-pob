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

	"github.com/Vilsol/go-pob/pob2"
)

// defenceSliceStats are the player outputs the PoE2 gate grades.
var defenceSliceStats = []string{
	"Str", "Dex", "Int", "Life", "Mana", "EnergyShield", "Spirit",
	"FireResist", "ColdResist", "LightningResist", "ChaosResist",
	"Armour", "Evasion", "BlockChance", "DeflectChance", "MovementSpeedMod",
	"LifeUnreserved", "ManaUnreserved",
}

const poe2OracleTolerance = 0.005

// poe2OracleClose is the PoE1 gate's tolerance rule: relative, with an absolute floor at zero.
func poe2OracleClose(got, want, tolerance float64) bool {
	if want == 0 {
		return math.Abs(got) < 1e-9
	}
	return math.Abs(got-want)/math.Abs(want) <= tolerance
}

const poe2MatchedFloor = 177900

const poe2NaNCeiling = 0

type poe2OracleRecord struct {
	Build  string         `json:"build"`
	Output map[string]any `json:"output"`
}

func loadPoE2Oracle(path string) ([]poe2OracleRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open oracle: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gunzip oracle: %w", err)
	}
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 64<<20), 64<<20)
	var recs []poe2OracleRecord
	for sc.Scan() {
		var r poe2OracleRecord
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("oracle record %d: %w", len(recs)+1, err)
		}
		recs = append(recs, r)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read oracle: %w", err)
	}
	return recs, nil
}

func poe2Workers() int {
	if v := os.Getenv("WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return runtime.NumCPU()
}

type poe2StatResult struct {
	stat     string
	exp, got float64
	ok, nan  bool
}

type poe2Tally struct{ seen, ok, nan, nonzero, nonzeroOK int }

// TestCorpusPoE2VsOracle grades the defence slice of every skills-off corpus build against PoB-PoE2's
// skills-off oracle and asserts a matched floor and a NaN ceiling.
func TestCorpusPoE2VsOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 corpus")
	}
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5-skillsoff")
	oraclePath := filepath.Join("..", "testdata", "oracle-poe2-0.5", "output-poe2-0.5-skillsoff.jsonl.gz")
	if _, err := os.Stat(oraclePath); err != nil {
		t.Skipf("no PoE2 skills-off oracle at %s", oraclePath)
	}
	if _, err := os.Stat(corpusDir); err != nil {
		t.Skipf("no PoE2 skills-off corpus at %s", corpusDir)
	}
	tree := loadTestTree(t)

	recs, err := loadPoE2Oracle(oraclePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) < 9000 {
		t.Fatalf("REFUSING: %d oracle records, expected >=9000; a truncated oracle gives a meaningless number", len(recs))
	}

	type result struct {
		built        bool
		err          error
		matched, nan int
		total        int
		stats        []poe2StatResult
		// every numeric output the oracle holds, a stat we never produce counting as 0
		allMatched, allTotal, allNonzero, allNonzeroOK int
	}
	results := make([]result, len(recs))
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
				raw, err := os.ReadFile(filepath.Join(corpusDir, recs[i].Build))
				if err != nil {
					res.err = err
					continue
				}
				build, err := pob2.ParseBuild(raw)
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
					st := poe2StatResult{stat: stat, exp: want, got: got}
					res.total++
					switch {
					case math.IsNaN(got) || math.IsInf(got, 0):
						st.nan = true
						res.nan++
					case poe2OracleClose(got, want, poe2OracleTolerance):
						st.ok = true
						res.matched++
					}
					res.stats = append(res.stats, st)
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
							res.allNonzeroOK++
						}
					}
				}
			}
		}()
	}
	wg.Wait()

	var built, failed, matched, total, nan int
	var firstErr string
	perStat := map[string]*poe2Tally{}
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
		for _, st := range r.stats {
			tl := perStat[st.stat]
			if tl == nil {
				tl = &poe2Tally{}
				perStat[st.stat] = tl
			}
			tl.seen++
			if st.ok {
				tl.ok++
			}
			if st.nan {
				tl.nan++
			}
			if st.exp != 0 {
				tl.nonzero++
				if st.ok {
					tl.nonzeroOK++
				}
			}
		}
	}

	fmt.Printf("\n=== CORPUS PoE2 0.5 vs ORACLE PoE2 0.5 (skills off, defence slice) ===\n")
	fmt.Printf("%d graded, %d failed to calculate\n", built, failed)
	fmt.Printf("matched %d / %d  (%.4f%%)   NaN/Inf on our side: %d\n", matched, total, 100*float64(matched)/float64(max(1, total)), nan)
	if firstErr != "" {
		fmt.Printf("first failure: %s\n", firstErr)
	}
	fmt.Printf("  %-20s %8s %8s %8s %10s\n", "stat", "seen", "ok", "nonzero", "nonzeroOK")
	for _, stat := range defenceSliceStats {
		tl := perStat[stat]
		if tl == nil {
			tl = &poe2Tally{}
		}
		fmt.Printf("  %-20s %8d %8d %8d %10d\n", stat, tl.seen, tl.ok, tl.nonzero, tl.nonzeroOK)
	}

	var allMatched, allTotal, allNonzero, allNonzeroOK int
	for _, r := range results {
		allMatched += r.allMatched
		allTotal += r.allTotal
		allNonzero += r.allNonzero
		allNonzeroOK += r.allNonzeroOK
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
	fmt.Printf("\n=== every numeric oracle output (report only; a stat we do not produce counts as 0) ===\n")
	fmt.Printf("matched %d / %d  (%.4f%%)   nonzero matched %d / %d  (%.4f%%)   %d distinct stats\n",
		allMatched, allTotal, 100*float64(allMatched)/float64(max(1, allTotal)),
		allNonzeroOK, allNonzero, 100*float64(allNonzeroOK)/float64(max(1, allNonzero)), len(all))
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		ma, mb := all[keys[a]].nonzero-all[keys[a]].nonzeroOK, all[keys[b]].nonzero-all[keys[b]].nonzeroOK
		if ma != mb {
			return ma > mb
		}
		return keys[a] < keys[b]
	})
	fmt.Printf("  most-missed nonzero stats:\n  %-34s %8s %8s %10s\n", "stat", "seen", "nonzero", "nonzeroOK")
	for _, k := range keys[:min(40, len(keys))] {
		a := all[k]
		fmt.Printf("  %-34s %8d %8d %10d\n", k, a.seen, a.nonzero, a.nonzeroOK)
	}

	if dir := os.Getenv("WORKLIST"); dir != "" {
		writePoE2Worklist(t, dir, recs, perStat, func(i int) (bool, int, int, []poe2StatResult) {
			r := results[i]
			return r.built, r.matched, r.total, r.stats
		})
	}

	if matched < poe2MatchedFloor {
		t.Errorf("matched %d PoE2 oracle stats, below floor %d — accuracy regressed by %d", matched, poe2MatchedFloor, poe2MatchedFloor-matched)
	} else if matched > poe2MatchedFloor {
		fmt.Printf("matched %d is above the floor %d — raise poe2MatchedFloor\n", matched, poe2MatchedFloor)
	}
	if nan > poe2NaNCeiling {
		t.Errorf("%d NaN/Inf stats, ceiling %d", nan, poe2NaNCeiling)
	}
	if failed > 0 {
		t.Errorf("%d builds failed to calculate; the denominator moved (first: %s)", failed, firstErr)
	}
}

func writePoE2Worklist(t *testing.T, dir string, recs []poe2OracleRecord, perStat map[string]*poe2Tally, result func(int) (bool, int, int, []poe2StatResult)) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	stats := make([]string, 0, len(perStat))
	for s, tl := range perStat {
		if tl.seen-tl.ok > 0 {
			stats = append(stats, s)
		}
	}
	sort.Slice(stats, func(a, b int) bool {
		ma, mb := perStat[stats[a]].seen-perStat[stats[a]].ok, perStat[stats[b]].seen-perStat[stats[b]].ok
		if ma != mb {
			return ma > mb
		}
		return stats[a] < stats[b]
	})
	var b strings.Builder
	b.WriteString("stat\tseen\tok\tmiss\tnan\tnonzero\tnonzeroOK\tclass\n")
	for _, s := range stats {
		tl := perStat[s]
		class := "partial"
		if tl.ok == 0 {
			class = "never"
		}
		fmt.Fprintf(&b, "%s\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n", s, tl.seen, tl.ok, tl.seen-tl.ok, tl.nan, tl.nonzero, tl.nonzeroOK, class)
	}
	var bb, pb strings.Builder
	bb.WriteString("build\twrong\ttotal\tstats\n")
	pb.WriteString("build\tstat\texp\tgot\tgot/exp\n")
	for i := range recs {
		built, matched, total, sts := result(i)
		if !built || matched == total {
			continue
		}
		var miss []string
		for _, st := range sts {
			if !st.ok {
				miss = append(miss, st.stat)
				fmt.Fprintf(&pb, "%s\t%s\t%.17g\t%.17g\t%.17g\n", recs[i].Build, st.stat, st.exp, st.got, st.got/st.exp)
			}
		}
		fmt.Fprintf(&bb, "%s\t%d\t%d\t%s\n", recs[i].Build, total-matched, total, strings.Join(miss, ","))
	}
	for name, content := range map[string]string{"stats.tsv": b.String(), "builds.tsv": bb.String(), "pairs.tsv": pb.String()} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("\nWORKLIST written to %s (stats.tsv, builds.tsv, pairs.tsv)\n", dir)
}
