package mcp

import (
	"math"
	"slices"
	"strings"
	"testing"
)

// fireballChestBuild is a Templar casting Fireball and a linked Elemental Focus from a blue-socketed robe, so both gems
// take the matching-colour quality, wearing a Hubris Circlet.
func fireballChestBuild(t *testing.T, s *Server) string {
	t.Helper()
	h := call(t, s, "new_build", map[string]any{"class": "Templar", "level": 90})["handle"].(string)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "craft", "base": "Vaal Regalia", "sockets": "B-B-B-B", "slot": "Body Armour"},
		map[string]any{"op": "craft", "base": "Hubris Circlet", "slot": "Helmet"})})
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Fireball", "Elemental Focus"}, "slot": "Body Armour"},
		map[string]any{"op": "set_main", "group": 1})})
	return h
}

// TestRankUniquesGainMatchesTheEquippedChange: a uniques row's gain is what equipping it gives, the same as its dDPS: a
// flask counts active, a chest whose default socket colours no longer match the gems loses their matching-colour quality, and a helmet that
// grants a skill (Eber's Unification's Void Gaze) counts it; PoB's item comparison leaves both out (SkillsTab.lua:
// 1257-1264 reads the equipped item, CalcSetup.lua:1452 builds granted skills in a full calculation only).
func TestRankUniquesGainMatchesTheEquippedChange(t *testing.T) {
	s := testServer(t)
	h := fireballChestBuild(t, s)
	for _, slot := range []string{"Body Armour", "Helmet", "Flask 1"} {
		list := rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques", "slot": slot, "limit": 25}))
		if len(list) < 10 {
			t.Fatalf("%s: %d rows", slot, len(list))
		}
		if slot == "Body Armour" && !slices.ContainsFunc(list, func(r map[string]any) bool { return flagged(r, "socket colours: ", "Body Armour") }) {
			t.Errorf("no body armour row says its gain with the current colours: %v", list)
		}
		for _, r := range list {
			d, _ := r["d"].(map[string]any)
			dps, _ := d["dDPS"].(float64)
			if g := r["gain"].(float64); math.Abs(g-dps) > math.Max(1, math.Abs(dps)*1e-3) {
				t.Errorf("%s: gain %v, dDPS %v: %v", r["name"], g, dps, r)
			}
		}
	}
	list := rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques", "slot": "Helmet", "limit": 25, "max_level_req": 69}))
	want := dryDelta(t, s, "edit_items", h, mainDPS, map[string]any{"op": "equip_unique", "name": "Eber's Unification", "slot": "Helmet"})
	i := slices.IndexFunc(list, func(r map[string]any) bool { return strings.HasPrefix(r["name"].(string), "Eber's Unification") })
	if i < 0 {
		t.Fatalf("Eber's Unification not ranked: %v", list)
	}
	if g := list[i]["gain"].(float64); math.Abs(g-want) > math.Max(1, math.Abs(want)*1e-3) {
		t.Fatalf("Eber's Unification: gain %v, equipping it %v", g, want)
	}
}

// TestScanUniquesGainMatchesTheEquippedChange: scan_build's uniques rows score a chest whose socket colours change the
// gems' matching-colour quality by equipping it, as rank_candidates does, so gain and dDPS agree.
func TestScanUniquesGainMatchesTheEquippedChange(t *testing.T) {
	s := testServer(t)
	h := fireballChestBuild(t, s)
	list := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"uniques"}, "slots": []string{"Body Armour"}, "per_slot": 5}))
	if len(list) < 5 {
		t.Fatalf("%d rows", len(list))
	}
	for _, r := range list {
		d, _ := r["d"].(map[string]any)
		dps, ok := d["dDPS"].(float64)
		if g := r["gain"].(float64); !ok || math.Abs(g-dps) > math.Max(1, math.Abs(dps)*1e-3) {
			t.Errorf("%s: gain %v, dDPS %v", r["name"], g, dps)
		}
	}
}
