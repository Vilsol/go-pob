package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds/parsecache"
)

// TestCorpus3_29VsOracle is THE AUTHORITATIVE ACCURACY GATE, since 2026-08-09.
func TestCorpus3_29VsOracle(t *testing.T) {
	gradeCorpus3_29(t, oracleDumpTolerance)
}

// TestOracle329TightTolerance grades the same pairs at oracleTightTolerance.
func TestOracle329TightTolerance(t *testing.T) {
	gradeCorpus3_29(t, oracleTightTolerance)
}

func gradeCorpus3_29(t *testing.T, tolerance float64) {
	t.Helper()
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}

	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v\nBuild it with `task oracle-corpus` after a dump.", err)
	}

	corpusDir, names := corpusNamesForLoadedData(t)
	guardPicks := loadGuardPicks(t)
	pairsPicks := loadPairsPicks(t)

	if len(names) < 8300 || len(oracle) < 8300 {
		t.Fatalf("REFUSING: %d builds and %d oracle records, expected >=8300 of each. "+
			"A short corpus or a truncated blob would produce a confident, meaningless percentage.",
			len(names), len(oracle))
	}

	type result struct {
		built, noOracle, excluded bool
		matched, total, nan       int
		stats                     []oracleStatResult
	}
	results := make([]result, len(names))

	workers := evalWorkers()
	t.Logf("grading %d builds against the 3.29 oracle across %d workers; %d guard picks pinned to the oracle's (guard-picks.tsv), %d builds with pairs() picks (pairs-picks.tsv)", len(names), workers, len(guardPicks), len(pairsPicks))

	runConcurrently(len(names), workers, func(i int, cache *EnvironmentCache) {
		res := &results[i]
		rec, ok := oracle[names[i]]
		if !ok {
			res.noOracle = true
			return
		}
		if rec.TreeSources < oracleMinTreeSources {
			res.excluded = true
			return
		}

		file, err := os.ReadFile(filepath.Join(corpusDir, names[i]))
		if err != nil {
			return
		}
		build, err := parsecache.ParseCached(file)
		if err != nil {
			return
		}

		cache.guardPick = guardPicks[names[i]]
		cache.pairsPicks = pairsPicks[names[i]]
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		cache.guardPick = ""
		cache.pairsPicks = nil
		if env == nil {
			return
		}
		res.built = true

		for _, pair := range oracleTablePairs(rec, env) {
			for _, k := range sortedKeys(pair.exp) {
				want := pair.exp[k]
				got := pair.got[k]
				res.total++
				st := oracleStatResult{stat: pair.prefix + k, exp: want, got: got}
				switch {
				case math.IsNaN(got) || math.IsInf(got, 0):
					res.nan++
					st.kind = oracleNaN
				case oracleClose(got, want, tolerance):
					res.matched++
					st.kind = oracleOK
				default:
					st.kind = oracleDiff
				}
				res.stats = append(res.stats, st)
			}
		}
	})

	var (
		built, noOracle, excluded int
		matched, total, nan       int
		perStat                   = map[string]*statTally{}
		worstBuilds               []buildTally
		nanPairs                  []string
	)
	for i, r := range results {
		switch {
		case r.noOracle:
			noOracle++
		case r.excluded:
			excluded++
		}
		if !r.built {
			continue
		}
		built++
		matched += r.matched
		total += r.total
		nan += r.nan
		if wrong := r.total - r.matched; wrong > 0 {
			worstBuilds = append(worstBuilds, buildTally{names[i], wrong, r.total})
		}
		for _, st := range r.stats {
			tally, ok := perStat[st.stat]
			if !ok {
				tally = &statTally{}
				perStat[st.stat] = tally
			}
			tally.seen++
			if st.kind == oracleOK {
				tally.ok++
			}
			if st.kind == oracleNaN {
				tally.nan++
				nanPairs = append(nanPairs, fmt.Sprintf("%s\t%s", names[i], st.stat))
			}
		}
	}

	pct := 0.0
	if total > 0 {
		pct = 100 * float64(matched) / float64(total)
	}
	tight := tolerance != oracleDumpTolerance
	if tight {
		fmt.Printf("\n=== CORPUS 3.29 vs ORACLE 3.29 at relative %g  (NOT the gate — this asserts a miss ceiling) ===\n", tolerance)
	} else {
		fmt.Printf("\n=== CORPUS 3.29 vs ORACLE 3.29  (AUTHORITATIVE — this asserts a floor) ===\n")
	}
	fmt.Printf("%d graded, %d excluded (treeSources < %d), %d with no oracle record\n",
		built, excluded, oracleMinTreeSources, noOracle)
	fmt.Printf("matched %d / %d  (%.4f%%)   NaN/Inf on our side: %d\n", matched, total, pct, nan)
	if nan > 0 {
		fmt.Print(formatNaNPairs(nanPairs, nanPairsPrintCap))
	}

	if tight {
		if wrong := total - matched; wrong > oracle329TightMissCeiling {
			t.Errorf("%d pairs outside relative %g, ceiling %d — a sub-tolerance drift returned",
				wrong, tolerance, oracle329TightMissCeiling)
		} else if wrong < oracle329TightMissCeiling {
			fmt.Printf("tight misses %d are below the ceiling %d — lower oracle329TightMissCeiling\n", wrong, oracle329TightMissCeiling)
		}
	} else if matched < oracle329MatchedFloor {
		t.Errorf("matched %d oracle stats, below floor %d — accuracy regressed by %d",
			matched, oracle329MatchedFloor, oracle329MatchedFloor-matched)
	}
	if nan > oracle329NaNCeiling {
		t.Errorf("%d NaN/Inf stats, ceiling %d — a poisoned value returned\n%s",
			nan, oracle329NaNCeiling, formatNaNPairs(nanPairs, nanPairsPrintCap))
	}

	type row struct {
		stat string
		t    *statTally
	}
	rows := make([]row, 0, len(perStat))
	for s, tl := range perStat {
		if tl.seen-tl.ok > 0 {
			rows = append(rows, row{s, tl})
		}
	}
	sort.Slice(rows, func(a, b int) bool {
		mi, mj := rows[a].t.seen-rows[a].t.ok, rows[b].t.seen-rows[b].t.ok
		if mi != mj {
			return mi > mj
		}
		return rows[a].stat < rows[b].stat
	})
	var neverMiss, partialMiss, neverStats, partialStats int
	for _, tl := range perStat {
		miss := tl.seen - tl.ok
		if miss == 0 {
			continue
		}
		if tl.ok == 0 {
			neverMiss += miss
			neverStats++
		} else {
			partialMiss += miss
			partialStats++
		}
	}
	wrong := total - matched
	fmt.Printf("\nMISS SPLIT — these are different defects, do not add them:\n")
	fmt.Printf("  NEVER right (ok==0)   %8d pairs in %4d stat(s)  %5.1f%% of all misses  <- unported / new-in-3.29\n",
		neverMiss, neverStats, 100*float64(neverMiss)/float64(max(1, wrong)))
	fmt.Printf("  SOMETIMES right       %8d pairs in %4d stat(s)  %5.1f%%                <- wrong arithmetic\n",
		partialMiss, partialStats, 100*float64(partialMiss)/float64(max(1, wrong)))
	fmt.Printf("  if every NEVER-right stat were ported perfectly: %.4f%%\n",
		100*float64(matched+neverMiss)/float64(total))

	fmt.Printf("\nTOP 40 STATS BY MISSES (of %d stats with any miss)\n", len(rows))
	fmt.Printf("  %-40s %8s %8s %8s\n", "stat", "seen", "ok", "miss")
	for _, r := range rows[:min(40, len(rows))] {
		fmt.Printf("  %-40s %8d %8d %8d\n", r.stat, r.t.seen, r.t.ok, r.t.seen-r.t.ok)
	}

	if dir := os.Getenv("WORKLIST"); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
		var b strings.Builder
		b.WriteString("stat\tseen\tok\tmiss\tnan\tclass\n")
		for _, r := range rows {
			class := "partial"
			if r.t.ok == 0 {
				class = "never"
			}
			fmt.Fprintf(&b, "%s\t%d\t%d\t%d\t%d\t%s\n",
				r.stat, r.t.seen, r.t.ok, r.t.seen-r.t.ok, r.t.nan, class)
		}
		_ = os.WriteFile(filepath.Join(dir, "stats.tsv"), []byte(b.String()), 0o600)

		var bb strings.Builder
		bb.WriteString("build\twrong\ttotal\tnan\ttopstats\n")
		for i, r := range results {
			if !r.built || r.total-r.matched == 0 {
				continue
			}
			var miss []string
			for _, st := range r.stats {
				if st.kind != oracleOK {
					miss = append(miss, st.stat)
				}
			}
			sort.Strings(miss)
			if len(miss) > 400 {
				miss = miss[:400]
			}
			fmt.Fprintf(&bb, "%s\t%d\t%d\t%d\t%s\n", names[i], r.total-r.matched, r.total, r.nan, strings.Join(miss, ","))
		}
		_ = os.WriteFile(filepath.Join(dir, "builds.tsv"), []byte(bb.String()), 0o600)

		var pb strings.Builder
		pb.WriteString("build\tstat\texp\tgot\tgot/exp\n")
		for i, r := range results {
			for _, st := range r.stats {
				if st.kind != oracleOK {
					fmt.Fprintf(&pb, "%s\t%s\t%.17g\t%.17g\t%.17g\n", names[i], st.stat, st.exp, st.got, st.got/st.exp)
				}
			}
		}
		_ = os.WriteFile(filepath.Join(dir, "pairs.tsv"), []byte(pb.String()), 0o600)

		_ = os.WriteFile(filepath.Join(dir, "totals.tsv"), []byte(worklistTotalsTSV(perStat)), 0o600)
		fmt.Printf("\nWORKLIST written to %s (stats.tsv, builds.tsv, pairs.tsv, totals.tsv)\n", dir)
	}

	sort.Slice(worstBuilds, func(a, b int) bool { return worstBuilds[a].wrong > worstBuilds[b].wrong })
	fmt.Printf("\nTOP 20 BUILDS BY WRONG PAIRS (of %d builds with any)\n", len(worstBuilds))
	for _, b := range worstBuilds[:min(20, len(worstBuilds))] {
		fmt.Printf("  %-52s %6d wrong of %6d\n", b.name, b.wrong, b.total)
	}
}

