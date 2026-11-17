package calculator

import (
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

func requireViscera(t *testing.T) {
	t.Helper()
	if raw.LatestVersion != "3.30" {
		t.Skipf("NOT-RUN: Viscera bases exist from 3.30 game data; the process loaded %s (POE_VERSION=3.30)", raw.LatestVersion)
	}
	initItemsTabData(t)
}

var pinnedVisceraBaseRe = regexp.MustCompile(`(?s)itemBases\["([^"]+)"\] = \{\n\tbaseId = "[^"]*",\n\ttype = "Viscera",\n\ttags = \{ ([^}]*)\}`)

// src/Data/Bases/wickerman.lua in the pinned PoB
func TestVisceraBasesMatchThePin(t *testing.T) {
	requireViscera(t)
	pobRoot := os.Getenv("POB_DIR")
	if pobRoot == "" {
		home, _ := os.UserHomeDir()
		pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
	}
	path := filepath.Join(pobRoot, "src", "Data", "Bases", "wickerman.lua")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("NOT-RUN: no pinned PoB Wickerman table at %s (set POB_DIR)", path)
	}
	want := map[string][]string{}
	for _, m := range pinnedVisceraBaseRe.FindAllStringSubmatch(string(b), -1) {
		var tags []string
		for _, tm := range regexp.MustCompile(`(\w+) = true`).FindAllStringSubmatch(m[2], -1) {
			tags = append(tags, tm[1])
		}
		want[m[1]] = tags
	}
	if len(want) < 100 {
		t.Fatalf("REFUSING: parsed %d Viscera bases out of %s", len(want), path)
	}
	for name, tags := range want {
		base := ItemBases()[name]
		if base == nil || base.Type != "Viscera" {
			t.Errorf("%s: ours %+v, pin a Viscera base", name, base)
			continue
		}
		if got := slices.Sorted(maps.Keys(base.Tags)); !slices.Equal(got, tags) {
			t.Errorf("%s tags: ours %v, pin %v", name, got, tags)
		}
	}
	for name, base := range ItemBases() {
		if _, ok := want[name]; base.Type == "Viscera" && !ok {
			t.Errorf("%s: a Viscera base the pin does not carry", name)
		}
	}
}

// spec/System/TestItemParse_spec.lua "pastes Viscera names and implicit rolls without adding item descriptions as modifiers".
// The spec's implicit-roll assertions need the advanced-copy "{ ... }" header block (Item.lua:584-645), which is not ported.
func TestVisceraPasteMatchesPoB(t *testing.T) {
	requireViscera(t)
	cases := []struct{ name, base, prefix, label string }{
		{"Greater Gladiator's Heart", "Gladiator's Heart", "Greater ", "Torso"},
		{"Lesser Gladiator's Heart", "Gladiator's Heart", "Lesser ", "Torso"},
		{"Gladiator's Heart", "Gladiator's Heart", "", "Torso"},
		{"Greater Sirus' Head", "Sirus' Head", "Greater ", "Head"},
		{"Experimental Gladiator's Heart", "Gladiator's Heart", "Experimental ", "Torso"},
		{"Lesser Warden's Arm", "Warden's Arm", "Lesser ", "Upper Limb"},
		{"Lesser Pocked Leg", "Pocked Leg", "Lesser ", "Lower Limb"},
	}
	const instruction = "Take this to the Wickerman in a town or hideout to attach it."
	for _, c := range cases {
		text := "Item Class: Viscera\n" + c.name + "\n--------\n" + c.label + "\n--------\nItem Level: 69\n--------\n{ Implicit Modifier }\n" +
			"+6(5-7) to Maximum Rage\n" +
			"(Every Rage grants 1% more Attack Damage. Maximum Rage is 30. Lose 10 Rage per second if you have not been Hit or gained Rage in the last 2 seconds)\n" +
			"--------\n" + instruction + "\n"
		check := func(label string, item *Item) {
			t.Helper()
			if item.Rarity != RarityNormal || item.Name != c.name || item.BaseName != c.base || item.NamePrefix != c.prefix || item.Type != "Viscera" {
				t.Errorf("%s %s: rarity %s name %q base %q prefix %q type %q", label, c.name, item.Rarity, item.Name, item.BaseName, item.NamePrefix, item.Type)
			}
			if item.ItemLevel == nil || *item.ItemLevel != 69 {
				t.Errorf("%s %s: item level %v, PoB 69", label, c.name, item.ItemLevel)
			}
			for _, ml := range slices.Concat(item.ImplicitModLines, item.ExplicitModLines) {
				if ml.Line == c.label || ml.Line == instruction {
					t.Errorf("%s %s: %q parsed as a modifier", label, c.name, ml.Line)
				}
			}
		}
		item := ParseItem(text)
		check("paste", item)
		check("saved", ParseItem(buildRawText(item)))
		plain := ParseItem("Item Class: Viscera\n" + c.name + "\n--------\n" + c.label + "\n--------\nItem Level: 69\n--------\n" +
			"+6 to Maximum Rage (implicit)\n--------\n" + instruction + "\n")
		check("plain", plain)
	}
}

