package calculator

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

// TestForeignModNames is a REPORT-ONLY census: which mod and tag names does this tree produce that appear in NO file of the pinned PoB?
func TestForeignModNames(t *testing.T) {
	pobRoot := os.Getenv("POB_DIR")
	if pobRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("NOT-RUN: cannot resolve $HOME to find the pinned PoB: %v", err)
		}
		pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
	}
	src := filepath.Join(pobRoot, "src")
	if st, err := os.Stat(src); err != nil || !st.IsDir() {
		t.Skipf("NOT-RUN: no pinned PoB source at %s (set POB_DIR). This census has NOT told you "+
			"there are no foreign names — it has told you it could not look.", src)
	}

	names := collectProducedModNames(t)
	if len(names) < 500 {
		t.Fatalf("REFUSING: extracted only %d mod names from calculator/; the call shapes this "+
			"census recognises have moved and it is no longer measuring anything", len(names))
	}

	tokens := pobIdentifierSet(t, src)
	if len(tokens) < 10000 {
		t.Fatalf("REFUSING: only %d identifiers in the pinned PoB Lua; the tree did not load", len(tokens))
	}

	if !tokens["CritChance"] {
		t.Fatal("REFUSING: control failed — `CritChance` is not in the pinned PoB identifier set, " +
			"so the set is broken and every name would report as foreign")
	}
	if tokens["ZzThisIsNotARealPoBIdentifier"] {
		t.Fatal("REFUSING: control failed — a synthetic name resolved, so the lookup matches anything")
	}

	var foreign, composed []string
	for _, n := range names {
		if tokens[modNameCore(n)] {
			continue
		}
		if strings.ContainsAny(n, "{ '") {
			composed = append(composed, n)
			continue
		}
		foreign = append(foreign, n)
	}
	sort.Strings(foreign)
	sort.Strings(composed)

	t.Logf("FOREIGN MOD NAMES: %d of %d produced names appear in no .lua of the pinned PoB",
		len(foreign)+len(composed), len(names))
	t.Logf("  CHECKABLE (%d) — real candidates; some are legitimate Go-side intermediates:", len(foreign))
	for _, n := range foreign {
		t.Logf("    %s", n)
	}
	t.Logf("  RUNTIME-COMPOSED (%d) — an identifier lookup cannot resolve these; judge by hand:", len(composed))
	for _, n := range composed {
		t.Logf("    %s", n)
	}
}

func collectProducedModNames(t *testing.T) []string {
	t.Helper()

	constructors := map[string]bool{
		"NewFloat": true, "NewFlag": true, "MOD": true, "FLAG": true,
		"Condition": true, "Multiplier": true, "ActorCondition": true,
	}

	seen := map[string]bool{}
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("REFUSING: cannot read the package directory: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("REFUSING: cannot parse %s: %v", name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			var fn string
			switch e := call.Fun.(type) {
			case *ast.SelectorExpr:
				fn = e.Sel.Name
			case *ast.Ident:
				fn = e.Name
			}
			if !constructors[fn] {
				return true
			}
			arg := call.Args[0]
			if fn == "ActorCondition" && len(call.Args) > 1 {
				arg = call.Args[1]
			}
			lit, ok := arg.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			v, err := strconv.Unquote(lit.Value)
			if err != nil || v == "" {
				return true
			}
			seen[v] = true
			return true
		})
	}

	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func modNameCore(n string) string {
	if i := strings.LastIndex(n, ":"); i >= 0 {
		n = n[i+1:]
	}
	return n
}

func pobIdentifierSet(t *testing.T, src string) map[string]bool {
	t.Helper()

	var files []string
	if err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "TreeData" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".lua") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("REFUSING: cannot walk the pinned PoB source: %v", err)
	}

	tokens := map[string]bool{}
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("REFUSING: cannot read %s: %v", path, err)
		}
		start := -1
		for i := 0; i <= len(b); i++ {
			var c byte
			if i < len(b) {
				c = b[i]
			}
			isIdent := c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
			switch {
			case isIdent && start < 0:
				start = i
			case !isIdent && start >= 0:
				tokens[string(b[start:i])] = true
				start = -1
			}
		}
	}
	return tokens
}
