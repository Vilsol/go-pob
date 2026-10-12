package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const ctUsage = `usage: config_tools.py <extract|inventory|compare|splice> [options]
  --pob DIR      the pinned PathOfBuilding checkout
  --repo DIR     the go-pob checkout (default: git rev-parse --show-toplevel)
  --corpus DIR   corpus to count config usage in
  --var NAME     repeatable
  --dry-run      splice only: print instead of writing`

var (
	ctBlockRE       = regexp.MustCompile(`(?s)/\*.*?\*/`)
	ctHandlerRE     = regexp.MustCompile(`(?m)^\t"([A-Za-z0-9_]+)":\s*func\(`)
	ctDeadHandlerRE = regexp.MustCompile(`(?m)^\s*"([A-Za-z0-9_]+)":\s*func\(`)
	ctLongBracketRE = regexp.MustCompile(`^(--)?\[(=*)\[`)
	ctVarRE         = regexp.MustCompile(`\bvar\s*=\s*"([^"]+)"`)
	ctTypeRE        = regexp.MustCompile(`\btype\s*=\s*"([^"]+)"`)
	ctSectionRE     = regexp.MustCompile(`\bsection\s*=\s*"([^"]+)"`)
	ctApplyRE       = regexp.MustCompile(`\bapply\s*=\s*function`)
	ctStringMaskRE  = regexp.MustCompile(`"(\\.|[^"\\])*"`)
	ctInputNameRE   = regexp.MustCompile(`<Input\b[^>]*\bname="([^"]+)"`)
	ctLeadWSRE      = regexp.MustCompile(`^[ \t]*`)
)

func ctRefuse(msg string) {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	os.Exit(3)
}

func ctReadText(path, what string) string {
	body, err := os.ReadFile(path) //nolint:gosec // an operator-supplied tree, as in the original
	if err != nil {
		ctRefuse(fmt.Sprintf("%s is not readable: %s", path, errnoText(err)))
	}
	if !utf8.Valid(body) {
		ctRefuse(fmt.Sprintf("%s is not valid UTF-8 (%s)", path, what))
	}
	return string(body)
}

func ctRepoRoot() string {
	cmd := exec.CommandContext(context.Background(), "git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.Error
		if errorsAs(err, &ee) {
			ctRefuse(fmt.Sprintf("cannot run git: %s", errnoText(err)))
		}
		ctRefuse("git rev-parse --show-toplevel failed; pass --repo, or run inside the checkout")
	}
	return strings.TrimSpace(string(out))
}

