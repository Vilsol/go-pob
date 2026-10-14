package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const rfSh = "scripts/regen_fixtures.sh"

const rfFakeLuajit = `#!/bin/sh
# $1 is oracle/<name>.lua, the rest are the fixture's output paths.
lua="$1"; shift
name=$(basename "$lua" .lua)
# ⚠️ THE FAILURE SWITCH IS A FILE IN THE SANDBOX, NOT AN ENV VAR. Fixture.Finding re-runs on an
# UNBROKEN sandbox and requires the report to DIFFER; an env var applies to BOTH runs, so the tool
# reports the same thing twice and the fixture proves nothing. cwd here is <sandbox>/pob/src.
if [ "$name" = "mod_parse" ] && [ -f ../../.fx-fail ]; then
	echo "fake luajit: line one"  >&2
	echo "fake luajit: line two"  >&2
	echo "fake luajit: line three" >&2
	echo "fake luajit: line four" >&2
	exit 5
fi
echo "fake luajit: $name ran in $(pwd)"
for f in "$@"; do
	# ⚠️ AN EXISTING DIRECTORY IS AN **INPUT**, NOT AN OUTPUT. build_stats takes
	# testdata/many-builds as its first argument; writing to it fails, and the whole run exited 1
	# with no fixture asserting why -- which also made the self-identity sweep VACUOUS, because it
	# was comparing two refusals rather than two reports.
	[ -d "$f" ] && continue
	# ⚠️ THE **BASENAME**, NOT THE PATH. The full path is sandbox-absolute, so writing it makes every
	# product differ between the two sides for a reason that is the harness's, not the port's --
	# measured, as "130 vs 130 bytes". The basename still proves the argument list arrived intact,
	# because each fixture's outputs have distinct names, and Spec.Files proves they landed in the
	# right DIRECTORY by being found there at all.
	printf '{"probe":"%s","arg":"%s"}\n' "$name" "$(basename "$f")" > "$f" 2>/dev/null || exit 9
done
exit 0
`

func rfSandbox(t *testing.T, repo string) (func() (string, error), string, string) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, rfSh)
	base := t.TempDir()
	luajit := filepath.Join(base, "luajit-det")
	const notExec = "./luajit-not-executable"
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		if err := os.WriteFile(luajit, []byte(rfFakeLuajit), 0o700); err != nil { //nolint:gosec
			return "", fmt.Errorf("sandbox: %w", err)
		}
		files := map[string]string{
			rfSh:                    string(sh),
			"pob/src/Launch.lua":    "-- a PoB source checkout\n",
			"luajit-not-executable": rfFakeLuajit,
		}
		for _, name := range rfOrder {
			files["oracle/"+name+".lua"] = "-- probe " + name + "\n"
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
		for _, d := range []string{"testdata/oracle", "testdata/many-builds"} {
			if err := os.MkdirAll(filepath.Join(sb, d), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}, luajit, notExec
}

func rfSpec(bin, luajit string, env []string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "regen-fixtures",
		PythonScript: rfSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"regen-fixtures"}, args...),
		Stderr:       difftest.StderrExact,
		Env: append([]string{
			"GOPOB_NICED=1", "POB_DIR=pob", "LUAJIT=" + luajit,
			"LUA_PATH=/sandbox/?.lua;;",
		}, env...),
		RefusalSites: 5,
	}
}

func TestRegenFixturesPortIsIndistinguishable(t *testing.T) {
	requireWorks(t, probeSetarch)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, luajit, notExec := rfSandbox(t, repo)

	shSrc := difftest.ReadOriginal(t, repo, rfSh)

	type fixture struct {
		difftest.Fixture
		args []string
		env  []string
	}
	fixtures := []fixture{
		{Fixture: difftest.Fixture{Name: "happy"}},

		{args: []string{"mod_parse", "item_tools"}, Fixture: difftest.Fixture{
			Name: "named-subset-runs-in-the-order-given", Passes: true,
			Break: func(string) error { return nil }}},

		{args: []string{""}, Fixture: difftest.Fixture{
			Name: "an-empty-argument-means-all-eight", Passes: true,
			Break: func(string) error { return nil }}},

		{args: []string{"nope"}, Fixture: difftest.Fixture{
			Name: "unknown-fixture-name", Refusal: "no such fixture: nope",
			Break: func(string) error { return nil }}},

		{env: []string{"LUAJIT=./luajit-absent"}, Fixture: difftest.Fixture{
			Name: "no-deterministic-luajit", Refusal: "FATAL: no deterministic luajit at ",
			Break: func(string) error { return nil }}},

		{env: []string{"LUAJIT=" + notExec}, Fixture: difftest.Fixture{
			Name: "luajit-is-not-executable", Refusal: "FATAL: no deterministic luajit at ",
			Break: func(string) error { return nil }}},

		{Fixture: difftest.Fixture{Name: "no-pob-src", Refusal: "FATAL: no pob/src",
			Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "pob/src")) }}},

		{env: []string{"LUA_PATH="}, Fixture: difftest.Fixture{
			Name: "lua-path-not-resolved", Refusal: "FATAL: LUA_PATH not resolved from pob",
			Break: func(string) error { return nil }}},

		{args: []string{"mod_parse"},
			Fixture: difftest.Fixture{Name: "a-probe-fails", Finding: true,
				Break: func(sb string) error { return put(sb, ".fx-fail", "") }}},

		{Fixture: difftest.Fixture{Name: "exit-carries-the-failure-count", NoFixture: true,
			Refusal: "exit $fail",
			Note: "reached by a-probe-fails, but its text is on STDOUT and Spec.RefusalOnStdout " +
				"is tool-wide; switching it would unmatch the four FATAL sites on stderr"}},
	}

	table := make([]difftest.Fixture, 0, len(fixtures))
	for _, f := range fixtures {
		table = append(table, f.Fixture)
	}
	spec := rfSpec(bin, luajit, nil)
	spec.Fixtures = table
	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range fixtures {
		if f.NoFixture {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), rfSpec(bin, luajit, f.env, f.args...), f.Fixture, mk)
		})
	}
}

// TestRegenFixturesWritesEveryFixtureFile: THE FIXTURES ARE THE PRODUCT AND THE EIGHT PROGRESS LINES ARE THE LABEL.
func TestRegenFixturesWritesEveryFixtureFile(t *testing.T) {
	requireWorks(t, probeSetarch)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, luajit, _ := rfSandbox(t, repo)

	spec := rfSpec(bin, luajit, nil)
	spec.Files = []string{
		"testdata/oracle/item_tools_cases.json",
		"testdata/oracle/item_tools_expected.json",
		"testdata/oracle/many_builds_expected.json",
		"testdata/oracle/jewel_radius_expected.json",
		"testdata/oracle/item_parse_cases.json",
		"testdata/oracle/mod_parse_expected.json",
	}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "products-are-identical", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

func TestRegenFixturesIsStableAcrossLocales(t *testing.T) {
	requireWorks(t, probeSetarch)
	repo := repoRoot(t)
	mk, luajit, _ := rfSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "regen-fixtures",
		PythonScript: rfSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		Env: []string{
			"GOPOB_NICED=1", "POB_DIR=pob", "LUAJIT=" + luajit,
			"LUA_PATH=/sandbox/?.lua;;",
		},
	}, mk, []string{"C", "en_US.utf8", "C.utf8"})
}
