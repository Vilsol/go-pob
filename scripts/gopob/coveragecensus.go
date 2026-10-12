package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
)

const ccModule = "github.com/Vilsol/go-pob/"

var (
	reCovBlock       = regexp.MustCompile(`^(.*):(\d+)\.(\d+),(\d+)\.(\d+) (\d+) (\d+)$`)
	reDefensiveGuard = regexp.MustCompile(
		`(==\s*nil|!=\s*nil|err\s*!=\s*nil|!ok\b|!found\b|!exists\b|!known\b` +
			`|len\([^)]*\)\s*[=<]=?\s*0|<\s*0|<\s*1)`)
	reDefensiveBody = regexp.MustCompile(
		`^(?:\}\s*)?(?:else\s+)?if\s+(?P<cond>.*?)\s*\{\s*` +
			`(?:return\b[^{}]*|continue|break|panic\([^{}]*\))\s*\}$`)
	reDefensiveSplit = regexp.MustCompile(`\|\||&&`)
	reMinion         = regexp.MustCompile(`\bMinion|AppliesToMinions`)
)

var ccEntryPointFiles = map[string]bool{
	"calculator/variant.go":           true,
	"calculator/item_cache.go":        true,
	"calculator/calc_tab_finalize.go": true,
	"calculator/output.go":            true,
	"data/tree_search.go":             true,
	"pob/methods_saved.go":            true,
	"pob/utils.go":                    true,
	"pob/types_saved.go":              true,
	"builds/parser.go":                true,
}

type ccRecord struct {
	file       string
	start, end int
	stmts      int
	testonly   *bool
	fn         string
	fnLine     int
	hasFn      bool
	guard      []string
	guardtext  string
	body       string
	bucket     string
	reason     string
}

type ccRule struct {
	bucket, reason string
	test           func(*ccRecord) bool
}

var ccRules = []ccRule{
	{"data", "calcs-mode", func(r *ccRecord) bool {
		return strings.Contains(r.guardtext+r.body, "OutputModeCalcs")
	}},
	{"data", "config-unset-arm", func(r *ccRecord) bool {
		return r.file == "calculator/configs.go" && strings.HasPrefix(r.body, "if !configChecked(val)")
	}},
	{"data", "config-type-assert", func(r *ccRecord) bool {
		return r.file == "calculator/configs.go" && strings.HasPrefix(r.body, "if !ok")
	}},
	{"data", "config-option", func(r *ccRecord) bool { return r.file == "calculator/configs.go" }},
	{"data", "mod-pattern", func(r *ccRecord) bool {
		return r.file == "calculator/mod_parser.go" || r.file == "calculator/mod_parser_index.go"
	}},
	{"data", "minion-data", func(r *ccRecord) bool {
		return reMinion.MatchString(r.guardtext + r.body)
	}},
	{"data", "entry-point", func(r *ccRecord) bool { return ccEntryPointFiles[r.file] }},
	{"data", "defensive-guard", func(r *ccRecord) bool {
		return r.stmts == 1 && ccIsDefensive(r.body)
	}},
}

func ccIsDefensive(body string) bool {
	m := reDefensiveBody.FindStringSubmatch(body)
	if m == nil {
		return false
	}
	cond := m[reDefensiveBody.SubexpIndex("cond")]
	for _, part := range reDefensiveSplit.Split(cond, -1) {
		if !reDefensiveGuard.MatchString(part) {
			return false
		}
	}
	return true
}

type ccArgs struct {
	profile, overlay, funcs, root, bucket, jsonOut string
	list                                           bool
}

const ccUsage = "usage: coverage_census.py --profile P [--overlay P] [--funcs F] [--root R] " +
	"[--list] [--bucket B] [--json J]"

