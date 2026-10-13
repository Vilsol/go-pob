package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const ppPy = "scripts/pob_pin.py"

func ppSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, ppPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			ppPy:                          string(py),
			"pob/src/Modules/Main.lua":    "-- Main\n",
			"pob/src/Launch.lua":          "-- Launch\n",
			"pob/src/HeadlessWrapper.lua": "-- Headless\n",
			"pob/runtime/lua/dkjson.lua":  "-- dkjson\n",
			"decoy/Launch.lua":            "-- installed\n",
			"decoy/Modules/keep":          "not a source checkout\n",
			"nomarkers/readme.txt":        "a directory with no PoB markers at all\n",
			"afile":                       "a file where a directory was expected\n",
			"scripts/.keep":               "",
		}
		for rel, b := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(b), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		const when = "2000-01-01T00:00:00 +0000"
		for _, args := range [][]string{
			{"init", "-q"}, {"add", "-A"},
			{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "pin"},
		} {
			cmd := exec.CommandContext(t.Context(), "git", args...) //nolint:gosec // the harness's own sandbox path
			cmd.Dir = filepath.Join(sb, "pob")
			cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+when, "GIT_COMMITTER_DATE="+when)
			if out, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("sandbox: git %s: %w: %s", args[0], err, out)
			}
		}
		rec := exec.CommandContext(t.Context(), buildPort(t, repo), "pob-pin", "--record") //nolint:gosec // the harness's own binary
		rec.Dir = sb
		rec.Env = append(os.Environ(), "POB_DIR="+filepath.Join(sb, "pob"))
		if out, err := rec.CombinedOutput(); err != nil {
			return "", fmt.Errorf("sandbox: --record: %w: %s", err, out)
		}
		return sb, nil
	}
}

func ppPut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func ppSpec(t *testing.T, bin, sbPobRel string, args ...string) difftest.Spec {
	t.Helper()
	return difftest.Spec{
		Tool: "pob-pin", PythonScript: ppPy, PythonArgs: args,
		GoBin: bin, GoArgs: append([]string{"pob-pin"}, args...),
		Stderr:          difftest.StderrExact,
		RefusalOnStdout: true,
		Env:             []string{"POB_DIR=" + sbPobRel},
		RefusalSites:    1,
	}
}

func TestPobPin(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ppSandbox(t, repo)
	pySrc := difftest.ReadOriginal(t, repo, ppPy)

	spec := ppSpec(t, bin, "pob", "")
	spec.PythonArgs = nil
	spec.GoArgs = []string{"pob-pin"}
	spec.Fixtures = []difftest.Fixture{
		{Name: "happy", Passes: true},

		{Name: "no-checkout", Refusal: "REFUSING (NO-CHECKOUT):",
			Break: func(string) error { return nil }},
		{Name: "a-file-not-a-directory", Refusal: "REFUSING (NO-CHECKOUT):",
			Break: func(string) error { return nil }},
		{Name: "the-installed-pob-decoy", Refusal: "This looks like an INSTALLED PoB",
			Break: func(string) error { return nil }},
		{Name: "no-markers-at-all", Refusal: "REFUSING (NOT-A-CHECKOUT):",
			Break: func(string) error { return nil }},
		{Name: "checkout-has-no-git", Refusal: "so its tracked file set cannot be determined",
			Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "pob", ".git")) }},

		{Name: "record-missing", Refusal: "the pin has no recorded identity",
			Break: func(sb string) error { return os.Remove(filepath.Join(sb, "scripts/pob_pin.json")) }},
		{Name: "record-malformed", Refusal: "is malformed (not valid JSON)",
			Break: func(sb string) error { return ppPut(sb, "scripts/pob_pin.json", "{ nope") }},
		{Name: "record-is-not-an-object", Refusal: "is not a JSON object",
			Break: func(sb string) error { return ppPut(sb, "scripts/pob_pin.json", "[]") }},
		{Name: "record-has-no-content-sha", Refusal: "has no 'content_sha256' field",
			Break: func(sb string) error {
				return ppPut(sb, "scripts/pob_pin.json", `{"scope": ["src"], "file_count": 1}`)
			}},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			switch f.Name {
			case "no-checkout":
				s.Env = []string{"POB_DIR=nope"}
			case "a-file-not-a-directory":
				s.Env = []string{"POB_DIR=afile"}
			case "the-installed-pob-decoy":
				s.Env = []string{"POB_DIR=decoy"}
			case "no-markers-at-all":
				s.Env = []string{"POB_DIR=nomarkers"}
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestPobPinMismatch: THE MISMATCH PATH IS THE GATE'S VERDICT.
func TestPobPinMismatch(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	s := ppSpec(t, bin, "pob")
	s.PythonArgs = nil
	s.GoArgs = []string{"pob-pin"}
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{
		Name: "content-moved", Finding: true,
		Break: func(sb string) error {
			return ppPut(sb, "pob/src/Launch.lua", "-- Launch, edited\n")
		},
	}, ppSandbox(t, repo))
}

// TestPobPinSelftest: THE SELFTEST IS PORTED RATHER THAN DROPPED.
func TestPobPinSelftest(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	s := ppSpec(t, bin, "pob", "--selftest")
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "selftest", Passes: true},
		ppSandbox(t, repo))
}
