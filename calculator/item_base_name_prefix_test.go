package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestBaseLinePrefixesAreStripped pins Item.lua:973 --.
func TestBaseLinePrefixesAreStripped(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"Scholar's Robe", "Quartz Flask", "Strapped Mitts", "Sapphire Ring"} {
		if ItemBaseByName(id) == nil {
			t.Fatalf("REFUSING: %q is not an item base in the loaded data — wrong fixture", id)
		}
	}

	for _, c := range []struct{ name, line, want string }{
		{"vestigial", "Vestigial Scholar's Robe", "Scholar's Robe"},
		{"superior", "Superior Quartz Flask", "Quartz Flask"},
		{"chained", "Superior Vestigial Scholar's Robe", "Scholar's Robe"},
		{"interior-survives", "Sapphire Synthesised Ring", ""},
		{"plain", "Strapped Mitts", "Strapped Mitts"},
	} {
		t.Run(c.name, func(t *testing.T) {
			i := &Item{Rarity: RarityUnique, Name: "Fixture", Requirements: map[string]float64{}}
			i.resolveBase(c.line, nil)
			if i.BaseName != c.want {
				t.Errorf("resolveBase(%q) resolved BaseName %q, PoB's three anchored gsubs give %q", c.line, i.BaseName, c.want)
			}
		})
	}
}
