package mcp

import (
	"math"
	"strings"
	"testing"
)

// swapsBuild is a Templar casting Freezing Pulse with Controlled Destruction and Blind, and self-casting Assassin's Mark.
func swapsBuild(t *testing.T, s *Server) string {
	t.Helper()
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse", "Controlled Destruction", "Blind"}, "level": 20, "quality": 0},
		map[string]any{"op": "add_group", "gems": []string{"Assassin's Mark"}, "level": 20, "quality": 0},
		map[string]any{"op": "set_main", "group": 1})})
	return h
}

// TestScanSwaps: the swaps part replaces the useless Blind with a damage support, and Assassin's Mark (no crit on the
// build) with a curse that raises cold damage; every row gains; apply commits the row's gain.
func TestScanSwaps(t *testing.T) {
	s := testServer(t)
	h := swapsBuild(t, s)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"swaps"}, "per_slot": 3})
	rows := res["results"].([]any)
	var support, curse map[string]any
	for _, r := range rows {
		r := r.(map[string]any)
		if r["part"] != "swaps" || num(r, "gain") <= 0 || !strings.HasPrefix(r["id"].(string), "gemswap|") {
			t.Fatalf("row %v", r)
		}
		switch {
		case support == nil && r["replaces"] == "Blind":
			support = r
		case curse == nil && r["replaces"] == "Assassin's Mark":
			curse = r
		}
	}
	if support == nil || curse == nil {
		t.Fatalf("no Blind or Assassin's Mark swap: %v", resultNames(res))
	}
	for _, r := range []map[string]any{support, curse} {
		applied := call(t, s, "scan_build", map[string]any{"handle": h, "apply": r["id"], "dry_run": true, "watch": []string{"MainDPS"}})
		if d := deltaOf(applied, "MainDPS"); math.Abs(d-num(r, "gain")) > 1 {
			t.Fatalf("apply %s moved MainDPS %v, row gain %v", r["id"], d, r["gain"])
		}
	}
}
