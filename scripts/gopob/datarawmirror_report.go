package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type dmSpec struct {
	goName   string
	file     string
	sources  []string
	extract  func(pob string) (map[string]any, error)
	mode     string
	scope    string
	subseq   []string
	optional []string
	unmapped string
}

func dmTables() []dmSpec {
	field := func(name string, fn func(any) (any, error)) func(string) (map[string]any, error) {
		return func(pob string) (map[string]any, error) { return dmSkillField(pob, name, fn) }
	}
	return []dmSpec{
		{
			goName: "Minions", file: "data/raw/minion_data.go",
			sources: []string{"src/Data/Minions.lua"},
			extract: func(pob string) (map[string]any, error) {
				return dmExtractMinionsFrom(pob, "src/Data/Minions.lua", "")
			},
			mode:   "exact",
			scope:  "every scalar field, weapon types, limit, skillList, and the mod NAME sequence",
			subseq: []string{"modList"},
		},
		{
			goName: "Spectres", file: "data/raw/spectre_data.go",
			sources: []string{"src/Data/Spectres.lua"},
			extract: func(pob string) (map[string]any, error) {
				return dmExtractMinionsFrom(pob, "src/Data/Spectres.lua", "ActiveSpectreLimit")
			},
			mode:   "subset",
			scope:  "same as Minions; ours is the 13 ids the 3.18 corpus references, by design",
			subseq: []string{"modList"},
		},
		{
			goName: "SkillBaseFlags", file: "data/raw/skill_flags.go",
			sources: []string{dmSkillGlob}, extract: field("baseFlags", dmFlagSet),
			mode: "exact", scope: "the full flag set per skill", subseq: []string{},
		},
		{
			goName: "SkillAddFlags", file: "data/raw/skill_flags.go",
			sources: []string{dmSkillGlob}, extract: field("addFlags", dmFlagSet),
			mode: "exact", scope: "the full flag set per support", subseq: []string{},
		},
		{
			goName: "PoBSkills", file: "data/raw/skill_flags.go",
			sources: []string{dmSkillGlob}, extract: dmExtractPoBSkills, mode: "exact",
			scope: "name, hidden, fromItem, fromTree, support", subseq: []string{},
		},
		{
			goName: "SkillMinionLists", file: "data/raw/minions.go",
			sources: []string{dmSkillGlob}, extract: field("minionList", dmStringList),
			mode: "exact", scope: "the whole minionList, in PoB's order", subseq: []string{},
		},
		{
			goName: "SupportAddMinionLists", file: "data/raw/minions.go",
			sources: []string{dmSkillGlob}, extract: field("addMinionList", dmStringList),
			mode: "exact", scope: "the whole addMinionList, in PoB's order", subseq: []string{},
		},
		{
			goName: "SkillAddSkillTypes", file: "data/raw/skill_add_skill_types.go",
			sources: []string{dmSkillGlob}, extract: field("addSkillTypes", dmSkillTypeList),
			mode: "subset",
			scope: "the resolved SkillType list. Ours is an OVERRIDE table holding only the ids where " +
				"PoB and go-pob-data disagree, so PoB-only rows are expected; the extractor applies " +
				"Lua's last-key-wins, which is the whole point of the one row we carry",
			subseq: []string{},
		},
		{
			goName: "SkillMinionUses", file: "data/raw/minion_uses.go",
			sources: []string{dmSkillGlob}, extract: field("minionUses", dmFlagSet),
			mode: "exact", scope: "the full slot set per skill", subseq: []string{},
		},
		{
			goName: "SkillMinionHasItemSet", file: "data/raw/minion_uses.go",
			sources: []string{dmSkillGlob},
			extract: field("minionHasItemSet", func(v any) (any, error) {
				b, ok := v.(bool)
				return ok && b, nil
			}),
			mode: "exact", scope: "the boolean", subseq: []string{},
		},
		{
			goName: "Gems", file: "data/raw/gems.go",
			sources: []string{"src/Data/Gems.lua"}, extract: dmExtractGems, mode: "exact",
			scope: "name and the full tag set; nine ids are renamed to go-pob-data's, per " +
				"builds.pobGemIDtoGameGemIDs",
			subseq: []string{},
		},
		{
			goName: "SkillParts", file: "data/raw/skill_parts.go",
			sources: []string{dmSkillGlob}, extract: field("parts", dmPartsRows), mode: "exact",
			scope: "each part's name and its boolean flag overrides, in order", subseq: []string{},
		},
		{
			goName: "SkillHiddenLevels", file: "data/raw/skill_hidden_levels.go",
			sources: []string{dmSkillGlob}, extract: dmExtractHiddenLevels, mode: "subset",
			scope: "stats and the positional level values, plus skillTypes WHERE WE DECLARE ONE -- " +
				"that field is an override for the hidden skills whose go-pob-data ActiveSkillTypes " +
				"is empty, so an empty one here means 'not overridden' rather than 'wrong'. Ours is " +
				"the curated set of hidden skills a corpus build actually reaches, so PoB-only rows " +
				"are expected",
			subseq: []string{}, optional: []string{"skillTypes"},
		},
		{
			goName: "SkillBaseMods", file: "data/raw/skill_base_mods.go",
			sources: []string{dmSkillGlob}, extract: field("baseMods", dmModCallNames),
			mode: "subset",
			scope: "the mod NAME sequence only. VALUES, TAGS AND FLAGS ARE NOT COMPARED -- " +
				"oracle/skill_base_mods_verify.py owns that against a luajit dump. The emitter " +
				"drops unrepresentable mods, so ours is a subsequence and a skill whose mods are " +
				"ALL dropped is absent entirely",
			subseq: []string{"*"},
		},
		{
			goName: "SkillStatMap", file: "data/raw/skill_stat_map.go",
			sources: []string{"src/Data/SkillStatMap.lua"}, extract: dmExtractSkillStatMap,
			mode: "subset",
			scope: "the stat key set, value/mult/div/base, and the mod NAME sequence. Mod values and " +
				"tags are out of scope",
			subseq: []string{"mods"},
		},
		{
			goName: "SkillStatMaps", file: "data/raw/skill_stat_maps.go",
			sources: []string{dmSkillGlob}, extract: dmExtractSkillStatMaps, mode: "subset",
			scope: "per skill: the stat key set, value/mult/div/base, and the mod NAME sequence. " +
				"The emitter drops unrepresentable mods and tallies them in the file header, " +
				"which task omissions gates",
			subseq: []string{"mods"},
		},
		{
			goName: "SkillHasGlobalEffect", file: "data/raw/has_global_effect.go",
			sources: []string{}, extract: nil, mode: "exact", subseq: []string{}, scope: "",
			unmapped: "DERIVED, not transcribed: computed from SkillBaseMods and SkillStatMaps by " +
				"this package, mirroring what src/Modules/Data.lua:686-707 computes at load " +
				"time. There is no PoB table to diff a row against, so it is verified only " +
				"transitively -- through the two tables it reads, which this sweep does grade.",
		},
	}
}

