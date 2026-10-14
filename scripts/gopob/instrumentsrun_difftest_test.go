package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const insSh = "scripts/instruments.sh"

const insGitStub = `#!/bin/sh
case "$1" in
rev-parse)
	case "$2" in
	--show-toplevel)
		if [ -e .stub-bad-toplevel ]; then
			echo "$(pwd)/definitely-not-here"
		else
			pwd
		fi
		;;
	--short) echo "abc1234" ;;
	esac
	;;
status) : ;;
log) echo "abc1234 a pinned subject line" ;;
esac
exit 0
`

const insGoStub = `#!/bin/sh
echo "go $* | OPERATOR=${OPERATOR:-unset} CORPUS=${CORPUS:-unset}" >> .stub-log
[ "$1" = "test" ] || exit 0
shift
run=""
while [ $# -gt 0 ]; do
	case "$1" in
	-run) run="$2"; shift 2 ;;
	*) shift ;;
	esac
done
case "$run" in
"")
	# The bulk step. Five log-shaped lines against a small LOG_SAMPLE, so the scrubber's CAP is
	# exercised rather than just its splitter.
	echo "WRN  a warning that must be split off"
	echo "TRC  a trace that must be split off"
	echo "DBG  a debug line that must be split off"
	echo "INF  an info line that must be split off"
	echo "ERR  an error line that must be split off"
	if [ -e .stub-unit-fails ]; then
		echo "--- FAIL: TestZulu"
		echo "    calc_offence_test.go:12: expected 3 got 4"
		echo "--- FAIL: TestAlpha"
		echo "--- FAIL: TestMike"
		exit 1
	fi
	echo "ok  	module/calculator	0.100s"
	;;
TestCorpus3_29VsOracle)
	echo "  matched 2961296 / 3208353  (92.2996%)   NaN/Inf on our side: 0"
	echo "  5634 graded, 471 excluded (treeSources < 5)"
	;;
TestCorpus3_29\$)
	# ⚠️ THE TWO COUNTS MUST BE ABLE TO DIFFER. With 6105/6105 on every fixture the corpus
	# invariant compares a number with itself, so a port reading the FIRST number twice agrees
	# perfectly -- measured, that mutation went UNCAUGHT until this branch existed. A control can
	# be right in every respect except its input.
	if [ -e .stub-corpus-panics ]; then
		echo "  6104/6105 built, 0 parse failure(s), 1 panic(s)"
	else
		echo "  6105/6105 built, 0 parse failure(s), 0 panic(s)"
	fi
	;;
TestModDBCensus)
	echo "LEVEL 2: PER-MOD-NAME CENSUS (10 builds, 3 mod names diverge)"
	echo "LEVEL 2: ENEMY PER-MOD-NAME CENSUS (10 builds, 1 mod names diverge)"
	;;
TestModOrderCensus)
	echo "  CROSS-STAGE     17 pairs"
	;;
TestTagCensus)
	echo "PLAYER TAG CENSUS (10 builds)"
	echo "    TAGS-DIFFER    4"
	echo "ENEMY TAG CENSUS (10 builds)"
	echo "    TAGS-DIFFER    0"
	;;
TestConfigCensus)
	echo "TOTAL      12      3      4"
	;;
TestCorpusIsDeterministic)
	echo "run1     8 builds"
	echo "run2     8 builds"
	if [ -e .stub-determinism-diverges ]; then
		echo "MATERIAL (rel > 1e-9): 5 builds"
		echo "FLOAT REASSOCIATION ONLY: 2 builds"
	else
		echo "IDENTICAL: every build agrees"
	fi
	;;
esac
exit 0
`

const insTaskStub = `#!/bin/sh
echo "task $*" >> .stub-log
case "$1" in
citations)
	# ⚠️ THE REAL SHAPE, COPIED FROM A REAL RUN. This stub used to print a TOTAL line, which is
	# what the chain's pattern looked for and what no producer emits -- so the fixture validated
	# the pattern against itself and agreed forever while the live chain printed
	# "(no headline matched)" for every citations step. A fixture written from the CONSUMER is not
	# data; it is the consumer's assumption wearing data's clothes.
	echo "  PORT   349 / 455 top-level funcs cited (76.7%) across 37 file(s)"
	echo "  0 broken"
	;;
omissions)      echo "OMISSION REASONS: 42 checked, 0 unmarked" ;;
task-test-refs) echo "every -run selector resolves" ;;
test-cache-key) echo "test cache key OK" ;;
esac
exit 0
`

