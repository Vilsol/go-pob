package calculator

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/internal/oraclebin"
)

const oracleMinTreeSources = 5

type oracleStatKind uint8

const (
	oracleOK oracleStatKind = iota
	oracleMissing
	oracleDiff
	oracleNaN
)

type oracleStatResult struct {
	stat     string
	exp, got float64
	kind     oracleStatKind
}

type oracleStatAgg struct {
	seen, ok, missing, mismatch, nan int
	nzSeen, nzOK                     int
	worstRel, worstExp, worstGot     float64
	worstBuild                       string

	discreteVals  map[float64]struct{}
	discreteBroke bool

	finite []oracleExemplar
}

type oracleExemplar struct {
	exp, got float64
	build    string
}

type oracleRecord = oraclebin.Record

type oracleTablePair struct {
	prefix   string
	exp, got map[string]float64
}

func oracleTablePairs(rec oracleRecord, env *Environment) []oracleTablePair {
	var minionOut, minionMain, minionOff, enemyOut map[string]float64
	if env.Minion != nil && env.Minion.Actor != nil {
		minionOut = env.Minion.Output
		minionMain = env.Minion.OutputTable[OutTableMainHand]
		minionOff = env.Minion.OutputTable[OutTableOffHand]
	}
	if env.Enemy != nil {
		enemyOut = env.Enemy.Output
	}
	return []oracleTablePair{
		{"", rec.Output, env.Player.Output},
		{"MainHand.", rec.MainHand, env.Player.OutputTable[OutTableMainHand]},
		{"OffHand.", rec.OffHand, env.Player.OutputTable[OutTableOffHand]},
		{"Enemy.", rec.EnemyOutput, enemyOut},
		{"Minion.", rec.Minion, minionOut},
		{"Minion.MainHand.", rec.MinionMain, minionMain},
		{"Minion.OffHand.", rec.MinionOff, minionOff},
	}
}

func sortedKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func loadOutputOracleUncached(path string) (map[string]oracleRecord, error) {
	return oraclebin.Load(path)
}

func reportOracleStats(agg map[string]*oracleStatAgg, included, built, matched, total, excluded, nanStats, nanBuilds int) {
	names := make([]string, 0, len(agg))
	for s := range agg {
		names = append(names, s)
	}
	sort.Strings(names)

	type row struct {
		stat string
		a    *oracleStatAgg
	}
	ordered := make([]row, 0, len(names))
	for _, s := range names {
		ordered = append(ordered, row{s, agg[s]})
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		wi := ordered[i].a.missing + ordered[i].a.mismatch + ordered[i].a.nan
		wj := ordered[j].a.missing + ordered[j].a.mismatch + ordered[j].a.nan
		if wi != wj {
			return wi > wj
		}
		if ordered[i].a.seen != ordered[j].a.seen {
			return ordered[i].a.seen > ordered[j].a.seen
		}
		return ordered[i].stat < ordered[j].stat
	})

	pct := 0.0
	if total > 0 {
		pct = 100 * float64(matched) / float64(total)
	}
	fmt.Printf("\n=== CORPUS 3.18 vs PINNED-POB ORACLE ===\n")
	fmt.Printf("%d builds included (%d excluded: <%d tree sources in the dump), %d calculated\n",
		included, excluded, oracleMinTreeSources, built)
	fmt.Printf("MATCHED %d / %d (%.1f%%) oracle stats across %d distinct keys\n",
		matched, total, pct, len(names))
	fmt.Printf("NaN/Inf: %d stats in %d builds (counted as mismatches, reported separately)\n",
		nanStats, nanBuilds)
	fmt.Printf("RANK WORK BY nzOK/nzSeen, NOT ok/seen: a 0-vs-0 agreement is classified ok, so a\n" +
		"stat PoB reports as 0 on most builds looks far healthier than it is.\n")
	fmt.Printf("TYPICAL is the MEDIAN-RATIO mismatch and is the one to read for direction and\n" +
		"magnitude. MAXREL is the maximum RELATIVE divergence: whenever a single build has\n" +
		"PoB=0 the ratio is infinite and that build wins the column outright, so MAXREL is an\n" +
		"outlier locator, NOT a representative example. Reading it as one has inverted a\n" +
		"cluster's diagnosis before.\n")
	fmt.Printf("%-42s %6s %6s %6s %6s %6s %7s %6s   %-40s %s\n",
		"STAT", "seen", "ok", "miss", "diff", "nan", "nzSeen", "nzOK",
		"TYPICAL median(PoB -> got)", "MAXREL(PoB -> got)")
	for _, r := range ordered {
		a := r.a
		if a.missing+a.mismatch+a.nan == 0 {
			continue
		}
		worst := ""
		if a.worstBuild != "" {
			worst = fmt.Sprintf("%.4g -> %.4g [%s]", a.worstExp, a.worstGot,
				strings.TrimSuffix(a.worstBuild, ".xml"))
		}
		fmt.Printf("%-42s %6d %6d %6d %6d %6d %7d %6d   %-40s %s\n",
			r.stat, a.seen, a.ok, a.missing, a.mismatch, a.nan, a.nzSeen, a.nzOK,
			medianExemplar(a.finite), worst)
	}

	var nzTotal, nzMatched int
	for _, a := range agg {
		nzTotal += a.nzSeen
		nzMatched += a.nzOK
	}
	nzPct := 0.0
	if nzTotal > 0 {
		nzPct = 100 * float64(nzMatched) / float64(nzTotal)
	}
	fmt.Printf("NONZERO-ONLY: %d / %d (%.1f%%) — excludes every stat PoB reports as 0\n",
		nzMatched, nzTotal, nzPct)

	reportDiscreteStats(agg)
}

