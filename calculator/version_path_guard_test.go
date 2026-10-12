package calculator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var versionedDataPath = regexp.MustCompile(`(corpus|oracle|oracle-calcs|dumps|pob-skills)-\d+[._]\d+`)

var pathLikePrefixes = []string{"../", "./", "/", "~/", "testdata/"}

func hardcodedVersionPath(value string) bool {
	if !versionedDataPath.MatchString(value) {
		return false
	}
	for _, p := range pathLikePrefixes {
		if strings.HasPrefix(value, p) {
			return true
		}
	}
	return versionedDataPath.FindString(value) == value
}

// TestNoHardcodedVersionPaths fails when any Go source writes a version-qualified corpus.
func TestNoHardcodedVersionPaths(t *testing.T) {
	type hit struct{ file, line, text string }
	var hits []hit

	repo := ".."
	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "frontend": true, "testdata": true,
		".instruments": true, "vendor": true,
	}

	err := filepath.WalkDir(repo, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		if d.Name() == "version_path_guard_test.go" {
			return nil
		}
		//nolint:gosec // G122: walks this repo's own source tree; no untrusted symlinks in play.
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil //nolint:nilerr // one unreadable file must not abort the scan
		}
		rel, relErr := filepath.Rel(repo, path)
		if relErr != nil {
			rel = path
		}
		fset := token.NewFileSet()
		f, parseErr := parser.ParseFile(fset, path, src, parser.ParseComments)
		if parseErr != nil {
			return nil //nolint:nilerr
		}
		ast.Inspect(f, func(n ast.Node) bool {
			bl, ok := n.(*ast.BasicLit)
			if !ok || bl.Kind != token.STRING {
				return true
			}
			value, unquoteErr := strconv.Unquote(bl.Value)
			if unquoteErr != nil {
				value = bl.Value
			}
			if hardcodedVersionPath(value) {
				hits = append(hits, hit{
					file: rel,
					line: strconv.Itoa(fset.Position(bl.Pos()).Line),
					text: value,
				})
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("REFUSING: could not walk %s: %v", repo, err)
	}

	if scanned := countGoFiles(t, repo, skipDirs); scanned < 100 {
		t.Fatalf("REFUSING: the walk saw %d .go files, expected hundreds. A broken walk reports a "+
			"clean tree in exactly the same way a clean tree does.", scanned)
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].file < hits[j].file })
	for _, h := range hits {
		t.Errorf("HARDCODED VERSION PATH at %s:%s\n    %q\n"+
			"    Resolve it from the loaded game data instead — corpusDirForLoadedData,\n"+
			"    corpusNamesForLoadedData, oracleForLoadedData, corpusEnvForLoadedData or\n"+
			"    dumpsDirForLoadedData in calculator/corpus_version_test.go. They REFUSE rather\n"+
			"    than falling back, so they cannot grade another league. Keep the version PIN\n"+
			"    (requireGameData) — it is the path that must follow data/raw's DefaultVersion.",
			h.file, h.line, h.text)
	}
}

func countGoFiles(t *testing.T, repo string, skipDirs map[string]bool) int {
	t.Helper()
	n := 0
	_ = filepath.WalkDir(repo, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // same tolerance as the scan it is counting for
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".go") {
			n++
		}
		return nil
	})
	return n
}

// TestHardcodedVersionPathDetectorSeparatesPathsFromProse is the two-way control.
func TestHardcodedVersionPathDetectorSeparatesPathsFromProse(t *testing.T) {
	mustFlag := []string{
		"../testdata/corpus-3.14",
		"../testdata/corpus-3.14/0001-juggernaut-molten-strike.xml",
		"../testdata/oracle-3.14/output-3.14.jsonl.gz",
		"../testdata/corpus-3.14/*.xml",
		"../testdata/corpus-9.99",
		"../testdata/oracle-calcs-9.99/calcs-damage-9.99.jsonl.gz",
		"~/go-pob-corpus/dumps-3.14",
		"dumps-3.14",
		"corpus-3_14",
	}
	for _, s := range mustFlag {
		if !hardcodedVersionPath(s) {
			t.Errorf("detector MISSED a hardcoded path: %q", s)
		}
	}

	mustPass := []string{
		"../testdata",
		"testdata",
		"corpus-",
		"oracle-",
		"dumps-",
		"output-",
		".jsonl.gz",
		"3.14",
		"3_14",
		"DefaultVersion is the game data an unqualified run loads, and since 2026-08-08 that is " +
			"3.14 — the version the pinned PoB, testdata/corpus-3.14 and testdata/oracle-3.14 all " +
			"belong to. It is now the ONLY graded version: 3.18 grading was removed on 2026-08-09.",
		"REFUSING: no dump set at %s — set DUMPS to a dump of the corpus, e.g. dumps-3.14",
		"the corpus and the loaded game data must be the same league",
	}
	for _, s := range mustPass {
		if hardcodedVersionPath(s) {
			t.Errorf("detector FALSE-POSITIVED on: %q", s)
		}
	}
}
