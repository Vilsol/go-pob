package mcp

import (
	"math"
	"slices"
	"strings"
	"testing"
)

// staffMasteryBuild is a Templar casting Freezing Pulse with a Staff Mastery allocated on its defence effect; its crit
// multiplier effect only applies while the build has not crit recently, which the default config assumes.
func staffMasteryBuild(t *testing.T, s *Server) string {
	t.Helper()
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse"}}, map[string]any{"op": "set_main", "group": 1})})
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "5230"}, map[string]any{"op": "set_mastery", "node": "5230", "effect": "28589"})})
	return h
}

// TestRankQuestionsConditionsAssumedOff: "if you haven't dealt a Critical Strike Recently" gains only because the
// config leaves Crit Recently off; the switch row names that condition, and worst_case scores it with the condition
// on, as setting it and switching the mastery gives.
func TestRankQuestionsConditionsAssumedOff(t *testing.T) {
	s := testServer(t)
	h := staffMasteryBuild(t, s)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "MainDPS", "node_types": []string{"mastery"}, "worst_case": true, "max_points": 1})
	var row map[string]any
	for _, r := range res["mastery_switches"].([]any) {
		if r := r.(map[string]any); r["effect"] == float64(65233) {
			row = r
		}
	}
	if row == nil || row["gain"].(float64) <= 0 || !flagged(row, "config: ", "conditionCritRecently=on") {
		t.Fatalf("crit multiplier switch: %v", row)
	}
	f := call(t, s, "new_build", map[string]any{"class": "Templar", "from_handle": h})["handle"].(string)
	call(t, s, "edit_build", map[string]any{"handle": f, "ops": ops(map[string]any{"op": "set_config", "var": "conditionCritRecently", "value": true})})
	want := dryDelta(t, s, "edit_tree", f, mainDPS, map[string]any{"op": "set_mastery", "node": "5230", "effect": "65233"})
	if wg, ok := row["worst_gain"].(float64); !ok || math.Abs(wg-want) > math.Max(0.11, 0.001*math.Abs(want)) {
		t.Fatalf("worst_gain %v; with Crit Recently on the switch gains %v", row["worst_gain"], want)
	}
}

// TestRankWorstCaseRescoresDefences: with Fortified forced on, removing Rampart costs EHP and max hit only through the
// assumption; worst_d carries what removing it costs with the conditions off, next to d's forced-on columns.
func TestRankWorstCaseRescoresDefences(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "MainDPS", "limit": 25, "worst_case": true,
		"keep": []any{map[string]any{"stat": "Armour", "op": "<=", "value": 0.9, "relative": true}}})
	row := removeRow(t, res, 1340)
	f := unforced(t, s, h, res)
	wd, ok := row["worst_d"].(map[string]any)
	d, _ := row["d"].(map[string]any)
	if !ok {
		t.Fatalf("Rampart row without worst_d: %v", row)
	}
	for col, key := range map[string]string{"dEHP": "TotalEHP", "dLife": "Life", "dDPS": mainDPS} {
		want := dryDelta(t, s, "edit_tree", f, key, map[string]any{"op": "deallocate", "node": "1340"})
		if got, _ := wd[col].(float64); math.Abs(got-want) > math.Max(1, 0.001*math.Abs(want)) {
			t.Fatalf("Rampart worst_d %s %v; with the conditions off removing it moves %s by %v (d: %v)", col, got, key, want, d)
		}
	}
	if wd["dEHP"] == d["dEHP"] {
		t.Fatalf("worst_d %v equals d %v", wd, d)
	}
}

// TestRankFlagsRemovedSourcesOfForcedConditions: Fortified is forced on, so removing the node that fortifies costs
// nothing in any calculation; its row names the condition and the line, and a node that only reads Fortify does not.
func TestRankFlagsRemovedSourcesOfForcedConditions(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "50515"})})
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "MainDPS", "limit": 25})
	if r := removeRow(t, res, 50515); !flagged(r, "config source: buffFortification", "chance to Fortify") {
		t.Fatalf("fortifying node row: %v", r)
	}
	if r := removeRow(t, res, 1340); flagged(r, "config source: ", "") {
		t.Fatalf("Rampart reads Fortify, it does not grant it: %v", r)
	}
}

// TestRankConfigFlagsNameOnlyMovingConditions: a config flag names the conditions whose toggle moves that row's gain,
// never every forced condition.
func TestRankConfigFlagsNameOnlyMovingConditions(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "MainDPS", "node_types": []string{"mastery"}, "limit": 25})
	n := len(res["forced_conditions"].([]any))
	for _, r := range rows(res) {
		flags, _ := r["flags"].([]any)
		for _, f := range flags {
			if s, _ := f.(string); strings.HasPrefix(s, "config: ") && strings.Count(s, ",")+1 >= min(n, 5) {
				t.Fatalf("%v names %d of %d forced conditions: %v", r["name"], strings.Count(s, ",")+1, n, s)
			}
		}
	}
}

// TestSourceMatchesOnlyLinesThatInflictTheCondition: a forced condition's source is a line or gem that inflicts it, not
// one that shares its first word: Lightning Penetration does not expose, Shockwave does not shock.
func TestSourceMatchesOnlyLinesThatInflictTheCondition(t *testing.T) {
	for _, c := range []struct {
		v, line string
		want    bool
	}{
		{"buffFortification", "Fortify", true},
		{"buffFortification", "Melee Hits have 20% chance to Fortify", true},
		{"buffFortification", "+1% to maximum Armour per Fortification", false},
		{"conditionEnemyShocked", "20% chance to Shock", true},
		{"conditionEnemyShocked", "Shockwave", false},
		{"conditionEnemyIntimidated", "Intimidating Cry", true},
		{"conditionEnemyLightningExposure", "Lightning Penetration", false},
		{"conditionEnemyLightningExposure", "Inflict Lightning Exposure on Hit", true},
		{"buffArcaneSurge", "Gain Arcane Surge when you or your Totems Hit an Enemy with a Spell", true},
		{"buffArcaneSurge", "Arcane Surge also grants 20% more Spell Damage to you", false},
		{"buffArcaneSurge", "Arcane Surge", true},
		{"buffArcaneSurge", "20% increased Effect of Arcane Surge on you", false},
	} {
		if got := sourceMatches(c.v, c.line); got != c.want {
			t.Errorf("%s / %q: %v, want %v", c.v, c.line, got, c.want)
		}
	}
}

// TestEditFlagsRemovedSourcesOfForcedConditions: a dry run that removes what fortifies, with Fortified forced on,
// warns though no stat moves.
func TestEditFlagsRemovedSourcesOfForcedConditions(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "50515"})})
	dry := call(t, s, "edit_tree", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "deallocate", "node": "50515"})})
	w, _ := dry["warnings_added"].([]any)
	if !slices.ContainsFunc(w, func(x any) bool { return strings.HasPrefix(x.(string), "config source: buffFortification") }) {
		t.Fatalf("removing the fortifying node: %v", dry)
	}
}
