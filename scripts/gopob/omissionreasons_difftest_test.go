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

const orPy = "scripts/omission_reasons.py"

func orEmitter() string {
	reasons := []string{
		"plain reason with no marker at all",
		"opt out properly [unverifiable: data-dependent]",
		"opt out emptily [unverifiable:]",
		"symbol present [go:mod.OrFixtureSymbol]",
		"symbol gone [go:mod.OrNoSuchSymbolAnywhere]",
		"claims absent but present [go-absent:mod.OrFixtureSymbol]",
		"claims absent and is [go-absent:mod.OrNoSuchSymbolAnywhere]",
		"resolves fine [pob:src/Modules/Calc.lua:5-7]",
		"no such lua [pob:src/Modules/NoSuch.lua:1-2]",
		"past the end [pob:src/Modules/Calc.lua:9999-10000]",
		"comment only [pob:src/Modules/Calc.lua:3-4]",
		"garbage range [pob:Calc.lua:not-a-number]",
		"templated %s omitted [go-absent:mod.OrNoSuchSymbolAnywhere]",
		"last lines [pob:src/Modules/Calc.lua:6-7]",
		"one past the end [pob:src/Modules/Calc.lua:7-8]",
		"quoted \"thing\" omitted [unverifiable: has a quote]",
		"test-only symbol [go-absent:mod.OrTestOnlySymbol]",
		"testdata-only symbol [go-absent:mod.OrTestdataOnlySymbol]",
	}
	var b strings.Builder
	b.WriteString("package main\n\n// a fixture emitter\n")
	for _, r := range reasons {
		fmt.Fprintf(&b, "\t\treturn \"\", mcUnrep(%q)\n", r)
	}
	return b.String()
}

func orGenerated() string {
	return strings.Join([]string{
		"package raw",
		"",
		"// Some mods are not representable and are omitted rather than mapped lossily:",
		"//   - 12 templated WIDGET omitted [go-absent:mod.OrNoSuchSymbolAnywhere]",
		"//   - 6 a reason no emitter site can produce [go-absent:mod.OrGoneForever]",
		"//   - 3 opt out properly [unverifiable: data-dependent]",
		"//   - 9 quoted \"thing\" omitted [unverifiable: has a quote]",
		"//   - 4 PREFIX templated WIDGET omitted [go-absent:mod.OrNoSuchSymbolAnywhere]",
		"// a plain comment, which closes the omission block",
		"//   - 99 this line is BELOW the block and must never be read",
		"",
		"var Table = map[string]string{}",
		"",
	}, "\n")
}

func orLua() string {
	return strings.Join([]string{
		"-- Calc.lua fixture",
		"local x = 1",
		"-- comment only",
		"",
		"function Calc:f(a)",
		"\treturn a",
		"end",
		"",
	}, "\n")
}

func orSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, orPy)
	pc := difftest.ReadOriginal(t, repo, pcPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			orPy:                                string(py),
			pcPy:                                string(pc),
			"scripts/gopob/skillstatmapemit.go": orEmitter(),
			"data/raw/skill_stat_maps.go":       orGenerated(),
			"pob/src/Modules/Calc.lua":          orLua(),
			"mod/common.go":                     "package mod\n\ntype OrFixtureSymbol struct{}\n",
			"mod/thing_test.go":                 "package mod\n\ntype OrTestOnlySymbol struct{}\n",
			"mod/testdata/sample.go":            "package testdata\n\ntype OrTestdataOnlySymbol struct{}\n",
			"moddb/db.go":                       "package moddb\n\nfunc Q() {}\n",
			"calculator/env.go":                 "package calculator\n\nfunc E() {}\n",
			"nopob/readme.txt":                  "not a PoB checkout\n",
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
		cmd := exec.CommandContext(t.Context(), "git", "init", "-q")
		cmd.Dir = sb
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("sandbox: git init: %w: %s", err, out)
		}
		return sb, nil
	}
}

func TestOmissionReasonsPortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := orSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, orPy)

	spec := difftest.Spec{
		Tool:         "omission-reasons",
		PythonScript: orPy,
		GoBin:        bin,
		GoArgs:       []string{"omission-reasons"},
		Env:          []string{"POB_DIR=pob"},
		Stderr:       difftest.StderrExact,
		RefusalSites: 2,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "pob-dir-is-not-a-checkout", Refusal: "REFUSING: not a PoB checkout (no src/): nopob",
				Break: func(string) error { return nil }},
			{Name: "emitter-missing",
				Refusal: "skillstatmapemit.go is not readable: No such file or directory",
				Break: func(sb string) error {
					return os.Remove(filepath.Join(sb, "scripts/gopob/skillstatmapemit.go"))
				}},
			{Name: "generated-table-missing",
				Refusal: "skill_stat_maps.go is not readable: No such file or directory",
				Break: func(sb string) error {
					return os.Remove(filepath.Join(sb, "data/raw/skill_stat_maps.go"))
				}},
			{Name: "emitter-unreadable", Refusal: "skillstatmapemit.go is not readable: Permission denied",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "scripts/gopob/skillstatmapemit.go"), 0o000)
				}},
			{Name: "go-source-not-utf8", Refusal: "common.go is not valid UTF-8 (Go source)",
				Break: func(sb string) error {
					return os.WriteFile(filepath.Join(sb, "mod/common.go"),
						[]byte("package mod\n\ntype OrFixtureSymbol struct{} // \xff\xfe\n"), 0o600)
				}},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "emitter-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			if f.Name == "pob-dir-is-not-a-checkout" {
				s.Env = []string{"POB_DIR=nopob"}
			}
			s.GoArgs = append([]string{"omission-reasons"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestOmissionReasonsReportValues: REPORT SUPPRESSES ON ANY NON-EMPTY VALUE.
func TestOmissionReasonsReportValues(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := orSandbox(t, repo)
	for _, v := range []string{"1", "0", "false", ""} {
		t.Run("REPORT="+v, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "omission-reasons",
				PythonScript: orPy,
				GoBin:        bin,
				GoArgs:       []string{"omission-reasons"},
				Env:          []string{"POB_DIR=pob", "REPORT=" + v},
				Stderr:       difftest.StderrExact,
			}, difftest.Fixture{Name: "report" + v}, mk)
		})
	}
}

func TestOmissionReasonsSelftest(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := orSandbox(t, repo)
	difftest.RunFixture(t, t.Context(), difftest.Spec{
		Tool:         "omission-reasons",
		PythonScript: orPy,
		PythonArgs:   []string{"--selftest"},
		GoBin:        bin,
		GoArgs:       []string{"omission-reasons", "--selftest"},
		Env:          []string{"POB_DIR=pob"},
		Stderr:       difftest.StderrExact,
	}, difftest.Fixture{Name: "selftest"}, mk)
}

func TestOmissionReasonsPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := orSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "omission-reasons",
		PythonScript: orPy,
		Env:          []string{"POB_DIR=pob", "REPORT=1"},
	}, mk, []string{"0", "12345", "999983"})
}

// TestOmissionReasonsEmitterWithNoReasons: AN EMITTER WITH NO Unrepresentable SITES AT ALL.
func TestOmissionReasonsEmitterWithNoReasons(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := orSandbox(t, repo)
	blank := func() (string, error) {
		sb, err := mk()
		if err != nil {
			return "", err
		}
		return sb, os.WriteFile(filepath.Join(sb, "scripts/gopob/skillstatmapemit.go"),
			[]byte("# an emitter that raises nothing at all\n"), 0o600)
	}
	difftest.RunFixture(t, t.Context(), difftest.Spec{
		Tool:         "omission-reasons",
		PythonScript: orPy,
		GoBin:        bin,
		GoArgs:       []string{"omission-reasons"},
		Env:          []string{"POB_DIR=pob"},
		Stderr:       difftest.StderrExact,
	}, difftest.Fixture{Name: "no-reasons"}, blank)
}
