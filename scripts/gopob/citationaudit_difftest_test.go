package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const (
	caPy      = "scripts/citation_audit.py"
	caGoFile  = "calculator/marked.go"
	caLuaPath = "pob/src/Modules/Thing.lua"
)

func caSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	files := map[string]string{}
	if py := difftest.ReadOriginal(t, repo, caPy); py != nil {
		files[caPy] = string(py)
	}
	for rel, src := range map[string]string{
		caGoFile:  filepath.Join(repo, "scripts/gopob/testdata/citationaudit_marked.go.txt"),
		caLuaPath: filepath.Join(repo, "scripts/gopob/testdata/citationaudit_thing.lua.txt"),
	} {
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		files[rel] = string(b)
	}
	files["go.mod"] = "module github.com/Vilsol/go-pob\n\ngo 1.26.2\n"
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		for rel, body := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func TestCitationAuditPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := caSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, caPy)

	spec := difftest.Spec{
		Tool:         "citation-audit",
		PythonScript: caPy,
		GoBin:        bin,
		GoArgs:       []string{"citation-audit"},
		Env:          []string{"POB_DIR=pob", "FILE=" + caGoFile},
		Stderr:       difftest.StderrExact,
		RefusalSites: 6,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-go-mod", Refusal: "no go.mod declaring module github.com/Vilsol/go-pob",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "go.mod")) }},

			{Name: "bad-gopob-root", Refusal: "GOPOB_ROOT does not contain a go.mod declaring module",
				Break: func(string) error { return nil }},

			{Name: "gofile-unreadable", Refusal: caGoFile + " is not readable:",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, caGoFile), 0o000)
				}},

			{Name: "gofile-invalid-utf8", Refusal: caGoFile + " is not valid UTF-8",
				Break: func(sb string) error { return put(sb, caGoFile, "package a\n// \xff\xfe\n") }},

			{Name: "bad-context", Refusal: "CONTEXT must be a non-negative integer, not two",
				Break: func(string) error { return nil }},

			{Name: "no-file-and-no-sweep", Refusal: "set FILE=<go file>, or SWEEP=1",
				Break: func(string) error { return nil }},

			{Name: "context-zero", Passes: true, Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "gofile-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			switch f.Name {
			case "bad-gopob-root":
				s.Env = append(append([]string{}, spec.Env...), "GOPOB_ROOT=pob")
			case "bad-context":
				s.Env = append(append([]string{}, spec.Env...), "CONTEXT=two")
			case "no-file-and-no-sweep":
				s.Env = []string{"POB_DIR=pob", "FILE="}
			case "context-zero":
				s.Env = append(append([]string{}, spec.Env...), "CONTEXT=0")
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestCitationAuditContextWidthsMatch: CONTEXT IS THE ONLY KNOB AND THE FIXTURES ABOVE USE ONE VALUE.
func TestCitationAuditContextWidthsMatch(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := caSandbox(t, repo)
	for _, w := range []string{"0", "1", "2", "5", "400"} {
		t.Run("ctx"+w, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "citation-audit",
				PythonScript: caPy,
				GoBin:        bin,
				GoArgs:       []string{"citation-audit"},
				Env:          []string{"POB_DIR=pob", "FILE=" + caGoFile, "CONTEXT=" + w},
				Stderr:       difftest.StderrExact,
			}, difftest.Fixture{Name: "ctx" + w}, mk)
		})
	}
}

// TestCitationAuditSweepMatchesTheRealTree: THE COMPILED BINARY, NEVER `go run`.
func TestCitationAuditSweepMatchesTheRealTree(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	if _, err := os.Stat(pobDir(t)); err != nil {
		t.Skip("pinned PoB not available")
	}
	bin := buildPort(t, repo)

	run := func(argv ...string) (string, int) {
		t.Helper()
		//nolint:gosec // argv is this test's own literal: python3 + the script, or the built binary
		cmd := exec.CommandContext(t.Context(), argv[0], argv[1:]...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "SWEEP=1")
		out, _ := cmd.Output()
		return string(out), cmd.ProcessState.ExitCode()
	}
	goOut, goRC := run(bin, "citation-audit")
	if !difftest.OriginalsDeleted(repo) {
		pyOut, pyRC := run("python3", filepath.Join(repo, caPy))
		if pyRC != goRC {
			t.Fatalf("sweep exit code: python=%d go=%d", pyRC, goRC)
		}
		if pyOut != goOut {
			t.Fatalf("sweep stdout differs\n--- python ---\n%s\n--- go ---\n%s", pyOut, goOut)
		}
	}
	if !strings.Contains(goOut, "range(s) across") {
		t.Fatalf("no summary line; the sweep may have examined nothing:\n%s", goOut)
	}
	var ranges, files, flagged int
	if _, err := fmt.Sscanf(strings.TrimSpace(lastLineWith(goOut, "range(s) across")),
		"citation shapes: %d range(s) across %d PORT file(s), %d flagged", &ranges, &files, &flagged); err != nil {
		t.Fatalf("cannot read the summary line: %v", err)
	}
	if ranges < 100 || files < 10 {
		t.Fatalf("sweep saw %d ranges across %d files; too few to be reading the real tree",
			ranges, files)
	}
	t.Logf("sweep agreed: %d ranges across %d PORT files, %d flagged, exit %d",
		ranges, files, flagged, goRC)
}

func lastLineWith(s, needle string) string {
	var found string
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, needle) {
			found = l
		}
	}
	return found
}