type dmDevKey struct{ table, key string }

var dmDeviations = func() map[dmDevKey]string {
	m := map[dmDevKey]string{
		{"SkillStatMaps", "Discipline"}: "data/raw/skill_stat_maps.go:38-45. PoB passes the " +
			"GlobalEffect/Aura tag in mod()'s FLAGS argument slot " +
			"(src/Data/Skills/act_int.lua:2621), so upstream the mod carries no tags, is never " +
			"promoted out of the skill's local mod list, and never reaches the player. Emitting " +
			"it as an aura would BEAT PoB's number rather than match it. We drop the mod instead, " +
			"which is not the same as PoB's untagged one -- the difference is only invisible " +
			"because neither reaches the player.",
	}
	for _, stat := range []string{
		"support_melee_physical_damage_+%_final",
		"support_melee_physical_damage_poison_and_bleeding_damage_+%_final_from_melee_hits",
		"support_melee_physical_damage_attack_speed_+%_final",
		"support_brutality_physical_damage_+%_final",
		"support_pulverise_melee_area_damage_+%_final",
		"support_pulverise_attack_speed_+%_final",
		"support_maim_chance_physical_damage_+%_final",
		"support_storm_barrier_damage_+%_final",
	} {
		m[dmDevKey{"SkillStatMap", stat}] = "data/raw/skill_stat_map.go:1785-1787 -- a support's " +
			"per-skill statMap entry hoisted into the global map. PoB keeps it per-skill; our " +
			"SkillStatMaps mirrors that exactly, so this copy is an unreached duplicate rather " +
			"than a divergent value."
	}
	return m
}()

