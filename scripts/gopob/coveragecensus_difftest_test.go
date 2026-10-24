package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const ccPy = "scripts/coverage_census.py"

func ccEnvGo() string {
	return `package calculator

func Alpha() {
	if OutputModeCalcs == mode {
		return
	}
	x := 1
	if err != nil {
		return
	}
	if a != nil && b.Thing {
		return
	}
	for _, v := range items {
		if v.AppliesToMinions {
			doThing()
		}
	}
	if ok {
		fine()
	}
}
`
}

func ccFixtureSources() map[string]string {
	out := map[string]string{
		"calculator/env.go": ccEnvGo(),
		"calculator/configs.go": `package calculator

func Beta() {
	if !configChecked(val) {
		return
	}
	if !ok {
		return
	}
	somethingElse()
	andMore()
}
`,
		"calculator/mod_parser.go": `package calculator

func Gamma() {
	register("increased golem damage per summoned Minion", func() {
		unreached()
	})
}
`,
		"pob/utils.go": `package pob

func Delta() {
	frontendOnly()
}
`,
	}
	for i := range 14 {
		out[fmt.Sprintf("utils/u%02d.go", i)] = fmt.Sprintf(`package utils

func U%02d() {
	if somethingUnclassifiable {
		a()
		b()
	}
}
`, i)
	}
	out["utils/zbusiest.go"] = `package utils

func Busiest() {
	if one {
		a()
		b()
	}
	if two {
		a()
		b()
	}
	if three {
		a()
		b()
	}
}
`
	return out
}

func ccFixtureProfile() string {
	var b strings.Builder
	b.WriteString("mode: set\n")
	add := func(file string, sl, el, stmts, count int) {
		fmt.Fprintf(&b, "github.com/Vilsol/go-pob/%s:%d.2,%d.3 %d %d\n", file, sl, el, stmts, count)
	}
	add("calculator/env.go", 4, 6, 1, 0)
	add("calculator/env.go", 8, 10, 1, 0)
	add("calculator/env.go", 11, 13, 1, 0)
	add("calculator/env.go", 15, 17, 1, 0)
	add("calculator/env.go", 19, 21, 1, 0)
	add("calculator/configs.go", 4, 6, 1, 0)
	add("calculator/configs.go", 7, 9, 1, 0)
	add("calculator/configs.go", 10, 11, 2, 0)
	add("calculator/mod_parser.go", 4, 6, 1, 0)
	add("pob/utils.go", 4, 4, 1, 0)
	for i := range 14 {
		add(fmt.Sprintf("utils/u%02d.go", i), 4, 7, 2, 0)
	}
	add("utils/zbusiest.go", 4, 7, 2, 0)
	add("utils/zbusiest.go", 8, 11, 2, 0)
	add("utils/zbusiest.go", 12, 15, 2, 0)
	add("calculator/env.go", 22, 22, 1, 3)
	add("calculator/env.go", 22, 22, 1, 0)
	return b.String()
}

func ccFixtureFuncs() string {
	var b strings.Builder
	for _, f := range []string{"calculator/env.go:3:\tAlpha\t40.0%",
		"calculator/configs.go:3:\tBeta\t0.0%", "calculator/mod_parser.go:3:\tGamma\t0.0%",
		"pob/utils.go:3:\tDelta\t0.0%"} {
		fmt.Fprintf(&b, "github.com/Vilsol/go-pob/%s\n", f)
	}
	for i := range 14 {
		fmt.Fprintf(&b, "github.com/Vilsol/go-pob/utils/u%02d.go:3:\tU%02d\t0.0%%\n", i, i)
	}
	fmt.Fprintf(&b, "github.com/Vilsol/go-pob/utils/zbusiest.go:3:\tBusiest\t0.0%%\n")
	b.WriteString("total:\t(statements)\t12.5%\n")
	return b.String()
}

func ccCensusSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	files := ccFixtureSources()
	py := difftest.ReadOriginal(t, repo, ccPy)
	files[ccPy] = string(py)
	files["cov.out"] = ccFixtureProfile()
	files["cov.func"] = ccFixtureFuncs()
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

func TestCoverageCensusPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ccCensusSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, ccPy)

	base := []string{"--profile", "cov.out", "--funcs", "cov.func", "--root", "."}
	spec := difftest.Spec{
		Tool:         "coverage-census",
		PythonScript: ccPy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"coverage-census"}, base...),
		Stderr:       difftest.StderrExact,
		RefusalSites: 8,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-profile-flag", Refusal: "coverage-census: --profile is required",
				Break: func(string) error { return nil }},
			{Name: "unknown-argument", Refusal: "coverage-census: unknown argument '--bogus'",
				Break: func(string) error { return nil }},
			{Name: "flag-without-value", Refusal: "coverage-census: --bucket needs a value",
				Break: func(string) error { return nil }},

			{Name: "profile-unreadable", Refusal: "coverage-census: cannot read cov.out",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "cov.out"), 0o000)
				}},
			{Name: "funcs-unreadable", Refusal: "coverage-census: cannot read cov.func",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "cov.func"), 0o000)
				}},
			{Name: "source-missing", Refusal: "coverage-census: cannot read pob/utils.go",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "pob/utils.go")) }},

			{Name: "unparsed-profile-line", Refusal: "cov.out: unparsed profile line: this is not a block",
				Break: func(sb string) error {
					b, err := os.ReadFile(filepath.Join(sb, "cov.out")) //nolint:gosec // harness temp
					if err != nil {
						return fmt.Errorf("fixture: %w", err)
					}
					return put(sb, "cov.out", string(b)+"this is not a block\n")
				}},

			{Name: "main-funnel-never-refuses", NoFixture: true,
				Refusal: "sys.exit(main()) where main() only ever returns 0",
				Note: "a lexical exit carrying no non-zero status; every real refusal in this " +
					"tool exits through its own sys.exit, above"},

			{Name: "list-and-bucket", Passes: true, Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		if f.NoFixture {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			if (f.Name == "profile-unreadable" || f.Name == "funcs-unreadable") && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			switch f.Name {
			case "no-profile-flag":
				s.PythonArgs = []string{"--root", "."}
			case "unknown-argument":
				s.PythonArgs = append(append([]string{}, base...), "--bogus")
			case "flag-without-value":
				s.PythonArgs = append(append([]string{}, base...), "--bucket")
			case "list-and-bucket":
				s.PythonArgs = append(append([]string{}, base...), "--list", "--bucket", "data")
			}
			if s.PythonArgs != nil {
				s.GoArgs = append([]string{"coverage-census"}, s.PythonArgs...)
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestCoverageCensusOverlayAndJSONMatch: THE OVERLAY AND THE JSON ARE SEPARATE PROGRAMS AS FAR AS COVERAGE GOES.
func TestCoverageCensusOverlayAndJSONMatch(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ccCensusSandbox(t, repo)

	for _, tc := range []struct {
		name  string
		args  []string
		files []string
	}{
		{"overlay", []string{"--profile", "cov.out", "--overlay", "cov.out", "--funcs", "cov.func",
			"--root", ".", "--list"}, nil},
		{"json", []string{"--profile", "cov.out", "--funcs", "cov.func", "--root", ".",
			"--json", "out.json"}, []string{"out.json"}},
		{"json-with-overlay", []string{"--profile", "cov.out", "--overlay", "cov.out",
			"--funcs", "cov.func", "--root", ".", "--json", "out.json"}, []string{"out.json"}},
		{"no-funcs", []string{"--profile", "cov.out", "--root", ".", "--list"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "coverage-census",
				PythonScript: ccPy,
				PythonArgs:   tc.args,
				GoBin:        bin,
				GoArgs:       append([]string{"coverage-census"}, tc.args...),
				Stderr:       difftest.StderrExact,
				Files:        tc.files,
			}, difftest.Fixture{Name: tc.name}, mk)
		})
	}
}

func TestCoverageCensusPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := ccCensusSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "coverage-census",
		PythonScript: ccPy,
		PythonArgs:   []string{"--profile", "cov.out", "--funcs", "cov.func", "--root", ".", "--list"},
	}, mk, []string{"0", "12345", "999983"})
}
