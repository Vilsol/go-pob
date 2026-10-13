package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const rcPy = "scripts/residual_clusters.py"

func rcMatrix() string {
	var b strings.Builder
	b.WriteString("build\tstat\tclass\tpob\tours\n")
	row := func(build, stat, cls, pob, ours string) {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\n", build, stat, cls, pob, ours)
	}
	for i := range 40 {
		bd := fmt.Sprintf("b%03d.xml", i)
		row(bd, "AlphaOne", "wrong", "100", "130")
		row(bd, "AlphaTwo", "wrong", "100", "130")
	}
	for i := range 37 {
		row(fmt.Sprintf("b%03d.xml", i), "Beta", "wrong", "50", "55")
	}
	for i := 40; i < 43; i++ {
		row(fmt.Sprintf("b%03d.xml", i), "Beta", "wrong", "50", "55")
	}
	for i := 100; i < 200; i++ {
		bd := fmt.Sprintf("b%03d.xml", i)
		row(bd, "Gamma", "wrong", "8", fmt.Sprintf("%d", 3+i%7))
	}
	for i := 250; i < 285; i++ {
		row(fmt.Sprintf("b%03d.xml", i), "Delta", "miss", "12", "0")
	}
	for i := 300; i < 330; i++ {
		row(fmt.Sprintf("b%03d.xml", i), "Epsilon", "wrong", "0", "5")
	}
	row("b999.xml", "Epsilon", "extra", "0", "1")
	for i := 400; i < 415; i++ {
		row(fmt.Sprintf("b%03d.xml", i), "Rounder", "wrong", "3", "1")
	}
	for i := 415; i < 430; i++ {
		row(fmt.Sprintf("b%03d.xml", i), "Rounder", "wrong", "100000", "33333")
	}
	for c := range 15 {
		for i := range 12 {
			row(fmt.Sprintf("b%d%03d.xml", 5+c, i), fmt.Sprintf("Tied%02d", c), "wrong", "10", "11")
		}
	}
	for i := range 3 {
		row(fmt.Sprintf("b9%02d.xml", i), "Tiny", "wrong", "4", "5")
	}
	return b.String()
}

func rcSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, rcPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		for rel, body := range map[string]string{
			rcPy: string(py), "matrix.tsv": rcMatrix(),
			"badheader.tsv": "a\tb\tc\td\te\n1\t2\t3\t4\t5\n",
			"empty.tsv":     "build\tstat\tclass\tpob\tours\n",
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

func TestResidualClustersPortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := rcSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, rcPy)

	spec := difftest.Spec{
		Tool:         "residual-clusters",
		PythonScript: rcPy,
		PythonArgs:   []string{"10"},
		GoBin:        bin,
		GoArgs:       []string{"residual-clusters", "10"},
		Env:          []string{"MATRIX=matrix.tsv"},
		Stderr:       difftest.StderrExact,
		RefusalSites: 6,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "min-builds-not-an-integer",
				Refusal: "REFUSING: MIN_BUILDS must be a non-negative integer, not abc",
				Break:   func(string) error { return nil }},
			{Name: "jaccard-not-a-number", Refusal: "REFUSING: JACCARD must be a number, not xyz",
				Break: func(string) error { return nil }},
			{Name: "no-matrix", Refusal: "REFUSING: no matrix at matrix.tsv. Generate it first:",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "matrix.tsv")) }},
			{Name: "matrix-unreadable", Refusal: "REFUSING: matrix.tsv is not readable:",
				Break: func(sb string) error { return os.Chmod(filepath.Join(sb, "matrix.tsv"), 0o000) }},
			{Name: "bad-header", Refusal: "does not look like a residual matrix (header",
				Break: func(string) error { return nil }},
			{Name: "empty-matrix", Refusal: "REFUSING: matrix is empty; nothing to cluster",
				Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "matrix-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			switch f.Name {
			case "min-builds-not-an-integer":
				s.PythonArgs = []string{"abc"}
			case "jaccard-not-a-number":
				s.PythonArgs = []string{"10", "xyz"}
			case "bad-header":
				s.Env = []string{"MATRIX=badheader.tsv"}
			case "empty-matrix":
				s.Env = []string{"MATRIX=empty.tsv"}
			}
			s.GoArgs = append([]string{"residual-clusters"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestResidualClustersParametersMatch: THE TWO PARAMETERS DECIDE THE CLUSTERING, and the default value of each hides the other's edges.
func TestResidualClustersParametersMatch(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := rcSandbox(t, repo)
	for _, args := range [][]string{
		{}, {"1"}, {"5"}, {"41"}, {"10", "1.0"}, {"10", "0.5"}, {"10", "0"}, {"10", ".93"},
	} {
		t.Run(strings.Join(append([]string{"a"}, args...), "-"), func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "residual-clusters",
				PythonScript: rcPy,
				PythonArgs:   args,
				GoBin:        bin,
				GoArgs:       append([]string{"residual-clusters"}, args...),
				Env:          []string{"MATRIX=matrix.tsv"},
				Stderr:       difftest.StderrExact,
			}, difftest.Fixture{Name: "params"}, mk)
		})
	}
}

func TestResidualClustersPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := rcSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "residual-clusters",
		PythonScript: rcPy,
		PythonArgs:   []string{"10"},
		Env:          []string{"MATRIX=matrix.tsv"},
	}, mk, []string{"0", "12345", "999983"})
}
