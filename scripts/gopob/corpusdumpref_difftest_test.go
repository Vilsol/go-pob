package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const cdrSh = "scripts/corpus_dump_ref.sh"

const cdrGoStub = `#!/bin/sh
echo "go-stub: args=$*"
echo "go-stub: DUMP=${DUMP:-unset}"
if [ -e .not-in-commit ]; then
	echo "go-stub: cwd is the SANDBOX — the dump would be of the WORKING TREE"
else
	echo "go-stub: cwd is the worktree"
fi
# ⚠️ AN ENV SWITCH, NOT A MARKER FILE. This stub runs with the THROWAWAY WORKTREE as its cwd -- a
# checkout of the commit in /tmp -- so a file planted in the sandbox is not visible to it at all.
# The first attempt used one and the stub never failed, which the fixture reported as "reached no
# refusal matching FAILED".
[ -n "${GO_STUB_FAIL:-}" ] && exit 3
exit 0
`

func cdrSandbox(t *testing.T, repo string) (func() (string, error), string) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, cdrSh)
	base := t.TempDir()
	binDir := filepath.Join(base, "bin")
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(cdrGoStub), 0o700); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			"calculator/keep.go": "package calculator\n",
			"README.md":          "# sandbox\n",
			".gitignore":         cdrSh + "\n",
		}
		for rel, b := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			mode := os.FileMode(0o600)
			if filepath.Ext(rel) == ".sh" {
				mode = 0o700
			}
			if err := os.WriteFile(dst, []byte(b), mode); err != nil { //nolint:gosec
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		const when = "2000-01-01T00:00:00 +0000"
		for _, args := range [][]string{
			{"init", "-q", "-b", "main"}, {"add", "-A"},
			{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "base"},
		} {
			cmd := exec.CommandContext(t.Context(), "git", args...) //nolint:gosec // the harness's own sandbox
			cmd.Dir = sb
			cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+when, "GIT_COMMITTER_DATE="+when)
			if out, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("sandbox: git %s: %w: %s", args[0], err, out)
			}
		}
		if err := os.MkdirAll(filepath.Join(sb, "realdir"), 0o750); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		if err := os.Symlink("realdir", filepath.Join(sb, "linkdir")); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		if sh != nil {
			dst := filepath.Join(sb, cdrSh)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, sh, 0o700); err != nil { //nolint:gosec // the sandbox's own copy of the original
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		if err := os.WriteFile(filepath.Join(sb, ".not-in-commit"), []byte("marker\n"), 0o600); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		return sb, nil
	}, binDir
}

func cdrSpec(bin, binDir string, env []string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "corpus-dump-ref",
		PythonScript: cdrSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"corpus-dump-ref"}, args...),
		Stderr:       difftest.StderrExact,
		Env: append([]string{
			"GOPOB_NICED=1",
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		}, env...),
		RefusalSites: 6,
	}
}

