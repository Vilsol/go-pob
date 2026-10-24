package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const rmPy = "scripts/residual_map.py"

func rmWorklist() (string, string) {
	var sb strings.Builder
	sb.WriteString("stat\tseen\tok\tmiss\tnan\tclass\n")
	row := func(stat string, seen, ok int) {
		fmt.Fprintf(&sb, "%s\t%d\t%d\t%d\t0\tpartial\n", stat, seen, ok, seen-ok)
	}
	row("TotalEHP", 100, 10)
	row("Life", 100, 40)
	row("FireResist", 100, 70)
	row("Minion.TotalDPS", 100, 55)
	row("Minion.Life", 100, 65)
	row("MainHand.CritChance", 100, 55)
	row("IgniteDPS", 100, 55)
	row("SomethingNobodyClassified", 100, 80)
	row("AlsoUnclassifiable", 100, 95)
	row("PerfectStat", 100, 100)
	stats := sb.String()

	var bb strings.Builder
	bb.WriteString("build\twrong\ttotal\ttopstats\n")
	for i := 1; i <= 12; i++ {
		fmt.Fprintf(&bb, "%04d-build.xml\t%d\t%d\tSomeStat\n", i, i*3, 100)
	}
	bb.WriteString("0013-empty.xml\t0\t0\tSomeStat\n")
	return stats, bb.String()
}

func rmSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, rmPy)
	stats, builds := rmWorklist()
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		for rel, body := range map[string]string{
			rmPy: string(py), "wl/stats.tsv": stats, "wl/builds.tsv": builds,
		} {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func TestResidualMapPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := rmSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, rmPy)

	spec := difftest.Spec{
		Tool:         "residual-map",
		PythonScript: rmPy,
		GoBin:        bin,
		GoArgs:       []string{"residual-map"},
		Env:          []string{"WL=wl"},
		Stderr:       difftest.StderrExact,
		RefusalSites: 9,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-wl", Refusal: "REFUSING: set WL=<dir> to a worklist directory",
				Break: func(string) error { return nil }},

			{Name: "stats-missing", Refusal: "REFUSING: wl/stats.tsv does not exist",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "wl/stats.tsv")) }},

			{Name: "builds-unreadable", Refusal: "REFUSING: wl/builds.tsv is not readable:",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "wl/builds.tsv"), 0o000)
				}},

			{Name: "builds-empty", Refusal: "REFUSING: wl/builds.tsv is empty;",
				Break: func(sb string) error { return put(sb, "wl/builds.tsv", "") }},

			{Name: "builds-wrong-columns",
				Refusal: "REFUSING: builds.tsv columns are ['name', 'bad', 'cols'], expected build/wrong/total.",
				Break: func(sb string) error {
					return put(sb, "wl/builds.tsv", "name\tbad\tcols\tx\n0001.xml\t1\t2\tz\n")
				}},

			{Name: "non-integer-cell", Refusal: "REFUSING: wl/builds.tsv line 2: 'lots' is not an integer.",
				Break: func(sb string) error {
					return put(sb, "wl/builds.tsv", "build\twrong\ttotal\tx\n0001.xml\t1\tlots\tz\n")
				}},

			{Name: "zero-graded-pairs", Refusal: "REFUSING: builds.tsv reports zero graded pairs.",
				Break: func(sb string) error {
					return put(sb, "wl/builds.tsv", "build\twrong\ttotal\tx\n0001.xml\t0\t0\tz\n")
				}},

			{Name: "stats-bad-header",
				Refusal: "does not look like a worklist stats.tsv (header: 'nope\\tseen\\tok\\n')",
				Break: func(sb string) error {
					return put(sb, "wl/stats.tsv", "nope\tseen\tok\nA\t1\t0\n")
				}},

			{Name: "zero-wrong-pairs", Refusal: "REFUSING: zero wrong pairs across the whole table.",
				Break: func(sb string) error {
					return put(sb, "wl/stats.tsv", "stat\tseen\tok\tmiss\tnan\tclass\nA\t10\t10\t0\t0\tok\n")
				}},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "builds-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			if f.Name == "no-wl" {
				s.Env = []string{"WL=  "}
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

func TestResidualMapPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := rmSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "residual-map",
		PythonScript: rmPy,
		Env:          []string{"WL=wl"},
	}, mk, []string{"0", "12345", "999983"})
}
