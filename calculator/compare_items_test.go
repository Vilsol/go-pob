package calculator

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// TestCompareItemExpandedAgainstPoB: CompareTab.lua:3466-3637: the expanded card's strings for 73 item pairs.
func TestCompareItemExpandedAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	raw, err := os.ReadFile("../testdata/ui-misc/compare_item_expanded.json")
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Pairs [][2]string   `json:"pairs"`
		Cards [][2][]string `json:"cards"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Pairs) < 50 || len(fx.Pairs) != len(fx.Cards) {
		t.Fatalf("REFUSING: %d pairs, %d cards", len(fx.Pairs), len(fx.Cards))
	}
	for k, p := range fx.Pairs {
		for side, args := range [][2]string{{p[0], p[1]}, {p[1], p[0]}} {
			got, want := CompareItemExpanded(args[0], args[1]), fx.Cards[k][side]
			if !slices.Equal(got, want) {
				t.Errorf("pair %d side %d:\n got  %q\n want %q", k, side, got, want)
			}
		}
	}
}
