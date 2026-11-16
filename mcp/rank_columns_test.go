package mcp

import (
	"fmt"
	"math"
	"testing"
)

// fullDPSBuild is a Templar whose Freezing Pulse Spell Totem counts in Full DPS while Fireball is the main skill, so
// MainDPS and FullDPS move apart.
func fullDPSBuild(t *testing.T, s *Server) string {
	t.Helper()
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse", "Spell Totem"}, "full_dps": true},
		map[string]any{"op": "add_group", "gems": []string{"Fireball"}},
		map[string]any{"op": "set_main", "group": 2})})
	return h
}

// TestRankChangeColumnsFollowTheRankedDPS: ranking by FullDPS, a row's dDPS is what applying it does to FullDPS, and
// the result names that stat.
func TestRankChangeColumnsFollowTheRankedDPS(t *testing.T) {
	s := testServer(t)
	h := fullDPSBuild(t, s)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "FullDPS", "limit": 3})
	list := rows(res)
	if len(list) == 0 {
		t.Fatal("no rows")
	}
	for _, r := range list {
		want := dryDelta(t, s, "edit_tree", h, "FullDPS", map[string]any{"op": "allocate", "node": fmt.Sprint(int64(r["id"].(float64)))})
		d, _ := r["d"].(map[string]any)
		got, _ := d["dDPS"].(float64)
		if want == 0 || math.Abs(got-want) > math.Max(1, 0.001*math.Abs(want)) {
			t.Fatalf("%v: dDPS %v, adding it moves FullDPS by %v", r["name"], got, want)
		}
	}
	if res["dDPS_stat"] != "FullDPS" {
		t.Fatalf("dDPS_stat %v", res["dDPS_stat"])
	}
}
