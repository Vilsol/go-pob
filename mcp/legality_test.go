package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

// TestItemLegality: an equipped ring with four prefixes, two bench crafts and an implicit above its base's range is
// named with the rule each breaks; body armour whose sockets are all off-colour for its gems is legal.
func TestItemLegality(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	ring := "Rarity: RARE\nDoom Loop\nPaua Ring\nItem Level: 84\nImplicits: 1\n+60 to maximum Mana\n+70 to maximum Life\n" +
		"+40 to maximum Energy Shield\nAdds 5 to 9 Physical Damage to Attacks\n+25 to maximum Mana\n" +
		"{crafted}+30% to Cold Resistance\n{crafted}+30% to Lightning Resistance"
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_text", "slot": "Ring 1", "text": ring},
		map[string]any{"op": "craft", "base": "Astral Plate", "slot": "Body Armour", "sockets": "B-B-B-B-B-B"})})
	res := call(t, s, "get_build", map[string]any{"handle": h, "section": "warnings"})
	flags := asList(res["audit"].(map[string]any)["illegal_items"])
	got := map[string]string{}
	for _, f := range flags {
		m := f.(map[string]any)
		got[m["slot"].(string)+" "+m["rule"].(string)] = m["note"].(string)
	}
	for _, want := range []string{"Ring 1 prefixes", "Ring 1 crafted", "Ring 1 implicit"} {
		if _, ok := got[want]; !ok {
			t.Fatalf("no %q flag: %v", want, flags)
		}
	}
	if len(got) != 3 {
		t.Fatalf("flags beyond the three broken rules: %v", flags)
	}
	if n := got["Ring 1 prefixes"]; !strings.Contains(n, "4 prefixes") {
		t.Fatalf("prefix note: %s", n)
	}
}

