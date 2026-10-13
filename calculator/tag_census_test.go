package calculator

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestTagCensus is the fourth and last modDB census, and the only one that can see TAGS.
func TestTagCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census; `task census WHICH=tag`")
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

	results := make([]tagResult, len(names))

	workers := evalWorkers()
	t.Logf("tag-censusing %d builds across %d workers", len(names), workers)

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
			ModDB      map[string][]tagDumpMod `json:"modDB"`
			EnemyModDB map[string][]tagDumpMod `json:"enemyModDB"`
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
		res.treeSources = countTagDumpTreeSources(dump.ModDB)
		res.player = tagCompare(dump.ModDB, env.Player.ModDB.Mods)
		if dump.EnemyModDB != nil {
			res.enemyOK = true
			res.enemy = tagCompare(dump.EnemyModDB, env.Enemy.ModDB.Mods)
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

	if out := os.Getenv("CENSUS_TSV"); out != "" {
		if err := writeTagCensusTSV(out, results); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", out)
	}

	reportTagCensus(t, results, false, "PLAYER")
	reportTagCensus(t, results, true, "ENEMY")
}

func writeTagCensusTSV(path string, results []tagResult) error {
	var b strings.Builder
	b.WriteString("build\tactor\tkind\tkey\tmissing\textra\n")
	for i := range results {
		r := &results[i]
		if !r.hasDump || r.excluded {
			continue
		}
		build := r.build + ".xml"
		fmt.Fprintf(&b, "%s\t-\t-\t-\t0\t0\n", build)
		sides := []struct {
			actor string
			ok    bool
			side  *tagSide
		}{{"player", true, &r.player}, {"enemy", r.enemyOK, &r.enemy}}
		for _, s := range sides {
			if !s.ok {
				continue
			}
			counts := map[string]int{}
			for _, d := range s.side.diffs {
				for _, it := range d.items {
					key := d.name + "|" + it.class + "|" + it.typ
					if it.key != "" {
						key += "/" + it.key
					}
					counts[key]++
				}
			}
			for _, k := range slices.Sorted(maps.Keys(counts)) {
				fmt.Fprintf(&b, "%s\t%s\ttag\t%s\t%d\t0\n", build, s.actor, k, counts[k])
			}
		}
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("writing tag census rows: %w", err)
	}
	return nil
}

type tagDumpMod struct {
	Source string          `json:"source"`
	Type   string          `json:"type"`
	Value  json.RawMessage `json:"value"`
	Flags  string          `json:"flags"`
	Kw     string          `json:"kw"`
	Tags   []string        `json:"tags"`
	Fmt    string          `json:"fmt"`
}

type tagResult struct {
	build         string
	hasDump       bool
	enemyOK       bool
	excluded      bool
	treeSources   int
	player, enemy tagSide
}

const tagCensusUnparsedCeiling = 746

type tagSide struct {
	unpairedPoB, unpairedOurs int
	pairs                     int
	tagsMatch                 int
	unparsed                  int
	unparsedEx                []string
	pobTags, ourTags          int
	dumpMods, dumpTags        int
	pobTagsUnpaired           int
	typeDisagree              int
	typeDisagreeEx            []string
	diffs                     []tagDivergence
}

type tagDivergence struct {
	name  string
	items []tagDelta
}

type tagDelta struct {
	class    string
	typ      string
	key      string
	pob, our string
}

