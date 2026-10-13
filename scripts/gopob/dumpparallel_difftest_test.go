package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const dpSh = "scripts/dump_parallel.sh"

const dpFakeLuajit = `#!/bin/sh
# $1 dump_build.lua   $2 builds dir   $3 out dir   -- the last two relative to the SANDBOX ROOT
cd ../.. || exit 9
out="$3"
n=0
for b in "$2"/*; do
	[ -e "$b" ] || continue
	n=$((n + 1))
	base=$(basename "$b" .xml)
	printf '{"build":"%s"}\n' "$base" > "$out/$base.json"
done
echo "fake luajit: wrote $n files to $out"
# ⚠️ A DOTFILE IN THE OUTPUT, SO THE FINAL COUNT'S RULE IS EXERCISED AT THE REPORTING END TOO.
# The emptiness guard uses ` + "`ls -A`" + ` (dotfiles COUNT) and the closing tally uses plain ` + "`ls`" + `
# (dotfiles DO NOT). The guard's half is covered by the target-holds-only-a-dotfile fixture; without
# this line nothing distinguishes the tally's half, and a port using one rule for both would be
# caught in only one direction.
: > "$out/.dumper-state"
# per-run failure switch: create <out>.fail before launching to make THIS run fail
[ -f "$out.fail" ] && exit 7
exit 0
`

func dpSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, dpSh)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			dpSh:                    string(sh),
			"pob/src/Launch.lua":    "-- a PoB source checkout\n",
			"pob/src/fakeluajit":    dpFakeLuajit,
			"oracle/dump_build.lua": "-- the real dumper, never run here\n",
			"builds/alpha.xml":      "<PathOfBuilding/>\n",
			"builds/bravo.xml":      "<PathOfBuilding/>\n",
			"builds/charlie.xml":    "<PathOfBuilding/>\n",
		}
		for rel, b := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			mode := os.FileMode(0o600)
			if filepath.Base(rel) == "fakeluajit" {
				mode = 0o700
			}
			if err := os.WriteFile(dst, []byte(b), mode); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

var dpDefaultArgs = []string{"pob", ".", "builds", "out", "2"}

func dpSpec(bin string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "dump-parallel",
		PythonScript: dpSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"dump-parallel"}, args...),
		Stderr:       difftest.StderrExact,
		Env:          []string{"LUAJIT=./fakeluajit", "NICE=0", "GOPOB_NICED=1"},
		RefusalSites: 7,
	}
}

// TestDumpParallelPortIsIndistinguishable: THE HAPPY PATH REALLY LAUNCHES `setarch -R unshare -rn`.
func TestDumpParallelPortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := dpSandbox(t, repo)

	shSrc := difftest.ReadOriginal(t, repo, dpSh)

	type fixture struct {
		difftest.Fixture
		args []string
	}
	fixtures := []fixture{
		{Fixture: difftest.Fixture{Name: "happy"}},

		{args: []string{}, Fixture: difftest.Fixture{
			Name: "no-arguments", Refusal: "FATAL: missing <pob-src-parent>",
			Break: func(string) error { return nil }}},
		{args: []string{"pob"}, Fixture: difftest.Fixture{
			Name: "one-argument", Refusal: "FATAL: missing <repo-root>",
			Break: func(string) error { return nil }}},
		{args: []string{"pob", ".", "builds", "out"}, Fixture: difftest.Fixture{
			Name: "four-arguments", Refusal: "FATAL: missing <runs>",
			Break: func(string) error { return nil }}},

		{args: []string{"pob", ".", "builds", "out", "two"}, Fixture: difftest.Fixture{
			Name: "runs-not-a-number", Refusal: "<runs> must be a positive integer, not 'two'",
			Break: func(string) error { return nil }}},

		{args: []string{"pob", ".", "builds", "out", "0"}, Fixture: difftest.Fixture{
			Name: "zero-runs", Refusal: "FATAL: <runs> must be at least 1, not 0",
			Break: func(string) error { return nil }}},

		{Fixture: difftest.Fixture{Name: "no-pob-src", Refusal: "FATAL: no pob/src",
			Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "pob/src")) }}},

		{Fixture: difftest.Fixture{Name: "no-builds-dir", Refusal: "FATAL: no builds dir builds",
			Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "builds")) }}},

		{Fixture: difftest.Fixture{
			Name: "target-directory-not-empty", Refusal: "out-2 exists and is NOT EMPTY",
			Break: func(sb string) error {
				if err := os.MkdirAll(filepath.Join(sb, "out-2"), 0o750); err != nil {
					return fmt.Errorf("fixture: %w", err)
				}
				return put(sb, "out-2/leftover.json", "{}")
			}}},

		{Fixture: difftest.Fixture{
			Name: "target-holds-only-a-dotfile", Refusal: "out-1 exists and is NOT EMPTY",
			Break: func(sb string) error {
				if err := os.MkdirAll(filepath.Join(sb, "out-1"), 0o750); err != nil {
					return fmt.Errorf("fixture: %w", err)
				}
				return put(sb, "out-1/.keep", "")
			}}},

		{Fixture: difftest.Fixture{Name: "one-run-fails", Refusal: "dump_parallel: run 2 FAILED",
			Break: func(sb string) error { return put(sb, "out-2.fail", "") }}},
	}

	table := make([]difftest.Fixture, 0, len(fixtures))
	for _, f := range fixtures {
		table = append(table, f.Fixture)
	}
	spec := dpSpec(bin, dpDefaultArgs...)
	spec.Fixtures = table
	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			args := f.args
			if args == nil {
				args = dpDefaultArgs
			}
			difftest.RunFixture(t, t.Context(), dpSpec(bin, args...), f.Fixture, mk)
		})
	}
}

