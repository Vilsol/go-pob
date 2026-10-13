package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

// TestFoldProgressCountsSpecsNotMentions: THE CONTROL THAT MATTERS IS THE NEGATIVE ONE.
func TestFoldProgressCountsSpecsNotMentions(t *testing.T) {
	repo := repoRoot(t)
	ported, err := portedScripts(filepath.Join(repo, "scripts", "gopob"))
	if err != nil {
		t.Fatal(err)
	}

	difftests, err := filepath.Glob(filepath.Join(repo, "scripts", "gopob", "*_difftest_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(difftests) == 0 {
		t.Fatal("no differential tests found; this control is vacuous")
	}
	if len(ported) < len(difftests) {
		t.Errorf("%d differential test files but only %d ported scripts; one names no script",
			len(difftests), len(ported))
	}
	for path := range ported {
		if difftest.OriginalsDeleted(repo) {
			if !strings.HasPrefix(path, "scripts/") && !strings.HasPrefix(path, "oracle/") {
				t.Errorf("counted %q as ported, but it names neither scripts/ nor oracle/ — a Spec's "+
					"PythonScript is the identity of the ORIGINAL and cannot become something else", path)
			}
			continue
		}
		if _, err := os.Stat(filepath.Join(repo, path)); err != nil {
			t.Errorf("counted %q as ported and no such script exists: %v", path, err)
		}
	}
	mentioned := t.TempDir()
	if err := os.WriteFile(filepath.Join(mentioned, "reader_difftest_test.go"), []byte(`package main

import "github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"

// This tool READS the script below; it is an INPUT, not a port of it.
var sandboxFiles = []string{"oracle/mentioned_but_not_ported.py"}

var spec = difftest.Spec{Tool: "reader", PythonScript: "oracle/the_real_port.py"}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := portedScripts(mentioned)
	if err != nil {
		t.Fatal(err)
	}
	if got["oracle/mentioned_but_not_ported.py"] {
		t.Error("a path mentioned only in a string literal counts as ported; the definition has " +
			"widened to 'mentioned somewhere' and the metric now over-reports")
	}
	if !got["oracle/the_real_port.py"] {
		t.Errorf("the Spec's own PythonScript is not counted, so the check above proves nothing "+
			"about the definition: %v", got)
	}
}

// TestFoldProgressResolvesConstantsAcrossFiles: A CONSTANT IS PACKAGE-SCOPED AND THIS SCANNER RESOLVED IT PER FILE.
func TestFoldProgressResolvesConstantsAcrossFiles(t *testing.T) {
	dir := t.TempDir()

	specFile := `package main

import "github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"

var specs = map[string]difftest.Spec{
	"a": {Tool: "a", PythonScript: crossFilePy},
}
`
	constFile := "package main\n\nconst crossFilePy = \"oracle/cross_file_probe.py\"\n"
	for name, body := range map[string]string{
		"spec_difftest_test.go":  specFile,
		"other_difftest_test.go": constFile,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	ported, err := portedScripts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !ported["oracle/cross_file_probe.py"] {
		t.Errorf("a PythonScript naming a constant from a SIBLING file was not resolved; "+
			"constants are package-scoped and the scan is per-file again. Counted: %v", ported)
	}

	same := t.TempDir()
	if err := os.WriteFile(filepath.Join(same, "one_difftest_test.go"),
		[]byte(specFile+"\nconst crossFilePy = \"oracle/cross_file_probe.py\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := portedScripts(same)
	if err != nil {
		t.Fatal(err)
	}
	if !got["oracle/cross_file_probe.py"] {
		t.Errorf("same-file constant resolution broke: %v", got)
	}
}

// TestFoldProgressFallsWhenADifftestIsRemoved: AND IT MUST GO DOWN.
func TestFoldProgressFallsWhenADifftestIsRemoved(t *testing.T) {
	repo := repoRoot(t)
	dir := t.TempDir()
	entries, err := filepath.Glob(filepath.Join(repo, "scripts", "gopob", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var dropped string
	for _, src := range entries {
		if filepath.Base(src) == "tablerowcensus_difftest_test.go" {
			dropped = src
			continue
		}
		b, err := os.ReadFile(src) //nolint:gosec // a repo path this test enumerated
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(src)), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if dropped == "" {
		t.Fatal("the file this control removes no longer exists; the control is vacuous")
	}
	before, err := portedScripts(filepath.Join(repo, "scripts", "gopob"))
	if err != nil {
		t.Fatal(err)
	}
	after, err := portedScripts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if after["scripts/table_row_census.py"] {
		t.Error("a tool still counts as ported with its differential test deleted")
	}
	if len(before)-len(after) != 1 {
		t.Errorf("removing one difftest moved the count %d -> %d; want a drop of exactly 1",
			len(before), len(after))
	}
}

// TestShellRefusalSitesAreCounted: THE SHELL COUNT WAS ZERO UNTIL 2026-08-19, AND ZERO LOOKED LIKE A MEASUREMENT.
func TestShellRefusalSitesAreCounted(t *testing.T) {
	repo := repoRoot(t)
	paths, err := filepath.Glob(filepath.Join(repo, "scripts", "*.sh"))
	if err != nil {
		t.Fatal(err)
	}
	total, withAny := 0, 0
	for _, p := range paths {
		_, _, sites := statScript(repo, p)
		total += sites
		if sites > 0 {
			withAny++
		}
	}
	if total == 0 {
		t.Fatal("scripts/*.sh reports 0 refusal sites; the shell definition is not being applied")
	}
	if withAny < len(paths)/2 {
		t.Errorf("only %d of %d shell scripts carry a counted refusal; the pattern is too narrow",
			withAny, len(paths))
	}
	t.Logf("%d shell refusal sites across %d of %d files", total, withAny, len(paths))
}

// TestShellRefusalDefinition: The definition, pinned by example so a future widening is a deliberate edit rather than drift.
func TestShellRefusalDefinition(t *testing.T) {
	dir := t.TempDir()
	for body, want := range map[string]int{
		"exit 1\n":                1,
		"exit 3\nexit 64\n":       2,
		"exit 0\n":                0,
		"exit\n":                  0,
		"exit $rc\n":              1,
		"exit \"$rc\"\n":          1,
		"f() { return 1; }\n":     0,
		"# exit 1 in a comment\n": 1,
	} {
		p := filepath.Join(dir, "s.sh")
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, got := statScript(dir, p); got != want {
			t.Errorf("%q counted %d, want %d", body, got, want)
		}
	}
}