const insStubGateVersions = `# shellcheck shell=bash
if [ -e .stub-source-fails ]; then
	echo "REFUSING: gopob gate-versions --sh failed; the version table is unreadable." >&2
	return 1 2>/dev/null || exit 1
fi
gv_field() { cut -d'|' -f"$2" <<<"$1"; }
gv_all_tests() {
	if [ -e .stub-no-alltests ]; then
		echo "REFUSING: the version table names no tests." >&2
		return 3
	fi
	echo "TestCorpus3_29VsOracle|TestCorpus3_29"
}
authoritative_row() {
	if [ -e .stub-no-table ]; then
		echo "REFUSING: no authoritative row in the version table." >&2
		return 3
	fi
	echo "3.29|TestCorpus3_29VsOracle|TestCorpus3_29|1"
}
`

const insDataRawMain = `package raw

const DefaultVersion = "3.29"
`

func insSandbox(t *testing.T, repo string) (func() (string, error), string) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, insSh)
	base := t.TempDir()
	binDir := filepath.Join(base, "bin")
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"git": insGitStub, "go": insGoStub, "task": insTaskStub,
	} {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte(body), 0o700); err != nil { //nolint:gosec
			t.Fatal(err)
		}
	}
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			insSh:                      string(sh),
			"scripts/gate_versions.sh": insStubGateVersions,
			"data/raw/main.go":         insDataRawMain,
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
		if err := os.MkdirAll(filepath.Join(sb, "dumps"), 0o750); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		return sb, nil
	}, binDir
}

func insSpec(bin, binDir string) difftest.Spec {
	return difftest.Spec{
		Tool:         "instruments",
		PythonScript: insSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		GoBin:        bin,
		GoArgs:       []string{"instruments"},
		Stderr:       difftest.StderrExact,
		Env: []string{
			"GOPOB_NICED=1",
			"INSTRUMENTS_UNDER_MISE=1",
			"INSTRUMENT_TIMINGS=0",
			"RUNDIR=out",
			"DUMPS=dumps",
			"LOG_SAMPLE=3",
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		},
		EnvUnset:     []string{"POE_VERSION", "ONLY", "COVERAGE"},
		RefusalSites: 7,
	}
}

func TestInstrumentsPortIsIndistinguishable(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := insSandbox(t, repo)

	shSrc := difftest.ReadOriginal(t, repo, insSh)

	spec := insSpec(bin, binDir)
	spec.Fixtures = []difftest.Fixture{
		{Name: "happy"},

		{Name: "the-repository-root-cannot-be-entered",
			Refusal: "REFUSING: could not enter the repository root (",
			Break:   func(sb string) error { return put(sb, ".stub-bad-toplevel", "") }},

		{Name: "the-version-table-cannot-be-sourced",
			Refusal: "REFUSING: scripts/gate_versions.sh could not be sourced;",
			Break:   func(sb string) error { return put(sb, ".stub-source-fails", "") }},

		{Name: "DefaultVersion-cannot-be-read",
			Refusal: "REFUSING: could not read DefaultVersion from data/raw/main.go",
			Break: func(sb string) error {
				return put(sb, "data/raw/main.go", "package raw\n\n// the constant moved\n")
			}},

		{Name: "the-run-directory-cannot-be-created",
			Refusal: "REFUSING: could not create the run directory (out).",
			Break:   func(sb string) error { return put(sb, "out", "a file where the directory goes") }},

		{Name: "the-chain-test-list-is-unreadable",
			Refusal: "REFUSING: the version table names no tests.",
			Break:   func(sb string) error { return put(sb, ".stub-no-alltests", "") }},

		{Name: "there-is-no-authoritative-row",
			Refusal: "REFUSING: no authoritative row in the version table.",
			Break:   func(sb string) error { return put(sb, ".stub-no-table", "") }},

		{Name: "exit-carries-the-verdict", NoFixture: true,
			Refusal: "exit 1",
			Note: "the chain's own verdict for FAILURES / NOT-RUN / a BAD invariant; it prints no " +
				"refusal line, and is reached by TestInstrumentsReportsNotRunWhenTheDumpsAreAbsent " +
				"and TestInstrumentsReportsAFailingInstrument as a Finding"},
	}

	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range spec.Fixtures {
		if f.NoFixture {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), spec, f, mk)
		})
	}
}

// TestInstrumentsReportsNotRunWhenTheDumpsAreAbsent: NOT-RUN IS THE CLASS THIS WHOLE CHAIN WAS BUILT AROUND.
func TestInstrumentsReportsNotRunWhenTheDumpsAreAbsent(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := insSandbox(t, repo)

	difftest.RunFixture(t, t.Context(), insSpec(bin, binDir),
		difftest.Fixture{Name: "the-dumps-directory-is-absent", Finding: true,
			Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "dumps")) }}, mk)
}

// TestInstrumentsReportsAFailingInstrument: A failing instrument must reach the FAILURES section with its assertion text.
func TestInstrumentsReportsAFailingInstrument(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := insSandbox(t, repo)

	difftest.RunFixture(t, t.Context(), insSpec(bin, binDir),
		difftest.Fixture{Name: "the-bulk-step-fails", Finding: true,
			Break: func(sb string) error { return put(sb, ".stub-unit-fails", "") }}, mk)
}

