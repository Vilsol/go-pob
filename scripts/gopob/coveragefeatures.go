package main

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const cfUsage = "usage: gopob coverage-features --census J [--root D] [--pool DIR] [--tsv F] " +
	"[--shared DIR] [--uniques DIR] [--cands N] [--grep RE]... [--grep-file F]"

var (
	cfMarkerRE       = regexp.MustCompile(`//pob:(?:block:)?(\S+)`)
	cfTopVarRE       = regexp.MustCompile(`^var (\w+) = `)
	cfFuncNameRE     = regexp.MustCompile(`^func (?:\([^)]*\) )?(\w+)`)
	cfTopLevelRE     = regexp.MustCompile(`^(func|var|type|const) `)
	cfRowKeyRE       = regexp.MustCompile("^\\s*(`[^`]*`|\"(?:[^\"\\\\]|\\\\.)*\")\\s*:\\s*")
	cfPatternFieldRE = regexp.MustCompile("^\\s*pattern:\\s*(`[^`]*`)")
	cfConfigKeyRE    = regexp.MustCompile(`^\t"(\w+)": func\(`)
	cfCaseRE         = regexp.MustCompile(`^\s*case ("[^"]+"(?:,\s*"[^"]+")*):`)
	cfInputRE        = regexp.MustCompile(`<Input name="([^"]+)"`)
	cfItemRE         = regexp.MustCompile(`(?s)<Item [^>]*>(.*?)</Item>`)
	cfGemRE          = regexp.MustCompile(`<Gem [^>]*nameSpec="([^"]*)"`)
	cfBraceRE        = regexp.MustCompile(`^(\{[^}]*\})+`)
	cfLadderRE       = regexp.MustCompile(`^\d+-ladder-(.+)\.xml$`)
)

var cfSkipReasons = map[string]bool{
	"entry-point": true, "defensive-guard": true, "config-unset-arm": true,
	"config-type-assert": true, "calcs-mode": true,
}

var cfSkipPrefixes = []string{"builds/parsecache/", "pob/", "utils/", "moddb/eval_trace.go"}

type cfRecord struct {
	File   string `json:"file"`
	Start  int    `json:"start"`
	Stmts  int    `json:"stmts"`
	Func   string `json:"func"`
	Reason string `json:"reason"`
}

type cfFeature struct {
	kind, key, site, marker string
	blocks, stmts           int
	corpus, shared, pool    int
	uniques                 []string
	cands                   []string
}

type cfPoolChar struct {
	id    string
	lines []string
}

