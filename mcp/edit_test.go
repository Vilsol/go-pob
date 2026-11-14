package mcp

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

func ops(list ...map[string]any) []any {
	out := make([]any, len(list))
	for i, o := range list {
		out[i] = o
	}
	return out
}

func points(res map[string]any) map[string]any {
	return res["points"].(map[string]any)
}

func TestEditTreeAllocatesPathsAndDeallocatesDependents(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch", "ascendancy": "Elementalist"})["handle"].(string)
	found := call(t, s, "search_data", map[string]any{"kind": "node", "query": "maximum life", "handle": h, "limit": 50})
	var target map[string]any
	for _, r := range found["results"].([]any) {
		row := r.(map[string]any)
		if c, ok := row["path_cost"].(float64); ok && c >= 3 && c <= 8 {
			target = row
			break
		}
	}
	if target == nil {
		t.Fatal("no life node 3-8 points from the Witch start")
	}
	res := assertSize(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": target["id"]})}, capWrite)
	if points(res)["used"] != target["path_cost"] || res["revision"] != float64(1) {
		t.Fatalf("allocating %v (path %v) used %v points at revision %v", target["name"], target["path_cost"], points(res)["used"], res["revision"])
	}
	b, _ := s.handles[h].parseRev(1)
	var firstHop int64
	for _, id := range classRoots(b, data.LatestTreeVersion) {
		if slices.Contains(b.Build.PassiveNodes, id) {
			firstHop = id
		}
	}
	res = call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "deallocate", "node": firstHop})})
	if points(res)["used"] != float64(0) {
		t.Fatalf("deallocating the first path node left %v points", points(res)["used"])
	}

	var foreign treeNode
	list, _ := treeIndex(data.LatestTreeVersion)
	for _, n := range list {
		if n.Ascendancy == "Juggernaut" && n.Type == "ascendancy" {
			foreign = n
			break
		}
	}
	e := callErr(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": foreign.ID})})
	if e.Code != CodeNotReachable || e.OpIndex == nil || *e.OpIndex != 0 {
		t.Fatalf("foreign ascendancy node: %s %v", e.Code, e.OpIndex)
	}
}

func TestEditTreeBatchIsAtomic(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	e := callErr(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "allocate", "node": "Heart of Oak"},
		map[string]any{"op": "allocate", "node": "Qwzxv Plorb"},
	)})
	if e.Code != CodeNotFound || e.OpIndex == nil || *e.OpIndex != 1 {
		t.Fatalf("second op: %s %v", e.Code, e.OpIndex)
	}
	if got := call(t, s, "get_build", map[string]any{})["handles"].([]any)[0].(map[string]any)["latest_revision"]; got != float64(0) {
		t.Fatalf("a failed batch committed revision %v", got)
	}
}

