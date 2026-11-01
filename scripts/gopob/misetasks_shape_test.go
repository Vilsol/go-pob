package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const (
	mtMaxEntries     = 12
	mtMaxScriptLines = 20
	mtMaxLineLen     = 310
)

var (
	mtTaskRE      = regexp.MustCompile(`^\[tasks\.([a-z0-9-]+)\]\s*$`)
	mtTableRE     = regexp.MustCompile(`^\[`)
	mtRunOneRE    = regexp.MustCompile(`^run = '(.*)'\s*$`)
	mtRunTripleRE = regexp.MustCompile(`^run = '''(.*)'''\s*$`)
	mtEntryRE     = regexp.MustCompile(`^\s+(?:'''(.*)'''|'(.*)'|\{ task = "[^"]*" \}),?\s*$`)
	mtRunArrayRE  = regexp.MustCompile(`^run = \[(.*)$`)
)

// mtTask is one task's commands: a mise.toml table's run entries, or a mise-tasks/ script's body.
type mtTask struct {
	name   string
	where  string
	script bool
	cmds   []string
	lines  []mtLine
}

type mtLine struct {
	text string
	at   int
}

func (tk *mtTask) add(text string, at int) {
	tk.lines = append(tk.lines, mtLine{text, at})
}

func mtParseToml(t *testing.T, body string) []mtTask {
	t.Helper()
	var out []mtTask
	var cur *mtTask
	inArray, inTriple := false, false
	var triple []string
	flush := func() {
		if cur != nil && len(cur.cmds) > 0 {
			out = append(out, *cur)
		}
		cur = nil
	}
	for i, ln := range strings.Split(body, "\n") {
		at := i + 1
		if inTriple {
			if strings.HasPrefix(strings.TrimSpace(ln), "'''") {
				inTriple = false
				cur.cmds = append(cur.cmds, mtLogical(triple)...)
				continue
			}
			triple = append(triple, ln)
			cur.add(ln, at)
			continue
		}
		if m := mtTaskRE.FindStringSubmatch(ln); m != nil {
			flush()
			cur = &mtTask{name: m[1], where: "mise.toml"}
			inArray = false
			continue
		}
		if mtTableRE.MatchString(ln) {
			flush()
			inArray = false
			continue
		}
		if cur == nil {
			continue
		}
		if inArray {
			if strings.TrimSpace(ln) == "]" {
				inArray = false
				continue
			}
			if strings.TrimSpace(ln) == "'''" || strings.HasSuffix(strings.TrimSpace(ln), "'''\n") {
				inTriple, triple = true, nil
				continue
			}
			if m := mtEntryRE.FindStringSubmatch(ln); m != nil {
				e := m[1] + m[2]
				if e == "" {
					e = strings.TrimSpace(ln)
				}
				cur.cmds = append(cur.cmds, e)
				cur.add(e, at)
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(ln), "'''") {
				inTriple, triple = true, nil
				continue
			}
			t.Fatalf("mise.toml:%d: a run entry this scan cannot read: %q — keep entries one per line, "+
				"as '...', '''...''' or { task = \"...\" }", at, ln)
		}
		if ln == "run = '''" {
			inTriple, triple = true, nil
			continue
		}
		if m := mtRunTripleRE.FindStringSubmatch(ln); m != nil {
			cur.cmds = append(cur.cmds, m[1])
			cur.add(m[1], at)
			continue
		}
		if m := mtRunOneRE.FindStringSubmatch(ln); m != nil {
			cur.cmds = append(cur.cmds, m[1])
			cur.add(m[1], at)
			continue
		}
		if m := mtRunArrayRE.FindStringSubmatch(ln); m != nil {
			rest := strings.TrimSpace(m[1])
			if rest == "" {
				inArray = true
				continue
			}
			for _, e := range strings.Split(strings.TrimSuffix(rest, "]"), ", ") {
				cur.cmds = append(cur.cmds, e)
				cur.add(e, at)
			}
			if !strings.HasSuffix(rest, "]") {
				t.Fatalf("mise.toml:%d: a run array this scan cannot read: %q — open it with `run = [` "+
					"and put one entry per line", at, ln)
			}
			continue
		}
		if strings.HasPrefix(ln, "run") {
			t.Fatalf("mise.toml:%d: a run key this scan cannot read: %q", at, ln)
		}
	}
	flush()
	return out
}

// mtLogical joins a script's backslash-continued lines and drops blanks, so a command is counted once.
func mtLogical(lines []string) []string {
	var out []string
	cur := ""
	for _, ln := range lines {
		s := strings.TrimSpace(ln)
		if strings.HasSuffix(s, "\\") {
			cur += strings.TrimSuffix(s, "\\") + " "
			continue
		}
		cur += s
		if strings.TrimSpace(cur) != "" {
			out = append(out, cur)
		}
		cur = ""
	}
	return out
}

func mtParseScripts(t *testing.T, dir string) []mtTask {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]mtTask, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() {
			t.Fatalf("mise-tasks/%s is a directory; its scripts would be tasks named %s:<file>, which this "+
				"scan does not read", e.Name(), e.Name())
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // a file in the repo's own task dir
		if err != nil {
			t.Fatal(err)
		}
		tk := mtTask{name: e.Name(), where: "mise-tasks/" + e.Name(), script: true}
		var body []string
		for i, ln := range strings.Split(string(b), "\n") {
			if s := strings.TrimSpace(ln); s == "" || strings.HasPrefix(s, "#") {
				continue
			}
			body = append(body, ln)
			tk.add(ln, i+1)
		}
		tk.cmds = mtLogical(body)
		out = append(out, tk)
	}
	return out
}

