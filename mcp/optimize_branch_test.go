package mcp

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// TestOptimizeSwapsALeafBranch: at full points, Blood Magic (three points on its path) gains Life only once the three
// points of Heartseeker's dead end (Heartseeker, its mastery and the path node that leads only to them) are freed;
// removing Heartseeker alone frees one point, so only the branch swap improves.
func TestOptimizeSwapsALeafBranch(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	base := lifeOf(t, s, h)
	res := call(t, s, "optimize", map[string]any{"handle": h, "budget": 100, "objective": map[string]any{"stat": "Life"},
		"slots":       []any{map[string]any{"kind": "nodes", "candidates": []string{"57279"}, "remove": []string{"65502"}}},
		"constraints": []any{map[string]any{"stat": "points_added", "op": "<=", "value": 0}}})
	top := rows(res)[0]
	var life float64
	for _, d := range top["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == "Life" {
			life = row["after"].(float64)
		}
	}
	if c := fmt.Sprint(top["choices"]); top["feasible"] != true || !strings.Contains(c, "57279") || life <= base {
		t.Fatalf("no branch swap for Blood Magic: base Life %v, top %v", base, top)
	}
}

// TestOptimizeRowsNameEveryRemovedNode: a removal row lists every node applying it takes away, the dead-end path and
// mastery as well as the node named.
func TestOptimizeRowsNameEveryRemovedNode(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "optimize", map[string]any{"handle": h, "budget": 100, "objective": map[string]any{"stat": "Life"}, "apply": true,
		"slots":       []any{map[string]any{"kind": "nodes", "candidates": []string{"57279"}, "remove": []string{"65502"}}},
		"constraints": []any{map[string]any{"stat": "points_added", "op": "<=", "value": 0}}})
	if res["applied"] == nil {
		t.Fatalf("nothing applied: %v", res)
	}
	choice := rows(res)[0]["choices"].([]any)[0].(map[string]any)["choice"].(map[string]any)
	listed, _ := choice["removed_nodes"].([]any)
	hd := s.handles[h]
	before, _ := hd.parseRev(0)
	after, _ := hd.parseRev(hd.cur)
	var want []string
	for _, id := range before.Build.PassiveNodes {
		if !slices.Contains(after.Build.PassiveNodes, id) {
			want = append(want, fmt.Sprintf("(%d)", id))
		}
	}
	if len(want) < 2 || len(listed) != len(want) {
		t.Fatalf("removed_nodes %v, applying took away %v", listed, want)
	}
	for _, w := range want {
		if !slices.ContainsFunc(listed, func(l any) bool { return strings.HasSuffix(l.(string), w) }) {
			t.Fatalf("removed_nodes %v misses %s", listed, w)
		}
	}
}
