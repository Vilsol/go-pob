package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestItemBaseMatchKeepsTheLongest pins Item.lua:944-959's `e-s > bestMatch.length`.
func TestItemBaseMatchKeepsTheLongest(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"Awl", "Ring", "Sapphire Ring"} {
		if ItemBaseByName(id) == nil {
			t.Fatalf("REFUSING: %q is not an item base in the loaded data — wrong fixture", id)
		}
	}

	for _, c := range []struct{ name, want string }{
		{"Awl Sapphire Ring", "Sapphire Ring"},
		{"Awl Ring", "Ring"},
		{"Chaotic Sapphire Ring of the Underground", "Sapphire Ring"},
	} {
		got, s := findItemBaseInName(c.name)
		if got != c.want {
			t.Errorf("findItemBaseInName(%q) = %q, PoB's longest-match picks %q", c.name, got, c.want)
			continue
		}
		if s < 0 || !strings.HasPrefix(c.name[s:], got) {
			t.Errorf("findItemBaseInName(%q) returned offset %d, which does not start %q", c.name, s, got)
		}
	}
}
