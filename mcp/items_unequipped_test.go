package mcp

import (
	"strings"
	"testing"
)

// TestReplacedItemsAreCounted: an equip keeps the item it replaces in the build's item list, as PoB does, and says so;
// the items section counts the unequipped items without listing them.
func TestReplacedItemsAreCounted(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	before, _ := call(t, s, "get_build", map[string]any{"handle": h, "section": "items"})["unequipped_count"].(float64)
	res := call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Iron Gauntlets", "slot": "Gloves"})})
	if txt := strings.Join(toStrings(res["applied"]), " "); !strings.Contains(txt, "kept in the item list unequipped") {
		t.Fatalf("applied = %v", res["applied"])
	}
	got := call(t, s, "get_build", map[string]any{"handle": h, "section": "items"})
	if n, _ := got["unequipped_count"].(float64); n != before+1 {
		t.Fatalf("unequipped_count %v, before the equip %v", n, before)
	}
	if _, ok := got["unequipped"]; ok {
		t.Fatalf("the summary items section lists the unequipped items: %v", got["unequipped"])
	}
}
