package mcp

import (
	"fmt"
	"strings"
	"testing"
)

// TestScanUnusableRowsRankLast: Bound Fate gains the most DPS of any unique here but loses over a fifth of TotalEHP:
// big_loss, not unusable; it and every unusable row rank after the clean rows, saying why.
func TestScanUnusableRowsRankLast(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	got := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"uniques"}, "per_slot": 3}))
	bound, clean := -1, false
	dirty := func(r map[string]any) bool { return r["unusable"] != nil || r["big_loss"] != nil }
	for i, r := range got {
		u, _ := r["unusable"].(string)
		if strings.HasPrefix(r["name"].(string), "Bound Fate") {
			bound = i
			if u != "" || !strings.Contains(fmt.Sprint(r["big_loss"]), "TotalEHP") {
				t.Errorf("Bound Fate unusable %q big_loss %v", u, r["big_loss"])
			}
		}
		if !dirty(r) && i > 0 && dirty(got[i-1]) {
			t.Fatalf("clean row %d ranks after an unusable one: %v", i+1, got)
		}
		clean = clean || !dirty(r)
	}
	if bound <= 0 || !clean {
		t.Fatalf("Bound Fate at %d, a clean row: %v; %v", bound+1, clean, got)
	}
}

// TestUnusableOrdering: a row that disables a skill, or has big_loss, ranks after clean rows by gain and by gain per
// divine.
func TestUnusableOrdering(t *testing.T) {
	mk := func() []map[string]any {
		return []map[string]any{
			{"part": "uniques", "name": "disables", "gain": 100.0, "gain_per_div": 50.0, "flags": []string{"disables: Cyclone"}},
			{"part": "uniques", "name": "ehp", "gain": 90.0, "gain_per_div": 40.0, "d": map[string]float64{"dEHP": -30}, "big_loss": []string{"TotalEHP -30%"}},
			{"part": "uniques", "name": "clean", "gain": 10.0, "gain_per_div": 5.0, "d": map[string]float64{"dEHP": -10}},
		}
	}
	marked := func() []map[string]any { r := mk(); markUnusable(r); markBigLoss(r); return r }
	for _, order := range [][]map[string]any{rankWorst(marked(), map[string]string{}), func() []map[string]any { r := marked(); sortPerDiv(r); return r }()} {
		if order[0]["name"] != "clean" || order[0]["unusable"] != nil || order[1]["unusable"] != "disables a skill" || order[2]["name"] != "ehp" || order[2]["unusable"] != nil {
			t.Fatalf("order %v", order)
		}
	}
}

// TestUnusableRequirementAndCost: a change that leaves an attribute requirement unmet, or too little mana to use a
// skill, is unusable and says which, so savings and gain per divine rankings and fit_budget pass it over.
func TestUnusableRequirementAndCost(t *testing.T) {
	rs := []map[string]any{
		{"name": "dex", "gain": 10.0, "flags": []string{"warning: You do not meet the Dexterity requirement"}},
		{"name": "mana", "gain": 10.0, "flags": []string{"warning: You do not have enough Mana to use: Static Strike"}},
		{"name": "clean", "gain": 1.0, "flags": []string{"warning: Unsupported mod in Body Armour"}},
	}
	markUnusable(rs)
	if rs[0]["unusable"] != "You do not meet the Dexterity requirement" || rs[1]["unusable"] != "You do not have enough Mana to use: Static Strike" || rs[2]["unusable"] != nil {
		t.Fatalf("marks %v", rs)
	}
	sortPerDiv(rs)
	if rs[0]["name"] != "clean" {
		t.Fatalf("order %v", rs)
	}
}

// TestUnusableNamesEveryFailedRequirement: a row that fails two attribute requirements names both.
func TestUnusableNamesEveryFailedRequirement(t *testing.T) {
	rows := []map[string]any{{"flags": []string{
		"warning: You do not meet the Strength requirement",
		"warning: You do not meet the Intelligence requirement",
	}}}
	markUnusable(rows)
	u, _ := rows[0]["unusable"].(string)
	if !strings.Contains(u, "Strength") || !strings.Contains(u, "Intelligence") {
		t.Fatalf("unusable %q", u)
	}
}
