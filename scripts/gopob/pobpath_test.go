package main

import (
	"fmt"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var pobCheckoutLiteral = regexp.MustCompile(`^(~/)?PathOfBuilding(-PoE2)?(/.*)?$`)

// Files under scripts/ that may still name the checkout, each with why.
var pobPathGuardExclusions = map[string]string{
	"scripts/gopob/pobpath.go": "the resolver itself",
}

func pobCheckoutLiterals(src []byte) []int {
	fset := token.NewFileSet()
	f := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(f, src, nil, 0)
	var lines []int
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			return lines
		}
		if tok != token.STRING {
			continue
		}
		if v, err := strconv.Unquote(lit); err == nil && pobCheckoutLiteral.MatchString(v) {
			lines = append(lines, fset.Position(pos).Line)
		}
	}
}

func TestPoBPathGuardSeesAHomeBuiltPath(t *testing.T) {
	t.Parallel()
	src := []byte("package x\n\nfunc f(home string) string {\n\treturn filepath.Join(home, \"PathOfBuilding\")\n}\n" +
		"var a = \"~/PathOfBuilding-PoE2\"\nvar b = `PathOfBuilding/src`\nvar c = \"<PathOfBuilding/>\"\n" +
		"var d = `xml:\"PathOfBuilding\"`\n// \"PathOfBuilding\"\n")
	got := pobCheckoutLiterals(src)
	want := []int{4, 6, 7}
	if !slices.Equal(got, want) {
		t.Fatalf("guard flagged lines %v, want %v", got, want)
	}
}

// TestPoBPathsComeFromTheResolver: a PoB checkout path built in a tool but outside pobpath.go drifts from the Taskfile's rule.
func TestPoBPathsComeFromTheResolver(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	scope := filepath.Join(repo, "scripts")
	seen := map[string]bool{}
	var bad []string
	err := filepath.WalkDir(scope, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != scope && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "testdata" || name == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(repo, path)
		rel = filepath.ToSlash(rel)
		if _, ok := pobPathGuardExclusions[rel]; ok {
			seen[rel] = true
			return nil
		}
		src, err := os.ReadFile(path) //nolint:gosec // walking this module
		if err != nil {
			return fmt.Errorf("guard: %w", err)
		}
		for _, line := range pobCheckoutLiterals(src) {
			bad = append(bad, rel+":"+strconv.Itoa(line))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for rel := range pobPathGuardExclusions {
		if !seen[rel] {
			t.Errorf("exclusion %s names no file; drop it", rel)
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		t.Fatalf("PoB checkout path built outside pobPinDir/pob2PinDir (use those, which mirror the Taskfile's POB_DEFAULT):\n%s",
			strings.Join(bad, "\n"))
	}
}
