package calculator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

type diagDumpMod struct {
	Source string          `json:"source"`
	Type   string          `json:"type"`
	Value  json.RawMessage `json:"value"`
	Flags  string          `json:"flags"`
	Kw     string          `json:"kw"`
	Fmt    string          `json:"fmt"`
	Tags   []string        `json:"tags"`
}

type diagDump struct {
	ModDB       map[string][]diagDumpMod `json:"modDB"`
	MinionModDB map[string][]diagDumpMod `json:"minionModDB"`
	EnemyModDB  map[string][]diagDumpMod `json:"enemyModDB"`
}

func diagActorModDB(d *diagDump, env *Environment, actor string) (map[string][]diagDumpMod, *moddb.ModDB, string) {
	switch actor {
	case "minion":
		if env.Minion == nil || env.Minion.Actor == nil {
			return d.MinionModDB, nil, "minion"
		}
		return d.MinionModDB, env.Minion.ModDB, "minion"
	case "enemy":
		if env.Enemy == nil {
			return d.EnemyModDB, nil, "enemy"
		}
		return d.EnemyModDB, env.Enemy.ModDB, "enemy"
	default:
		return d.ModDB, env.Player.ModDB, "player"
	}
}

func diagDumpsDir(t *testing.T) string {
	t.Helper()
	return dumpsDirForLoadedData(t)
}

func diagCorpusNames(t *testing.T) []string {
	t.Helper()
	_, files := corpusNamesForLoadedData(t)
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, strings.TrimSuffix(f, ".xml"))
	}
	sort.Strings(names)
	return names
}

func requireHealthyDump(t *testing.T, buildName string, healthy bool) {
	t.Helper()
	if healthy || os.Getenv("DIAG_ALLOW_UNHEALTHY") == "1" {
		return
	}
	t.Fatalf("REFUSING: the PoB dump for %s has fewer than %d distinct Tree: sources, which means "+
		"headless PoB never applied its passive tree. Every tree-sourced mod would show as "+
		"\"ours only\", which looks exactly like an over-application bug and is not one.\n"+
		"Re-dump the corpus (scripts/dump_shard.sh, ~5 min at 12 shards) — the dumps on disk may "+
		"predate the HeadlessWrapper Inflate fix. DIAG_ALLOW_UNHEALTHY=1 overrides.",
		buildName, oracleMinTreeSources)
}

func diagLoadDump(dir, name string) (*diagDump, bool) {
	raw, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		return nil, false
	}
	var d diagDump
	if json.Unmarshal(raw, &d) != nil {
		return nil, false
	}
	tree := map[string]bool{}
	for _, mods := range d.ModDB {
		for _, m := range mods {
			s := normalizeSource(m.Source)
			if strings.HasPrefix(s, "Tree:") {
				tree[s] = true
			}
		}
	}
	return &d, len(tree) >= 5
}

