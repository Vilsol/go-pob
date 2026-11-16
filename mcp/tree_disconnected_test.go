package mcp

import (
	"testing"
)

// PassiveSpec.lua:311 drops nodes left unconnected when the spec loads, so a path:false allocation that does not
// touch the allocated tree never counts.
func TestAllocateWithoutPathRefusesADisconnectedNode(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch", "ascendancy": "Elementalist"})["handle"].(string)
	found := call(t, s, "search_data", map[string]any{"kind": "node", "query": "maximum life", "handle": h, "limit": 50})
	var far, near map[string]any
	for _, r := range found["results"].([]any) {
		row := r.(map[string]any)
		c, _ := row["path_cost"].(float64)
		if far == nil && c >= 4 {
			far = row
		}
	}
	if far == nil {
		t.Fatal("no life node 4+ points from the Witch start")
	}
	e := callErr(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": far["id"], "path": false})})
	if e.Code != CodeNotReachable || e.OpIndex == nil || *e.OpIndex != 0 {
		t.Fatalf("disconnected path:false allocation: %s %v %q", e.Code, e.OpIndex, e.Message)
	}
	if got := s.handles[h].latest(); got != 0 {
		t.Fatalf("the refused edit committed revision %d", got)
	}

	res := call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": far["id"]})})
	used := points(res)["used"]
	b, _ := s.handles[h].parseRev(1)
	plan := planPaths(b, specTreeVersion(b))
	for _, r := range found["results"].([]any) {
		row := r.(map[string]any)
		id := int64(row["id"].(float64))
		if p := plan.path(id); len(p) == 1 {
			near = row
			break
		}
	}
	if near == nil {
		t.Fatal("no life node next to the allocated path")
	}
	res = call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": near["id"], "path": false})})
	if points(res)["used"] != used.(float64)+1 {
		t.Fatalf("adjacent path:false allocation used %v points, before %v", points(res)["used"], used)
	}
}
