package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	ttRefusalExit = 3
	ttNamesFile   = "scripts/task_test_refs.names"
)

var ttSelfPaths = []string{
	"scripts/task_test_refs.py",
	"scripts/gopob/tasktestrefs.go",
	"scripts/gopob/tasktestrefs_run.go",
}

const (
	ttSpaceClass    = pySpaceClass
	ttNonSpaceClass = pyNonSpaceClass
)

var (
	ttRunRE = regexp.MustCompile(`-run` + ttSpaceClass + `+(?:'([^']+)'|"([^"]+)"|([^` +
		pySpaceChars + `'"]+))`)
	ttFuncRE      = regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]*)\(`)
	ttTemplateRE  = regexp.MustCompile(`\{\{|\$\{|\$\(|^\$[A-Za-z_]`)
	ttCommentRE   = regexp.MustCompile(`(?:^|` + ttSpaceClass + `)#`)
	ttGoCommentRE = regexp.MustCompile(`(?:^|` + ttSpaceClass + `)//`)
	ttRunGoRE     = regexp.MustCompile(`"-run",` + ttSpaceClass + `*"([^"]+)"`)
	ttRunAssignRE = regexp.MustCompile(`\bRUN=(Test[A-Za-z0-9_]*)`)
	ttQuoteRE     = regexp.MustCompile(`^` + ttSpaceClass + `*(?:>` + ttSpaceClass + `?)+`)
	ttFenceRE     = regexp.MustCompile(`^` + ttSpaceClass + `*(?:` + "```" + `|~~~)`)
)

var ttScriptish = map[string]bool{".sh": true, ".py": true, ".yml": true, ".yaml": true, ".go": true}

func ttRefuse(msg string) {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	os.Exit(ttRefusalExit)
}

func ttReadText(path, what string) string {
	body, err := os.ReadFile(path) //nolint:gosec // a repo-relative path this tool discovered
	if err != nil {
		ttRefuse(fmt.Sprintf("%s is not readable: %s", path, errnoText(err)))
	}
	if !utf8.Valid(body) {
		ttRefuse(fmt.Sprintf("%s is not valid UTF-8 (%s)", path, what))
	}
	s := string(body)
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

func ttWriteText(path, body, what string) {
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil { //nolint:gosec // matching the original's mode
		ttRefuse(fmt.Sprintf("%s is not writable: %s (%s)", path, errnoText(err), what))
	}
}

var ttRoots = []struct {
	sub   string
	globs []string
}{
	{"", []string{"*.md"}},
	{"scripts", []string{"*.sh", "*.py"}},
	{"scripts/gopob", []string{"*.go"}},
	{".github", []string{"*.sh", "*.yml", "*.yaml"}},
}

var ttExtra = []string{"Taskfile.yml"}

func ttDrivers(root string) []string {
	var out []string
	for _, r := range ttRoots {
		base := root
		if r.sub != "" {
			base = filepath.Join(root, r.sub)
		}
		if st, err := os.Stat(base); err != nil || !st.IsDir() {
			continue
		}
		for _, g := range r.globs {
			matches, err := filepath.Glob(filepath.Join(base, g))
			if err != nil {
				continue
			}
			sort.Strings(matches)
			for _, p := range matches {
				if st, err := os.Stat(p); err != nil || !st.Mode().IsRegular() {
					continue
				}
				if strings.HasSuffix(filepath.Base(p), "_test.go") {
					continue
				}
				rel, err := filepath.Rel(root, p)
				if err != nil {
					continue
				}
				out = append(out, filepath.ToSlash(rel))
			}
		}
	}
	for _, e := range ttExtra {
		if st, err := os.Stat(filepath.Join(root, e)); err == nil && st.Mode().IsRegular() &&
			!ttContains(out, e) {
			out = append(out, e)
		}
	}
	seen := map[string]bool{}
	var uniq []string
	for _, o := range out {
		if ttContains(ttSelfPaths, o) || seen[o] {
			continue
		}
		seen[o] = true
		uniq = append(uniq, o)
	}
	sort.Strings(uniq)
	return uniq
}

func ttContains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func ttKnownTests(root string) map[string]bool {
	cmd := exec.CommandContext(context.Background(), "git", "ls-files", "-z", "--", "*_test.go")
	cmd.Dir = root
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		ttRefuse("git ls-files failed: " + strings.TrimSpace(stderr.String()))
	}
	names := map[string]bool{}
	for _, rel := range strings.Split(stdout.String(), "\x00") {
		if rel == "" || ttSkippedTree(rel) {
			continue
		}
		path := filepath.Join(root, rel)
		if st, err := os.Stat(path); err != nil || !st.Mode().IsRegular() {
			continue
		}
		for _, m := range ttFuncRE.FindAllStringSubmatch(ttReadText(path, "a Go test file"), -1) {
			names[m[1]] = true
		}
	}
	return names
}

func ttSkippedTree(rel string) bool {
	for _, p := range strings.Split(filepath.ToSlash(rel), "/") {
		if p == "node_modules" || p == "frontend" {
			return true
		}
	}
	return false
}

func ttGateRowsFromTSV(text string) []string {
	var out []string
	for _, line := range ttSplitLines(text) {
		cols := strings.Split(line, "\t")
		if len(cols) < 3 {
			continue
		}
		for _, c := range cols[1:3] {
			c = strings.TrimSpace(c)
			if strings.HasPrefix(c, "Test") {
				out = append(out, c)
			}
		}
	}
	return out
}

func ttGateVersionTests(root, path string) []string {
	if path != "" {
		if st, err := os.Stat(path); err != nil || !st.Mode().IsRegular() {
			return nil
		}
		return ttGateRowsFromTSV(ttReadText(path, "a planted gate-version table"))
	}
	argv := []string{"go", "run", "./scripts/gopob", "gate-versions", "--tsv"}
	if bin := os.Getenv("GOPOB_BIN"); bin != "" {
		argv = []string{bin, "gate-versions", "--tsv"}
	}
	cmd := exec.CommandContext(context.Background(), argv[0], argv[1:]...) //nolint:gosec // the operator's own binary
	cmd.Dir = root
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	rc := 0
	if err != nil {
		rc = cmd.ProcessState.ExitCode()
	}
	if rc != 0 || strings.TrimSpace(stdout.String()) == "" {
		//go-only: BY DESIGN: added by the fold, the original shells out to a different
		ttRefuse(fmt.Sprintf("`gopob gate-versions --tsv` produced nothing (rc=%d): %s",
			rc, strings.TrimSpace(stderr.String())))
	}
	return ttGateRowsFromTSV(stdout.String())
}

func ttInlineSpans(line string) [][2]int {
	r := []rune(line)
	var spans [][2]int
	i := 0
	for i < len(r) {
		if r[i] != '`' {
			i++
			continue
		}
		maxN := 0
		for i+maxN < len(r) && r[i+maxN] == '`' {
			maxN++
		}
		matched := false
		for n := maxN; n >= 1 && !matched; n-- {
			for j := i + n; j+n <= len(r); j++ {
				if ttRunOfBackticks(r, j, n) {
					spans = append(spans, [2]int{i, j + n})
					i = j + n
					matched = true
					break
				}
			}
		}
		if !matched {
			i += maxN
		}
	}
	return spans
}

func ttRunOfBackticks(r []rune, at, n int) bool {
	for k := 0; k < n; k++ {
		if r[at+k] != '`' {
			return false
		}
	}
	return true
}

func ttSplitLinesKeepEnds(text string) []string {
	if text == "" {
		return nil
	}
	parts := strings.SplitAfter(text, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func ttSplitLines(text string) []string {
	out := ttSplitLinesKeepEnds(text)
	for i, s := range out {
		out[i] = strings.TrimSuffix(s, "\n")
	}
	return out
}

func ttMentionSpans(rel, text string) [][2]int {
	var spans [][2]int
	suffix := strings.ToLower(filepath.Ext(rel))
	pos := 0
	if ttScriptish[suffix] {
		cre := ttCommentRE
		if suffix == ".go" {
			cre = ttGoCommentRE
		}
		for _, line := range ttSplitLinesKeepEnds(text) {
			if loc := cre.FindStringIndex(line); loc != nil {
				spans = append(spans, [2]int{pos + ttRuneLen(line[:loc[0]]), pos + ttRuneLen(line)})
			}
			pos += ttRuneLen(line)
		}
		return spans
	}
	if suffix != ".md" {
		return spans
	}
	fenced := false
	for _, line := range ttSplitLinesKeepEnds(text) {
		bare := ttQuoteRE.ReplaceAllString(line, "")
		if ttFenceRE.MatchString(bare) {
			fenced = !fenced
			pos += ttRuneLen(line)
			continue
		}
		if !fenced {
			for _, s := range ttInlineSpans(line) {
				spans = append(spans, [2]int{pos + s[0], pos + s[1]})
			}
		}
		pos += ttRuneLen(line)
	}
	return spans
}

func ttRuneLen(s string) int { return len([]rune(s)) }

type ttRef struct {
	name string
	live bool
}

func ttReferenced(rel, text string) []ttRef {
	spans := ttMentionSpans(rel, text)
	var out []ttRef
	runeAt := ttByteToRune(text)
	pattern, groups := ttRunRE, 3
	if strings.EqualFold(filepath.Ext(rel), ".go") {
		pattern, groups = ttRunGoRE, 1
	}
	locs := pattern.FindAllStringSubmatchIndex(text, -1)
	locs = append(locs, ttRunAssignRE.FindAllStringSubmatchIndex(text, -1)...)
	for _, loc := range locs {
		value := ""
		for g := 1; g <= groups && value == ""; g++ {
			value = ttGroup(text, loc, g)
		}
		if ttTemplateRE.MatchString(value) {
			continue
		}
		start := runeAt[loc[0]]
		live := true
		for _, s := range spans {
			if s[0] <= start && start < s[1] {
				live = false
				break
			}
		}
		for _, alt := range strings.Split(value, "|") {
			alt = strings.Trim(strings.TrimSpace(alt), "^$")
			alt = strings.TrimSpace(alt)
			alt = strings.SplitN(alt, "/", 2)[0]
			if strings.HasPrefix(alt, "Test") {
				out = append(out, ttRef{alt, live})
			}
		}
	}
	return out
}

func ttGroup(text string, loc []int, n int) string {
	if 2*n+1 >= len(loc) || loc[2*n] < 0 {
		return ""
	}
	return text[loc[2*n]:loc[2*n+1]]
}

func ttByteToRune(text string) []int {
	out := make([]int, len(text)+1)
	r, prev := 0, 0
	for i := range text {
		for j := prev; j < i; j++ {
			out[j] = r - 1
		}
		out[i] = r
		r++
		prev = i + 1
	}
	for j := prev; j <= len(text); j++ {
		out[j] = r
	}
	return out
}
