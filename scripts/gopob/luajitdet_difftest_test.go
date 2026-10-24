package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const ljSh = "scripts/build_luajit_deterministic.sh"

const ljFakeDet = `#!/bin/sh
[ "$1" = "-v" ] && { echo "LuaJIT 2.1.1744318430 -- Copyright (C) 2005-2025"; echo "second line"; exit 0; }
# ⚠️ IT REFUSES TO RUN WITHOUT ADDR_NO_RANDOMIZE, which is what setarch -R sets and what
# /proc/self/personality reports (00040000 with, 00000000 without -- measured). Nothing else in the
# fixture can see the flag: a fake binary is unaffected by ASLR, its output is constant either way,
# and the tool prints only the distinct COUNT. Encoding "setarch -R is required" as an executable
# fact is what turns a dropped -R from an uncaught mutation into a zero-order refusal.
case "$(cat /proc/self/personality 2>/dev/null)" in
*40000) ;;
*) echo "fake luajit: ASLR is on; setarch -R was not applied" >&2; exit 9 ;;
esac
echo "alpha,beta,gamma|1,2,3"
exit 0
`

const ljFakeStock = `#!/bin/sh
[ "$1" = "-v" ] && { echo "LuaJIT 2.1.0-beta3 -- Copyright (C) 2005-2025"; exit 0; }
echo "order-$$"
exit 0
`

func ljSandbox(t *testing.T, repo string, det, stock string) (func() (string, error), string) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, ljSh)
	base := t.TempDir()
	binDir := filepath.Join(base, "bin")
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "luajit"), []byte(stock), 0o700); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		if err := os.MkdirAll(filepath.Join(sb, "scripts"), 0o750); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		if err := os.WriteFile(filepath.Join(sb, ljSh), sh, 0o700); err != nil { //nolint:gosec
			return "", fmt.Errorf("sandbox: %w", err)
		}
		if err := os.MkdirAll(filepath.Join(sb, ".luajit-det/bin"), 0o750); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		if det != "" {
			if err := os.WriteFile(filepath.Join(sb, ".luajit-det/bin/luajit"), []byte(det), 0o700); err != nil { //nolint:gosec
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}, binDir
}

func ljSpec(bin, binDir string) difftest.Spec {
	return difftest.Spec{
		Tool:         "build-luajit-det",
		PythonScript: ljSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   []string{"--verify"},
		GoBin:        bin,
		GoArgs:       []string{"build-luajit-det", "--verify"},
		Stderr:       difftest.StderrExact,
		Env: []string{
			"GOPOB_NICED=1",
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		},
		RefusalSites: 4,
	}
}

// TestBuildLuajitDetPortIsIndistinguishable: ONLY `--verify` IS EXERCISED, AND THE SPEC SAYS SO.
func TestBuildLuajitDetPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	requireWorks(t, probeSetarch)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := ljSandbox(t, repo, ljFakeDet, ljFakeStock)

	shSrc := difftest.ReadOriginal(t, repo, ljSh)

	spec := ljSpec(bin, binDir)
	spec.Fixtures = []difftest.Fixture{
		{Name: "happy"},

		{Name: "no-binary-built-yet", Refusal: "FATAL: no binary at ",
			Break: func(sb string) error {
				return os.Remove(filepath.Join(sb, ".luajit-det/bin/luajit"))
			}},

		{Name: "the-binary-is-not-executable", Refusal: "FATAL: no binary at ",
			Break: func(sb string) error {
				return os.Chmod(filepath.Join(sb, ".luajit-det/bin/luajit"), 0o600)
			}},

		{Name: "the-wrong-luajit-was-built", Refusal: "FATAL: built LuaJIT 2.1.9999999999, expected",
			Break: func(sb string) error {
				return put(sb, ".luajit-det/bin/luajit",
					"#!/bin/sh\n[ \"$1\" = \"-v\" ] && { echo \"LuaJIT 2.1.9999999999 -- x\"; exit 0; }\n"+
						"echo 'a|1'\nexit 0\n")
			}},

		{Name: "the-build-is-not-deterministic",
			Refusal: "distinct iteration orders across 20 processes — NOT deterministic.",
			Break: func(sb string) error {
				return put(sb, ".luajit-det/bin/luajit",
					"#!/bin/sh\n[ \"$1\" = \"-v\" ] && { echo \"LuaJIT 2.1.1744318430 -- x\"; exit 0; }\n"+
						"echo \"order-$$\"\nexit 0\n")
			}},

		{Name: "the-build-fails", NoFixture: true,
			Refusal: "FATAL: build failed, see",
			Note: "the build path clones LuaJIT from github and compiles it with mise's gcc; a " +
				"sandbox cannot reach it, and --verify is what every fixture here exercises"},
	}

	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range spec.Fixtures {
		if f.NoFixture {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "the-binary-is-not-executable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-600 file is still executable")
			}
			difftest.RunFixture(t, t.Context(), spec, f, mk)
		})
	}
}

// TestBuildLuajitDetWarnsWhenTheProbeStopsDiscriminating: THE NEGATIVE CONTROL HAS TWO OUTCOMES AND THE SECOND IS A WARNING, NOT A FAILURE.
func TestBuildLuajitDetWarnsWhenTheProbeStopsDiscriminating(t *testing.T) {
	t.Parallel()
	requireWorks(t, probeSetarch)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := ljSandbox(t, repo, ljFakeDet,
		"#!/bin/sh\n[ \"$1\" = \"-v\" ] && { echo \"LuaJIT 2.1.0-beta3 -- x\"; exit 0; }\n"+
			"echo 'alpha,beta,gamma|1,2,3'\nexit 0\n")

	difftest.RunFixture(t, t.Context(), ljSpec(bin, binDir),
		difftest.Fixture{Name: "the-stock-binary-is-also-deterministic", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

// TestBuildLuajitDetSkipsTheControlWithNoStockLuajit: AND THE CONTROL IS SKIPPED ENTIRELY WHEN THERE IS NO STOCK luajit ON PATH.
func TestBuildLuajitDetSkipsTheControlWithNoStockLuajit(t *testing.T) {
	t.Parallel()
	requireWorks(t, probeSetarch)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, _ := ljSandbox(t, repo, ljFakeDet, ljFakeStock)

	spec := ljSpec(bin, "")
	spec.Env = []string{"GOPOB_NICED=1", "PATH=" + ljPathWithoutLuajit()}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "no-stock-luajit-on-path", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

func ljPathWithoutLuajit() string {
	var keep []string
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		if fi, err := os.Stat(filepath.Join(dir, "luajit")); err == nil && !fi.IsDir() {
			continue
		}
		keep = append(keep, dir)
	}
	return strings.Join(keep, string(os.PathListSeparator))
}