func coverageFeatures(argv []string) int {
	var census, rootDir, pool, shared, uniquesDir, tsv string
	var greps []string
	cands := 5
	rootDir = "."
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if i+1 >= len(argv) {
			fmt.Fprintf(os.Stderr, "coverage-features: %s needs a value\n%s\n", arg, cfUsage)
			return 64
		}
		v := argv[i+1]
		i++
		switch arg {
		case "--census":
			census = v
		case "--root":
			rootDir = v
		case "--pool":
			pool = v
		case "--shared":
			shared = v
		case "--uniques":
			uniquesDir = v
		case "--tsv":
			tsv = v
		case "--grep":
			greps = append(greps, v)
		case "--grep-file":
			b, err := os.ReadFile(v) //nolint:gosec // operator path
			if err != nil {
				fmt.Fprintf(os.Stderr, "coverage-features: cannot read %s: %s\n", v, errnoText(err))
				return 1
			}
			for _, ln := range strings.Split(string(b), "\n") {
				if ln = strings.TrimSpace(ln); ln != "" {
					greps = append(greps, ln)
				}
			}
		case "--cands":
			if _, err := fmt.Sscanf(v, "%d", &cands); err != nil {
				fmt.Fprintf(os.Stderr, "coverage-features: --cands needs an integer\n")
				return 64
			}
		default:
			fmt.Fprintf(os.Stderr, "coverage-features: unknown argument %s\n%s\n", pyRepr(arg), cfUsage)
			return 64
		}
	}
	if census == "" && len(greps) == 0 {
		fmt.Fprintf(os.Stderr, "coverage-features: --census or --grep is required\n%s\n", cfUsage)
		return 64
	}

	corpus, rc := corpusDir(rootDir)
	if rc != 0 {
		return rc
	}
	corpusLines, corpusInputs, inCorpus, nBuilds, err := cfLoadCorpus(corpus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage-features: %v\n", err)
		return 1
	}
	if nBuilds == 0 {
		fmt.Fprintf(os.Stderr, "coverage-features: REFUSING: no builds under %s\n", corpus)
		return 1
	}
	var poolChars []cfPoolChar
	poolSkipped := 0
	if pool != "" {
		poolChars, poolSkipped, err = cfLoadPool(pool, inCorpus)
		if err != nil {
			fmt.Fprintf(os.Stderr, "coverage-features: %v\n", err)
			return 1
		}
		if len(poolChars) == 0 {
			fmt.Fprintf(os.Stderr, "coverage-features: REFUSING: --pool %s holds no character outside the corpus "+
				"(%d skipped as already in it)\n", pool, poolSkipped)
			return 1
		}
	}
	fmt.Printf("CORPUS %s: %d builds, %d distinct item lines\n", corpus, nBuilds, len(corpusLines))
	var sharedLines map[string][]int
	var sharedInputs map[string]int
	if shared != "" {
		var n int
		sharedLines, sharedInputs, _, n, err = cfLoadCorpus(shared)
		if err != nil || n == 0 {
			fmt.Fprintf(os.Stderr, "coverage-features: REFUSING: --shared %s holds no build XML (%v)\n", shared, err)
			return 1
		}
		fmt.Printf("SHARED %s: %d builds, usage signal only (not importable as they stand)\n", shared, n)
	}
	var uniques []cfPoolChar
	if uniquesDir != "" {
		uniques, err = cfLoadUniques(uniquesDir)
		if err != nil || len(uniques) == 0 {
			fmt.Fprintf(os.Stderr, "coverage-features: REFUSING: --uniques %s holds no unique (%v)\n", uniquesDir, err)
			return 1
		}
		fmt.Printf("UNIQUES %s: %d uniques\n", uniquesDir, len(uniques))
	}
	if pool != "" {
		fmt.Printf("POOL %s: %d characters outside the corpus (%d skipped as already in it)\n",
			pool, len(poolChars), poolSkipped)
	}

	if len(greps) > 0 {
		fmt.Println("GREP\tpattern\tcorpus_builds\tshared_builds\tpool_chars\tpob_uniques\tcandidates")
		for _, g := range greps {
			rx, err := regexp.Compile("(?i)" + g)
			if err != nil {
				fmt.Fprintf(os.Stderr, "coverage-features: bad --grep %s\n", pyRepr(g))
				return 64
			}
			n, _ := cfCorpusCount(rx, corpusLines)
			sh, _ := cfCorpusCount(rx, sharedLines)
			p, ids := cfPoolMatch(rx, poolChars, cands)
			_, names := cfPoolMatch(rx, uniques, 1<<30)
			fmt.Printf("GREP\t%s\t%d\t%d\t%d\t%s\t%s\n", g, n, sh, p, strings.Join(names, ";"), strings.Join(ids, ","))
		}
		return 0
	}

	raw, err := os.ReadFile(census) //nolint:gosec // operator path
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage-features: cannot read %s: %s\n", census, errnoText(err))
		return 1
	}
	var records []cfRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		fmt.Fprintf(os.Stderr, "coverage-features: %s is not coverage-census --json output: %v\n", census, err)
		return 1
	}
	if len(records) == 0 {
		fmt.Fprintf(os.Stderr, "coverage-features: REFUSING: %s lists no dead blocks\n", census)
		return 1
	}

	src := map[string][]string{}
	feats := map[string]*cfFeature{}
	var order []string
	dropped := map[string]int{}
	for _, r := range records {
		if cfSkipReasons[r.Reason] {
			dropped["bucket "+r.Reason]++
			continue
		}
		if p := cfSkipPrefix(r.File); p != "" {
			dropped["tree "+p]++
			continue
		}
		lines, ok := src[r.File]
		if !ok {
			b, err := os.ReadFile(filepath.Join(rootDir, r.File)) //nolint:gosec // a census path under root
			if err != nil {
				fmt.Fprintf(os.Stderr, "coverage-features: cannot read %s: %s\n", r.File, errnoText(err))
				return 1
			}
			lines = strings.Split(string(b), "\n")
			src[r.File] = lines
		}
		kind, key, marker := cfAttribute(r, lines)
		id := kind + "\x00" + key
		f := feats[id]
		if f == nil {
			f = &cfFeature{kind: kind, key: key, site: fmt.Sprintf("%s:%d", r.File, r.Start), marker: marker}
			feats[id] = f
			order = append(order, id)
		}
		f.blocks++
		f.stmts += r.Stmts
		if f.marker == "-" && marker != "-" {
			f.marker = marker
		}
	}

	for _, id := range order {
		f := feats[id]
		switch {
		case strings.HasPrefix(f.kind, "parser:"):
			rx, err := regexp.Compile(cfAnchor(f.kind, strings.Trim(f.key, "`\"")))
			if err != nil {
				f.corpus, f.shared, f.pool = -1, -1, -1
				continue
			}
			f.corpus, _ = cfCorpusCount(rx, corpusLines)
			f.shared, _ = cfCorpusCount(rx, sharedLines)
			f.pool, f.cands = cfPoolMatch(rx, poolChars, cands)
			_, f.uniques = cfPoolMatch(rx, uniques, 1<<30)
		case f.kind == "config":
			f.corpus = corpusInputs[f.key]
			f.shared = sharedInputs[f.key]
			f.pool = -1
		default:
			f.corpus, f.shared, f.pool = -1, -1, -1
		}
	}

	sort.SliceStable(order, func(i, j int) bool {
		a, b := feats[order[i]], feats[order[j]]
		if a.pool+a.shared != b.pool+b.shared {
			return a.pool+a.shared > b.pool+b.shared
		}
		return a.stmts > b.stmts
	})

	var dk []string
	for k := range dropped {
		dk = append(dk, k)
	}
	sort.Strings(dk)
	fmt.Println("DROPPED (no PoB feature):")
	for _, k := range dk {
		fmt.Printf("  %5d  %s\n", dropped[k], k)
	}
	byKind := map[string]int{}
	for _, id := range order {
		byKind[strings.SplitN(feats[id].kind, ":", 2)[0]]++
	}
	fmt.Printf("FEATURES %d  (parser %d, config %d, trigger %d, code %d)\n", len(order),
		byKind["parser"], byKind["config"], byKind["trigger"], byKind["code"])
	fmt.Println("corpus/shared/pool: -1 = not sizeable by this tool (size it by hand); corpus = presence " +
		"rung over item lines (parser) or <Input> (config)")

	var out strings.Builder
	out.WriteString("kind\tkey\tblocks\tstmts\tgo_site\tlua_marker\tcorpus_builds\tshared_builds\tpool_chars\tpob_uniques\tcandidates\n")
	for _, id := range order {
		f := feats[id]
		fmt.Fprintf(&out, "%s\t%s\t%d\t%d\t%s\t%s\t%d\t%d\t%d\t%s\t%s\n", f.kind, f.key, f.blocks, f.stmts,
			f.site, f.marker, f.corpus, f.shared, f.pool, strings.Join(f.uniques, ";"), strings.Join(f.cands, ","))
	}
	if tsv != "" {
		if err := os.WriteFile(tsv, []byte(out.String()), 0o600); err != nil { //nolint:gosec // operator path
			fmt.Fprintf(os.Stderr, "coverage-features: cannot write %s: %s\n", tsv, errnoText(err))
			return 1
		}
		fmt.Printf("wrote %s\n", tsv)
		return 0
	}
	fmt.Print(out.String())
	return 0
}

