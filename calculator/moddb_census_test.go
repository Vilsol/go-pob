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

// TestModDBCensus compares the *inputs* of the calculation.
func TestModDBCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census; `task census WHICH=modb`")
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
	if n := os.Getenv("LIMIT"); n != "" {
		if l, convErr := strconv.Atoi(n); convErr == nil && l < len(names) {
			names = names[:l]
		}
	}

	results := make([]censusResult, len(names))

	workers := evalWorkers()
	t.Logf("censusing %d builds across %d workers", len(names), workers)

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
			ModDB       map[string][]censusDumpMod `json:"modDB"`
			EnemyModDB  map[string][]censusDumpMod `json:"enemyModDB"`
			MinionModDB map[string][]censusDumpMod `json:"minionModDB"`
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
		res.ok = true
		res.build = names[i]
		res.player = censusCompare(dump.ModDB, env.Player.ModDB.Mods)
		res.treeSources = countTreeSources(res.player.pobSrc)

		if dump.EnemyModDB != nil {
			res.enemyOK = true
			res.enemy = censusCompare(dump.EnemyModDB, env.Enemy.ModDB.Mods)
		}
		if dump.MinionModDB != nil && env.Minion != nil && env.Minion.ModDB != nil {
			res.minionOK = true
			res.minion = censusCompare(dump.MinionModDB, env.Minion.ModDB.Mods)
		}
	})

	if out := os.Getenv("CENSUS_TSV"); out != "" {
		if err := writeCensusTSV(out, results); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", out)
	}

	player := censusSides(results, false, false)
	reportSourceCensus(player, "PLAYER")
	reportNameCensus(player, "PLAYER")

	enemy := censusSides(results, true, true)
	if len(enemy) == 0 {
		t.Fatalf("no dump in %s carries an \"enemyModDB\" key, so the enemy census would report a "+
			"silent, meaningless \"0 missing\". Re-dump with an oracle/dump_build.lua new enough to "+
			"emit the enemy actor.", dumps)
	}
	reportSourceCensus(enemy, "ENEMY")
	reportNameCensus(enemy, "ENEMY")
}

type censusDumpMod struct {
	Source string          `json:"source"`
	Type   string          `json:"type"`
	Value  json.RawMessage `json:"value"`
}

func censusCompare(dumped map[string][]censusDumpMod, ourMods map[string][]mod.Mod) censusSide {
	ours := map[string][]censusMod{}
	ourSrc := map[string]bool{}
	for name, mods := range ourMods {
		for _, m := range mods {
			src := normalizeSource(string(m.GetSource()))
			ourSrc[src] = true
			ours[name] = append(ours[name], censusMod{typ: string(m.Type()), val: ourModValue(m)})
		}
	}

	pob := map[string][]censusMod{}
	pobSrc := map[string]bool{}
	for name, mods := range dumped {
		for _, m := range mods {
			pobSrc[normalizeSource(m.Source)] = true
			pob[name] = append(pob[name], censusMod{typ: m.Type, val: pobModValue(m.Type, m.Value)})
		}
	}

	var side censusSide
	for s := range pobSrc {
		if !ourSrc[s] {
			side.missSrc = append(side.missSrc, s)
		}
	}
	for s := range ourSrc {
		if !pobSrc[s] {
			side.extraSrc = append(side.extraSrc, s)
		}
		side.ourSrc = append(side.ourSrc, s)
	}
	for s := range pobSrc {
		side.pobSrc = append(side.pobSrc, s)
	}

	seen := map[string]bool{}
	for name := range pob {
		seen[name] = true
	}
	for name := range ours {
		seen[name] = true
	}
	for name := range seen {
		missing, extra := diffMultiset(pob[name], ours[name])
		side.names = append(side.names, nameDelta{
			name:    name,
			missing: missing,
			extra:   extra,
			pobHas:  len(pob[name]) > 0,
			ourHas:  len(ours[name]) > 0,
		})
	}
	return side
}

func countTreeSources(pobSrc []string) int {
	n := 0
	for _, s := range pobSrc {
		if sourceKind(s) == "Tree" {
			n++
		}
	}
	return n
}

func censusSides(results []censusResult, enemy bool, dropBrokenTrees bool) []censusSide {
	out := make([]censusSide, 0, len(results))
	for i := range results {
		r := &results[i]
		if !r.ok {
			continue
		}
		if enemy && !r.enemyOK {
			continue
		}
		if dropBrokenTrees && r.treeSources < oracleMinTreeSources {
			continue
		}
		if enemy {
			out = append(out, r.enemy)
		} else {
			out = append(out, r.player)
		}
	}
	return out
}

type censusMod struct {
	typ string
	val float64
}

type nameDelta struct {
	name           string
	missing, extra int
	pobHas, ourHas bool
}

type censusSide struct {
	pobSrc, ourSrc    []string
	missSrc, extraSrc []string
	names             []nameDelta
}

type censusResult struct {
	ok          bool
	enemyOK     bool
	minionOK    bool
	build       string
	treeSources int
	player      censusSide
	enemy       censusSide
	minion      censusSide
}

