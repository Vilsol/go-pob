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

// offenceSliceStats are main-skill outputs of CalcActiveSkill and CalcOffence. The skills-off oracle
// already runs the main socket group (CalcSetup.lua:2250, :2380 process it whether or not it is
// enabled), so these grade on the skills-off blob.
var offenceSliceStats = []string{
	"TotalDPS", "CombinedDPS", "AverageDamage", "AverageHit", "Speed",
	"HitChance", "AccuracyHitChance", "Accuracy", "CritChance", "CritMultiplier", "CritEffect",
	"ManaCost", "ManaPerSecondCost", "GemLevel", "GemQuality", "Repeats", "DpsMultiplier",
	"HeavyStunBuildupAvg", "IgniteDPS", "AreaOfEffectMod", "ProjectileCount", "CastRate",
	"LifeCost", "Duration", "DurationMod", "ChainMax", "Cooldown", "ActiveTotemLimit", "AreaOfEffectRadius",
	"ProjectileSpeedMod", "ActiveMinionLimit", "AverageBurstHits", "DurationSecondary", "IceCrystalLife",
}

const poe2OffenceMatchedFloor = 95432

const poe2SkillsOnMatchedFloor = 179808

type poe2SliceGrade struct {
	records, built, failed                         int
	firstErr                                       string
	matched, total, nan, nonzero, nonzeroOK        int
	perStat                                        map[string]*poe2Tally
	allMatched, allTotal, allNonzero, allNonzeroOK int
	all                                            map[string]*poe2Tally
	recs                                           []poe2OracleRecord
	builds                                         []poe2SliceBuild
}

type poe2SliceBuild struct {
	built          bool
	matched, total int
	stats          []poe2StatResult
}

// gradePoE2Slice runs every corpus build and grades slice, plus every numeric oracle output (a stat
// we never produce counting as 0), against the oracle blob. It skips when either input is absent.
func gradePoE2Slice(t *testing.T, corpusDir, oraclePath string, slice []string) *poe2SliceGrade {
	t.Helper()
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 corpus")
	}
	if _, err := os.Stat(oraclePath); err != nil {
		t.Skipf("no PoE2 oracle at %s", oraclePath)
	}
	if _, err := os.Stat(corpusDir); err != nil {
		t.Skipf("no PoE2 corpus at %s", corpusDir)
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
		poe2SliceBuild
		err                                            error
		nan                                            int
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
				for _, stat := range slice {
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

	g := &poe2SliceGrade{records: len(recs), perStat: map[string]*poe2Tally{}, all: map[string]*poe2Tally{}, recs: recs}
	for i, r := range results {
		g.builds = append(g.builds, r.poe2SliceBuild)
		if !r.built {
			g.failed++
			if g.firstErr == "" {
				g.firstErr = fmt.Sprintf("%s: %v", recs[i].Build, r.err)
			}
			continue
		}
		g.built++
		g.matched += r.matched
		g.total += r.total
		g.nan += r.nan
		g.allMatched += r.allMatched
		g.allTotal += r.allTotal
		g.allNonzero += r.allNonzero
		g.allNonzeroOK += r.allNonzeroOK
		for _, st := range r.stats {
			tl := g.perStat[st.stat]
			if tl == nil {
				tl = &poe2Tally{}
				g.perStat[st.stat] = tl
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
				g.nonzero++
				if st.ok {
					tl.nonzeroOK++
					g.nonzeroOK++
				}
			}
		}
	}
	for _, m := range allPerStat {
		for k, tl := range m {
			a := g.all[k]
			if a == nil {
				a = &poe2Tally{}
				g.all[k] = a
			}
			a.seen += tl.seen
			a.ok += tl.ok
			a.nonzero += tl.nonzero
			a.nonzeroOK += tl.nonzeroOK
		}
	}
	return g
}

func (g *poe2SliceGrade) printSlice(title string, slice []string) {
	fmt.Printf("\n=== %s ===\n", title)
	fmt.Printf("%d graded, %d failed to calculate\n", g.built, g.failed)
	fmt.Printf("matched %d / %d  (%.4f%%)   nonzero matched %d / %d  (%.4f%%)   NaN/Inf on our side: %d\n",
		g.matched, g.total, 100*float64(g.matched)/float64(max(1, g.total)),
		g.nonzeroOK, g.nonzero, 100*float64(g.nonzeroOK)/float64(max(1, g.nonzero)), g.nan)
	if g.firstErr != "" {
		fmt.Printf("first failure: %s\n", g.firstErr)
	}
	fmt.Printf("  %-20s %8s %8s %8s %10s\n", "stat", "seen", "ok", "nonzero", "nonzeroOK")
	for _, stat := range slice {
		tl := g.perStat[stat]
		if tl == nil {
			tl = &poe2Tally{}
		}
		fmt.Printf("  %-20s %8d %8d %8d %10d\n", stat, tl.seen, tl.ok, tl.nonzero, tl.nonzeroOK)
	}
}

func (g *poe2SliceGrade) printAll(title string) {
	fmt.Printf("\n=== %s ===\n", title)
	fmt.Printf("matched %d / %d  (%.4f%%)   nonzero matched %d / %d  (%.4f%%)   %d distinct stats\n",
		g.allMatched, g.allTotal, 100*float64(g.allMatched)/float64(max(1, g.allTotal)),
		g.allNonzeroOK, g.allNonzero, 100*float64(g.allNonzeroOK)/float64(max(1, g.allNonzero)), len(g.all))
	keys := make([]string, 0, len(g.all))
	for k := range g.all {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		ma, mb := g.all[keys[a]].nonzero-g.all[keys[a]].nonzeroOK, g.all[keys[b]].nonzero-g.all[keys[b]].nonzeroOK
		if ma != mb {
			return ma > mb
		}
		return keys[a] < keys[b]
	})
	fmt.Printf("  most-missed nonzero stats:\n  %-34s %8s %8s %10s\n", "stat", "seen", "nonzero", "nonzeroOK")
	for _, k := range keys[:min(40, len(keys))] {
		a := g.all[k]
		fmt.Printf("  %-34s %8d %8d %10d\n", k, a.seen, a.nonzero, a.nonzeroOK)
	}
}

