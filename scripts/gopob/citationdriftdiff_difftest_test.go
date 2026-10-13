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

const cddPy = "scripts/citation_drift_diff.py"

var cddWideAnchor = "local wideStableAnchorLineThatIsLongerThanTheWideLineBelowIt = " +
	strings.Repeat("z", 140)

func cddOldLua(name string) string {
	switch name {
	case "Anchored.lua":
		return strings.Join([]string{
			"-- header", "local distinctiveAnchorAlphaLongestSurvivingLine = 1",
			"  return 1", "end", "local filler = 2", "",
		}, "\n")
	case "Offset.lua":
		return strings.Join([]string{
			"-- header", "local first = 1", "local second = 2",
			"local offsetAnchorDistinctiveGamma = 3", "  return 4", "end", "",
		}, "\n")
	case "MultiOld.lua":
		return strings.Join([]string{
			"-- header", "local repeatedDistinctiveDelta = 1", "  return 1", "end",
			"local repeatedDistinctiveDelta = 1", "",
		}, "\n")
	case "MultiPin.lua":
		return strings.Join([]string{
			"-- header", "local doubledInThePinEpsilon = 1", "  return 1", "end", "",
		}, "\n")
	case "Rank.lua":
		return strings.Join([]string{
			"local rankAlphaOne = 1", "local rankBetaTwo = 2", "local rankGammaThree = 3",
			"local rankAlphaOne = 1", "local rankBetaTwo = 2", "local rankGammaThree = 3", "",
		}, "\n")
	case "Weak.lua":
		return strings.Join([]string{
			"local aaa = 1", "local bbb = 2", "local ccc = 3", "local ddd = 4", "end", "",
		}, "\n")
	case "Unchanged.lua":
		return strings.Join([]string{
			"-- header", "local unchangedAnchorBetaLongEnoughToAnchor = 1",
			"   return 2", "end", "",
		}, "\n")
	case "Fuzzy.lua":
		return strings.Join([]string{"local a = 1", "local b = 2", "end", "else", "", ""}, "\n")
	case "Gone.lua":
		return strings.Join([]string{
			"-- header", "local vanishedSymbolGamma = 1", "  return vanishedSymbolGamma",
			"end", "",
		}, "\n")
	case "Empty.lua":
		return strings.Join([]string{"-- header", "", "   ", "-- only comments", ""}, "\n")
	case "Wide.lua":
		return "-- header\nlocal wideAnchorZeta = \"" + strings.Repeat("ü", 90) + "\"\nend\n"
	}
	return ""
}

func cddPinLua(name string) string {
	switch name {
	case "Anchored.lua":
		return strings.Join([]string{
			"-- a new line upstream", "-- header",
			"local distinctiveAnchorAlphaLongestSurvivingLine = 1",
			"  return 99", "end", "local filler = 2", "",
		}, "\n")
	case "Offset.lua":
		return strings.Join([]string{
			"-- two new lines", "-- upstream", "-- header", "local first = 1", "local second = 2",
			"local offsetAnchorDistinctiveGamma = 3", "  return 44", "end", "",
		}, "\n")
	case "MultiOld.lua":
		return strings.Join([]string{
			"-- header", "local repeatedDistinctiveDelta = 1", "  return 77", "end", "",
		}, "\n")
	case "MultiPin.lua":
		return strings.Join([]string{
			"-- header", "local doubledInThePinEpsilon = 1", "  return 88", "end",
			"local doubledInThePinEpsilon = 1", "",
		}, "\n")
	case "Rank.lua":
		return strings.Join([]string{
			"local rankGammaThree = 3", "local rankBetaTwo = 2", "local rankAlphaOne = 1",
			"local rankAlphaOne = 1", "local rankBetaTwo = 2", "local zzz = 9",
			"local q1 = 1", "local q2 = 2", "local q3 = 3",
			"local q4 = 4", "local q5 = 5", "local q6 = 6", "",
		}, "\n")
	case "Weak.lua":
		return strings.Join([]string{
			"local zzz = 9", "local yyy = 8", "local xxx = 7", "local www = 6", "end", "",
		}, "\n")
	case "Unchanged.lua":
		return strings.Join([]string{
			"-- header", "\tlocal unchangedAnchorBetaLongEnoughToAnchor   = 1  -- new comment",
			"        return 2", "end", "",
		}, "\n")
	case "Fuzzy.lua":
		return strings.Join([]string{"local a = 1", "local b = 3", "end", "else", "", ""}, "\n")
	case "Gone.lua":
		return strings.Join([]string{
			"-- an entirely unrelated file now", "local somethingElseEntirely = 5",
			"  print(somethingElseEntirely)", "end", "",
		}, "\n")
	case "Empty.lua":
		return strings.Join([]string{"-- header", "", "   ", "-- only comments", ""}, "\n")
	case "Wide.lua":
		return "-- header\nlocal wideAnchorZeta = \"" + strings.Repeat("ö", 90) + "\"\nend\n"
	}
	return ""
}

