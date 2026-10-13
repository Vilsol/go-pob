package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const pddPy = "scripts/pob_dump_diff.py"

func pddDump(spec map[string]string) string {
	var b strings.Builder
	b.WriteString(`{"modDB": {`)
	b.WriteString(`"Keyed": [` + spec["tree"] + `, {"source": "Item:Lioneye's Fall"}],`)
	b.WriteString(`"PtrOnly": [{"fn": "function: 0x` + spec["addr"] + `"}],`)
	b.WriteString(`"Reordered": ` + spec["reordered"] + `,`)
	b.WriteString(`"Nested": [[{"n": 1}], [{"n": 2}]]`)
	b.WriteString(`},`)
	b.WriteString(`"output": {"Same": ` + spec["same"] + `, "Moved": ` + spec["moved"] + `,`)
	b.WriteString(spec["outonly"])
	b.WriteString(`"Bool": ` + spec["bool"] + `, "Str": ` + spec["str"] + `},`)
	b.WriteString(`"breakdown": {"Trace": ` + spec["trace"] + `},`)
	b.WriteString(`"census": {"Typed": ` + spec["typed"] + `, "Name": ` + spec["name"] + `,`)
	b.WriteString(`"Nested": {"Deep": ` + spec["deep"] + `}` + spec["bulk"] + `}}`)
	return b.String()
}

func pddBulk(n, offset int) string {
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, `, "Bulk%02d": %d`, i, i+offset)
	}
	return b.String()
}

func pddSideA(bulk int) string {
	return pddDump(map[string]string{
		"tree":      `{"source": "Tree:100"}, {"source": "Tree:200"}, {"source": "Tree:300"}`,
		"addr":      "55d1a2b3c4d5",
		"reordered": `[{"a": 1}, {"a": 2}, {"a": 3}]`,
		"same":      "5", "moved": "10", "outonly": `"OnlyA": 1,`,
		"bool": "true", "str": `"a"`, "trace": "1", "typed": "5",
		"name": `"Lioneye's Fall"`, "deep": "1", "bulk": pddBulk(bulk, 0),
	})
}

func pddSideB(bulk int) string {
	return pddDump(map[string]string{
		"tree":      `{"source": "Tree:100"}, {"source": "Tree:200"}`,
		"addr":      "7f0099887766",
		"reordered": `[{"a": 3}, {"a": 1}, {"a": 2}]`,
		"same":      "5.0", "moved": "12", "outonly": `"OnlyB": 2,`,
		"bool": "false", "str": `"b"`, "trace": "2", "typed": `"5"`,
		"name": `"Lioneye's Fall"`, "deep": "9", "bulk": pddBulk(bulk, 100),
	})
}

func pddSandbox(t *testing.T, repo string, bulk int) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, pddPy)
	ex := difftest.ReadOriginal(t, repo, eooPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			pddPy: string(py), eooPy: string(ex),
			"A/0001-alpha.json": pddSideA(bulk), "A/0002-beta.json": pddSideA(bulk),
			"B/0001-alpha.json": pddSideB(bulk), "B/0002-beta.json": pddSideB(bulk),
			"same/0001-alpha.json": pddSideA(bulk), "same/0002-beta.json": pddSideA(bulk),
			"nodumps/readme.txt":       "no dumps here\n",
			"otherset/0001-alpha.json": pddSideB(bulk), "otherset/0009-zeta.json": pddSideB(bulk),
		}
		for i := range 12 {
			files[fmt.Sprintf("bigset/%04d-extra.json", i)] = pddSideB(bulk)
		}
		for rel, body := range files {
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

func TestPobDumpDiffPortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pddSandbox(t, repo, 31)

	pySrc := difftest.ReadOriginal(t, repo, pddPy)

	base := []string{"A", "B"}
	spec := difftest.Spec{
		Tool:            "pob-dump-diff",
		PythonScript:    pddPy,
		PythonArgs:      base,
		GoBin:           bin,
		GoArgs:          append([]string{"pob-dump-diff"}, base...),
		Stderr:          difftest.StderrExact,
		RefusalOnStdout: true,
		RefusalSites:    2,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "usage: pob_dump_diff.py <dirA> <dirB> [--verbose]",
				Break: func(string) error { return nil }},
			{Name: "dir-missing", Refusal: "REFUSING: [Errno 2] No such file or directory: 'nope'",
				Break: func(string) error { return nil }},
			{Name: "no-json-dumps", Refusal: "REFUSING: nodumps holds no .json dumps -- nothing was compared",
				Break: func(string) error { return nil }},
			{Name: "different-build-sets", Refusal: "only in A: ['0002-beta.json']",
				Break: func(string) error { return nil }},

			{Name: "dump-unreadable", Refusal: "0001-alpha.json is not readable: Permission denied",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "B/0001-alpha.json"), 0o000)
				}},
			{Name: "dump-not-json", Refusal: "0001-alpha.json is not valid JSON",
				Break: func(sb string) error { return put(sb, "B/0001-alpha.json", `{"output":`) }},
			{Name: "dump-not-an-object", Refusal: "0001-alpha.json is not a JSON object",
				Break: func(sb string) error { return put(sb, "B/0001-alpha.json", `[]`) }},
			{Name: "dump-trailing-data", Refusal: "0001-alpha.json is not valid JSON",
				Break: func(sb string) error {
					return put(sb, "B/0001-alpha.json", `{"output": {"K": 1}}{"extra": 2}`)
				}},
			{Name: "build-sets-over-cap", Refusal: "'0009-extra.json']",
				Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "dump-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			switch f.Name {
			case "no-args":
				s.PythonArgs = []string{"A"}
			case "dir-missing":
				s.PythonArgs = []string{"nope", "B"}
			case "no-json-dumps":
				s.PythonArgs = []string{"nodumps", "B"}
			case "different-build-sets":
				s.PythonArgs = []string{"A", "otherset"}
			case "build-sets-over-cap":
				s.PythonArgs = []string{"A", "bigset"}
			}
			s.GoArgs = append([]string{"pob-dump-diff"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}

	difftest.AssertKeyDeletionAgrees(t, t.Context(), spec, []string{"A/0001-alpha.json"}, mk)
}

// TestPobDumpDiffTruncationBoundary: THE 30-ROW TRUNCATION IS A THRESHOLD AND NEEDS BOTH SIDES.
func TestPobDumpDiffTruncationBoundary(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	for _, tc := range []struct {
		name string
		bulk int
		args []string
	}{
		{"under-28", 28, []string{"A", "B"}},
		{"at-30", 30, []string{"A", "B"}},
		{"over-31", 31, []string{"A", "B"}},
		{"verbose-31", 31, []string{"A", "B", "--verbose"}},
		{"verbose-30", 30, []string{"A", "B", "--verbose"}},
		{"blob-neutral", 31, []string{"A", "same"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mk := pddSandbox(t, repo, tc.bulk)
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:            "pob-dump-diff",
				PythonScript:    pddPy,
				PythonArgs:      tc.args,
				GoBin:           bin,
				GoArgs:          append([]string{"pob-dump-diff"}, tc.args...),
				Stderr:          difftest.StderrExact,
				RefusalOnStdout: true,
			}, difftest.Fixture{Name: tc.name}, mk)
		})
	}
}

func TestPobDumpDiffPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := pddSandbox(t, repo, 31)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "pob-dump-diff",
		PythonScript: pddPy,
		PythonArgs:   []string{"A", "same"},
	}, mk, []string{"0", "12345", "999983"})
}