// check asserts the slice's floor and NaN ceiling and that no build dropped out of the denominator.
func (g *poe2SliceGrade) check(t *testing.T, floor int) {
	t.Helper()
	if dir := os.Getenv("WORKLIST"); dir != "" {
		writePoE2Worklist(t, dir, g.recs, g.perStat, func(i int) (bool, int, int, []poe2StatResult) {
			b := g.builds[i]
			return b.built, b.matched, b.total, b.stats
		})
		keys := make([]string, 0, len(g.all))
		for k := range g.all {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString("stat\tseen\tok\tnonzero\tnonzeroOK\n")
		for _, k := range keys {
			a := g.all[k]
			fmt.Fprintf(&b, "%s\t%d\t%d\t%d\t%d\n", k, a.seen, a.ok, a.nonzero, a.nonzeroOK)
		}
		if err := os.WriteFile(filepath.Join(dir, "all.tsv"), []byte(b.String()), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if g.matched < floor {
		t.Errorf("matched %d PoE2 oracle stats, below floor %d — accuracy regressed by %d", g.matched, floor, floor-g.matched)
	} else if g.matched > floor {
		fmt.Printf("matched %d is above the floor %d — raise the floor\n", g.matched, floor)
	}
	if g.nan > poe2NaNCeiling {
		t.Errorf("%d NaN/Inf stats, ceiling %d", g.nan, poe2NaNCeiling)
	}
	if g.failed > 0 {
		t.Errorf("%d builds failed to calculate; the denominator moved (first: %s)", g.failed, g.firstErr)
	}
}

// TestCorpusPoE2OffenceVsOracle grades the offence slice of every skills-off corpus build against
// PoB-PoE2's skills-off oracle, whose main skill is the build's own.
func TestCorpusPoE2OffenceVsOracle(t *testing.T) {
	g := gradePoE2Slice(t, filepath.Join("..", "testdata", "corpus-poe2-0.5-skillsoff"),
		filepath.Join("..", "testdata", "oracle-poe2-0.5", "output-poe2-0.5-skillsoff.jsonl.gz"), offenceSliceStats)
	g.printSlice("CORPUS PoE2 0.5 vs ORACLE PoE2 0.5 (skills off, offence slice)", offenceSliceStats)
	g.check(t, poe2OffenceMatchedFloor)
}

// TestCorpusPoE2SkillsOnVsOracle grades the defence slice and every numeric output of every skills-on
// corpus build against PoB-PoE2's skills-on oracle, where buffs, auras, curses and reservations apply.
func TestCorpusPoE2SkillsOnVsOracle(t *testing.T) {
	g := gradePoE2Slice(t, filepath.Join("..", "testdata", "corpus-poe2-0.5"),
		filepath.Join("..", "testdata", "oracle-poe2-0.5", "output-poe2-0.5.jsonl.gz"), defenceSliceStats)
	g.printSlice("CORPUS PoE2 0.5 vs ORACLE PoE2 0.5 (skills on, defence slice)", defenceSliceStats)
	g.printAll("skills on: every numeric oracle output (report only; a stat we do not produce counts as 0)")
	g.check(t, poe2SkillsOnMatchedFloor)
}
