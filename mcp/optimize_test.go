package mcp

import (
	"encoding/json"
	"testing"
)

func lifeOf(t *testing.T, s *Server, h string) float64 {
	t.Helper()
	return stat(call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"Life"}}), "Life")
}

// TestOptimizeExhaustiveIsOptimal scores every combination of a 4-ring x 4-node-subset space through the edit
// tools and checks the exhaustive search's first result is the best of them.
func TestOptimizeExhaustiveIsOptimal(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	rings := []string{"Kalandra's Touch", "Mark of the Elder", "Zana's Ingenuity"}
	nodes := []string{"Heart of Oak", "Blood Siphon"}
	res := assertSize(t, s, "optimize", map[string]any{"handle": h,
		"slots": []any{
			map[string]any{"kind": "item", "slot": "Ring 1", "uniques": rings},
			map[string]any{"kind": "nodes", "candidates": nodes},
		},
		"objective": map[string]any{"stat": "Life"},
	}, capOptimize)
	space := res["space"].(map[string]any)
	if space["method"] != "exhaustive" || space["size"] != float64(16) || res["proven_optimal"] != true {
		t.Fatalf("space = %v proven = %v", space, res["proven_optimal"])
	}
	if used := res["evaluations"].(map[string]any)["used"]; used != float64(16) {
		t.Fatalf("used %v evaluations for 16 combinations", used)
	}

	best := 0.0
	ringChoices := append([]string{""}, rings...)
	subsets := [][]string{{}, {nodes[0]}, {nodes[1]}, nodes}
	for _, ring := range ringChoices {
		for _, sub := range subsets {
			if ring != "" {
				call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_unique", "name": ring, "slot": "Ring 1"})})
			}
			for _, n := range sub {
				call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": n})})
			}
			best = max(best, lifeOf(t, s, h))
			call(t, s, "revert_build", map[string]any{"handle": h, "to_revision": 0})
		}
	}
	top := rows(res)[0]
	var after float64
	for _, d := range top["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == "Life" {
			after = row["after"].(float64)
		}
	}
	if after != best {
		t.Fatalf("optimize's best Life %v, brute force %v (%v)", after, best, top["choices"])
	}

	applied := call(t, s, "optimize", map[string]any{"handle": h,
		"slots":     []any{map[string]any{"kind": "item", "slot": "Ring 1", "uniques": rings}, map[string]any{"kind": "nodes", "candidates": nodes}},
		"objective": map[string]any{"stat": "Life"}, "apply": true,
	})
	if applied["applied"].(map[string]any)["revision"] == nil || lifeOf(t, s, h) != best {
		t.Fatalf("apply left Life %v, best %v", lifeOf(t, s, h), best)
	}
}

// supportSpace is C(9,5)=126 support sets x 5 ring choices = 630 combinations, one sweep 131 evaluations.
func supportSpace(h string, seed int, budget int) map[string]any {
	return map[string]any{"handle": h, "seed": seed, "budget": budget,
		"slots": []any{
			map[string]any{"kind": "supports", "k": 5, "candidates": []string{"Fortify", "Ruthless", "Close Combat", "Multistrike"}},
			map[string]any{"kind": "item", "slot": "Ring 1", "uniques": []string{"Kalandra's Touch", "Mark of the Elder", "Zana's Ingenuity", "Andvarius"}},
		},
		"objective":   map[string]any{"stat": "MainDPS"},
		"constraints": []any{map[string]any{"stat": "Life", "op": ">=", "value": 0.99, "relative": true}},
	}
}

func TestOptimizeCoordinateAscentIsDeterministicAndBudgeted(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	strip := func(m map[string]any) string {
		delete(m, "elapsed_ms")
		b, _ := json.Marshal(m)
		return string(b)
	}
	first := call(t, s, "optimize", supportSpace(h, 7, 300))
	space := first["space"].(map[string]any)
	if space["method"] != "coordinate_ascent" || space["size"].(float64) <= exhaustiveLimit || first["proven_optimal"] != false {
		t.Fatalf("space = %v", space)
	}
	used := first["evaluations"].(map[string]any)["used"].(float64)
	if used > 300 {
		t.Fatalf("used %v of a 300 budget", used)
	}
	t.Logf("%.0f evaluations in %v ms", used, first["elapsed_ms"])
	if again := call(t, s, "optimize", supportSpace(h, 7, 300)); strip(again) != strip(first) {
		t.Fatalf("seed 7 gave two answers:\n%s\n%s", strip(first), strip(again))
	}
	best := rows(first)[0]
	if best["feasible"] != true || best["score"].(float64) < first["objective"].(map[string]any)["current"].(float64) {
		t.Fatalf("best = %v", best)
	}

	timed := supportSpace(h, 7, 2000)
	timed["max_ms"] = 1
	res := call(t, s, "optimize", timed)
	if res["stopped"] != "time" || res["evaluations"].(map[string]any)["used"].(float64) > 5 {
		t.Fatalf("max_ms 1: stopped %v after %v", res["stopped"], res["evaluations"])
	}
}

func TestOptimizeRefusals(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	small := supportSpace(h, 1, 10)
	if e := callErr(t, s, "optimize", small); e.Code != CodeSpaceTooLarge || len(e.SizePerSlot) != 2 || len(e.Suggestions) == 0 {
		t.Fatalf("budget below one sweep: %s %v %v", e.Code, e.SizePerSlot, e.Suggestions)
	}
	var all []string
	for _, g := range gems() {
		if g.Support && len(all) < 60 {
			all = append(all, g.Name)
		}
	}
	huge := map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"},
		"slots": []any{map[string]any{"kind": "supports", "k": 5, "candidates": all}}}
	if e := callErr(t, s, "optimize", huge); e.Code != CodeSpaceTooLarge {
		t.Fatalf("C(60,5): %s", e.Code)
	}
	bad := supportSpace(h, 1, 300)
	bad["objective"] = map[string]any{"stat": "Lief"}
	if e := callErr(t, s, "optimize", bad); e.Code != CodeUnknownStat || e.DidYouMean[0] != "Life" {
		t.Fatalf("unknown objective: %s %v", e.Code, e.DidYouMean)
	}
	bad = supportSpace(h, 1, 300)
	bad["constraints"] = []any{map[string]any{"stat": "Resistance", "op": ">=", "value": 75}}
	if e := callErr(t, s, "optimize", bad); e.Code != CodeUnknownStat {
		t.Fatalf("unknown constraint: %s", e.Code)
	}
	dup := map[string]any{"handle": h, "objective": map[string]any{"stat": "Life"}, "slots": []any{
		map[string]any{"kind": "item", "slot": "Ring 1", "uniques": []string{"Mark of the Elder"}},
		map[string]any{"kind": "item", "slot": "Ring 1", "uniques": []string{"Mark of the Shaper"}},
	}}
	if e := callErr(t, s, "optimize", dup); e.Code != CodeInvalidArgument || e.SlotIndex == nil || *e.SlotIndex != 1 {
		t.Fatalf("duplicate slot: %s %v", e.Code, e.SlotIndex)
	}
	wrong := map[string]any{"handle": h, "objective": map[string]any{"stat": "Life"}, "slots": []any{
		map[string]any{"kind": "item", "slot": "Helmet", "uniques": []string{"Mark of the Elder"}},
	}}
	if e := callErr(t, s, "optimize", wrong); e.Code != CodeInvalidSlot || e.SlotIndex == nil {
		t.Fatalf("ring in helmet slot: %s", e.Code)
	}
}
