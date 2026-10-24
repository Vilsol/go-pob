package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The same relative Lua path in two trees: //pob2: resolves on POB2_DIR, //pob: on POB_DIR.
// pob2Lua is kept apart from its line ranges so the citation sweep does not read this file.
const pob2Lua = "src/Modules/CalcSetup.lua"

func pob2Sandbox(t *testing.T, repo string) string {
	t.Helper()
	sb := t.TempDir()
	td := filepath.Join(repo, "scripts/gopob/testdata/pob2marker")
	for dst, src := range map[string]string{
		"calculator/marked.go":           "marked.go.txt",
		"poe1/src/Modules/CalcSetup.lua": "poe1_calcsetup.lua.txt",
		"poe2/src/Modules/CalcSetup.lua": "poe2_calcsetup.lua.txt",
		"scripts/port_manifest.txt":      "",
		"go.mod":                         "",
	} {
		var body []byte
		switch {
		case src != "":
			b, err := os.ReadFile(filepath.Join(td, src))
			if err != nil {
				t.Fatal(err)
			}
			body = b
		case dst == "scripts/port_manifest.txt":
			body = []byte("calculator/marked.go  PORT  src/Modules/CalcSetup.lua\n")
		case dst == "go.mod":
			body = []byte("module github.com/Vilsol/go-pob\n\ngo 1.26.2\n")
		}
		full := filepath.Join(sb, dst)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "calculator/marked.go"}} {
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = sb
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return sb
}

func pob2Run(t *testing.T, bin, sb string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), bin, args...)
	cmd.Dir = sb
	cmd.Env = append(append(os.Environ(), "GOPOB_ROOT="+sb), env...)
	out, _ := cmd.CombinedOutput()
	return string(out)
}

func TestPob2MarkersResolveAgainstThePoE2Pin(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	sb := pob2Sandbox(t, repo)
	poe1, poe2 := filepath.Join(sb, "poe1"), filepath.Join(sb, "poe2")
	pins := func(first, second string) []string {
		return []string{"POB_DIR=" + first, "POB2_DIR=" + second, "REPORT=1"}
	}

	t.Run("citations: each marker on its own pin", func(t *testing.T) {
		out := pob2Run(t, bin, sb, pins(poe1, poe2), "pob-citations")
		if !strings.Contains(out, "2 citation(s) on 2 symbol(s)") || !strings.Contains(out, "  0 broken\n") {
			t.Fatalf("want both markers to resolve, got:\n%s", out)
		}
	})
	t.Run("citations: the pob2 marker fails against the PoE1 pin", func(t *testing.T) {
		out := pob2Run(t, bin, sb, pins(poe1, poe1), "pob-citations")
		if !strings.Contains(out, "BROKEN calculator/marked.go:6  pob2:"+pob2Lua+":5-7  RANGE") ||
			!strings.Contains(out, "  1 broken\n") {
			t.Fatalf("want only the pob2 marker broken, got:\n%s", out)
		}
	})
	t.Run("citations: a pob marker never moves to the PoE2 pin", func(t *testing.T) {
		out := pob2Run(t, bin, sb, pins(poe2, poe2), "pob-citations")
		if !strings.Contains(out, "BROKEN calculator/marked.go:3  "+pob2Lua+":1-3  EMPTY") ||
			!strings.Contains(out, "  1 broken\n") {
			t.Fatalf("want only the pob marker broken, got:\n%s", out)
		}
	})
	t.Run("citation-audit reads each marker from its own pin", func(t *testing.T) {
		out := pob2Run(t, bin, sb, append(pins(poe1, poe2), "FILE=calculator/marked.go", "CONTEXT=0"),
			"citation-audit")
		if !strings.Contains(out, "pob2:"+pob2Lua+":5-7") ||
			!strings.Contains(out, "secondGameSetup") || !strings.Contains(out, "firstGameSetup") {
			t.Fatalf("want both pins' Lua shown, got:\n%s", out)
		}
	})
	t.Run("port-diff FUNC follows the pob2 marker", func(t *testing.T) {
		out := pob2Run(t, bin, sb, append(pins(poe1, poe2), "FUNC=SetupSecondGame"), "port-diff")
		if !strings.Contains(out, "secondGameSetup") || strings.Contains(out, "firstGameSetup") {
			t.Fatalf("want the PoE2 Lua only, got:\n%s", out)
		}
	})
	t.Run("port-diff LUA with a pob2 prefix reads the PoE2 pin", func(t *testing.T) {
		spec := pob2Lua + ":5-7"
		out := pob2Run(t, bin, sb, append(pins(poe1, poe2), "LUA=pob2:"+spec), "port-diff")
		if !strings.Contains(out, "secondGameSetup") {
			t.Fatalf("want the PoE2 Lua, got:\n%s", out)
		}
		out = pob2Run(t, bin, sb, append(pins(poe1, poe2), "LUA="+pob2Lua+":1-3"), "port-diff")
		if !strings.Contains(out, "firstGameSetup") {
			t.Fatalf("want the PoE1 Lua without the prefix, got:\n%s", out)
		}
	})
}
