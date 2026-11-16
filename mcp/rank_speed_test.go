package mcp

import (
	"testing"
)

// TestRankFlagsGainsThatLowerSpeed: Melee Physical Damage trades attack speed for damage; a row whose DPS rises while
// the skill attacks more slowly says so, and one that raises speed does not.
func TestRankFlagsGainsThatLowerSpeed(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Marauder", "level": 70})["handle"].(string)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Siege Axe", "slot": "Weapon 1"})})
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Heavy Strike"}, "slot": "Weapon 1"},
		map[string]any{"op": "set_main", "group": 1})})
	list := rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "supports", "limit": 25}))
	seen := map[string]bool{}
	for _, r := range list {
		name := r["name"].(string)
		switch name {
		case "Melee Physical Damage", "Faster Attacks":
			seen[name] = true
			if slower := flagged(r, "slower: ", "Speed"); slower != (name == "Melee Physical Damage") {
				t.Errorf("%s: slower flag %v: %v", name, slower, r)
			}
		}
	}
	if len(seen) != 2 {
		t.Fatalf("rows %v lack Melee Physical Damage or Faster Attacks", list)
	}
}
