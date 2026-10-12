package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

func foldProgress(args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: fold-progress takes no arguments, got %q\n", args[0])
		return 64
	}
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return root.RefusalCode
	}

	ported, err := portedScripts(filepath.Join(repo, "scripts", "gopob"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: cannot read the ports: %v\n", err)
		return root.RefusalCode
	}

	groups := []struct{ label, dir, glob string }{
		{"scripts/*.py", "scripts", "*.py"},
		{"scripts/*.sh", "scripts", "*.sh"},
		{"oracle/*.py", "oracle", "*.py"},
		{"oracle/*.sh", "oracle", "*.sh"},
	}

	if difftest.OriginalsDeleted(repo) {
		return foldComplete(repo, groups, len(ported))
	}
	var totF, totL, totS, doneF, doneL, doneS int
	fmt.Printf("%-14s %14s %18s %16s\n", "", "files", "LOC", "refusal sites")
	for _, g := range groups {
		paths, err := filepath.Glob(filepath.Join(repo, g.dir, g.glob))
		if err != nil {
			fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
			return root.RefusalCode
		}
		sort.Strings(paths)
		var f, l, s, df, dl, ds int
		for _, p := range paths {
			rel, lines, sites := statScript(repo, p)
			f++
			l += lines
			s += sites
			if ported[rel] {
				df++
				dl += lines
				ds += sites
			}
		}
		fmt.Printf("%-14s %6d /%6d %8d /%8d %7d /%7d\n", g.label, df, f, dl, l, ds, s)
		totF += f
		totL += l
		totS += s
		doneF += df
		doneL += dl
		doneS += ds
	}
	fmt.Printf("%-14s %6d /%6d %8d /%8d %7d /%7d\n", "TOTAL", doneF, totF, doneL, totL, doneS, totS)
	fmt.Println()
	fmt.Printf("ported %.1f%% of files, %.1f%% of LOC, %.1f%% of refusal sites\n",
		pct(doneF, totF), pct(doneL, totL), pct(doneS, totS))
	fmt.Printf("originals still on disk: %d (the spec's completion test; it reaches 0 only at "+
		"step 11, when the harness no longer needs them)\n", totF)
	return 0
}

var foldKept = map[string]string{
	"scripts/gate_versions.sh": "sourceable shim — a `.` in the caller's shell, which a binary " +
		"cannot be. Its version TABLE is already Go (internal/gateversions); what remains is the " +
		"three-line export.",
}

func foldComplete(repo string, groups []struct{ label, dir, glob string }, specs int) int {
	var stray []string
	kept := 0
	for _, g := range groups {
		paths, err := filepath.Glob(filepath.Join(repo, g.dir, g.glob))
		if err != nil {
			fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
			return root.RefusalCode
		}
		sort.Strings(paths)
		for _, p := range paths {
			rel, _ := filepath.Rel(repo, p)
			rel = filepath.ToSlash(rel)
			if _, ok := foldKept[rel]; ok {
				kept++
				continue
			}
			stray = append(stray, rel)
		}
	}

	fmt.Println("THE FOLD IS COMPLETE — the originals are gone and the goldens are the contract.")
	fmt.Println()
	fmt.Printf("  %d tools carry a Go implementation named by a difftest.Spec\n", specs)
	fmt.Printf("  %d declared exception(s) remain in the tree:\n", kept)
	for _, rel := range sortedKeys(foldKept) {
		if _, err := os.Stat(filepath.Join(repo, rel)); err == nil {
			fmt.Printf("    %s — %s\n", rel, foldKept[rel])
		}
	}
	fmt.Println()
	if len(stray) > 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: %d original(s) present that the fold retired and "+
			"foldKept does not declare:\n", len(stray))
		for _, s := range stray {
			fmt.Fprintf(os.Stderr, "  %s\n", s)
		}
		fmt.Fprintln(os.Stderr, "  Either it is a deliberate exception — add it to foldKept WITH "+
			"THE REASON — or the fold has been partly undone.")
		return 1
	}
	fmt.Println("No retired-language script has crept back.")
	return 0
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func statScript(repo, path string) (string, int, int) {
	rel, _ := filepath.Rel(repo, path)
	rel = filepath.ToSlash(rel)
	b, err := os.ReadFile(path) //nolint:gosec // a repo-relative path this tool enumerated
	if err != nil {
		return rel, 0, 0
	}
	lines := strings.Count(string(b), "\n")
	if len(b) > 0 && !strings.HasSuffix(string(b), "\n") {
		lines++
	}
	sites := 0
	switch {
	case strings.HasSuffix(rel, ".py"):
		sites = difftest.RefusalSites(b)
	case strings.HasSuffix(rel, ".sh"):
		sites = difftest.ShellRefusalSites(b)
	}
	return rel, lines, sites
}

func portedScripts(dir string) (map[string]bool, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", dir, err)
	}
	fset := token.NewFileSet()
	out := map[string]bool{}
	{
		files := make([]*ast.File, 0, len(paths))
		consts := map[string]string{}
		for _, path := range paths {
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return nil, fmt.Errorf("parsing %s: %w", path, err)
			}
			files = append(files, file)
			for _, d := range file.Decls {
				gd, ok := d.(*ast.GenDecl)
				if !ok || gd.Tok != token.CONST {
					continue
				}
				for _, sp := range gd.Specs {
					vs, ok := sp.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range vs.Names {
						if i < len(vs.Values) {
							if s, ok := stringOf(vs.Values[i], nil); ok {
								consts[name.Name] = s
							}
						}
					}
				}
			}
		}
		for _, file := range files {
			ast.Inspect(file, func(n ast.Node) bool {
				cl, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				switch {
				case isDifftestSpec(cl.Type):
					scanSpecLit(cl, consts, out)
				case isDifftestSpecContainer(cl.Type):
					for _, elt := range cl.Elts {
						e := elt
						if kv, ok := elt.(*ast.KeyValueExpr); ok {
							e = kv.Value
						}
						if inner, ok := e.(*ast.CompositeLit); ok {
							scanSpecLit(inner, consts, out)
						}
					}
				}
				return true
			})
		}
	}
	return out, nil
}

func scanSpecLit(cl *ast.CompositeLit, consts map[string]string, out map[string]bool) {
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		k, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch k.Name {
		case "PythonScript":
			if s, ok := stringOf(kv.Value, consts); ok && s != "" {
				out[s] = true
			}
		case "Library":
			lit, ok := kv.Value.(*ast.CompositeLit)
			if !ok {
				continue
			}
			for _, e := range lit.Elts {
				if s, ok := stringOf(e, consts); ok && s != "" {
					out[s] = true
				}
			}
		}
	}
}

func isDifftestSpecContainer(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.MapType:
		return isDifftestSpec(t.Value)
	case *ast.ArrayType:
		return isDifftestSpec(t.Elt)
	}
	return false
}

func isDifftestSpec(e ast.Expr) bool {
	se, ok := e.(*ast.SelectorExpr)
	if !ok || se.Sel.Name != "Spec" {
		return false
	}
	x, ok := se.X.(*ast.Ident)
	return ok && x.Name == "difftest"
}

func stringOf(e ast.Expr, consts map[string]string) (string, bool) {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(v.Value)
		return s, err == nil
	case *ast.Ident:
		s, ok := consts[v.Name]
		return s, ok
	}
	return "", false
}
