package mcp

import (
	"fmt"
	"strings"
	"testing"
)

// TestOptimizeCountsMasteryEffectsAsOneChoice: a mastery's effects exclude each other, so a space whose raw count of
// subsets exceeds the budget while its valid selections fit is searched exhaustively.
func TestOptimizeCountsMasteryEffectsAsOneChoice(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "optimize", map[string]any{"handle": h, "budget": 80, "max_ms": 60000, "objective": map[string]any{"stat": "Life"},
		"slots":       []any{map[string]any{"kind": "nodes", "candidates": []string{"29933", "40362", "12382"}, "remove": []string{"8001", "12795"}, "path": false}},
		"constraints": []any{map[string]any{"stat": "points_added", "op": "<=", "value": 0}}})
	space := res["space"].(map[string]any)
	if space["method"] != "exhaustive" || res["proven_optimal"] != true {
		t.Fatalf("space %v proven %v stopped %v evaluations %v", space, res["proven_optimal"], res["stopped"], res["evaluations"])
	}
}

// TestOptimizeSkipsAnExhaustiveSearchTheTimeLimitWouldCut: a space within the budget but not within max_ms is searched
// locally, and the result says why.
func TestOptimizeSkipsAnExhaustiveSearchTheTimeLimitWouldCut(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	args := withoutPaths(swapSpace(h, []string{"29933", "40362"}, 600))
	args["max_ms"] = 5
	res := call(t, s, "optimize", args)
	if m := res["space"].(map[string]any)["method"]; m != "coordinate_ascent" || res["exhaustive_skipped"] == nil {
		t.Fatalf("method %v, exhaustive_skipped %v", m, res["exhaustive_skipped"])
	}
}

// TestOptimizeLocalSearchFindsAThreeForOneSwap: at full points a 3-point Life path needs three removals at once; no
// single move or one-for-one swap improves on the build, so only a k-swap reaches it.
func TestOptimizeLocalSearchFindsAThreeForOneSwap(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	base := lifeOf(t, s, h)
	res := call(t, s, "optimize", swapSpace(h, []string{"15073", "46910", "37504"}, 100))
	if m := res["space"].(map[string]any)["method"]; m != "coordinate_ascent" {
		t.Fatalf("method %v", m)
	}
	top := rows(res)[0]
	var life float64
	for _, d := range top["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == "Life" {
			life = row["after"].(float64)
		}
	}
	if top["feasible"] != true || life < base+250 {
		t.Fatalf("base Life %v, top after %v evaluations: %v", base, res["evaluations"], top)
	}
}

// TestCoordinateAscentTriesCompoundMoves: three single moves that each score below the start, and any two of them, beat
// it all together; the beam never holds a pair, so only a compound of the best single moves reaches it.
func TestCoordinateAscentTriesCompoundMoves(t *testing.T) {
	slots := make([]optSlot, 3)
	for i := range slots {
		slots[i] = optSlot{name: fmt.Sprint(i), kind: slotOne, n: 3, start: []int{0}}
	}
	score := func(st state) float64 {
		k := st.key()
		switch {
		case k == "0|0|0":
			return 10
		case k == "2|2|2":
			return 20
		case strings.Count(k, "2") == 1 && strings.Count(k, "0") == 2:
			return 9.5
		}
		return 5
	}
	var best *evaluation
	for _, seed := range []int64{1, 2, 3} {
		best = nil
		coordinateAscent(slots, seed, func(st state) *evaluation {
			ev := &evaluation{state: st.clone(), score: score(st), feasible: true, changes: changesOf(slots, st)}
			if best == nil || better(ev, best) {
				best = ev
			}
			return ev
		})
		if best.state.key() != "2|2|2" {
			t.Fatalf("seed %d: best %v (%v)", seed, best.state, best.score)
		}
	}
}