func coverageCensus(argv []string) int {
	args, rc := ccParseArgs(argv)
	if rc != 0 {
		return rc
	}
	corpus, rc := ccLoadProfile(args.profile)
	if rc != 0 {
		return rc
	}
	var overlay map[string]ccBlock
	if args.overlay != "" {
		overlay, rc = ccLoadProfile(args.overlay)
		if rc != 0 {
			return rc
		}
	}
	var funcs map[string][]ccFunc
	if args.funcs != "" {
		funcs, rc = ccLoadFuncs(args.funcs)
		if rc != 0 {
			return rc
		}
	}
	src := &ccSource{root: args.root, cache: map[string][]string{}}

	totalBlocks := len(corpus)
	totalStmts := 0
	var deadKeys []string
	for k, b := range corpus {
		totalStmts += b.stmts
		if b.count == 0 {
			deadKeys = append(deadKeys, k)
		}
	}
	sort.Slice(deadKeys, func(i, j int) bool { return ccKeyLess(corpus[deadKeys[i]], corpus[deadKeys[j]]) })

	records := make([]*ccRecord, 0, len(deadKeys))
	for _, k := range deadKeys {
		b := corpus[k]
		guard := src.guardChain(b.file, b.sl)
		r := &ccRecord{
			file: b.file, start: b.sl, end: b.el, stmts: b.stmts,
			guard: guard, guardtext: strings.Join(guard, " "),
			body: src.snippet(b.file, b.sl, b.el, 160),
		}
		if args.overlay != "" {
			t := overlay[k].count != 0
			r.testonly = &t
		}
		if fn, ok := ccEnclosingFunc(funcs, b.file, b.sl); ok {
			r.fn, r.fnLine, r.hasFn = fn.name, fn.line, true
		}
		r.bucket, r.reason = ccClassify(r)
		records = append(records, r)
	}

	fmt.Println("=== UNEXERCISED-CODE CENSUS ===")
	fmt.Printf("instrumented blocks %d   statements %d\n", totalBlocks, totalStmts)
	deadStmts := 0
	for _, r := range records {
		deadStmts += r.stmts
	}
	fmt.Printf("never executed      %d blocks (%.1f%%)   %d statements (%.1f%%)\n",
		len(records), 100*float64(len(records))/float64(totalBlocks),
		deadStmts, 100*float64(deadStmts)/float64(totalStmts))
	if args.overlay != "" {
		testonly := 0
		for _, r := range records {
			if r.testonly != nil && *r.testonly {
				testonly++
			}
		}
		fmt.Printf("  of which reached by a unit test but by no corpus build: %d\n", testonly)
		fmt.Printf("  executed by nothing in the repository:                  %d\n",
			len(records)-testonly)
	}

	fmt.Println("\n--- buckets ---")
	for _, e := range ccCount(records, func(r *ccRecord) string { return r.bucket + "\x00" + r.reason }, nil) {
		parts := strings.SplitN(e.key, "\x00", 2)
		extra := ""
		if args.overlay != "" {
			n := 0
			for _, r := range records {
				if r.testonly != nil && *r.testonly && r.bucket+"\x00"+r.reason == e.key {
					n++
				}
			}
			extra = fmt.Sprintf("   (%d testonly)", n)
		}
		fmt.Printf("  %5d  %-14s %s%s\n", e.n, parts[0], parts[1], extra)
	}

	fmt.Println("\n--- unclassified, by file ---")
	for _, e := range ccCount(records, func(r *ccRecord) string { return r.file },
		func(r *ccRecord) bool { return r.bucket == "unclassified" }) {
		fmt.Printf("  %5d  %s\n", e.n, e.key)
	}

	if args.list {
		fmt.Println("\n--- dead blocks ---")
		for _, r := range records {
			if args.bucket != "" && r.bucket != args.bucket {
				continue
			}
			tag := "DEAD"
			if args.overlay != "" && r.testonly != nil && *r.testonly {
				tag = "TESTONLY"
			}
			fn := "?"
			if r.hasFn {
				fn = r.fn
			}
			fmt.Printf("\n%s %s:%d-%d [%s/%s] %s\n", tag, r.file, r.start, r.end, r.bucket, r.reason, fn)
			for _, g := range r.guard[minInt(1, len(r.guard)):] {
				fmt.Printf("    ^ %s\n", truncRunes(g, 150))
			}
			fmt.Printf("    > %s\n", r.body)
		}
	}

	if args.jsonOut != "" {
		if err := os.WriteFile(args.jsonOut, []byte(ccJSON(records)), 0o600); err != nil { //nolint:gosec // operator path
			fmt.Fprintf(os.Stderr, "coverage-census: cannot write %s: %s\n", args.jsonOut, errnoText(err))
			return 1
		}
	}
	return 0
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ccClassify(r *ccRecord) (string, string) {
	for _, rule := range ccRules {
		if rule.test(r) {
			return rule.bucket, rule.reason
		}
	}
	return "unclassified", "-"
}

type ccTally struct {
	key string
	n   int
}

func ccCount(records []*ccRecord, key func(*ccRecord) string, keep func(*ccRecord) bool) []ccTally {
	order := []string{}
	seen := map[string]int{}
	for _, r := range records {
		if keep != nil && !keep(r) {
			continue
		}
		k := key(r)
		if _, ok := seen[k]; !ok {
			order = append(order, k)
		}
		seen[k]++
	}
	out := make([]ccTally, 0, len(order))
	for _, k := range order {
		out = append(out, ccTally{k, seen[k]})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].n > out[j].n })
	return out
}

