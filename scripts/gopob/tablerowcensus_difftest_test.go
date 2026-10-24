package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const trcPy = "scripts/table_row_census.py"

const (
	trcPin  = "pin/ModParser.lua"
	trcOurs = "our/mod_parser.go"
)

func synthPin(n int) string {
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, "\t[\"unique row alpha %05d\"] = 1,\n", i)
	}
	return b.String()
}

func trcSandbox(t *testing.T, repo, pob string) func() (string, error) {
	t.Helper()
	pin, err := os.ReadFile(filepath.Join(pob, "src/Modules/ModParser.lua"))
	if err != nil {
		t.Skipf("pinned PoB ModParser.lua not readable: %v", err)
	}
	ours, err := os.ReadFile(filepath.Join(repo, "scripts/gopob/testdata/trc/mod_parser.go.frozen"))
	if err != nil {
		t.Fatal(err)
	}
	py := difftest.ReadOriginal(t, repo, trcPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		for rel, body := range map[string][]byte{trcPy: py, trcPin: pin, trcOurs: ours} {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, body, 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func put(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func TestTableRowCensusPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	pob := pobDir(t)
	bin := buildPort(t, repo)
	mk := trcSandbox(t, repo, pob)

	pySrc := difftest.ReadOriginal(t, repo, trcPy)

	args := []string{trcPin, trcOurs}
	spec := difftest.Spec{
		Tool:         "table-row-census",
		PythonScript: trcPy,
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"table-row-census"}, args...),
		Stderr:       difftest.StderrExact,
		RefusalSites: 7,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "REFUSING: usage: table_row_census.py",
				Break: func(string) error { return nil }},

			{Name: "bad-min", Refusal: "min_family_size must be a non-negative integer, not xyz",
				Break: func(string) error { return nil }},

			{Name: "pin-missing", Refusal: trcPin + " is not readable:",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, trcPin)) }},
			{Name: "pin-unreadable", Refusal: trcPin + " is not readable:",
				Break: func(sb string) error { return os.Chmod(filepath.Join(sb, trcPin), 0o000) }},

			{Name: "pin-invalid-utf8", Refusal: trcPin + " is not valid UTF-8",
				Break: func(sb string) error {
					return os.WriteFile(filepath.Join(sb, trcPin),
						[]byte("\t[\"a\xff\xfe row\"] = 1,\n"), 0o600)
				}},

			{Name: "pin-499-keys", Refusal: "only 499 keys parsed from the pin",
				Break: func(sb string) error { return put(sb, trcPin, synthPin(499)) }},

			{Name: "control-have-row-absent",
				Refusal: "a row this parser demonstrably carries was not found",
				Break: func(sb string) error {
					return put(sb, trcOurs, "package calculator\n// nothing the pin carries\n")
				}},

			{Name: "control-absent-row-found",
				Refusal: "a nonsense row was 'found'",
				Break: func(sb string) error {
					b, err := os.ReadFile(filepath.Join(sb, trcOurs)) //nolint:gosec // harness temp
					if err != nil {
						return fmt.Errorf("fixture: %w", err)
					}
					return put(sb, trcOurs, string(b)+"\n// zz not a real modifier line zz\n")
				}},

			{Name: "pin-500-keys", Passes: true,
				Break: func(sb string) error { return put(sb, trcPin, synthPin(500)) }},

			{Name: "row-cap-binds", Passes: true, Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "pin-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			switch f.Name {
			case "no-args":
				s.PythonArgs, s.GoArgs = nil, []string{"table-row-census"}
			case "bad-min":
				s.PythonArgs = append(append([]string{}, args...), "xyz")
				s.GoArgs = append([]string{"table-row-census"}, s.PythonArgs...)
			case "row-cap-binds":
				s.PythonArgs = append(append([]string{}, args...), "1")
				s.GoArgs = append([]string{"table-row-census"}, s.PythonArgs...)
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestTableRowCensusPythonIsStableAcrossHashSeeds: THE SEED SWEEP MATTERS MORE HERE THAN ANYWHERE ELSE IN THE FOLD.
func TestTableRowCensusPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	pob := pobDir(t)
	mk := trcSandbox(t, repo, pob)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "table-row-census",
		PythonScript: trcPy,
		PythonArgs:   []string{trcPin, trcOurs},
	}, mk, []string{"0", "12345", "999983"})
}
