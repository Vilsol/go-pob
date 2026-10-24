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

const otcPy = "scripts/optional_token_census.py"

func otcSandbox(t *testing.T, repo, pob string) func() (string, error) {
	t.Helper()
	pin, err := os.ReadFile(filepath.Join(pob, "src/Modules/ModParser.lua"))
	if err != nil {
		t.Skipf("pinned PoB ModParser.lua not readable: %v", err)
	}
	ours, err := os.ReadFile(filepath.Join(repo, "calculator/mod_parser.go"))
	if err != nil {
		t.Fatal(err)
	}
	py := difftest.ReadOriginal(t, repo, otcPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		for rel, body := range map[string][]byte{otcPy: py, trcPin: pin, trcOurs: ours} {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, body, 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func TestOptionalTokenCensusPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	pob := pobDir(t)
	bin := buildPort(t, repo)
	mk := otcSandbox(t, repo, pob)

	pySrc := difftest.ReadOriginal(t, repo, otcPy)

	args := []string{trcPin, trcOurs}
	spec := difftest.Spec{
		Tool:         "optional-token-census",
		PythonScript: otcPy,
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"optional-token-census"}, args...),
		Stderr:       difftest.StderrExact,
		RefusalSites: 6,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "REFUSING: usage: optional_token_census.py",
				Break: func(string) error { return nil }},

			{Name: "pin-missing", Refusal: trcPin + " is not readable:",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, trcPin)) }},
			{Name: "pin-unreadable", Refusal: trcPin + " is not readable:",
				Break: func(sb string) error { return os.Chmod(filepath.Join(sb, trcPin), 0o000) }},

			{Name: "pin-invalid-utf8", Refusal: trcPin + " is not valid UTF-8",
				Break: func(sb string) error {
					return os.WriteFile(filepath.Join(sb, trcPin),
						[]byte("\t[\"a\xff\xfe row\"] = 1,\n"), 0o600)
				}},

			{Name: "pin-499-keys", Refusal: "only 499 keys parsed from the pin",
				Break: func(sb string) error { return put(sb, trcPin, synthPin(499)) }},

			{Name: "expander-lost-the-known-case", NoFixture: true,
				Refusal: "the known optional-token case did not expand",
				Note: "guards maximalExpand against itself and reads no input; covered by " +
					"TestMaximalExpandMatchesPythonOnEveryPinKey over all 4206 pin keys"},
			{Name: "expander-altered-a-plain-key", NoFixture: true,
				Refusal: "a key with no optional was altered by the expander",
				Note:    "same: an implementation self-check, unreachable from any sandbox state"},

			{Name: "pin-500-keys", Passes: true,
				Break: func(sb string) error { return put(sb, trcPin, synthPin(500)) }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		if f.NoFixture {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "pin-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			if f.Name == "no-args" {
				s.PythonArgs, s.GoArgs = nil, []string{"optional-token-census"}
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestMaximalExpandMatchesPythonOnEveryPinKey: THE ONE CONSTRUCT IN THE PILOT THAT COULD NOT BE PORTED CHARACTER FOR CHARACTER, SO IT OWES THE STRONGEST CONTROL.
func TestMaximalExpandMatchesPythonOnEveryPinKey(t *testing.T) {
	t.Parallel()
	pob := pobDir(t)
	pin, err := os.ReadFile(filepath.Join(pob, "src/Modules/ModParser.lua"))
	if err != nil {
		t.Skipf("pinned PoB ModParser.lua not readable: %v", err)
	}
	matches := rePinKey.FindAllStringSubmatch(string(pin), -1)
	keys := make([]string, 0, len(matches))
	for _, m := range matches {
		keys = append(keys, m[1])
	}
	if len(keys) < 500 {
		t.Fatalf("only %d keys; the extraction is broken and this control means nothing", len(keys))
	}

	script := filepath.Join(t.TempDir(), "expand.py")
	if err := os.WriteFile(script, []byte(`import re, sys
for k in sys.stdin.read().split("\x00")[:-1]:
    print(("" if re.search(r"%[adws]\?", k) else re.sub(r"(?<!%)(.)\?", r"\1", k)).replace("\n", "\\n"))
`), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "python3", script)
	cmd.Stdin = strings.NewReader(strings.Join(keys, "\x00") + "\x00")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running the python reference: %v", err)
	}
	want := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	if len(want) != len(keys) {
		t.Fatalf("reference produced %d lines for %d keys", len(want), len(keys))
	}

	expanded, differing := 0, 0
	for i, k := range keys {
		got := strings.ReplaceAll(maximalExpand(k), "\n", `\n`)
		if got != k {
			expanded++
		}
		if got != want[i] {
			differing++
			if differing <= 5 {
				t.Errorf("key %q\n  go     %q\n  python %q", k, got, want[i])
			}
		}
	}
	if expanded == 0 {
		t.Fatal("VACUOUS: the expander altered none of the keys, so `return k` would pass")
	}
	t.Logf("%d keys, %d altered by the expander, %d differing from python", len(keys), expanded, differing)
}
