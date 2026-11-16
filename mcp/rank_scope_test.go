package mcp

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// TestRankRemoveNodesHonoursNodeTypes: node_types limits removal rows as it limits additions.
func TestRankRemoveNodesHonoursNodeTypes(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	list := rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "MainDPS", "limit": 25, "node_types": []string{"notable"}}))
	if len(list) == 0 {
		t.Fatal("no rows")
	}
	for _, r := range list {
		if r["type"] != "notable" {
			t.Fatalf("node_types notable, row %v", r)
		}
	}
}

// TestOptimizeRemoveResolvesAllocatedNodes: a name many nodes share means the allocated ones when removing.
func TestOptimizeRemoveResolvesAllocatedNodes(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	hd, _ := s.get(h)
	b, _ := hd.parseRev(hd.cur)
	e := callErr(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"},
		"slots": []any{map[string]any{"kind": "nodes", "remove": []string{"Strength"}}}})
	if e.Code != CodeAmbiguous || len(e.Candidates) < 2 {
		t.Fatalf("remove Strength: %s %s %v", e.Code, e.Message, e.Candidates)
	}
	for _, c := range e.Candidates {
		if id := c.(map[string]any)["id"].(int64); !slices.Contains(b.Build.PassiveNodes, id) {
			t.Fatalf("candidate %v is not allocated", c)
		}
	}
	id := e.Candidates[0].(map[string]any)["id"].(int64)
	call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"},
		"slots": []any{map[string]any{"kind": "nodes", "remove": []string{fmt.Sprint(id)}}}})
}

// TestOptimizeSwitchesAllocatedMasteriesByName: listing an allocated mastery offers its other effects as switches,
// labelled as such and never as an add.
func TestOptimizeSwitchesAllocatedMasteriesByName(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"}, "top": 5,
		"slots": []any{map[string]any{"kind": "nodes", "candidates": []string{"8629"}, "k": 1}}})
	switched := false
	for _, r := range rows(res) {
		choice := r["choices"].([]any)[0].(map[string]any)["choice"].(map[string]any)
		if strings.Contains(fmt.Sprint(choice["add"]), "8629") {
			t.Fatalf("allocated mastery listed as an add: %v", choice)
		}
		if sw, ok := choice["switch"].([]any); ok && len(sw) > 0 {
			if !strings.Contains(fmt.Sprint(sw), "(8629)") || !strings.Contains(fmt.Sprint(sw), "replaces") {
				t.Fatalf("switch label %v", sw)
			}
			switched = true
		}
	}
	if !switched {
		t.Fatalf("no switch row: %v", rows(res))
	}
}

// TestRankAndOptimizeHintsFitTheTool: a cut result says how to get more with what that tool takes.
func TestRankAndOptimizeHintsFitTheTool(t *testing.T) {
	for _, c := range []struct {
		kind  string
		limit int
		bad   string
	}{{"uniques", 8, "node_types"}, {"supports", 25, "node_types"}, {"nodes", 25, "raise limit"}} {
		if h := rankMoreHint(c.kind, c.limit); strings.Contains(h, c.bad) || h == "" {
			t.Fatalf("%s limit %d: %q", c.kind, c.limit, h)
		}
	}
	for _, top := range []int{1, 3, 5} {
		if h := optimizeMoreHint(top); strings.Contains(h, "lower top") || h == "" {
			t.Fatalf("top %d: %q", top, h)
		}
	}
}
