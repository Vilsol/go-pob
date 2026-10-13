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

const pcPy = "scripts/pob_citations.py"

func pcLuaCalc() string {
	var b strings.Builder
	b.WriteString("-- Calc.lua, a fixture\n")
	b.WriteString("local x = 1\n")
	b.WriteString("\n")
	b.WriteString("-- a comment above a function\n")
	b.WriteString("function Calc:doThing(a)\n")
	b.WriteString("\treturn a + 1\n")
	b.WriteString("end\n")
	b.WriteString("\n")
	b.WriteString("-- only comment and blank below\n")
	b.WriteString("\n")
	b.WriteString("local marker = uniqueTokenHere\n")
	b.WriteString("local other = 2\n")
	b.WriteString("-- this comment mentions function but declares none\n")
	b.WriteString("local z = 3\n")
	b.WriteString("local twiceToken = 1\n")
	b.WriteString("local alsoTwice = twiceToken\n")
	b.WriteString("local tail = 4\n")
	return b.String()
}

func pcGoSources() map[string]string {
	return map[string]string{
		"a/good.go": `package a

//pob:src/Modules/Calc.lua:5-7
func Good() {}

//pob:src/Modules/Calc.lua:9-10
func EmptyRange() {}
`,
		"a/broken.go": `package a

//pob:src/Modules/NoSuch.lua:1-2
func NoFile() {}

//pob:src/Modules/Calc.lua:900-901
func OutOfRange() {}

//pob:not a citation at all
func Malformed() {}

//pob:Calc.lua:5-7
func ShortPath() {}
`,
		"a/funcline.go": `package a

//pob:src/Modules/Calc.lua:11-12
func MidBody() {}

//pob:block:src/Modules/Calc.lua:11-12
func DeclaredBlock() {}

//pob:src/Modules/Calc.lua:13-14
func CommentSaysFunction() {}
`,
		"a/bounds.go": `package a

//pob:src/Modules/Calc.lua:16-17
func LastLineFits() {}

//pob:src/Modules/Calc.lua:17-18
func OnePastTheEnd() {}
`,
		"a/attach.go": `package a

// prose above the marker
//pob:src/Modules/Calc.lua:5-7
// and prose below it
func FarMarker() {}
`,
		"a/filemarker.go": `//pob:src/Modules/Calc.lua
package a

func Inner() {
	//pob:src/Modules/Calc.lua:5-7
	_ = 1
}
`,
		"a/pkgmarker.go": `package a //pob:src/Modules/Calc.lua

func Trailing() {}
`,
		"a/lexer.go": "package a\n\n" +
			"/*\nfunc NotReal() {}\nCalc.lua:5 cited inside a block comment\nspanning\nseveral\nlines\n*/\n\n" +
			"var raw = `a raw string with // not a comment and Calc.lua:6 inside`\n\n" +
			"var prov = \"provenance: Calc.lua:900\"\n\n" +
			"var lit = func (x int) int { return x }\n\n" +
			"//pob:src/Modules/Calc.lua:11-12\n" +
			"func AfterBlock() {}\n\n" +
			"// drift: quotes `local marker = uniqueTokenHere` against Calc.lua:5-7\n" +
			"func Drifting() {}\n\n" +
			"// drift too: quotes `twiceToken = 1` against Calc.lua:5-7\n" +
			"func DriftingTwice() {}\n",
		"a/unlisted.go":   "package a\n\nfunc Unlisted() {}\n",
		"a/plain_test.go": "package a\n\nfunc TestX() {}\n",
	}
}

func pcManifestText() string {
	return strings.Join([]string{
		"# fixture manifest",
		"a/good.go        PORT       src/Modules/Calc.lua",
		"a/broken.go      PORT       src/Modules/Calc.lua",
		"a/funcline.go    PORT       src/Modules/Calc.lua",
		"a/bounds.go      PORT       src/Modules/Calc.lua",
		"a/attach.go      PORT       src/Modules/Calc.lua",
		"a/pkgmarker.go   PORT       src/Modules/Calc.lua",
		"a/filemarker.go  PORT       src/Modules/Calc.lua",
		"a/lexer.go       GO-NATIVE  the lexer fixture, no Lua counterpart",
		"",
	}, "\n")
}

func pcGrandfatherEnv(t *testing.T, repo string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo, "scripts/gopob/testdata/pob-citations-grandfather.txt"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	path := filepath.Join(t.TempDir(), "grandfather.txt")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	return "POB_CITATIONS_GRANDFATHER=" + path
}

func pcSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, pcPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			pcPy:                        string(py),
			"pob/src/Modules/Calc.lua":  pcLuaCalc(),
			"pob/src/Classes/Other.lua": "function Other:f()\nend\n",
			"scripts/port_manifest.txt": pcManifestText(),
			"nopob/readme.txt":          "not a PoB checkout\n",
		}
		for rel, body := range pcGoSources() {
			files[rel] = body
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

func TestPobCitationsPortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pcSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, pcPy)

	spec := difftest.Spec{
		Tool:         "pob-citations",
		PythonScript: pcPy,
		GoBin:        bin,
		GoArgs:       []string{"pob-citations"},
		Env:          []string{"POB_DIR=pob", pcGrandfatherEnv(t, repo)},
		Stderr:       difftest.StderrExact,
		RefusalSites: 2,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "show-without-a-citation",
				Refusal: "REFUSING: --show needs a citation, e.g. --show src/Modules/CalcPerform.lua:2962-2970",
				Break:   func(string) error { return nil }},
			{Name: "pob-dir-is-not-a-checkout", Refusal: "REFUSING: not a PoB checkout (no src/): nopob",
				Break: func(string) error { return nil }},
			{Name: "manifest-missing",
				Refusal: "scripts/port_manifest.txt is not readable: No such file or directory",
				Break: func(sb string) error {
					return os.Remove(filepath.Join(sb, "scripts/port_manifest.txt"))
				}},
			{Name: "a-directory-under-the-pob-is-unreadable", Finding: true,
				Break: func(sb string) error {
					d := filepath.Join(sb, "pob/src/Modules")
					t.Cleanup(func() { _ = os.Chmod(d, 0o750) })
					return os.Chmod(d, 0o000)
				}},
			{Name: "lua-unreadable", Refusal: "Calc.lua is not readable: Permission denied",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "pob/src/Modules/Calc.lua"), 0o000)
				}},
			{Name: "lua-not-utf8", Refusal: "Calc.lua is not valid UTF-8",
				Break: func(sb string) error {
					bad := strings.Replace(pcLuaCalc(), "local other = 2", "local other = \xff2", 1)
					return os.WriteFile(filepath.Join(sb, "pob/src/Modules/Calc.lua"),
						[]byte(bad), 0o600)
				}},
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
			case "show-without-a-citation":
				s.PythonArgs = []string{"--show"}
			case "pob-dir-is-not-a-checkout":
				s.Env = []string{"POB_DIR=nopob", pcGrandfatherEnv(t, repo)}
			}
			s.GoArgs = append([]string{"pob-citations"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestPobCitationsShowModes: --show IS A SEPARATE ENTRY POINT AND ITS OWN FAILURE MODES.
func TestPobCitationsShowModes(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pcSandbox(t, repo)
	for _, spec := range []string{
		"src/Modules/Calc.lua:5-7",
		"src/Modules/Calc.lua:5",
		"src/Modules/Calc.lua",
		"src/Modules/NoSuch.lua:1-2",
		"src/Modules/Calc.lua:900-901",
		"Calc.lua:5-7",
		"not a citation",
	} {
		t.Run(strings.NewReplacer("/", "_", ":", "-", " ", "_").Replace(spec), func(t *testing.T) {
			args := []string{"--show", spec}
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "pob-citations",
				PythonScript: pcPy,
				PythonArgs:   args,
				GoBin:        bin,
				GoArgs:       append([]string{"pob-citations"}, args...),
				Env:          []string{"POB_DIR=pob", pcGrandfatherEnv(t, repo)},
				Stderr:       difftest.StderrExact,
			}, difftest.Fixture{Name: "show"}, mk)
		})
	}
}

// TestPobCitationsReportMode: REPORT=1 must never exit nonzero, which is the mode a ratchet measurement runs in.
func TestPobCitationsReportMode(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pcSandbox(t, repo)
	difftest.RunFixture(t, t.Context(), difftest.Spec{
		Tool:         "pob-citations",
		PythonScript: pcPy,
		GoBin:        bin,
		GoArgs:       []string{"pob-citations"},
		Env:          []string{"POB_DIR=pob", "REPORT=1", pcGrandfatherEnv(t, repo)},
		Stderr:       difftest.StderrExact,
	}, difftest.Fixture{Name: "report"}, mk)
}

func TestPobCitationsPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := pcSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "pob-citations",
		PythonScript: pcPy,
		Env:          []string{"POB_DIR=pob", "REPORT=1", pcGrandfatherEnv(t, repo)},
	}, mk, []string{"0", "12345", "999983"})
}