// TestInstrumentsRendersTheDivergentDeterminismHeadline: THE DETERMINISM HEADLINE HAS TWO SHAPES AND THE HAPPY PATH ONLY EVER SHOWS ONE.
func TestInstrumentsRendersTheDivergentDeterminismHeadline(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := insSandbox(t, repo)

	difftest.RunFixture(t, t.Context(), insSpec(bin, binDir),
		difftest.Fixture{Name: "determinism-diverges", Passes: true,
			Break: func(sb string) error { return put(sb, ".stub-determinism-diverges", "") }}, mk)
}

// TestInstrumentsWritesTheSameReportsAndDrivesTheSameChildren: THE FILES IN $RUNDIR ARE THE PRODUCT; the consolidated table is the label.
func TestInstrumentsWritesTheSameReportsAndDrivesTheSameChildren(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := insSandbox(t, repo)

	spec := insSpec(bin, binDir)
	spec.Files = []string{
		".stub-log",
		"out/unit.txt", "out/unit.logs.txt",
		"out/oracle.txt", "out/corpus.txt",
		"out/moddb.txt", "out/modorder.txt", "out/tags.txt", "out/config.txt",
		"out/determinism.txt",
		"out/citations.txt", "out/omissions.txt", "out/taskrefs.txt", "out/cachekey.txt",
	}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "every-report-and-every-child-invocation", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

// TestInstrumentsHonoursTheOnlyFilter: ONLY= SELECTS, AND AN UNSELECTED INSTRUMENT MUST LEAVE NO ROW AT ALL — not a row reading SKIP.
func TestInstrumentsHonoursTheOnlyFilter(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := insSandbox(t, repo)

	spec := insSpec(bin, binDir)
	spec.Env = append(spec.Env, "ONLY=oracle configcensus")
	spec.EnvUnset = []string{"POE_VERSION", "COVERAGE"}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "only-two-instruments", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

// TestInstrumentsFailsTheCorpusInvariantWhenABuildPanics: THE CORPUS INVARIANT'S WHOLE JOB IS COMPARING TWO COUNTS.
func TestInstrumentsFailsTheCorpusInvariantWhenABuildPanics(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := insSandbox(t, repo)

	difftest.RunFixture(t, t.Context(), insSpec(bin, binDir),
		difftest.Fixture{Name: "a-corpus-build-panicked", Finding: true,
			Break: func(sb string) error { return put(sb, ".stub-corpus-panics", "") }}, mk)
}

// TestInstrumentsExpandsATildeInDumps: A LEADING ~/ IS EXPANDED BY THE SCRIPT.
func TestInstrumentsExpandsATildeInDumps(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := insSandbox(t, repo)

	spec := insSpec(bin, binDir)
	spec.Env = insReplaceEnv(spec.Env, "DUMPS", "~/dumps")
	spec.Env = insReplaceEnv(spec.Env, "HOME", ".")
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "dumps-is-tilde-relative", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

func insReplaceEnv(env []string, key, val string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, key+"=") {
			out = append(out, e)
		}
	}
	return append(out, key+"="+val)
}

// TestInstrumentsAddsTheCoverageStepWhenAskedTo: COVERAGE=1 adds a step AND removes the "coverage skipped" line from the header.
func TestInstrumentsAddsTheCoverageStepWhenAskedTo(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk, binDir := insSandbox(t, repo)

	spec := insSpec(bin, binDir)
	spec.Env = append(spec.Env, "COVERAGE=1")
	spec.EnvUnset = []string{"POE_VERSION", "ONLY"}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "coverage-is-opted-in", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

func TestInstrumentsIsStableAcrossLocales(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	mk, binDir := insSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "instruments",
		PythonScript: insSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		Env: []string{
			"GOPOB_NICED=1", "INSTRUMENTS_UNDER_MISE=1", "INSTRUMENT_TIMINGS=0",
			"RUNDIR=out", "DUMPS=dumps", "LOG_SAMPLE=3",
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		},
	}, mk, []string{"C", "en_US.utf8", "C.utf8"})
}

// TestInsMMSSMatchesThePrintfFormat: THE ONE PIECE OF ARITHMETIC THE DIFFERENTIAL CANNOT SEE, table-tested instead.
func TestInsMMSSMatchesThePrintfFormat(t *testing.T) {
	for _, c := range []struct {
		secs int
		want string
	}{
		{0, "0m00s"},
		{9, "0m09s"},
		{59, "0m59s"},
		{60, "1m00s"},
		{61, "1m01s"},
		{599, "9m59s"},
		{600, "10m00s"},
		{3661, "61m01s"},
	} {
		if got := insMMSS(c.secs); got != c.want {
			t.Errorf("insMMSS(%d) = %q, want %q", c.secs, got, c.want)
		}
	}
}