func writeCensusTSV(path string, results []censusResult) error {
	var b strings.Builder
	b.WriteString("build\tactor\tkind\tkey\tmissing\textra\n")
	for i := range results {
		r := &results[i]
		if !r.ok {
			continue
		}
		fmt.Fprintf(&b, "%s\t-\t-\t-\t0\t0\n", r.build)
		sides := []struct {
			actor string
			ok    bool
			side  *censusSide
		}{{"player", true, &r.player}, {"minion", r.minionOK, &r.minion}, {"enemy", r.enemyOK, &r.enemy}}
		for _, s := range sides {
			if !s.ok {
				continue
			}
			for _, d := range s.side.names {
				if d.missing > 0 || d.extra > 0 {
					fmt.Fprintf(&b, "%s\t%s\tname\t%s\t%d\t%d\n", r.build, s.actor, d.name, d.missing, d.extra)
				}
			}
			for _, src := range s.side.missSrc {
				fmt.Fprintf(&b, "%s\t%s\tsrc\t%s\t1\t0\n", r.build, s.actor, src)
			}
			for _, src := range s.side.extraSrc {
				fmt.Fprintf(&b, "%s\t%s\tsrc\t%s\t0\t1\n", r.build, s.actor, src)
			}
		}
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("writing census rows: %w", err)
	}
	return nil
}

func normalizeSource(s string) string {
	switch {
	case s == "":
		return "Other:<none>"
	case strings.HasPrefix(s, "Tree:"), strings.HasPrefix(s, "Tree") && len(s) > len("Tree"):
		rest := strings.TrimPrefix(strings.TrimPrefix(s, "Tree"), ":")
		if i := strings.IndexAny(rest, " ("); i >= 0 {
			rest = rest[:i]
		}
		rest = strings.TrimSpace(rest)
		if id, err := strconv.Atoi(rest); err == nil && id >= 65536 {
			return "TreeCluster:" + rest
		}
		return "Tree:" + rest
	case strings.HasPrefix(s, "Item:"):
		rest := s[len("Item:"):]
		if i := strings.Index(rest, ":"); i >= 0 {
			rest = rest[i+1:]
		}
		return "Item:" + rest
	}
	if i := strings.Index(s, ":"); i > 0 {
		return s
	}
	return "Other:" + s
}

func sourceKind(key string) string {
	if i := strings.Index(key, ":"); i > 0 {
		return key[:i]
	}
	return key
}

func ourModValue(m mod.Mod) float64 {
	v := m.Value()
	if v == nil {
		return 0
	}
	switch v.Type() {
	case mod.ModValueMultiTypeFloat:
		return v.Float()
	case mod.ModValueMultiTypeFlag:
		if v.Flag() {
			return 1
		}
		return 0
	}
	return 0
}

func pobModValue(typ string, raw json.RawMessage) float64 {
	if typ == "LIST" {
		return 0
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return f
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		if b {
			return 1
		}
		return 0
	}
	return 0
}

func diffMultiset(pob, ours []censusMod) (int, int) {
	var missing, extra int
	byType := func(mods []censusMod) map[string][]float64 {
		out := map[string][]float64{}
		for _, m := range mods {
			out[m.typ] = append(out[m.typ], m.val)
		}
		for _, v := range out {
			sort.Float64s(v)
		}
		return out
	}
	p, o := byType(pob), byType(ours)
	types := map[string]bool{}
	for k := range p {
		types[k] = true
	}
	for k := range o {
		types[k] = true
	}
	for typ := range types {
		a, b := p[typ], o[typ]
		i, j := 0, 0
		for i < len(a) && j < len(b) {
			switch {
			case censusNear(a[i], b[j]):
				i, j = i+1, j+1
			case a[i] < b[j]:
				i++
				missing++
			default:
				j++
				extra++
			}
		}
		missing += len(a) - i
		extra += len(b) - j
	}
	return missing, extra
}

func censusNear(a, b float64) bool {
	return math.Abs(a-b) < 1e-4 || (b != 0 && math.Abs((a-b)/b) < 0.005)
}

