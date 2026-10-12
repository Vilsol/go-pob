package builds

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var defaultVersionRe = regexp.MustCompile(`(?m)^const DefaultVersion = "([^"]+)"$`)

func corpusDirForLoadedData(t *testing.T) string {
	t.Helper()

	version := os.Getenv("POE_VERSION")
	if version == "" {
		src, err := os.ReadFile(filepath.Join("..", "data", "raw", "main.go"))
		if err != nil {
			t.Skipf("cannot read data/raw/main.go for DefaultVersion: %v", err)
		}
		m := defaultVersionRe.FindSubmatch(src)
		if m == nil {
			t.Fatal("REFUSING: data/raw/main.go declares no `const DefaultVersion = \"...\"`. " +
				"The corpus path would be a guess, and a guess here reads another league's builds.")
		}
		version = string(m[1])
	}

	dir := filepath.Join("..", "testdata", "corpus-"+version)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Skipf("no corpus for game-data version %q at %s — the corpus and the loaded game data "+
			"must be the same league or the measurement is meaningless (fetch it with `mise run testdata`)", version, dir)
	}
	return dir
}

// TestCorpusVersionResolutionTracksDefaultVersion is the control for the source read above.
func TestCorpusVersionResolutionTracksDefaultVersion(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "data", "raw", "main.go"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	m := defaultVersionRe.FindSubmatch(src)
	if m == nil {
		t.Fatal("data/raw/main.go no longer declares `const DefaultVersion = \"...\"` on its own " +
			"line. corpusDirForLoadedData reads it from there and would silently start skipping.")
	}
	if len(m[1]) == 0 {
		t.Fatal("DefaultVersion is empty")
	}
	t.Logf("DefaultVersion = %s", m[1])
}
