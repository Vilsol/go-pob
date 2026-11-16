package mcp

import (
	"fmt"
	"strings"
	"testing"
)

// TestOptimizePicksTheRankedMasteryEffect: two effects of this mastery give the same MainDPS; optimize takes the one
// rank_candidates names, not whichever calculation came out a rounding error ahead.
func TestOptimizePicksTheRankedMasteryEffect(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "import_build", map[string]any{"source": fixture("1.xml")})["handle"].(string)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "MainDPS", "node_types": []string{"mastery"}, "limit": 25})
	var ranked map[string]any
	for _, r := range rows(res) {
		if r["id"] == float64(60834) {
			ranked = r
		}
	}
	if ranked == nil {
		t.Fatalf("no row for the mastery: %v", rows(res))
	}
	o := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"},
		"slots": []any{map[string]any{"kind": "nodes", "candidates": []string{"60834"}, "k": 1}}})
	add := fmt.Sprint(rows(o)[0]["choices"].([]any)[0].(map[string]any)["choice"].(map[string]any)["add"])
	_, effect, _ := strings.Cut(ranked["name"].(string), ": ")
	if !strings.Contains(add, effect) {
		t.Fatalf("optimize picked %v, rank_candidates %q", add, effect)
	}
}

// TestOptimizePrefersFewerChangesOnTies: removing any of these nodes leaves MainDPS as it is, so every subset ties;
// results list them by how few changes they make.
func TestOptimizePrefersFewerChangesOnTies(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"}, "top": 5,
		"slots": []any{map[string]any{"kind": "nodes", "remove": []string{"8001", "8629", "1340"}}}})
	prev := -1
	for _, r := range rows(res) {
		if r["score"] != rows(res)[0]["score"] {
			t.Fatalf("not a tie: %v", rows(res))
		}
		n := len(r["choices"].([]any)[0].(map[string]any)["choice"].(map[string]any)["remove"].([]any))
		if n < prev {
			t.Fatalf("a %d-change result ranks after a %d-change one: %v", n, prev, rows(res))
		}
		prev = n
	}
}

// TestOptimizeFillsTopWithNearDuplicates: Hypothermia's quality does not move this build's watched stats, so every
// quality is a near duplicate of the current one; with five distinct states, top 5 still returns five rows, the extra
// ones naming the row they nearly duplicate.
func TestOptimizeFillsTopWithNearDuplicates(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse", "Hypothermia"}, "level": 20, "quality": 0}, map[string]any{"op": "set_main", "group": 1})})
	res := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"}, "top": 5,
		"slots": []any{map[string]any{"kind": "gem", "group": 1, "gem": "Hypothermia", "levels": []int{20}, "qualities": []int{0, 5, 10, 15, 20}}}})
	got := rows(res)
	if len(got) != 5 {
		t.Fatalf("top 5 returned %d rows: %v", len(got), got)
	}
	if got[4]["near_duplicate_of"] == nil {
		t.Errorf("row 5 does not name the row it nearly duplicates: %v", got[4])
	}
}
