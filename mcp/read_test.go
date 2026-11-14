package mcp

import (
	"testing"
)

func TestGetBuildSectionsStayWithinBudget(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	for _, sec := range sections {
		res := assertSize(t, s, "get_build", map[string]any{"handle": h, "section": sec}, capSection)
		if res["section"] != sec {
			t.Fatalf("section %s answered %v", sec, res["section"])
		}
		assertSize(t, s, "get_build", map[string]any{"handle": h, "section": sec, "detail": true}, capSectionDetail)
	}
	sum := call(t, s, "get_build", map[string]any{"handle": h})
	if sum["class"] != "Marauder" || sum["main_skill"].(map[string]any)["name"] != "Vaal Double Strike" {
		t.Fatalf("summary = %v", sum)
	}
	items := call(t, s, "get_build", map[string]any{"handle": h, "section": "items"})
	first := items["items"].([]any)[0].(map[string]any)
	if first["slot"] != "Weapon 1" || len(first["key_mods"].([]any)) == 0 || len(first["key_mods"].([]any)) > 4 {
		t.Fatalf("first item row = %v", first)
	}
	skills := call(t, s, "get_build", map[string]any{"handle": h, "section": "skills"})
	mains := 0
	for _, g := range skills["groups"].([]any) {
		if g.(map[string]any)["main"] == true {
			mains++
		}
	}
	if mains != 1 {
		t.Fatalf("%d main groups", mains)
	}
	tree := call(t, s, "get_build", map[string]any{"handle": h, "section": "tree"})
	if len(tree["notables"].([]any)) == 0 || tree["small_node_count"].(float64) == 0 {
		t.Fatalf("tree section = %v", tree)
	}
	if e := callErr(t, s, "get_build", map[string]any{"handle": h, "section": "gear"}); e.Code != CodeInvalidArgument {
		t.Fatalf("unknown section: %s", e.Code)
	}
}

func TestGetBuildWithoutHandleListsHandles(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "get_build", map[string]any{})
	list := res["handles"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["handle"] != h {
		t.Fatalf("handles = %v", list)
	}
}

func TestGetStatsGroupsExplainAndUnknownKeys(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := assertSize(t, s, "get_stats", map[string]any{"handle": h}, capStats)
	for _, k := range []string{"MainDPS", "Life", "FireResist", "TotalEHP"} {
		if _, ok := res["stats"].(map[string]any)[k]; !ok {
			t.Fatalf("summary stats lack %s", k)
		}
	}
	res = call(t, s, "get_stats", map[string]any{"handle": h, "groups": []string{"resist"}, "stats": []string{"Lfie"}})
	if stat(res, "FireResistTotal") == 0 {
		t.Fatal("the resist group has no FireResistTotal")
	}
	unknown := res["unknown"].([]any)[0].(map[string]any)
	if unknown["key"] != "Lfie" || unknown["did_you_mean"].([]any)[0] != "Life" {
		t.Fatalf("unknown = %v", unknown)
	}

	res = assertSize(t, s, "get_stats", map[string]any{"handle": h, "explain": "Life", "stats": []string{"Life"}}, capStatsDetail)
	ex := res["explain"].(map[string]any)
	if ex["value"] != stat(res, "Life") || ex["base"].(float64) <= 0 {
		t.Fatalf("explain = %v", ex)
	}
	if n := len(ex["sources"].([]any)); n == 0 || n > 8 {
		t.Fatalf("explain lists %d sources", n)
	}
	if e := callErr(t, s, "get_stats", map[string]any{"handle": h, "groups": []string{"loot"}}); e.Code != CodeInvalidArgument {
		t.Fatalf("unknown group: %s", e.Code)
	}
}
