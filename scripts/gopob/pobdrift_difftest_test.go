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

const pbdPy = "scripts/pob_drift.py"

func pbdLua(name string, cand bool) string {
	switch name {
	case "Unch.lua":
		return "-- Unch\nfunction m.unch(a)\n\treturn a + 1\nend\n"
	case "Space.lua":
		if cand {
			return "-- Space\nfunction  m.sp( a )\n    return   a + 1\nend\n"
		}
		return "-- Space\nfunction m.sp(a)\n\treturn a + 1\nend\n"
	case "Moved.lua":
		if cand {
			return "\n\n\n-- Moved\nfunction m.mv(a)\n\treturn a + 2\nend\n"
		}
		return "-- Moved\nfunction m.mv(a)\n\treturn a + 2\nend\n"
	case "Twice.lua":
		if cand {
			return "\n-- Twice\nfunction m.tw(a)\n\treturn a + 3\nend\n\nfunction m.tw(a)\n\treturn a + 3\nend\n"
		}
		return "-- Twice\nfunction m.tw(a)\n\treturn a + 3\nend\n"
	case "Chg.lua":
		if cand {
			return "-- Chg\nfunction m.ch(a)\n\treturn a * 99\nend\n"
		}
		return "-- Chg\nfunction m.ch(a)\n\treturn a + 4\nend\n"
	case "Del.lua":
		if cand {
			return "-- Del\nfunction m.brandNew(a)\n\treturn a + 5\nend\n"
		}
		return "-- Del\nfunction m.oldName(a)\n\treturn a + 5\nend\n"
	case "Bare.lua":
		if cand {
			return "-- Bare\nlocal z = 777\n"
		}
		return "-- Bare\nlocal q = 111\nlocal r = 222\nlocal s = 333\n"
	default:
		return "-- Gone\nfunction m.gone(a)\n\treturn a + 6\nend\n"
	}
}

var pbdLuaNames = []string{"Unch.lua", "Space.lua", "Moved.lua", "Twice.lua", "Chg.lua", "Del.lua",
	"Bare.lua", "Gone.lua"}

func pbdGoSrc() string {
	var b strings.Builder
	b.WriteString("package calculator\n")
	for _, nm := range pbdLuaNames {
		sym := strings.TrimSuffix(nm, ".lua")
		fmt.Fprintf(&b, "\n//pob:src/Modules/%s:2-4\nfunc Pbd%s() {}\n", nm, sym)
	}
	b.WriteString("\n//pob:src/Modules/Moved.lua:2-3, src/Modules/Moved.lua:3-4\nfunc PbdTwoRanges() {}\n")
	b.WriteString("\n//pob:src/Modules/Chg.lua\nfunc PbdWholeFile() {}\n")
	return b.String()
}

func pbdSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, pbdPy)
	pcSrc := difftest.ReadOriginal(t, repo, pcPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			pbdPy:                       string(py),
			pcPy:                        string(pcSrc),
			"go.mod":                    "module github.com/Vilsol/go-pob\n\ngo 1.24\n",
			"calculator/marked.go":      pbdGoSrc(),
			"scripts/port_manifest.txt": "# fixture manifest\ncalculator/marked.go PORT\n",
		}
		for _, nm := range pbdLuaNames {
			files["pin/src/Modules/"+nm] = pbdLua(nm, false)
			if nm != "Gone.lua" {
				files["cand/src/Modules/"+nm] = pbdLua(nm, true)
			}
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

func pbdSpec(bin string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "pob-drift",
		PythonScript: pbdPy,
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"pob-drift"}, args...),
		Env:          []string{"POB_DIR=pin"},
		Stderr:       difftest.StderrExact,
	}
}

func pbdWriteIn(sb, rel, body string) error {
	dst := filepath.Join(sb, rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

// TestPobDriftAgainstPython is the report path plus every refusal a SANDBOX STATE can reach.
func TestPobDriftAgainstPython(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pbdSandbox(t, repo)
	s := pbdSpec(bin, "--candidate", "cand")
	s.RefusalSites = 2
	s.Fixtures = []difftest.Fixture{
		{Name: "happy"},
		{
			Name:    "candidate-absent",
			Refusal: "candidate checkout does not exist: cand",
			Break: func(sb string) error {
				return os.RemoveAll(filepath.Join(sb, "cand"))
			},
		},
		{
			Name:      "selftest-reports-a-broken-check",
			NoFixture: true,
			Refusal:   "SELFTEST:",
			Note: "sys.exit(main()) is non-zero only when --selftest finds a broken check, and " +
				"the selftest reads no sandbox state; a mutation control covers it",
		},
	}
	difftest.AssertRefusalSiteCount(t, s, pbdReadPy(t, repo))
	for _, f := range s.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.NoFixture {
				t.Skip("no fixture reaches this site: " + f.Note)
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

func pbdReadPy(t *testing.T, repo string) []byte {
	t.Helper()
	b := difftest.ReadOriginal(t, repo, pbdPy)
	return b
}

// TestPobDriftModes sweeps the reporting shapes.
func TestPobDriftModes(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pbdSandbox(t, repo)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"default-top", []string{"--candidate", "cand"}},
		{"top-0-means-all", []string{"--candidate", "cand", "--top", "0"}},
		{"top-negative", []string{"--candidate", "cand", "--top", "-3"}},
		{"top-1-truncates", []string{"--candidate", "cand", "--top", "1"}},
		{"details", []string{"--candidate", "cand", "--top", "9", "--details"}},
		{"details-top-0", []string{"--candidate", "cand", "--top", "0", "--details"}},
		{"identity", []string{"--candidate", "pin"}},
		{"identity-details", []string{"--candidate", "pin", "--details"}},
		{"fix-dry-run", []string{"--candidate", "cand", "--fix", "--dry-run"}},
		{"selftest", []string{"--selftest"}},
		{"help", []string{"--help"}},
		{"help-short", []string{"-h"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), pbdSpec(bin, tc.args...),
				difftest.Fixture{Name: tc.name}, mk)
		})
	}
}