// legalityOf runs the item audit over a build whose item sets equip the given items, slot by slot, and whose one tree
// sockets the "jewel" items; the first set is active.
func legalityOf(t *testing.T, sets ...map[string]string) []map[string]any {
	t.Helper()
	testServer(t)
	b := &pob.PathOfBuilding{}
	b.Items.ActiveItemSet = 1
	id := 0
	for i, slots := range sets {
		set := pob.ItemSet{ID: strconv.Itoa(i + 1)}
		title := slots["title"]
		set.Title = &title
		for _, slot := range calculator.OrderedItemSlotNames() {
			if text, ok := slots[slot]; ok {
				id++
				b.Items.Items = append(b.Items.Items, pob.Item{ID: id, Text: text})
				set.Slots = append(set.Slots, pob.Slot{ItemID: id, Name: slot})
			}
		}
		b.Items.ItemSets = append(b.Items.ItemSets, set)
		if text, ok := slots["jewel"]; ok {
			id++
			b.Items.Items = append(b.Items.Items, pob.Item{ID: id, Text: text})
			if len(b.Tree.Specs) == 0 {
				b.Tree.Specs = []pob.Spec{{}}
			}
			b.Tree.Specs[0].Sockets = append(b.Tree.Specs[0].Sockets, pob.Socket{NodeID: 26196, ItemID: id})
		}
	}
	raw, err := json.Marshal(itemLegalityFlags(b))
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestItemLegalityEverySet: two bench crafts on a body armour are found in an item set that is not active, and the
// flag names that set.
func TestItemLegalityEverySet(t *testing.T) {
	body := "Rarity: RARE\nDoom Coat\nAstral Plate\n+90 to maximum Life\n{crafted}+40% to Cold Resistance\n{crafted}+50 to maximum Mana"
	flags := legalityOf(t,
		map[string]string{"title": "Leveling", "Ring 1": "Rarity: RARE\nDoom Loop\nPaua Ring\n+50 to maximum Life"},
		map[string]string{"title": "Endgame", "Body Armour": body})
	if len(flags) != 1 || flags[0]["rule"] != "crafted" || fmt.Sprint(flags[0]["sets"]) != `[item set "Endgame"]` {
		t.Fatalf("want one crafted flag naming the Endgame set: %v", flags)
	}
}

// TestItemLegalityCustomLines: gloves with five typed explicit mods and two {custom} lines no mod words hold seven
// modifiers; without the custom lines they are legal.
func TestItemLegalityCustomLines(t *testing.T) {
	gloves := "Rarity: RARE\nDoom Grip\nPhantom Mitts\n+12% chance to Suppress Spell Damage\n+110 to maximum Life\n+60 to maximum Mana\n" +
		"+20% to Chaos Resistance\n{crafted}(20-25)% of Physical Damage Converted to Lightning Damage"
	custom := "\n{custom}55% increased Critical Strike Chance against Shocked Enemies\n{custom}+48% lightning resistance"
	if flags := legalityOf(t, map[string]string{"Gloves": gloves}); len(flags) != 0 {
		t.Fatalf("legal gloves flagged: %v", flags)
	}
	flags := legalityOf(t, map[string]string{"Gloves": gloves + custom})
	if len(flags) != 1 || flags[0]["rule"] != "modifiers" || !strings.Contains(flags[0]["note"].(string), "at least 7 explicit modifiers") {
		t.Fatalf("want a seven-modifier flag: %v", flags)
	}
}

// TestItemLegalityExplicitRange: a jewel's +15% global critical strike multiplier is above the (9-12)% its only mod
// with that wording rolls; +12% is legal.
func TestItemLegalityExplicitRange(t *testing.T) {
	jewel := "Rarity: MAGIC\nCobalt Jewel of Potency\nCobalt Jewel\n+%d%% to Global Critical Strike Multiplier"
	if flags := legalityOf(t, map[string]string{"jewel": fmt.Sprintf(jewel, 12)}); len(flags) != 0 {
		t.Fatalf("legal jewel flagged: %v", flags)
	}
	flags := legalityOf(t, map[string]string{"jewel": fmt.Sprintf(jewel, 15)})
	if len(flags) != 1 || flags[0]["rule"] != "value" || fmt.Sprint(flags[0]["sets"]) != `[tree "Default" (active)]` {
		t.Fatalf("want a value flag: %v", flags)
	}
}

// TestItemLegalityBenchHybrid: movement speed and Onslaught on kill crafted on boots are one bench hybrid, so legal.
func TestItemLegalityBenchHybrid(t *testing.T) {
	boots := "Rarity: RARE\nVortex Stride\nLeviathan Greaves\n+90 to maximum Life\n{crafted}19% increased Movement Speed\n" +
		"{crafted}12% chance to gain Onslaught for 4 seconds on Kill"
	if flags := legalityOf(t, map[string]string{"Boots": boots}); len(flags) != 0 {
		t.Fatalf("bench hybrid flagged: %v", flags)
	}
}

// TestItemLegalityBenchHybridAnyRollOrOrder: a ring's bench mana hybrid is one craft at an older craft's roll, in either
// line order and with another line between; two separate crafts still flag.
func TestItemLegalityBenchHybridAnyRollOrOrder(t *testing.T) {
	ring := "Rarity: RARE\nMana Loop\nAmethyst Ring\nItem Level: 84\nImplicits: 1\n+19% to Chaos Resistance\n"
	for name, lines := range map[string]string{
		"an older roll":      "+70 to maximum Life\n{crafted}+45 to maximum Mana\n{crafted}Regenerate 5 Mana per second",
		"the other order":    "+70 to maximum Life\n{crafted}Regenerate 3 Mana per second\n{crafted}+35 to maximum Mana",
		"a line between":     "{crafted}+35 to maximum Mana\n+70 to maximum Life\n{crafted}Regenerate 3 Mana per second",
		"the current hybrid": "+70 to maximum Life\n{crafted}+35 to maximum Mana\n{crafted}Regenerate 3 Mana per second",
	} {
		if flags := legalityOf(t, map[string]string{"Ring 1": ring + lines}); len(flags) != 0 {
			t.Errorf("%s: a bench hybrid flagged: %v", name, flags)
		}
	}
	flags := legalityOf(t, map[string]string{"Ring 1": ring + "+70 to maximum Life\n{crafted}+50 to maximum Mana\n{crafted}+20% to Fire Resistance"})
	if len(flags) != 1 || flags[0]["rule"] != "crafted" {
		t.Errorf("two separate crafts: %v", flags)
	}
}

// TestItemLegalityFixturesClean: no item of any set in the shared build fixtures is flagged.
func TestItemLegalityFixturesClean(t *testing.T) {
	testServer(t)
	files, _ := filepath.Glob(fixture("*.xml"))
	if len(files) == 0 {
		t.Fatal("no fixtures")
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		b, err := builds.ParseBuild(raw)
		if err != nil {
			t.Fatal(err)
		}
		if flags := itemLegalityFlags(b); len(flags) != 0 {
			t.Errorf("%s: %+v", filepath.Base(f), flags)
		}
	}
}