func TestCorpusDumpRefPortIsIndistinguishable(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := cdrSandbox(t, repo)

	shSrc := difftest.ReadOriginal(t, repo, cdrSh)

	type fixture struct {
		difftest.Fixture
		args []string
		env  []string
	}
	fixtures := []fixture{
		{args: []string{"HEAD", "out.txt.gz"}, Fixture: difftest.Fixture{Name: "happy"}},

		{args: []string{}, Fixture: difftest.Fixture{
			Name: "no-arguments", Refusal: "usage: corpus_dump_ref.sh <ref> <out>",
			Break: func(string) error { return nil }}},

		{args: []string{"HEAD"}, Fixture: difftest.Fixture{
			Name: "only-a-ref", Refusal: "usage: corpus_dump_ref.sh <ref> <out>",
			Break: func(string) error { return nil }}},

		{args: []string{"HEAD", ""}, Fixture: difftest.Fixture{
			Name: "empty-out-argument", Refusal: "usage: corpus_dump_ref.sh <ref> <out>",
			Break: func(string) error { return nil }}},

		{args: []string{"no-such-ref", "out.txt.gz"}, Fixture: difftest.Fixture{
			Name: "ref-does-not-resolve", Refusal: "not a commit: no-such-ref",
			Break: func(string) error { return nil }}},

		{args: []string{"HEAD^{tree}", "out.txt.gz"}, Fixture: difftest.Fixture{
			Name: "a-tree-is-not-a-commit", Refusal: "not a commit: HEAD^{tree}",
			Break: func(string) error { return nil }}},

		{Fixture: difftest.Fixture{Name: "worktree-add-fails", NoFixture: true,
			Refusal: "could not check",
			Note: "reachable (a file at .git/worktrees), but git's own fatal: names the mktemp " +
				"directory, so the two sides can only ever print different random suffixes"}},

		{Fixture: difftest.Fixture{Name: "cd-into-the-worktree-fails", NoFixture: true,
			Refusal: `cd "$WT" || exit 1`,
			Note: "git worktree add has just created the directory on the line above; losing it " +
				"before the cd needs a concurrent deletion no sandbox can stage"}},

		{args: []string{"HEAD", "linkdir/out.txt.gz"}, Fixture: difftest.Fixture{
			Name: "the-out-path-is-resolved-through-a-symlink", Passes: true,
			Break: func(string) error { return nil }}},

		{args: []string{"HEAD", "out.txt.gz"}, env: []string{"GO_STUB_FAIL=1"},
			Fixture: difftest.Fixture{
				Name: "the-dump-fails", Refusal: "FAILED (go test exit 3)",
				Break: func(string) error { return nil }}},
	}

	table := make([]difftest.Fixture, 0, len(fixtures))
	for _, f := range fixtures {
		table = append(table, f.Fixture)
	}
	spec := cdrSpec(bin, binDir, nil, "HEAD", "out.txt.gz")
	spec.Fixtures = table
	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range fixtures {
		if f.NoFixture {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), cdrSpec(bin, binDir, f.env, f.args...), f.Fixture, mk)
		})
	}
}

// TestCorpusDumpRefLeavesNoWorktreeBehind: THE WORKTREE MUST BE GONE AFTERWARDS, AND NO OUTPUT SHOWS THAT.
func TestCorpusDumpRefLeavesNoWorktreeBehind(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := cdrSandbox(t, repo)

	check := func(t *testing.T, name string, argv []string) {
		t.Helper()
		sb, err := mk()
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), argv[0], argv[1:]...) //nolint:gosec // the harness's own sandbox
		cmd.Dir = sb
		cmd.Env = append(os.Environ(), "GOPOB_NICED=1",
			"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", name, err, out)
		}
		list := exec.CommandContext(t.Context(), "git", "worktree", "list", "--porcelain")
		list.Dir = sb
		out, err := list.Output()
		if err != nil {
			t.Fatal(err)
		}
		if n := countWorktrees(string(out)); n != 1 {
			t.Errorf("%s: %d worktrees registered afterwards, want 1; the throwaway checkout was "+
				"left behind and `git worktree add` will refuse that path next time\n%s",
				name, n, out)
		}
	}

	if !difftest.OriginalsDeleted(repo) {
		check(t, "shell", []string{"bash", cdrSh, "HEAD", "out.txt.gz"})
	}
	check(t, "go", []string{bin, "corpus-dump-ref", "HEAD", "out.txt.gz"})
}

func countWorktrees(porcelain string) int {
	n := 0
	for _, line := range pyUniversalLines(porcelain) {
		if len(line) >= 9 && line[:9] == "worktree " {
			n++
		}
	}
	return n
}

func TestCorpusDumpRefIsStableAcrossLocales(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	mk, binDir := cdrSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "corpus-dump-ref",
		PythonScript: cdrSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   []string{"HEAD", "out.txt.gz"},
		Env: []string{
			"GOPOB_NICED=1",
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		},
	}, mk, []string{"C", "en_US.utf8", "C.utf8"})
}