func dmIsSubsequence(small, big []any) bool {
	i := 0
	for _, x := range small {
		found := false
		for i < len(big) {
			y := big[i]
			i++
			if dmEqual(x, y) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func dmAsList(v any) ([]any, bool) {
	l, ok := v.([]any)
	return l, ok
}

func dmDiffRow(ours, theirs any, subseq, optional []string) []string {
	om, ourIsMap := ours.(map[string]any)
	tm, theirIsMap := theirs.(map[string]any)
	star := len(subseq) == 1 && subseq[0] == "*"
	if !ourIsMap || !theirIsMap {
		if star {
			ol, ok1 := dmAsList(ours)
			tl, ok2 := dmAsList(theirs)
			if ok1 && ok2 {
				if dmIsSubsequence(ol, tl) {
					return nil
				}
				return []string{fmt.Sprintf("not a subsequence of PoB's: ours=%s pob=%s",
					dmShort(ours), dmShort(theirs))}
			}
		}
		if dmEqual(ours, theirs) {
			return nil
		}
		return []string{fmt.Sprintf("ours=%s pob=%s", dmShort(ours), dmShort(theirs))}
	}

	keys := map[string]bool{}
	for k := range om {
		keys[k] = true
	}
	for k := range tm {
		keys[k] = true
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	var out []string
	for _, key := range sorted {
		a := om[key]
		b := tm[key]
		if dmContains(optional, key) && !dmTruthy(a) {
			continue
		}
		if dmContains(subseq, key) || star {
			al, ok1 := dmAsList(a)
			bl, ok2 := dmAsList(b)
			if ok1 && ok2 {
				if !dmIsSubsequence(al, bl) {
					out = append(out, fmt.Sprintf("%s: ours=%s is not a subsequence of pob=%s",
						key, dmShort(a), dmShort(b)))
				}
				continue
			}
		}
		if !dmEqual(a, b) {
			out = append(out, fmt.Sprintf("%s: ours=%s pob=%s", key, dmShort(a), dmShort(b)))
		}
	}
	return out
}

func dmContains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func dmTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case int:
		return t != 0
	case float64:
		return t != 0
	case []any:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	}
	return true
}

type dmRow struct {
	key   string
	diffs []string
}

type dmStats struct {
	ours, pob, shared, onlyOurs, onlyPoB, differing, declared int
}

func dmSortedStringKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func dmCompare(spec dmSpec, ours, theirs map[string]any) (string, dmStats, []dmRow, []dmRow) {
	var onlyOurs, onlyPoB, shared []string
	for _, k := range dmSortedStringKeys(ours) {
		if _, ok := theirs[k]; ok {
			shared = append(shared, k)
		} else {
			onlyOurs = append(onlyOurs, k)
		}
	}
	for _, k := range dmSortedStringKeys(theirs) {
		if _, ok := ours[k]; !ok {
			onlyPoB = append(onlyPoB, k)
		}
	}

	var rows []dmRow
	for _, key := range shared {
		if d := dmDiffRow(ours[key], theirs[key], spec.subseq, spec.optional); len(d) > 0 {
			rows = append(rows, dmRow{key, d})
		}
	}
	for _, key := range onlyOurs {
		rows = append(rows, dmRow{key, []string{"PRESENT IN OURS, ABSENT FROM PoB"}})
	}
	if spec.mode == "exact" {
		for _, key := range onlyPoB {
			rows = append(rows, dmRow{key, []string{"PRESENT IN PoB, ABSENT FROM OURS"}})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if len(rows[i].diffs) != len(rows[j].diffs) {
			return len(rows[i].diffs) > len(rows[j].diffs)
		}
		return rows[i].key < rows[j].key
	})

	var declared, kept []dmRow
	for _, r := range rows {
		if _, ok := dmDeviations[dmDevKey{spec.goName, r.key}]; ok {
			declared = append(declared, r)
		} else {
			kept = append(kept, r)
		}
	}
	differed := map[string]bool{}
	for _, r := range declared {
		differed[r.key] = true
	}
	var stale []string
	for k := range dmDeviations {
		if k.table != spec.goName || differed[k.key] {
			continue
		}
		if _, ok := ours[k.key]; ok {
			stale = append(stale, k.key)
		}
	}
	sort.Strings(stale)
	for _, key := range stale {
		kept = append(kept, dmRow{key, []string{"DECLARED AS A DEVIATION BUT NO LONGER DIFFERS -- " +
			"delete the entry from DEVIATIONS in scripts/dataraw_mirror.py"}})
	}

	stats := dmStats{
		ours: len(ours), pob: len(theirs), shared: len(shared),
		onlyOurs: len(onlyOurs), onlyPoB: len(onlyPoB), differing: len(kept),
		declared: len(declared),
	}
	verdict := "IDENTICAL"
	if len(kept) > 0 {
		verdict = "DIFFERS"
	}
	return verdict, stats, kept, declared
}

func dmGoDump(root string) map[string]any {
	f, err := os.CreateTemp("", "dataraw-mirror-*.json")
	if err != nil {
		//go-only: tempfile raises OSError; reaching it needs a broken TMPDIR.
		dmRefuse("cannot create the dump file: " + errnoText(err))
	}
	path := f.Name()
	_ = f.Close()
	defer func() { _ = os.Remove(path) }()

	cmd := exec.CommandContext(context.Background(), "go", "test", "./data/raw/", //nolint:gosec // a fixed argv
		"-run", "TestDumpMirrorTables", "-count=1")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "MIRROR_DUMP="+path)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprint(os.Stderr, stdout.String()+stderr.String())
		dmRefuse("the Go dump failed; nothing was measured")
	}
	obj, _ := pyLoadJSONObject(path)
	return dmRetype(obj).(map[string]any)
}

