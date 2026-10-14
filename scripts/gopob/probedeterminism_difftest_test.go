package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const pdmSh = "oracle/probe_determinism.sh"

const pdmFakeLuajit = `#!/bin/sh
# the last argument is the output path; everything before it is the probe and its inputs
for last; do :; done
# ⚠️ THE MARKER IS FOUND RELATIVE TO **cwd**, WHICH IS <sandbox>/pob/src, NOT relative to the output
# path -- the outputs live in a mktemp workdir with no relation to the sandbox at all. Looking beside
# them found nothing and the varying branch was simply never taken, so the Finding fixture reported
# a happy path under another name.
# ⚠️ IT REFUSES WITHOUT ADDR_NO_RANDOMIZE, which is what setarch -R sets and /proc/self/personality
# reports (00040000 with, 00000000 without). A fake binary is unaffected by ASLR and its output is
# constant either way, so dropping -R was an UNCAUGHT mutation until this line existed -- the same
# gap, and the same fix, as build-luajit-det and dump-shard.
case "$(cat /proc/self/personality 2>/dev/null)" in
*40000) ;;
*) echo "fake luajit: ASLR is on; setarch -R was not applied" >&2; exit 9 ;;
esac
# ⚠️ A RUN THAT FAILS WITHOUT WRITING, so the "run N FAILED" path has a fixture. Without one, a port
# that md5'd the missing file anyway counted an ABSENT output as agreeing with the others.
case "$last" in
*-2.json) [ -e ../../.fail-run-2 ] && exit 1 ;;
esac
if [ -e ../../.vary ]; then
	printf 'order-%s\n' "$$" > "$last"
else
	printf 'a stable body\n' > "$last"
fi
exit 0
`

const pdmFakeMise = `#!/bin/sh
echo 'mise ERROR "luajit" couldn'"'"'t exec process: No such file or directory' >&2
exit 1
`

func pdmSandbox(t *testing.T, repo string) (func() (string, error), string) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, pdmSh)
	base := t.TempDir()
	luajit := filepath.Join(base, "fakeluajit")
	if err := os.WriteFile(luajit, []byte(pdmFakeLuajit), 0o700); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "mise"), []byte(pdmFakeMise), 0o700); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			pdmSh:                "-- placeholder\n",
			"pob/src/Launch.lua": "-- a PoB source checkout\n",
		}
		files[pdmSh] = string(sh)
		for _, p := range pdmProbes {
			files["oracle/"+p.lua] = "-- probe " + p.name + "\n"
		}
		for _, f := range []string{
			"testdata/oracle/item_tools_cases.json",
			"testdata/oracle/item_parse_cases.json",
			"testdata/oracle/item_bases_cases.json",
		} {
			files[f] = "[]\n"
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
		if err := os.MkdirAll(filepath.Join(sb, "testdata/many-builds"), 0o750); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		return sb, nil
	}, luajit
}

func pdmSpec(bin, luajit string, env []string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "probe-determinism",
		PythonScript: pdmSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"probe-determinism"}, args...),
		Stderr:       difftest.StderrExact,
		Env: append([]string{
			"GOPOB_NICED=1", "POB_DIR=pob", "LUAJIT=" + luajit,
			"PATH=" + filepath.Dir(luajit) + string(os.PathListSeparator) + os.Getenv("PATH"),
		}, env...),
		RefusalSites:    6,
		StderrNormalise: pdmCollapseMiseErrors,
	}
}

const pdmMiseToken = "<mise: could not run luajit>" //nolint:gosec // a placeholder token, not a credential

func pdmCollapseMiseErrors(b []byte) []byte {
	lines := strings.SplitAfter(string(b), "\n")
	isStart := func(l string) bool { return strings.HasPrefix(l, `mise ERROR "luajit"`) }
	var out strings.Builder
	for i := 0; i < len(lines); i++ {
		if !isStart(lines[i]) {
			out.WriteString(lines[i])
			continue
		}
		end := i
		for j := i + 1; j < len(lines) && !isStart(lines[j]); j++ {
			if strings.HasPrefix(lines[j], "mise ") {
				end = j
			}
		}
		out.WriteString(pdmMiseToken + "\n")
		i = end
	}
	return []byte(out.String())
}

