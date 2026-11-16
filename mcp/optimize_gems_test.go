package mcp

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// TestOptimizeVariesGemLevelAndQuality: a gem slot scores the levels and qualities listed, the best is what
// set_gem_level gives, and a corrupted gem never takes both its extra level and quality above 20.
func TestOptimizeVariesGemLevelAndQuality(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse", "Controlled Destruction"}, "level": 1, "quality": 0}, map[string]any{"op": "set_main", "group": 1})})
	slot := map[string]any{"kind": "gem", "group": 1, "gem": "Freezing Pulse", "levels": []int{1, 10, 20}, "qualities": []int{0, 20}}
	res := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"}, "slots": []any{slot}})
	top := rows(res)[0]
	choice := fmt.Sprint(top["choices"].([]any)[0].(map[string]any)["choice"])
	if !strings.Contains(choice, "level 20") {
		t.Fatalf("best choice %v", choice)
	}
	want := dryDelta(t, s, "edit_skills", h, mainDPS, map[string]any{"op": "set_gem_level", "group": 1, "gem": "Freezing Pulse", "level": 20, "quality": 20})
	if got := top["d"].(map[string]any)["dDPS"].(float64); math.Abs(got-want) > math.Max(0.11, 0.001*math.Abs(want)) {
		t.Fatalf("row dDPS %v, set_gem_level 20/20 gives %v", got, want)
	}
	if e := callErr(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"},
		"slots": []any{map[string]any{"kind": "gem", "group": 1, "gem": "Freezing Pulse", "levels": []int{21}}}}); e.Code != CodeInvalidValue {
		t.Fatalf("level 21 uncorrupted: %s %s", e.Code, e.Message)
	}
	slot = map[string]any{"kind": "gem", "group": 1, "gem": "Freezing Pulse", "levels": []int{20, 21}, "qualities": []int{20, 23}, "corrupted": true}
	res = call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"}, "top": 5, "slots": []any{slot}})
	for _, r := range rows(res) {
		if c := fmt.Sprint(r["choices"].([]any)[0].(map[string]any)["choice"]); strings.Contains(c, "level 21") && strings.Contains(c, "quality 23") {
			t.Fatalf("corrupted twice: %v", c)
		}
	}
}
