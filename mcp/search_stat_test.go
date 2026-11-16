package mcp

import (
	"strings"
	"testing"
)

func TestSearchStatFindsExertKeys(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Intimidating Cry"}, "slot": "Helmet"})})
	stats := call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"ExertedAttackUptimeRatio"}})
	if _, ok := stats["stats"].(map[string]any)["ExertedAttackUptimeRatio"]; !ok {
		t.Fatalf("the control build has no exert output: %v", stats)
	}
	res := call(t, s, "search_data", map[string]any{"kind": "stat", "query": "exert", "handle": h, "limit": 25})
	found := false
	for _, r := range asList(res["results"]) {
		if k, _ := r.(map[string]any)["key"].(string); strings.Contains(k, "Exert") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no exert key: %v", res["results"])
	}
}

// An empty stat search with a handle says the build calculates no such output (the exert keys exist only for
// attacks an exerting warcry boosts) instead of suggesting other words.
func TestSearchStatExplainsAbsentBuildOutputs(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch"})["handle"].(string)
	res := call(t, s, "search_data", map[string]any{"kind": "stat", "query": "Zorblax", "handle": h})
	if hint, _ := res["hint"].(string); !strings.Contains(hint, "only for builds that use them") {
		t.Fatalf("hint = %q: %v", res["hint"], res)
	}
	res = call(t, s, "search_data", map[string]any{"kind": "stat", "query": "Zorblax"})
	if hint, _ := res["hint"].(string); !strings.Contains(hint, "handle") {
		t.Fatalf("hint without a handle = %q: %v", res["hint"], res)
	}
}
