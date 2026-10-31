package moddb

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The kernel is shared by both games, so a game's data reaches it only through a hook.
func TestKernelImportsNoGameData(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		checked++
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if path == "github.com/Vilsol/go-pob/data/raw" {
				t.Errorf("%s imports %s", name, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no non-test Go files read")
	}
}
