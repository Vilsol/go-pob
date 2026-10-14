package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const dsSh = "scripts/dump_shard.sh"

const dsFakeLuajit = `#!/bin/sh
# $1 dump_build.lua   $2 shard dir   $3 out dir  -- both relative to the SANDBOX ROOT
cd ../.. || exit 9
# ADDR_NO_RANDOMIZE IS WHAT setarch -R ACTUALLY DOES, AND /proc/self/personality IS WHERE IT SHOWS.
# Without this line nothing in the fixture can observe the flag: dropping -R still runs the shard,
# still dumps every build, and goes UNCAUGHT -- measured. ASLR's real effect (LuaJIT pairs() order)
# needs a real interpreter over real data, but the FLAG itself does not.
# (No backticks in here: this is a Go RAW STRING and one would end it -- which it did, and the
# compile error then made the mutation control report a spurious CAUGHT.)
echo "personality=$(cat /proc/self/personality 2>/dev/null)"
for b in "$2"/*.xml; do
	[ -e "$b" ] || continue
	base=$(basename "$b" .xml)
	echo "shard saw $base"
	printf '{"build":"%s"}\n' "$base" > "$3/$base.json"
done
exit 0
`

func dsSandbox(t *testing.T, repo string, builds int) func() (string, error) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, dsSh)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			dsSh:                      string(sh),
			"pob/src/Launch.lua":      "-- a PoB source checkout\n",
			"pob/src/fakeluajit":      dsFakeLuajit,
			"pob/src/luajit-det-fake": dsFakeLuajit,
			"oracle/dump_build.lua":   "-- the real dumper, never run here\n",
		}
		for i := range builds {
			files[fmt.Sprintf("builds/b%02d.xml", i)] = "<PathOfBuilding/>\n"
		}
		for rel, b := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			mode := os.FileMode(0o600)
			if strings.Contains(filepath.Base(rel), "luajit") || filepath.Ext(rel) == ".sh" {
				mode = 0o700
			}
			if err := os.WriteFile(dst, []byte(b), mode); err != nil { //nolint:gosec
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

var dsDefaultArgs = []string{"pob", ".", "builds", "out", "3"}

func dsSpec(bin string, env []string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "dump-shard",
		PythonScript: dsSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"dump-shard"}, args...),
		Stderr:       difftest.StderrExact,
		Env: append([]string{
			"LUAJIT=./fakeluajit", "NICE=0", "GOPOB_NICED=1",
		}, env...),
		RefusalSites: 10,
	}
}

