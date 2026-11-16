package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

func TestImportBuildReadsACodeFromATextFile(t *testing.T) {
	s := testServer(t)
	xml, err := os.ReadFile(fixture("3.xml"))
	if err != nil {
		t.Fatal(err)
	}
	code, err := pob.CompressEncode(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "build code.txt")
	if err := os.WriteFile(path, []byte(code+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fromFile := call(t, s, "import_build", map[string]any{"source": path})
	fromCode := call(t, s, "import_build", map[string]any{"source": code})
	if fromFile["source_kind"] != "file" || stat(fromFile["summary"].(map[string]any), mainDPS) != stat(fromCode["summary"].(map[string]any), mainDPS) {
		t.Fatalf("from file %v %v, from code %v", fromFile["source_kind"], fromFile["summary"], fromCode["summary"])
	}
}
