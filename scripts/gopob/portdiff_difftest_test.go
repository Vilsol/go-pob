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

const pdPy = "scripts/port_diff.py"

func pdLua() string {
	return strings.Join([]string{
		"-- ConfigOptions.lua fixture",
		"local pdUniqueToken = 1",
		"function Cfg:doThing(a)",
		"\treturn a + pdUniqueToken",
		"end",
		"-- pdSharedName appears here",
		"local pdSharedName = 2",
		"local tail = 3",
		"",
	}, "\n")
}

func pdLuaBusy() string {
	var b strings.Builder
	b.WriteString("-- ModCache.lua fixture: the busiest file\n")
	for i := range 5 {
		fmt.Fprintf(&b, "local pdSharedName_%d = %d\n", i, i)
	}
	return b.String()
}

func pdLuaTied(tag string) string {
	return fmt.Sprintf("-- %s: two hits, tied with its sibling\nlocal pdSharedName = 1\n"+
		"local pdSharedName2 = 2\n", tag)
}

func pdLuaWide() string {
	return "-- wide.lua\nlocal pdSharedName = \"" + strings.Repeat("ü", 60) + "\"\n"
}

func pdGoSrc() string {
	return strings.Join([]string{
		"package a",
		"",
		"// pdSharedName is mentioned here too.",
		"//pob:src/Modules/ConfigOptions.lua:3-5",
		"func PdCited() {",
		"\t_ = 1",
		"}",
		"",
		"// this one carries no marker at all",
		"func PdUncited() {",
		"\t_ = 2",
		"}",
		"",
		"//pob:src/Modules/NoSuchFile.lua:1-2",
		"func PdCitesMissingLua() {",
		"\t_ = 3",
		"}",
		"",
		"// ⚠️ A ONE-LINE FUNC, whose opening line is already brace-balanced. The body scan breaks",
		"// on `depth <= 0 && j > start`; without the second clause it would stop on the DECLARATION",
		"// of every ordinary func too, and only a balanced first line tells the two apart.",
		"func PdOneLine() { _ = 4 }",
		"",
		"// more pdSharedName mentions, so the Go side has enough hits to truncate",
		"func PdMentions() {",
		"\t_ = \"pdSharedName one\"",
		"\t_ = \"pdSharedName two\"",
		"\t_ = \"pdSharedName three\"",
		"}",
		"",
	}, "\n")
}

func pdSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, pdPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			pdPy:                                string(py),
			"go.mod":                            "module github.com/Vilsol/go-pob\n\ngo 1.24\n",
			"a/thing.go":                        pdGoSrc(),
			"a/thing_test.go":                   "package a\n\n// pdSharedName in a test file\nfunc TestX() {}\n",
			"frontend/src/gen.go":               "package frontend\n\n// pdSharedName in frontend\nfunc F() {}\n",
			"pob/src/Modules/ConfigOptions.lua": pdLua(),
			"pob/src/Data/ModCache.lua":         pdLuaBusy(),
			"pob/src/Aaa/Tied.lua":              pdLuaTied("Aaa/Tied.lua"),
			"pob/src/Zzz/Tied.lua":              pdLuaTied("Zzz/Tied.lua"),
			"pob/src/Modules/Wide.lua":          pdLuaWide(),
			"pob/src/Zz/a.lua":                  "-- Zz/a.lua\nlocal pdSharedName = 1\n",
			"pob/src/Zz.b/a.lua":                "-- Zz.b/a.lua\nlocal pdSharedName = 1\n",
			"emptypob/readme.txt":               "no src/ here\n",
			"srcless/src/readme.txt":            "a src/ with no .lua in it\n",
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
		for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}} {
			cmd := exec.CommandContext(t.Context(), "git", args...)
			cmd.Dir = sb
			if out, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("sandbox: git %s: %w: %s", args[0], err, out)
			}
		}
		return sb, nil
	}
}

func TestPortDiffPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pdSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, pdPy)

	spec := difftest.Spec{
		Tool:         "port-diff",
		PythonScript: pdPy,
		GoBin:        bin,
		GoArgs:       []string{"port-diff"},
		Env:          []string{"POB_DIR=pob", "NAME=pdSharedName"},
		Stderr:       difftest.StderrExact,
		RefusalSites: 2,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-mode", Refusal: "REFUSING: pass exactly one of NAME=, SYMBOL=, FUNC=, LUA=, GREP=",
				Break: func(string) error { return nil }},
			{Name: "two-modes", Refusal: "REFUSING: pass exactly one of NAME=, SYMBOL=, FUNC=, LUA=, GREP=",
				Break: func(string) error { return nil }},
			{Name: "context-not-an-integer", Refusal: "REFUSING: CONTEXT='abc' is not an integer",
				Break: func(string) error { return nil }},
			{Name: "max-hits-not-an-integer", Refusal: "REFUSING: MAX_HITS='lots' is not an integer",
				Break: func(string) error { return nil }},
			{Name: "pob-dir-missing", Refusal: "REFUSING: pinned PoB not found at nope",
				Break: func(string) error { return nil }},
			{Name: "pob-has-no-src", Refusal: "REFUSING: no Lua under emptypob/src",
				Break: func(string) error { return nil }},
			{Name: "src-holds-no-lua", Refusal: "REFUSING: no Lua under srcless/src",
				Break: func(string) error { return nil }},
			{Name: "func-not-found", Refusal: "REFUSING: no Go func named 'PdNoSuchFunc'",
				Break: func(string) error { return nil }},
			{Name: "lua-spec-malformed", Refusal: "REFUSING: LUA='bogus' is not `src/<path>.lua:<start>[-<end>]`",
				Break: func(string) error { return nil }},
			{Name: "lua-file-missing", Refusal: "does not exist — set POB_DIR",
				Break: func(string) error { return nil }},
			{Name: "grep-invalid-regex", Refusal: "REFUSING: GREP='[unclosed' is not a valid regex",
				Break: func(string) error { return nil }},
			{Name: "grep-no-hits", Refusal: "REFUSING: 'ZzzNothingMatchesThis' matches nothing",
				Break: func(string) error { return nil }},
			{Name: "symbol-on-neither-side", Refusal: "REFUSING: 'ZzzNeitherSide' appears on NEITHER side.",
				Break: func(string) error { return nil }},
			{Name: "name-on-neither-side", Refusal: "REFUSING: 'ZzzNeitherSide' appears on NEITHER side — far more likely",
				Break: func(string) error { return nil }},
			{Name: "lua-not-utf8", Refusal: "ConfigOptions.lua is not valid UTF-8",
				Break: func(sb string) error {
					bad := strings.Replace(pdLua(), "local tail = 3", "local tail = \xff3", 1)
					return os.WriteFile(filepath.Join(sb, "pob/src/Modules/ConfigOptions.lua"),
						[]byte(bad), 0o600)
				}},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	envFor := map[string][]string{
		"no-mode":                 {"POB_DIR=pob"},
		"two-modes":               {"POB_DIR=pob", "NAME=x", "SYMBOL=y"},
		"context-not-an-integer":  {"POB_DIR=pob", "NAME=pdSharedName", "CONTEXT=abc"},
		"max-hits-not-an-integer": {"POB_DIR=pob", "NAME=pdSharedName", "MAX_HITS=lots"},
		"pob-dir-missing":         {"POB_DIR=nope", "NAME=pdSharedName"},
		"pob-has-no-src":          {"POB_DIR=emptypob", "NAME=pdSharedName"},
		"src-holds-no-lua":        {"POB_DIR=srcless", "NAME=pdSharedName"},
		"func-not-found":          {"POB_DIR=pob", "FUNC=PdNoSuchFunc"},
		"lua-spec-malformed":      {"POB_DIR=pob", "LUA=bogus"},
		"lua-file-missing":        {"POB_DIR=pob", "LUA=src/Modules/Gone.lua:1-2"},
		"grep-invalid-regex":      {"POB_DIR=pob", "GREP=[unclosed"},
		"grep-no-hits":            {"POB_DIR=pob", "GREP=ZzzNothingMatchesThis"},
		"symbol-on-neither-side":  {"POB_DIR=pob", "SYMBOL=ZzzNeitherSide"},
		"name-on-neither-side":    {"POB_DIR=pob", "NAME=ZzzNeitherSide"},
	}

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			if env, ok := envFor[f.Name]; ok {
				s.Env = env
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestPortDiffResolvesHomeLikePython: HOME no longer picks the default PoB; with or without it the tool looks
// beside the repository (none here: GIT_DIR hides the sandbox repo, so the path is stable) and refuses.
func TestPortDiffResolvesHomeLikePython(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{"home-empty", []string{"HOME=", "NAME=pdSharedName", "GIT_DIR=no-git"}},
		{"home-absent", []string{"NAME=pdSharedName", "GIT_DIR=no-git"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := difftest.Spec{
				Tool: "port-diff", PythonScript: pdPy, GoBin: bin,
				GoArgs: []string{"port-diff"},
				Env:    tc.env, EnvUnset: []string{"POB_DIR"},
				Stderr: difftest.StderrExact,
			}
			if tc.name == "home-absent" {
				s.EnvUnset = append(s.EnvUnset, "HOME")
			}
			difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: tc.name},
				pdSandbox(t, repo))
		})
	}
}

