package mcp

import (
	"math"
	"testing"
)

// TestRankOffenceDefenceNamesEHPItIgnores: PoB's Offence/Defence defence score counts life, armour, energy shield,
// evasion and regeneration (CalcsTab.lua:749-754), so removing Mind Over Matter scores 0 while EHP falls; the row says
// how far EHP moves and that the score leaves it out.
func TestRankOffenceDefenceNamesEHPItIgnores(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "34098"})})
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "OffenceDefence", "limit": 25, "watch": []string{"TotalEHP"}})
	r := removeRow(t, res, 34098)
	base := res["base_stats"].(map[string]any)["TotalEHP"].(float64)
	want := (r["after"].(map[string]any)["TotalEHP"].(float64)/base - 1) * 100
	if want > -5 || num(r, "defence_pct") != 0 {
		t.Fatalf("Mind Over Matter row: %v (EHP moves %.1f%%)", r, want)
	}
	if got := num(r, "ehp_pct"); math.Abs(got-want) > 0.11 || !flagged(r, "defence: ", "EHP") {
		t.Fatalf("Mind Over Matter row: ehp_pct %v, EHP moves %.1f%%: %v", got, want, r)
	}
}

// TestRankOffenceDefenceScoresUniquesAndSupports: uniques (flasks among them) and supports rows score Offence/Defence
// as nodes do, the DPS fraction plus PoB's defence score of applying the row (a support's add, its swap on a full
// group), not 0.
func TestRankOffenceDefenceScoresUniquesAndSupports(t *testing.T) {
	s := testServer(t)
	h := fireballChestBuild(t, s)
	for _, args := range []map[string]any{
		{"kind": "uniques", "slot": "Flask 1"},
		{"kind": "uniques", "slot": "Body Armour"},
		{"kind": "supports"},
	} {
		args["handle"], args["stat"], args["limit"] = h, "OffenceDefence", 25
		list := rows(call(t, s, "rank_candidates", args))
		scored := 0
		for _, r := range list {
			g, ok := r["gain"].(float64)
			if !ok {
				g, ok = r["gain_added"].(float64)
			}
			if !ok {
				g, _ = r["gain_swapped"].(float64)
			}
			if g != 0 {
				scored++
				if math.Abs(g-num(r, "offence_pct")-num(r, "defence_pct")) > 1 {
					t.Errorf("%v: gain %v is not offence_pct + defence_pct: %v", args["slot"], g, r)
				}
			}
		}
		if scored == 0 {
			t.Errorf("%v %v: every row scores 0: %v", args["kind"], args["slot"], list)
		}
	}
}
