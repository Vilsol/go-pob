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

const ttPy = "scripts/task_test_refs.py"

func ttFixtureMarkdown() string {
	return strings.Join([]string{
		"# A driver page",
		"",
		"A fenced block is a USE:",
		"",
		"```shell",
		"go test ./calculator/ -run TestFencedUse",
		"```",
		"",
		"A bare command line outside any fence is also a USE:",
		"",
		"    go test ./calculator/ -run TestIndentedUse",
		"",
		"go test ./calculator/ -run TestLooseUse",
		"",
		"An inline `go test -run TestInlineMention` is a MENTION, and so is",
		"a doubled ``go test -run TestDoubledMention`` span.",
		"",
		"An unclosed ``run -run TestUnclosedIsAUse and then some prose.",
		"",
		"A triple ```run -run TestBacktrackedMention` closes on a single.",
		"",
		"> ```",
		"> go test ./calculator/ -run TestQuotedFenceUse",
		"> ```",
		"",
		"Quoting forms: `-run 'TestSingleQuoted'` is a mention but",
		"",
		"```",
		"go test -run 'TestSingleQuoted'",
		"go test -run \"TestDoubleQuoted\"",
		"go test -run '^TestAnchored$'",
		"go test -run TestAlternationA|TestAlternationB",
		"go test -run TestSubtestPath/some_subtest",
		"go test -run {{.TEMPLATED}}",
		"go test -run ${SHELL_VAR}",
		"go test -run $(command)",
		"go test -run $BARE_VAR",
		"```",
		"",
		"A dead one lives in the mentions section only: `-run TestLongGoneFromTheTree`.",
		"",
	}, "\n")
}

func ttFixtureShell() string {
	return strings.Join([]string{
		"#!/usr/bin/env bash",
		"# A # comment naming -run TestCommentMention is a MENTION and never fails.",
		"go test ./calculator/ -run TestShellUse",
		"go test ./calculator/ -run TestTrailingComment   # -run TestAfterHashMention",
		"",
	}, "\n")
}

func ttFixtureTaskfile() string {
	return strings.Join([]string{
		"version: '3'",
		"tasks:",
		"  gate:",
		"    cmds:",
		"      - go test ./calculator/ -run TestTaskfileUse",
		"",
	}, "\n")
}

func ttFixtureWorkflow() string {
	return strings.Join([]string{
		"name: ci",
		"jobs:",
		"  test:",
		"    steps:",
		"      - run: go test ./calculator/ -run TestWorkflowUse",
		"",
	}, "\n")
}

func ttFixtureTests() string {
	names := []string{
		"TestFencedUse", "TestIndentedUse", "TestLooseUse", "TestQuotedFenceUse",
		"TestSingleQuoted", "TestDoubleQuoted", "TestAnchored", "TestAlternationA",
		"TestAlternationB", "TestSubtestPath", "TestShellUse", "TestTrailingComment",
		"TestTaskfileUse", "TestWorkflowUse", "TestUnclosedIsAUse",
		"TestCorpus3_29VsOracle", "TestCorpus3_29",
	}
	var b strings.Builder
	b.WriteString("package calculator\n\nimport \"testing\"\n")
	for _, n := range names {
		fmt.Fprintf(&b, "\nfunc %s(t *testing.T) { _ = t }\n", n)
	}
	return b.String()
}

func ttFixtureNames() string {
	names := []string{
		"TestAlternationA", "TestAlternationB", "TestAnchored", "TestCorpus3_29",
		"TestCorpus3_29VsOracle", "TestDoubleQuoted", "TestFencedUse", "TestIndentedUse",
		"TestLooseUse", "TestQuotedFenceUse", "TestShellUse", "TestSingleQuoted",
		"TestSubtestPath", "TestTaskfileUse", "TestTrailingComment", "TestUnclosedIsAUse",
		"TestWorkflowUse",
	}
	return strings.Join(names, "\n") + "\n"
}

func ttSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, ttPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			ttPy:                                string(py),
			"scripts/task_test_refs.names":      ttFixtureNames(),
			"DRIVER.md":                         ttFixtureMarkdown(),
			"scripts/driver.sh":                 ttFixtureShell(),
			"Taskfile.yml":                      ttFixtureTaskfile(),
			".github/workflow.yml":              ttFixtureWorkflow(),
			"calculator/fixture_test.go":        ttFixtureTests(),
			"frontend/skipme_test.go":           "package frontend\n\nfunc TestSkippedTree(t *testing.T) {}\n",
			"scripts/node_modules/skip_test.go": "package nm\n\nfunc TestAlsoSkipped(t *testing.T) {}\n",
		}
		for rel, body := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}} {
			cmd := exec.CommandContext(t.Context(), "git", args...)
			cmd.Dir = sb
			if out, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("sandbox: git %s: %w: %s", args[0], err, out)
			}
		}
		return sb, nil
	}
}

func ttSpecFor(t *testing.T, bin string, extraEnv []string, args ...string) difftest.Spec {
	t.Helper()
	env := append([]string{
		"GOPOB_BIN=" + bin,
		"SELFTEST=",
		"SELFTEST_DIR=",
	}, extraEnv...)
	return difftest.Spec{
		Tool:         "task-test-refs",
		PythonScript: ttPy,
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"task-test-refs"}, args...),
		Env:          env,
		Stderr:       difftest.StderrExact,
	}
}

