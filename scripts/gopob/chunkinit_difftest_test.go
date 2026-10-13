package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const ckPy = "scripts/chunk_init.py"

const ckSource = "package raw\n\n" +
	"type V struct {\n\tA string\n\tB int\n}\n\n" +
	"var Table = map[string]V{\n" +
	"\t// src/Data/Skills/act_str.lua:1201\n" +
	"\t\"plain\": {A: \"x\", B: 1},\n" +
	"\t\"brace in string\": {A: \"}\", B: 2},\n" +
	"\t\"open in string\": {A: \"{\", B: 3},\n" +
	"\t// a comment with } } in it\n" +
	"\t\"after comment\": {A: \"y\", B: 4},\n" +
	"\t\"block comment\": /* } } */ {A: \"z\", B: 5},\n" +
	"\t\"raw string\": {A: `raw } } raw`, B: 6},\n" +
	"\t\"escaped quote\": {A: \"he said \\\"}\\\" loudly\", B: 7},\n" +
	"\t\"colon: in key\": {A: \"v:w\", B: 8},\n" +
	"\t\"rune\": {A: string('}'), B: 9},\n" +
	"\t\"nested\": {A: \"n\", B: len([]int{1, 2, 3})},\n" +
	"}\n\n" +
	"func Unrelated() int { return 1 }\n"

const ckNotAMap = "package raw\n\nvar NotAMap = []string{\n\t\"a\",\n\t\"b\",\n}\n"

func ckSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, ckPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		for rel, body := range map[string]string{
			ckPy: string(py), "in.go": ckSource, "notamap.go": ckNotAMap,
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

func TestChunkInitPortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ckSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, ckPy)

	base := []string{"in.go", "Table", "--per", "3"}
	spec := difftest.Spec{
		Tool:         "chunk-init",
		PythonScript: ckPy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"chunk-init"}, base...),
		Stderr:       difftest.StderrExact,
		Files:        []string{"in.go"},
		RefusalSites: 2,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "chunk-init: need exactly <file.go> and <VarName>, got 0",
				Break: func(string) error { return nil }},
			{Name: "unknown-flag", Refusal: "chunk-init: unknown argument '--bogus'",
				Break: func(string) error { return nil }},
			{Name: "per-without-value", Refusal: "chunk-init: --per needs a value",
				Break: func(string) error { return nil }},
			{Name: "per-not-a-number", Refusal: "chunk-init: --per must be a positive integer, not two",
				Break: func(string) error { return nil }},
			{Name: "per-zero", Refusal: "chunk-init: --per must be a positive integer, not 0",
				Break: func(string) error { return nil }},
			{Name: "file-unreadable", Refusal: "chunk-init: cannot read in.go:",
				Break: func(sb string) error { return os.Chmod(filepath.Join(sb, "in.go"), 0o000) }},
			{Name: "no-such-var", Refusal: "REFUSING: no `var Missing = ...{` declaration found",
				Break: func(string) error { return nil }},
			{Name: "not-a-map", Refusal: "REFUSING: []string is not a map type",
				Break: func(string) error { return nil }},
			{Name: "unbalanced-braces", Refusal: "REFUSING: unbalanced braces in Table",
				Break: func(sb string) error {
					return put(sb, "in.go", strings.Replace(ckSource, "}\n\nfunc Unrelated", "\n\nfunc Unrelated", 1))
				}},
			{Name: "no-entries", Refusal: "REFUSING: Table has no top-level entries",
				Break: func(sb string) error {
					return put(sb, "in.go", "package raw\n\nvar Table = map[string]int{\n}\n")
				}},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "file-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			switch f.Name {
			case "no-args":
				s.PythonArgs = nil
			case "unknown-flag":
				s.PythonArgs = append(append([]string{}, base...), "--bogus")
			case "per-without-value":
				s.PythonArgs = []string{"in.go", "Table", "--per"}
			case "per-not-a-number":
				s.PythonArgs = []string{"in.go", "Table", "--per", "two"}
			case "per-zero":
				s.PythonArgs = []string{"in.go", "Table", "--per", "0"}
			case "no-such-var":
				s.PythonArgs = []string{"in.go", "Missing"}
			case "not-a-map":
				s.PythonArgs = []string{"notamap.go", "NotAMap"}
			}
			if f.Name != "happy" {
				s.Files = nil
			}
			s.GoArgs = append([]string{"chunk-init"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestChunkInitChunkSizesMatch: CHUNK SIZE CHANGES THE OUTPUT STRUCTURE, and one value exercises one shape.
func TestChunkInitChunkSizesMatch(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ckSandbox(t, repo)
	for _, per := range []string{"1", "2", "5", "1000"} {
		t.Run("per"+per, func(t *testing.T) {
			args := []string{"in.go", "Table", "--per", per}
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "chunk-init",
				PythonScript: ckPy,
				PythonArgs:   args,
				GoBin:        bin,
				GoArgs:       append([]string{"chunk-init"}, args...),
				Stderr:       difftest.StderrExact,
				Files:        []string{"in.go"},
			}, difftest.Fixture{Name: "per" + per}, mk)
		})
	}
}

// TestChunkInitCheckModeMatchesAndWritesNothing: --check EXITS 1 FOR "THE FILE WOULD CHANGE".
func TestChunkInitCheckModeMatchesAndWritesNothing(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ckSandbox(t, repo)

	run := func(t *testing.T, argv []string, dir string) (string, int) {
		t.Helper()
		//nolint:gosec // argv is this test's own literal
		cmd := exec.CommandContext(t.Context(), argv[0], argv[1:]...)
		cmd.Dir = dir
		out, _ := cmd.CombinedOutput()
		return string(out), cmd.ProcessState.ExitCode()
	}
	for _, tc := range []struct{ name, varName string }{
		{"would-change", "Table"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pySb, err := mk()
			if err != nil {
				t.Fatal(err)
			}
			goSb, err := mk()
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"in.go", tc.varName, "--check"}
			goOut, goRC := run(t, append([]string{bin, "chunk-init"}, args...), goSb)
			if !difftest.OriginalsDeleted(repoRoot(t)) {
				pyOut, pyRC := run(t, append([]string{"python3", filepath.Join(pySb, ckPy)}, args...), pySb)
				if pyRC != goRC || pyOut != goOut {
					t.Fatalf("--check differs: python rc=%d %q, go rc=%d %q", pyRC, pyOut, goRC, goOut)
				}
			}
			if goRC != 1 {
				t.Fatalf("--check on a file that would change exited %d, want 1", goRC)
			}
			for _, sb := range []string{pySb, goSb} {
				b, err := os.ReadFile(filepath.Join(sb, "in.go")) //nolint:gosec // harness temp
				if err != nil {
					t.Fatal(err)
				}
				if string(b) != ckSource {
					t.Fatalf("--check WROTE to %s; a gate wired to it would rewrite the tree", sb)
				}
			}
		})
	}
}

func TestChunkInitPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := ckSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "chunk-init",
		PythonScript: ckPy,
		PythonArgs:   []string{"in.go", "Table", "--per", "3"},
	}, mk, []string{"0", "12345", "999983"})
}
