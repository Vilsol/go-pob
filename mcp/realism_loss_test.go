package mcp

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

const staunchingFlask = `Rarity: MAGIC
Quicksilver Flask of Staunching
Quicksilver Flask
Grants Immunity to Bleeding for 11 seconds if used while Bleeding
Grants Immunity to Corrupted Blood for 11 seconds if used while affected by Corrupted Blood`

func warningsAdded(res map[string]any) []string {
	list := res["warnings_added"].([]any)
	out := make([]string, 0, len(list))
	for _, w := range list {
		out = append(out, w.(string))
	}
	return out
}

// TestEditWarnsLostImmunities: removing the build's only bleed and Corrupted Blood immunity, a flask, warns in the edit
// result.
func TestEditWarnsLostImmunities(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "equip_text", "text": staunchingFlask, "slot": "Flask 1"},
		map[string]any{"op": "set_flask_active", "slot": "Flask 1", "active": true})})
	res := call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "unequip", "slot": "Flask 1"})})
	w := warningsAdded(res)
	for _, want := range []string{"loses: BleedAvoidChance 100 -> 0", "loses: CorruptedBloodImmunity"} {
		if !slices.Contains(w, want) {
			t.Errorf("warnings_added %v, want %q", w, want)
		}
	}
}

// TestEditWarnsLostSuppression: dropping spell suppression from its 100% cap warns in the edit result and flags a
// ranked row.
func TestEditWarnsLostSuppression(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "customMods", "value": "+100% chance to Suppress Spell Damage"})})
	hh, _ := s.get(h)
	b, _ := hh.parseRev(hh.cur)
	before, _ := hh.envAt(hh.cur)
	res := call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "customMods", "value": "+78% chance to Suppress Spell Damage"})})
	if w := warningsAdded(res); !slices.Contains(w, "loses: SpellSuppressionChance 100 -> 78") {
		t.Fatalf("warnings_added %v", w)
	}
	c, _ := hh.parseRev(hh.cur)
	after, _ := hh.envAt(hh.cur)
	if f := fmt.Sprint(realismFlags(b, c, before, after)); !strings.Contains(f, "loses: SpellSuppressionChance 100 -> 78") {
		t.Fatalf("flags %v", f)
	}
}
