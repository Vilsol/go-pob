package raw

import (
	"context"
	"errors"
	"testing"

	"github.com/Vilsol/go-pob-data/loader"
)

func TestItemArtPaths(t *testing.T) {
	if err := InitializeAll(LatestVersion, nil); err != nil {
		t.Fatalf("REFUSING: game data %s did not initialise: %v", LatestVersion, err)
	}

	for _, c := range []struct {
		name, base string
		unique     bool
		want       string
	}{
		{"Headhunter", "Leather Belt", true, "Art/2DItems/Belts/Headhunter"},
		{"Tabula Rasa", "Simple Robe", true, "Art/2DItems/Armours/BodyArmours/TabulaRasa"},
		{"Doom Grip", "Leather Belt", false, "Art/2DItems/Belts/Belt3"},
		// a rare named like a unique keeps its base art
		{"Headhunter", "Leather Belt", false, "Art/2DItems/Belts/Belt3"},
		// a unique the stash layout does not list falls back to its base
		{"Not A Unique", "Vaal Regalia", true, "Art/2DItems/Armours/BodyArmours/BodyStrDexInt1C"},
		// a gem and a weapon share the name; items take the weapon
		{"Energy Blade", "Energy Blade", false, ""},
		{"x", "Crimson Jewel", false, "Art/2DItems/Jewels/basicstr"},
		{"x", "No Such Base", false, ""},
	} {
		got := ItemArtPath(c.name, c.base, c.unique)
		if c.base == "Energy Blade" {
			if got == "" || got == "Art/2DItems/Gems/StormBladeGem" {
				t.Errorf("ItemArtPath(%q, %q) = %q, want a weapon's art", c.name, c.base, got)
			}
			continue
		}
		if got != c.want {
			t.Errorf("ItemArtPath(%q, %q, %v) = %q, want %q", c.name, c.base, c.unique, got, c.want)
		}
	}

	for name, want := range map[string]string{
		"Fireball":          "Art/2DItems/Gems/Fireball",
		"fireball":          "Art/2DItems/Gems/Fireball",
		"Vaal Fireball":     "Art/2DItems/Gems/VaalGems/VaalFireball",
		"Added Fire Damage": "Art/2DItems/Gems/Support/AddedFireDamage",
		"Energy Blade":      "Art/2DItems/Gems/StormBladeGem",
		"Nope":              "",
		// supports are shown with "Support" after their name
		"Added Fire Damage Support":   "Art/2DItems/Gems/Support/AddedFireDamage",
		"Life Leech Support":          "Art/2DItems/Gems/Support/LifeLeech",
		"Awakened Spell Echo Support": GemArtPath("Awakened Spell Echo"),
		"Nope Support":                "",
	} {
		if got := GemArtPath(name); got != want {
			t.Errorf("GemArtPath(%q) = %q, want %q", name, got, want)
		}
	}
	// transfigured gems carry their base gem's item
	if got, want := GemArtPath("Cleave of Rage"), GemArtPath("Cleave"); got == "" || got != want {
		t.Errorf("GemArtPath(Cleave of Rage) = %q, want Cleave's %q", got, want)
	}
}

func TestItemArtPathsWhenTablesFailToLoad(t *testing.T) {
	if err := InitializeAll(LatestVersion, nil); err != nil {
		t.Fatalf("REFUSING: game data %s did not initialise: %v", LatestVersion, err)
	}
	tables, loaded := itemArtTables, itemArtLoaded
	t.Cleanup(func() { itemArtTables, itemArtLoaded = tables, loaded })
	calls := 0
	itemArtTables = []table{{"ItemVisualIdentity", func(context.Context, string, loader.AssetCache) error {
		calls++
		return errors.New("offline")
	}}}
	itemArtLoaded = false

	if got := ItemArtPath("Headhunter", "Leather Belt", true); got != "" {
		t.Errorf("ItemArtPath with no art tables = %q, want empty", got)
	}
	if got := GemArtPath("Fireball"); got != "" {
		t.Errorf("GemArtPath with no art tables = %q, want empty", got)
	}
	if calls != 2 {
		t.Errorf("a failed load ran %d times over two lookups, want 2 (each lookup retries)", calls)
	}
}
