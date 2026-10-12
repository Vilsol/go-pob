package calculator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
)

func requireGameData(t *testing.T, version string) {
	t.Helper()
	registerVersionInTestCacheKey()
	if raw.LatestVersion != version {
		t.Skipf("SKIP: fixtures are %s but this process loaded %s game data. "+
			"Game data is process-global; re-run with POE_VERSION=%s.",
			version, raw.LatestVersion, version)
	}
}

var retiredGradingVersions = []string{"3.18", "3.20"}

// TestNoRetiredVersionGrading fails if any Go source, shell script or task definition reads a retired version's test data.
func TestNoRetiredVersionGrading(t *testing.T) {
	repo := ".."

	pats := make([]*regexp.Regexp, 0, 4*len(retiredGradingVersions))
	for _, v := range retiredGradingVersions {
		esc := regexp.QuoteMeta(v)
		under := regexp.QuoteMeta(strings.ReplaceAll(v, ".", "_"))
		pats = append(pats,
			regexp.MustCompile(`testdata/(corpus|oracle|pob-skills)-`+esc),
			regexp.MustCompile(`dumps-`+esc),
			regexp.MustCompile(`oracle-`+under),
			regexp.MustCompile(`POE_VERSION=["']?`+esc),
		)
	}

	type hit struct{ file, line, text string }
	var hits []hit

	const okMarker = "retired-version-ok:"
	exempt := func(line string) bool {
		i := strings.Index(line, okMarker)
		return i >= 0 && strings.TrimSpace(line[i+len(okMarker):]) != ""
	}

	goDirs := []string{"calculator", "data", "data/raw", "builds", "mod", "moddb", "pob", "utils",
		"wasm", "scripts/gopob", "scripts/gopob/internal/difftest", "scripts/gopob/internal/root",
		"scripts/gopob/internal/toolchain", "scripts/gopob/internal/gateversions", "scripts/corpus",
		"internal/oraclebin", "builds/parsecache"}
	for _, dir := range goDirs {
		abs := filepath.Join(repo, dir)
		entries, err := os.ReadDir(abs)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			if e.Name() == "game_data_guard_test.go" {
				continue
			}
			path := filepath.Join(abs, e.Name())
			src, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
			if err != nil {
				continue
			}
			goLines := strings.Split(string(src), "\n")
			ast.Inspect(f, func(n ast.Node) bool {
				bl, ok := n.(*ast.BasicLit)
				if !ok || bl.Kind != token.STRING {
					return true
				}
				if ln := fset.Position(bl.Pos()).Line; ln-1 < len(goLines) && exempt(goLines[ln-1]) {
					return true
				}
				for _, p := range pats {
					if p.MatchString(bl.Value) {
						hits = append(hits, hit{
							file: filepath.Join(dir, e.Name()),
							line: fset.Position(bl.Pos()).String(),
							text: bl.Value,
						})
						break
					}
				}
				return true
			})
		}
	}

	shellFiles := []string{"Taskfile.yml"}
	if entries, err := os.ReadDir(filepath.Join(repo, "scripts")); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".sh") {
				shellFiles = append(shellFiles, filepath.Join("scripts", e.Name()))
			}
		}
	}
	for _, rel := range shellFiles {
		src, err := os.ReadFile(filepath.Join(repo, rel))
		if err != nil {
			continue
		}
		for i, line := range strings.Split(string(src), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") || exempt(line) {
				continue
			}
			for _, p := range pats {
				if p.MatchString(line) {
					hits = append(hits, hit{file: rel, line: strconv.Itoa(i + 1), text: trimmed})
					break
				}
			}
		}
	}

	for _, dir := range []string{"scripts", "oracle"} {
		entries, err := os.ReadDir(filepath.Join(repo, dir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".py") {
				continue
			}
			rel := filepath.Join(dir, e.Name())
			src, err := os.ReadFile(filepath.Join(repo, rel))
			if err != nil {
				continue
			}
			raw := strings.Split(string(src), "\n")
			for i, line := range strings.Split(stripPythonProse(string(src)), "\n") {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" || (i < len(raw) && exempt(raw[i])) {
					continue
				}
				for _, p := range pats {
					if p.MatchString(line) {
						hits = append(hits, hit{file: rel, line: strconv.Itoa(i + 1), text: trimmed})
						break
					}
				}
			}
		}
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].file < hits[j].file })
	for _, h := range hits {
		t.Errorf("RETIRED-VERSION GRADING at %s:%s\n    %s\n"+
			"    %v is RETIRED: the pinned PoB is 3.29, the game does not roll back, and a\n"+
			"    divergence from an older PoB is not a defect. Their testdata is kept as\n"+
			"    historical DATA; code must not read it. Point this at the authoritative\n"+
			"    corpus (see corpusDirForLoadedData) or delete it.",
			h.file, h.line, h.text, retiredGradingVersions)
	}
}

func stripPythonProse(src string) string {
	var out strings.Builder
	i, n := 0, len(src)
	for i < n {
		if src[i] == '\n' {
			out.WriteByte('\n')
			i++
			continue
		}
		if strings.HasPrefix(src[i:], `"""`) || strings.HasPrefix(src[i:], "'''") {
			q := src[i : i+3]
			j := strings.Index(src[i+3:], q)
			if j < 0 {
				j = n
			} else {
				j = i + 3 + j + 3
			}
			for k := i; k < j; k++ {
				if src[k] == '\n' {
					out.WriteByte('\n')
				} else {
					out.WriteByte(' ')
				}
			}
			i = j
			continue
		}
		if src[i] == '"' || src[i] == '\'' {
			q := src[i]
			out.WriteByte(src[i])
			i++
			for i < n && src[i] != '\n' {
				out.WriteByte(src[i])
				if src[i] == '\\' && i+1 < n {
					i++
					if i < n {
						out.WriteByte(src[i])
					}
					i++
					continue
				}
				if src[i] == q {
					i++
					break
				}
				i++
			}
			continue
		}
		if src[i] == '#' {
			for i < n && src[i] != '\n' {
				out.WriteByte(' ')
				i++
			}
			continue
		}
		out.WriteByte(src[i])
		i++
	}
	return out.String()
}
