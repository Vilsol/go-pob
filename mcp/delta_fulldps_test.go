package mcp

import (
	"testing"
)

// TestEditDeltaListsFullDPSFirst: on a build with Full DPS skills, an edit's delta leads with FullDPS.
func TestEditDeltaListsFullDPSFirst(t *testing.T) {
	s := testServer(t)
	h := fullDPSBuild(t, s)
	res := call(t, s, "edit_build", map[string]any{"handle": h, "dry_run": true, "ops": ops(
		map[string]any{"op": "set_config", "var": "customMods", "value": "50% increased Spell Damage"})})
	delta := res["delta"].([]any)
	if len(delta) == 0 || delta[0].(map[string]any)["stat"] != "FullDPS" {
		t.Fatalf("delta %v", delta)
	}
}
