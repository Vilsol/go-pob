package mod

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// TestKeywordFlagsByNameCoversEveryConstant asserts.
func TestKeywordFlagsByNameCoversEveryConstant(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "common.go", nil, 0)
	if err != nil {
		t.Fatalf("REFUSING: cannot parse common.go, so this test cannot answer: %v", err)
	}

	declared := map[string]int64{}
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
			return true
		}
		name := vs.Names[0].Name
		if !strings.HasPrefix(name, "KeywordFlag") || name == "KeywordFlag" {
			return true
		}
		call, ok := vs.Values[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		if id, ok := call.Fun.(*ast.Ident); !ok || id.Name != "KeywordFlag" {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.INT {
			return true
		}
		v, err := strconv.ParseInt(lit.Value, 0, 64)
		if err != nil {
			t.Fatalf("cannot parse %s = %s: %v", name, lit.Value, err)
		}
		declared[strings.TrimPrefix(name, "KeywordFlag")] = v
		return true
	})

	if len(declared) < 20 {
		t.Fatalf("REFUSING: parsed only %d KeywordFlag constants from common.go; the parse is "+
			"broken and this test would check almost nothing", len(declared))
	}

	for name, want := range declared {
		got, ok := KeywordFlagsByName[name]
		if !ok {
			t.Errorf("KeywordFlag%s is declared but missing from KeywordFlagsByName", name)
			continue
		}
		if int64(got) != want {
			t.Errorf("KeywordFlagsByName[%q] = 0x%08x, the constant is 0x%08x", name, int64(got), want)
		}
	}
	for name := range KeywordFlagsByName {
		if _, ok := declared[name]; !ok {
			t.Errorf("KeywordFlagsByName has %q, which no KeywordFlag constant declares", name)
		}
	}
	if len(KeywordFlagsByName) != len(declared) {
		t.Errorf("KeywordFlagsByName has %d entries for %d constants", len(KeywordFlagsByName), len(declared))
	}
}