// src/Classes/Item.lua:2208-2213, src/Classes/ItemsTab.lua:2671-2674
func TestVisceraSlotFromBaseTag(t *testing.T) {
	for _, part := range []string{"Head", "Body", "Arm", "Leg"} {
		item := &Item{Type: "Viscera", Base: &ItemBase{Type: "Viscera", Tags: map[string]bool{"viscera_" + map[string]string{"Head": "head", "Body": "body", "Arm": "arm", "Leg": "leg"}[part]: true}}}
		if got := item.getPrimarySlot(); got != part {
			t.Errorf("primary slot of a %s Viscera: %q", part, got)
		}
		for _, slot := range wickermanSlotNames {
			if got := itemValidForSlot(item, slot, slotTypeOf(slot), func(string) *Item { return nil }); got != (slot == part) {
				t.Errorf("%s Viscera in %s: valid %v", part, slot, got)
			}
		}
		if itemValidForSlot(item, "Body Armour", "Body Armour", func(string) *Item { return nil }) {
			t.Errorf("%s Viscera fits Body Armour", part)
		}
	}
}

// src/Classes/ItemsTab.lua:204-214: the Wickerman slots follow the flasks and show on a 3.30 tree only
func TestWickermanSlotsFollowFlasksOnA330Tree(t *testing.T) {
	names := OrderedItemSlotNames()
	at := slices.Index(names, "Flask 5")
	if at < 0 || !slices.Equal(names[len(names)-4:], []string{"Head", "Body", "Arm", "Leg"}) {
		t.Fatalf("slot order %v", names)
	}
	for _, v := range []struct {
		version pob.TreeVersion
		shown   bool
	}{{"3_29", false}, {"3_30", true}} {
		build := &pob.PathOfBuilding{Tree: pob.Tree{ActiveSpec: 1, Specs: []pob.Spec{{TreeVersion: v.version}}}}
		for _, name := range wickermanSlotNames {
			if got := slotShown(build, itemSlot{name: name}, false); got != v.shown {
				t.Errorf("%s on a %s tree: shown %v", name, v.version, got)
			}
		}
	}
}

// spec/System/TestItemDBControl_spec.lua "appends typed Wickerman filters after Grafts"
func TestItemDBWickermanSlotFilters(t *testing.T) {
	graft := slices.Index(ItemDBSlots, "Graft 2")
	if !slices.Equal(ItemDBSlots[graft+1:], []string{"Head", "Body", "Arm", "Leg"}) {
		t.Fatalf("slot filters after Graft 2: %v", ItemDBSlots[graft+1:])
	}
	for _, kind := range []ItemDBKind{ItemDBRare, ItemDBUnique} {
		for k, slot := range ItemDBSlots[graft+1:] {
			f := ItemDBFilter{Slot: graft + 2 + k}
			if doesItemMatchFilters(kind, &Item{Type: "Graft", Base: &ItemBase{}}, f, ItemDBContext{}) {
				t.Errorf("%s: a Graft matches the %s filter", kind, slot)
			}
			if !doesItemMatchFilters(kind, &Item{Type: slot, Base: &ItemBase{}}, f, ItemDBContext{}) {
				t.Errorf("%s: an item of type %s misses the %s filter", kind, slot, slot)
			}
		}
	}
}
