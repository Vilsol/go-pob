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

func TestGetBuildSkillsPagesEveryGroup(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch"})["handle"].(string)
	const n = 60
	for done := 0; done < n; done += 10 {
		batch := make([]any, 0, 10)
		for range 10 {
			batch = append(batch, map[string]any{"op": "add_group", "gems": []string{"Fireball", "Spell Echo Support", "Elemental Focus Support", "Controlled Destruction Support", "Added Fire Damage Support", "Concentrated Effect Support"}})
		}
		call(t, s, "edit_skills", map[string]any{"handle": h, "ops": batch})
	}
	seen := map[float64]int{}
	cursor := ""
	for page := 0; ; page++ {
		args := map[string]any{"handle": h, "section": "skills", "detail": true}
		if cursor != "" {
			args["cursor"] = cursor
		}
		res := assertSize(t, s, "get_build", args, capSectionDetail)
		for _, g := range res["groups"].([]any) {
			seen[g.(map[string]any)["group"].(float64)]++
		}
		next, ok := res["next_cursor"].(string)
		if !ok {
			break
		}
		if page == 0 && (res["truncated"] != true || res["omitted"].(float64) <= 0) {
			t.Fatalf("a truncated page does not say how many groups it omitted: %v", keys(res))
		}
		cursor = next
	}
	for g := 1; g <= n; g++ {
		if seen[float64(g)] != 1 {
			t.Fatalf("group %d seen %d times across pages", g, seen[float64(g)])
		}
	}
}

// A gem saved with only its name resolves as PoB's FindSkillGem does (SkillsTab.lua:1193-1201), so a legacy support
// stays a support.
func TestGetBuildNameOnlyGemsKeepTheirKind(t *testing.T) {
	s := testServer(t)
	xml := `<PathOfBuilding><Build level="90" className="Witch" ascendClassName="None" mainSocketGroup="1"/>` +
		`<Skills><Skill enabled="true" mainActiveSkill="1">` +
		`<Gem nameSpec="Fireball" level="20" quality="0" enabled="true"/>` +
		`<Gem nameSpec="Awakened Lightning Penetration" level="5" quality="20" enabled="true"/>` +
		`<Gem nameSpec="Elemental Focus Support" level="20" quality="0" enabled="true"/>` +
		`</Skill></Skills><Tree activeSpec="1"><Spec classId="3" ascendClassId="0" nodes=""/></Tree><Items/><Config/></PathOfBuilding>`
	h := call(t, s, "import_build", map[string]any{"source": xml})["handle"].(string)
	gems := call(t, s, "get_build", map[string]any{"handle": h, "section": "skills"})["groups"].([]any)[0].(map[string]any)["gems"].([]any)
	want := []bool{false, true, true}
	if len(gems) != len(want) {
		t.Fatalf("gems = %v", gems)
	}
	for i, g := range gems {
		if g.(map[string]any)["support"] != want[i] {
			t.Fatalf("gem %v: support %v, want %v", g.(map[string]any)["name"], g.(map[string]any)["support"], want[i])
		}
	}
}

// TestExplainFullDPSListsItsSkills checks explain=FullDPS lists the Full DPS skills whose TotalDPS x count sum to it
// (Calcs.lua:254-255), and that a requested stat PoB did not compute is named rather than passed off as 0.
func TestExplainFullDPSListsItsSkills(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch", "level": 90})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Fireball", "Elemental Focus Support"}, "full_dps": true},
		map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse"}, "full_dps": true},
	)})
	res := call(t, s, "get_stats", map[string]any{"handle": h, "explain": "FullDPS", "stats": []string{"FullDPS", "NetManaRegen"}})
	ex := res["explain"].(map[string]any)
	srcs, _ := ex["sources"].([]any)
	if stat(res, "FullDPS") <= 0 || len(srcs) != 2 {
		t.Fatalf("FullDPS %v, sources %v", stat(res, "FullDPS"), ex)
	}
	if nc, _ := res["not_computed"].([]any); len(nc) != 1 || nc[0] != "NetManaRegen" {
		t.Fatalf("not_computed = %v", res["not_computed"])
	}
}
