package mcp

import (
	"strings"
	"testing"
)

func firstResult(res map[string]any) map[string]any {
	list := res["results"].([]any)
	if len(list) == 0 {
		return nil
	}
	return list[0].(map[string]any)
}

func TestSearchDataEveryKindWithinBudget(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	cases := []struct {
		args  map[string]any
		check func(first map[string]any) bool
	}{
		{map[string]any{"kind": "gem", "query": "fireball"}, func(f map[string]any) bool { return f["name"] == "Fireball" && f["match"] == "exact" }},
		{map[string]any{"kind": "gem", "query": "complex trauma"}, func(f map[string]any) bool { return f["name"] == "Boneshatter of Complex Trauma" }},
		{map[string]any{"kind": "support", "query": "elemental focus"}, func(f map[string]any) bool { return f["name"] == "Elemental Focus" && f["support"] == true }},
		{map[string]any{"kind": "gem", "tags": []string{"fire", "spell"}}, func(f map[string]any) bool { return f != nil }},
		{map[string]any{"kind": "unique", "query": "kaom's heart", "slot": "Body Armour"}, func(f map[string]any) bool { return f["base"] == "Glorious Plate" }},
		{map[string]any{"kind": "base", "query": "astral plate"}, func(f map[string]any) bool { return f["name"] == "Astral Plate" }},
		{map[string]any{"kind": "node", "query": "heart of oak", "handle": h}, func(f map[string]any) bool { return f["name"] == "Heart of Oak" && f["path_cost"] != nil }},
		{map[string]any{"kind": "keystone", "query": "blood magic"}, func(f map[string]any) bool { return f["type"] == "keystone" }},
		{map[string]any{"kind": "mastery", "query": "life mastery"}, func(f map[string]any) bool { return len(f["effects"].([]any)) > 0 }},
		{map[string]any{"kind": "affix", "base": "Astral Plate", "query": "maximum life"}, func(f map[string]any) bool { return f["type"] == "Prefix" }},
		{map[string]any{"kind": "config", "query": "is the enemy a boss"}, func(f map[string]any) bool { return f["var"] == "enemyIsBoss" }},
		{map[string]any{"kind": "stat", "query": "effective hit pool"}, func(f map[string]any) bool { return f["key"] == "TotalEHP" }},
	}
	for _, c := range cases {
		res := assertSize(t, s, "search_data", c.args, capSearch)
		if n := len(res["results"].([]any)); n > 10 {
			t.Fatalf("%v returned %d rows", c.args, n)
		}
		if f := firstResult(res); !c.check(f) {
			t.Fatalf("%v: first result %v", c.args, f)
		}
	}
}

func TestSearchDataPagesWithCursor(t *testing.T) {
	s := testServer(t)
	page1 := call(t, s, "search_data", map[string]any{"kind": "support", "limit": 5})
	cursor, ok := page1["next_cursor"].(string)
	if !ok || page1["total"].(float64) <= 5 {
		t.Fatalf("page 1 = %v", page1)
	}
	page2 := call(t, s, "search_data", map[string]any{"kind": "support", "limit": 5, "cursor": cursor})
	if firstResult(page1)["name"] == firstResult(page2)["name"] {
		t.Fatal("page 2 repeats page 1")
	}
	big := assertSize(t, s, "search_data", map[string]any{"kind": "stat", "limit": 50}, capSearch*5)
	if len(big["results"].([]any)) == 0 {
		t.Fatal("limit 50 returned nothing")
	}
}

func TestSearchDataRefusals(t *testing.T) {
	s := testServer(t)
	if e := callErr(t, s, "search_data", map[string]any{"kind": "affix", "query": "life"}); e.Code != CodeInvalidArgument {
		t.Fatalf("affix without base: %s", e.Code)
	}
	if e := callErr(t, s, "search_data", map[string]any{"kind": "spell"}); e.Code != CodeInvalidArgument || len(e.Valid) != len(searchKinds) {
		t.Fatalf("unknown kind: %s %v", e.Code, e.Valid)
	}
	if e := callErr(t, s, "search_data", map[string]any{"kind": "affix", "base": "Astral Bucket"}); e.Code != CodeNotFound || len(e.DidYouMean) == 0 {
		t.Fatalf("misspelt base: %s %v", e.Code, e.DidYouMean)
	}
}

// TestSupportSearchCompatibilityNeedsAnActiveSkill: before the build has an active skill no support is marked
// incompatible; once it has one, its supports are marked compatible.
func TestSupportSearchCompatibilityNeedsAnActiveSkill(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Marauder"})["handle"].(string)
	args := map[string]any{"kind": "support", "query": "melee physical damage", "handle": h}
	if f := firstResult(call(t, s, "search_data", args)); f["name"] != "Melee Physical Damage" || f["compatible"] != nil {
		t.Fatalf("no skill yet: %v", f)
	}
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Sunder"}})})
	if f := firstResult(call(t, s, "search_data", args)); f["compatible"] != true {
		t.Fatalf("with Sunder: %v", f)
	}
}

// TestSearchAffixBenchAndVeiled: kind=affix lists the base's crafting bench and veiled mods beside its prefixes and
// suffixes, as PoB's "Add modifier" does.
func TestSearchAffixBenchAndVeiled(t *testing.T) {
	s := testServer(t)
	res := assertSize(t, s, "search_data", map[string]any{"kind": "affix", "base": "Heavy Belt", "query": "lucky while focused"}, capSearch)
	types := map[string]bool{}
	for _, r := range res["results"].([]any) {
		row := r.(map[string]any)
		if strings.HasPrefix(row["label"].(string), "Your Critical Strike Chance is Lucky while Focused") {
			types[row["type"].(string)] = true
		}
	}
	if !types["Crafted Suffix"] || !types["Veiled Suffix"] {
		t.Fatalf("bench and veiled rows: %v", res["results"])
	}
}
