package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type csClass int

const (
	csProse csClass = iota
	csMarker
	csPoB
	csLuaCode
	csGoCode
	csBlock
	csTestDoc
)

var csClassNames = map[csClass]string{
	csProse: "prose", csMarker: "marker", csPoB: "pob-comment",
	csLuaCode: "lua-code", csGoCode: "go-code", csBlock: "block", csTestDoc: "test-doc",
}

var (
	csMarkerRE           = regexp.MustCompile(`^//(go:|nolint|pob:|go-only:|lint:|export |\+build| ?retired-version-ok:)`)
	csCodeStartRE        = regexp.MustCompile(`^(local [A-Za-z_]|if .* then$|elseif .* then$|else$|end[,)]?$|for .* do$|t_insert\()`)
	csLuaTokRE           = regexp.MustCompile(`~=|\bthen\b|\blocal [A-Za-z_]|\belseif\b|[~=]= nil\b|= nil\b|:[A-Z][A-Za-z]*\(|[A-Za-z_\])"]\s*\.\.\s*[A-Za-z_"(]|\bipairs\(|\bpairs\(|\bm_[a-z]+\(|\bt_insert\(|^end$|^else$|\bfunction\s*\(`)
	csNormRE             = regexp.MustCompile(`[^a-z0-9]+`)
	csLuaCommRE          = regexp.MustCompile(`--\s*(.*)$`)
	csGenHeadRE          = regexp.MustCompile(`(?im)^// code generated .* do not edit\.$`)
	csMinPoBNorm         = 12
	csTestFuncRE         = regexp.MustCompile(`^(Test|Benchmark|Fuzz)([^a-z]|$)`)
	csSentenceEndRE      = regexp.MustCompile(`[.!?](\s|$)`)
	csClauseRE           = regexp.MustCompile(`: | — | -- |; |, | \(`)
	csMaxTestDoc         = 160
	csIdentRE            = regexp.MustCompile(`[A-Za-z0-9_]`)
	csBlankBeforeCloseRE = regexp.MustCompile(`\n[ \t]*\n([ \t]*[})])`)
)

func csGeneratedFiles(manifest string) (map[string]bool, error) {
	raw, err := os.ReadFile(manifest) //nolint:gosec // an operator-supplied path
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", manifest, err)
	}
	out := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[1] == "GENERATED" {
			out[filepath.Clean(f[0])] = true
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s classifies no file GENERATED", manifest)
	}
	return out, nil
}

func csNorm(s string) string {
	return strings.Trim(csNormRE.ReplaceAllString(strings.ToLower(s), " "), " ")
}