func TestEditTreeSetsMasteryAndDryRunsDoNotCommit(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	dry := call(t, s, "edit_tree", map[string]any{"handle": h, "dry_run": true, "watch": []string{"Life"},
		"ops": ops(map[string]any{"op": "set_mastery", "node": 292, "effect": "+30 to maximum Life"})})
	if dry["dry_run"] != true || dry["revision"] != float64(0) {
		t.Fatalf("dry run = %v", dry)
	}
	moved := false
	for _, r := range dry["delta"].([]any) {
		if r.(map[string]any)["stat"] == "Life" {
			moved = true
		}
	}
	if !moved {
		t.Fatalf("picking +30 life moved no Life: %v", dry["delta"])
	}
	if latest := call(t, s, "get_build", map[string]any{})["handles"].([]any)[0].(map[string]any)["latest_revision"]; latest != float64(0) {
		t.Fatal("a dry run committed")
	}
	e := callErr(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_mastery", "node": 292, "effect": "+1 to Banana"})})
	if e.Code != CodeNotFound || len(e.Candidates) == 0 {
		t.Fatalf("unknown effect: %s %v", e.Code, e.Candidates)
	}
}

func TestEditItemsEquipsUniquesCraftsAndUnequips(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	before := call(t, s, "get_stats", map[string]any{"handle": h})
	dry := call(t, s, "edit_items", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "equip_unique", "name": "Kaoms Heart"})})
	if item := dry["item"].([]any)[0].(map[string]any); item["name"] != "Kaom's Heart, Glorious Plate" || item["text"] == "" {
		t.Fatalf("dry-run item = %v", item)
	}
	res := assertSize(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_unique", "name": "Kaom's Heart"})}, capWrite)
	after := call(t, s, "get_stats", map[string]any{"handle": h})
	if stat(after, "Life") < stat(before, "Life")+500 {
		t.Fatalf("Kaom's Heart took Life %v -> %v (%v)", stat(before, "Life"), stat(after, "Life"), res["applied"])
	}

	e := callErr(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_unique", "name": "Kaom's Heart", "slot": "Ring 1"})})
	if e.Code != CodeInvalidSlot || len(e.ValidSlots) != 1 || e.ValidSlots[0] != "Body Armour" {
		t.Fatalf("ring slot: %s %v", e.Code, e.ValidSlots)
	}

	affixes := call(t, s, "search_data", map[string]any{"kind": "affix", "base": "Two-Stone Ring", "limit": 50})
	var prefixes []string
	for _, r := range affixes["results"].([]any) {
		if row := r.(map[string]any); row["type"] == "Prefix" && len(prefixes) < 4 {
			prefixes = append(prefixes, row["label"].(string))
		}
	}
	res = call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Two-Stone Ring", "slot": "Ring 2",
		"affixes": []string{"+# to maximum Life", "+#% to Fire Resistance"}})})
	if res["applied"].([]any)[0] == "" {
		t.Fatal("craft applied nothing")
	}
	e = callErr(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Two-Stone Ring", "affixes": prefixes})})
	if e.Code != CodeAffixLimit || len(e.Chosen) != 3 {
		t.Fatalf("a fourth prefix: %s %v", e.Code, e.Chosen)
	}

	text := "Rarity: RARE\nTest Loop\nTwo-Stone Ring\n+50 to maximum Life\nThis line names no modifier at all"
	res = call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_text", "text": text, "slot": "Ring 1"})})
	bad := res["unsupported_lines"].([]any)[0].([]any)
	if len(bad) != 1 || bad[0] != "This line names no modifier at all" {
		t.Fatalf("unsupported_lines = %v", res["unsupported_lines"])
	}

	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "unequip", "slot": "Helmet"})})
	items := call(t, s, "get_build", map[string]any{"handle": h, "section": "items"})
	if empty := items["empty_slots"].([]any); len(empty) != 1 || empty[0] != "Helmet" {
		t.Fatalf("empty slots after unequip = %v", empty)
	}
}

func TestEditSkillsBuildsAMainSkill(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch"})["handle"].(string)
	res := assertSize(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Fireball", "Elemental Focus Support"}},
	)}, capWrite)
	if res["revision"] != float64(1) {
		t.Fatalf("add_group = %v", res)
	}
	sum := call(t, s, "get_build", map[string]any{"handle": h})
	ms := sum["main_skill"].(map[string]any)
	if ms["name"] != "Fireball" || len(ms["supports"].([]any)) != 1 || ms["group"] != float64(1) {
		t.Fatalf("main skill = %v", ms)
	}
	full := stat(call(t, s, "get_stats", map[string]any{"handle": h}), "MainDPS")
	if full <= 0 {
		t.Fatal("Fireball has no DPS")
	}

	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_gem_level", "group": 1, "gem": "Fireball", "level": 1})})
	if low := stat(call(t, s, "get_stats", map[string]any{"handle": h}), "MainDPS"); low >= full {
		t.Fatalf("level 1 Fireball MainDPS %v, level 20 %v", low, full)
	}
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_gem", "group": 1, "gem": "Spell Echo"},
		map[string]any{"op": "remove_gem", "group": 1, "gem": "Elemental Focus"},
		map[string]any{"op": "add_group", "gems": []string{"Frostbolt"}},
		map[string]any{"op": "set_main", "group": 2},
	)})
	skills := call(t, s, "get_build", map[string]any{"handle": h, "section": "skills"})["groups"].([]any)
	g1 := skills[0].(map[string]any)["gems"].([]any)
	if len(skills) != 2 || g1[1].(map[string]any)["name"] != "Spell Echo" || skills[1].(map[string]any)["main"] != true {
		t.Fatalf("groups = %v", skills)
	}
	e := callErr(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_gem", "group": 1, "gem": "Qwzxv Plorb"})})
	if e.Code != CodeNotFound {
		t.Fatalf("unknown gem: %s", e.Code)
	}
	if e := callErr(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_main", "group": 7})}); e.Code != CodeInvalidValue {
		t.Fatalf("missing group: %s", e.Code)
	}
}

