package mcp

import (
	"fmt"
	"strings"
	"testing"
)

// overBudget is the fixture with three Life nodes added and its level lowered until it uses two passive points more
// than the level grants.
func overBudget(t *testing.T, s *Server) string {
	t.Helper()
	h := importFixture(t, s)
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "29933"}, map[string]any{"op": "allocate", "node": "40362"}, map[string]any{"op": "allocate", "node": "58271"})})
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_level", "level": 86})})
	return h
}

// TestOptimizeRepairsAnOverBudgetStart: a build over its passive point budget is no answer; optimize says so and its
// best result fits the budget at the least cost, here the Life nodes that add no damage.
func TestOptimizeRepairsAnOverBudgetStart(t *testing.T) {
	s := testServer(t)
	h := overBudget(t, s)
	res := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"}, "max_ms": 60000,
		"slots": []any{map[string]any{"kind": "nodes", "remove": []string{"29933", "40362", "58271", "8001", "12795", "8629", "8872", "6", "9469", "12809", "13807", "15085", "16860", "24528"}}}})
	pts := res["points"].(map[string]any)
	if pts["used"].(float64) <= pts["available"].(float64) {
		t.Fatalf("fixture is not over budget: %v", pts)
	}
	if res["start"] == nil {
		t.Fatalf("no word that the start is over budget: %v", keys(res))
	}
	top := rows(res)[0]
	total, dps := -1.0, -1.0
	for _, d := range top["delta"].([]any) {
		row := d.(map[string]any)
		switch row["stat"] {
		case pointsTotalKey:
			total = row["after"].(float64)
		case mainDPS:
			dps = row["after"].(float64) - row["before"].(float64)
		}
	}
	if top["feasible"] != true || total < 0 || total > pts["available"].(float64) || dps != 0 {
		t.Fatalf("top result %v: points_total %v of %v, MainDPS moves %v", top, total, pts["available"], dps)
	}
}

// TestOptimizeNeverAppliesAWorseResult: with the current ring excluded, every choice loses Life; the best of them is
// marked worse than the current build and apply leaves the build alone.
func TestOptimizeNeverAppliesAWorseResult(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	before := lifeOf(t, s, h)
	res := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "Life"}, "apply": true,
		"slots": []any{map[string]any{"kind": "item", "slot": "Ring 1", "keep_current": false, "uniques": []string{"Kalandra's Touch", "Andvarius"}}}})
	top := rows(res)[0]
	if top["score"].(float64) >= before {
		t.Fatalf("fixture: best ring gives %v Life, current %v", top["score"], before)
	}
	if top["worse_than_current"] != true || res["applied"] != nil || lifeOf(t, s, h) != before {
		t.Fatalf("top %v, applied %v, Life %v -> %v", top, res["applied"], before, lifeOf(t, s, h))
	}
}

// TestOptimizeRefusesAnOverAllocatedAscendancy: with all 8 ascendancy points spent, adding Defy Pain needs two more;
// PoB warns past 8 (Build.lua:902,926), so that result is infeasible and the current build stays the best.
func TestOptimizeRefusesAnOverAllocatedAscendancy(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "Aspect of Carnage"}, map[string]any{"op": "allocate", "node": "War Bringer"})})
	res := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "Life"},
		"slots": []any{map[string]any{"kind": "nodes", "candidates": []string{"Defy Pain"}}}})
	for _, r := range rows(res) {
		if picks := fmt.Sprint(r["choices"]); strings.Contains(picks, "Defy Pain") && r["feasible"] != false {
			t.Fatalf("over-allocated result is feasible: %v", r)
		}
	}
}
