package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const eooPy = "oracle/extract_output_oracle.py"

func eooDumpAlpha() string {
	return `{
  "modDB": {
    "Keyed": [
      {"source": "Tree:100"},
      {"source": "Tree:100"},
      {"source": "Tree:65535"},
      {"source": "Tree:65536"},
      {"source": "Tree:200 (Notable)"},
      {"source": "Tree:300(Keystone)"},
      {"source": "Tree::400"},
      {"source": "Tree:٣"},
      {"source": "Tree:²"},
      {"source": "Tree:"},
      {"source": "Item:Lioneye's Fall"},
      {"source": 5},
      {"noSource": 1},
      "not-a-dict"
    ],
    "NotAList": "scalar"
  },
  "output": {
    "IntVal": 5,
    "FloatVal": 5.0,
    "NegZero": -0,
    "NegZeroFloat": -0.0,
    "Normalised": 1.5000,
    "Big": 2880521056.8914,
    "Small": 0.00026740809142486,
    "ExpHighIn": 9900000000000000.0,
    "ExpHighOut": 1e16,
    "ExpLowIn": 0.0001,
    "ExpLowOut": 0.00001,
    "IntegralLargeFloat": 2000000000.0,
    "BoolDropped": true,
    "StringDropped": "x",
    "NestedDropped": {"a": 1},
    "ListDropped": [1, 2],
    "NullDropped": null,
    "Damageü": 7,
    "A<B&C>D": 8,
    "ZZLast": 1,
    "AAFirst": 2
  },
  "outputMainHand": {"MainHandOnly": 1},
  "outputOffHand": [],
  "minionOutput": null
}`
}

func eooDumpError() string {
	return `{"error": "boom — ünïcode <&> 🔥", "output": {"Ignored": 1}}`
}

func eooDumpMinimal() string {
	return `{"output": {"Only": 1}, "enemyOutput": {}}`
}

func eooSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, eooPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			eooPy:                   string(py),
			"dumps/0001-Beta.json":  eooDumpError(),
			"dumps/0002-alpha.json": eooDumpAlpha(),
			"dumps/0003-gamma.json": eooDumpMinimal(),
			"dumps/notes.txt":       "a non-dump file, which must be ignored rather than parsed\n",
			"nodumps/readme.txt":    "a directory with no .json in it at all\n",
			"notadir":               "a regular file where a dumps directory is expected\n",
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

func TestExtractOutputOraclePortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := eooSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, eooPy)

	base := []string{"dumps", "out.jsonl"}
	spec := difftest.Spec{
		Tool:         "extract-output-oracle",
		PythonScript: eooPy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"extract-output-oracle"}, base...),
		Stderr:       difftest.StderrExact,
		Files:        []string{"out.jsonl"},
		RefusalSites: 2,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "usage: extract_output_oracle.py <dumpsDir> <out.jsonl>",
				Break: func(string) error { return nil }},
			{Name: "dumps-dir-missing", Refusal: "REFUSING: nope is not a readable directory: No such file or directory",
				Break: func(string) error { return nil }},
			{Name: "dumps-dir-is-a-file", Refusal: "REFUSING: notadir is not a readable directory: Not a directory",
				Break: func(string) error { return nil }},
			{Name: "no-json-dumps", Refusal: "REFUSING: nodumps holds no .json dumps",
				Break: func(string) error { return nil }},
			{Name: "out-path-is-a-directory", Refusal: "REFUSING: cannot write out.jsonl: Is a directory",
				Break: func(sb string) error { return os.Mkdir(filepath.Join(sb, "out.jsonl"), 0o750) }},

			{Name: "dump-unreadable", Refusal: "0002-alpha.json is not readable: Permission denied",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "dumps/0002-alpha.json"), 0o000)
				}},
			{Name: "dump-not-json", Refusal: "0002-alpha.json is not valid JSON",
				Break: func(sb string) error { return put(sb, "dumps/0002-alpha.json", `{"output":`) }},
			{Name: "dump-invalid-utf8", Refusal: "0002-alpha.json is not valid JSON",
				Break: func(sb string) error {
					return os.WriteFile(filepath.Join(sb, "dumps/0002-alpha.json"),
						[]byte("{\"output\": {\"K\xffX\": 1}}"), 0o600)
				}},
			{Name: "dump-trailing-data", Refusal: "0002-alpha.json is not valid JSON",
				Break: func(sb string) error {
					return put(sb, "dumps/0002-alpha.json", `{"output": {"K": 1}}{"extra": 2}`)
				}},
			{Name: "dump-not-an-object", Refusal: "0002-alpha.json is not a JSON object",
				Break: func(sb string) error { return put(sb, "dumps/0002-alpha.json", `[]`) }},

			{Name: "one-dump-only", Passes: true, Break: func(sb string) error {
				for _, f := range []string{"dumps/0001-Beta.json", "dumps/0003-gamma.json"} {
					if err := os.Remove(filepath.Join(sb, f)); err != nil {
						return fmt.Errorf("fixture: %w", err)
					}
				}
				return nil
			}},
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
				s.PythonArgs = []string{"dumps"}
			case "dumps-dir-missing":
				s.PythonArgs = []string{"nope", "out.jsonl"}
			case "dumps-dir-is-a-file":
				s.PythonArgs = []string{"notadir", "out.jsonl"}
			case "no-json-dumps":
				s.PythonArgs = []string{"nodumps", "out.jsonl"}
			}
			if f.Name != "happy" && f.Name != "one-dump-only" {
				s.Files = nil
			}
			s.GoArgs = append([]string{"extract-output-oracle"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}

	difftest.AssertKeyDeletionAgrees(t, t.Context(), spec, []string{"dumps/0002-alpha.json", "dumps/0001-Beta.json"}, mk)
}

func TestExtractOutputOraclePythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := eooSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "extract-output-oracle",
		PythonScript: eooPy,
		PythonArgs:   []string{"dumps", "out.jsonl"},
		Files:        []string{"out.jsonl"},
	}, mk, []string{"0", "12345", "999983"})
}
