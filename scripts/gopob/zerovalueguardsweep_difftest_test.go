package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const zvPy = "scripts/zero_value_guard_sweep.py"

func zvFiles() map[string]string {
	m := map[string]string{
		"go.mod": "module github.com/Vilsol/go-pob\n\ngo 1.26.2\n",
		"calculator/a.go": `package calculator

func A() {
	if x.Busiest > 0 {
		one()
	}
	if y.Busiest != 0 {
		two()
	}
	if z.Busiest > 0 {
		three()
	}
	// if c.Commented > 0 { skipped, it is a comment }
	 * if s.Starred > 0 { also skipped }
	if q.TieAlpha > 0 {
		a()
	}
	if q.TieBravo != 0 {
		b()
	}
	if q.NoLuaCounterpart > 0 {
		c()
	}
}
`,
		"moddb/b.go": `package moddb

func B() {
	if m.TieCharlie > 0 {
		a()
	}
	if m.TieDelta != 0 {
		b()
	}
	if m.TieEcho > 0 {
		c()
	}
	if m.TieFoxtrot != 0 {
		d()
	}
}
`,
		"calculator/a_test.go": `package calculator

func TestX() {
	if t.OnlyInATest > 0 {
		a()
	}
}
`,
	}
	names := make([]string, 0, 21)
	names = append(names, "busiest", "tieAlpha", "tieBravo", "tieCharlie", "tieDelta",
		"tieEcho", "tieFoxtrot", "onlyInATest")
	extra := ""
	for i := range 12 {
		f := fmt.Sprintf("TieExtra%02d", i)
		extra += fmt.Sprintf("\tif m.%s > 0 {\n\t\ta()\n\t}\n", f)
		names = append(names, fmt.Sprintf("tieExtra%02d", i))
	}
	extra += "\tif m.Latecomer > 0 {\n\t\ta()\n\t}\n" +
		"\tif m.Latecomer != 0 {\n\t\tb()\n\t}\n" +
		"\tif m.Latecomer > 0 {\n\t\tc()\n\t}\n"
	names = append(names, "latecomer")
	m["moddb/c.go"] = "package moddb\n\nfunc C() {\n" + extra + "}\n"
	lua := "local x\n"
	for _, f := range names {
		lua += fmt.Sprintf("if a.%s then end\n", f)
	}
	m["pob/src/Modules/One.lua"] = lua
	m["pob/src/Classes/Two.lua"] = "if b.busiest then end\nif c.busiest and z then end\n" +
		"if d.busiest or w then end\n"
	return m
}

func zvSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, zvPy)
	files := zvFiles()
	files[zvPy] = string(py)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
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

func TestZeroValueGuardSweepPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := zvSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, zvPy)

	spec := difftest.Spec{
		Tool:         "zero-value-guard-sweep",
		PythonScript: zvPy,
		GoBin:        bin,
		GoArgs:       []string{"zero-value-guard-sweep"},
		Env:          []string{"POB_DIR=pob"},
		Stderr:       difftest.StderrExact,
		RefusalSites: 4,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-go-mod", Refusal: "no go.mod declaring module github.com/Vilsol/go-pob",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "go.mod")) }},
			{Name: "bad-gopob-root", Refusal: "GOPOB_ROOT does not contain a go.mod declaring module",
				Break: func(string) error { return nil }},

			{Name: "no-lua-found", Refusal: "REFUSING: no Lua presence tests found under",
				Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "pob")) }},
			{Name: "no-go-guards-found", Refusal: "REFUSING: no `> 0` / `!= 0` field guards found under",
				Break: func(sb string) error {
					if err := os.RemoveAll(filepath.Join(sb, "calculator")); err != nil {
						return fmt.Errorf("fixture: %w", err)
					}
					return os.RemoveAll(filepath.Join(sb, "moddb"))
				}},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			if f.Name == "bad-gopob-root" {
				s.Env = append(append([]string{}, spec.Env...), "GOPOB_ROOT=pob")
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

func TestZeroValueGuardSweepPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := zvSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "zero-value-guard-sweep",
		PythonScript: zvPy,
		Env:          []string{"POB_DIR=pob"},
	}, mk, []string{"0", "12345", "999983"})
}
