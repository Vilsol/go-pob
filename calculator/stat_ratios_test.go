package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

type statRatioResult struct {
	build    string
	exp, got float64
	state    int
}

// TestStatRatios prints the got/PoB ratio for ONE OR MORE stats across the whole graded corpus, CLUSTERED BY EXACT RATIO.
func TestStatRatios(t *testing.T) {
	stat := os.Getenv("STAT")
	if stat != "" && os.Getenv("CORPUS") != "true" {
		t.Fatalf("STAT=%s was set but CORPUS is not \"true\", so this would have skipped SILENTLY "+
			"and exited 0. Run `task stat-ratios STAT=%s`, or set CORPUS=true and pass -v.", stat, stat)
	}
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	statNames := strings.Fields(stat)
	if len(statNames) == 0 {
		t.Skip("set STAT=<name as the oracle table prints it, e.g. MainHand.PhysicalMinBase>")
	}
	seenName := make(map[string]bool, len(statNames))
	uniq := statNames[:0:0]
	for _, s := range statNames {
		if !seenName[s] {
			seenName[s] = true
			uniq = append(uniq, s)
		}
	}
	statNames = uniq

	precision := envInt("PRECISION", 4)
	maxMembers := envInt("MEMBERS", 12)

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("cannot load the oracle blob: %v", err)
	}

	corpusDir, names := corpusNamesForLoadedData(t)

	results := make([][]statRatioResult, len(statNames))
	for s := range statNames {
		results[s] = make([]statRatioResult, len(names))
	}

	prefixOnce := make([]sync.Once, len(statNames))
	prefix := make([]string, len(statNames))
	bare := make([]string, len(statNames))
	prefixKnown := make([]bool, len(statNames))

	runConcurrently(len(names), evalWorkers(), func(i int, cache *EnvironmentCache) {
		rec, ok := oracle[names[i]]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			return
		}
		xml, err := os.ReadFile(filepath.Join(corpusDir, names[i]))
		if err != nil {
			return
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			return
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil {
			return
		}
		pairs := oracleTablePairs(rec, env)

		for s, name := range statNames {
			prefixOnce[s].Do(func() {
				order := make([]string, 0, len(pairs))
				for _, p := range pairs {
					order = append(order, p.prefix)
				}
				sort.Slice(order, func(a, b int) bool { return len(order[a]) > len(order[b]) })
				for _, p := range order {
					if p != "" && strings.HasPrefix(name, p) {
						prefix[s], bare[s], prefixKnown[s] = p, strings.TrimPrefix(name, p), true
						return
					}
				}
				prefix[s], bare[s], prefixKnown[s] = "", name, true
			})

			var pair *oracleTablePair
			for _, p := range pairs {
				if p.prefix == prefix[s] {
					pair = &p
					break
				}
			}
			if pair == nil || pair.exp == nil {
				continue
			}
			exp, hasExp := pair.exp[bare[s]]
			g, hasGot := 0.0, false
			if pair.got != nil {
				g, hasGot = pair.got[bare[s]]
			}

			r := statRatioResult{build: strings.TrimSuffix(names[i], ".xml"), exp: exp, got: g}
			switch {
			case !hasExp && !hasGot:
				r.state = 0
			case !hasExp:
				r.state = 4
			case math.Abs(g-exp) < 1e-4 || (exp != 0 && math.Abs((g-exp)/exp) < 0.005):
				r.state = 1
			case exp != 0 && g == 0:
				r.state = 2
			default:
				r.state = 3
			}
			results[s][i] = r
		}
	})

	for s, name := range statNames {
		if !prefixKnown[s] {
			t.Fatalf("no build produced a table for %q — is the stat name right?", name)
		}
		reportStatRatios(t, name, prefix[s], bare[s], results[s], corpusDir, precision, maxMembers)
	}
}

