package calculator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestModOrderCensus is the third census level and the only harness in the project that can see ORDER.
//
//	CONTENT — the multisets differ. Counted, reported as a total, then set aside. Not the finding.
func TestModOrderCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census; `task census WHICH=order`")
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	dumps := dumpsDirForLoadedData(t)

	corpusDir := corpusDirForLoadedData(t)

	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	limited := false
	if n := os.Getenv("LIMIT"); n != "" {
		if l, convErr := strconv.Atoi(n); convErr == nil && l < len(names) {
			names = names[:l]
			limited = true
		}
	}

	results := make([]orderResult, len(names))

	workers := evalWorkers()
	t.Logf("order-censusing %d builds across %d workers", len(names), workers)

	runConcurrently(len(names), workers, func(i int, cache *EnvironmentCache) {
		file, readErr := os.ReadFile(filepath.Join(corpusDir, names[i]))
		if readErr != nil {
			return
		}
		build, parseErr := builds.ParseBuild(file)
		if parseErr != nil {
			return
		}
		raw, dumpErr := os.ReadFile(filepath.Join(dumps, strings.TrimSuffix(names[i], ".xml")+".json"))
		if dumpErr != nil {
			return
		}
		var dump struct {
			ModDB      map[string][]censusDumpMod `json:"modDB"`
			EnemyModDB map[string][]censusDumpMod `json:"enemyModDB"`
		}
		if json.Unmarshal(raw, &dump) != nil {
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

		res := &results[i]
		res.build = strings.TrimSuffix(names[i], ".xml")
		res.hasDump = true
		res.treeSources = countDumpTreeSources(dump.ModDB)
		res.player = orderCompare(dump.ModDB, env.Player.ModDB.Mods)
		if dump.EnemyModDB != nil {
			res.enemyOK = true
			res.enemy = orderCompare(dump.EnemyModDB, env.Enemy.ModDB.Mods)
		}
	})

	var withDump, excluded int
	for i := range results {
		if !results[i].hasDump {
			continue
		}
		withDump++
		if results[i].treeSources < oracleMinTreeSources {
			excluded++
			results[i].excluded = true
		}
	}
	if withDump == 0 {
		t.Fatalf("no build in %s produced a comparable dump — wrong DUMPS directory?", dumps)
	}
	t.Logf("builds with a usable dump: %d, excluded for a broken passive tree: %d", withDump, excluded)
	if want := oracleExcludedForLoadedData(t); !limited && excluded != want {
		t.Fatalf("broken-tree exclusions = %d, want %d: the dump set changed and this baseline is "+
			"no longer comparable to the oracle's", excluded, want)
	}

	unstable := loadPinUnstableOrders(t)
	player := reportOrderCensus(t, results, false, "PLAYER", unstable)
	enemy := reportOrderCensus(t, results, true, "ENEMY", unstable)
	if !limited {
		var stale []string
		for k, seen := range unstable {
			if !seen {
				stale = append(stale, k)
			}
		}
		sort.Strings(stale)
		if len(stale) > 0 {
			t.Errorf("%d pairs in testdata/oracle/mod_order_pin_unstable.tsv no longer diverge; remove them:\n%s",
				len(stale), strings.Join(stale, "\n"))
		}
	}

	if !limited {
		if player > modOrderCrossStageCeiling {
			t.Errorf("PLAYER cross-stage order divergences = %d, ceiling %d — a pipeline stage "+
				"moved relative to PoB's call order", player, modOrderCrossStageCeiling)
		}
		if enemy > modOrderCrossStageCeiling {
			t.Errorf("ENEMY cross-stage order divergences = %d, ceiling %d — a pipeline stage "+
				"moved relative to PoB's call order", enemy, modOrderCrossStageCeiling)
		}
	}
}

const modOrderCrossStageCeiling = 0

type orderResult struct {
	build         string
	hasDump       bool
	enemyOK       bool
	excluded      bool
	treeSources   int
	player, enemy orderSide
}

type orderSide struct {
	content int
	match   int
	orders  []orderDivergence
}

type orderDivergence struct {
	name                 string
	perm                 []int
	pobStages, ourStages string
	crossStage           bool
	movedIn              string
}

func countDumpTreeSources(dumped map[string][]censusDumpMod) int {
	seen := map[string]bool{}
	for _, mods := range dumped {
		for _, m := range mods {
			s := normalizeSource(m.Source)
			if sourceKind(s) == "Tree" {
				seen[s] = true
			}
		}
	}
	return len(seen)
}

func orderToken(typ, src string, val float64) string {
	q := math.Round(val*1e6) / 1e6
	return typ + "|" + src + "|" + strconv.FormatFloat(q, 'g', -1, 64)
}

