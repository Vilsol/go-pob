package calculator2

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

// poe2BreakdownMatchedFloor is the (build, stat) breakdowns from the Calcs tab calculation that agree with the dumps,
// and poe2BreakdownExtraCeiling the ones we write that PoB does not, over the whole corpus.
const (
	poe2BreakdownMatchedFloor = 1336650
	poe2BreakdownExtraCeiling = 0
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

func renderBreakdowns(env *Environment) map[string][]string {
	out := map[string][]string{}
	if bd := env.Player.Breakdown; bd != nil {
		for k, e := range bd.Entries {
			if lines := renderBreakdownDump(e); len(lines) > 0 {
				out[k] = lines
			}
		}
	}
	return out
}

// TestCorpusPoE2BreakdownsVsDumps grades every player breakdown PoB-PoE2's CALCS pass writes, line for line as
// the dumper renders it, against ours from the Calcs tab calculation, and prints the headline again for the main
// calculation. Report only. BREAKDOWN_WORKLIST=<file> writes the disagreeing (build, stat) pairs; LIMIT=<n> takes the
// first n builds. BREAKDOWN_DUMPS=<dir> reads the dumps from there instead.
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
		main map[string][]string
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
					var env, calcsEnv *Environment
					func() {
						defer func() {
							if r := recover(); r != nil {
								err = fmt.Errorf("panic: %v", r)
							}
						}()
						env, err = BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
						if err == nil {
							calcsEnv = BuildCalcsOutput(env, BuildOutputUsed(env))
						}
					}()
					if err != nil {
						res.err = err
						return res
					}
					res.main = renderBreakdowns(env)
					res.ours = renderBreakdowns(calcsEnv)
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
	failed, matched, total, extra, reordered := 0, 0, 0, 0, 0
	mainMatched, mainExtra := 0, 0
	for i, r := range results {
		if r.err != nil {
			failed++
			continue
		}
		for k, want := range r.pob {
			if got, ok := r.main[k]; ok && strings.Join(got, "\n") == strings.Join(want, "\n") {
				mainMatched++
			}
		}
		for k := range r.main {
			if _, ok := r.pob[k]; !ok {
				mainExtra++
			}
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
			} else if ok && sameLines(got, want) {
				reordered++
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
	t.Logf("matched %d / %d (build, stat) pairs PoB writes; %d we write that PoB does not; %d more hold PoB's lines in another order", matched, total, extra, reordered)
	t.Logf("main calculation: matched %d / %d; %d we write that PoB does not", mainMatched, total, mainExtra)
	if limit, _ := strconv.Atoi(os.Getenv("LIMIT")); limit == 0 && os.Getenv("BREAKDOWN_DUMPS") == "" {
		if matched < poe2BreakdownMatchedFloor {
			t.Errorf("matched %d breakdowns, below floor %d", matched, poe2BreakdownMatchedFloor)
		} else if matched > poe2BreakdownMatchedFloor {
			fmt.Printf("matched %d is above the floor %d — raise poe2BreakdownMatchedFloor\n", matched, poe2BreakdownMatchedFloor)
		}
		if extra > poe2BreakdownExtraCeiling {
			t.Errorf("%d breakdowns PoB does not write, above ceiling %d", extra, poe2BreakdownExtraCeiling)
		}
	}
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
	if os.Getenv("BREAKDOWN_CENSUS") == "" {
		return
	}
	// BREAKDOWN_CENSUS=1: the keys PoB writes and we never do, by the builds carrying them
	var left []string
	for k, s := range stats {
		if s.ours == 0 && s.extra == 0 {
			left = append(left, k)
		}
	}
	sort.Slice(left, func(i, j int) bool {
		if stats[left[i]].pob != stats[left[j]].pob {
			return stats[left[i]].pob > stats[left[j]].pob
		}
		return left[i] < left[j]
	})
	carriers := 0
	for _, k := range left {
		carriers += stats[k].pob
	}
	t.Logf("=== %d keys we never write, %d of %d carriers ===", len(left), carriers, total)
	for _, k := range left {
		t.Logf("  left %-40s %d", k, stats[k].pob)
	}
}

// TestCorpusPoE2ModSectionsVsLua grades every CalcSections.lua mod table against PoB-PoE2's AddModSection over its
// main calculation, as oracle/poe2_mod_sections.lua writes it per build into MODSECTION_DUMPS=<dir>. Report only;
// MODSECTION_WORKLIST=<file> writes the disagreeing (build, entry) pairs.
func TestCorpusPoE2ModSectionsVsLua(t *testing.T) {
	dir := os.Getenv("MODSECTION_DUMPS")
	if os.Getenv("CORPUS") != "true" || dir == "" {
		t.Skip("CORPUS=true MODSECTION_DUMPS=<dir> grades the mod tables")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("no mod table dumps at %s", dir)
	}
	corpusDir := filepath.Join("..", "testdata", "corpus-poe2-0.5")
	tree := loadTestTree(t)
	type oracleEntry struct {
		Multi bool   `json:"multi"`
		Error string `json:"error"`
		Rows  []struct {
			Value      string `json:"value"`
			Name       string `json:"name"`
			Flags      string `json:"flags"`
			Tags       string `json:"tags"`
			Source     string `json:"source"`
			SourceName string `json:"sourceName"`
			Totals     []any  `json:"totals"`
		} `json:"rows"`
	}
	// a tooltip line AddLine was handed as a number is the number as tostring gives it
	lines := func(totals []any) []string {
		if totals == nil {
			return nil
		}
		out := make([]string, len(totals))
		for i, v := range totals {
			if f, ok := v.(float64); ok {
				out[i] = luaToString(f)
			} else {
				out[i] = fmt.Sprint(v)
			}
		}
		return out
	}
	var w *bufio.Writer
	if p := os.Getenv("MODSECTION_WORKLIST"); p != "" {
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		w = bufio.NewWriter(f)
		defer w.Flush()
	}
	builds, failed, matched, reordered, total, rowsTotal, luaErrors := 0, 0, 0, 0, 0, 0, 0
	byEntry := map[string][2]int{}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var oracle map[string]oracleEntry
		if err := json.Unmarshal(raw, &oracle); err != nil {
			t.Fatal(err)
		}
		builds++
		xml, err := os.ReadFile(filepath.Join(corpusDir, name+".xml"))
		if err != nil {
			t.Fatal(err)
		}
		build, err := pob2.ParseBuild(xml)
		if err != nil {
			t.Fatal(err)
		}
		env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
		if err != nil {
			failed++
			continue
		}
		for k, o := range oracle {
			if o.Error != "" {
				luaErrors++
				continue
			}
			var want []ModSectionRow
			for _, r := range o.Rows {
				want = append(want, ModSectionRow{Value: r.Value, Name: r.Name, Flags: r.Flags, Tags: r.Tags, Source: r.Source, SourceName: r.SourceName, Totals: lines(r.Totals)})
			}
			p := strings.Split(k, "|")
			got := env.ModSection(ModSectionData{ModName: strings.Split(p[0], ","), Multi: o.Multi, ModType: p[1], Cfg: p[2], ModSource: p[3], Enemy: p[4] == "true", Actor: p[5]})
			for i := range got {
				got[i].ItemID, got[i].NodeID = 0, 0
			}
			if len(got) == 0 {
				got = nil
			}
			if len(want) == 0 && len(got) == 0 {
				continue
			}
			total++
			rowsTotal += len(want)
			s := byEntry[k]
			s[1]++
			if reflect.DeepEqual(got, want) {
				matched++
				s[0]++
			} else if sameRows(got, want) {
				reordered++
			} else if w != nil {
				g, _ := json.Marshal(got)
				wa, _ := json.Marshal(want)
				fmt.Fprintf(w, "%s\t%s\n\tpob:  %s\n\tours: %s\n", name, k, wa, g)
			}
			byEntry[k] = s
		}
	}
	t.Logf("=== CORPUS PoE2 0.5 Calcs mod tables vs PoB-PoE2 (main calculation) ===")
	t.Logf("%d builds, %d failed, %d entries PoB errors on", builds, failed, luaErrors)
	t.Logf("matched %d / %d (build, entry) tables with rows on either side (%d PoB rows); %d more hold PoB's rows in another order", matched, total, rowsTotal, reordered)
	keys := make([]string, 0, len(byEntry))
	for k, s := range byEntry {
		if s[0] < s[1] {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return byEntry[keys[i]][1]-byEntry[keys[i]][0] > byEntry[keys[j]][1]-byEntry[keys[j]][0]
	})
	for _, k := range keys {
		t.Logf("  %-60s matched %d / %d", k, byEntry[k][0], byEntry[k][1])
	}
}

// sameLines is got and want holding the same lines in any order: a row list built over pairs() follows LuaJIT's
// table order.
func sameLines(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := map[string]int{}
	for _, l := range want {
		seen[l]++
	}
	for _, l := range got {
		if seen[l] == 0 {
			return false
		}
		seen[l]--
	}
	return true
}

// sameRows is got and want holding the same rows in any order: table.sort leaves the rows its comparator ties in an
// order that follows the modDB's, which follows pairs() over the passive tree.
func sameRows(got, want []ModSectionRow) bool {
	if len(got) != len(want) {
		return false
	}
	key := func(r ModSectionRow) string { b, _ := json.Marshal(r); return string(b) }
	seen := map[string]int{}
	for _, r := range want {
		seen[key(r)]++
	}
	for _, r := range got {
		if seen[key(r)] == 0 {
			return false
		}
		seen[key(r)]--
	}
	return true
}
