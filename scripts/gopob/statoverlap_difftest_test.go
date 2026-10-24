package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const soSh = "scripts/stat_overlap.sh"

var soSets = map[string][]string{
	"Short":            {"1201-Deadeye.v2", "3542-Necromancer_Aura", "9001-Slayer-x"},
	"MediumLengthStat": {"1201-Deadeye.v2", "3542-Necromancer_Aura", "4444-Trickster"},
	"AVeryLongStatNameThatOverflowsTheColumnAndThenSome": {
		"1201-Deadeye.v2", "2000-A", "3542-Necromancer_Aura", "5000-B", "6000-C", "7000-D", "8000-E",
	},
}

const fakeGo = "#!/usr/bin/env bash\nexit 0\n"

const fakeGoProductive = `#!/usr/bin/env bash
cat <<'IDS'
=== RUN   TestStatRatios
recovered while evaluating 0007-Aa_b.c-D
=== STAT RATIOS: Extracted (table "", key "Extracted") ===
    9001-Slayer-x had ratio 1.5
    1201-Deadeye.v2 had ratio 2.0
    9001-Slayer-x again
=== STAT RATIOS: Extracted2 (table "Minion.", key "Extracted2") ===
    3542-Necromancer_Aura here
    2000-A
IDS
exit 1
`

func shadowBin(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	//nolint:gosec // a shadow binary in a per-test temp dir; it has to be executable to shadow
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")
}

func soSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, soSh)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			"go.mod": "module github.com/Vilsol/go-pob\n\ngo 1.26.2\n",
			soSh:     string(sh),
		}
		for stat, ids := range soSets {
			files[filepath.Join("out", stat+".txt")] = strings.Join(ids, "\n") + "\n"
		}
		for rel, body := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func TestStatOverlapPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	requirePOSIX(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := soSandbox(t, repo)
	emptyGo := shadowBin(t, "go", fakeGo)
	productiveGo := shadowBin(t, "go", fakeGoProductive)

	shSrc := difftest.ReadOriginal(t, repo, soSh)

	twoStats := "Short MediumLengthStat AVeryLongStatNameThatOverflowsTheColumnAndThenSome"
	spec := difftest.Spec{
		Tool:         "stat-overlap",
		PythonScript: soSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		GoBin:        bin,
		GoArgs:       []string{"stat-overlap"},
		Env:          []string{"STATS=" + twoStats, "OUT=out"},
		Stderr:       difftest.StderrExact,
		RefusalSites: 7,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-stats", Refusal: `REFUSING: set STATS="<stat> <stat> ..." (2 or more).`,
				Break: func(string) error { return nil }},

			{Name: "one-stat", Refusal: "REFUSING: overlap needs at least 2 stats; got 1.",
				Break: func(string) error { return nil }},

			{Name: "empty-extraction", Refusal: "REFUSING: extracted 0 failing builds for 'Missing'",
				Break: func(string) error { return nil }},

			{Name: "extraction-runs", Passes: true, Break: func(string) error { return nil }},

			{Name: "no-go-mod", Refusal: "no go.mod declaring module github.com/Vilsol/go-pob",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "go.mod")) }},

			{Name: "bad-gopob-root", Refusal: "GOPOB_ROOT does not contain a go.mod declaring module",
				Break: func(string) error { return nil }},

			{Name: "cd-into-gopob-root-fails", NoFixture: true,
				Refusal: "cd into GOPOB_ROOT fails after its go.mod was read",
				Note: "unreachable without a race between reading go.mod and chdir; the Go port " +
					"has no separate chdir at all, so there is nothing to diverge"},
			{Name: "cd-into-resolved-root-fails", NoFixture: true,
				Refusal: "cd into the walked-up root fails after its go.mod was read",
				Note:    "same: defensive, unreachable from any sandbox state"},

			{Name: "exactly-two-stats", Passes: true, Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range spec.Fixtures {
		if f.NoFixture {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			switch f.Name {
			case "no-stats":
				s.Env = []string{"STATS=", "OUT=out"}
			case "one-stat":
				s.Env = []string{"STATS=Short", "OUT=out"}
			case "empty-extraction":
				s.Env = []string{"STATS=Short Missing", "OUT=out", emptyGo}
			case "bad-gopob-root":
				s.Env = append(append([]string{}, spec.Env...), "GOPOB_ROOT=out")
			case "exactly-two-stats":
				s.Env = []string{"STATS=Short MediumLengthStat", "OUT=out"}
			case "extraction-runs":
				s.Env = []string{"STATS=Short Extracted Extracted2", "OUT=out", productiveGo}
				s.Files = []string{"out/Extracted.txt", "out/Extracted2.txt"}
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestStatOverlapIsStableAcrossLocales: LC_ALL, NOT PYTHONHASHSEED.
func TestStatOverlapIsStableAcrossLocales(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := soSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "stat-overlap",
		PythonScript: soSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		Env:          []string{"STATS=Short MediumLengthStat", "OUT=out"},
	}, mk, []string{"C", "en_US.utf8", "C.utf8"})
}
