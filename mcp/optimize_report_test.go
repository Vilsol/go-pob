package mcp

import (
	"math"
	"strings"
	"testing"
)

// TestOptimizeAlwaysSaysWhyItStopped: an exhausted space stops on "space", a search that finds nothing better before
// its budget on "converged".
func TestOptimizeAlwaysSaysWhyItStopped(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	small := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "Life"},
		"slots": []any{map[string]any{"kind": "nodes", "candidates": []string{"Heart of Oak", "Blood Siphon"}}}})
	if small["stopped"] != "space" {
		t.Fatalf("exhaustive search stopped %v", small["stopped"])
	}
	big := call(t, s, "optimize", supportSpace(h, 7, 600))
	used := big["evaluations"].(map[string]any)["used"].(float64)
	if big["stopped"] != "converged" || used >= 600 {
		t.Fatalf("coordinate ascent stopped %v after %v of 600", big["stopped"], used)
	}
}

// TestOptimizeRowsCarryChangeColumnsAndPareto: each result row carries the change columns applying it gives, and
// pareto:true lists results no other evaluated one beats on both the objective and EHP.
func TestOptimizeRowsCarryChangeColumnsAndPareto(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	args := supportSpace(h, 7, 300)
	args["pareto"] = true
	res := assertSize(t, s, "optimize", args, capOptimizeMax)
	top := rows(res)[0]
	d, ok := top["d"].(map[string]any)
	if !ok {
		t.Fatalf("row without change columns: %v", top)
	}
	args["apply"] = true
	delete(args, "pareto")
	applied := call(t, s, "optimize", args)
	for _, row := range applied["delta"].([]any) {
		if r := row.(map[string]any); r["stat"] == mainDPS {
			if want := r["after"].(float64) - r["before"].(float64); math.Abs(d["dDPS"].(float64)-want) > math.Max(1, 0.001*math.Abs(want)) {
				t.Fatalf("dDPS %v, applying the row moves MainDPS by %v", d["dDPS"], want)
			}
		}
	}
	front := res["pareto"].([]any)
	if len(front) == 0 {
		t.Fatal("empty pareto front")
	}
	for i, a := range front {
		am := a.(map[string]any)
		for j, b := range front {
			bm := b.(map[string]any)
			if i != j && num(bm, "MainDPS") >= num(am, "MainDPS") && num(bm, "TotalEHP") >= num(am, "TotalEHP") &&
				(num(bm, "MainDPS") > num(am, "MainDPS") || num(bm, "TotalEHP") > num(am, "TotalEHP")) {
				t.Fatalf("pareto entry %v is dominated by %v", am, bm)
			}
		}
	}
}

// TestOptimizeMergesNearDuplicateRows: two choices that build the same stats are one row, which counts the other.
func TestOptimizeMergesNearDuplicateRows(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "Life"}, "top": 5,
		"slots": []any{map[string]any{"kind": "item", "slot": "Ring 1", "uniques": []string{"Mark of the Elder", "Mark of the Elder", "Andvarius"}}}})
	list := rows(res)
	merged := 0
	for _, r := range list {
		if n, ok := r["near_duplicates"].(float64); ok {
			merged += int(n)
		}
	}
	if len(list) != 3 || merged != 1 {
		t.Fatalf("%d rows, %d merged: %v", len(list), merged, list)
	}
}

// TestOptimizeHintKeepsWhatRankOneLoses: a top result that costs more than a fifth of TotalEHP gets a hint naming
// the constraint that keeps it, until a constraint bounds it.
func TestOptimizeHintKeepsWhatRankOneLoses(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	args := map[string]any{"handle": h, "objective": map[string]any{"stat": "Life", "goal": "minimize"},
		"slots": []any{map[string]any{"kind": "item", "slot": "Body Armour", "uniques": []string{"Tabula Rasa"}}}}
	if hint := call(t, s, "optimize", args)["hint"].(string); !strings.Contains(hint, `"stat":"TotalEHP","op":">=","value":0.9,"relative":true`) {
		t.Fatalf("hint %q names no TotalEHP constraint", hint)
	}
	args["constraints"] = []any{map[string]any{"stat": "TotalEHP", "op": ">=", "value": 0.5, "relative": true}}
	if hint := call(t, s, "optimize", args)["hint"].(string); strings.Contains(hint, "TotalEHP") {
		t.Fatalf("hint %q repeats a constraint the call has", hint)
	}
}
