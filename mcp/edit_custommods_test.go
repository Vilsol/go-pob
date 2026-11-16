package mcp

import (
	"testing"
)

// TestEditRemovingAnUnparsedNodeWarnsAboutCustomMods: an edit that removes a node whose line PoB cannot parse, on a
// build with a custom modifier sharing its words, says one may stand in for it and still applies; without one, or with
// only unrelated custom modifiers, it does not.
func TestEditRemovingAnUnparsedNodeWarnsAboutCustomMods(t *testing.T) {
	s := testServer(t)
	for _, c := range []struct {
		h    string
		want bool
	}{{customModsFixture(t, s), true}, {importFixture(t, s), false}, {unrelatedCustomMods(t, s), false}} {
		dry := call(t, s, "edit_tree", map[string]any{"handle": c.h, "dry_run": true, "ops": ops(map[string]any{"op": "deallocate", "node": "37647"})})
		w, _ := dry["warnings_added"].([]any)
		if got := customModsFlagged(w); got != c.want {
			t.Fatalf("custom modifiers %v: flagged %v: %v", c.want, got, w)
		}
	}
}

func unrelatedCustomMods(t *testing.T, s *Server) string {
	t.Helper()
	h := importFixture(t, s)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "customMods",
		"value": "Warcries exert 999 additional attacks\n#a note about Maim\nStatic Strike deals 20% more damage"})})
	return h
}
