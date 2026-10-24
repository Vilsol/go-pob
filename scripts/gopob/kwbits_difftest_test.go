package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

const pySource = "scripts/kw_bits_parity.py"

var sandboxFiles = []string{
	"go.mod",
	pySource,
	"mod/common.go",
	"oracle/mod_common.py",
}

func repoRoot(t *testing.T) string {
	t.Helper()
	r, err := root.Resolve()
	if err != nil {
		t.Fatalf("cannot resolve the repo root: %v", err)
	}
	return r
}

func pobDir(t *testing.T) string {
	t.Helper()
	return pobPinDir()
}

var (
	portBinMu  sync.Mutex
	portBins   = map[string]string{}
	portBinDir string
)

// buildPort builds the tool once per repo per test process; every test then runs that binary.
func buildPort(t *testing.T, repo string) string {
	t.Helper()
	portBinMu.Lock()
	defer portBinMu.Unlock()
	if bin, ok := portBins[repo]; ok {
		return bin
	}
	if portBinDir == "" {
		dir, err := os.MkdirTemp("", "gopob-port-") //nolint:usetesting // shared by every test in the process; TestMain removes it
		if err != nil {
			t.Fatalf("building the port: %v", err)
		}
		portBinDir = dir
	}
	bin := filepath.Join(portBinDir, strconv.Itoa(len(portBins)), difftest.ExeName("gopob"))
	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", bin, "./scripts/gopob")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the port: %v\n%s", err, out)
	}
	portBins[repo] = bin
	return bin
}

func sandboxFactory(t *testing.T, repo, pob string) func() (string, error) {
	t.Helper()
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		for _, rel := range sandboxFiles {
			b, err := os.ReadFile(filepath.Join(repo, rel))
			if err != nil {
				if difftest.OriginalsDeleted(repo) {
					continue
				}
				return "", fmt.Errorf("sandbox source %s: %w", rel, err)
			}
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, b, 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		lua, err := os.ReadFile(filepath.Join(pob, "src/Data/Global.lua"))
		if err != nil {
			return "", fmt.Errorf("pinned PoB Global.lua: %w", err)
		}
		dst := filepath.Join(sb, "pob", "src/Data/Global.lua")
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		if err := os.WriteFile(dst, lua, 0o600); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		return sb, nil
	}
}

