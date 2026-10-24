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
	tfMaxEntries = 12
	tfMaxLineLen = 310
)

var (
	tfTaskRE  = regexp.MustCompile(`^  ([a-z0-9-]+):\s*$`)
	tfCmdsRE  = regexp.MustCompile(`^    cmds:\s*$`)
	tfEntryRE = regexp.MustCompile(`^      - `)
)

type tfTask struct {
	name     string
	entries  []string
	longest  int
	atLine   int
	longLine int
}

func tfParse(t *testing.T) []tfTask {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "Taskfile.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var out []tfTask
	cur := tfTask{}
	inCmds := false
	flush := func() {
		if cur.name != "" && len(cur.entries) > 0 {
			out = append(out, cur)
		}
	}
	for i, ln := range strings.Split(string(b), "\n") {
		if m := tfTaskRE.FindStringSubmatch(ln); m != nil {
			flush()
			cur = tfTask{name: m[1], atLine: i + 1}
			inCmds = false
			continue
		}
		if tfCmdsRE.MatchString(ln) {
			inCmds = true
			continue
		}
		if !inCmds {
			continue
		}
		if strings.TrimSpace(ln) != "" && !strings.HasPrefix(ln, "      ") {
			flush()
			cur = tfTask{}
			inCmds = false
			continue
		}
		if tfEntryRE.MatchString(ln) {
			e := strings.TrimSpace(ln)[2:]
			cur.entries = append(cur.entries, e)
			if len(e) > cur.longest {
				cur.longest, cur.longLine = len(e), i+1
			}
		}
	}
	flush()
	if len(out) < 50 {
		t.Fatalf("only %d tasks with a cmds block parsed out of Taskfile.yml — the scan stopped "+
			"matching, and a cap that holds over nothing is not a cap", len(out))
	}
	return out
}

func TestTaskfileCmdsBlocksStaySmall(t *testing.T) {
	t.Parallel()
	tasks := tfParse(t)

	var wide []string
	for _, tk := range tasks {
		if len(tk.entries) > tfMaxEntries {
			wide = append(wide, tk.name)
		}
	}
	sort.Strings(wide)
	if len(wide) > 0 {
		t.Errorf("%d task(s) exceed %d cmds entries: %s\n"+
			"  A cmds block that long is orchestration, and orchestration belongs in a subcommand\n"+
			"  where golangci-lint reads it and a test can run it. Nothing lints a Taskfile.",
			len(wide), tfMaxEntries, strings.Join(wide, ", "))
	}
}

func TestTaskfileCommandsStayShort(t *testing.T) {
	t.Parallel()
	var long []string
	for _, tk := range tfParse(t) {
		if tk.longest > tfMaxLineLen {
			long = append(long, tk.name+" (Taskfile.yml:"+itoa(tk.longLine)+", "+itoa(tk.longest)+" chars)")
		}
	}
	sort.Strings(long)
	if len(long) > 0 {
		t.Errorf("%d command line(s) exceed %d characters:\n  %s\n"+
			"  A command this long is a shell script on one line. Move it behind a subcommand —\n"+
			"  the arguments become flags a parser can refuse, instead of text nothing checks.",
			len(long), tfMaxLineLen, strings.Join(long, "\n  "))
	}
}

// TestTaskfileCommandsDoNotBranch: AND THE SHAPE THAT MATTERS MOST IS NOT LENGTH AT ALL: it is a cmds entry that BRANCHES.
func TestTaskfileCommandsDoNotBranch(t *testing.T) {
	t.Parallel()
	branch := regexp.MustCompile(`\|\||\bif\b.*;\s*then\b|\bfor\b.*;\s*do\b|\bwhile\b|\bcase\b.*\bin\b`)
	exempt := map[string]bool{
		"import-characters": true,
		"corpus-harvest":    true,
		"oracle-corpus":     true,
		"ladder-status":     true,
		"coverage-census":   true,
	}
	var branching []string
	for _, tk := range tfParse(t) {
		if exempt[tk.name] {
			continue
		}
		for _, e := range tk.entries {
			if branch.MatchString(e) {
				branching = append(branching, tk.name+": "+truncate(e, 90))
				break
			}
		}
	}
	sort.Strings(branching)
	if len(branching) > 0 {
		t.Errorf("%d task(s) have a cmds entry that BRANCHES:\n  %s\n"+
			"  Control flow in a Taskfile is shell that nothing lints and no test runs. Put it in a\n"+
			"  subcommand, or add the task to the exemption list with a reason.",
			len(branching), strings.Join(branching, "\n  "))
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