func csPoBComments(src string) (map[string]bool, error) {
	set := map[string]bool{}
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error { //nolint:gosec // an operator-supplied path
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".lua") {
			return nil
		}
		raw, err := os.ReadFile(p) //nolint:gosec // an operator-supplied path
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if m := csLuaCommRE.FindStringSubmatch(line); m != nil {
				if n := csNorm(m[1]); len(n) >= csMinPoBNorm {
					set[n] = true
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", src, err)
	}
	return set, nil
}

func csLineCode(body string) (bool, bool) {
	t := strings.TrimSpace(body)
	if t == "" || strings.ContainsAny(t, "`") || strings.Contains(t, "⚠️") {
		return false, false
	}
	if !strings.HasPrefix(body, "\t") && !csCodeStartRE.MatchString(t) {
		return false, false
	}
	return true, csLuaTokRE.MatchString(t)
}

func csClassify(text string, pob map[string]bool) csClass {
	if strings.HasPrefix(text, "/*") {
		lines := strings.Split(strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/"), "\n")
		code, total := 0, 0
		for i, l := range lines {
			t := strings.TrimSpace(l)
			if t == "" {
				continue
			}
			if i <= 1 && strings.HasPrefix(t, "TODO") {
				return csBlock
			}
			total++
			if isCode, _ := csLineCode(t); isCode {
				code++
			}
		}
		if total > 0 && code*2 >= total {
			return csBlock
		}
		return csProse
	}
	if csMarkerRE.MatchString(text) {
		return csMarker
	}
	body := strings.TrimPrefix(text, "//")
	if n := csNorm(body); len(n) >= csMinPoBNorm && pob[n] {
		return csPoB
	}
	if isCode, isLua := csLineCode(body); isCode {
		if isLua {
			return csLuaCode
		}
		return csGoCode
	}
	return csProse
}

type csStats struct {
	lines    map[csClass]int
	maxProse int
	long     []string
	samples  []string
	show     csClass
	showAll  bool
	tests    bool
}

func csTestDocLines(f *ast.File, pob map[string]bool) map[*ast.Comment]string {
	out := map[*ast.Comment]string{}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv != nil || fd.Doc == nil || !csTestFuncRE.MatchString(fd.Name.Name) {
			continue
		}
		var first *ast.Comment
		var words []string
		for _, c := range fd.Doc.List {
			if csClassify(c.Text, pob) != csProse {
				if first != nil {
					break
				}
				continue
			}
			body := strings.TrimPrefix(c.Text, "//")
			if strings.HasPrefix(c.Text, "/*") {
				body = strings.TrimSuffix(strings.TrimPrefix(c.Text, "/*"), "*/")
			}
			if strings.TrimSpace(body) == "" {
				if first != nil {
					break
				}
				continue
			}
			if first == nil {
				first = c
			}
			words = append(words, strings.Fields(body)...)
			if csSentenceEndRE.MatchString(strings.Join(words, " ")) {
				break
			}
		}
		if first == nil {
			continue
		}
		body := strings.TrimPrefix(strings.Join(words, " "), "⚠️ ")
		if loc := csSentenceEndRE.FindStringIndex(body); loc != nil {
			body = body[:loc[0]+1]
		}
		body = strings.TrimRight(body, ":;,— ")
		if !strings.ContainsAny(body[len(body)-1:], ".!?") {
			body += "."
		}
		name := fd.Name.Name
		if rest, ok := strings.CutPrefix(body, name); !ok || rest != "" && csIdentRE.MatchString(rest[:1]) {
			body = name + ": " + body
		}
		if len(body) > csMaxTestDoc {
			for _, loc := range csClauseRE.FindAllStringIndex(body, -1) {
				head := body[:loc[0]]
				if len(head) >= len(name)+30 && strings.Count(head, "`")%2 == 0 && strings.Count(head, `"`)%2 == 0 {
					body = strings.TrimRight(head, ".") + "."
					break
				}
			}
		}
		out[first] = "// " + body
	}
	return out
}

func csStrip(path string, src []byte, pob map[string]bool, drop map[csClass]bool, st *csStats, nSamples int) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	type cut struct {
		from, to int
		repl     string
	}
	var cuts []cut
	var testDoc map[*ast.Comment]string
	if st.tests {
		testDoc = csTestDocLines(f, pob)
	}
	for _, g := range f.Comments {
		run, runStart := 0, token.NoPos
		for _, c := range g.List {
			cls := csClassify(c.Text, pob)
			repl, isTestDoc := testDoc[c]
			if isTestDoc {
				cls = csTestDoc
			}
			if cls == csProse {
				if run == 0 {
					runStart = c.Pos()
				}
				run += strings.Count(c.Text, "\n") + 1
				if st.maxProse > 0 && run == st.maxProse+1 {
					st.long = append(st.long, fmt.Sprintf("%s:%d", path, fset.Position(runStart).Line))
				}
			} else {
				run = 0
			}
			st.lines[cls] += strings.Count(c.Text, "\n") + 1
			if len(st.samples) < nSamples && ((st.showAll && drop[cls]) || (!st.showAll && cls == st.show)) {
				text := c.Text
				if isTestDoc {
					text = repl
				}
				st.samples = append(st.samples, fmt.Sprintf("%s:%d: %s", path, fset.Position(c.Pos()).Line, text))
			}
			start := fset.Position(c.Pos()).Offset
			end := fset.Position(c.End()).Offset
			if isTestDoc && repl != c.Text {
				cuts = append(cuts, cut{start, end, repl})
			}
			if !drop[cls] {
				continue
			}
			lineStart := bytes.LastIndexByte(src[:start], '\n') + 1
			if len(bytes.TrimSpace(src[lineStart:start])) == 0 {
				if end < len(src) && src[end] == '\n' {
					end++
				}
				cuts = append(cuts, cut{lineStart, end, ""})
			} else {
				from := start
				for from > lineStart && (src[from-1] == ' ' || src[from-1] == '\t') {
					from--
				}
				cuts = append(cuts, cut{from, end, ""})
			}
		}
	}
	if len(cuts) == 0 {
		return src, nil
	}
	sort.Slice(cuts, func(i, j int) bool { return cuts[i].from < cuts[j].from })
	var out bytes.Buffer
	prev := 0
	for _, c := range cuts {
		if c.from < prev {
			continue
		}
		out.Write(src[prev:c.from])
		out.WriteString(c.repl)
		prev = c.to
	}
	out.Write(src[prev:])
	formatted, err := format.Source(csBlankBeforeCloseRE.ReplaceAll(out.Bytes(), []byte("\n$1")))
	if err != nil {
		return nil, fmt.Errorf("%s: stripped source no longer formats: %w", path, err)
	}
	return formatted, nil
}