func ccParseArgs(argv []string) (ccArgs, int) {
	a := ccArgs{root: "."}
	valued := map[string]*string{
		"--profile": &a.profile, "--overlay": &a.overlay, "--funcs": &a.funcs,
		"--root": &a.root, "--bucket": &a.bucket, "--json": &a.jsonOut,
	}
	for i := 0; i < len(argv); {
		switch arg := argv[i]; {
		case valued[arg] != nil:
			if i+1 >= len(argv) {
				fmt.Fprintf(os.Stderr, "coverage-census: %s needs a value\n%s\n", arg, ccUsage)
				return a, 1
			}
			*valued[arg] = argv[i+1]
			i += 2
		case arg == "--list":
			a.list = true
			i++
		default:
			fmt.Fprintf(os.Stderr, "coverage-census: unknown argument %s\n%s\n", pyRepr(arg), ccUsage)
			return a, 1
		}
	}
	if a.profile == "" {
		fmt.Fprintf(os.Stderr, "coverage-census: --profile is required\n%s\n", ccUsage)
		return a, 1
	}
	return a, 0
}

type ccBlock struct {
	file           string
	sl, sc, el, ec int
	stmts, count   int
}

func ccKeyLess(a, b ccBlock) bool {
	switch {
	case a.file != b.file:
		return a.file < b.file
	case a.sl != b.sl:
		return a.sl < b.sl
	case a.sc != b.sc:
		return a.sc < b.sc
	case a.el != b.el:
		return a.el < b.el
	default:
		return a.ec < b.ec
	}
}

func ccLoadProfile(path string) (map[string]ccBlock, int) {
	b, err := os.ReadFile(path) //nolint:gosec // an operator-supplied path
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage-census: cannot read %s: %s\n", path, errnoText(err))
		return nil, 1
	}
	blocks := map[string]ccBlock{}
	for _, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		m := reCovBlock.FindStringSubmatch(line)
		if m == nil {
			fmt.Fprintf(os.Stderr, "%s: unparsed profile line: %s\n", path, line)
			return nil, 1
		}
		name := strings.TrimPrefix(m[1], ccModule)
		blk := ccBlock{file: name,
			sl: atoi(m[2]), sc: atoi(m[3]), el: atoi(m[4]), ec: atoi(m[5]),
			stmts: atoi(m[6]), count: atoi(m[7])}
		k := fmt.Sprintf("%s\x00%d\x00%d\x00%d\x00%d", blk.file, blk.sl, blk.sc, blk.el, blk.ec)
		if prev, ok := blocks[k]; ok {
			blk.stmts = prev.stmts
			blk.count = prev.count + blk.count
		}
		blocks[k] = blk
	}
	return blocks, 0
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

type ccFunc struct {
	line int
	name string
	pct  float64
}

func ccLoadFuncs(path string) (map[string][]ccFunc, int) {
	b, err := os.ReadFile(path) //nolint:gosec // an operator-supplied path
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage-census: cannot read %s: %s\n", path, errnoText(err))
		return nil, 1
	}
	out := map[string][]ccFunc{}
	for _, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		if line == "" || strings.HasPrefix(line, "total:") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		loc, name, pct := f[0], f[1], f[2]
		parts := strings.Split(loc, ":")
		head := parts[0]
		ln := atoi(parts[1])
		head = strings.TrimPrefix(head, ccModule)
		v, _ := strconv.ParseFloat(strings.TrimSuffix(pct, "%"), 64)
		out[head] = append(out[head], ccFunc{ln, name, v})
	}
	for _, v := range out {
		sort.Slice(v, func(i, j int) bool {
			if v[i].line != v[j].line {
				return v[i].line < v[j].line
			}
			return v[i].name < v[j].name
		})
	}
	return out, 0
}