func TestProbeDeterminismPortIsIndistinguishable(t *testing.T) {
	requireWorks(t, probeSetarch)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, luajit := pdmSandbox(t, repo)

	shSrc := difftest.ReadOriginal(t, repo, pdmSh)

	type fixture struct {
		difftest.Fixture
		args []string
	}
	fixtures := []fixture{
		{args: []string{"item_tools", "3"}, Fixture: difftest.Fixture{Name: "happy"}},

		{args: []string{"all", "2"}, Fixture: difftest.Fixture{
			Name: "all-runs-every-probe-in-order", Passes: true,
			Break: func(string) error { return nil }}},

		{args: nil, Fixture: difftest.Fixture{
			Name: "no-arguments-means-item_tools-and-8", Passes: true,
			Break: func(string) error { return nil }}},
		{args: []string{"", ""}, Fixture: difftest.Fixture{
			Name: "empty-arguments-take-the-defaults", Passes: true,
			Break: func(string) error { return nil }}},

		{args: []string{"item_tools", "3"}, Fixture: difftest.Fixture{
			Name: "a-probe-that-varies-per-process", Finding: true,
			Break: func(sb string) error { return put(sb, ".vary", "") }}},

		{args: []string{"item_tools", "3"}, Fixture: difftest.Fixture{
			Name: "a-run-that-fails", Passes: true,
			Break: func(sb string) error { return put(sb, ".fail-run-2", "") }}},

		{args: []string{"nope", "3"}, Fixture: difftest.Fixture{
			Name: "unknown-probe", Refusal: "unknown probe: nope (one of item_tools",
			Break: func(string) error { return nil }}},

		{args: []string{"item_tools", "lots"}, Fixture: difftest.Fixture{
			Name:    "runs-not-a-number",
			Refusal: "REFUSED: <runs> must be a positive integer, not 'lots'",
			Break:   func(string) error { return nil }}},

		{args: []string{"item_tools", "1"}, Fixture: difftest.Fixture{
			Name: "one-run-has-no-power", Refusal: "REFUSED: <runs> must be at least 2",
			Break: func(string) error { return nil }}},

		{args: []string{"item_tools", "3"}, Fixture: difftest.Fixture{
			Name: "no-pob-checkout", Refusal: "no PoB checkout at pob (set POB_DIR)",
			Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "pob/src")) }}},

		{Fixture: difftest.Fixture{Name: "exit-carries-the-verdict", NoFixture: true,
			Refusal: `exit "$FAILED"`,
			Note: "reached by a-probe-that-varies-per-process, but its text is on STDOUT and " +
				"Spec.RefusalOnStdout is tool-wide; switching it would unmatch five stderr sites"}},

		{Fixture: difftest.Fixture{Name: "the-canary-cannot-discriminate", NoFixture: true,
			Refusal: "REFUSED: the canary produced",
			Note: "guarded by `[ \"${LUAJIT:-}\" = \"\" ]`, and every fixture here sets LUAJIT " +
				"because a fake binary is the only way to control determinism"}},
	}

	table := make([]difftest.Fixture, 0, len(fixtures))
	for _, f := range fixtures {
		table = append(table, f.Fixture)
	}
	spec := pdmSpec(bin, luajit, nil, "item_tools", "3")
	spec.Fixtures = table
	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range fixtures {
		if f.NoFixture {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), pdmSpec(bin, luajit, nil, f.args...), f.Fixture, mk)
		})
	}
}

func TestProbeDeterminismIsStableAcrossLocales(t *testing.T) {
	repo := repoRoot(t)
	mk, luajit := pdmSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "probe-determinism",
		PythonScript: pdmSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   []string{"all", "2"},
		Env:          []string{"GOPOB_NICED=1", "POB_DIR=pob", "LUAJIT=" + luajit},
	}, mk, []string{"C", "en_US.utf8", "C.utf8"})
}