func cfAnchor(kind, pat string) string {
	switch kind {
	case "parser:specialModList":
		return "^" + pat + "$"
	case "parser:jewelSelfFuncs":
		return "^" + regexp.QuoteMeta(strings.ToLower(pat)) + "$"
	case "parser:preFlagList":
		return "^" + pat
	}
	return pat
}

func cfSkipPrefix(file string) string {
	for _, p := range cfSkipPrefixes {
		if strings.HasPrefix(file, p) {
			return p
		}
	}
	return ""
}

func cfAttribute(r cfRecord, lines []string) (string, string, string) {
	var marker string
	top, topVar := -1, ""
	for i := r.Start - 1; i >= 0 && i < len(lines); i-- {
		if cfTopLevelRE.MatchString(lines[i]) {
			top = i
			if m := cfTopVarRE.FindStringSubmatch(lines[i]); m != nil {
				topVar = m[1]
			}
			break
		}
	}
	marker = "-"
	inner := false
	for i := r.Start - 1; i > top && i < len(lines); i-- {
		if m := cfMarkerRE.FindStringSubmatch(lines[i]); m != nil {
			marker, inner = m[1], true
			break
		}
	}
	for j := top - 1; marker == "-" && j >= 0 && strings.HasPrefix(lines[j], "//"); j-- {
		if m := cfMarkerRE.FindStringSubmatch(lines[j]); m != nil {
			marker = m[1]
		}
	}

	switch r.File {
	case "calculator/mod_parser.go", "calculator/mod_parser_index.go":
		if topVar == "" {
			break
		}
		for i := r.Start - 1; i > top; i-- {
			m := cfRowKeyRE.FindStringSubmatch(lines[i])
			if m == nil {
				m = cfPatternFieldRE.FindStringSubmatch(lines[i])
			}
			if m == nil {
				continue
			}
			rowMarker := "-"
			for j := i - 1; j < r.Start && j > top; j++ {
				if mm := cfMarkerRE.FindStringSubmatch(lines[j]); mm != nil {
					rowMarker = mm[1]
					break
				}
			}
			return "parser:" + topVar, m[1], rowMarker
		}
	case "calculator/configs.go":
		for i := r.Start - 1; i > top; i-- {
			if m := cfConfigKeyRE.FindStringSubmatch(lines[i]); m != nil {
				return "config", m[1], marker
			}
		}
	case "calculator/calc_triggers.go":
		for i := r.Start - 1; i > top && topVar == ""; i-- {
			if m := cfCaseRE.FindStringSubmatch(lines[i]); m != nil {
				return "trigger", m[1], marker
			}
		}
	}
	if topVar != "" {
		for i := r.Start - 1; i > top; i-- {
			if m := cfRowKeyRE.FindStringSubmatch(lines[i]); m != nil {
				return "code", r.File + " var " + topVar + " " + m[1], marker
			}
		}
		return "code", r.File + " var " + topVar, marker
	}
	name := r.Func
	if top >= 0 {
		if m := cfFuncNameRE.FindStringSubmatch(lines[top]); m != nil {
			name = m[1]
		}
	}
	if inner {
		return "code", r.File + " " + name + " @" + marker, marker
	}
	return "code", r.File + " " + name, marker
}

