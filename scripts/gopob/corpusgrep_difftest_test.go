package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const cgPy = "scripts/corpus_grep.py"

const (
	cgWrapped = "With at least 40 Dexterity\n            in Radius, Dual Strike deals more damage"
	cgEscaped = "Lioneye&apos;s Fall grants a thing"
)

const cgSandboxVersion = "9.99"

func cgCorpus() map[string]string {
	out := map[string]string{
		"b01": "<Build>\n  <Item>" + cgWrapped + "</Item>\n</Build>\n",
		"b02": "<Build>\n  <Item>" + cgEscaped + "</Item>\n</Build>\n",
		"b03": "<Build>\n  <Item>commontoken here and commontoken again</Item>\n</Build>\n",
		"b04": "<Build>\n  <Item>nothing of interest</Item>\n</Build>\n",
	}
	for i := 5; i <= 24; i++ {
		out[fmt.Sprintf("b%02d", i)] = "<Build>\n  <Item>commontoken</Item>\n</Build>\n"
	}
	return out
}

func cgSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, cgPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			"go.mod":           "module github.com/Vilsol/go-pob\n\ngo 1.26.2\n",
			cgPy:               string(py),
			"data/raw/main.go": "package raw\n\nconst DefaultVersion = \"" + cgSandboxVersion + "\"\n",
		}
		for b, body := range cgCorpus() {
			files[filepath.Join("testdata", "corpus-"+cgSandboxVersion, b+".xml")] = body
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

func TestCorpusGrepPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := cgSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, cgPy)

	spec := difftest.Spec{
		Tool:         "corpus-grep",
		PythonScript: cgPy,
		GoBin:        bin,
		GoArgs:       []string{"corpus-grep"},
		Env:          []string{"PATTERN=commontoken"},
		Stderr:       difftest.StderrExact,
		RefusalSites: 6,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-pattern", Refusal: "set PATTERN=<text|regex>",
				Break: func(string) error { return nil }},

			{Name: "no-go-mod", Refusal: "no go.mod declaring module github.com/Vilsol/go-pob",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "go.mod")) }},

			{Name: "bad-gopob-root", Refusal: "GOPOB_ROOT does not contain a go.mod declaring module",
				Break: func(string) error { return nil }},

			{Name: "corpus-dir-not-a-directory", Refusal: "CORPUS_DIR=nope is not a directory",
				Break: func(string) error { return nil }},
			{Name: "no-corpus-for-version", Refusal: "no corpus for game-data version '0.01'",
				Break: func(string) error { return nil }},
			{Name: "no-default-version-const", Refusal: "no `const DefaultVersion` in data/raw/main.go",
				Break: func(sb string) error {
					return put(sb, "data/raw/main.go", "package raw\n\n// the const moved\n")
				}},
			{Name: "version-source-unreadable", Refusal: "cannot read data/raw/main.go",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "data/raw/main.go"), 0o000)
				}},
			{Name: "empty-corpus", Refusal: "no builds under",
				Break: func(sb string) error {
					files, err := filepath.Glob(filepath.Join(sb, "testdata", "corpus-"+cgSandboxVersion, "*.xml"))
					if err != nil {
						return fmt.Errorf("fixture: %w", err)
					}
					for _, f := range files {
						if err := os.Remove(f); err != nil {
							return fmt.Errorf("fixture: %w", err)
						}
					}
					return nil
				}},
			{Name: "lookbehind-refused", Refusal: "which python allows and RE2 does not",
				Break: func(string) error { return nil }},

			{Name: "untranslatable-lua-class", Refusal: "is a Lua class this script cannot translate",
				Break: func(string) error { return nil }},

			{Name: "bad-regex", Refusal: "corpus-grep: bad pattern",
				Break: func(string) error { return nil }},

			{Name: "supported-regex", Passes: true, Break: func(string) error { return nil }},

			{Name: "case-insensitive-match", Passes: true, Break: func(string) error { return nil }},

			{Name: "lua-lazy-after-class", Passes: true, Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "version-source-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			switch f.Name {
			case "no-pattern":
				s.Env = []string{"PATTERN="}
			case "bad-gopob-root":
				s.Env = append(append([]string{}, spec.Env...), "GOPOB_ROOT=testdata")
			case "corpus-dir-not-a-directory":
				s.Env = append(append([]string{}, spec.Env...), "CORPUS_DIR=nope")
			case "no-corpus-for-version":
				s.Env = append(append([]string{}, spec.Env...), "POE_VERSION=0.01")
			case "lookbehind-refused":
				s.Env = []string{`PATTERN=(?<=a)b`}
			case "untranslatable-lua-class":
				s.Env = []string{"PATTERN=%z+", "LUA=1"}
			case "bad-regex":
				s.Env = []string{"PATTERN=[unclosed"}
			case "supported-regex":
				s.Env = []string{`PATTERN=common(token|thing)`}
			case "case-insensitive-match":
				s.Env = []string{"PATTERN=COMMONTOKEN"}
			case "lua-lazy-after-class":
				s.Env = []string{"PATTERN=common%d-token", "LUA=1"}
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestCorpusGrepFindsWhatALineGrepMisses: THE TWO TRAPS ARE THE TOOL, and the default PATTERN exercises neither.
func TestCorpusGrepFindsWhatALineGrepMisses(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := cgSandbox(t, repo)
	for name, pattern := range map[string]string{
		"wrapped-across-newlines":   "With at least 40 Dexterity in Radius",
		"entity-escaped-apostrophe": "Lioneye's Fall",
		"two-occurrences-one-build": "commontoken here and commontoken",
	} {
		t.Run(name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "corpus-grep",
				PythonScript: cgPy,
				GoBin:        bin,
				GoArgs:       []string{"corpus-grep"},
				Env:          []string{"PATTERN=" + pattern, "NAMES=1"},
				Stderr:       difftest.StderrExact,
			}, difftest.Fixture{Name: name}, mk)
		})
	}
}

func TestCorpusGrepPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := cgSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "corpus-grep",
		PythonScript: cgPy,
		Env:          []string{"PATTERN=commontoken"},
	}, mk, []string{"0", "12345", "999983"})
}
