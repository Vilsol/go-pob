package mcp

import (
	"strings"
	"testing"
)

// Item.lua:546-555 reads "Corrupted" as the item's corrupted flag, also when a variant prefix makes it a mod line;
// ModMaster.lua:732-734's crafting-bench lines change what the item can roll and carry no stat.
func TestUnparsedWarningsSkipItemLinesThatAreNotMods(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch"})["handle"].(string)
	ring := "Rarity: UNIQUE\nTest Band\nGold Ring\nVariant: Old\nVariant: Current\nSelected Variant: 2\nImplicits: 1\n" +
		"+12% to all Elemental Resistances\n{variant:2}Corrupted\n"
	wand := "Rarity: RARE\nDoom Song\nImbued Wand\nImplicits: 1\n33% increased Spell Damage\n+20 to maximum Life\n" +
		"{crafted}Can have up to 3 Crafted Modifiers\n{crafted}Prefixes Cannot Be Changed\nGlorps the frobnicator twice\n"
	res := call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "equip_text", "slot": "Ring 1", "text": ring},
		map[string]any{"op": "equip_text", "slot": "Weapon 1", "text": wand})})
	var got []string
	for _, w := range res["warnings_added"].([]any) {
		if s, _ := w.(string); strings.HasPrefix(s, "Unsupported mod") {
			got = append(got, s)
		}
	}
	if len(got) != 1 || !strings.Contains(got[0], "Glorps the frobnicator twice") {
		t.Fatalf("unsupported-mod warnings = %q, want only the unparseable wand line", got)
	}
}