func TestPortDiffModes(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pdSandbox(t, repo)
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{"name", []string{"POB_DIR=pob", "NAME=pdSharedName"}},
		{"symbol", []string{"POB_DIR=pob", "SYMBOL=pdSharedName"}},
		{"func-cited", []string{"POB_DIR=pob", "FUNC=PdCited"}},
		{"func-uncited", []string{"POB_DIR=pob", "FUNC=PdUncited"}},
		{"func-cites-missing-lua", []string{"POB_DIR=pob", "FUNC=PdCitesMissingLua"}},
		{"func-one-line", []string{"POB_DIR=pob", "FUNC=PdOneLine"}},
		{"lua-range", []string{"POB_DIR=pob", "LUA=src/Modules/ConfigOptions.lua:3-5"}},
		{"lua-single-line", []string{"POB_DIR=pob", "LUA=src/Modules/ConfigOptions.lua:3"}},
		{"lua-past-eof", []string{"POB_DIR=pob", "LUA=src/Modules/ConfigOptions.lua:7-999"}},
		{"grep", []string{"POB_DIR=pob", "GREP=pdUnique.oken"}},
		{"maxhits-truncating", []string{"POB_DIR=pob", "NAME=pdSharedName", "MAX_HITS=2"}},
		{"maxhits-at-the-count", []string{"POB_DIR=pob", "NAME=pdSharedName", "MAX_HITS=3"}},
		{"context-zero", []string{"POB_DIR=pob", "SYMBOL=pdSharedName", "CONTEXT=0"}},
		{"context-wide", []string{"POB_DIR=pob", "SYMBOL=pdSharedName", "CONTEXT=99"}},
		{"symbol-all-hits", []string{"POB_DIR=pob", "SYMBOL=pdSharedName", "MAX_HITS=99"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "port-diff",
				PythonScript: pdPy,
				GoBin:        bin,
				GoArgs:       []string{"port-diff"},
				Env:          tc.env,
				Stderr:       difftest.StderrExact,
			}, difftest.Fixture{Name: tc.name}, mk)
		})
	}
}

func TestPortDiffPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := pdSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "port-diff",
		PythonScript: pdPy,
		Env:          []string{"POB_DIR=pob", "NAME=pdSharedName"},
	}, mk, []string{"0", "12345", "999983"})
}