func rewrite(sandbox, rel string, f func(string) string) error {
	p := filepath.Join(sandbox, rel)
	b, err := os.ReadFile(p) //nolint:gosec // a harness-owned temp path
	if err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	if err := os.WriteFile(p, []byte(f(string(b))), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

var reKWBlock = regexp.MustCompile(`(?s)KW_BITS\s*=\s*\[(.*?)\]`)

var reFlagDef = regexp.MustCompile(`(?m)^KeywordFlag\.\w+\s*=\s*0x[0-9a-fA-F]+`)

func keepFlags(b []byte, n int) []byte {
	var out []byte
	prev := 0
	for i, loc := range reFlagDef.FindAllIndex(b, -1) {
		if i < n {
			continue
		}
		out = append(out, b[prev:loc[0]]...)
		out = append(out, "--"...)
		prev = loc[0]
	}
	return append(out, b[prev:]...)
}

var reKWRow = regexp.MustCompile(`\(0x[0-9a-fA-F]+,\s*"KeywordFlag\w+"\),?`)

func keepRows(s string, n int) string {
	m := reKWBlock.FindStringSubmatchIndex(s)
	if m == nil {
		return s
	}
	rows := reKWRow.FindAllString(s[m[2]:m[3]], n)
	return s[:m[0]] + "KW_BITS = [\n" + strings.Join(rows, "\n") + "\n]" + s[m[1]:]
}

func TestKwBitsParityPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	pob := pobDir(t)
	if _, err := os.Stat(filepath.Join(pob, "src/Data/Global.lua")); err != nil {
		t.Skipf("pinned PoB not readable at %s — set POB_DIR", pob)
	}
	bin := buildPort(t, repo)
	mk := sandboxFactory(t, repo, pob)

	pySrc := difftest.ReadOriginal(t, repo, pySource)

	spec := difftest.Spec{
		Tool:         "kw-bits-parity",
		PythonScript: pySource,
		GoBin:        bin,
		GoArgs:       []string{"kw-bits-parity"},
		Stderr:       difftest.StderrExact,
		RefusalSites: 5,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-go-mod", Refusal: "no go.mod declaring module github.com/Vilsol/go-pob at or above this tool", Break: func(sb string) error {
				return os.Remove(filepath.Join(sb, "go.mod"))
			}},

			{Name: "bad-gopob-root", Refusal: "GOPOB_ROOT does not contain a go.mod declaring module",
				Break: func(sb string) error { return nil }},

			{Name: "pob-missing", Refusal: "pinned PoB not readable at", Break: func(sb string) error {
				return os.Remove(filepath.Join(sb, "pob", "src/Data/Global.lua"))
			}},

			{Name: "pob-unreadable", Refusal: "pinned PoB not readable at", Break: func(sb string) error {
				return os.Chmod(filepath.Join(sb, "pob", "src/Data/Global.lua"), 0o000)
			}},

			{Name: "too-few-flags", Refusal: "parsed only 19 flags from", Break: func(sb string) error {
				p := filepath.Join(sb, "pob", "src/Data/Global.lua")
				b, err := os.ReadFile(p) //nolint:gosec // a harness-owned temp path
				if err != nil {
					return fmt.Errorf("fixture: %w", err)
				}
				return os.WriteFile(p, keepFlags(b, 19), 0o600)
			}},

			{Name: "at-flag-threshold", Passes: true, Break: func(sb string) error {
				p := filepath.Join(sb, "pob", "src/Data/Global.lua")
				b, err := os.ReadFile(p) //nolint:gosec // a harness-owned temp path
				if err != nil {
					return fmt.Errorf("fixture: %w", err)
				}
				return os.WriteFile(p, keepFlags(b, 20), 0o600)
			}},

			{Name: "value-mismatch", Refusal: "1 mismatch(es). The emitters decode numbers PoB WROTE", Break: func(sb string) error {
				p := filepath.Join(sb, "pob", "src/Data/Global.lua")
				b, err := os.ReadFile(p) //nolint:gosec // a harness-owned temp path
				if err != nil {
					return fmt.Errorf("fixture: %w", err)
				}
				out := regexp.MustCompile(`(?m)^(KeywordFlag\.Attack\s*=\s*)0x[0-9a-fA-F]+`).
					ReplaceAll(b, []byte("${1}0x00000042"))
				if bytes.Equal(out, b) {
					return fmt.Errorf("fixture: KeywordFlag.Attack not found in Global.lua — the " +
						"break landed nowhere and both sides would agree about nothing")
				}
				return os.WriteFile(p, out, 0o600)
			}},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "pob-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable and this fixture " +
					"cannot fail")
			}
			s := spec
			s.Env = []string{"POB_DIR=pob"}
			if f.Name == "bad-gopob-root" {
				s.Env = append(s.Env, "GOPOB_ROOT=pob")
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestKwBitsParityPythonIsStableAcrossHashSeeds: THE PYTHON MUST BE STABLE AGAINST ITSELF BEFORE IT IS COMPARED TO ANYTHING.
func TestKwBitsParityPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	pob := pobDir(t)
	if _, err := os.Stat(filepath.Join(pob, "src/Data/Global.lua")); err != nil {
		t.Skipf("pinned PoB not readable at %s — set POB_DIR", pob)
	}
	mk := sandboxFactory(t, repo, pob)
	spec := difftest.Spec{
		Tool:         "kw-bits-parity",
		PythonScript: pySource,
		Env:          []string{"POB_DIR=pob"},
	}
	difftest.SelfIdentity(t, t.Context(), spec, mk, []string{"0", "12345", "999983"})
}

func originalsGone(t *testing.T) bool {
	t.Helper()
	return difftest.OriginalsDeleted(repoRoot(t))
}
