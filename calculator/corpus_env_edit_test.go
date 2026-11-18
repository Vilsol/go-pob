package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// corpusEnvWithEdit loads a corpus build after replacing old with replacement exactly once, so a test can pin a
// value PoB computed on the edited build.
func corpusEnvWithEdit(t *testing.T, file, old, replacement string) *Environment {
	t.Helper()
	rawXML, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), file+".xml"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	if n := strings.Count(string(rawXML), old); n != 1 {
		t.Fatalf("REFUSING: %s carries %q %d times, want exactly once", file, old, n)
	}
	b, err := builds.ParseBuild([]byte(strings.Replace(string(rawXML), old, replacement, 1)))
	if err != nil {
		t.Fatalf("REFUSING: edited %s did not parse: %v", file, err)
	}
	return (&Calculator{PoB: b}).BuildOutput(OutputModeMain)
}
