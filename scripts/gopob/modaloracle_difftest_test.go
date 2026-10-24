package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const moSh = "scripts/modal_oracle.sh"

const moStubExtract = `#!/usr/bin/env python3
import sys
print("extract-stub: dump=%s out=%s" % (sys.argv[1], sys.argv[2]))
open(sys.argv[2], "w").write('{"build":"b.xml"}\n')
`

const moStubModal = `#!/usr/bin/env python3
import sys
print("modal-stub: argv=%r" % (sys.argv[1:],))
open(sys.argv[2], "w").write("report\n")
open(sys.argv[3], "w").write('{"voted":true}\n')
`

func moSandbox(t *testing.T, repo string, runs int) func() (string, error) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, moSh)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			moSh:                              string(sh),
			"oracle/extract_output_oracle.py": moStubExtract,
			"oracle/modal_blob.py":            moStubModal,
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
		for i := 1; i <= runs; i++ {
			d := filepath.Join(sb, fmt.Sprintf("dumps-%d", i))
			if err := os.MkdirAll(d, 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(filepath.Join(d, "b.json"), []byte("{}"), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func moSpec(bin string, env []string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "modal-oracle",
		PythonScript: moSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"modal-oracle"}, args...),
		Stderr:       difftest.StderrExact,
		Env:          append([]string{"GOPOB_NICED=1"}, env...),
		RefusalSites: 5,
	}
}

var moDefaultArgs = []string{"dumps", "2", "out.jsonl.gz", "work"}

func TestModalOraclePortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := moSandbox(t, repo, 2)

	shSrc := difftest.ReadOriginal(t, repo, moSh)

	type fixture struct {
		difftest.Fixture
		args []string
		env  []string
	}
	fixtures := []fixture{
		{Fixture: difftest.Fixture{Name: "happy"}},

		{args: []string{}, Fixture: difftest.Fixture{
			Name: "no-arguments", Refusal: "FATAL: missing <dumps-base>",
			Break: func(string) error { return nil }}},

		{args: []string{"dumps", "2"}, Fixture: difftest.Fixture{
			Name: "two-arguments", Refusal: "FATAL: missing <out.jsonl.gz>",
			Break: func(string) error { return nil }}},

		{args: []string{"dumps", "many", "out.jsonl.gz", "work"}, Fixture: difftest.Fixture{
			Name:    "runs-not-a-number",
			Refusal: "<runs> must be a positive integer, not 'many'",
			Break:   func(string) error { return nil }}},

		{args: []string{"dumps", "0", "out.jsonl.gz", "work"}, Fixture: difftest.Fixture{
			Name: "zero-runs", Refusal: "FATAL: <runs> must be at least 1, not 0",
			Break: func(string) error { return nil }}},

		{Fixture: difftest.Fixture{Name: "missing-dump-dir", Refusal: "FATAL: no dump dir dumps-2",
			Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "dumps-2")) }}},

		{Fixture: difftest.Fixture{Name: "an-extraction-fails",
			Refusal: "FATAL: an extraction failed; see work/run-1.jsonl.log",
			Break: func(sb string) error {
				return put(sb, "oracle/extract_output_oracle.py",
					"#!/usr/bin/env python3\nimport sys\nsys.exit('extract-stub: boom')\n")
			}}},

		{Fixture: difftest.Fixture{Name: "reuses-a-fresh-extraction", Passes: true,
			Break: func(sb string) error {
				if err := os.MkdirAll(filepath.Join(sb, "work"), 0o750); err != nil {
					return fmt.Errorf("fixture: %w", err)
				}
				return put(sb, "work/run-1.jsonl", `{"build":"already-done.xml"}`+"\n")
			}}},

		{Fixture: difftest.Fixture{Name: "an-empty-extraction-is-redone", Passes: true,
			Break: func(sb string) error {
				if err := os.MkdirAll(filepath.Join(sb, "work"), 0o750); err != nil {
					return fmt.Errorf("fixture: %w", err)
				}
				return put(sb, "work/run-1.jsonl", "")
			}}},

		{Fixture: difftest.Fixture{Name: "re-extracts-when-the-dump-is-newer", Passes: true,
			Break: func(sb string) error {
				if err := os.MkdirAll(filepath.Join(sb, "work"), 0o750); err != nil {
					return fmt.Errorf("fixture: %w", err)
				}
				if err := put(sb, "work/run-1.jsonl", `{"build":"stale.xml"}`+"\n"); err != nil {
					return err
				}
				old := time.Now().Add(-time.Hour)
				return os.Chtimes(filepath.Join(sb, "work/run-1.jsonl"), old, old)
			}}},

		{env: []string{"MINFRAC=0"}, Fixture: difftest.Fixture{
			Name: "min-fraction-override", Passes: true,
			Break: func(string) error { return nil }}},
	}

	table := make([]difftest.Fixture, 0, len(fixtures))
	for _, f := range fixtures {
		table = append(table, f.Fixture)
	}
	spec := moSpec(bin, nil, moDefaultArgs...)
	spec.Fixtures = table
	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			args := f.args
			if args == nil {
				args = moDefaultArgs
			}
			difftest.RunFixture(t, t.Context(), moSpec(bin, f.env, args...), f.Fixture, mk)
		})
	}
}

// TestModalOracleWritesTheBlobAndReport: THE BLOB AND ITS REPORT ARE THE PRODUCT; the six progress lines are the label.
func TestModalOracleWritesTheBlobAndReport(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := moSandbox(t, repo, 2)

	spec := moSpec(bin, nil, moDefaultArgs...)
	spec.Files = []string{"out.jsonl.gz", "out.jsonl.gz.report.txt",
		"work/run-1.jsonl", "work/run-1.jsonl.log", "work/run-2.jsonl.log"}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "products-are-identical", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

func TestModalOracleIsStableAcrossLocales(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := moSandbox(t, repo, 2)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "modal-oracle",
		PythonScript: moSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   moDefaultArgs,
		Env:          []string{"GOPOB_NICED=1"},
	}, mk, []string{"C", "en_US.utf8", "C.utf8"})
}
