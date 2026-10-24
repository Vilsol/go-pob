package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLintGoCacheIsPerWorktree: a shared golangci-lint cache replays findings from other worktrees.
func TestLintGoCacheIsPerWorktree(t *testing.T) {
	t.Parallel()
	task, err := exec.LookPath("task")
	if err != nil {
		t.Skip("no task binary")
	}
	repo := repoRoot(t)
	bin := t.TempDir()
	fake := "#!/bin/sh\nprintf '%s' \"$GOLANGCI_LINT_CACHE\" > " + filepath.Join(bin, "cache") + "\n"
	if err := os.WriteFile(filepath.Join(bin, "golangci-lint"), []byte(fake), 0o700); err != nil { //nolint:gosec // an executable stub
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), task, "lint-go")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "GOLANGCI_LINT_CACHE=/shared")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("task lint-go: %v\n%s", err, out)
	}
	got, err := os.ReadFile(filepath.Join(bin, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	cache := string(got)
	if !strings.HasPrefix(cache, repo+string(os.PathSeparator)) {
		t.Fatalf("GOLANGCI_LINT_CACHE=%q, want a directory under %s", cache, repo)
	}
	check := exec.CommandContext(t.Context(), "git", "check-ignore", "-q", cache)
	check.Dir = repo
	if err := check.Run(); err != nil {
		t.Fatalf("%s is not gitignored: %v", cache, err)
	}
}