func ccEnclosingFunc(funcs map[string][]ccFunc, rel string, line int) (ccFunc, bool) {
	var best ccFunc
	found := false
	for _, f := range funcs[rel] {
		if f.line <= line {
			best, found = f, true
		} else {
			break
		}
	}
	return best, found
}

type ccSource struct {
	root  string
	cache map[string][]string
}

func (s *ccSource) lines(rel string) []string {
	if v, ok := s.cache[rel]; ok {
		return v
	}
	b, err := os.ReadFile(filepath.Join(s.root, rel)) //nolint:gosec // a repo-relative source path
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage-census: cannot read %s: %s\n", rel, errnoText(err))
		os.Exit(1)
	}
	s.cache[rel] = strings.Split(string(b), "\n")
	return s.cache[rel]
}

func (s *ccSource) guardChain(rel string, line int) []string {
	src := s.lines(rel)
	idx := line - 1
	if idx >= len(src) {
		return []string{}
	}
	depth := len(src[idx]) - len(strings.TrimLeft(src[idx], "\t"))
	var chain []string
	for i := idx - 1; i >= 0; i-- {
		text := src[i]
		stripped := strings.TrimSpace(text)
		if stripped == "" {
			continue
		}
		d := len(text) - len(strings.TrimLeft(text, "\t"))
		if d < depth && strings.HasSuffix(stripped, "{") {
			chain = append(chain, stripped)
			depth = d
			if d == 0 {
				break
			}
		}
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	if chain == nil {
		return []string{}
	}
	return chain
}

func (s *ccSource) snippet(rel string, sl, el, width int) string {
	src := s.lines(rel)
	var body []string
	for i := sl - 1; i < el && i < len(src); i++ {
		if t := strings.TrimSpace(src[i]); t != "" {
			body = append(body, t)
		}
	}
	return truncRunes(strings.Join(body, " "), width)
}

func ccJSON(records []*ccRecord) string {
	var b strings.Builder
	if len(records) == 0 {
		return "[]"
	}
	b.WriteString("[\n")
	for i, r := range records {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString(" {\n")
		fields := []struct {
			k, v string
		}{
			{"body", pyJSONString(r.body)},
			{"bucket", pyJSONString(r.bucket)},
			{"end", strconv.Itoa(r.end)},
			{"file", pyJSONString(r.file)},
			{"func", ccNullableString(r.hasFn, r.fn)},
			{"funcline", ccNullableInt(r.hasFn, r.fnLine)},
			{"guard", pyJSONStringList(r.guard)},
			{"guardtext", pyJSONString(r.guardtext)},
			{"reason", pyJSONString(r.reason)},
			{"start", strconv.Itoa(r.start)},
			{"stmts", strconv.Itoa(r.stmts)},
			{"testonly", ccJSONBool(r.testonly)},
		}
		for j, f := range fields {
			if j > 0 {
				b.WriteString(",\n")
			}
			fmt.Fprintf(&b, "  %s: %s", pyJSONString(f.k), f.v)
		}
		b.WriteString("\n }")
	}
	b.WriteString("\n]")
	return b.String()
}

func ccNullableString(ok bool, v string) string {
	if !ok {
		return "null"
	}
	return pyJSONString(v)
}

func ccNullableInt(ok bool, v int) string {
	if !ok {
		return "null"
	}
	return strconv.Itoa(v)
}

func ccJSONBool(v *bool) string {
	switch {
	case v == nil:
		return "null"
	case *v:
		return "true"
	default:
		return "false"
	}
}

func pyJSONStringList(xs []string) string {
	if len(xs) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteString("[\n")
	for i, x := range xs {
		if i > 0 {
			b.WriteString(",\n")
		}
		fmt.Fprintf(&b, "   %s", pyJSONString(x))
	}
	b.WriteString("\n  ]")
	return b.String()
}

func pyJSONString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\b':
			b.WriteString(`\b`)
		case r == '\f':
			b.WriteString(`\f`)
		case r >= 0x20 && r <= 0x7e:
			b.WriteRune(r)
		case r > 0xffff:
			hi, lo := utf16.EncodeRune(r)
			fmt.Fprintf(&b, `\u%04x\u%04x`, hi, lo)
		default:
			fmt.Fprintf(&b, `\u%04x`, r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
