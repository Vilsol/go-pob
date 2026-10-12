package calculator

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// TestSplitPersonalityPathAgainstPoB grades the nodes the tree draws in Split Personality's colour against the pin's spec.splitPersonalityPath on a fresh load.
func TestSplitPersonalityPathAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	raw, err := os.ReadFile("../testdata/tree-ui/split_personality_path.json")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var want map[string][]int64
	if err := json.Unmarshal(raw, &want); err != nil || len(want) < 10 {
		t.Fatalf("REFUSING: the fixture does not decode: %v", err)
	}
	nonEmpty := 0
	for file, ids := range want {
		got := GetMiscCalculator(loadCorpusBuild(t, file)).SplitPersonalityPath()
		if len(ids) > 0 {
			nonEmpty++
		}
		if !slices.Equal(got, ids) {
			t.Errorf("%s: ours %d nodes, the pin's %d; ours lacks %v, has %v the pin lacks", file, len(got), len(ids), setMinus(ids, got), setMinus(got, ids))
		}
	}
	if nonEmpty < 10 {
		t.Fatalf("REFUSING: only %d builds in the fixture have a path", nonEmpty)
	}
}
