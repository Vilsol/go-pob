package calculator

import (
	"context"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestUnparsedGameModeLineCarriesTheDefaultRange: Item.lua:1321 -- the GAME-mode arm that keeps a line ModParser could not read stamps `modLine.range = main.defaultItemAffixQuality`.
func TestUnparsedGameModeLineCarriesTheDefaultRange(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(context.Background()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	const raw = `Rarity: RARE
Loath Cry
Onyx Amulet
Item Level: 84
LevelReq: 68
Implicits: 0
+25 to all Attributes
Zzzz Not A Modifier Zzzz
+30% to Fire Resistance`

	i := ParseItem(raw)
	if len(i.ExplicitModLines) != 3 {
		t.Fatalf("REFUSING: got %d explicit lines, want 3 -- the arm under test was not reached", len(i.ExplicitModLines))
	}
	if i.ExplicitModLines[1].Extra == "" {
		t.Fatal("REFUSING: the middle line PARSED, so it never reaches the unparsed arm")
	}
	if len(i.ExplicitModLines[1].ModList) != 0 {
		t.Fatal("REFUSING: the middle line emitted mods, so it never reaches the unparsed arm")
	}

	for n, ml := range i.ExplicitModLines {
		if ml.Range == nil {
			t.Errorf("explicit[%d] %q: range is nil, want %v", n, ml.Line, *defaultItemAffixQuality)
			continue
		}
		if *ml.Range != *defaultItemAffixQuality {
			t.Errorf("explicit[%d] %q: range %v, want %v", n, ml.Line, *ml.Range, *defaultItemAffixQuality)
		}
	}
}

// TestBaseTypeLineIsNotAModLine: Item.lua:1318's LAST conjunct: a line equal to the base's `type`, or to `subType ..
func TestBaseTypeLineIsNotAModLine(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(context.Background()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	const raw = `Rarity: RARE
Loath Cry
Onyx Amulet
Amulet
Item Level: 84
LevelReq: 68
Implicits: 0
+25 to all Attributes`

	i := ParseItem(raw)
	if i.Base == nil || i.Base.Type != "Amulet" {
		t.Fatalf("REFUSING: base did not resolve to an Amulet (%v)", i.Base)
	}
	for n, ml := range i.ExplicitModLines {
		if ml.Line == "Amulet" {
			t.Errorf("explicit[%d] is the base TYPE line %q", n, ml.Line)
		}
	}
	if len(i.ExplicitModLines) != 1 {
		t.Errorf("explicit lines: got %d, want 1 (the attribute mod alone)", len(i.ExplicitModLines))
	}
}