const oracle329MatchedFloor = 13174499

const oracle329NaNCeiling = 0

const oracleTightTolerance = 1e-9

const oracle329TightMissCeiling = 7

type statTally struct{ seen, ok, nan int }

type buildTally struct {
	name         string
	wrong, total int
}

func worklistTotalsTSV(perStat map[string]*statTally) string {
	var b strings.Builder
	b.WriteString("scope\tseen\tok\tmiss\tnan\n")
	var all statTally
	for _, tl := range perStat {
		all.seen += tl.seen
		all.ok += tl.ok
		all.nan += tl.nan
	}
	fmt.Fprintf(&b, "ALL\t%d\t%d\t%d\t%d\n", all.seen, all.ok, all.seen-all.ok, all.nan)

	byActor := map[string]*statTally{}
	for stat, tl := range perStat {
		scope := "player"
		for _, a := range []string{"MainHand", "OffHand", "Minion", "Enemy", "Parent", "PartyMember"} {
			if strings.HasPrefix(stat, a+".") {
				scope = a + "."
				break
			}
		}
		t, ok := byActor[scope]
		if !ok {
			t = &statTally{}
			byActor[scope] = t
		}
		t.seen += tl.seen
		t.ok += tl.ok
		t.nan += tl.nan
	}
	scopes := make([]string, 0, len(byActor))
	for k := range byActor {
		scopes = append(scopes, k)
	}
	sort.Strings(scopes)
	for _, k := range scopes {
		t := byActor[k]
		fmt.Fprintf(&b, "%s\t%d\t%d\t%d\t%d\n", k, t.seen, t.ok, t.seen-t.ok, t.nan)
	}
	return b.String()
}