// TestDiagNameRank ranks builds by how many more mods of NAME we emit than PoB, excluding builds whose PoB dump is broken.
func TestDiagNameRank(t *testing.T) {
	requireOperatorRun(t, "report-only ranking; `task diag`")
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	dir := diagDumpsDir(t)
	name := os.Getenv("NAME")
	if name == "" {
		name = "Life"
	}
	names := diagCorpusNames(t)
	if n := os.Getenv("LIMIT"); n != "" {
		if l, err := strconv.Atoi(n); err == nil && l < len(names) {
			names = names[:l]
		}
	}
	corpusDir := corpusDirForLoadedData(t)

	type row struct {
		name       string
		pobN, ourN int
		healthy    bool
	}
	rows := make([]row, len(names))

	var nNoDump, nNoXML, nParseErr, nNoEnv int64
	runConcurrently(len(names), evalWorkers(), func(i int, cache *EnvironmentCache) {
		d, healthy := diagLoadDump(dir, names[i])
		if d == nil {
			atomic.AddInt64(&nNoDump, 1)
			return
		}
		file, err := os.ReadFile(filepath.Join(corpusDir, names[i]+".xml"))
		if err != nil {
			atomic.AddInt64(&nNoXML, 1)
			return
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			atomic.AddInt64(&nParseErr, 1)
			return
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil {
			atomic.AddInt64(&nNoEnv, 1)
			return
		}
		rows[i] = row{names[i], len(d.ModDB[name]), len(env.Player.ModDB.Mods[name]), healthy}
	})
	fmt.Printf("skips: noDump=%d noXML=%d parseErr=%d noEnv=%d of %d\n", nNoDump, nNoXML, nParseErr, nNoEnv, len(names))

	var healthyRows []row
	for _, r := range rows {
		if r.name != "" && r.healthy {
			healthyRows = append(healthyRows, r)
		}
	}
	sort.Slice(healthyRows, func(i, j int) bool {
		return healthyRows[i].ourN-healthyRows[i].pobN > healthyRows[j].ourN-healthyRows[j].pobN
	})
	var sumP, sumO int
	for _, r := range healthyRows {
		sumP += r.pobN
		sumO += r.ourN
	}
	fmt.Printf("\n=== %s across %d healthy builds: pob=%d ours=%d (ratio %.2f) ===\n",
		name, len(healthyRows), sumP, sumO, float64(sumO)/float64(sumP))
	for i, r := range healthyRows {
		if i >= 25 {
			break
		}
		fmt.Printf("  %+5d  pob=%-4d ours=%-4d %s\n", r.ourN-r.pobN, r.pobN, r.ourN, r.name)
	}
}

// TestDiagBuildMods dumps both sides' mods of NAME for BUILD.
func TestDiagBuildMods(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	dir := diagDumpsDir(t)
	name := os.Getenv("NAME")
	if name == "" {
		name = "Life"
	}
	if strings.Contains(name, ",") {
		t.Fatalf("NAME=%q is a comma-list, which this diag does NOT support — it would look up a\n"+
			"single mod name containing a comma, find nothing, and print `pob=0 ours=0` as though\n"+
			"the build were clean. Use ONE name, or NAME='*' to diff every name at once.\n"+
			"(TestDiagEvalTags is the one that accepts a comma-list.)", name)
	}
	buildName := os.Getenv("BUILD")
	if buildName == "" {
		t.Skip("BUILD unset")
	}
	actor := strings.ToLower(os.Getenv("ACTOR"))

	d, healthy := diagLoadDump(dir, buildName)
	if d == nil {
		t.Fatalf("no dump for %s", buildName)
	}
	requireHealthyDump(t, buildName, healthy)
	file, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), buildName+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatal(err)
	}
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

	pobMods, ourDB, actorLabel := diagActorModDB(d, env, actor)
	if ourDB == nil {
		t.Fatalf("%s: build grants no %s actor (PoB dump has %d %s mods of %q)",
			buildName, actorLabel, len(pobMods[name]), actorLabel, name)
	}

	if name == "*" || strings.EqualFold(name, "all") {
		reportAllNames(buildName, actorLabel, healthy, pobMods, ourDB)
		return
	}

	type entry struct {
		src, typ string
		val      float64
	}

	pobList := make([]entry, 0, len(pobMods[name]))
	ourList := make([]entry, 0, len(ourDB.Mods[name]))
	for _, m := range pobMods[name] {
		pobList = append(pobList, entry{normalizeSource(m.Source), m.Type, pobModValue(m.Type, m.Value)})
	}
	for _, m := range ourDB.Mods[name] {
		ourList = append(ourList, entry{normalizeSource(string(m.GetSource())), string(m.Type()), ourModValue(m)})
	}

	agg := func(list []entry, bySource bool) map[string]int {
		out := map[string]int{}
		for _, e := range list {
			src := ""
			if bySource {
				src = e.src
			}
			out[fmt.Sprintf("%-44s %-8s %10.4f", src, e.typ, e.val)]++
		}
		return out
	}
	differingRows := func(p, o map[string]int) int {
		seen, n := map[string]bool{}, 0
		for k := range p {
			seen[k] = true
		}
		for k := range o {
			seen[k] = true
		}
		for k := range seen {
			if p[k] != o[k] {
				n++
			}
		}
		return n
	}

	diffBySource := differingRows(agg(pobList, true), agg(ourList, true))
	diffBlind := differingRows(agg(pobList, false), agg(ourList, false))

	bySource := diffBySource <= diffBlind
	why := fmt.Sprintf("auto: %d differing rows by-source vs %d source-blind", diffBySource, diffBlind)
	switch os.Getenv("BYSOURCE") {
	case "1":
		bySource, why = true, "forced by BYSOURCE=1"
	case "0":
		bySource, why = false, "forced by BYSOURCE=0"
	}

	p, o := agg(pobList, bySource), agg(ourList, bySource)
	keys := map[string]bool{}
	for k := range p {
		keys[k] = true
	}
	for k := range o {
		keys[k] = true
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	mode, col := "SOURCE-BLIND", "(source ignored) / TYPE / VALUE"
	if bySource {
		mode, col = "BY-SOURCE", "SOURCE / TYPE / VALUE"
	}
	fmt.Printf("\n=== %s / %s / actor=%s (healthy dump=%v) pob=%d ours=%d ===\n",
		buildName, name, actorLabel, healthy, len(pobList), len(ourList))
	fmt.Printf("grouping: %s — %s. Flip with BYSOURCE=1 / BYSOURCE=0.\n", mode, why)

	differing := 0
	for _, k := range sorted {
		if p[k] != o[k] {
			differing++
		}
	}
	fmt.Printf("%4s %4s  %s\n", "pob", "our", col)
	for _, k := range sorted {
		mark := "  "
		if p[k] != o[k] {
			mark = "**"
		}
		fmt.Printf("%4d %4d %s %s\n", p[k], o[k], mark, k)
	}
	fmt.Printf("%d of %d rows differ.\n", differing, len(sorted))
	if bySource && differing == len(sorted) && len(sorted) > 5 {
		fmt.Printf("EVERY row differs, which usually means the grouping is wrong rather than the mods:\n" +
			"re-run with BYSOURCE=0 before concluding anything.\n")
	}

	reportListPayloads(pobMods[name], ourDB.Mods[name])
}

