package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var orTables = [][3]string{
	{
		"scripts/gopob/skillstatmapemit.go",
		"data/raw/skill_stat_maps.go",
		"are not representable and are omitted rather than mapped lossily",
	},
}

var orGoRoots = []string{"mod", "moddb", "calculator", "data"}

var (
	orReasonRE     = regexp.MustCompile(`mcUnrep\(\s*((?:"(?:[^"\\]|\\.)*"\s*\+?\s*)+)`)
	orLiteralRE    = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)
	orWordRE       = regexp.MustCompile(`^\w+$`)
	orProvRE       = regexp.MustCompile(`\[(pob|go|go-absent|unverifiable):\s*(.*?)\]\s*$`)
	orHeaderLineRE = regexp.MustCompile(`^//\s+-\s+(\d+)\s+(.*?)\s*$`)
	orEntryRE      = regexp.MustCompile(`^([\w.\-/]+\.lua)(?::(\d+)(?:-(\d+))?)?$`)
	orIdentRE      = regexp.MustCompile(`^[\w.]+$`)
)

type orReason struct {
	line int
	text string
}

type orHeader struct {
	line, n int
	text    string
}

func orEmitterReasons(path string) []orReason {
	src := pcReadText(path, "the emitter")
	ms := orReasonRE.FindAllStringSubmatchIndex(src, -1)
	out := make([]orReason, 0, len(ms))
	for _, m := range ms {
		var lit string
		for _, x := range orLiteralRE.FindAllStringSubmatch(src[m[2]:m[3]], -1) {
			lit += x[1]
		}
		lit = strings.ReplaceAll(lit, `\"`, `"`)
		lit = strings.ReplaceAll(lit, `\\`, `\`)
		out = append(out, orReason{strings.Count(src[:m[0]], "\n") + 1, lit})
	}
	return out
}

func orHeaderReasons(path, opener string) []orHeader {
	lines := strings.Split(pcReadText(path, "the generated table"), "\n")
	var out []orHeader
	inside := false
	for i, l := range lines {
		if strings.Contains(l, opener) {
			inside = true
			continue
		}
		if !inside {
			continue
		}
		m := orHeaderLineRE.FindStringSubmatch(l)
		if m == nil {
			break
		}
		n, _ := strconv.Atoi(m[1])
		out = append(out, orHeader{i + 1, n, m[2]})
	}
	return out
}

func orGoFiles(root string) []string {
	generated := map[string]bool{}
	for _, t := range orTables {
		generated[t[1]] = true
	}
	var out []string
	for _, base := range orGoRoots {
		d := pyPathJoin(root, base)
		_ = filepath.WalkDir(d, func(p string, e os.DirEntry, err error) error { //nolint:gosec // an operator-supplied root
			if err != nil {
				return nil //nolint:nilerr // a missing package directory is not this gate's business
			}
			if e.IsDir() {
				if e.Name() == ".git" || e.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
				if rel, rerr := filepath.Rel(root, p); rerr == nil &&
					generated[filepath.ToSlash(rel)] {
					return nil
				}
				out = append(out, p)
			}
			return nil
		})
	}
	return out
}

var orGoCache = map[string]string{}

func orGoDefines(root, ident string) bool {
	blob, ok := orGoCache[root]
	if !ok {
		files := orGoFiles(root)
		parts := make([]string, 0, len(files))
		for _, p := range files {
			parts = append(parts, pcReadText(p, "Go source"))
		}
		blob = strings.Join(parts, "\n")
		orGoCache[root] = blob
	}
	needle := ident
	if orIdentRE.MatchString(ident) {
		bits := strings.Split(ident, ".")
		needle = bits[len(bits)-1]
	}
	if orWordRE.MatchString(needle) {
		return regexp.MustCompile(`\b` + regexp.QuoteMeta(needle) + `\b`).MatchString(blob)
	}
	return strings.Contains(blob, needle)
}

func orCheckProvenance(root, pob string, lines map[string]int, where, reason string,
	fail func(code, where, msg string)) {
	m := orProvRE.FindStringSubmatch(reason)
	if m == nil {
		fail("NO-PROVENANCE", where,
			fmt.Sprintf("reason carries no [pob:...] / [go:...] / [go-absent:...] / "+
				"[unverifiable:...] marker: %s", pyRepr(reason)))
		return
	}
	kind, body := m[1], strings.TrimSpace(m[2])

	if body == "" {
		fail("EMPTY-WHY", where,
			fmt.Sprintf("[%s:] carries nothing -- an [unverifiable:] must say WHY nothing can be "+
				"cited, and the other kinds must name a file or a symbol", kind))
		return
	}
	switch kind {
	case "unverifiable":
		return
	case "go":
		if !orGoDefines(root, body) {
			fail("GO-MISSING", where,
				fmt.Sprintf("[go:%s] names a symbol that is no longer in the Go tree, so the "+
					"reason no longer parses against reality", body))
		}
		return
	case "go-absent":
		if orGoDefines(root, body) {
			fail("GO-STALE", where,
				fmt.Sprintf("[go-absent:%s] claims that symbol is missing, but it EXISTS now -- "+
					"the omission reason is stale and the mods it drops may be portable", body))
		}
		return
	}

	em := orEntryRE.FindStringSubmatch(body)
	if em == nil {
		fail("MALFORMED", where, fmt.Sprintf("[pob:%s] is not path.lua[:start[-end]]", body))
		return
	}
	path := em[1]
	cands := pcCandidates(path, lines)
	if len(cands) == 0 {
		fail("NOFILE", where, fmt.Sprintf("[pob:%s] names no file in the pinned PoB", path))
		return
	}
	rel := cands[0]
	if em[2] == "" {
		return
	}
	start, _ := strconv.Atoi(em[2])
	end := start
	if em[3] != "" {
		end, _ = strconv.Atoi(em[3])
	}
	if end < start || end > lines[rel] {
		fail("RANGE", where, fmt.Sprintf("[pob:%s] range %d-%d is outside %s (%d lines)",
			path, start, end, rel, lines[rel]))
		return
	}
	if !pcLuaHasCode(pob, rel, start, end) {
		fail("EMPTY", where,
			fmt.Sprintf("[pob:%s] range %d-%d holds no Lua -- blank lines and comments only, "+
				"which is what a citation looks like after the file it points into has moved",
				path, start, end))
	}
}

func orTemplateToRE(reason string) *regexp.Regexp {
	parts := strings.Split(reason, "%s")
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	return regexp.MustCompile("^" + strings.Join(parts, ".+?") + "$")
}

func omissionReasons(args []string) int {
	pob := pobPinDir()
	if len(args) > 0 && args[0] == "--selftest" {
		return orSelftest(pob)
	}
	root := pcRepoRoot()
	lines := pcIndexPoB(pob)

	type orFail struct{ code, where, msg string }
	var failures []orFail
	fail := func(code, where, msg string) { failures = append(failures, orFail{code, where, msg}) }

	checked := 0
	for _, tbl := range orTables {
		emitter, generated, opener := tbl[0], tbl[1], tbl[2]
		ep := pyPathJoin(root, emitter)
		gp := pyPathJoin(root, generated)
		reasons := orEmitterReasons(ep)
		if len(reasons) == 0 {
			fail("NO-REASONS", emitter, "no Unrepresentable(...) sites found -- has the emitter "+
				"been restructured? this gate is now blind")
			continue
		}
		for _, r := range reasons {
			checked++
			orCheckProvenance(root, pob, lines, fmt.Sprintf("%s:%d", emitter, r.line), r.text, fail)
		}
		pats := make([]*regexp.Regexp, len(reasons))
		for i, r := range reasons {
			pats[i] = orTemplateToRE(r.text)
		}
		for _, h := range orHeaderReasons(gp, opener) {
			matched := false
			for _, p := range pats {
				if p.MatchString(h.text) {
					matched = true
					break
				}
			}
			if !matched {
				fail("STALE-HEADER", fmt.Sprintf("%s:%d", generated, h.line),
					fmt.Sprintf("%d mods are omitted for a reason no emitter site can produce: "+
						"%s. Regenerate the table -- the header was written by an older emitter.",
						h.n, pyRepr(h.text)))
			}
		}
		fmt.Printf("%-40s %d reasons, %d header lines\n",
			emitter, len(reasons), len(orHeaderReasons(gp, opener)))
	}

	if len(failures) == 0 {
		fmt.Printf("\nOMISSION REASONS: %d checked, all resolve\n", checked)
		return 0
	}
	fmt.Printf("\nOMISSION REASONS: %d checked, %d FAILED\n", checked, len(failures))
	for _, f := range failures {
		fmt.Printf("  %-14s %s\n                 %s\n", f.code, f.where, f.msg)
	}
	if os.Getenv("REPORT") != "" {
		return 0
	}
	return 1
}

func orSelftest(pob string) int {
	root := pcRepoRoot()
	lines := pcIndexPoB(pob)
	cases := []struct{ reason, want string }{
		{"a reason with no marker at all", "NO-PROVENANCE"},
		{"stale go claim [go-absent:mod.TypeLimit]", "GO-STALE"},
		{"renamed symbol [go:mod.ThisTypeDoesNotExist]", "GO-MISSING"},
		{"bad path [pob:src/Classes/NoSuchFile.lua:1-2]", "NOFILE"},
		{"past EOF [pob:src/Classes/ModStore.lua:999999-1000000]", "RANGE"},
		{"no why [unverifiable:]", "EMPTY-WHY"},
		{"garbage range [pob:ModStore.lua:not-a-number]", "MALFORMED"},
		{"fine [pob:src/Classes/ModStore.lua:525-528]", ""},
		{"fine [go:mod.TypeLimit]", ""},
		{"fine [go-absent:mod.NoSuchTagAnywhere]", ""},
		{"fine [unverifiable: data-dependent]", ""},
	}
	bad := 0
	for _, tc := range cases {
		var got []string
		orCheckProvenance(root, pob, lines, "selftest", tc.reason,
			func(c, _, _ string) { got = append(got, c) })
		joined := strings.Join(got, ",")
		ok := joined == tc.want
		w := tc.want
		if w == "" {
			w = "(pass)"
		}
		g := joined
		if g == "" {
			g = "-"
		}
		verdict := "BROKEN"
		if ok {
			verdict = "ok"
		}
		fmt.Printf("  %-8s expect %-14s got %-14s %s\n", verdict, w, g, tc.reason)
		if !ok {
			bad++
		}
	}
	fmt.Printf("\nSELFTEST: %d of %d checks behave as declared\n", len(cases)-bad, len(cases))
	if bad > 0 {
		return 1
	}
	return 0
}