func cfLoadCorpus(dir string) (map[string][]int, map[string]int, map[string]bool, int, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.xml"))
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("glob %s: %w", dir, err)
	}
	sort.Strings(files)
	lines := map[string][]int{}
	inputs := map[string]int{}
	ladder := map[string]bool{}
	for bi, path := range files {
		if m := cfLadderRE.FindStringSubmatch(filepath.Base(path)); m != nil {
			ladder[m[1]] = true
		}
		b, err := os.ReadFile(path) //nolint:gosec // a path this tool globbed
		if err != nil {
			return nil, nil, nil, 0, fmt.Errorf("read %s: %w", path, err)
		}
		s := string(b)
		seenIn := map[string]bool{}
		for _, m := range cfInputRE.FindAllStringSubmatch(s, -1) {
			if !seenIn[m[1]] {
				seenIn[m[1]] = true
				inputs[m[1]]++
			}
		}
		seen := map[string]bool{}
		texts := cfItemRE.FindAllStringSubmatch(s, -1)
		texts = append(texts, cfGemRE.FindAllStringSubmatch(s, -1)...)
		for _, m := range texts {
			for _, ln := range strings.Split(html.UnescapeString(m[1]), "\n") {
				ln = cfNormLine(ln)
				if ln == "" || seen[ln] {
					continue
				}
				seen[ln] = true
				lines[ln] = append(lines[ln], bi)
			}
		}
	}
	return lines, inputs, ladder, len(files), nil
}

func cfNormLine(ln string) string {
	ln = strings.TrimSpace(ln)
	ln = cfBraceRE.ReplaceAllString(ln, "")
	return strings.ToLower(strings.TrimSpace(ln))
}

func cfCorpusCount(rx *regexp.Regexp, lines map[string][]int) (int, []int) {
	builds := map[int]bool{}
	for ln, bs := range lines {
		if rx.MatchString(ln) {
			for _, b := range bs {
				builds[b] = true
			}
		}
	}
	out := make([]int, 0, len(builds))
	for b := range builds {
		out = append(out, b)
	}
	sort.Ints(out)
	return len(out), out
}