func reportListPayloads(pobMods []diagDumpMod, ourMods []mod.Mod) {
	var ours []mod.Mod
	for _, m := range ourMods {
		if v := m.Value(); v != nil && v.Type() == mod.ModValueMultiTypeList {
			ours = append(ours, m)
		}
	}
	var pob []diagDumpMod
	for _, m := range pobMods {
		if strings.EqualFold(m.Type, "LIST") {
			pob = append(pob, m)
		}
	}
	if len(ours) == 0 && len(pob) == 0 {
		return
	}

	fmt.Printf("\n--- LIST PAYLOADS (invisible to the table above and to every census) ---\n")
	fmt.Printf("OURS (%d):\n", len(ours))
	for i, m := range ours {
		fmt.Printf("  [%d] %s\n", i+1, describeListPayload(m.Value().List()))
		for _, tg := range m.Tags() {
			fmt.Printf("        tag %s %+v\n", tg.Type(), tg)
		}
	}
	fmt.Printf("PoB (%d):\n", len(pob))
	for i, m := range pob {
		fmt.Printf("  [%d] src=%s fmt=%s\n", i+1, normalizeSource(m.Source), m.Fmt)
		if len(m.Tags) > 0 {
			fmt.Printf("        tagTypes=%v\n", m.Tags)
		}
	}
	fmt.Printf("Counts equal does NOT mean payloads equal — read the values, and read the tags.\n")
}

func describeListPayload(v any) string {
	nested := func(label string, m mod.Mod, extra string) string {
		if m == nil {
			return fmt.Sprintf("%s{<nil mod>}%s", label, extra)
		}
		val := "?"
		if mv := m.Value(); mv != nil {
			switch mv.Type() {
			case mod.ModValueMultiTypeFloat:
				val = fmt.Sprintf("%g", mv.Float())
			case mod.ModValueMultiTypeFlag:
				val = fmt.Sprintf("%t", mv.Flag())
			case mod.ModValueMultiTypeList:
				val = "<nested list>"
			}
		}
		out := fmt.Sprintf("%s%s -> %s %s %s", label, extra, m.Name(), m.Type(), val)
		for _, tg := range m.Tags() {
			out += fmt.Sprintf("  [nested tag %s %+v]", tg.Type(), tg)
		}
		return out
	}

	switch p := v.(type) {
	case mod.ExtraAura:
		return nested("ExtraAura", p.Mod, fmt.Sprintf("{OnlyAllies:%t}", p.OnlyAllies))
	case mod.ExtraAuraEffect:
		return nested("ExtraAuraEffect", p.Mod, "")
	case mod.MinionModifier:
		return nested("MinionModifier", p.Mod, "")
	case mod.LinkedSupport:
		return fmt.Sprintf("LinkedSupport{TargetSlotName:%q}", p.TargetSlotName)
	case mod.SkillData:
		return fmt.Sprintf("SkillData{Key:%q Value:%g Merge:%q}", p.Key, p.Value, p.Merge)
	case nil:
		return "<nil payload>"
	default:
		return fmt.Sprintf("%T%+v", v, v)
	}
}

