package calculator

import (
	"context"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestFoilUniqueIsNotAModLine: Item.lua:434-443 and :533-535.
func TestFoilUniqueIsNotAModLine(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(context.Background()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	const foiled = `Rarity: UNIQUE
Ashes of the Stars
Onyx Amulet
Quality: 20
LevelReq: 68
Implicits: 1
+20 to all Attributes
+2 to Level of all Skill Gems
Corrupted
Foil Unique (Cobalt)`

	control := strings.ReplaceAll(foiled, "\nFoil Unique (Cobalt)", "")

	got := ParseItem(foiled)
	ctl := ParseItem(control)

	if got.Rarity != RarityRelic {
		t.Errorf("rarity: got %q, want %q", got.Rarity, RarityRelic)
	}
	if ctl.Rarity != RarityUnique {
		t.Errorf("control rarity: got %q, want %q", ctl.Rarity, RarityUnique)
	}
	if got.FoilType != "Cobalt" {
		t.Errorf("foilType: got %q, want %q", got.FoilType, "Cobalt")
	}
	if ctl.FoilType != "" {
		t.Errorf("control foilType: got %q, want empty", ctl.FoilType)
	}
	for _, ml := range append(append([]ModLine{}, got.ImplicitModLines...), got.ExplicitModLines...) {
		if strings.Contains(ml.Line, "Foil Unique") {
			t.Errorf("the foil line is still a mod line: %q", ml.Line)
		}
	}
	if len(got.ExplicitModLines) != len(ctl.ExplicitModLines) {
		t.Errorf("explicit line count: foiled %d, control %d -- the foil line must consume no index",
			len(got.ExplicitModLines), len(ctl.ExplicitModLines))
	}
	if len(ctl.ExplicitModLines) == 0 {
		t.Fatal("REFUSING: the control parsed no explicit lines, so the count comparison is vacuous")
	}

	bare := ParseItem(strings.ReplaceAll(foiled, "Foil Unique (Cobalt)", "Foil Unique"))
	if bare.FoilType != "Rainbow" {
		t.Errorf("bare foil line: got foilType %q, want %q", bare.FoilType, "Rainbow")
	}
	if bare.Rarity != RarityRelic {
		t.Errorf("bare foil line rarity: got %q, want %q", bare.Rarity, RarityRelic)
	}
}
