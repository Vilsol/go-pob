package calculator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
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

	files := trackedGoFiles(t, repo, skipDirs)
	for _, rel := range files {
		if filepath.Base(rel) == "version_path_guard_test.go" {
			continue
		}
		path := filepath.Join(repo, rel)
		//nolint:gosec // G122: walks this repo's own source tree; no untrusted symlinks in play.
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			continue // one unreadable file must not abort the scan
		}
		fset := token.NewFileSet()
		f, parseErr := parser.ParseFile(fset, path, src, parser.ParseComments)
		if parseErr != nil {
			continue
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
	}

	if scanned := len(files); scanned < 100 {
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

// trackedGoFiles lists the repository's tracked .go files outside skipDirs, relative to repo. Asking git rather than
// walking keeps Go's test cache from keying on every entry of the repository root.
func trackedGoFiles(t *testing.T, repo string, skipDirs map[string]bool) []string {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "git", "-C", repo, "ls-files", "-z", "--", "*.go").Output()
	if err != nil {
		t.Fatalf("REFUSING: could not list %s: %v", repo, err)
	}
	var files []string
	for _, rel := range strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		dirs := strings.Split(rel, "/")
		if slices.ContainsFunc(dirs[:len(dirs)-1], func(d string) bool { return skipDirs[d] }) {
			continue
		}
		files = append(files, filepath.FromSlash(rel))
	}
	return files
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
