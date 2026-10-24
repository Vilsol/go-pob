package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/builds/parsecache"
	"github.com/Vilsol/go-pob/data/raw"
)

func corpusDirForVersion(version string) (string, error) {
	registerVersionInTestCacheKey()
	if version == "" {
		return "", fmt.Errorf("REFUSING: no game-data version — nothing says which corpus to walk")
	}
	dir := filepath.Join("..", "testdata", "corpus-"+version)
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("REFUSING: no corpus for game-data version %q at %s: %w\n"+
			"The corpus and the loaded game data must be the same league or the measurement is "+
			"meaningless. Re-run with POE_VERSION set to a version that has a corpus, or fetch it with "+
			"`mise run testdata`.", version, dir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("REFUSING: %s is not a directory", dir)
	}
	return dir, nil
}

func corpusDirForLoadedData(t testing.TB) string {
	t.Helper()
	dir, err := corpusDirForVersion(raw.LatestVersion)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func corpusNamesForLoadedData(t *testing.T) (string, []string) {
	t.Helper()
	dir := corpusDirForLoadedData(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatalf("REFUSING: %s holds no .xml builds", dir)
	}
	sort.Strings(names)
	return dir, names
}

func oracleBlobForVersion(version string) (string, error) {
	if version == "" {
		return "", fmt.Errorf("REFUSING: no game-data version — nothing says which oracle to load")
	}
	path := filepath.Join("..", "testdata", "oracle-"+version, "output-"+version+".jsonl.gz")
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("REFUSING: no oracle blob for game-data version %q at %s: %w (fetch it with `mise run testdata`)",
			version, path, err)
	}
	return path, nil
}

var oracleForLoadedData = sync.OnceValues(func() (map[string]oracleRecord, error) {
	path, err := oracleBlobForVersion(raw.LatestVersion)
	if err != nil {
		return nil, err
	}
	return loadOutputOracle(path)
})

func oracleExcludedForLoadedData(t *testing.T) int {
	t.Helper()
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	n := 0
	for _, rec := range oracle {
		if rec.TreeSources < oracleMinTreeSources {
			n++
		}
	}
	return n
}

var corpusTestCaches sync.Map

func corpusCacheForTest(t *testing.T) *EnvironmentCache {
	t.Helper()
	if c, ok := corpusTestCaches.Load(t.Name()); ok {
		return c.(*EnvironmentCache)
	}
	c, _ := corpusTestCaches.LoadOrStore(t.Name(), NewEnvironmentCache())
	return c.(*EnvironmentCache)
}

func corpusEnvForLoadedData(t *testing.T, file string) *Environment {
	t.Helper()
	if !strings.HasSuffix(file, ".xml") {
		file += ".xml"
	}
	rawXML, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), file))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := parsecache.ParseCached(rawXML)
	if err != nil {
		t.Fatalf("REFUSING: %s did not parse: %v", file, err)
	}
	return BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, corpusCacheForTest(t))
}

// dumpsDirCandidate resolves DUMPS without a test; a non-empty second value says why there is no dump set.
func dumpsDirCandidate() (string, string) {
	home, homeErr := os.UserHomeDir()
	dir := os.Getenv("DUMPS")
	switch {
	case dir == "":
		if homeErr != nil {
			return "", fmt.Sprintf("DUMPS unset and no home directory: %v", homeErr)
		}
		dir = filepath.Join(home, "go-pob-corpus", "dumps-"+raw.LatestVersion)
	case strings.HasPrefix(dir, "~/") && homeErr == nil:
		dir = filepath.Join(home, dir[2:])
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return dir, fmt.Sprintf("no dump set at %s — set DUMPS to a dump of the %s corpus "+
			"(scripts/dump_shard.sh), or leave it unset to skip", dir, raw.LatestVersion)
	}
	return dir, ""
}

func dumpsDirForLoadedData(t *testing.T) string {
	t.Helper()
	dir, skip := dumpsDirCandidate()
	if skip != "" {
		t.Skip(skip)
	}

	_, names := corpusNamesForLoadedData(t)
	covered, sampled := dumpsCoverCorpus(dir, names)
	if sampled == 0 || covered*2 < sampled {
		t.Fatalf("REFUSING: %s covers %d of %d sampled %s corpus builds. The dumps and the loaded "+
			"game data must be the same league or every mod reads as diverging — which looks "+
			"exactly like the defect this instrument exists to find.",
			dir, covered, sampled, raw.LatestVersion)
	}
	return dir
}

func dumpsCoverCorpus(dir string, names []string) (int, int) {
	covered, sampled := 0, 0
	for i := 0; i < len(names) && sampled < 40; i += 1 + len(names)/40 {
		sampled++
		stem := strings.TrimSuffix(names[i], ".xml")
		if _, err := os.Stat(filepath.Join(dir, stem+".json")); err == nil {
			covered++
		}
	}
	return covered, sampled
}

// TestDumpsCoverageSeparatesTheLeagues is the control for dumpsDirForLoadedData's content check.
func TestDumpsCoverageSeparatesTheLeagues(t *testing.T) {
	_, names := corpusNamesForLoadedData(t)

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory: %v", err)
	}
	right := filepath.Join(home, "go-pob-corpus", "dumps-"+raw.LatestVersion)
	if _, err := os.Stat(right); err != nil {
		t.Skipf("no dump set at %s to control against", right)
	}

	if covered, sampled := dumpsCoverCorpus(right, names); covered*2 < sampled {
		t.Fatalf("the %s dump set covers only %d of %d sampled builds — the guard would reject "+
			"the CORRECT directory", raw.LatestVersion, covered, sampled)
	}

	entries, err := os.ReadDir(filepath.Join(home, "go-pob-corpus"))
	if err != nil {
		t.Skipf("no dump root to scan: %v", err)
	}
	mine := "dumps-" + raw.LatestVersion
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "dumps-") || strings.HasPrefix(e.Name(), mine) {
			continue
		}
		dir := filepath.Join(home, "go-pob-corpus", e.Name())
		covered, sampled := dumpsCoverCorpus(dir, names)
		if covered*2 >= sampled {
			t.Fatalf("%s covers %d of %d sampled %s builds — the wrong-league guard does not fire",
				dir, covered, sampled, raw.LatestVersion)
		}
		return
	}
	t.Skip("no other-league dump set on disk to drive the negative arm")
}

// TestCorpusDirRefusesAVersionWithNoCorpus is the refusal control.
func TestCorpusDirRefusesAVersionWithNoCorpus(t *testing.T) {
	if _, err := corpusDirForVersion("0.1-no-such-league"); err == nil {
		t.Fatal("corpusDirForVersion accepted a version with no corpus directory")
	}
	if _, err := corpusDirForVersion(""); err == nil {
		t.Fatal("corpusDirForVersion accepted an empty version")
	}
	if _, err := oracleBlobForVersion("0.1-no-such-league"); err == nil {
		t.Fatal("oracleBlobForVersion accepted a version with no blob")
	}

	if _, err := corpusDirForVersion(raw.LatestVersion); err != nil {
		t.Fatalf("corpusDirForVersion refused the loaded version %q: %v", raw.LatestVersion, err)
	}
}
