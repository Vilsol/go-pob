package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const cdPy = "scripts/citation_drift.py"

const (
	cdLuaOld = `-- pob thing
local sameAnchorIdentifierAAAA = 1
local movedAnchorIdentifierBBBB = 2
local changedAnchorIdentifierCCCC = 3
local goneAnchorIdentifierDDDD = 4
-- only comments here
-- and here
local tail = 5
local equalLengthAAA = 111
local equalLengthBBB = 222
local tenCharTok = 77
local secondMovedIdentifierEEEE = 6
local absentTokenOne = absentTokenTwo + absentTokenThree + presentTokenFour
`
	cdLuaNew = `-- pob thing
local sameAnchorIdentifierAAAA = 1
local inserted = 0
local changedAnchorIdentifierCCCC = 3 + extra
local movedAnchorIdentifierBBBB = 2
-- only comments here
-- and here
local tail = 5
local padding = 0
local equalLengthAAA = 111
local tenCharTok = 77 + more
local extraPadding = 1
local secondMovedIdentifierEEEE = 6
local presentTokenFour = 0
`
	cdMarkedFile = "testdata/citationdrift_marked.go.txt"
)

const cdOldRev = "oldrev"

func cdTemplate(t *testing.T, repo string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	dir := t.TempDir()
	py := difftest.ReadOriginal(t, repo, cdPy)
	write := func(rel, body string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module github.com/Vilsol/go-pob\n\ngo 1.26.2\n")
	write(cdPy, string(py))
	marked, err := os.ReadFile(filepath.Join(repo, "scripts", "gopob", cdMarkedFile))
	if err != nil {
		t.Fatal(err)
	}
	write("calculator/marked.go", string(marked))

	py3, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not on PATH")
	}
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(py3, filepath.Join(dir, "bin", "python3")); err != nil {
		t.Fatal(err)
	}

	pob := filepath.Join(dir, "pob")
	write("pob/src/Modules/Thing.lua", cdLuaOld)
	git := func(args ...string) {
		t.Helper()
		//nolint:gosec // fixed git arguments this test wrote itself, against a temp repo
		c := exec.CommandContext(t.Context(), "git", append([]string{"-C", pob}, args...)...)
		c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("add", "-A")
	git("commit", "-q", "-m", "old")
	git("tag", cdOldRev)
	write("pob/src/Modules/Thing.lua", cdLuaNew)
	git("add", "-A")
	git("commit", "-q", "-m", "new")
	return dir
}

func cdSandbox(t *testing.T, tmpl string) func() (string, error) {
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		if err := difftest.CopyTree(sb, tmpl); err != nil {
			return "", err
		}
		py3, err := exec.LookPath("python3")
		if err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		_ = os.Remove(filepath.Join(sb, "bin", "python3"))
		if err := os.MkdirAll(filepath.Join(sb, "bin"), 0o750); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		if err := os.Symlink(py3, filepath.Join(sb, "bin", "python3")); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		return sb, nil
	}
}

func TestCitationDriftPortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := cdSandbox(t, cdTemplate(t, repo))

	pySrc := difftest.ReadOriginal(t, repo, cdPy)

	spec := difftest.Spec{
		Tool:         "citation-drift",
		PythonScript: cdPy,
		GoBin:        bin,
		GoArgs:       []string{"citation-drift"},
		Env:          []string{"POB_DIR=pob", "OLD=" + cdOldRev},
		Stderr:       difftest.StderrExact,
		RefusalSites: 5,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-go-mod", Refusal: "no go.mod declaring module github.com/Vilsol/go-pob",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "go.mod")) }},

			{Name: "bad-gopob-root", Refusal: "GOPOB_ROOT does not contain a go.mod declaring module",
				Break: func(string) error { return nil }},

			{Name: "lua-unreadable", Refusal: "pob/src/Modules/Thing.lua is not readable:",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "pob/src/Modules/Thing.lua"), 0o000)
				}},

			{Name: "lua-invalid-utf8", Refusal: "pob/src/Modules/Thing.lua is not valid UTF-8",
				Break: func(sb string) error {
					return put(sb, "pob/src/Modules/Thing.lua", "local a\xff\xfe = 1\n")
				}},

			{Name: "git-missing", Refusal: "cannot run git: No such file or directory",
				Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "lua-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			s := spec
			switch f.Name {
			case "bad-gopob-root":
				s.Env = append(append([]string{}, spec.Env...), "GOPOB_ROOT=pob")
			case "git-missing":
				s.Env = append(append([]string{}, spec.Env...), "PATH=bin")
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestCitationDriftFixtureReachesEveryVerdict: THE HAPPY FIXTURE MUST REACH EVERY VERDICT OR IT IS A ONE-BRANCH TEST WEARING A SIX-BRANCH COSTUME.
func TestCitationDriftFixtureReachesEveryVerdict(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := cdSandbox(t, cdTemplate(t, repo))
	sb, err := mk()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), bin, "citation-drift")
	cmd.Dir = sb
	cmd.Env = append(os.Environ(), "POB_DIR=pob", "OLD="+cdOldRev)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running the port: %v", err)
	}
	tally := parseTally(string(out))
	if len(tally) == 0 {
		t.Fatal("no tally block in the output; this control cannot see anything")
	}
	for _, verdict := range []string{"SAME", "MOVED", "CHANGED", "GONE", "EMPTY-OLD", "NOFILE-OLD"} {
		if tally[verdict] == 0 {
			t.Errorf("the fixture produces no %s row; that branch is untested (tally %v)",
				verdict, tally)
		}
	}
	t.Logf("tally: %v", tally)
}

func parseTally(out string) map[string]int {
	tally := map[string]int{}
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) != 2 || !strings.HasPrefix(l, "  ") {
			continue
		}
		n, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		tally[f[1]] = n
	}
	return tally
}

func TestCitationDriftPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := cdSandbox(t, cdTemplate(t, repo))
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "citation-drift",
		PythonScript: cdPy,
		Env:          []string{"POB_DIR=pob", "OLD=" + cdOldRev},
	}, mk, []string{"0", "12345", "999983"})
}