func orderCompare(dumped map[string][]censusDumpMod, ourMods map[string][]mod.Mod) orderSide {
	pob := map[string][]string{}
	for name, mods := range dumped {
		seq := make([]string, 0, len(mods))
		for _, m := range mods {
			seq = append(seq, orderToken(m.Type, normalizeSource(m.Source), pobModValue(m.Type, m.Value)))
		}
		pob[name] = seq
	}

	ours := map[string][]string{}
	for name, mods := range ourMods {
		seq := make([]string, 0, len(mods))
		for _, m := range mods {
			seq = append(seq, orderToken(string(m.Type()), normalizeSource(string(m.GetSource())), ourModValue(m)))
		}
		ours[name] = seq
	}

	var side orderSide
	for name, p := range pob {
		o, ok := ours[name]
		if !ok {
			side.content++
			continue
		}
		if !sameMultiset(p, o) {
			side.content++
			continue
		}
		perm := permutationOf(p, o)
		if isIdentity(perm) {
			side.match++
			continue
		}
		d := orderDivergence{
			name:      name,
			perm:      perm,
			pobStages: collapseStages(stagesOf(p)),
			ourStages: collapseStages(stagesOf(o)),
		}
		d.crossStage = d.pobStages != d.ourStages
		if !d.crossStage {
			d.movedIn = movedStages(stagesOf(o), perm)
		}
		side.orders = append(side.orders, d)
	}
	for name := range ours {
		if _, ok := pob[name]; !ok {
			side.content++
		}
	}
	return side
}

func sameMultiset(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, v := range a {
		counts[v]++
	}
	for _, v := range b {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}
	return true
}

func permutationOf(pob, ours []string) []int {
	queues := make(map[string][]int, len(pob))
	for i, v := range pob {
		queues[v] = append(queues[v], i)
	}
	perm := make([]int, len(ours))
	for j, v := range ours {
		q := queues[v]
		perm[j] = q[0]
		queues[v] = q[1:]
	}
	return perm
}

func isIdentity(perm []int) bool {
	for i, v := range perm {
		if i != v {
			return false
		}
	}
	return true
}

func orderStage(normalizedSrc string) string {
	switch k := sourceKind(normalizedSrc); k {
	case "Tree", "TreeCluster":
		return "Tree"
	case "Other":
		return normalizedSrc
	default:
		return k
	}
}

func stagesOf(seq []string) []string {
	out := make([]string, len(seq))
	for i, v := range seq {
		parts := strings.SplitN(v, "|", 3)
		if len(parts) >= 2 {
			out[i] = orderStage(parts[1])
		}
	}
	return out
}

