package main

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const batchedFixture = `=== RUN   TestStatRatios
recovered while evaluating 0007-Preamble_Build.x

=== STAT RATIOS: Alpha (table "", key "Alpha") ===
CORPUS testdata/corpus-3.29 (game data 3.29)
ok 3 · WRONG-NONZERO 2 in 1 ratio clusters
    0.5000      2   9001-Slayer-x, 1201-Deadeye.v2
                    e.g. 1201-Deadeye.v2  PoB 10  ours 5

=== CLASS=wrong for Alpha — 2 build(s) ===
  9001-Slayer-x       PoB 10  ours 5
  1201-Deadeye.v2     PoB 10  ours 5

=== STAT RATIOS: Beta (table "Minion.", key "Beta") ===
CORPUS testdata/corpus-3.29 (game data 3.29)
    2.0000      2   3542-Necromancer_Aura, 1201-Deadeye.v2

=== CLASS=wrong for Beta — 2 build(s) ===
  3542-Necromancer_Aura  PoB 1  ours 2
  1201-Deadeye.v2        PoB 1  ours 2
--- PASS: TestStatRatios (34.70s)
PASS
ok  	github.com/Vilsol/go-pob/calculator	34.812s
`

// TestSplitBatchedStatRatiosAttributesBySection is the cheap half of the equivalence argument: it pins WHICH ids land in WHICH set, without a corpus.
func TestSplitBatchedStatRatiosAttributesBySection(t *testing.T) {
	t.Parallel()
	got, missing := splitBatchedStatRatios(batchedFixture, []string{"Alpha", "Beta"})
	if len(missing) != 0 {
		t.Fatalf("both sections are present, got missing %v", missing)
	}
	want := map[string][]string{
		"Alpha": {"0007-Preamble_Build.x", "1201-Deadeye.v2", "9001-Slayer-x"},
		"Beta":  {"0007-Preamble_Build.x", "1201-Deadeye.v2", "3542-Necromancer_Aura"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attribution:\n got %v\nwant %v", got, want)
	}
	for s, ids := range got {
		if !sort.StringsAreSorted(ids) {
			t.Errorf("%s: not in byte order: %v", s, ids)
		}
	}
}

// TestSplitBatchedStatRatiosNamesAMissingSection is the "empty must never mean I did not look" half.
func TestSplitBatchedStatRatiosNamesAMissingSection(t *testing.T) {
	t.Parallel()
	sets, missing := splitBatchedStatRatios(batchedFixture, []string{"Alpha", "Gamma"})
	if !reflect.DeepEqual(missing, []string{"Gamma"}) {
		t.Fatalf("expected Gamma named as missing, got %v", missing)
	}
	if _, ok := sets["Gamma"]; ok {
		t.Errorf("a stat with no section must have NO set, got %v", sets["Gamma"])
	}
}

var equivStats = []string{"CombinedDPS", "TotalDPS"}

// TestStatRatiosBatchEqualsSingleRuns is the whole licence for batching.
func TestStatRatiosBatchEqualsSingleRuns(t *testing.T) {
	t.Parallel()
	if os.Getenv("OPERATOR") != "true" || os.Getenv("CORPUS") != "true" {
		t.Skip("set OPERATOR=true CORPUS=true (task test-operator-slow) — this compares K+1 real corpus passes")
	}
	repo := repoRoot(t)
	ctx := t.Context()

	batched, missing := runStatRatiosBatch(ctx, repo, equivStats)
	if len(missing) != 0 {
		t.Fatalf("the batched run printed no section for %v — it did not measure them", missing)
	}
	for _, s := range equivStats {
		single := runStatRatios(ctx, repo, s)
		if len(single) == 0 {
			t.Fatalf("%s extracted 0 builds from the SINGLE form — the fixture has no work, so an "+
				"equivalence over it would compare two empty sets", s)
		}
		if !reflect.DeepEqual(batched[s], single) {
			t.Errorf("%s: batched and single disagree\n batched %d ids\n single  %d ids\n"+
				" only-batched %s\n only-single %s",
				s, len(batched[s]), len(single),
				soHead(soOnlyIn(batched[s], single)), soHead(soOnlyIn(single, batched[s])))
		}
	}
}

func soHead(ids []string) string {
	if len(ids) <= 8 {
		return strings.Join(ids, " ")
	}
	return fmt.Sprintf("%s … (%d total)", strings.Join(ids[:8], " "), len(ids))
}

func soOnlyIn(a, b []string) []string {
	in := make(map[string]bool, len(b))
	for _, s := range b {
		in[s] = true
	}
	var out []string
	for _, s := range a {
		if !in[s] {
			out = append(out, s)
		}
	}
	return out
}