func ctStripLuaStrings(s string) string {
	var out strings.Builder
	i, n := 0, len(s)
	blank := func(text string) {
		for k := range len(text) {
			if text[k] == '\n' {
				out.WriteByte('\n')
			} else {
				out.WriteByte(' ')
			}
		}
	}
	for i < n {
		if m := ctLongBracketRE.FindStringSubmatch(s[i:]); m != nil {
			closer := "]" + m[2] + "]"
			rel := strings.Index(s[i+len(m[0]):], closer)
			end := n
			if rel >= 0 {
				end = i + len(m[0]) + rel + len(closer)
			}
			blank(s[i:end])
			i = end
			continue
		}
		if strings.HasPrefix(s[i:], "--") {
			for i < n && s[i] != '\n' {
				out.WriteByte(' ')
				i++
			}
			continue
		}
		if s[i] == '"' || s[i] == '\'' {
			q := s[i]
			out.WriteByte(' ')
			i++
			for i < n {
				if s[i] == '\n' {
					break
				}
				if s[i] == '\\' {
					out.WriteString("  ")
					i += 2
					continue
				}
				if s[i] == q {
					break
				}
				out.WriteByte(' ')
				i++
			}
			if i < n && s[i] == '\n' {
				continue
			}
			out.WriteByte(' ')
			i++
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

type ctOption struct {
	varName, typ, section    string
	hasVar, hasType, hasSect bool
	line, endline            int
	body, apply              string
	hasApply                 bool
	defaultState             bool
	defaultPlaceholderState  bool
}

func ctExtractOptions(pob string) []ctOption {
	path := pyPathJoin(pob, "src/Modules/ConfigOptions.lua")
	text := ctReadText(path, "ConfigOptions.lua")
	lines := strings.Split(text, "\n")
	masked := strings.Split(ctStripLuaStrings(text), "\n")

	start := -1
	for i, l := range masked {
		if strings.HasPrefix(l, "return {") {
			start = i
			break
		}
	}
	if start < 0 {
		ctRefuse(fmt.Sprintf("%s has no `return {` at column 0 -- this is not ConfigOptions.lua, "+
			"or its shape changed upstream and every option below would be silently missed", path))
	}

	type span struct{ a, b int }
	var opts []span
	depth := 0
	curStart := -1
	for i := start; i < len(masked); i++ {
		for _, ch := range []byte(masked[i]) {
			switch ch {
			case '{':
				if depth == 1 && curStart < 0 {
					curStart = i
				}
				depth++
			case '}':
				depth--
				if depth == 1 && curStart >= 0 {
					opts = append(opts, span{curStart, i})
					curStart = -1
				}
			}
		}
		if depth == 0 && i > start {
			break
		}
	}

	out := make([]ctOption, 0, len(opts))
	for _, sp := range opts {
		body := strings.Join(lines[sp.a:sp.b+1], "\n")
		rec := ctOption{
			line: sp.a + 1, endline: sp.b + 1, body: body,
			defaultState:            strings.Contains(body, "defaultState"),
			defaultPlaceholderState: strings.Contains(body, "defaultPlaceholderState"),
		}
		if m := ctVarRE.FindStringSubmatch(body); m != nil {
			rec.varName, rec.hasVar = m[1], true
		}
		if m := ctTypeRE.FindStringSubmatch(body); m != nil {
			rec.typ, rec.hasType = m[1], true
		}
		if m := ctSectionRE.FindStringSubmatch(body); m != nil {
			rec.section, rec.hasSect = m[1], true
		}
		if loc := ctApplyRE.FindStringIndex(body); loc != nil {
			rec.apply, rec.hasApply = body[loc[0]:], true
		}
		out = append(out, rec)
	}
	return out
}

func ctConfigsPath(repo string) string { return pyPathJoin(repo, "calculator/configs.go") }

const ctTableDecl = "var configurations = map[string]ConfigApplyFunc{"

func ctSplitLiveDead(repo string) ([]string, []string) {
	src := ctReadText(ctConfigsPath(repo), "configs.go")
	if !strings.Contains(src, ctTableDecl) {
		ctRefuse(fmt.Sprintf("%s has no `%s` -- the handler table moved or was renamed, and every "+
			"handler would read as absent", ctConfigsPath(repo), ctTableDecl))
	}
	body := src[strings.Index(src, ctTableDecl):]

	blocks := ctBlockRE.FindAllStringIndex(body, -1)
	live := make([]string, 0, len(blocks)+8)
	dead := make([]string, 0, len(blocks))
	pos := 0
	for _, loc := range blocks {
		live = append(live, ctAllSub(ctHandlerRE, body[pos:loc[0]])...)
		dead = append(dead, ctAllSub(ctDeadHandlerRE, body[loc[0]:loc[1]])...)
		pos = loc[1]
	}
	live = append(live, ctAllSub(ctHandlerRE, body[pos:])...)
	liveSet := map[string]bool{}
	for _, l := range live {
		liveSet[l] = true
	}
	seen := map[string]bool{}
	var deadOut []string
	for _, d := range dead {
		if seen[d] {
			continue
		}
		seen[d] = true
		if !liveSet[d] {
			deadOut = append(deadOut, d)
		}
	}
	return live, deadOut
}

func ctAllSub(re *regexp.Regexp, s string) []string {
	ms := re.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m[1])
	}
	return out
}

type ctDeadBlock struct {
	src                      string
	blockStart, blockEnd     int
	handlerStart, handlerEnd int
}

func ctFindDeadBlock(repo, v string) (ctDeadBlock, bool) {
	src := ctReadText(ctConfigsPath(repo), "configs.go")
	hre := regexp.MustCompile(`(?m)^([ \t]*)"` + regexp.QuoteMeta(v) + `":\s*func\(`)
	for _, loc := range ctBlockRE.FindAllStringIndex(src, -1) {
		block := src[loc[0]:loc[1]]
		h := hre.FindStringIndex(block)
		if h == nil {
			continue
		}
		masked := ctStringMaskRE.ReplaceAllStringFunc(block, func(m string) string {
			return strings.Repeat(" ", len(m))
		})
		p := h[1] - 1
		pdepth := 0
		for p < len(block) {
			if masked[p] == '(' {
				pdepth++
			} else if masked[p] == ')' {
				pdepth--
				if pdepth == 0 {
					break
				}
			}
			p++
		}
		rel := strings.Index(block[p:], "{")
		if rel < 0 {
			ctRefuse(fmt.Sprintf("the %s handler in %s has no opening brace after its parameter "+
				"list", v, ctConfigsPath(repo)))
		}
		i := p + rel
		depth := 0
		j := i
		for j < len(block) {
			switch masked[j] {
			case '{':
				depth++
			case '}':
				depth--
			}
			if masked[j] == '}' && depth == 0 {
				break
			}
			j++
		}
		end := j + 1
		if end < len(block) && block[end] == ',' {
			end++
		}
		return ctDeadBlock{src, loc[0], loc[1], loc[0] + h[0], loc[0] + end}, true
	}
	return ctDeadBlock{}, false
}

func ctDedentHandler(text string) string {
	lines := strings.Split(text, "\n")
	base := ctLeadWSRE.FindString(lines[0])
	out := make([]string, len(lines))
	for i, l := range lines {
		if strings.HasPrefix(l, base) {
			l = l[len(base):]
		} else {
			l = strings.TrimLeft(l, " \t\n\v\f\r")
		}
		if strings.TrimSpace(l) != "" {
			out[i] = "\t" + l
		} else {
			out[i] = ""
		}
	}
	return strings.Join(out, "\n")
}

func ctSplice(repo string, vars []string, dryRun bool) int {
	for _, v := range vars {
		found, ok := ctFindDeadBlock(repo, v)
		if !ok {
			fmt.Fprintf(os.Stderr, "NOT FOUND (or already live): %s\n", v)
			return 1
		}
		src := found.src
		prefix := src[found.blockStart+2 : found.handlerStart]
		suffix := src[found.handlerEnd:found.blockEnd]
		suffix = suffix[:len(suffix)-2]

		handler := ctDedentHandler(strings.TrimSpace(src[found.handlerStart:found.handlerEnd]))
		if !strings.HasSuffix(handler, ",") {
			handler += ","
		}
		handler = strings.ReplaceAll(handler,
			"func(val interface{}, modList *ModList, enemyModList *ModList)",
			"func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList)")

		var parts []string
		if strings.TrimSpace(prefix) != "" {
			parts = append(parts, "/*"+strings.TrimRight(prefix, " \t\n\v\f\r")+"\n\t*/\n")
		}
		parts = append(parts, handler+"\n")
		if strings.TrimSpace(suffix) != "" {
			parts = append(parts, "\t/*\n"+strings.TrimRight(suffix, "\n \t")+"\n\t*/")
		}
		newText := strings.TrimRight(strings.Join(parts, ""), "\n")

		out := src[:found.blockStart] + newText + src[found.blockEnd:]
		if dryRun {
			fmt.Printf("--- %s ---\n", v)
			fmt.Println(newText)
		} else {
			if err := os.WriteFile(ctConfigsPath(repo), []byte(out), 0o600); err != nil {
				ctRefuse(fmt.Sprintf("cannot write %s: %s", ctConfigsPath(repo), errnoText(err)))
			}
		}
	}
	if !dryRun {
		//nolint:gosec // the path is the operator's own --repo, passed as one argv element
		cmd := exec.CommandContext(context.Background(), "gofmt", "-w", ctConfigsPath(repo))
		if err := cmd.Run(); err != nil {
			var ee *exec.Error
			if errorsAs(err, &ee) {
				ctRefuse(fmt.Sprintf("cannot run gofmt: %s", errnoText(err)))
			}
			fmt.Fprintln(os.Stderr, "gofmt failed -- a spliced draft is not valid Go yet")
		}
	}
	return 0
}

func ctCorpusCounts(corpus string) (map[string]int, []string) {
	counts := map[string]int{}
	var order []string
	entries, err := os.ReadDir(corpus)
	if err != nil {
		return counts, order
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		text := ctReadText(pyPathJoin(corpus, e.Name()), "a corpus build")
		seen := map[string]bool{}
		for _, m := range ctInputNameRE.FindAllStringSubmatch(text, -1) {
			seen[m[1]] = true
		}
		names := make([]string, 0, len(seen))
		for n := range seen {
			names = append(names, n)
		}
		for _, n := range names {
			if _, had := counts[n]; !had {
				order = append(order, n)
			}
			counts[n]++
		}
	}
	return counts, order
}

type ctArgs struct {
	cmd, pob, repo, corpus string
	vars                   []string
	dryRun                 bool
}

func ctParseArgs(argv []string) ctArgs {
	valued := map[string]bool{"--pob": true, "--repo": true, "--corpus": true, "--var": true}
	cmds := []string{"extract", "inventory", "compare", "splice"}
	home, _ := pyPathHome()
	a := ctArgs{pob: pyPathJoin(home, "PathOfBuilding")}
	for i := 0; i < len(argv); {
		x := argv[i]
		switch {
		case valued[x]:
			if i+1 >= len(argv) {
				ctRefuse(fmt.Sprintf("%s needs a value\n%s", x, ctUsage))
			}
			switch x {
			case "--var":
				a.vars = append(a.vars, argv[i+1])
			case "--pob":
				a.pob = argv[i+1]
			case "--repo":
				a.repo = argv[i+1]
			case "--corpus":
				a.corpus = argv[i+1]
			}
			i += 2
		case x == "--dry-run":
			a.dryRun = true
			i++
		case strings.HasPrefix(x, "-"):
			ctRefuse(fmt.Sprintf("unknown argument %s\n%s", pyRepr(x), ctUsage))
		case a.cmd == "":
			known := false
			for _, c := range cmds {
				if c == x {
					known = true
				}
			}
			if !known {
				ctRefuse(fmt.Sprintf("unknown command %s; expected one of %s\n%s",
					pyRepr(x), strings.Join(cmds, ", "), ctUsage))
			}
			a.cmd = x
			i++
		default:
			ctRefuse(fmt.Sprintf("unexpected argument %s\n%s", pyRepr(x), ctUsage))
		}
	}
	if a.cmd == "" {
		ctRefuse(fmt.Sprintf("a command is required\n%s", ctUsage))
	}
	return a
}

func configTools(argv []string) int {
	args := ctParseArgs(argv)
	repo := args.repo
	if repo == "" {
		repo = ctRepoRoot()
	}
	corpus := args.corpus
	if corpus == "" {
		corpus = pyPathJoin(repo, "testdata/corpus-3.29")
	}

	if args.cmd == "splice" {
		return ctSplice(repo, args.vars, args.dryRun)
	}

	opts := ctExtractOptions(args.pob)
	byVar := map[string]ctOption{}
	var varOrder []string
	for _, o := range opts {
		if o.hasVar {
			if _, had := byVar[o.varName]; !had {
				varOrder = append(varOrder, o.varName)
			}
			byVar[o.varName] = o
		}
	}

	switch args.cmd {
	case "extract":
		sel := args.vars
		if len(sel) == 0 {
			sel = append([]string{}, varOrder...)
			sort.Strings(sel)
		}
		for _, v := range sel {
			o, ok := byVar[v]
			if !ok {
				fmt.Printf("!! no such option: %s\n", v)
				continue
			}
			ds, dps := "", ""
			if o.defaultState {
				ds = "  defaultState"
			}
			if o.defaultPlaceholderState {
				dps = "  defaultPlaceholderState"
			}
			fmt.Printf("=== %s  type=%s  section=%s  L%d-%d%s%s\n",
				v, ctNone(o.typ, o.hasType), ctNone(o.section, o.hasSect), o.line, o.endline, ds, dps)
			if o.hasApply {
				fmt.Println(o.apply)
			} else {
				fmt.Println("    (no apply function)")
			}
			fmt.Println()
		}
		return 0

	case "compare":
		live, _ := ctSplitLiveDead(repo)
		liveSet := map[string]bool{}
		for _, l := range live {
			liveSet[l] = true
		}
		for _, v := range args.vars {
			o, ok := byVar[v]
			t := "?"
			if ok {
				t = ctNone(o.typ, o.hasType)
			}
			fmt.Printf("=== %s  type=%s\n", v, t)
			fmt.Println("--- PoB ---")
			if ok && o.hasApply {
				fmt.Println(o.apply)
			} else {
				fmt.Println("(none)")
			}
			state := "DEAD"
			if liveSet[v] {
				state = "LIVE"
			}
			fmt.Printf("--- go-pob (%s) ---\n", state)
			if f, found := ctFindDeadBlock(repo, v); found {
				fmt.Println(f.src[f.handlerStart:f.handlerEnd])
			} else {
				src := ctReadText(ctConfigsPath(repo), "configs.go")
				re := regexp.MustCompile(`(?m)^\t"` + regexp.QuoteMeta(v) + `":\s*func\(`)
				if loc := re.FindStringIndex(src); loc != nil {
					end := loc[0] + 1200
					if end > len(src) {
						end = len(src)
					}
					fmt.Println(src[loc[0]:end])
				} else {
					fmt.Println("(not found)")
				}
			}
			fmt.Println()
		}
		return 0
	}

	return ctInventory(repo, corpus, byVar, varOrder)
}

func ctNone(v string, present bool) string {
	if !present {
		return "None"
	}
	return v
}

func ctInventory(repo, corpus string, byVar map[string]ctOption, varOrder []string) int {
	live, dead := ctSplitLiveDead(repo)
	counts := map[string]int{}
	var countOrder []string
	if fi, err := os.Stat(corpus); err == nil && fi.IsDir() {
		counts, countOrder = ctCorpusCounts(corpus)
	}
	known := map[string]bool{}
	for _, v := range live {
		known[v] = true
	}
	for _, v := range dead {
		known[v] = true
	}
	var absent []string
	for _, v := range varOrder {
		if !known[v] {
			absent = append(absent, v)
		}
	}
	var unknown []string
	for _, v := range countOrder {
		if !known[v] {
			if _, declared := byVar[v]; !declared {
				unknown = append(unknown, v)
			}
		}
	}

	deadSet := 0
	for _, v := range dead {
		if counts[v] > 0 {
			deadSet++
		}
	}
	absentSet := 0
	for _, v := range absent {
		if counts[v] > 0 {
			absentSet++
		}
	}
	fmt.Printf("INVENTORY live=%d dead=%d absent=%d dead-and-set=%d absent-and-set=%d "+
		"undeclared-and-set=%d\n", len(live), len(dead), len(absent), deadSet, absentSet, len(unknown))
	fmt.Printf("live    %d\n", len(live))
	fmt.Printf("dead    %d\n", len(dead))
	fmt.Printf("absent  %d  (in ConfigOptions.lua, no entry in configs.go's table)\n", len(absent))
	fmt.Println("        ABSENT MEANS 'NO TABLE ENTRY', NOT 'UNIMPLEMENTED'. Several of the largest")
	fmt.Println("        absent vars are fully implemented by bespoke paths and adding a handler")
	fmt.Println("        would DOUBLE-APPLY them: bandit and both pantheon vars (417 builds each)")
	fmt.Println("        via env.go/applySoulMod, enemyLevel at env.go:44, and the whole")
	fmt.Println("        enemy*Damage family consumed directly at calc_defence.go:915. Check for a")
	fmt.Println("        bespoke consumer before ranking anything in this column as work.")
	fmt.Printf("dead-and-set-by-a-corpus-build   %d\n", deadSet)
	fmt.Printf("absent-and-set-by-a-corpus-build %d\n", absentSet)
	seen := map[string]int{}
	var dupOrder []string
	for _, v := range live {
		if seen[v] == 0 {
			dupOrder = append(dupOrder, v)
		}
		seen[v]++
	}
	var dup []string
	for _, v := range dupOrder {
		if seen[v] > 1 {
			dup = append(dup, v)
		}
	}
	if len(dup) > 0 {
		fmt.Printf("DUPLICATE LIVE KEYS: %s\n", pyReprValue(dup))
	}
	fmt.Println()
	fmt.Printf("%-42s %-16s %6s %s\n", "var", "type", "builds", "state")
	sortedUnknown := append([]string{}, unknown...)
	sort.Strings(sortedUnknown)
	fmt.Printf("undeclared-and-set (not in ConfigOptions.lua at all) %d: %s\n",
		len(unknown), pyReprValue(sortedUnknown))

	type row struct{ v, state string }
	rows := make([]row, 0, len(dead)+len(absent))
	for _, v := range dead {
		rows = append(rows, row{v, "DEAD"})
	}
	for _, v := range absent {
		rows = append(rows, row{v, "ABSENT"})
	}
	sort.SliceStable(rows, func(i, j int) bool { return counts[rows[i].v] > counts[rows[j].v] })
	for _, r := range rows {
		if counts[r.v] == 0 {
			continue
		}
		t := "-"
		if o, ok := byVar[r.v]; ok && o.hasType && o.typ != "" {
			t = o.typ
		}
		fmt.Printf("%-42s %-16s %6d %s\n", r.v, t, counts[r.v], r.state)
	}
	return 0
}
