package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const tckSh = "scripts/test_cache_key.sh"

const tckGoStub = `#!/bin/sh
case "$1" in
clean)
	[ -e .stub-clean-fails ] && exit 1
	exit 0
	;;
test)
	n=$(cat .stub-calls 2>/dev/null || echo 0)
	n=$((n + 1))
	echo "$n" > .stub-calls
	# ⚠️ A LOG FILE, NOT stderr. The tool pipes each probe through "2>&1 | tail -1", so ANYTHING this
	# stub writes to either stream is captured and then DISCARDED -- only the last line is the verdict.
	# (No backticks anywhere in here: this is a Go RAW STRING and one ends it. Third time today.)
	# Three mutations went UNCAUGHT that way: gv_field splitting on whitespace instead of pipes, and
	# POE_VERSION and CORPUS not being passed at all. What the tool hands its children is a PRODUCT,
	# and a product has to be declared to be compared.
	echo "go-stub: run $n POE_VERSION=${POE_VERSION:-unset} CORPUS=${CORPUS:-unset} args=$*" >> .stub-log
	# ⚠️ BUILD CHATTER BEFORE THE VERDICT, because the tool keeps only the LAST line and with a
	# single-line stub first and last are the same -- a port reading lines[0] passed everything.
	# Real "go test" prints compile output above its verdict, so this is the shape, not a prop.
	echo "# module/calculator"
	if [ -e .stub-never-caches ]; then
		echo "ok  	module/calculator	0.100s"
		exit 0
	fi
	if [ -e .stub-always-caches ]; then
		echo "ok  	module/calculator	(cached)"
		exit 0
	fi
	if [ -e .stub-cold-is-junk ] && [ "$n" = 1 ]; then
		echo "FAIL	module/calculator [build failed]"
		exit 2
	fi
	# The honest machine: run 1 cold, run 2 cached, run 3 (a different POE_VERSION) cold again.
	if [ "$n" = 2 ] && [ -z "${POE_VERSION:-}" ]; then
		echo "ok  	module/calculator	(cached)"
	elif [ "$n" = 3 ] && [ -e .stub-version-not-in-key ]; then
		echo "ok  	module/calculator	(cached)"
	else
		echo "ok  	module/calculator	0.100s"
	fi
	exit 0
	;;
esac
exit 0
`

const tckStubGateVersions = `# shellcheck shell=bash
gv_field() { cut -d'|' -f"$2" <<<"$1"; }
authoritative_row() {
	# ⚠️ THE REAL SHIM PRINTS BEFORE IT FAILS, and the stub must too. scripts/gate_versions.sh
	# refuses with "REFUSING: gopob gate-versions --sh failed; the version table is unreadable."
	# and returns 1 -- a silent failure here would leave the site with NO TEXT to match, and a
	# fixture that can only assert "exit 3" is one that a DIFFERENT exit-3 path satisfies too.
	# (No backticks: this is a Go RAW STRING and one would end it. Second time this session.)
	if [ -e .stub-no-table ]; then
		echo "REFUSING: the version table is unreadable." >&2
		return 3
	fi
	echo "3.29|TestCorpus3_29VsOracle|TestCorpus3_29|1"
}
`

func tckSandbox(t *testing.T, repo string) (func() (string, error), string) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, tckSh)
	base := t.TempDir()
	binDir := filepath.Join(base, "bin")
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(tckGoStub), 0o700); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			tckSh:                      string(sh),
			"scripts/gate_versions.sh": tckStubGateVersions,
		}
		for rel, b := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(b), 0o700); err != nil { //nolint:gosec
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}, binDir
}

func tckSpec(bin, binDir string) difftest.Spec {
	return difftest.Spec{
		Tool:         "test-cache-key",
		PythonScript: tckSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		GoBin:        bin,
		GoArgs:       []string{"test-cache-key"},
		Stderr:       difftest.StderrExact,
		Env: []string{
			"GOPOB_NICED=1",
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		},
		RefusalSites: 3,
	}
}

func TestTestCacheKeyPortIsIndistinguishable(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := tckSandbox(t, repo)

	shSrc := difftest.ReadOriginal(t, repo, tckSh)

	spec := tckSpec(bin, binDir)
	spec.Fixtures = []difftest.Fixture{
		{Name: "happy"},

		{Name: "the-version-table-refuses", Refusal: "REFUSING: the version table is unreadable.",
			Break: func(sb string) error { return put(sb, ".stub-no-table", "") }},

		{Name: "the-test-cache-cannot-be-cleared", Refusal: "FAIL: could not clear the test cache",
			Break: func(sb string) error { return put(sb, ".stub-clean-fails", "") }},

		{Name: "the-cold-run-is-neither", Refusal: "REFUSING: the cold run neither passed nor cached",
			Break: func(sb string) error { return put(sb, ".stub-cold-is-junk", "") }},

		{Name: "caching-is-off-entirely", Refusal: "an identical re-run was NOT cached",
			Break: func(sb string) error { return put(sb, ".stub-never-caches", "") }},

		{Name: "POE_VERSION-is-not-in-the-cache-key",
			Refusal: "the 3.18 run was served from the 3.20 cache entry",
			Break:   func(sb string) error { return put(sb, ".stub-version-not-in-key", "") }},

		{Name: "a-cleared-cache-still-serves-a-hit",
			Refusal: "a freshly cleared cache still served a cached result",
			Break:   func(sb string) error { return put(sb, ".stub-always-caches", "") }},
	}

	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), spec, f, mk)
		})
	}
}

func TestTestCacheKeyIsStableAcrossLocales(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	mk, binDir := tckSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "test-cache-key",
		PythonScript: tckSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		Env: []string{
			"GOPOB_NICED=1",
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		},
	}, mk, []string{"C", "en_US.utf8", "C.utf8"})
}

// TestTestCacheKeyPassesTheRightArgumentsToEachRun: WHAT THE TOOL HANDS EACH RUN IS THE PRODUCT; the one summary line is the label.
func TestTestCacheKeyPassesTheRightArgumentsToEachRun(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := tckSandbox(t, repo)

	spec := tckSpec(bin, binDir)
	spec.Files = []string{".stub-log"}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "three-runs-with-the-right-arguments", Passes: true,
			Break: func(string) error { return nil }}, mk)
}