func reportDiscreteStats(agg map[string]*oracleStatAgg) {
	type row struct {
		stat   string
		agg    *oracleStatAgg
		misses int
	}
	rows := make([]row, 0, 16)
	for stat, a := range agg {
		misses := a.nzSeen - a.nzOK
		if misses > 0 && a.isDiscrete() {
			rows = append(rows, row{stat, a, misses})
		}
	}
	if len(rows) == 0 {
		return
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].misses != rows[j].misses {
			return rows[i].misses > rows[j].misses
		}
		return rows[i].stat < rows[j].stat
	})

	fmt.Printf("\nDISCRETE STATS (small-integer, few distinct values) — HIGH SIGNAL PER ROW.\n")
	fmt.Printf("These are worth reading BEFORE the ranking above: a stat that can only be 1, 2 or 3\n")
	fmt.Printf("tells you WHAT broke, where a DPS row six orders of magnitude wide only tells you\n")
	fmt.Printf("that something did.\n")
	fmt.Printf("%-44s %7s %7s  %s\n", "STAT", "nzSeen", "nzMiss", "PoB values seen")
	for _, r := range rows {
		vals := make([]float64, 0, len(r.agg.discreteVals))
		for v := range r.agg.discreteVals {
			vals = append(vals, v)
		}
		sort.Float64s(vals)
		parts := make([]string, 0, len(vals))
		for _, v := range vals {
			parts = append(parts, strconv.FormatFloat(v, 'g', -1, 64))
		}
		fmt.Printf("%-44s %7d %7d  {%s}\n", r.stat, r.agg.nzSeen, r.misses, strings.Join(parts, " "))
	}
}

const (
	discreteMaxDistinct = 8
	discreteMaxAbs      = 64
)

func (a *oracleStatAgg) noteDiscrete(exp float64) {
	if a.discreteBroke {
		return
	}
	if exp != math.Trunc(exp) || math.Abs(exp) > discreteMaxAbs || math.IsNaN(exp) {
		a.discreteBroke = true
		a.discreteVals = nil
		return
	}
	if a.discreteVals == nil {
		a.discreteVals = map[float64]struct{}{}
	}
	a.discreteVals[exp] = struct{}{}
	if len(a.discreteVals) > discreteMaxDistinct {
		a.discreteBroke = true
		a.discreteVals = nil
	}
}

func (a *oracleStatAgg) isDiscrete() bool {
	return !a.discreteBroke && len(a.discreteVals) > 1
}

func medianExemplar(finite []oracleExemplar) string {
	if len(finite) == 0 {
		return "(all mismatches have PoB=0)"
	}
	s := make([]oracleExemplar, len(finite))
	copy(s, finite)
	sort.SliceStable(s, func(i, j int) bool {
		return s[i].got/s[i].exp < s[j].got/s[j].exp
	})
	m := s[len(s)/2]
	return fmt.Sprintf("%.4g -> %.4g [%s]", m.exp, m.got, strings.TrimSuffix(m.build, ".xml"))
}

const oracleDumpTolerance = 0.005

func oracleClose(got, want, tolerance float64) bool {
	if want == 0 {
		return math.Abs(got) < 1e-9
	}

	return math.Abs(got-want)/math.Abs(want) <= tolerance
}

var oracleCache sync.Map

type oracleCacheEntry struct {
	once sync.Once
	recs map[string]oracleRecord
	err  error
}

func loadOutputOracle(path string) (map[string]oracleRecord, error) {
	v, _ := oracleCache.LoadOrStore(path, &oracleCacheEntry{})
	e := v.(*oracleCacheEntry)
	e.once.Do(func() {
		e.recs, e.err = loadOutputOracleUncached(path)
	})
	return e.recs, e.err
}
