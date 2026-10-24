package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const ecdPy = "oracle/extract_calcs_damage.py"

func ecdDumpAlpha() string {
	return `{
  "mainSkill": "Cyclone",
  "output": {
    "IntVal": 5,
    "FloatVal": 5.0,
    "NegZero": -0,
    "Big": 2880521056.8914,
    "ExpHighIn": 9900000000000000.0,
    "ExpHighOut": 1e16,
    "ExpLowIn": 0.0001,
    "ExpLowOut": 0.00001,
    "IntegralLargeFloat": 2000000000.0,
    "Angle<Br>&Co": 3,
    "Naïve": 4,
    "Above💩BMP": 7,
    "BoolDropped": true,
    "StrDropped": "no",
    "NestedDropped": {"a": 1}
  },
  "outputMainHand": {"HitMin": 12, "HitMax": 30.5},
  "outputOffHand": []
}`
}

func ecdDumps() map[string]string {
	return map[string]string{
		"dumps/0001-Beta.json":    ecdDumpAlpha(),
		"dumps/0002-error.json":   `{"error": "build failed to load", "mainSkill": "Ignored", "output": {"X": 1}}`,
		"dumps/0003-falsy.json":   `{"mainSkill": "", "output": {"A": 1}}`,
		"dumps/0004-zero.json":    `{"mainSkill": 0, "output": {"A": 1}}`,
		"dumps/0005-false.json":   `{"mainSkill": false, "output": {"A": 1}}`,
		"dumps/0006-null.json":    `{"mainSkill": null, "output": {"A": 1}}`,
		"dumps/0007-absent.json":  `{"output": {"A": 1}}`,
		"dumps/0008-numeric.json": `{"mainSkill": 42, "output": {"A": 1}}`,
		"dumps/0009-optional.json": `{"mainSkill": "S", "output": [],
			"outputMainHand": null, "outputOffHand": {"B": 2}}`,
		"nodumps/readme.txt": "a directory with no .json in it at all\n",
		"notadir":            "a regular file where a dumps directory is expected\n",
	}
}

func ecdSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	src := difftest.ReadOriginal(t, repo, ecdPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{ecdPy: string(src)}
		for k, v := range ecdDumps() {
			files[k] = v
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

func ecdPut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func TestExtractCalcsDamage(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ecdSandbox(t, repo)
	pySrc := difftest.ReadOriginal(t, repo, ecdPy)

	base := []string{"dumps", "out.jsonl"}
	spec := difftest.Spec{
		Tool:         "extract-calcs-damage",
		PythonScript: ecdPy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"extract-calcs-damage"}, base...),
		Stderr:       difftest.StderrExact,
		Files:        []string{"out.jsonl"},
		RefusalSites: 2,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "usage: extract_calcs_damage.py <dumpsDir> <out.jsonl>",
				Break: func(string) error { return nil }},
			{Name: "dumps-dir-missing", Refusal: "REFUSING: nope is not a readable directory: No such file or directory",
				Break: func(string) error { return nil }},
			{Name: "dumps-dir-is-a-file", Refusal: "REFUSING: notadir is not a readable directory: Not a directory",
				Break: func(string) error { return nil }},
			{Name: "no-json-dumps", Refusal: "REFUSING: nodumps holds no .json dumps",
				Break: func(string) error { return nil }},
			{Name: "out-path-is-a-directory", Refusal: "REFUSING: cannot write out.jsonl: Is a directory",
				Break: func(sb string) error { return os.Mkdir(filepath.Join(sb, "out.jsonl"), 0o750) }},

			{Name: "dump-unreadable", Refusal: "0002-error.json is not readable: Permission denied",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "dumps/0002-error.json"), 0o000)
				}},
			{Name: "dump-not-json", Refusal: "0002-error.json is not valid JSON",
				Break: func(sb string) error { return ecdPut(sb, "dumps/0002-error.json", `{"output":`) }},
			{Name: "dump-invalid-utf8", Refusal: "0002-error.json is not valid JSON",
				Break: func(sb string) error {
					return os.WriteFile(filepath.Join(sb, "dumps/0002-error.json"),
						[]byte("{\"output\": {\"K\xffX\": 1}}"), 0o600)
				}},
			{Name: "dump-trailing-data", Refusal: "0002-error.json is not valid JSON",
				Break: func(sb string) error {
					return ecdPut(sb, "dumps/0002-error.json", `{"output": {"K": 1}}{"extra": 2}`)
				}},
			{Name: "dump-not-an-object", Refusal: "0002-error.json is not a JSON object",
				Break: func(sb string) error { return ecdPut(sb, "dumps/0002-error.json", `[]`) }},

			{Name: "one-dump-only", Passes: true, Break: func(sb string) error {
				entries, err := os.ReadDir(filepath.Join(sb, "dumps"))
				if err != nil {
					return fmt.Errorf("fixture: %w", err)
				}
				for _, e := range entries {
					if e.Name() == "0001-Beta.json" {
						continue
					}
					if err := os.Remove(filepath.Join(sb, "dumps", e.Name())); err != nil {
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
			s.GoArgs = append([]string{"extract-calcs-damage"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}

	difftest.AssertKeyDeletionAgrees(t, t.Context(), spec, []string{"dumps/0001-Beta.json", "dumps/0003-falsy.json"}, mk)
}

func TestExtractCalcsDamagePythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := ecdSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "extract-calcs-damage",
		PythonScript: ecdPy,
		PythonArgs:   []string{"dumps", "out.jsonl"},
		Files:        []string{"out.jsonl"},
	}, mk, []string{"0", "12345", "999983"})
}