func movedStages(stages []string, perm []int) string {
	seen := map[string]bool{}
	for j, v := range perm {
		if j != v {
			seen[stages[j]] = true
		}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return strings.Join(out, "+")
}

func formatPerm(perm []int) string {
	parts := make([]string, len(perm))
	for i, v := range perm {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

func collapseStages(stages []string) string {
	out := make([]string, 0, len(stages))
	for i, k := range stages {
		if i == 0 || stages[i-1] != k {
			out = append(out, k)
		}
	}
	return strings.Join(out, ",")
}

func loadPinUnstableOrders(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile("../testdata/oracle/mod_order_pin_unstable.tsv")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	out := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 4 {
			t.Fatalf("REFUSING: malformed line %q", line)
		}
		out[f[0]+"\t"+f[1]+"\t"+f[2]] = false
	}
	return out
}

func reportOrderCensus(t *testing.T, results []orderResult, enemy bool, actor string, unstable map[string]bool) int {
	t.Helper()

	type shapeAgg struct {
		n       int
		example string
	}
	bump := func(m map[string]*shapeAgg, key, example string) {
		a := m[key]
		if a == nil {
			a = &shapeAgg{example: example}
			m[key] = a
		}
		a.n++
	}

	crossName := map[string]*shapeAgg{}
	crossNameInst := map[string]int{}
	crossPerm := map[string]*shapeAgg{}
	stageShapes := map[string]*shapeAgg{}
	withinStage := map[string]*shapeAgg{}

	var buildCount, orderPairs, contentPairs, matchPairs int
	var crossPairs, withinPairs, buildsWithCross, unstablePairs int

	for i := range results {
		r := &results[i]
		if !r.hasDump || r.excluded {
			continue
		}
		if enemy && !r.enemyOK {
			continue
		}
		side := r.player
		if enemy {
			side = r.enemy
		}
		buildCount++
		contentPairs += side.content
		matchPairs += side.match
		orderPairs += len(side.orders)

		cross := false
		for _, d := range side.orders {
			if !d.crossStage {
				withinPairs++
				bump(withinStage, d.movedIn, r.build+" / "+d.name)
				continue
			}
			if _, ok := unstable[actor+"\t"+r.build+"\t"+d.name]; ok {
				unstable[actor+"\t"+r.build+"\t"+d.name] = true
				unstablePairs++
				continue
			}
			cross = true
			crossPairs++
			if f := os.Getenv("ORDER_LIST"); f != "" {
				if fh, err := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
					fmt.Fprintf(fh, "%s\t%s\t%s\n", actor, r.build, d.name)
					fh.Close()
				}
			}
			bump(crossName, d.name, r.build)
			crossNameInst[d.name] += len(d.perm)
			bump(crossPerm, d.name+"  ["+formatPerm(d.perm)+"]", r.build)
			bump(stageShapes, d.pobStages+"  ->  "+d.ourStages, r.build+" / "+d.name)
		}
		if cross {
			buildsWithCross++
		}
	}

	total := orderPairs + contentPairs + matchPairs
	fmt.Printf("\n=== %s MOD-ORDER CENSUS (%d builds compared) ===\n", actor, buildCount)
	fmt.Printf("Per (build, mod name), bucketed. Order is only judged where the multiset already matches.\n")
	if total == 0 {
		fmt.Printf("nothing to compare\n")
		return 0
	}
	fmt.Printf("  MATCH   %8d pairs (%.1f%%)  same multiset, same sequence\n",
		matchPairs, 100*float64(matchPairs)/float64(total))
	fmt.Printf("  ORDER   %8d pairs (%.1f%%)  same multiset, DIFFERENT sequence\n",
		orderPairs, 100*float64(orderPairs)/float64(total))
	fmt.Printf("  CONTENT %8d pairs (%.1f%%)  multisets differ; set aside, see TestModDBCensus\n",
		contentPairs, 100*float64(contentPairs)/float64(total))

	fmt.Printf("\n  ORDER split by whether a PIPELINE STAGE moved:\n")
	fmt.Printf("    CROSS-STAGE  %8d pairs   <- THE MEASUREMENT: a stage sits on the wrong side of another\n", crossPairs)
	fmt.Printf("    PIN-UNSTABLE %8d pairs      cross-stage, excluded: the pin's own order moves with allocation\n", unstablePairs)
	fmt.Printf("                                   (listed in testdata/oracle/mod_order_pin_unstable.tsv)\n")
	fmt.Printf("    WITHIN-STAGE %8d pairs      stage sequence intact; see table D — the Tree rows are PoB's\n", withinPairs)
	fmt.Printf("                                   LuaJIT hash order and mean nothing, the Item/Skill rows are real\n")
	fmt.Printf("    builds carrying at least one CROSS-STAGE divergence: %d of %d\n", buildsWithCross, buildCount)
	fmt.Printf("    distinct cross-stage mod names: %d   permutation shapes: %d   stage transpositions: %d\n",
		len(crossName), len(crossPerm), len(stageShapes))

	type row struct {
		key string
		n   int
		ex  string
	}
	printRows := func(title, head string, rows []row, n int) {
		if len(rows) == 0 {
			return
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].n != rows[j].n {
				return rows[i].n > rows[j].n
			}
			return rows[i].key < rows[j].key
		})
		shown := rows
		if len(shown) > n {
			shown = shown[:n]
		}
		fmt.Printf("\n  %s\n  %s\n", title, head)
		for _, r := range shown {
			fmt.Printf("  %6d  %s\n", r.n, r.key)
			if r.ex != "" {
				fmt.Printf("          e.g. %s\n", r.ex)
			}
		}
	}
	rowsOf := func(m map[string]*shapeAgg, decorate func(string) string) []row {
		out := make([]row, 0, len(m))
		for k, v := range m {
			key := k
			if decorate != nil {
				key = decorate(k)
			}
			out = append(out, row{key: key, n: v.n, ex: v.example})
		}
		return out
	}

	printRows(fmt.Sprintf("TABLE A: %s CROSS-STAGE divergences by mod name (top 30 of %d)", actor, len(crossName)),
		"builds  MOD NAME",
		rowsOf(crossName, func(k string) string {
			return fmt.Sprintf("%-46s %6d mods in sequence", k, crossNameInst[k])
		}), 30)

	printRows(fmt.Sprintf("TABLE B: %s CROSS-STAGE exact permutation shapes (top 30 of %d)", actor, len(crossPerm)),
		"builds  MOD NAME  [permutation taking PoB's sequence to ours]",
		rowsOf(crossPerm, nil), 30)

	printRows(fmt.Sprintf("TABLE C: %s STAGE TRANSPOSITIONS, name-independent (top 30 of %d)", actor, len(stageShapes)),
		"pairs   POB STAGE SEQUENCE  ->  OURS",
		rowsOf(stageShapes, nil), 30)

	printRows(fmt.Sprintf("TABLE D: %s WITHIN-STAGE reorders by stage moved (of %d)", actor, len(withinStage)),
		"pairs   STAGE(S) CONTAINING A MOVED MOD",
		rowsOf(withinStage, nil), 20)

	return crossPairs
}
