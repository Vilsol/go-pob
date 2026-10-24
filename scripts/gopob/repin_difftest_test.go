package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const rpPy = "scripts/repin.py"

const rpGoStub = `#!/bin/sh
{ echo "go-stub: args=$*"; echo "go-stub: CORPUS=${CORPUS:-unset}"; } >> .stub-log
[ -e .stub-no-findings ] && { echo "ok  	module/calculator	0.1s"; exit 0; }
printf '=== RUN   TestPins\n'
if [ -e .stub-closed ]; then
	printf '    \033[31mKNOWN DIVERGENCE b0002.xml Armour now MATCHES the pinned PoB (1234.5)\033[0m\n'
fi
# ⚠️ THE COLOUR WRAPS THE NUMBER ITSELF, NOT THE WHOLE LINE, AND THAT IS WHAT MAKES THE ANSI STRIP
# LOAD-BEARING. With the escapes only at the ends the value regexes still match and a port that
# never stripped agreed on everything -- measured. An escape BETWEEN "go-pob's " and the digits
# breaks the pattern outright, which is what real coloured test output does to a value.
printf '    KNOWN DIVERGENCE b0001.xml Life = 4321.5, pinned at go-pob'"'"'s \033[1m1111.25\033[0m\n'
printf '    KNOWN DIVERGENCE b0003.xml EnergyShield = -7.5e+02, pinned at go-pob'"'"'s 250\n'
echo "FAIL	module/calculator	0.2s"
exit 1
`

const rpPinFile = `package calculator

import "testing"

func TestPinnedLife(t *testing.T) {
	assertDivergesFromOracle(t, "b0001.xml", "Life", 1111.25, "life is low")
}

func TestNotPinned(t *testing.T) {
	assertMatchesOracle(t, "b0004.xml", "Mana")
}

func TestPinnedOthers(t *testing.T) {
	assertDivergesFromOracleDump(t, "b0003.xml", "EnergyShield", 250, "es is high")
	assertDivergesFromOracle(t, "b0002.xml", "Armour", 99.5, "armour")
}

// ⚠️ FOUR PINNED FUNCTIONS, NOT TWO, BECAUSE THE -run REGEX ORDER IS A MAP-ORDER CONTROL AND A
// MAP-ORDER CONTROL IS FLAKY. With two names the two implementations coincide half the time, so a
// mutation dropping the sort passed as often as it failed; with four the chance of agreement by
// luck is one in twenty-four. the rule for these: run repeatedly and print the count.
func TestPinnedEvasion(t *testing.T) {
	assertDivergesFromOracle(t, "b0005.xml", "Evasion", 12.5, "evasion")
}

func TestPinnedWard(t *testing.T) {
	assertDivergesFromOracle(t, "b0006.xml", "Ward", 7, "ward")
}
`

func rpSandbox(t *testing.T, repo string) (func() (string, error), string) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, rpPy)
	base := t.TempDir()
	binDir := filepath.Join(base, "bin")
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(rpGoStub), 0o700); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			rpPy:                          string(py),
			"calculator/pins_test.go":     rpPinFile,
			"calculator/nopins_test.go":   "package calculator\n\n// no pins here\n",
			"calculator/sub/more_test.go": rpPinFile,
		}
		for rel, b := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(b), 0o600); err != nil {
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
		return sb, nil
	}, binDir
}

func rpSpec(bin, binDir string, env ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "repin",
		PythonScript: rpPy,
		GoBin:        bin,
		GoArgs:       []string{"repin"},
		Stderr:       difftest.StderrExact,
		Env: append([]string{
			"GOPOB_NICED=1",
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		}, env...),
		RefusalSites: 2,
	}
}

func TestRepinPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := rpSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, rpPy)

	type fixture struct {
		difftest.Fixture
		env []string
	}
	fixtures := []fixture{
		{Fixture: difftest.Fixture{Name: "happy"}},

		{env: []string{"CHECK=1"}, Fixture: difftest.Fixture{
			Name: "check-reports-without-writing", Passes: true,
			Break: func(string) error { return nil }}},
		{env: []string{"CHECK=false"}, Fixture: difftest.Fixture{
			Name: "check-false-is-OFF", Passes: true,
			Break: func(string) error { return nil }}},
		{env: []string{"CHECK=0"}, Fixture: difftest.Fixture{
			Name: "check-zero-is-OFF", Passes: true,
			Break: func(string) error { return nil }}},

		{Fixture: difftest.Fixture{Name: "no-divergence-pins-anywhere", Passes: true,
			Break: func(sb string) error {
				for _, p := range []string{"calculator/pins_test.go", "calculator/sub/more_test.go"} {
					if err := put(sb, p, "package calculator\n"); err != nil {
						return err
					}
				}
				return nil
			}}},

		{Fixture: difftest.Fixture{Name: "every-pin-is-on-its-measured-value", Passes: true,
			Break: func(sb string) error { return put(sb, ".stub-no-findings", "") }}},

		{Fixture: difftest.Fixture{Name: "a-closed-pin-is-reported-not-rewritten", Passes: true,
			Break: func(sb string) error { return put(sb, ".stub-closed", "") }}},

		{Fixture: difftest.Fixture{Name: "sys-exit-main", NoFixture: true,
			Refusal: "sys.exit(main())",
			Note: "main() returns 0 on every path; the only non-zero exits are the git guard above " +
				"and a traceback, and the counter is lexical"}},

		{Fixture: difftest.Fixture{Name: "not-inside-a-git-checkout",
			Refusal: "REFUSING: not inside a git checkout",
			Break:   func(sb string) error { return os.RemoveAll(filepath.Join(sb, ".git")) }}},
	}

	table := make([]difftest.Fixture, 0, len(fixtures))
	for _, f := range fixtures {
		table = append(table, f.Fixture)
	}
	spec := rpSpec(bin, binDir)
	spec.Fixtures = table
	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range fixtures {
		if f.NoFixture {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), rpSpec(bin, binDir, f.env...), f.Fixture, mk)
		})
	}
}

// TestRepinRewritesTheRightLiterals: THE REWRITTEN SOURCE IS THE PRODUCT.
func TestRepinRewritesTheRightLiterals(t *testing.T) {
	t.Parallel()
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := rpSandbox(t, repo)

	spec := rpSpec(bin, binDir)
	spec.Files = []string{
		"calculator/pins_test.go",
		"calculator/sub/more_test.go",
		".stub-log",
	}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "products-are-identical", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

func TestRepinIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk, binDir := rpSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "repin",
		PythonScript: rpPy,
		Env: []string{
			"GOPOB_NICED=1",
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		},
	}, mk, []string{"0", "12345", "999983"})
}