func reportStatRatios(t *testing.T, stat, prefix, bare string, results []statRatioResult,
	corpusDir string, precision, maxMembers int,
) {
	t.Helper()

	var nOK, nMiss, nExtra int
	byRatio := map[string][]statRatioResult{}
	var unratioed []statRatioResult
	for _, r := range results {
		switch r.state {
		case 1:
			nOK++
		case 2:
			nMiss++
		case 4:
			nExtra++
		case 3:
			if r.exp == 0 {
				unratioed = append(unratioed, r)
				continue
			}
			key := strconv.FormatFloat(r.got/r.exp, 'f', precision, 64)
			byRatio[key] = append(byRatio[key], r)
		}
	}

	clusters := make([]string, 0, len(byRatio))
	for k := range byRatio {
		clusters = append(clusters, k)
	}
	sort.Slice(clusters, func(a, b int) bool {
		if len(byRatio[clusters[a]]) != len(byRatio[clusters[b]]) {
			return len(byRatio[clusters[a]]) > len(byRatio[clusters[b]])
		}
		return clusters[a] < clusters[b]
	})

	nDiff := 0
	for _, v := range byRatio {
		nDiff += len(v)
	}

	fmt.Printf("\n=== STAT RATIOS: %s (table %q, key %q) ===\n", stat, prefix, bare)
	fmt.Printf("CORPUS %s (game data %s)\n", corpusDir, raw.LatestVersion)
	fmt.Printf("ok %d · WRONG-NONZERO %d in %d ratio clusters · miss(PoB!=0, ours=0) %d · extra(ours only) %d · PoB=0-but-wrong %d\n",
		nOK, nDiff, len(clusters), nMiss, nExtra, len(unratioed))
	fmt.Printf("Clustered at %d decimal places (PRECISION=n to change).\n\n", precision)

	if len(clusters) == 0 {
		fmt.Printf("No nonzero-vs-nonzero divergence. If `miss` above is large the defect is a\n" +
			"STARVED PRODUCER, which is a different bug from a wrong factor — chase that instead.\n")
	}

	fmt.Printf("%10s %6s   %s\n", "RATIO", "BUILDS", "MEMBERS")
	for _, c := range clusters {
		members := byRatio[c]
		sort.Slice(members, func(a, b int) bool { return members[a].build < members[b].build })
		shown := members
		if maxMembers > 0 && len(shown) > maxMembers {
			shown = shown[:maxMembers]
		}
		names := make([]string, 0, len(shown))
		for _, m := range shown {
			names = append(names, m.build)
		}
		more := ""
		if len(members) > len(shown) {
			more = fmt.Sprintf(" … +%d more", len(members)-len(shown))
		}
		fmt.Printf("%10s %6d   %s%s\n", c, len(members), strings.Join(names, ", "), more)
		if len(members) > 0 {
			m := members[0]
			fmt.Printf("%10s %6s   e.g. %s  PoB %.6g  ours %.6g\n", "", "", m.build, m.exp, m.got)
		}
	}

	if len(clusters) > 0 {
		biggest := byRatio[clusters[0]]
		fmt.Printf("\nREAD IT LIKE THIS: a cluster of %d builds at exactly %s is ONE root cause, and the\n"+
			"ratio usually names it — 0.05 is 1/20, 0.8 is 1/1.25, 2.0 is something counted twice.\n"+
			"A long tail of one-build ratios is many causes, or float noise, and is NOT a worklist.\n",
			len(biggest), clusters[0])
	}
	if len(unratioed) > 0 {
		fmt.Printf("\n%d builds have PoB=0 with a nonzero value from us — no ratio exists for those;\n"+
			"they are an EXTRA-value defect and TestKeySetCensus is the instrument for them.\n", len(unratioed))
	}

	if class := os.Getenv("CLASS"); class != "" {
		lc := strings.ToLower(class)
		wanted, ok := map[string]int{"ok": 1, "miss": 2, "wrong": 3, "extra": 4, "zero": 3}[lc]
		if !ok {
			t.Fatalf("REFUSING: CLASS=%q is not one of ok|miss|wrong|extra|zero.\n"+
				"  miss  = PoB nonzero, ours exactly 0 (a STARVED PRODUCER)\n"+
				"  wrong = both nonzero and they differ (a wrong FACTOR)\n"+
				"  extra = ours only — PoB never emits the key\n"+
				"  zero  = PoB emits 0 and ours is nonzero (an OVER-APPLICATION)\n"+
				"Those are different defects with different fixes; naming one is the point.", class)
		}
		var picked []statRatioResult
		for _, r := range results {
			if r.state != wanted {
				continue
			}
			if lc == "zero" && r.exp != 0 {
				continue
			}
			if lc == "wrong" && r.exp == 0 {
				continue
			}
			picked = append(picked, r)
		}
		sort.Slice(picked, func(a, b int) bool { return picked[a].build < picked[b].build })
		limit := envInt("LIMIT", 40)
		fmt.Printf("\n=== CLASS=%s for %s — %d build(s) ===\n", strings.ToLower(class), stat, len(picked))
		if len(picked) == 0 {
			fmt.Printf("(none — this class is EMPTY for %s, which is a measurement, not a failure)\n", stat)
		}
		for i, r := range picked {
			if limit > 0 && i >= limit {
				fmt.Printf("… +%d more (LIMIT=0 for all)\n", len(picked)-i)
				break
			}
			fmt.Printf("  %-44s PoB %-16.8g ours %.8g\n", r.build, r.exp, r.got)
		}
	}

	fmt.Printf("\nSPLIT AGAIN (it scrolled): ok %d · WRONG-NONZERO %d · miss(PoB!=0, ours=0) %d · extra(ours only) %d · PoB=0-but-wrong %d\n",
		nOK, nDiff, nMiss, nExtra, len(unratioed))
	if nMiss > nDiff {
		fmt.Printf("⚠️  miss EXCEEDS wrong-nonzero: the dominant defect is a STARVED PRODUCER, not a wrong\n" +
			"    factor. Chase what fails to write the value; the ratio clusters above are the minority.\n" +
			"    `CLASS=miss` lists those builds.\n")
	}
}

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
