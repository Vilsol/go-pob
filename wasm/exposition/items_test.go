package exposition

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

func corpusBuild(t *testing.T, file string) *pob.PathOfBuilding {
	t.Helper()
	xml, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus-3.29", file))
	if err != nil {
		t.Fatalf("REFUSING: %v (fetch the corpus with `mise run testdata`)", err)
	}
	build, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	return build
}

const pastedHelmet = `Item Class: Helmets
Rarity: Rare
Doom Visor
Hubris Circlet
--------
Quality: +20% (augmented)
Energy Shield: 250 (augmented)
--------
Requirements:
Level: 69
Int: 154
--------
Sockets: B-B-B-G
--------
Item Level: 84
--------
+2 to Level of Socketed Minion Gems
+80 to maximum Life
+40% to Fire Resistance
+35% to Cold Resistance (crafted)
`

func TestParseItemPaste(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	build := corpusBuild(t, "0280-elementalist-fireball.xml")

	p, err := ParseItem(build, pastedHelmet, calculator.ItemTooltipStats{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Doom Visor, Hubris Circlet" || p.Rarity != "RARE" || p.Type != "Helmet" || p.Quality != 20 || p.ItemLevel != 84 {
		t.Errorf("header: %q %q %q q%d ilvl%d", p.Name, p.Rarity, p.Type, p.Quality, p.ItemLevel)
	}
	if p.Sockets != "B-B-B-G" {
		t.Errorf("sockets %q", p.Sockets)
	}
	want := []ItemModLine{
		{"+2 to Level of Socketed Minion Gems", "MAGIC"}, {"+80 to maximum Life", "MAGIC"},
		{"+40% to Fire Resistance", "MAGIC"}, {"+35% to Cold Resistance", "CRAFTED"},
	}
	if !slices.Equal(p.Explicits, want) {
		t.Errorf("explicits %v, want %v", p.Explicits, want)
	}
	if !slices.Equal(p.FitSlots, []string{"Helmet"}) {
		t.Errorf("fits %v", p.FitSlots)
	}

	if _, err := ParseItem(build, "Rarity: Rare\nNot A Thing\nNor A Base\n", calculator.ItemTooltipStats{}); err == nil {
		t.Error("an item with no base parsed")
	}
	if _, err := ParseItem(build, "  ", calculator.ItemTooltipStats{}); err == nil {
		t.Error("an empty paste parsed")
	}
}

// TestCompareThenEquip: Equip lands the pasted text so that a recompute reads the same item the compare used, and the compare leaves the build as it was.
func TestCompareThenEquip(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	build := corpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	items := len(build.Items.Items)

	cmp, err := misc.CompareItem(pastedHelmet)
	if err != nil {
		t.Fatal(err)
	}
	if len(cmp) != 1 || cmp[0].SlotName != "Helmet" || cmp[0].Replacing == nil || cmp[0].Replacing.Name != "Honourhome, Soldier Helmet" {
		t.Fatalf("compare: %+v", cmp)
	}
	if len(build.Items.Items) != items {
		t.Fatal("the compare added an item")
	}
	if _, err := EquipItem(build, pastedHelmet, "Body Armour"); err == nil {
		t.Error("a helmet equipped in Body Armour")
	}
	if _, err := EquipItem(build, pastedHelmet, "Helmet"); err != nil {
		t.Fatal(err)
	}
	after := GetMiscCalculator(build).Base()
	for _, k := range []string{"Life", "EnergyShield", "FireResist", "ColdResist"} {
		if after.Output[k] != cmp[0].Output.Output[k] {
			t.Errorf("%s: compare %v, after equipping %v", k, cmp[0].Output.Output[k], after.Output[k])
		}
	}
	var helmet *EquippedSlot
	for _, e := range EquippedItems(build) {
		if e.SlotName == "Helmet" {
			helmet = &e
		}
	}
	if helmet == nil || helmet.Name != "Doom Visor, Hubris Circlet" {
		t.Errorf("equipped helmet: %+v", helmet)
	}
}

// TestParseItemOfABuildItemMatchesAFreshParse: a build's item is read from the shared cache.
func TestParseItemOfABuildItemMatchesAFreshParse(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	build := corpusBuild(t, "0001-necromancer-unknown.xml")
	if len(build.Items.Items) < 100 {
		t.Fatalf("REFUSING: %d items", len(build.Items.Items))
	}
	for round := range 2 {
		for _, it := range build.Items.Items {
			got, gotErr := ParseItem(build, it.Text, calculator.ItemTooltipStats{})
			fresh, freshErr := parseItem(it.Text)
			if (gotErr == nil) != (freshErr == nil) {
				t.Fatalf("round %d item %d: error %v, fresh %v", round, it.ID, gotErr, freshErr)
			}
			if freshErr != nil {
				continue
			}
			if want := parsedItem(build, fresh, calculator.ItemTooltipStats{}); !reflect.DeepEqual(got, want) {
				t.Fatalf("round %d item %d: cached %+v\nfresh %+v", round, it.ID, got, want)
			}
		}
	}
	for _, e := range EquippedItems(build) {
		fresh := calculator.ParseItem(e.Text)
		if e.Name != fresh.Name || e.Rarity != string(fresh.Rarity) {
			t.Errorf("equipped %s: %q %q, fresh %q %q", e.SlotName, e.Name, e.Rarity, fresh.Name, fresh.Rarity)
		}
	}
}

// TestSlotItemListsMatchesEachSlotsList: the lists read every item once for all slots, and must equal each slot's own list.
func TestSlotItemListsMatchesEachSlotsList(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	filled := 0
	for _, file := range []string{"0001-necromancer-unknown.xml", "0280-elementalist-fireball.xml", "0667-guardian-summon-raging-spirit.xml"} {
		build := corpusBuild(t, file)
		lists := SlotItemLists(build)
		if len(lists) != len(calculator.OrderedItemSlotNames()) {
			t.Fatalf("%s: %d lists", file, len(lists))
		}
		for _, l := range lists {
			if want := calculator.SlotItemIDs(build, l.SlotName); !slices.Equal(l.ItemIDs, want) {
				t.Errorf("%s %s: %v, want %v", file, l.SlotName, l.ItemIDs, want)
			}
			filled += len(l.ItemIDs)
		}
	}
	if filled < 100 {
		t.Fatalf("REFUSING: only %d listed items", filled)
	}
}