func TestEditSkillsReportsUnsupportedSupports(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "ops": ops(
		map[string]any{"op": "add_gem", "group": 4, "gem": "Spell Echo"},
		map[string]any{"op": "add_gem", "group": 4, "gem": "Fortify"},
	)})
	bad, _ := res["unsupported"].([]any)
	if len(bad) != 1 || bad[0].(map[string]any)["gem"] != "Spell Echo" {
		t.Fatalf("unsupported = %v", res["unsupported"])
	}
	found := call(t, s, "search_data", map[string]any{"kind": "support", "handle": h, "query": "spell echo"})
	if f := firstResult(found); f["compatible"] != false {
		t.Fatalf("Spell Echo on Vaal Double Strike: %v", f)
	}
	found = call(t, s, "search_data", map[string]any{"kind": "support", "handle": h, "query": "fortify"})
	if f := firstResult(found); f["compatible"] != true {
		t.Fatalf("Fortify on Vaal Double Strike: %v", f)
	}
}

func TestEditBuildConfigLevelAndPantheon(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := assertSize(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "set_level", "level": 50},
		map[string]any{"op": "set_config", "var": "Is the enemy a Boss?", "value": "Uber"},
		map[string]any{"op": "set_pantheon", "major": "Brine King"},
		map[string]any{"op": "append_notes", "text": "levelling"},
	)}, capWrite)
	if len(res["applied"].([]any)) != 4 {
		t.Fatalf("applied = %v", res["applied"])
	}
	sum := call(t, s, "get_build", map[string]any{"handle": h})
	if sum["level"] != float64(50) || sum["pantheon"].(map[string]any)["major"] != "TheBrineKing" {
		t.Fatalf("summary = level %v pantheon %v", sum["level"], sum["pantheon"])
	}
	cfg := call(t, s, "get_build", map[string]any{"handle": h, "section": "config"})["config"].([]any)
	boss := ""
	for _, c := range cfg {
		if row := c.(map[string]any); row["var"] == "enemyIsBoss" {
			boss, _ = row["value"].(string)
		}
	}
	if boss != "Uber" {
		t.Fatalf("enemyIsBoss = %q", boss)
	}

	e := callErr(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "enemyIsBoss", "value": "Mega"})})
	if e.Code != CodeInvalidValue || len(e.Allowed) != 4 {
		t.Fatalf("bad list value: %s %v", e.Code, e.Allowed)
	}
	if e := callErr(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_level", "level": 101})}); e.Code != CodeInvalidValue {
		t.Fatalf("level 101: %s", e.Code)
	}
	if e := callErr(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "dance"})}); e.Code != CodeInvalidArgument {
		t.Fatalf("unknown op: %s", e.Code)
	}
}

func TestCompareBuildsReportsWhatChanged(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_unique", "name": "Kaom's Heart"})})
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "remove_gem", "group": 4, "gem": "Melee Splash"})})
	res := assertSize(t, s, "compare_builds", map[string]any{"a": h + "@0", "b": h}, capCompare)
	st := res["structure"].(map[string]any)
	items := st["items_changed"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["slot"] != "Body Armour" {
		t.Fatalf("items_changed = %v", items)
	}
	gems := st["gems_changed"].([]any)
	if len(gems) != 1 || gems[0].(map[string]any)["removed"].([]any)[0] != "Melee Splash" {
		t.Fatalf("gems_changed = %v", gems)
	}
	if res["a"] != h+"@0" || res["b"] != h+"@2" || len(res["stats"].([]any)) == 0 {
		t.Fatalf("compare = %v", res)
	}
	if e := callErr(t, s, "compare_builds", map[string]any{"a": h + "@x", "b": h}); e.Code != CodeInvalidArgument {
		t.Fatalf("bad ref: %s", e.Code)
	}
}
