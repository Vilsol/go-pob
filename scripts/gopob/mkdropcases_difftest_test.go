package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const mkPy = "scripts/mk_drop_cases.py"

func mkCensus() string {
	var b strings.Builder
	b.WriteString("builds\toccurrences\tkinds\tnormalised\tsample_line\tleftover_extra\n")
	for _, r := range [][6]string{
		{"9", "1", "k", "n", `+10 to "Maximum" Life`, "x"},
		{"3", "1", "k", "n", "Tie A — first in file", "y"},
		{"3", "1", "k", "n", "Tie B — second", "y"},
		{"3", "1", "k", "n", "Tie C — third", "y"},
		{"1", "1", "k", "n", strings.Repeat("A", 120), "y"},
		{"1", "1", "k", "n", "", "y"},
		{"1", "1", "k", "n", "Lioneye's Fall · ünïcode · <&>", "y"},
	} {
		b.WriteString(strings.Join(r[:], "\t") + "\n")
	}
	for i := range 15 {
		fmt.Fprintf(&b, "2\t1\tk\tn\tTied row %02d, distinguishable by name\ty\n", i)
	}
	return b.String()
}

func mkSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, mkPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		for rel, body := range map[string]string{
			mkPy: string(py), "census.tsv": mkCensus(),
			"badheader.tsv": "a\tb\tc\n1\t2\t3\n", "empty.tsv": "builds\toccurrences\tkinds\tnormalised\tsample_line\tleftover_extra\n",
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

func TestMkDropCasesPortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := mkSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, mkPy)

	base := []string{"census.tsv", "out.json"}
	spec := difftest.Spec{
		Tool:         "mk-drop-cases",
		PythonScript: mkPy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"mk-drop-cases"}, base...),
		Stderr:       difftest.StderrExact,
		Files:        []string{"out.json"},
		RefusalSites: 7,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "REFUSING: usage: mk_drop_cases.py",
				Break: func(string) error { return nil }},
			{Name: "top-not-a-number", Refusal: "REFUSING: top must be a non-negative integer, not many",
				Break: func(string) error { return nil }},
			{Name: "tsv-unreadable", Refusal: "REFUSING: census.tsv is not readable:",
				Break: func(sb string) error { return os.Chmod(filepath.Join(sb, "census.tsv"), 0o000) }},
			{Name: "bad-header", Refusal: "REFUSING: header is ['a', 'b', 'c'], expected",
				Break: func(string) error { return nil }},
			{Name: "empty-census", Refusal: "REFUSING: census TSV is empty",
				Break: func(string) error { return nil }},
			{Name: "builds-not-an-integer", Refusal: "census.tsv line 2: builds='lots' is not an integer",
				Break: func(sb string) error {
					return put(sb, "census.tsv",
						"builds\toccurrences\tkinds\tnormalised\tsample_line\tleftover_extra\n"+
							"lots\t1\tk\tn\tA line\ty\n")
				}},
			{Name: "output-unwritable", Refusal: "REFUSING: cannot write nodir/out.json:",
				Break: func(string) error { return nil }},

			{Name: "top-zero", Passes: true, Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "tsv-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			switch f.Name {
			case "no-args":
				s.PythonArgs = []string{"census.tsv"}
			case "top-not-a-number":
				s.PythonArgs = append(append([]string{}, base...), "many")
			case "bad-header":
				s.PythonArgs = []string{"badheader.tsv", "out.json"}
			case "empty-census":
				s.PythonArgs = []string{"empty.tsv", "out.json"}
			case "output-unwritable":
				s.PythonArgs = []string{"census.tsv", "nodir/out.json"}
			case "top-zero":
				s.PythonArgs = append(append([]string{}, base...), "0")
			}
			if f.Name != "happy" && f.Name != "top-zero" {
				s.Files = nil
			}
			s.GoArgs = append([]string{"mk-drop-cases"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestMkDropCasesTopValuesMatch: `top` DECIDES WHAT IS WRITTEN.
func TestMkDropCasesTopValuesMatch(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := mkSandbox(t, repo)
	for _, top := range []string{"1", "3", "6", "999"} {
		t.Run("top"+top, func(t *testing.T) {
			args := []string{"census.tsv", "out.json", top}
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "mk-drop-cases",
				PythonScript: mkPy,
				PythonArgs:   args,
				GoBin:        bin,
				GoArgs:       append([]string{"mk-drop-cases"}, args...),
				Stderr:       difftest.StderrExact,
				Files:        []string{"out.json"},
			}, difftest.Fixture{Name: "top" + top}, mk)
		})
	}
}

func TestMkDropCasesPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := mkSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "mk-drop-cases",
		PythonScript: mkPy,
		PythonArgs:   []string{"census.tsv", "out.json"},
	}, mk, []string{"0", "12345", "999983"})
}