// TestDumpParallelRefusesBeforeCreatingAnything: AGREEMENT IS NOT THE WHOLE TEST HERE.
func TestDumpParallelRefusesBeforeCreatingAnything(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := dpSandbox(t, repo)

	check := func(t *testing.T, name string, argv []string) {
		t.Helper()
		sb, err := mk()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(sb, "out-2"), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sb, "out-2/leftover.json"), []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}

		cmd := exec.CommandContext(t.Context(), argv[0], argv[1:]...) //nolint:gosec // the harness's own sandbox
		cmd.Dir = sb
		cmd.Env = append(os.Environ(), "LUAJIT=./fakeluajit", "NICE=0", "GOPOB_NICED=1")
		out, err := cmd.CombinedOutput()
		var ee *exec.ExitError
		if !errors.As(err, &ee) || ee.ExitCode() != 3 {
			t.Fatalf("%s: want exit 3, got %v\n%s", name, err, out)
		}
		for _, forbidden := range []string{"out-1", "out-1.log", "out-2.log"} {
			if _, err := os.Stat(filepath.Join(sb, forbidden)); err == nil {
				t.Errorf("%s: %s was created before the refusal — the runner is NOT "+
					"non-destructive, and a concurrent run's output is at risk. That is the exact "+
					"regression the `refuse before creating anything` loop exists to prevent.",
					name, forbidden)
			}
		}
		if _, err := os.Stat(filepath.Join(sb, "out-2/leftover.json")); err != nil {
			t.Errorf("%s: the colliding run's own output was removed: %v", name, err)
		}
	}

	if !difftest.OriginalsDeleted(repo) {
		check(t, "shell", []string{"bash", dpSh, "pob", ".", "builds", "out", "2"})
	}
	check(t, "go", []string{bin, "dump-parallel", "pob", ".", "builds", "out", "2"})
}

// TestDumpParallelCapturesEachRunsLog: THE PER-RUN LOG IS DECLARED AS A COMPARED FILE, AND IT CAUGHT A REAL DIVERGENCE.
func TestDumpParallelCapturesEachRunsLog(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := dpSandbox(t, repo)

	spec := dpSpec(bin, dpDefaultArgs...)
	spec.Files = []string{"out-1.log", "out-2.log"}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "logs-are-identical", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

func TestDumpParallelIsStableAcrossLocales(t *testing.T) {
	repo := repoRoot(t)
	mk := dpSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "dump-parallel",
		PythonScript: dpSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   dpDefaultArgs,
		Env:          []string{"LUAJIT=./fakeluajit", "NICE=0", "GOPOB_NICED=1"},
	}, mk, []string{"C", "en_US.utf8", "C.utf8"})
}