// TestPobDriftFixWrites is the only path that MUTATES the tree, so the contract is the FILE, not stdout.
func TestPobDriftFixWrites(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pbdSandbox(t, repo)
	s := pbdSpec(bin, "--candidate", "cand", "--fix")
	s.Files = []string{"calculator/marked.go"}
	difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "fix-writes"}, mk)
}

// TestPobDriftArgumentGrammar is the hand-rolled parser, which exists precisely because argparse and Go's flag package word every one of these differently.
func TestPobDriftArgumentGrammar(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pbdSandbox(t, repo)
	for _, tc := range []struct {
		name    string
		args    []string
		refusal string
	}{
		{"unknown-flag", []string{"--nope"}, "unknown argument '--nope'"},
		{"positional", []string{"wat"}, "unknown argument 'wat'"},
		{"top-needs-a-value", []string{"--top"}, "--top needs a value"},
		{"candidate-needs-a-value", []string{"--candidate"}, "--candidate needs a value"},
		{"top-not-an-integer", []string{"--candidate", "cand", "--top", "abc"},
			"--top wants an integer, not 'abc'"},
		{"top-empty", []string{"--candidate", "cand", "--top", ""},
			"--top wants an integer, not ''"},
		{"equals-form-valued", []string{"--candidate=cand", "--top=4"}, ""},
		{"equals-form-on-a-flag", []string{"--details=1"}, "--details takes no value"},
		{"no-candidate-at-all", []string{}, "no candidate checkout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), pbdSpec(bin, tc.args...),
				difftest.Fixture{Name: tc.name, Refusal: tc.refusal}, mk)
		})
	}
}

// TestPobDriftSandboxRefusals covers the refusals that need a broken TREE rather than a bad flag.
func TestPobDriftSandboxRefusals(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pbdSandbox(t, repo)
	for _, tc := range []struct {
		name    string
		args    []string
		brk     func(string) error
		refusal string
	}{
		{"pin-absent", []string{"--candidate", "cand", "--pin", "nosuch"}, nil,
			"pin checkout does not exist: nosuch"},
		{
			"candidate-is-not-a-pob-checkout", []string{"--candidate", "notpob"},
			func(sb string) error { return pbdWriteIn(sb, "notpob/readme.txt", "not a PoB tree\n") },
			"candidate is not a PoB checkout",
		},
		{
			"candidate-shares-no-cited-file", []string{"--candidate", "empty"},
			func(sb string) error {
				for _, d := range []string{"empty/Modules/Other.lua", "empty/Classes/Thing.lua"} {
					if err := pbdWriteIn(sb, d, "local x = 1\n"); err != nil {
						return err
					}
				}
				return nil
			},
			"not one of the 8 cited Lua files exists in the candidate",
		},
		{
			"candidate-is-not-utf8", []string{"--candidate", "cand"},
			func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "cand/src/Modules/Unch.lua"),
					[]byte("-- Unch\nlocal y = \xff\xfe\n"), 0o600)
			},
			"cand/src/Modules/Unch.lua is not valid UTF-8 (cited Lua in the candidate)",
		},
		{
			"pin-is-not-utf8", []string{"--candidate", "cand"},
			func(sb string) error {
				return os.WriteFile(filepath.Join(sb, "pin/src/Modules/Unch.lua"),
					[]byte("-- Unch\nfunction m.unch(a)\n\treturn \xff\xfe\nend\n"), 0o600)
			},
			"pin/src/Modules/Unch.lua is not valid UTF-8",
		},
		{
			"markers-do-not-resolve-against-the-pin", []string{"--candidate", "cand"},
			func(sb string) error {
				return pbdWriteIn(sb, "calculator/marked.go",
					"package calculator\n\n//pob:src/Modules/Unch.lua:900-901\nfunc PbdOff() {}\n")
			},
			"marker(s) do not resolve against the PIN",
		},
		{
			"no-markers-anywhere", []string{"--candidate", "cand"},
			func(sb string) error {
				return pbdWriteIn(sb, "calculator/marked.go", "package calculator\n\nfunc PbdBare() {}\n")
			},
			"no //pob: markers found",
		},
		{
			"port-manifest-absent", []string{"--candidate", "cand"},
			func(sb string) error { return os.Remove(filepath.Join(sb, "scripts/port_manifest.txt")) },
			"port_manifest.txt is not readable",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), pbdSpec(bin, tc.args...),
				difftest.Fixture{Name: tc.name, Break: tc.brk, Refusal: tc.refusal}, mk)
		})
	}
}

// TestPobDriftPythonIsStableAcrossHashSeeds guards the port against a source of nondeterminism a differential run cannot see.
func TestPobDriftPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := pbdSandbox(t, repo)
	s := difftest.Spec{
		Tool:         "pob-drift",
		PythonScript: pbdPy,
		PythonArgs:   []string{"--candidate", "cand", "--top", "0", "--details"},
		Env:          []string{"POB_DIR=pin"},
	}
	difftest.SelfIdentity(t, t.Context(), s, mk, []string{"0", "12345", "999983"})
}