func cfPoolMatch(rx *regexp.Regexp, pool []cfPoolChar, limit int) (int, []string) {
	n := 0
	var ids []string
	for _, c := range pool {
		for _, ln := range c.lines {
			if rx.MatchString(ln) {
				n++
				if len(ids) < limit {
					ids = append(ids, c.id)
				}
				break
			}
		}
	}
	return n, ids
}

type cfJSONItem struct {
	Name          string            `json:"name"`
	TypeLine      string            `json:"typeLine"`
	ImplicitMods  []json.RawMessage `json:"implicitMods"`
	ExplicitMods  []json.RawMessage `json:"explicitMods"`
	CraftedMods   []json.RawMessage `json:"craftedMods"`
	EnchantMods   []json.RawMessage `json:"enchantMods"`
	FracturedMods []json.RawMessage `json:"fracturedMods"`
	CrucibleMods  []json.RawMessage `json:"crucibleMods"`
	ScourgeMods   []json.RawMessage `json:"scourgeMods"`
	UtilityMods   []json.RawMessage `json:"utilityMods"`
	Socketed      []cfJSONItem      `json:"socketedItems"`
	FrameType     int               `json:"frameType"`
}

const cfGemFrame = 4

func cfLoadPool(dir string, inCorpus map[string]bool) ([]cfPoolChar, int, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.items.json"))
	if err != nil {
		return nil, 0, fmt.Errorf("glob %s: %w", dir, err)
	}
	sort.Strings(files)
	var out []cfPoolChar
	skipped := 0
	for _, path := range files {
		id := strings.TrimSuffix(filepath.Base(path), ".items.json")
		if inCorpus[id] {
			skipped++
			continue
		}
		c := cfPoolChar{id: id}
		seen := map[string]bool{}
		for _, p := range []string{path, filepath.Join(dir, id+".passives.json")} {
			b, err := os.ReadFile(p) //nolint:gosec // a path this tool globbed
			if err != nil {
				continue
			}
			var doc struct {
				Items []cfJSONItem `json:"items"`
			}
			if json.Unmarshal(b, &doc) != nil {
				continue
			}
			for _, it := range doc.Items {
				cfItemLines(it, seen, &c.lines)
			}
		}
		out = append(out, c)
	}
	return out, skipped, nil
}

func cfItemLines(it cfJSONItem, seen map[string]bool, out *[]string) {
	add := func(s string) {
		for _, ln := range strings.Split(s, "\n") {
			ln = cfNormLine(ln)
			if ln != "" && !seen[ln] {
				seen[ln] = true
				*out = append(*out, ln)
			}
		}
	}
	add(it.Name)
	add(it.TypeLine)
	if it.FrameType == cfGemFrame {
		return
	}
	for _, group := range [][]json.RawMessage{it.ImplicitMods, it.ExplicitMods, it.CraftedMods, it.EnchantMods,
		it.FracturedMods, it.CrucibleMods, it.ScourgeMods, it.UtilityMods} {
		for _, m := range group {
			var s string
			if json.Unmarshal(m, &s) == nil {
				add(s)
				continue
			}
			var d struct {
				Description string `json:"description"`
			}
			if json.Unmarshal(m, &d) == nil {
				add(d.Description)
			}
		}
	}
	for _, s := range it.Socketed {
		cfItemLines(s, seen, out)
	}
}

var (
	cfUniqueBlockRE = regexp.MustCompile(`(?s)\[\[\n(.*?)\]\]`)
	cfRangeRE       = regexp.MustCompile(`\((-?[\d.]+)-(-?[\d.]+)\)`)
)

func cfLoadUniques(dir string) ([]cfPoolChar, error) {
	var out []cfPoolChar
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error { //nolint:gosec // the operator's --uniques directory
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".lua") {
			return err
		}
		b, err := os.ReadFile(path) //nolint:gosec // a path under the operator's --uniques
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		for _, m := range cfUniqueBlockRE.FindAllStringSubmatch(string(b), -1) {
			rows := strings.Split(m[1], "\n")
			c := cfPoolChar{id: strings.TrimSpace(rows[0])}
			for _, ln := range rows[1:] {
				if ln = cfNormLine(cfRangeRE.ReplaceAllString(ln, "$1")); ln != "" {
					c.lines = append(c.lines, ln)
				}
			}
			out = append(out, c)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", dir, err)
	}
	return out, nil
}