type dmResult struct {
	spec     dmSpec
	verdict  string
	reason   string
	stats    *dmStats
	rows     []dmRow
	declared []dmRow
}

func dmRun(pob, root string, maxRows int, reportOnly bool) int {
	dump := dmGoDump(root)
	tables := dmTables()

	speced := map[string]bool{}
	for _, t := range tables {
		speced[t.goName] = true
	}
	var undeclared []string
	for k := range dump {
		if !speced[k] {
			undeclared = append(undeclared, k)
		}
	}
	sort.Strings(undeclared)
	var missingFromDump []string
	for _, t := range tables {
		if t.extract != nil {
			if _, ok := dump[t.goName]; !ok {
				missingFromDump = append(missingFromDump, t.goName)
			}
		}
	}
	sort.Strings(missingFromDump)

	var results []dmResult
	for _, spec := range tables {
		switch {
		case spec.unmapped != "":
			results = append(results, dmResult{spec: spec, verdict: "UNMAPPED", reason: spec.unmapped})
			continue
		}
		theirsRaw, ok := dump[spec.goName]
		if !ok {
			results = append(results, dmResult{spec: spec, verdict: "UNMAPPED",
				reason: "not present in the Go dump (data/raw/mirror_dump_test.go)"})
			continue
		}
		theirs, err := spec.extract(pob)
		if err != nil {
			results = append(results, dmResult{spec: spec, verdict: "UNMAPPED",
				reason: fmt.Sprintf("%s: %s", dmExcName(err), err.Error())})
			continue
		}
		if len(theirs) == 0 {
			results = append(results, dmResult{spec: spec, verdict: "UNMAPPED",
				reason: "the extractor returned 0 rows from a source that exists -- the " +
					"table's shape in PoB has changed and the extractor no longer matches it"})
			continue
		}
		oursMap, _ := theirsRaw.(map[string]any)
		verdict, stats, rows, declared := dmCompare(spec, oursMap, theirs)
		results = append(results, dmResult{spec, verdict, "", &stats, rows, declared})
	}

	fmt.Printf("data/raw MIRROR SWEEP   POB_DIR=%s\n", pob)
	fmt.Println(strings.Repeat("=", 100))
	fileSet := map[string]bool{}
	for _, t := range tables {
		fileSet[t.file] = true
	}
	ident, diff, unmapped := 0, 0, 0
	for _, r := range results {
		switch r.verdict {
		case "IDENTICAL":
			ident++
		case "DIFFERS":
			diff++
		case "UNMAPPED":
			unmapped++
		}
	}
	fmt.Printf("DENOMINATOR: %d files, %d tables  ->  %d IDENTICAL / %d DIFFERS / %d UNMAPPED\n",
		len(fileSet), len(tables), ident, diff, unmapped)
	fmt.Println()
	fmt.Printf("%-24s%-12s%7s%7s%8s%11s%10s%10s%11s  MODE\n",
		"TABLE", "VERDICT", "OURS", "PoB", "SHARED", "ONLY-OURS", "ONLY-PoB", "DECLARED", "DIFFERING")
	fmt.Println(strings.Repeat("-", 110))
	for _, r := range results {
		if r.stats == nil {
			fmt.Printf("%-24s%-12s%7s%7s%8s%11s%10s%10s%11s  %s\n",
				r.spec.goName, r.verdict, "-", "-", "-", "-", "-", "-", "-", r.spec.mode)
			continue
		}
		s := r.stats
		fmt.Printf("%-24s%-12s%7d%7d%8d%11d%10d%10d%11d  %s\n",
			r.spec.goName, r.verdict, s.ours, s.pob, s.shared, s.onlyOurs, s.onlyPoB,
			s.declared, s.differing, r.spec.mode)
	}

	unexpected := 0
	for _, r := range results {
		if r.verdict == "UNMAPPED" && r.spec.unmapped == "" {
			unexpected++
		}
	}
	if unmapped > 0 {
		fmt.Println()
		fmt.Println("UNMAPPED -- UNVERIFIED, NOT CORRECT. This is the residual risk.")
		fmt.Println(strings.Repeat("-", 100))
		for _, r := range results {
			if r.verdict != "UNMAPPED" {
				continue
			}
			kind := "⚠️  UNEXPECTED"
			if r.spec.unmapped != "" {
				kind = "DECLARED"
			}
			fmt.Printf("  [%s] %s  (%s)\n", kind, r.spec.goName, r.spec.file)
			for _, line := range dmWrap(r.reason, 94) {
				fmt.Printf("      %s\n", line)
			}
		}
	}

	for _, r := range results {
		if r.verdict != "DIFFERS" {
			continue
		}
		fmt.Println()
		fmt.Printf("DIFFERS: %s  (%s)  %d rows\n", r.spec.goName, r.spec.file, r.stats.differing)
		fmt.Printf("  SCOPE: %s\n", strings.Join(dmWrap(r.spec.scope, 88), "\n"+strings.Repeat(" ", 9)))
		fmt.Println(strings.Repeat("-", 100))
		shown := r.rows
		if maxRows < len(shown) {
			shown = shown[:maxRows]
		}
		for _, row := range shown {
			fmt.Printf("  %s\n", row.key)
			d := row.diffs
			if len(d) > 6 {
				d = d[:6]
			}
			for _, s := range d {
				fmt.Printf("      %s\n", s)
			}
			if len(row.diffs) > 6 {
				fmt.Printf("      ... and %d more fields\n", len(row.diffs)-6)
			}
		}
		if len(r.rows) > maxRows {
			fmt.Printf("  ... and %d more rows (--rows N)\n", len(r.rows)-maxRows)
		}
	}

	declaredTotal := 0
	for _, r := range results {
		declaredTotal += len(r.declared)
	}
	if declaredTotal > 0 {
		fmt.Println()
		fmt.Printf("DECLARED DEVIATIONS -- %d rows we do NOT mirror, on purpose.\n", declaredTotal)
		fmt.Println("A reason is not a verification. Re-read these before an upgrade.")
		fmt.Println(strings.Repeat("-", 110))
		for _, r := range results {
			for _, row := range r.declared {
				fmt.Printf("  %s / %s\n", r.spec.goName, row.key)
				d := row.diffs
				if len(d) > 3 {
					d = d[:3]
				}
				for _, s := range d {
					fmt.Printf("      %s\n", s)
				}
				for _, line := range dmWrap(dmDeviations[dmDevKey{r.spec.goName, row.key}], 92) {
					fmt.Printf("      | %s\n", line)
				}
			}
		}
	}

	fmt.Println()
	fmt.Println("SCOPES (what IDENTICAL means for each table)")
	fmt.Println(strings.Repeat("-", 100))
	for _, r := range results {
		if r.verdict == "UNMAPPED" {
			continue
		}
		fmt.Printf("  %s [%s]: %s\n", r.spec.goName, r.spec.mode,
			strings.Join(dmWrap(r.spec.scope, 92), "\n"+strings.Repeat(" ", 6)))
	}

	bad := false
	if len(undeclared) > 0 {
		fmt.Println()
		fmt.Println("⚠️  IN THE Go DUMP BUT NOT IN THIS SPEC -- unswept: " + strings.Join(undeclared, ", "))
		bad = true
	}
	if len(missingFromDump) > 0 {
		fmt.Println()
		fmt.Println("⚠️  SPECED BUT ABSENT FROM THE Go DUMP: " + strings.Join(missingFromDump, ", "))
		bad = true
	}

	ok := !bad && diff == 0 && unexpected == 0
	fmt.Println()
	fmt.Printf("RESULT: %d DIFFERS, %d UNMAPPED (%d of them unexpected), %d declared deviations\n",
		diff, unmapped, unexpected, declaredTotal)
	tail := "FAILING -- see above"
	if ok {
		tail = "every table IDENTICAL within its declared scope"
	}
	fmt.Println("        " + tail)
	if reportOnly || ok {
		return 0
	}
	return 1
}