func TestDumpShardPortIsIndistinguishable(t *testing.T) {
	requireWorks(t, probeNamespaces)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := dsSandbox(t, repo, 7)

	shSrc := difftest.ReadOriginal(t, repo, dsSh)

	type fixture struct {
		difftest.Fixture
		args []string
		env  []string
	}
	fixtures := []fixture{
		{Fixture: difftest.Fixture{Name: "happy"}},

		{args: []string{}, Fixture: difftest.Fixture{
			Name: "no-arguments", Refusal: "FATAL: missing <pob-dir>",
			Break: func(string) error { return nil }}},

		{args: []string{"pob", ".", "builds", "out"}, Fixture: difftest.Fixture{
			Name: "four-arguments", Refusal: "FATAL: missing <shards>",
			Break: func(string) error { return nil }}},

		{args: []string{"pob", ".", "builds", "out", "lots"}, Fixture: difftest.Fixture{
			Name:    "shards-not-a-number",
			Refusal: "<shards> must be a positive integer, not 'lots'",
			Break:   func(string) error { return nil }}},

		{args: []string{"pob", ".", "builds", "out", "0"}, Fixture: difftest.Fixture{
			Name: "zero-shards", Refusal: "FATAL: <shards> must be at least 1, not 0",
			Break: func(string) error { return nil }}},

		{Fixture: difftest.Fixture{Name: "no-pob-src", Refusal: "FATAL: no pob/src",
			Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "pob/src")) }}},

		{Fixture: difftest.Fixture{Name: "no-builds-dir", Refusal: "FATAL: no builds dir builds",
			Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "builds")) }}},

		{Fixture: difftest.Fixture{Name: "builds-dir-has-no-xml",
			Refusal: "a sharded run over nothing would look like success",
			Break: func(sb string) error {
				matches, err := filepath.Glob(filepath.Join(sb, "builds", "*.xml"))
				if err != nil {
					return fmt.Errorf("fixture: %w", err)
				}
				for _, m := range matches {
					if err := os.Remove(m); err != nil {
						return fmt.Errorf("fixture: %w", err)
					}
				}
				return nil
			}}},

		{Fixture: difftest.Fixture{Name: "output-directory-not-empty",
			Refusal: "out exists and is NOT EMPTY",
			Break: func(sb string) error {
				if err := os.MkdirAll(filepath.Join(sb, "out"), 0o750); err != nil {
					return fmt.Errorf("fixture: %w", err)
				}
				return put(sb, "out/leftover.json", "{}")
			}}},

		{Fixture: difftest.Fixture{Name: "a-shortfall-is-not-success",
			Refusal: "SHORTFALL of 1",
			Break: func(sb string) error {
				return put(sb, "pob/src/fakeluajit", `#!/bin/sh
cd ../.. || exit 9
for b in "$2"/*.xml; do
	[ -e "$b" ] || continue
	base=$(basename "$b" .xml)
	[ "$base" = "b00" ] && continue
	printf '{"build":"%s"}\n' "$base" > "$3/$base.json"
done
exit 0
`)
			}}},

		{Fixture: difftest.Fixture{Name: "a-shard-fails-after-dumping-everything", Finding: true,
			Break: func(sb string) error {
				return put(sb, "pob/src/fakeluajit", `#!/bin/sh
cd ../.. || exit 9
for b in "$2"/*.xml; do
	[ -e "$b" ] || continue
	base=$(basename "$b" .xml)
	printf '{"build":"%s"}\n' "$base" > "$3/$base.json"
done
exit 3
`)
			}}},

		{Fixture: difftest.Fixture{Name: "exit-carries-a-failed-shard", NoFixture: true,
			Refusal: `exit "$rc"`,
			Note: "reached by a-shard-fails-after-dumping-everything, but it reports on STDOUT " +
				"and Spec.RefusalOnStdout is tool-wide; switching it would unmatch nine stderr sites"}},

		{env: []string{"LUAJIT=./luajit-det-fake"}, Fixture: difftest.Fixture{
			Name: "deterministic-luajit-is-sound", Passes: true,
			Break: func(string) error { return nil }}},
	}

	table := make([]difftest.Fixture, 0, len(fixtures))
	for _, f := range fixtures {
		table = append(table, f.Fixture)
	}
	spec := dsSpec(bin, nil, dsDefaultArgs...)
	spec.Fixtures = table
	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range fixtures {
		if f.NoFixture {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			args := f.args
			if args == nil {
				args = dsDefaultArgs
			}
			difftest.RunFixture(t, t.Context(), dsSpec(bin, f.env, args...), f.Fixture, mk)
		})
	}
}

// TestDumpShardSplitsRoundRobin: THE ROUND-ROBIN SPLIT IS INVISIBLE IN THE TOOL'S OWN OUTPUT, and it is the reason this script is not three lines.
func TestDumpShardSplitsRoundRobin(t *testing.T) {
	requireWorks(t, probeNamespaces)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := dsSandbox(t, repo, 7)

	spec := dsSpec(bin, nil, dsDefaultArgs...)
	spec.Files = []string{"out.shard-0.log", "out.shard-1.log", "out.shard-2.log"}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "each-shard-saw-the-same-builds", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

func TestDumpShardIsStableAcrossLocales(t *testing.T) {
	requireWorks(t, probeNamespaces)
	repo := repoRoot(t)
	mk := dsSandbox(t, repo, 7)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "dump-shard",
		PythonScript: dsSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   dsDefaultArgs,
		Env:          []string{"LUAJIT=./fakeluajit", "NICE=0", "GOPOB_NICED=1"},
	}, mk, []string{"C", "en_US.utf8", "C.utf8"})
}
