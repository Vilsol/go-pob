package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Build.lua:910-914: a build saved with characterLevelAutoMode takes the level its allocated points need, on load
// and after every change.
func TestImportAppliesTheAutoLevelEstimate(t *testing.T) {
	s := testServer(t)
	xml, err := os.ReadFile(fixture("3.xml"))
	if err != nil {
		t.Fatal(err)
	}
	auto := strings.Replace(string(xml), `level="90"`, `level="100" characterLevelAutoMode="true"`, 1)
	path := filepath.Join(t.TempDir(), "auto.xml")
	if err := os.WriteFile(path, []byte(auto), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := call(t, s, "import_build", map[string]any{"source": path})["summary"].(map[string]any)
	need := sum["points"].(map[string]any)["level_needed"].(float64)
	if need >= 100 || sum["level"] != need {
		t.Fatalf("auto-level import: level %v, points need %v", sum["level"], need)
	}
	fixed := call(t, s, "import_build", map[string]any{"source": fixture("3.xml")})["summary"].(map[string]any)
	if fixed["level"] != float64(90) {
		t.Fatalf("a build without auto mode changed level to %v", fixed["level"])
	}
}
