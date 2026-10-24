package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func cgRun(t *testing.T, bin, sb string, env ...string) (string, string, int) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), bin, "corpus-grep")
	cmd.Dir = sb
	cmd.Env = append(os.Environ(), "GOPOB_ROOT="+sb, "PATTERN=", "PATTERNS_FILE=", "LUA=", "NAMES=1", "CORPUS_DIR=", "POE_VERSION=")
	cmd.Env = append(cmd.Env, env...)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	rc := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		rc = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return stdout.String(), stderr.String(), rc
}

func cgMultiSandbox(t *testing.T) (string, string) {
	t.Helper()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	sb, err := cgSandbox(t, repo)()
	if err != nil {
		t.Fatal(err)
	}
	extra := map[string]string{
		"c01": "<Build>\n  <Item>Kitava’s Thirst</Item><Item>KITAVA'ſ Thirst</Item>\n</Build>\n",
		"c02": "<Build>\n  <Item>Kitava&apos;s Thirst</Item>\n</Build>\n",
		"c03": "<Build>\n  <Item>kitava's\n      Thirst</Item>\n</Build>\n",
	}
	for name, body := range extra {
		if err := put(sb, filepath.Join("testdata", "corpus-"+cgSandboxVersion, name+".xml"), body); err != nil {
			t.Fatal(err)
		}
	}
	return bin, sb
}

func TestCorpusGrepPatternsFileIsTheSingleRunsConcatenated(t *testing.T) {
	t.Parallel()
	bin, sb := cgMultiSandbox(t)
	patterns := []string{
		"commontoken",
		"Lioneye's Fall",
		"With at least 40 Dexterity in Radius",
		"kitava's thirst",
		`kitava'(s) thirst`,
		"common%d-token",
		"no such line anywhere",
	}
	var want strings.Builder
	for _, p := range patterns {
		out, errOut, rc := cgRun(t, bin, sb, "PATTERN="+p)
		if rc != 0 {
			t.Fatalf("PATTERN=%q: rc %d: %s", p, rc, errOut)
		}
		want.WriteString(out)
	}
	file := filepath.Join(t.TempDir(), "patterns.txt")
	if err := os.WriteFile(file, []byte(strings.Join(patterns, "\n")+"\n\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, errOut, rc := cgRun(t, bin, sb, "PATTERNS_FILE="+file)
	if rc != 0 {
		t.Fatalf("rc %d: %s", rc, errOut)
	}
	if got != want.String() {
		t.Fatalf("PATTERNS_FILE output differs from the single runs\n--- got\n%s\n--- want\n%s", got, want.String())
	}
}

func TestCorpusGrepLiteralCountsWhatTheRegexCounts(t *testing.T) {
	t.Parallel()
	bin, sb := cgMultiSandbox(t)
	matched := func(p string) string {
		out, errOut, rc := cgRun(t, bin, sb, "PATTERN="+p)
		if rc != 0 {
			t.Fatalf("PATTERN=%q: rc %d: %s", p, rc, errOut)
		}
		_, rest, _ := strings.Cut(out, "MATCHED")
		return rest
	}
	literal, regex := matched("kitava's thirst"), matched(`kitava'(s) thirst`)
	if !strings.HasPrefix(literal, " 3 of 27 builds, 3 occurrence(s)") {
		t.Fatalf("literal path: %q", literal)
	}
	if literal != regex {
		t.Fatalf("literal path %q != regex path %q", literal, regex)
	}
}

func TestCorpusGrepPatternsFileRefusals(t *testing.T) {
	t.Parallel()
	bin, sb := cgMultiSandbox(t)
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for name, tc := range map[string]struct {
		env  []string
		want string
	}{
		"both":      {[]string{"PATTERN=x", "PATTERNS_FILE=" + write("a", "x\n")}, "set PATTERN or PATTERNS_FILE, not both"},
		"missing":   {[]string{"PATTERNS_FILE=" + filepath.Join(dir, "nope")}, "cannot read PATTERNS_FILE="},
		"blank":     {[]string{"PATTERNS_FILE=" + write("b", "\n  \n")}, "has no patterns"},
		"bad-regex": {[]string{"PATTERNS_FILE=" + write("c", "commontoken\n[unclosed\n")}, "corpus-grep: bad pattern '[unclosed'"},
	} {
		t.Run(name, func(t *testing.T) {
			out, errOut, rc := cgRun(t, bin, sb, tc.env...)
			if rc == 0 || out != "" || !strings.Contains(errOut, tc.want) {
				t.Fatalf("rc %d, stdout %q, stderr %q; want a refusal containing %q", rc, out, errOut, tc.want)
			}
		})
	}
}