func dmWrap(text string, width int) []string {
	words := strings.Fields(text)
	var out []string
	line := ""
	for _, w := range words {
		if line != "" && len([]rune(line))+1+len([]rune(w)) > width {
			out = append(out, line)
			line = w
		} else {
			line = strings.TrimSpace(line + " " + w)
		}
	}
	if line != "" {
		out = append(out, line)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

const dmUsage = `usage: dataraw_mirror.py [--rows N] [--selftest]

Grade every hand-transcribed table in data/raw/ against the PoB Lua it came from.

  --rows N     max DIFFERS rows printed per table (default 20)
  --selftest   prove each verdict is reachable
  -h, --help   this text

  POB_DIR=<dir>  the checkout to grade against; REPORT=1 for the report-only run
`

func dmParseArgs(argv []string) (int, bool) {
	rows, selftestOnly := "20", false
	for i := 0; i < len(argv); i++ {
		x, val, hasVal := argv[i], "", false
		if strings.HasPrefix(x, "--") && strings.Contains(x, "=") {
			parts := strings.SplitN(x, "=", 2)
			x, val, hasVal = parts[0], parts[1], true
		}
		switch x {
		case "--rows":
			if !hasVal {
				if i+1 >= len(argv) {
					dmRefuse(fmt.Sprintf("--rows needs a value\n\n%s", dmUsage))
				}
				i++
				val = argv[i]
			}
			rows = val
		case "--selftest":
			if hasVal {
				dmRefuse(fmt.Sprintf("--selftest takes no value\n\n%s", dmUsage))
			}
			selftestOnly = true
		case "-h", "--help":
			fmt.Print(dmUsage)
			os.Exit(0)
		default:
			dmRefuse(fmt.Sprintf("unknown argument %s\n\n%s", pyRepr(argv[i]), dmUsage))
		}
	}
	n, err := strconv.Atoi(strings.TrimSpace(rows))
	if err != nil {
		dmRefuse(fmt.Sprintf("--rows wants an integer, not %s", pyRepr(rows)))
	}
	return n, selftestOnly
}

func dataRawMirror(argv []string) int {
	rows, selftestOnly := dmParseArgs(argv)
	if selftestOnly {
		return dmSelftest()
	}
	pob := pobPinDir()
	pob = dmExpandUser(pob)
	if !pbdIsDir(pyPathJoin(pob, "src")) {
		dmRefuse(fmt.Sprintf("%s has no src/ -- set POB_DIR", pob))
	}
	return dmRun(pob, dmRepoRoot(), rows, os.Getenv("REPORT") == "1")
}

func dmExpandUser(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, ok := pyUserHome(); ok {
			return home + p[1:]
		}
	}
	return p
}