func mtParse(t *testing.T) []mtTask {
	t.Helper()
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "mise.toml"))
	if err != nil {
		t.Fatal(err)
	}
	toml := mtParseToml(t, string(b))
	if len(toml) < 100 {
		t.Fatalf("only %d tasks with a run key parsed out of mise.toml — the scan stopped "+
			"matching, and a cap that holds over nothing is not a cap", len(toml))
	}
	scripts := mtParseScripts(t, filepath.Join(root, "mise-tasks"))
	if len(scripts) == 0 {
		t.Fatal("no scripts parsed out of mise-tasks/ — the scan stopped matching")
	}
	return append(toml, scripts...)
}

func TestMiseTasksRunBlocksStaySmall(t *testing.T) {
	t.Parallel()
	var wide []string
	for _, tk := range mtParse(t) {
		limit := mtMaxEntries
		if tk.script {
			limit = mtMaxScriptLines
		}
		if len(tk.cmds) > limit {
			wide = append(wide, tk.name+" ("+tk.where+", "+itoa(len(tk.cmds))+" > "+itoa(limit)+")")
		}
	}
	sort.Strings(wide)
	if len(wide) > 0 {
		t.Errorf("%d task(s) exceed their command cap: %s\n"+
			"  A run block that long is orchestration, and orchestration belongs in a subcommand\n"+
			"  where golangci-lint reads it and a test can run it. Nothing lints a task.",
			len(wide), strings.Join(wide, ", "))
	}
}

func TestMiseTasksCommandsStayShort(t *testing.T) {
	t.Parallel()
	var long []string
	for _, tk := range mtParse(t) {
		for _, ln := range tk.lines {
			if len(ln.text) > mtMaxLineLen {
				at := ln.at
				if !tk.script {
					at = mtTomlLine(t, ln.text)
				}
				long = append(long, tk.name+" ("+mtFile(tk)+":"+itoa(at)+", "+itoa(len(ln.text))+" chars)")
			}
		}
	}
	sort.Strings(long)
	if len(long) > 0 {
		t.Errorf("%d command line(s) exceed %d characters:\n  %s\n"+
			"  A command this long is a shell script on one line. Break it with \\ continuations, or move\n"+
			"  it behind a subcommand — the arguments become flags a parser can refuse.",
			len(long), mtMaxLineLen, strings.Join(long, "\n  "))
	}
}

func mtFile(tk mtTask) string {
	if tk.script {
		return tk.where
	}
	return "mise.toml"
}

func mtTomlLine(t *testing.T, text string) int {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "mise.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for i, ln := range strings.Split(string(b), "\n") {
		if strings.Contains(ln, text) {
			return i + 1
		}
	}
	return 0
}

// TestMiseTasksDoNotBranch: AND THE SHAPE THAT MATTERS MOST IS NOT LENGTH AT ALL: it is a command that BRANCHES.
func TestMiseTasksDoNotBranch(t *testing.T) {
	t.Parallel()
	exempt := map[string]bool{
		// a REFUSING guard when the version cannot be read
		"oracle-corpus":           true,
		"oracle-bin":              true,
		"corpus-harvest-selftest": true,
		// a corpus pass whose failing tests still leave a profile to read
		"coverage-census": true,
		// a warning when the graded total is missing
		"residual-clusters": true,
	}
	var branching []string
	seen := map[string]bool{}
	for _, tk := range mtParse(t) {
		seen[tk.name] = true
		if exempt[tk.name] {
			continue
		}
		for _, e := range tk.cmds {
			if mtBranchRE.MatchString(e) {
				branching = append(branching, tk.name+": "+truncate(e, 90))
				break
			}
		}
	}
	for name := range exempt {
		if !seen[name] {
			t.Errorf("the exemption list names %q, which is no task — drop it from the list", name)
		}
	}
	sort.Strings(branching)
	if len(branching) > 0 {
		t.Errorf("%d task(s) have a command that BRANCHES:\n  %s\n"+
			"  Control flow in a task is shell that nothing lints and no test runs. Put it in a\n"+
			"  subcommand, use a {%% if %%} on a usage flag, or add the task to the exemption list with a reason.",
			len(branching), strings.Join(branching, "\n  "))
	}
}

var mtBranchRE = regexp.MustCompile(`\|\||\bif\b.*;\s*then\b|^\s*if\b|\bfor\b.*;\s*do\b|\bwhile\b|\bcase\b.*\bin\b`)

func TestMiseTasksShapeScanControls(t *testing.T) {
	t.Parallel()
	toml := mtParseToml(t, strings.Join([]string{
		"[tasks.one]",
		"run = 'echo one'",
		"",
		"[tasks.many]",
		"run = [",
		"  'a',",
		"  { task = \"one\" },",
		"  '''b 'q' c''',",
		"  '''",
		"x \\",
		"  y",
		"z",
		"''',",
		"]",
		"",
		"[tasks.script]",
		"run = '''",
		"if [ -n \"$x\" ]; then",
		"  echo y",
		"fi",
		"'''",
	}, "\n"))
	got := map[string]int{}
	for _, tk := range toml {
		got[tk.name] = len(tk.cmds)
	}
	if got["one"] != 1 || got["many"] != 5 || got["script"] != 3 {
		t.Fatalf("the scan miscounted its own fixture: %v", got)
	}
	if !mtBranchRE.MatchString("if [ -n \"$x\" ]; then") || !mtBranchRE.MatchString("a || b") {
		t.Error("the branch pattern no longer matches the shapes it exists to refuse")
	}
	if mtBranchRE.MatchString("{% if usage.check %}--check{% endif %}") {
		t.Error("the branch pattern matches a template conditional on a usage flag, which is not shell")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