func commentStrip(args []string) int {
	apply := false
	nSamples := 0
	pobSrc := filepath.Join(os.Getenv("HOME"), "PathOfBuilding/src")
	drop := map[csClass]bool{csProse: true, csGoCode: true}
	var dirs []string
	show := ""
	maxProse := 0
	manifest := "scripts/port_manifest.txt"
	tests := false
	for _, a := range args {
		switch {
		case a == "--apply":
			apply = true
		case a == "--tests":
			tests = true
		case strings.HasPrefix(a, "--samples="):
			n, err := strconv.Atoi(strings.TrimPrefix(a, "--samples="))
			if err != nil {
				fmt.Fprintf(os.Stderr, "REFUSING: %q: %v\n", a, err)
				return 2
			}
			nSamples = n
		case strings.HasPrefix(a, "--pob="):
			pobSrc = strings.TrimPrefix(a, "--pob=")
		case strings.HasPrefix(a, "--show="):
			show = strings.TrimPrefix(a, "--show=")
		case strings.HasPrefix(a, "--manifest="):
			manifest = strings.TrimPrefix(a, "--manifest=")
		case strings.HasPrefix(a, "--max-prose="):
			n, err := strconv.Atoi(strings.TrimPrefix(a, "--max-prose="))
			if err != nil || n < 1 {
				fmt.Fprintf(os.Stderr, "REFUSING: %q needs a positive line count\n", a)
				return 2
			}
			maxProse = n
		case a == "--drop-lua-code":
			drop[csLuaCode] = true
		case a == "--keep-go-code":
			drop[csGoCode] = false
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(os.Stderr, "REFUSING: unknown flag %q\n", a)
			return 2
		default:
			dirs = append(dirs, a)
		}
	}
	if len(dirs) == 0 {
		fmt.Fprintln(os.Stderr, "REFUSING: name at least one package directory")
		return 2
	}
	generated, err := csGeneratedFiles(manifest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %v — without it a generated file would be stripped\n", err)
		return 2
	}
	pob, err := csPoBComments(pobSrc)
	if err != nil || len(pob) == 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: no PoB comments read from %s (%v) — every PoB comment would be deleted as prose\n", pobSrc, err)
		return 2
	}
	st := &csStats{lines: map[csClass]int{}, showAll: true, maxProse: maxProse, tests: tests}
	if show != "" {
		st.showAll = false
		found := false
		for c, n := range csClassNames {
			if n == show {
				st.show, found = c, true
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "REFUSING: --show=%q is not a class\n", show)
			return 2
		}
	}
	files, changed, skippedGen := 0, 0, 0
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
			return 2
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") != tests {
				continue
			}
			path := filepath.Join(dir, name)
			src, err := os.ReadFile(path) //nolint:gosec // an operator-supplied path
			if err != nil {
				fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
				return 2
			}
			if csGenHeadRE.Match(src) || generated[filepath.Clean(path)] {
				skippedGen++
				continue
			}
			files++
			out, err := csStrip(path, src, pob, drop, st, nSamples)
			if err != nil {
				fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
				return 1
			}
			if !bytes.Equal(out, src) {
				changed++
				if apply {
					if err := os.WriteFile(path, out, 0o644); err != nil { //nolint:gosec // source files keep their mode
						fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
						return 1
					}
				}
			}
		}
	}
	mode := "DRY RUN"
	if apply {
		mode = "APPLIED"
	}
	fmt.Printf("comment-strip %s: %d files (%d generated skipped), %d would change; %d PoB comment lines indexed\n",
		mode, files, skippedGen, changed, len(pob))
	for _, c := range []csClass{csProse, csMarker, csPoB, csLuaCode, csGoCode, csBlock, csTestDoc} {
		verb := "keep"
		if drop[c] {
			verb = "DROP"
		}
		fmt.Printf("  %-12s %7d comment lines  %s\n", csClassNames[c], st.lines[c], verb)
	}
	for _, s := range st.samples {
		fmt.Println("  -", s)
	}
	if maxProse > 0 {
		if len(st.long) > 0 {
			fmt.Printf("FAIL: %d comment group(s) carry more than %d lines of go-pob prose:\n", len(st.long), maxProse)
			for _, l := range st.long {
				fmt.Println("  ", l)
			}
			return 1
		}
		fmt.Printf("PASS: no comment group carries more than %d lines of go-pob prose\n", maxProse)
	}
	return 0
}