func countTagDumpTreeSources(dumped map[string][]tagDumpMod) int {
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

func tagCompare(dumped map[string][]tagDumpMod, ourMods map[string][]mod.Mod) tagSide {
	var side tagSide

	ours := map[string]map[string][][]canonTag{}
	for name, mods := range ourMods {
		buckets := map[string][][]canonTag{}
		for _, m := range mods {
			tok := orderToken(string(m.Type()), normalizeSource(string(m.GetSource())), ourModValue(m))
			tags := make([]canonTag, 0, len(m.Tags()))
			for _, tg := range m.Tags() {
				tags = append(tags, canonOurTag(tg))
			}
			buckets[tok] = append(buckets[tok], tags)
		}
		ours[name] = buckets
	}

	pob := map[string]map[string][][]canonTag{}
	for name, mods := range dumped {
		buckets := map[string][][]canonTag{}
		for _, m := range mods {
			side.dumpMods++
			seg, ok := pobTagSegment(name, m)
			if !ok {
				side.unparsed++
				if len(side.unparsedEx) < 3 {
					side.unparsedEx = append(side.unparsedEx,
						fmt.Sprintf("%s [%s] %s", name, m.Type, truncate(m.Fmt, 160)))
				}
				continue
			}
			pobTags := parsePobTags(seg)
			side.dumpTags += len(pobTags)

			disagree := len(pobTags) != len(m.Tags)
			for i := 0; !disagree && i < len(pobTags); i++ {
				disagree = pobTags[i].typ != m.Tags[i]
			}
			if disagree {
				side.typeDisagree++
				if len(side.typeDisagreeEx) < 3 {
					types := make([]string, len(pobTags))
					for i, pt := range pobTags {
						types[i] = pt.typ
					}
					side.typeDisagreeEx = append(side.typeDisagreeEx, fmt.Sprintf("%s [%s] fmt %v, tags %v: %s",
						name, m.Type, types, m.Tags, truncate(m.Fmt, 160)))
				}
			}

			tok := orderToken(m.Type, normalizeSource(m.Source), pobModValue(m.Type, m.Value))
			buckets[tok] = append(buckets[tok], pobTags)
		}
		pob[name] = buckets
	}

	for name, buckets := range pob {
		ourBuckets := ours[name]
		for tok, pobMods := range buckets {
			pobLeft, ourLeft := matchByTags(pobMods, ourBuckets[tok])
			n := min(len(pobLeft), len(ourLeft))
			matched := len(pobMods) - len(pobLeft)

			side.pairs += matched
			side.tagsMatch += matched
			for _, tags := range pobMods {
				side.pobTags += len(tags)
			}
			for _, tags := range ourBuckets[tok] {
				side.ourTags += len(tags)
			}

			for i := 0; i < n; i++ {
				side.pairs++
				deltas := diffTags(pobLeft[i], ourLeft[i])
				if len(deltas) == 0 {
					side.tagsMatch++
					continue
				}
				side.diffs = append(side.diffs, tagDivergence{name: name, items: deltas})
			}
			for i := n; i < len(pobLeft); i++ {
				side.unpairedPoB++
				side.pobTagsUnpaired += len(pobLeft[i])
				side.pobTags -= len(pobLeft[i])
			}
			side.unpairedOurs += len(ourLeft) - n
			for i := n; i < len(ourLeft); i++ {
				side.ourTags -= len(ourLeft[i])
			}
			delete(ourBuckets, tok)
		}
	}

	for _, buckets := range ours {
		for _, remaining := range buckets {
			side.unpairedOurs += len(remaining)
		}
	}
	return side
}

func matchByTags(pobMods, ourMods [][]canonTag) ([][]canonTag, [][]canonTag) {
	pobLeft := make([][]canonTag, 0, len(pobMods))
	ourLeft := make([][]canonTag, 0, len(ourMods))
	sig := func(tags []canonTag) string {
		parts := make([]string, len(tags))
		for i, t := range tags {
			parts[i] = t.String()
			if t.unmapped {
				parts[i] = "\x01UNMAPPED\x01" + t.typ
			}
		}
		sort.Strings(parts)
		return strings.Join(parts, "\x00")
	}
	avail := map[string]int{}
	for _, tags := range ourMods {
		avail[sig(tags)]++
	}
	for _, tags := range pobMods {
		s := sig(tags)
		if avail[s] > 0 {
			avail[s]--
			continue
		}
		pobLeft = append(pobLeft, tags)
	}
	used := map[string]int{}
	for _, tags := range ourMods {
		s := sig(tags)
		if avail[s] > used[s] {
			used[s]++
			ourLeft = append(ourLeft, tags)
		}
	}
	return pobLeft, ourLeft
}

func diffTags(pob, our []canonTag) []tagDelta {
	byType := func(tags []canonTag) map[string][]canonTag {
		out := map[string][]canonTag{}
		for _, t := range tags {
			out[t.typ] = append(out[t.typ], t)
		}
		return out
	}
	p, o := byType(pob), byType(our)

	types := make([]string, 0, len(p)+len(o))
	for k := range p {
		types = append(types, k)
	}
	for k := range o {
		if _, dup := p[k]; !dup {
			types = append(types, k)
		}
	}
	sort.Strings(types)

	var out []tagDelta
	for _, typ := range types {
		pl, ol := p[typ], o[typ]

		pobCount := map[string]int{}
		for _, t := range pl {
			pobCount[t.params]++
		}
		ourCount := map[string]int{}
		unmapped := 0
		for _, t := range ol {
			if t.unmapped {
				unmapped++
				continue
			}
			ourCount[t.params]++
		}
		if unmapped > 0 {
			out = append(out, tagDelta{class: "UNMAPPED", typ: typ})
		}
		for k, pc := range pobCount {
			if oc := ourCount[k]; oc > 0 {
				n := min(pc, oc)
				pobCount[k] -= n
				ourCount[k] -= n
			}
		}
		pobLeft := leftovers(pobCount)
		ourLeft := leftovers(ourCount)

		n := min(len(pobLeft), len(ourLeft))
		for i := 0; i < n; i++ {
			out = append(out, paramDeltas(typ, canonTag{typ: typ, params: pobLeft[i]}, canonTag{typ: typ, params: ourLeft[i]})...)
		}
		for i := n; i < len(pobLeft); i++ {
			out = append(out, tagDelta{class: "ABSENT", typ: typ, pob: pobLeft[i]})
		}
		for i := n; i < len(ourLeft); i++ {
			out = append(out, tagDelta{class: "EXTRA", typ: typ, our: ourLeft[i]})
		}
	}
	return out
}

func leftovers(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for k, n := range counts {
		if n > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		for i := 0; i < counts[k]; i++ {
			out = append(out, k)
		}
	}
	return out
}

func paramDeltas(typ string, pob, our canonTag) []tagDelta {
	split := func(params string) map[string]string {
		out := map[string]string{}
		if params == "" {
			return out
		}
		for _, kv := range strings.Split(params, "/") {
			if k, v, found := strings.Cut(kv, "="); found {
				out[k] = v
			}
		}
		return out
	}
	pm, om := split(pob.params), split(our.params)
	keys := make([]string, 0, len(pm)+len(om))
	for k := range pm {
		keys = append(keys, k)
	}
	for k := range om {
		if _, dup := pm[k]; !dup {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	out := make([]tagDelta, 0, 2)
	for _, k := range keys {
		if pm[k] == om[k] {
			continue
		}
		out = append(out, tagDelta{class: "PARAM", typ: typ, key: k, pob: pm[k], our: om[k]})
	}
	return out
}

func reportTagCensus(t *testing.T, results []tagResult, enemy bool, actor string) {
	sides := make([]tagSide, 0, len(results))
	for i := range results {
		r := &results[i]
		if !r.hasDump || r.excluded {
			continue
		}
		if enemy {
			if !r.enemyOK {
				continue
			}
			sides = append(sides, r.enemy)
			continue
		}
		sides = append(sides, r.player)
	}
	if len(sides) == 0 {
		t.Errorf("%s: no comparable build — the tag census would report a meaningless zero", actor)
		return
	}

	var pairs, match, unpairedPoB, unpairedOurs, unparsed, pobTags, ourTags int
	var unparsedEx, typeDisagreeEx []string
	var dumpMods, dumpTags, pobTagsUnpaired, typeDisagree int
	byType := map[string]map[string]int{}
	byTypeKey := map[string]int{}
	byName := map[string]int{}
	byPair := map[string]int{}
	diffBuilds := 0
	for _, s := range sides {
		pairs += s.pairs
		match += s.tagsMatch
		unpairedPoB += s.unpairedPoB
		unpairedOurs += s.unpairedOurs
		unparsed += s.unparsed
		for _, ex := range s.unparsedEx {
			if len(unparsedEx) < 3 {
				unparsedEx = append(unparsedEx, ex)
			}
		}
		pobTags += s.pobTags
		ourTags += s.ourTags
		dumpMods += s.dumpMods
		dumpTags += s.dumpTags
		pobTagsUnpaired += s.pobTagsUnpaired
		typeDisagree += s.typeDisagree
		for _, ex := range s.typeDisagreeEx {
			if len(typeDisagreeEx) < 3 {
				typeDisagreeEx = append(typeDisagreeEx, ex)
			}
		}
		if len(s.diffs) > 0 {
			diffBuilds++
		}
		for _, d := range s.diffs {
			byName[d.name] += len(d.items)
			for _, it := range d.items {
				if byType[it.typ] == nil {
					byType[it.typ] = map[string]int{}
				}
				byType[it.typ][it.class]++
				if it.class == "PARAM" {
					byTypeKey[it.typ+"/"+it.key]++
					byPair[it.typ+"/"+it.key+"\x00"+it.pob+"\x00"+it.our]++
				} else {
					byTypeKey[it.typ+" ("+it.class+")"]++
					byPair[it.typ+" ("+it.class+")"+"\x00"+it.pob+"\x00"+it.our]++
				}
			}
		}
	}
	differ := 0
	deltas := 0
	for _, s := range sides {
		differ += len(s.diffs)
		for _, d := range s.diffs {
			deltas += len(d.items)
		}
	}

	if match+differ != pairs {
		t.Errorf("%s: TAGS-MATCH %d + TAGS-DIFFER %d != pairs %d — a paired mod escaped both classes",
			actor, match, differ, pairs)
	}
	if unparsed > tagCensusUnparsedCeiling {
		t.Errorf("%s: %d mods whose tag segment could not be located in fmt (ceiling %d) — the "+
			"census is under-reporting by that many.\nExemplars:\n  %s",
			actor, unparsed, tagCensusUnparsedCeiling, strings.Join(unparsedEx, "\n  "))
	} else if unparsed > 0 {
		t.Logf("%s: %d UNPARSED against a ceiling of %d. Exemplars:\n  %s",
			actor, unparsed, tagCensusUnparsedCeiling, strings.Join(unparsedEx, "\n  "))
	}
	if pobTags+pobTagsUnpaired != dumpTags {
		t.Errorf("%s: paired tags %d + unpaired tags %d != dumped tags %d — tags are being "+
			"dropped between the dump and the classification", actor, pobTags, pobTagsUnpaired, dumpTags)
	}
	if typeDisagree != 0 {
		t.Errorf("%s: %d mods where the tag types parsed from fmt disagree with the dump's own "+
			"tags array — the fmt parser is misreading the encoding.\nExemplars:\n  %s",
			actor, typeDisagree, strings.Join(typeDisagreeEx, "\n  "))
	}
	unmapped := 0
	for _, m := range byType {
		unmapped += m["UNMAPPED"]
	}
	if unmapped != 0 {
		t.Errorf("%s: %d tags of a type canonOurTag does not translate — extend the type switch "+
			"in tag_canon_test.go, or the census silently stops comparing them", actor, unmapped)
	}

	fmt.Printf("\n=== %s TAG CENSUS (%d builds) ===\n", actor, len(sides))
	fmt.Printf("Tags compared ONLY on mods already matching (type, normalized source, value).\n")
	fmt.Printf("  paired mods       %8d   (PoB tags on them %d, ours %d)\n", pairs, pobTags, ourTags)
	fmt.Printf("  TAGS-MATCH        %8d   %5.1f%% of pairs\n", match, 100*float64(match)/float64(max(pairs, 1)))
	fmt.Printf("  TAGS-DIFFER       %8d   %5.1f%% of pairs, %d tag-level deltas, in %d builds\n",
		differ, 100*float64(differ)/float64(max(pairs, 1)), deltas, diffBuilds)
	fmt.Printf("  unpaired (PoB)    %8d   set aside: content divergence, TestModDBCensus's job\n", unpairedPoB)
	fmt.Printf("  unpaired (ours)   %8d\n", unpairedOurs)
	fmt.Printf("  UNPARSED          %8d   ceiling %d\n", unparsed, tagCensusUnparsedCeiling)
	fmt.Printf("  dumped mods       %8d   dumped tags %d — recomputable by scripts/tag_census_check.py\n",
		dumpMods, dumpTags)
	fmt.Printf("  fmt vs tags array %8d   disagreements, asserted 0\n", typeDisagree)

	fmt.Printf("\n  --- BY TAG TYPE (a systematic defect is one type across many mods) ---\n")
	fmt.Printf("  %-24s %8s %8s %8s %8s\n", "TAG TYPE", "PARAM", "ABSENT", "EXTRA", "UNMAPPED")
	typeNames := make([]string, 0, len(byType))
	for k := range byType {
		typeNames = append(typeNames, k)
	}
	total := func(k string) int {
		m := byType[k]
		return m["PARAM"] + m["ABSENT"] + m["EXTRA"] + m["UNMAPPED"]
	}
	sort.Slice(typeNames, func(i, j int) bool {
		if total(typeNames[i]) != total(typeNames[j]) {
			return total(typeNames[i]) > total(typeNames[j])
		}
		return typeNames[i] < typeNames[j]
	})
	for _, k := range typeNames {
		m := byType[k]
		fmt.Printf("  %-24s %8d %8d %8d %8d\n", k, m["PARAM"], m["ABSENT"], m["EXTRA"], m["UNMAPPED"])
	}

	printRanked("BY TAG TYPE + PARAM KEY (names the exact field that diverges)", byTypeKey, 30)
	printRanked("BY MOD NAME (tag-level deltas)", byName, 30)

	type pairRow struct {
		key string
		n   int
	}
	rows := make([]pairRow, 0, len(byPair))
	for k, n := range byPair {
		rows = append(rows, pairRow{k, n})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].n != rows[j].n {
			return rows[i].n > rows[j].n
		}
		return rows[i].key < rows[j].key
	})
	if len(rows) > 40 {
		rows = rows[:40]
	}
	fmt.Printf("\n  --- TOP CONCRETE TAG DIVERGENCES (PoB -> ours) ---\n")
	for _, r := range rows {
		f := strings.Split(r.key, "\x00")
		fmt.Printf("  %8d  %-34s  %q -> %q\n", r.n, f[0], f[1], f[2])
	}
}

func printRanked(title string, counts map[string]int, n int) {
	type row struct {
		key string
		n   int
	}
	rows := make([]row, 0, len(counts))
	for k, c := range counts {
		rows = append(rows, row{k, c})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].n != rows[j].n {
			return rows[i].n > rows[j].n
		}
		return rows[i].key < rows[j].key
	})
	fmt.Printf("\n  --- %s (%d distinct) ---\n", title, len(rows))
	if len(rows) > n {
		rows = rows[:n]
	}
	for _, r := range rows {
		fmt.Printf("  %8d  %s\n", r.n, r.key)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