func TestTaskTestRefsAgainstPython(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ttSandbox(t, repo)
	s := ttSpecFor(t, bin, nil)
	s.RefusalSites = 2
	s.Fixtures = []difftest.Fixture{
		{Name: "happy"},
		{
			Name:    "a-driver-is-not-utf8",
			Refusal: "is not valid UTF-8",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "DRIVER.md"),
					[]byte("# bad\n\xff\xfe\n"), 0o600)
			},
		},
		{
			Name:    "the-gate-versions-emitter-produces-nothing",
			Refusal: "`gopob gate-versions --tsv` produced nothing",
			Break: func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "scripts/task_test_refs.names"),
					[]byte(ttFixtureNames()), 0o600)
			},
		},
	}
	difftest.AssertRefusalSiteCount(t, s, ttReadPy(t, repo))
	for _, f := range s.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "the-gate-versions-emitter-produces-nothing" {
				t.Skip("covered by TestTaskTestRefsGateEmitterRefusal")
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

func ttReadPy(t *testing.T, repo string) []byte {
	t.Helper()
	b := difftest.ReadOriginal(t, repo, ttPy)
	return b
}

// TestTaskTestRefsGateEmitterRefusal points GOPOB_BIN at a binary that exits non-zero, which is the only way to reach the bridge's refusal.
func TestTaskTestRefsGateEmitterRefusal(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ttSandbox(t, repo)
	dud := filepath.Join(t.TempDir(), "dud.sh")
	if err := os.WriteFile(dud, []byte("#!/bin/sh\necho 'the emitter is broken' >&2\nexit 4\n"), 0o700); err != nil { //nolint:gosec // a fixture stub
		t.Fatal(err)
	}
	s := ttSpecFor(t, bin, []string{"GOPOB_BIN=" + dud})
	s.Env[0] = "GOPOB_BIN=" + dud
	difftest.RunFixture(t, t.Context(), s,
		difftest.Fixture{Name: "gate-emitter-fails",
			Refusal: "`gopob gate-versions --tsv` produced nothing (rc=4)",
			Break:   func(string) error { return nil }}, mk)
}

// TestTaskTestRefsModes sweeps the paths that are FINDINGS rather than refusals — exit 1 — plus the selftest and the name-set guard's two refusals.
func TestTaskTestRefsModes(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ttSandbox(t, repo)
	for _, tc := range []struct {
		name string
		env  []string
		args []string
		brk  func(string) error
	}{
		{"selftest", []string{"SELFTEST=1", "SELFTEST_DIR=" + t.TempDir()}, nil, nil},
		{
			"a-dead-use-is-a-finding", nil, nil,
			func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "scripts/dead.sh"),
					[]byte("go test ./calculator/ -run TestNoSuchTestAnywhere\n"), 0o600)
			},
		},
		{
			"a-dead-mention-is-not-a-finding", nil, nil,
			func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "scripts/prose.sh"),
					[]byte("# go test -run TestNoSuchTestAnywhere is the historical record\n"), 0o600)
			},
		},
		{
			"the-name-set-shrank", nil, nil,
			func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "scripts/task_test_refs.names"),
					[]byte(ttFixtureNames()+"TestVanishedFromEveryDriver\n"), 0o600)
			},
		},
		{
			"the-name-set-gained", nil, nil,
			func(sb string) error {
				body := strings.Replace(ttFixtureNames(), "TestShellUse\n", "", 1)
				return os.WriteFile(filepath.Join(sb, "scripts/task_test_refs.names"),
					[]byte(body), 0o600)
			},
		},
		{
			"the-names-file-is-missing", nil, nil,
			func(sb string) error { return os.Remove(filepath.Join(sb, "scripts/task_test_refs.names")) },
		},
		{
			"the-names-file-is-empty", nil, nil,
			func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "scripts/task_test_refs.names"), nil, 0o600)
			},
		},
		{
			"no-test-functions-at-all", nil, nil,
			func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "calculator/fixture_test.go"),
					[]byte("package calculator\n"), 0o600)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), ttSpecFor(t, bin, tc.env, tc.args...),
				difftest.Fixture{Name: tc.name, Break: tc.brk}, mk)
		})
	}
}

// TestTaskTestRefsUpdateNames declares the FILE.
func TestTaskTestRefsUpdateNames(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ttSandbox(t, repo)
	s := ttSpecFor(t, bin, nil, "--update-names")
	s.Files = []string{"scripts/task_test_refs.names"}
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "update-names"}, mk)
}

func TestTaskTestRefsPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ttSandbox(t, repo)
	s := difftest.Spec{
		Tool:         "task-test-refs",
		PythonScript: ttPy,
		Env:          []string{"GOPOB_BIN=" + bin, "SELFTEST=", "SELFTEST_DIR="},
	}
	difftest.SelfIdentity(t, t.Context(), s, mk, []string{"0", "12345", "999983"})
}

// TestTaskTestRefsFixtureReachesTheCode is the control on the SANDBOX rather than the tool.
func TestTaskTestRefsFixtureReachesTheCode(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ttSandbox(t, repo)
	sb, err := mk()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), bin, "task-test-refs")
	cmd.Dir = sb
	cmd.Env = append(os.Environ(), "GOPOB_BIN="+bin, "SELFTEST=", "SELFTEST_DIR=")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("the happy fixture did not pass: %v\n%s", err, out)
	}
	body := string(out)
	for _, want := range []string{
		"historical mentions (not failures): 6",
		"DRIVER.md: TestInlineMention` named in prose",
		"DRIVER.md: TestDoubledMention`` named in prose",
		"DRIVER.md: TestBacktrackedMention` named in prose",
		"DRIVER.md: TestLongGoneFromTheTree`. named in prose",
		"scripts/driver.sh: TestCommentMention named in prose",
		"scripts/driver.sh: TestAfterHashMention named in prose",
		"task test refs OK: 17 -run use(s) across 4 discovered driver(s), 17 test functions",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("the fixture never reached the code it claims to cover: %q missing\n%s", want, body)
		}
	}
}
