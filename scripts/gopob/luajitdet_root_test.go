package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func ljWorktreePair(t *testing.T) (string, string) {
	t.Helper()
	base := t.TempDir()
	main := filepath.Join(base, "main")
	if err := os.MkdirAll(main, 0o750); err != nil {
		t.Fatal(err)
	}
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run(main, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(main, "seed"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(main, "add", "seed")
	run(main, "-c", "commit.gpgsign=false", "commit", "-qm", "seed")
	linked := filepath.Join(base, "linked")
	run(main, "worktree", "add", "-q", "-b", "wt", linked)
	return main, linked
}

func TestDetLuajitBinResolvesTheMainCheckoutFromALinkedWorktree(t *testing.T) {
	main, linked := ljWorktreePair(t)
	t.Setenv("GOPOB_ROOT", "")
	t.Chdir(linked)

	got := detLuajitBin()
	want := filepath.Join(main, ".luajit-det", "bin", "luajit")
	if got != want {
		t.Fatalf("detLuajitBin() from a linked worktree = %q, want the MAIN checkout's %q", got, want)
	}
}

func TestDetLuajitBinHonoursGopobRoot(t *testing.T) {
	_, linked := ljWorktreePair(t)
	t.Setenv("GOPOB_ROOT", linked)
	want := filepath.Join(linked, ".luajit-det", "bin", "luajit")
	if got := detLuajitBin(); got != want {
		t.Fatalf("detLuajitBin() with GOPOB_ROOT=%q = %q, want %q", linked, got, want)
	}
}

// TestRegenDataUsesTheSharedDetLuajitFromALinkedWorktree: THE LOAD-BEARING ONE.
func TestRegenDataUsesTheSharedDetLuajitFromALinkedWorktree(t *testing.T) {
	main, linked := ljWorktreePair(t)
	t.Setenv("GOPOB_ROOT", "")
	t.Chdir(linked)

	bin := filepath.Join(main, ".luajit-det", "bin", "luajit")
	if err := os.MkdirAll(filepath.Dir(bin), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o700); err != nil { //nolint:gosec // a probe stub
		t.Fatal(err)
	}

	got := (&rdRun{repo: linked}).luajitCmd()
	want := []string{"setarch", "-R", bin}
	if len(got) != len(want) {
		t.Fatalf("luajitCmd() = %v, want %v (a short slice is the stock-luajit fallback)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("luajitCmd() = %v, want %v", got, want)
		}
	}
}
