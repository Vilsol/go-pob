package mcp

import (
	"slices"
	"strings"
	"testing"
)

// TestEditFlagsOppositeFullDPSMoves: an edit that raises MainDPS while FullDPS falls says so, as a ranked row does.
func TestEditFlagsOppositeFullDPSMoves(t *testing.T) {
	s := testServer(t)
	h := fullDPSBuild(t, s)
	for _, c := range []struct {
		ops  []any
		want bool
	}{
		{ops(map[string]any{"op": "add_gem", "group": 2, "gem": "Controlled Destruction"}, map[string]any{"op": "disable_group", "group": 1}), true},
		{ops(map[string]any{"op": "add_gem", "group": 2, "gem": "Controlled Destruction"}), false},
	} {
		dry := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "ops": c.ops})
		w, _ := dry["warnings_added"].([]any)
		if got := slices.ContainsFunc(w, func(x any) bool { return strings.HasPrefix(x.(string), "full dps: ") }); got != c.want {
			t.Errorf("%v: full dps flag %v: %v", c.ops, got, w)
		}
	}
}