// TestDiagGroupingPicksTheReadableOne pins the empirical grouping choice.
func TestDiagGroupingPicksTheReadableOne(t *testing.T) {
	agg := func(srcs []string, bySource bool) map[string]int {
		out := map[string]int{}
		for _, s := range srcs {
			if !bySource {
				s = ""
			}
			out[s]++
		}
		return out
	}
	differing := func(p, o map[string]int) int {
		seen, n := map[string]bool{}, 0
		for k := range p {
			seen[k] = true
		}
		for k := range o {
			seen[k] = true
		}
		for k := range seen {
			if p[k] != o[k] {
				n++
			}
		}
		return n
	}

	pob := []string{"Item:Bone Helmet", "Item:Gold Ring", "Item:To Dust"}
	ours := []string{"Other:<none>", "Other:<none>", "Other:<none>"}
	bySource := differing(agg(pob, true), agg(ours, true))
	blind := differing(agg(pob, false), agg(ours, false))
	if !(blind < bySource) {
		t.Fatalf("source-blind must win on the minion shape: bySource=%d blind=%d", bySource, blind)
	}

	same := []string{"Item:Bone Helmet", "Item:Gold Ring"}
	if differing(agg(same, true), agg(same, true)) != 0 {
		t.Fatal("identical sourced lists must show no differing rows by-source")
	}
}

func reportAllNames(buildName, actorLabel string, healthy bool, pobMods map[string][]diagDumpMod, ourDB *moddb.ModDB) {
	bySource := os.Getenv("BYSOURCE") == "1"
	key := func(src, typ string, val float64) string {
		if !bySource {
			src = ""
		}
		return fmt.Sprintf("%-44s %-8s %10.4f", src, typ, val)
	}

	pob := map[string]map[string]int{}
	for n, mods := range pobMods {
		m := map[string]int{}
		for _, e := range mods {
			m[key(normalizeSource(e.Source), e.Type, pobModValue(e.Type, e.Value))]++
		}
		pob[n] = m
	}
	ours := map[string]map[string]int{}
	for n, mods := range ourDB.Mods {
		m := map[string]int{}
		for _, mm := range mods {
			m[key(normalizeSource(string(mm.GetSource())), string(mm.Type()), ourModValue(mm))]++
		}
		ours[n] = m
	}

	all := map[string]bool{}
	for n := range pob {
		all[n] = true
	}
	for n := range ours {
		all[n] = true
	}
	names := make([]string, 0, len(all))
	for n := range all {
		names = append(names, n)
	}
	sort.Strings(names)

	mode := "SOURCE-BLIND"
	if bySource {
		mode = "BY-SOURCE"
	}
	fmt.Printf("\n=== ALL NAMES / %s / actor=%s (healthy dump=%v) — pob %d names, ours %d ===\n",
		buildName, actorLabel, healthy, len(pob), len(ours))
	fmt.Printf("grouping: %s (BYSOURCE=1 to key on the mod source)\n", mode)

	differing := 0
	for _, n := range names {
		p, o := pob[n], ours[n]
		rows := map[string]bool{}
		for k := range p {
			rows[k] = true
		}
		for k := range o {
			rows[k] = true
		}
		ks := make([]string, 0, len(rows))
		for k := range rows {
			if p[k] != o[k] {
				ks = append(ks, k)
			}
		}
		if len(ks) == 0 {
			continue
		}
		differing++
		sort.Strings(ks)
		fmt.Printf("--- %s\n", n)
		for _, k := range ks {
			fmt.Printf("%4d %4d ** %s\n", p[k], o[k], k)
		}
	}
	fmt.Printf("\n%d of %d mod names differ. A name here is an INPUT divergence; it says nothing "+
		"about whether the value is wrong downstream.\n", differing, len(names))
	fmt.Printf("This compares mod IDENTITY. It is BLIND to what a TAG evaluated to — use " +
		"TestDiagEvalTags for that — and LIST payloads are only shown in single-NAME mode.\n")
}
