package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestShelledScriptsStillExist is the harness for the shape "A PORT THAT SHELLS ITS OWN ORIGINAL IS NOT A PORT; IT IS A WRAPPER".
func TestShelledScriptsStillExist(t *testing.T) {
	root := repoRootForExecScan(t)

	type finding struct{ file, path string }
	var missing []finding

	err := filepath.WalkDir(filepath.Join(root, "scripts", "gopob"), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if perr != nil {
			t.Logf("%s: not parsed, so it was NOT scanned for dead shell-outs: %v", p, perr)
			return nil //nolint:nilerr // deliberate: the toolchain owns the parse failure, and the skip is logged
		}
		declared := declaredDeadLines(fset, f)
		rel, _ := filepath.Rel(root, p)
		ast.Inspect(f, func(n ast.Node) bool {
			for _, arg := range execProgramArgs(n) {
				script, ok := literalScriptPath(arg)
				if !ok {
					continue
				}
				if _, statErr := os.Stat(filepath.Join(root, script)); statErr == nil {
					continue
				}
				if declared[fset.Position(arg.Pos()).Line] {
					continue
				}
				missing = append(missing, finding{rel, script})
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking the tool tree: %v", err)
	}

	sort.Slice(missing, func(i, j int) bool {
		if missing[i].file != missing[j].file {
			return missing[i].file < missing[j].file
		}
		return missing[i].path < missing[j].path
	})
	for _, m := range missing {
		t.Errorf("%s shells %s, which does not exist in the repository\n"+
			"    the fold replaced it; point this call at the folded subcommand instead", m.file, m.path)
	}
}

func declaredDeadLines(fset *token.FileSet, f *ast.File) map[int]bool {
	lines := map[int]bool{}
	for _, group := range f.Comments {
		for _, c := range group.List {
			if !strings.Contains(c.Text, "fold-dead-ok:") {
				continue
			}
			ln := fset.Position(c.Pos()).Line
			lines[ln] = true
			lines[ln+1] = true
		}
	}
	return lines
}

func execProgramArgs(n ast.Node) []ast.Expr {
	switch e := n.(type) {
	case *ast.CallExpr:
		sel, ok := e.Fun.(*ast.SelectorExpr)
		if !ok {
			return nil
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "exec" {
			return nil
		}
		switch sel.Sel.Name {
		case "Command":
			return e.Args
		case "CommandContext":
			if len(e.Args) > 1 {
				return e.Args[1:]
			}
		}
	case *ast.CompositeLit:
		at, ok := e.Type.(*ast.ArrayType)
		if !ok {
			return nil
		}
		if id, ok := at.Elt.(*ast.Ident); !ok || id.Name != "string" {
			return nil
		}
		if len(e.Elts) == 0 {
			return nil
		}
		switch first := e.Elts[0].(type) {
		case *ast.BasicLit:
			if first.Kind != token.STRING {
				return nil
			}
			if s, err := strconv.Unquote(first.Value); err != nil || !strings.HasPrefix(s, "./") {
				return nil
			}
			return e.Elts[:1]
		case *ast.CallExpr:
			if id, ok := first.Fun.(*ast.Ident); ok && id.Name == "pyPathJoin" {
				return e.Elts[:1]
			}
			return nil
		}
		return nil
	}
	return nil
}

func literalScriptPath(e ast.Expr) (string, bool) {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(v.Value)
		if err != nil || !isScriptPath(s) {
			return "", false
		}
		return strings.TrimPrefix(s, "./"), true
	case *ast.CallExpr:
		id, ok := v.Fun.(*ast.Ident)
		if !ok || id.Name != "pyPathJoin" || len(v.Args) < 2 {
			return "", false
		}
		parts := make([]string, 0, len(v.Args)-1)
		for _, a := range v.Args[1:] {
			lit, ok := a.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return "", false
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return "", false
			}
			parts = append(parts, s)
		}
		joined := filepath.Join(parts...)
		if !isScriptPath(joined) {
			return "", false
		}
		return joined, true
	}
	return "", false
}

func isScriptPath(s string) bool {
	if strings.ContainsAny(s, " \t\n*?[") {
		return false
	}
	return strings.HasSuffix(s, ".sh") || strings.HasSuffix(s, ".py")
}

func repoRootForExecScan(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s — this test cannot locate the repository root", dir)
		}
		dir = parent
	}
}
