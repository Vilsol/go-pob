package calculator

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
)

// TestCorpusToolsFollowLoadedGameData is the ACCEPTANCE test for the sizing instruments.
func TestCorpusToolsFollowLoadedGameData(t *testing.T) {
	requireOperatorRun(t, "repo hygiene: the instruments follow POE_VERSION")
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}

	bin := os.Args[0]
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("cannot re-exec the test binary (%s): %s", bin, err)
	}

	entries, err := os.ReadDir("../testdata/corpus-" + raw.LatestVersion)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	want := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") {
			want++
		}
	}
	if want == 0 {
		t.Fatalf("REFUSING: no builds under ../testdata/corpus-%s, so this test could not fail",
			raw.LatestVersion)
	}

	t.Run("corpus-predicate", func(t *testing.T) {
		cmd := exec.CommandContext(t.Context(), bin, "-test.run=^TestCorpusPredicate$", "-test.v")
		cmd.Env = append(os.Environ(), "PREDICATE=true", "CORPUS=true")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("child failed: %v\n%s", err, out)
		}

		m := regexp.MustCompile(`MATCHED \d+ of (\d+) builds`).FindSubmatch(out)
		if m == nil {
			t.Fatalf("corpus-predicate printed no MATCHED line under %s game data — it either "+
				"SKIPPED (a version guard pinned to another corpus) or refused.\n%s",
				raw.LatestVersion, out)
		}
		got, _ := strconv.Atoi(string(m[1]))
		if got != want {
			t.Errorf("corpus-predicate walked %d builds; the corpus for the loaded %s game data "+
				"holds %d. A sizing tool measuring a different league than the engine loaded "+
				"produces a full, confident, meaningless number.", got, raw.LatestVersion, want)
		}
	})

	t.Run("corpus-dump", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "dump.txt.gz")

		cmd := exec.CommandContext(t.Context(), bin, "-test.run=^TestCorpusDump$", "-test.v")
		cmd.Env = append(os.Environ(), "DUMP="+path)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("child failed: %v\n%s", err, out)
		}

		recs, meta, err := readRunDump(path)
		if err != nil {
			t.Fatalf("the dump the run-differ produced is unreadable: %v", err)
		}
		if len(recs) != want {
			t.Errorf("corpus-dump wrote %d build records; the corpus for the loaded %s game data "+
				"holds %d builds", len(recs), raw.LatestVersion, want)
		}
		if meta.dataVersion != raw.LatestVersion {
			t.Errorf("the dump records game-data version %q, this process loaded %q",
				meta.dataVersion, raw.LatestVersion)
		}
	})
}

// TestCorpusGrepFollowsLoadedVersion pins the PRESENCE rung to the same corpus as the other two.
func TestCorpusGrepFollowsLoadedVersion(t *testing.T) {
	requireOperatorRun(t, "repo hygiene: the instruments follow POE_VERSION")

	entries, err := os.ReadDir("../testdata/corpus-" + raw.LatestVersion)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	want := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") {
			want++
		}
	}

	run := func(t *testing.T, pattern string) []byte {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "go", "run", "./scripts/gopob", "corpus-grep")
		cmd.Dir = ".."
		cmd.Env = append(os.Environ(), "PATTERN="+pattern, "POE_VERSION="+raw.LatestVersion)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("corpus-grep failed: %v\n%s", err, out)
		}
		return out
	}

	denominator := func(t *testing.T, out []byte) int {
		t.Helper()
		m := regexp.MustCompile(`MATCHED \d+ of (\d+) builds`).FindSubmatch(out)
		if m == nil {
			t.Fatalf("corpus-grep printed no MATCHED line:\n%s", out)
		}
		n, _ := strconv.Atoi(string(m[1]))
		return n
	}

	miss := run(t, "ZZZ-no-such-mod-line-ZZZ")
	if got := denominator(t, miss); got != want {
		t.Errorf("corpus-grep scanned %d builds for the negative control; corpus-%s holds %d",
			got, raw.LatestVersion, want)
	}
	if !strings.Contains(string(miss), "MATCHED 0 of") {
		t.Errorf("the negative control matched something:\n%s", miss)
	}

	hit := run(t, "Life")
	if got := denominator(t, hit); got != want {
		t.Errorf("corpus-grep scanned %d builds; corpus-%s holds %d", got, raw.LatestVersion, want)
	}
	if strings.Contains(string(hit), "MATCHED 0 of") {
		t.Errorf("the positive control matched nothing, so the scan is broken:\n%s", hit)
	}
}

// TestStatRatiosFollowsLoadedGameData pins the target-finder to the authoritative corpus.
func TestStatRatiosFollowsLoadedGameData(t *testing.T) {
	requireOperatorRun(t, "repo hygiene: the instruments follow POE_VERSION")
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	bin := os.Args[0]
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("cannot re-exec the test binary (%s): %s", bin, err)
	}

	cmd := exec.CommandContext(t.Context(), bin, "-test.run=^TestStatRatios$", "-test.v")
	cmd.Env = append(os.Environ(), "CORPUS=true", "STAT=Life")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child failed: %v\n%s", err, out)
	}

	want := "CORPUS ../testdata/corpus-" + raw.LatestVersion
	if !strings.Contains(string(out), want) {
		t.Fatalf("stat-ratios did not report %q under %s game data — it either SKIPPED (a version "+
			"guard pinned to another corpus) or graded another league.\n%s",
			want, raw.LatestVersion, out)
	}
	if !strings.Contains(string(out), "=== STAT RATIOS: Life") {
		t.Fatalf("stat-ratios printed no table:\n%s", out)
	}
}

// TestBuildRankingFollowsLoadedGameData is the same treatment for the per-build ranking.
func TestBuildRankingFollowsLoadedGameData(t *testing.T) {
	requireOperatorRun(t, "repo hygiene: the instruments follow POE_VERSION")
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	bin := os.Args[0]
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("cannot re-exec the test binary (%s): %s", bin, err)
	}

	cmd := exec.CommandContext(t.Context(), bin, "-test.run=^TestBuildRanking$", "-test.v")
	cmd.Env = append(os.Environ(), "CORPUS=true", "TOP=1", "STATS=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child failed: %v\n%s", err, out)
	}

	want := "CORPUS ../testdata/corpus-" + raw.LatestVersion
	if !strings.Contains(string(out), want) {
		t.Fatalf("build-ranking did not report %q under %s game data.\n%s",
			want, raw.LatestVersion, out)
	}
}
