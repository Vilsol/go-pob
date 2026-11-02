package calculator2

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

var (
	dumpHexColorRe = regexp.MustCompile(`\^[0-9a-fA-F]{6}`)
	dumpDigitRe    = regexp.MustCompile(`\^[0-9]`)
)

// the generator functions the breakdown table also holds, which the dumper prints as "function: 0x..."
var breakdownGenerators = map[string]bool{
	"multiChain": true, "simple": true, "mod": true, "slot": true, "area": true, "effMult": true, "dot": true, "critDot": true, "leech": true,
}

// dumpStripColor is the dumper's stripColor.
func dumpStripColor(s string) string {
	return dumpDigitRe.ReplaceAllString(dumpHexColorRe.ReplaceAllString(s, ""), "")
}

// renderBreakdownDump is the dumper's renderBreakdown: the lines, then a line per slot, then a line per row.
func renderBreakdownDump(e *BreakdownEntry) []string {
	var lines []string
	for _, l := range e.Lines {
		lines = append(lines, dumpStripColor(l))
	}
	for _, s := range e.Slots {
		src := s.SourceName
		if src == "" {
			src = s.Source
		}
		lines = append(lines, fmt.Sprintf("  slot %s: base=%s%s%s total=%s", src, luaToString(s.Base), s.Inc, s.More, s.Total))
	}
	if e.RowList != nil && e.ColList != nil {
		for _, row := range e.RowList {
			var parts []string
			for _, c := range e.ColList {
				if v, ok := row[c.Key]; ok {
					parts = append(parts, c.Key+"="+dumpStripColor(v))
				}
			}
			if len(parts) > 0 {
				lines = append(lines, "  "+strings.Join(parts, " "))
			}
		}
	}
	return lines
}

// TestCorpusPoE2BreakdownsVsDumps grades every player breakdown PoB-PoE2's CALCS pass writes, line for line as
// the dumper renders it, against ours from the main calculation. Report only. BREAKDOWN_WORKLIST=<file> writes
// the disagreeing (build, stat) pairs; LIMIT=<n> takes the first n builds. BREAKDOWN_DUMPS=<dir> reads the dumps
// from there instead, such as the dumper's output with mainEnv's breakdown in place of calcsEnv's, which the main
// calculation is graded against exactly.
func TestCorpusPoE2BreakdownsVsDumps(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 corpus")
	}
	oracleDir := filepath.Join("..", "testdata", "oracle-poe2-0.5")
	realOracle, err := filepath.EvalSymlinks(oracleDir)
	if err != nil {
		t.Skipf("no PoE2 oracle at %s", oracleDir)
	}
	dumpDir := filepath.Join(filepath.Dir(realOracle), "dumps")
	if d := os.Getenv("BREAKDOWN_DUMPS"); d != "" {
		dumpDir = d
	}
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5")
	entries, err := os.ReadDir(dumpDir)
	if err != nil {
		t.Skipf("no PoE2 dumps at %s", dumpDir)
	}
	tree := loadTestTree(t)
	var builds []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			builds = append(builds, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	sort.Strings(builds)
	if limit, _ := strconv.Atoi(os.Getenv("LIMIT")); limit > 0 && len(builds) > limit {
		builds = builds[:limit]
	}

	type result struct {
		err  error
		pob  map[string][]string
		ours map[string][]string
	}
	results := make([]result, len(builds))
	work := make(chan int)
	var wg sync.WaitGroup
	for range poe2Workers() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				results[i] = func() result {
					var res result
					raw, err := os.ReadFile(filepath.Join(dumpDir, builds[i]+".json"))
					if err != nil {
						return result{err: err}
					}
					var d struct {
						Breakdown map[string][]string `json:"breakdown"`
					}
					if err := json.Unmarshal(raw, &d); err != nil {
						return result{err: err}
					}
					res.pob = d.Breakdown
					for k := range breakdownGenerators {
						delete(res.pob, k)
					}
					xml, err := os.ReadFile(filepath.Join(corpusDir, builds[i]+".xml"))
					if err != nil {
						return result{err: err}
					}
					build, err := pob2.ParseBuild(xml)
					if err != nil {
						return result{err: err}
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
						return res
					}
					res.ours = map[string][]string{}
					if bd := env.Player.Breakdown; bd != nil {
						for k, e := range bd.Entries {
							if lines := renderBreakdownDump(e); len(lines) > 0 {
								res.ours[k] = lines
							}
						}
					}
					return res
				}()
			}
		}()
	}
	for i := range builds {
		work <- i
	}
	close(work)
	wg.Wait()

	type stat struct{ pob, ours, ok, extra int }
	stats := map[string]*stat{}
	get := func(k string) *stat {
		if stats[k] == nil {
			stats[k] = &stat{}
		}
		return stats[k]
	}
	var w *bufio.Writer
	if p := os.Getenv("BREAKDOWN_WORKLIST"); p != "" {
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		w = bufio.NewWriter(f)
		defer w.Flush()
	}
	failed, matched, total, extra := 0, 0, 0, 0
	for i, r := range results {
		if r.err != nil {
			failed++
			continue
		}
		for k, want := range r.pob {
			s := get(k)
			s.pob++
			total++
			got, ok := r.ours[k]
			if ok {
				s.ours++
			}
			if ok && strings.Join(got, "\n") == strings.Join(want, "\n") {
				s.ok++
				matched++
			} else if w != nil && ok {
				fmt.Fprintf(w, "%s\t%s\n\tpob:  %q\n\tours: %q\n", builds[i], k, want, got)
			}
		}
		for k, got := range r.ours {
			if _, ok := r.pob[k]; !ok {
				get(k).extra++
				extra++
				if w != nil {
					fmt.Fprintf(w, "%s\t%s\tEXTRA\n\tours: %q\n", builds[i], k, got)
				}
			}
		}
	}
	t.Logf("=== CORPUS PoE2 0.5 player breakdowns vs DUMPS (skills on) ===")
	t.Logf("%d builds, %d failed", len(builds), failed)
	t.Logf("matched %d / %d (build, stat) pairs PoB writes; %d we write that PoB does not", matched, total, extra)
	keys := make([]string, 0, len(stats))
	for k, s := range stats {
		if s.ours > 0 || s.extra > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		s := stats[k]
		t.Logf("  %-40s matched %d / %d   ours %d   extra %d", k, s.ok, s.pob, s.ours, s.extra)
	}
}