func cddGoSrc() string {
	return strings.Join([]string{
		"package a",
		"",
		"//pob:src/Modules/Anchored.lua:2-4",
		"func CddAnchored() {}",
		"",
		"//pob:src/Modules/Offset.lua:2-6",
		"func CddOffset() {}",
		"",
		"//pob:src/Modules/MultiOld.lua:2-4",
		"func CddMultiOld() {}",
		"",
		"//pob:src/Modules/MultiPin.lua:2-4",
		"func CddMultiPin() {}",
		"",
		"//pob:src/Modules/Rank.lua:1-3",
		"func CddRank() {}",
		"",
		"//pob:src/Modules/Weak.lua:1-5",
		"func CddWeak() {}",
		"",
		"//pob:src/Modules/Unchanged.lua:2-4",
		"func CddUnchanged() {}",
		"",
		"//pob:src/Modules/Fuzzy.lua:1-4",
		"func CddFuzzy() {}",
		"",
		"//pob:src/Modules/Gone.lua:2-4",
		"func CddGone() {}",
		"",
		"//pob:src/Modules/Empty.lua:2-4",
		"func CddEmptyOld() {}",
		"",
		"//pob:src/Modules/Wide.lua:2-4",
		"// ⚠️ A DECLARATION LONGER THAN 110 CHARACTERS, so the symbol line is cut. Only a long one",
		"// can tell a 110-rune cut from no cut at all.",
		"func CddWideSymbolNameThatKeepsGoingAndGoingWellPastOneHundredAndTenCharactersSoTheCutIsVisible(a int, b int) {}",
		"",
		"//pob:src/Modules/Anchored.lua:900-901",
		"type CddPastEOF struct{}",
		"",
		"// ⚠️ A WHOLE-FILE MARKER HAS NO RANGE and must be skipped by the sweep, not treated as 0.",
		"//pob:src/Modules/Anchored.lua",
		"var CddWholeFile = 1",
		"",
	}, "\n")
}

func cddSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, cddPy)
	pc := difftest.ReadOriginal(t, repo, pcPy)
	names := []string{"Anchored.lua", "Offset.lua", "MultiOld.lua", "MultiPin.lua", "Weak.lua", "Rank.lua",
		"Unchanged.lua", "Fuzzy.lua", "Gone.lua", "Empty.lua", "Wide.lua"}
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			cddPy:       string(py),
			pcPy:        string(pc),
			"go.mod":    "module github.com/Vilsol/go-pob\n\ngo 1.24\n",
			"a/mark.go": cddGoSrc(),
		}
		for _, nm := range names {
			files["pin/src/Modules/"+nm] = cddPinLua(nm)
			files["old/src/Modules/"+nm] = cddOldLua(nm)
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

// TestCitationDriftDiffPortIsIndistinguishable: OLD IS A DIRECTORY HERE, NOT A GIT REF.
func TestCitationDriftDiffPortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := cddSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, cddPy)

	spec := difftest.Spec{
		Tool:         "citation-drift-diff",
		PythonScript: cddPy,
		GoBin:        bin,
		GoArgs:       []string{"citation-drift-diff"},
		Env:          []string{"POB_DIR=pin", "OLD=old"},
		Stderr:       difftest.StderrExact,
		RefusalSites: 0,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "old-tree-unreadable", Refusal: "Anchored.lua is not readable: Permission denied",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "old/src/Modules/Anchored.lua"), 0o000)
				}},
			{Name: "go-source-unreadable", Refusal: "mark.go is not readable: Permission denied",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "a/mark.go"), 0o000)
				}},
			{Name: "pin-lua-not-utf8", Refusal: "Anchored.lua is not valid UTF-8",
				Break: func(sb string) error {
					bad := strings.Replace(cddPinLua("Anchored.lua"), "local filler = 2",
						"local filler = \xff2", 1)
					return os.WriteFile(filepath.Join(sb, "pin/src/Modules/Anchored.lua"),
						[]byte(bad), 0o600)
				}},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "go-source-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			if f.Name == "old-tree-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			difftest.RunFixture(t, t.Context(), spec, f, mk)
		})
	}
}

func TestCitationDriftDiffModes(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := cddSandbox(t, repo)
	for _, tc := range []struct {
		name string
		env  []string
		args []string
	}{
		{"all", []string{"POB_DIR=pin", "OLD=old"}, nil},
		{"filter-env", []string{"POB_DIR=pin", "OLD=old", "FILE=a/mark.go"}, nil},
		{"filter-argv", []string{"POB_DIR=pin", "OLD=old"}, []string{"a/mark"}},
		{"filter-matches-nothing", []string{"POB_DIR=pin", "OLD=old", "FILE=zzz"}, nil},
		{"selftest", []string{"POB_DIR=pin", "OLD=old"}, []string{"--selftest"}},
		{"old-tree-empty", []string{"POB_DIR=pin", "OLD=pin/src"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "citation-drift-diff",
				PythonScript: cddPy,
				PythonArgs:   tc.args,
				GoBin:        bin,
				GoArgs:       append([]string{"citation-drift-diff"}, tc.args...),
				Env:          tc.env,
				Stderr:       difftest.StderrExact,
			}, difftest.Fixture{Name: tc.name}, mk)
		})
	}
}

func TestCitationDriftDiffPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := cddSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "citation-drift-diff",
		PythonScript: cddPy,
		Env:          []string{"POB_DIR=pin", "OLD=old"},
	}, mk, []string{"0", "12345", "999983"})
}
