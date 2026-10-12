package calculator

import (
	"encoding/json"
	"os"
	"testing"
)

// TestSpectreLibraryAgainstPoB: Produced by oracle/spectre_library.lua from the pinned PoB.
func TestSpectreLibraryAgainstPoB(t *testing.T) {
	raw, err := os.ReadFile("../testdata/oracle/spectre_library_expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var want []struct {
		ID    string   `json:"id"`
		Lines []string `json:"lines"`
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	got := SpectreLibrary()
	if len(got) != len(want) || len(want) < 100 {
		t.Fatalf("%d spectres, PoB %d", len(got), len(want))
	}
	wrong := 0
	for i, w := range want {
		g := got[i]
		if g.ID != w.ID {
			t.Fatalf("row %d: %s, PoB %s", i, g.ID, w.ID)
		}
		if !sameStrings(g.Tooltip, w.Lines) {
			wrong++
			if wrong <= 5 {
				t.Errorf("%s:\n got %q\nPoB %q", w.ID, g.Tooltip, w.Lines)
			}
		}
	}
	if wrong > 0 {
		t.Errorf("%d of %d tooltips differ", wrong, len(want))
	}
}
