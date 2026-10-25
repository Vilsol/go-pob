package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(date string, args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=a", "GIT_AUTHOR_EMAIL=a@example.com", "GIT_COMMITTER_NAME=a",
			"GIT_COMMITTER_EMAIL=a@example.com", "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date, "GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commit := func(date, msg string) { run(date, "commit", "--allow-empty", "-q", "-m", msg) }
	run("2026-01-01T00:00:00Z", "init", "-q", "-b", "main")
	commit("2026-01-01T00:00:00Z", "chore: initial commit")
	run("2026-01-01T00:00:00Z", "tag", "v0.0.0")
	commit("2026-01-02T10:00:00Z", "feat: unscoped feature")
	commit("2026-01-02T11:00:00Z", "fix(b): second scope")
	commit("2026-01-02T12:00:00Z", "chore: hidden")
	commit("2026-01-02T13:00:00Z", "feat(a): Zeta")
	commit("2026-01-02T14:00:00Z", "feat(a): alpha")
	commit("2026-01-02T23:30:00-02:00", "perf: faster")
	run("2026-01-03T01:30:00Z", "tag", "v0.1.0")
	commit("2026-01-04T00:00:00Z", "test: hidden test")
	commit("2026-01-05T00:00:00Z", "fix: a fix")
	run("2026-01-05T00:00:00Z", "tag", "v0.1.1")
	commit("2026-01-06T00:00:00Z", "refactor(core)!: drop the old API")
	commit("2026-01-07T00:00:00Z", "fix: another\n\nBREAKING CHANGE: the file moves")
	commit("2026-01-08T00:00:00Z", "Revert \"fix: a fix\"\n\nThis reverts commit 0123456789abcdef.")
	commit("2026-01-09T00:00:00Z", "docs: not listed")
	run("2026-01-09T00:00:00Z", "tag", "not-a-version")
	return dir
}

const want = `VERSION[0.2.0][2026/01/08]

--- Breaking Changes ---
* the file moves
* core: drop the old API

--- Fixed ---
* another

--- Reverts ---
* Revert "fix: a fix"

--- Code Refactoring ---
* core: drop the old API


VERSION[0.1.1][2026/01/05]

--- Fixed ---
* a fix


VERSION[0.1.0][2026/01/03]

--- New ---
* a: alpha
* a: Zeta
* unscoped feature

--- Fixed ---
* b: second scope

--- Performance Improvements ---
* faster
`

func TestGenerateWritesEveryTagAndTheNextVersion(t *testing.T) {
	dir := repo(t)
	got, err := generate(dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	again, err := generate(dir, "HEAD")
	if err != nil || again != got {
		t.Fatalf("a second run differs: %v\n%s", err, again)
	}
}

func TestGenerateWritesNoNextVersionWithoutReleasableCommits(t *testing.T) {
	got, err := generate(repo(t), "v0.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if tail := want[strings.Index(want, "VERSION[0.1.1]"):]; got != tail {
		t.Fatalf("got\n%s\nwant\n%s", got, tail)
	}
}