func reportSourceCensus(sides []censusSide, actor string) {
	type kindAgg struct {
		pobKeys, ourKeys        map[string]bool
		missBuilds, extraBuilds int
		missInst, extraInst     int
	}
	kinds := map[string]*kindAgg{}
	kindOf := func(k string) *kindAgg {
		a := kinds[k]
		if a == nil {
			a = &kindAgg{pobKeys: map[string]bool{}, ourKeys: map[string]bool{}}
			kinds[k] = a
		}
		return a
	}
	missBuilds := map[string]int{}
	extraBuilds := map[string]int{}

	buildCount := len(sides)
	for _, r := range sides {
		for _, s := range r.pobSrc {
			kindOf(sourceKind(s)).pobKeys[s] = true
		}
		for _, s := range r.ourSrc {
			kindOf(sourceKind(s)).ourKeys[s] = true
		}
		seenKind := map[string]bool{}
		for _, s := range r.missSrc {
			missBuilds[s]++
			a := kindOf(sourceKind(s))
			a.missInst++
			if !seenKind["m"+sourceKind(s)] {
				seenKind["m"+sourceKind(s)] = true
				a.missBuilds++
			}
		}
		for _, s := range r.extraSrc {
			extraBuilds[s]++
			a := kindOf(sourceKind(s))
			a.extraInst++
			if !seenKind["e"+sourceKind(s)] {
				seenKind["e"+sourceKind(s)] = true
				a.extraBuilds++
			}
		}
	}

	fmt.Printf("\n=== LEVEL 1: %sSOURCE CENSUS (%d builds with a PoB dump) ===\n", censusLabel(actor), buildCount)
	fmt.Printf("Distinct normalized mod sources per build, compared as sets. Tags not compared.\n")
	fmt.Printf("%-14s %8s %8s %10s %10s %10s %10s\n",
		"KIND", "pobKeys", "ourKeys", "missBlds", "missSrcs", "xtraBlds", "xtraSrcs")
	kindNames := make([]string, 0, len(kinds))
	for k := range kinds {
		kindNames = append(kindNames, k)
	}
	sort.Slice(kindNames, func(i, j int) bool {
		return kinds[kindNames[i]].missBuilds > kinds[kindNames[j]].missBuilds
	})
	for _, k := range kindNames {
		a := kinds[k]
		fmt.Printf("%-14s %8d %8d %10d %10d %10d %10d\n",
			k, len(a.pobKeys), len(a.ourKeys), a.missBuilds, a.missInst, a.extraBuilds, a.extraInst)
	}

	printTop := func(title string, counts map[string]int, kind string, n int) {
		type row struct {
			key string
			n   int
		}
		rows := make([]row, 0)
		for k, c := range counts {
			if kind == "" || sourceKind(k) == kind {
				rows = append(rows, row{k, c})
			}
		}
		if len(rows) == 0 {
			return
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].n != rows[j].n {
				return rows[i].n > rows[j].n
			}
			return rows[i].key < rows[j].key
		})
		if len(rows) > n {
			rows = rows[:n]
		}
		fmt.Printf("\n  %s\n", title)
		for _, r := range rows {
			fmt.Printf("  %6d builds  %s\n", r.n, r.key)
		}
	}

	for _, k := range kindNames {
		if kinds[k].missBuilds == 0 && kinds[k].extraBuilds == 0 {
			continue
		}
		printTop(censusLabel(actor)+"MISSING "+k+" (PoB has, we do not)", missBuilds, k, 15)
		printTop(censusLabel(actor)+"EXTRA "+k+" (we have, PoB does not)", extraBuilds, k, 15)
	}
}

func censusLabel(actor string) string {
	if actor == "PLAYER" {
		return ""
	}
	return actor + " "
}

func reportNameCensus(sides []censusSide, actor string) {
	type agg struct {
		pobBuilds, ourBuilds    int
		missBuilds, extraBuilds int
		missInst, extraInst     int
	}
	byName := map[string]*agg{}
	buildCount := len(sides)
	for _, r := range sides {
		for _, d := range r.names {
			a := byName[d.name]
			if a == nil {
				a = &agg{}
				byName[d.name] = a
			}
			if d.pobHas {
				a.pobBuilds++
			}
			if d.ourHas {
				a.ourBuilds++
			}
			if d.missing > 0 {
				a.missBuilds++
				a.missInst += d.missing
			}
			if d.extra > 0 {
				a.extraBuilds++
				a.extraInst += d.extra
			}
		}
	}

	type row struct {
		name string
		a    *agg
	}
	rows := make([]row, 0, len(byName))
	for n, a := range byName {
		if a.missBuilds == 0 && a.extraBuilds == 0 {
			continue
		}
		rows = append(rows, row{n, a})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].a.missBuilds != rows[j].a.missBuilds {
			return rows[i].a.missBuilds > rows[j].a.missBuilds
		}
		if rows[i].a.extraBuilds != rows[j].a.extraBuilds {
			return rows[i].a.extraBuilds > rows[j].a.extraBuilds
		}
		return rows[i].name < rows[j].name
	})

	fmt.Printf("\n=== LEVEL 2: %sPER-MOD-NAME CENSUS (%d builds, %d mod names diverge) ===\n", censusLabel(actor), buildCount, len(rows))
	fmt.Printf("Multiset compare per name, bucketed by mod type, values within 1e-4 / 0.5%%. Tags NOT compared.\n")
	fmt.Printf("pobB/ourB = builds where that side has the name at all; missB/missN = builds/instances we lack.\n")
	fmt.Printf("%-52s %6s %6s %6s %8s %6s %8s\n",
		"MOD NAME", "pobB", "ourB", "missB", "missN", "xtraB", "xtraN")
	for _, r := range rows {
		a := r.a
		fmt.Printf("%-52s %6d %6d %6d %8d %6d %8d\n",
			r.name, a.pobBuilds, a.ourBuilds, a.missBuilds, a.missInst, a.extraBuilds, a.extraInst)
	}
}
