package mcp

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// swapSpace adds 1-point Life nodes and removes allocated 1-point nodes that grant no Life, at no net points: every
// gain is a swap of one removal for one add.
func swapSpace(h string, adds []string, budget int) map[string]any {
	return map[string]any{"handle": h, "budget": budget, "max_ms": 60000, "objective": map[string]any{"stat": "Life"},
		"slots":       []any{map[string]any{"kind": "nodes", "candidates": adds, "remove": []string{"8001", "12795", "8629", "8872"}}},
		"constraints": []any{map[string]any{"stat": "points_added", "op": "<=", "value": 0}}}
}

func TestOptimizeSwapsNodesAtFullPoints(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	base := lifeOf(t, s, h)
	for _, seed := range []int{1, 2, 3} {
		args := swapSpace(h, []string{"29933", "40362", "58271", "65034", "65167", "31628"}, 300)
		args["seed"] = seed
		res := call(t, s, "optimize", args)
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
		if top["feasible"] != true || top["points_added"].(float64) > 0 || life <= base {
			t.Fatalf("seed %d found no swap after %v evaluations: base Life %v, top %v", seed, res["evaluations"], base, top)
		}
	}
}

// withoutPaths turns paths off in a swap space: its adds are 1-point nodes either way, and its removals are not
// offered with their dead-end paths.
func withoutPaths(args map[string]any) map[string]any {
	args["slots"].([]any)[0].(map[string]any)["path"] = false
	return args
}

// TestOptimizeExhaustsASpaceWithinBudget: a 512-state space under a 600 budget is searched exhaustively.
func TestOptimizeExhaustsASpaceWithinBudget(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "optimize", withoutPaths(swapSpace(h, []string{"29933", "40362", "58271", "65034", "65167"}, 600)))
	space := res["space"].(map[string]any)
	if space["size"] != float64(512) || space["method"] != "exhaustive" || res["proven_optimal"] != true {
		t.Fatalf("space %v proven %v stopped %v", space, res["proven_optimal"], res["stopped"])
	}
}

// TestOptimizeRowsNameNodeIDsAndCollapseDependents: removing a node and its dependent is the same build as removing
// the node alone, so it is one row, and every node is named with its id.
func TestOptimizeRowsNameNodeIDsAndCollapseDependents(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	base, _ := s.handles[h].parseRev(0)
	v := specTreeVersion(base)
	deps := data.TreeVersions[v].CalculateDependentNodes(slices.Clone(base.Build.PassiveNodes), 6, classRoots(base, v))
	if len(deps) != 1 {
		t.Fatalf("Twin Terrors has dependents %v", deps)
	}
	res := call(t, s, "optimize", map[string]any{"handle": h, "top": 5, "objective": map[string]any{"stat": "Life"},
		"slots": []any{map[string]any{"kind": "nodes", "candidates": []string{"29933"}, "remove": []string{"6", fmt.Sprint(deps[0])}}}})
	seen := map[string]bool{}
	for _, r := range rows(res) {
		choice := r["choices"].([]any)[0].(map[string]any)["choice"].(map[string]any)
		for _, n := range append(choice["add"].([]any), choice["remove"].([]any)...) {
			if !strings.Contains(n.(string), "(") {
				t.Fatalf("node named without its id: %v", choice)
			}
		}
		k := fmt.Sprint(r["points_added"], r["delta"])
		if seen[k] {
			t.Fatalf("two rows describe one build: %v", rows(res))
		}
		seen[k] = true
	}
	if got := len(rows(res)); got != 5 {
		t.Fatalf("%d rows for 6 distinct builds: %v", got, rows(res))
	}
}
