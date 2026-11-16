package mcp

import (
	"slices"
	"strings"
	"testing"
)

// customModsFixture is the shared fixture with a custom modifier standing in for Dismembering's line PoB cannot parse.
func customModsFixture(t *testing.T, s *Server) string {
	t.Helper()
	h := importFixture(t, s)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "customMods", "value": "10% chance to Maim on Hit"})})
	return h
}

func customModsFlagged(list []any) bool {
	return slices.ContainsFunc(list, func(x any) bool {
		s := x.(string)
		return strings.HasPrefix(s, "customMods: ") && strings.Contains(s, "Dismembering")
	})
}

// TestRankRemovingAnUnparsedNodeWarnsAboutCustomMods: removing a node whose line PoB cannot parse shows no loss; with
// custom modifiers set, its ranked removal says one may stand in for it and still applies.
func TestRankRemovingAnUnparsedNodeWarnsAboutCustomMods(t *testing.T) {
	s := testServer(t)
	h := customModsFixture(t, s)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "node_types": []string{"notable"}, "limit": 25})
	row := removeRow(t, res, 37647)
	if f, _ := row["flags"].([]any); !customModsFlagged(f) {
		t.Fatalf("rank row: %v", row)
	}
	res = call(t, s, "rank_candidates", map[string]any{"handle": importFixture(t, s), "kind": "remove_nodes", "node_types": []string{"notable"}, "limit": 25})
	if f, _ := removeRow(t, res, 37647)["flags"].([]any); customModsFlagged(f) {
		t.Fatalf("flagged without custom modifiers: %v", f)
	}
}
