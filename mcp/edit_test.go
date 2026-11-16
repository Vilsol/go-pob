package mcp

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
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
	item := dry["item"].([]any)[0].(map[string]any)
	if b, _ := json.Marshal(item); item["name"] != "Kaom's Heart, Glorious Plate" || item["text"] != nil || len(b) > 400 ||
		!slices.ContainsFunc(item["mods"].([]any), func(m any) bool { return strings.HasSuffix(m.(string), "to maximum Life") }) {
		t.Fatalf("dry-run item = %s", b)
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
	e = callErr(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Two-Stone Ring", "affixes": prefixes, "strict": true})})
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

// TestDeltaAlwaysShowsResistsWatchAndObjective dry-runs an edit that moves nothing and one that moves Life: every
// resistance comes back with its total and headroom, and MainDPS and watch keys either as a row or in unchanged.
func TestDeltaAlwaysShowsResistsWatchAndObjective(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	st := call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"FireResist", "FireResistTotal", "FireResistOverCap", "MissingFireResist", "Armour", mainDPS}})["stats"].(map[string]any)
	noop := assertSize(t, s, "edit_build", map[string]any{"handle": h, "dry_run": true, "watch": []string{"Armour"},
		"ops": ops(map[string]any{"op": "append_notes", "text": "x"})}, capWrite)
	if len(noop["delta"].([]any)) != 0 {
		t.Fatalf("a notes edit moved stats: %v", noop["delta"])
	}
	un := noop["unchanged"].(map[string]any)
	if un[mainDPS] != num(st, mainDPS) || un["Armour"] != num(st, "Armour") {
		t.Fatalf("unchanged = %v, stats %v", un, st)
	}
	fire := noop["resists"].(map[string]any)["Fire"].(map[string]any)
	if len(noop["resists"].(map[string]any)) != 4 || fire["resist"] != num(st, "FireResist") || fire["total"] != num(st, "FireResistTotal") ||
		fire["headroom"] != round(num(st, "FireResistOverCap")-num(st, "MissingFireResist")) || fire["headroom_change"] != nil {
		t.Fatalf("resists = %v, stats %v", noop["resists"], st)
	}

	moved := assertSize(t, s, "edit_tree", map[string]any{"handle": h, "dry_run": true, "watch": []string{"Armour"},
		"ops": ops(map[string]any{"op": "allocate", "node": "Heart of Oak"})}, capWrite)
	seen := map[string]bool{}
	for _, r := range moved["delta"].([]any) {
		seen[r.(map[string]any)["stat"].(string)] = true
	}
	movedUn, _ := moved["unchanged"].(map[string]any)
	for _, k := range []string{mainDPS, "Armour"} {
		_, same := movedUn[k]
		if seen[k] == same {
			t.Fatalf("%s: in delta %v, in unchanged %v", k, seen[k], same)
		}
	}
	if !seen["Life"] || len(moved["resists"].(map[string]any)) != 4 {
		t.Fatalf("allocate Heart of Oak: %v", moved)
	}
}

// TestCraftSkipsWhatTheBaseCannotRollAndResolvesIntents crafts Martyr Gloves, which roll no plain "+# to Armour": strict
// fails listing the armour affixes it can roll, the default skips it, and intent words pick per-base labels.
func TestCraftSkipsWhatTheBaseCannotRollAndResolvesIntents(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	craft := func(base string, affixes []string, strict bool) map[string]any {
		return call(t, s, "edit_items", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "craft", "base": base, "slot": "Gloves", "affixes": affixes, "strict": strict})})
	}
	e := callErr(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Martyr Gloves", "affixes": []string{"+# to Armour"}, "strict": true})})
	if e.Code != CodeAffixLimit || !slices.Contains(e.AllowedBase, "Prefix: +# to Armour/+# to maximum Life") || !strings.Contains(e.Hint, "free: 3 prefix, 3 suffix") {
		t.Fatalf("strict: %s %v %q", e.Code, e.AllowedBase, e.Hint)
	}

	res := craft("Martyr Gloves", []string{"+# to Armour", "life", "fire_res", "armour"}, false)
	skipped := res["skipped"].([]any)
	if len(skipped) != 1 || skipped[0].(map[string]any)["affix"] != "+# to Armour" || len(skipped[0].(map[string]any)["can_roll"].([]any)) == 0 {
		t.Fatalf("skipped = %v", res["skipped"])
	}
	note := res["applied"].([]any)[0].(string)
	for _, want := range []string{"Prefix: +# to maximum Life", "Suffix: +#% to Fire Resistance", "Prefix: #% increased Armour and Energy Shield"} {
		if !strings.Contains(note, want) {
			t.Fatalf("applied %q lacks %q", note, want)
		}
	}
	feet := call(t, s, "edit_items", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "craft", "base": "Titan Greaves", "affixes": []string{"armour"}})})
	if note := feet["applied"].([]any)[0].(string); !strings.HasSuffix(note, ": Prefix: +# to Armour") {
		t.Fatalf("armour on Titan Greaves: %q", note)
	}

	full := craft("Martyr Gloves", []string{"life", "mana", "+# to maximum Energy Shield/+# to maximum Life", "armour"}, false)
	if sk := full["skipped"].([]any); len(sk) != 1 || sk[0].(map[string]any)["reason"] != "no free prefix slot" {
		t.Fatalf("a fourth prefix: %v", full["skipped"])
	}
}

// TestSetGemsKeepsGemsThatStay changes one support: the gems that stay keep their level and quality, and the new
// gem takes the default level and the quality of the slot it takes over, as PoB's skills tab changes a slot's gem.
func TestSetGemsKeepsGemsThatStay(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch", "level": 90})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Fireball", "Elemental Focus Support", "Spell Echo"}},
		map[string]any{"op": "set_gem_level", "group": 1, "gem": "Fireball", "level": 21, "quality": 23},
		map[string]any{"op": "set_gem_level", "group": 1, "gem": "Spell Echo", "level": 18, "quality": 7},
	)})
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "set_gems", "group": 1, "gems": []string{"Fireball", "Spell Echo", "Controlled Destruction"}},
	)})
	groups := call(t, s, "get_build", map[string]any{"handle": h, "section": "skills"})["groups"].([]any)
	gems := groups[0].(map[string]any)["gems"].([]any)
	want := []struct {
		name           string
		level, quality float64
	}{{"Fireball", 21, 23}, {"Spell Echo", 18, 7}, {"Controlled Destruction", 20, 0}}
	if len(gems) != len(want) {
		t.Fatalf("gems = %v", gems)
	}
	for i, w := range want {
		g := gems[i].(map[string]any)
		if g["name"] != w.name || g["level"] != w.level || g["quality"] != w.quality {
			t.Fatalf("gem %d = %v, want %v", i, g, w)
		}
	}
}

// TestUnsocketedGroupsWarn adds a group without a slot: every write and get_build skills say it sits in no item and
// name the best free linked item, until a slot is set.
func TestUnsocketedGroupsWarn(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Marauder", "level": 80})["handle"].(string)
	res := assertSize(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Sunder", "Melee Physical Damage", "Brutality", "Fortify"}},
	)}, capWrite)
	first := func(res map[string]any) map[string]any {
		t.Helper()
		list, _ := res["unsocketed"].([]any)
		if len(list) != 1 {
			t.Fatalf("unsocketed = %v", res["unsocketed"])
		}
		return list[0].(map[string]any)
	}
	if u := first(res); u["group"] != float64(1) || u["best_slot"] != nil {
		t.Fatalf("no item yet: %v", u)
	}
	res = assertSize(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "craft", "slot": "Body Armour", "base": "Legion Plate", "affixes": []string{"life"}, "sockets": "R-R-R-R-R-R"},
		map[string]any{"op": "craft", "slot": "Helmet", "base": "Royal Burgonet", "affixes": []string{"life"}, "sockets": "R-R-R-R"},
	)}, capWrite)
	if u := first(res); u["best_slot"] != "Body Armour" || u["links"] != float64(6) {
		t.Fatalf("best free item: %v", u)
	}
	groups := call(t, s, "get_build", map[string]any{"handle": h, "section": "skills"})["groups"].([]any)
	if g := groups[0].(map[string]any); g["socketed"] != false || g["best_slot"] != "Body Armour" {
		t.Fatalf("skills section: %v", g)
	}
	res = call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_slot", "group": 1, "slot": "Body Armour"})})
	if res["unsocketed"] != nil {
		t.Fatalf("still unsocketed: %v", res)
	}
	res = call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Leap Slam", "Faster Attacks"}})})
	if u := first(res); u["group"] != float64(2) || u["best_slot"] != "Helmet" || u["links"] != float64(4) {
		t.Fatalf("body armour is taken, helmet is free: %v", u)
	}
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_gems", "group": 2, "slot": "Helmet", "gems": []string{"Leap Slam", "Faster Attacks"}})})
	groups = call(t, s, "get_build", map[string]any{"handle": h, "section": "skills"})["groups"].([]any)
	if g := groups[1].(map[string]any); g["slot"] != "Helmet" || g["socketed"] != nil {
		t.Fatalf("set_gems slot: %v", g)
	}
}

// TestLegacyGemNamesSayWhatToUse: an awakened support PoB lists as legacy is not offered, and the error names the
// gem to use instead.
func TestLegacyGemNamesSayWhatToUse(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch"})["handle"].(string)
	e := callErr(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Fireball", "Awakened Spell Echo"}})})
	if e.Code != CodeNotFound || !strings.Contains(e.Hint, "legacy") || len(e.DidYouMean) == 0 || e.DidYouMean[0] != "Spell Echo" {
		t.Fatalf("legacy gem: %s %q %v", e.Code, e.Hint, e.DidYouMean)
	}
}

// TestEditSkillsDescriptionNamesOfferedAwakenedSupports: the description's legacy note never covers an awakened
// support PoB still offers (Awakened Empower is one), since a model reads it as "do not use".
func TestEditSkillsDescriptionNamesOfferedAwakenedSupports(t *testing.T) {
	var desc string
	for _, d := range tools() {
		if d.name == "edit_skills" {
			desc = d.desc
		}
	}
	if !strings.Contains(desc, "legacy") {
		t.Fatalf("edit_skills description lost its legacy note: %q", desc)
	}
	offered := 0
	for _, g := range gems() {
		if !strings.HasPrefix(g.Name, "Awakened ") {
			continue
		}
		if _, err := resolveGem(g.Name); err != nil {
			continue
		}
		offered++
		if !strings.Contains(desc, g.Name) {
			t.Errorf("edit_skills calls awakened supports legacy but %s is offered", g.Name)
		}
	}
	if offered == 0 {
		t.Fatal("no awakened support is offered; the check saw nothing")
	}
}

// qualitySupports is a Fireball group whose three supports are quality 20.
func qualitySupports(t *testing.T, s *Server) string {
	t.Helper()
	h := call(t, s, "new_build", map[string]any{"class": "Witch", "level": 90})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Fireball", "Spell Echo", "Elemental Focus", "Controlled Destruction"}},
		map[string]any{"op": "set_gem_level", "group": 1, "gem": "Spell Echo", "quality": 20},
		map[string]any{"op": "set_gem_level", "group": 1, "gem": "Elemental Focus", "quality": 20},
		map[string]any{"op": "set_gem_level", "group": 1, "gem": "Controlled Destruction", "quality": 20},
	)})
	return h
}

func groupGems(t *testing.T, s *Server, h string, group int) map[string]map[string]any {
	t.Helper()
	groups := call(t, s, "get_build", map[string]any{"handle": h, "section": "skills"})["groups"].([]any)
	out := map[string]map[string]any{}
	for _, g := range groups[group-1].(map[string]any)["gems"].([]any) {
		out[g.(map[string]any)["name"].(string)] = g.(map[string]any)
	}
	return out
}

// TestSetGemsReplacingAGemKeepsItsQuality swaps a quality 20 support for another: PoB's skills tab keeps the slot's
// quality when the gem in it changes, and only a gem in a new slot takes the build's default quality.
func TestSetGemsReplacingAGemKeepsItsQuality(t *testing.T) {
	s := testServer(t)
	h := qualitySupports(t, s)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "set_gems", "group": 1, "gems": []string{"Fireball", "Spell Echo", "Concentrated Effect", "Controlled Destruction", "Added Fire Damage"}},
	)})
	gems := groupGems(t, s, h, 1)
	if g := gems["Concentrated Effect"]; g == nil || g["quality"] != 20.0 || g["level"] != 20.0 {
		t.Fatalf("Concentrated Effect replacing a quality 20 Elemental Focus = %v", g)
	}
	if g := gems["Added Fire Damage"]; g == nil || g["quality"] != 0.0 {
		t.Fatalf("Added Fire Damage in a new slot = %v", g)
	}
	if _, ok := gems["Elemental Focus"]; ok || len(gems) != 5 {
		t.Fatalf("gems = %v", gems)
	}
}

// TestGemEditsShowLevelAndQuality: what an edit or dry run applied names each gem it placed with its level/quality, as
// PoB's skill text does, so a quality 0 support shows.
func TestGemEditsShowLevelAndQuality(t *testing.T) {
	s := testServer(t)
	h := qualitySupports(t, s)
	res := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "ops": ops(
		map[string]any{"op": "set_gems", "group": 1, "gems": []string{"Fireball", "Spell Echo", "Concentrated Effect", "Controlled Destruction"}},
		map[string]any{"op": "add_gem", "group": 1, "gem": "Added Fire Damage"},
		map[string]any{"op": "add_group", "gems": []string{"Frostblink"}, "level": 5},
	)})
	applied := fmt.Sprint(res["applied"])
	for _, want := range []string{"Concentrated Effect 20/20", "Spell Echo 20/20", "Added Fire Damage 20/0", "Frostblink 5/0"} {
		if !strings.Contains(applied, want) {
			t.Fatalf("applied = %s, want %q", applied, want)
		}
	}
}

func TestCompareBuildsReportsMasteryEffectChanges(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_mastery", "node": 292, "effect": "+30 to maximum Life"})})
	st := assertSize(t, s, "compare_builds", map[string]any{"a": h + "@0", "b": h}, capCompare)["structure"].(map[string]any)
	if len(st["nodes_added"].([]any))+len(st["nodes_removed"].([]any)) != 0 {
		t.Fatalf("an effect swap moved nodes: %v", st)
	}
	list, _ := st["masteries_changed"].([]any)
	if len(list) != 1 {
		t.Fatalf("masteries_changed = %v", st["masteries_changed"])
	}
	row := list[0].(map[string]any)
	if row["id"] != float64(292) || row["b"] != "+30 to maximum Life" || row["a"] == "" || row["a"] == row["b"] {
		t.Fatalf("mastery row = %v", row)
	}
}

func TestEditTreeMasteryHintOnlyWithoutAnEffect(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	bare := call(t, s, "edit_tree", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "allocate", "node": 12382})})
	if hint, _ := bare["hint"].(string); !strings.Contains(hint, "set_mastery") {
		t.Fatalf("allocating a mastery alone gives no set_mastery hint: %v", bare["hint"])
	}
	for _, batch := range [][]any{
		ops(map[string]any{"op": "set_mastery", "node": 12382, "effect": "+30 to maximum Life"}),
		ops(map[string]any{"op": "allocate", "node": 12382}, map[string]any{"op": "set_mastery", "node": 12382, "effect": "+30 to maximum Life"}),
	} {
		res := call(t, s, "edit_tree", map[string]any{"handle": h, "dry_run": true, "ops": batch})
		if hint, _ := res["hint"].(string); strings.Contains(hint, "set_mastery") {
			t.Fatalf("batch %v picks an effect but still hints: %s", batch, hint)
		}
	}
}

// TestResistHeadroomIsTheMarginOverTheCap reads headroom as PoB's ResistOverCap minus MissingResist (CalcDefence.lua
// 627-629): positive over the cap, negative below it, and a resistance below its cap is flagged.
func TestResistHeadroomIsTheMarginOverTheCap(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch", "level": 90})["handle"].(string)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "craft", "slot": "Ring 1", "base": "Ruby Ring", "affixes": []string{"fire_res", "all_res"}},
		map[string]any{"op": "craft", "slot": "Ring 2", "base": "Ruby Ring", "affixes": []string{"fire_res", "all_res"}},
	)})
	res := call(t, s, "edit_build", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "append_notes", "text": "x"})})
	st := call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"FireResistOverCap", "MissingFireResist", "ColdResistOverCap", "MissingColdResist"}})["stats"].(map[string]any)
	resists := res["resists"].(map[string]any)
	fire, cold := resists["Fire"].(map[string]any), resists["Cold"].(map[string]any)
	if num(st, "FireResistOverCap") <= 0 || num(st, "MissingColdResist") <= 0 {
		t.Fatalf("fixture: fire must be over its cap and cold under it: %v", st)
	}
	if fire["headroom"] != num(st, "FireResistOverCap") || fire["under_cap"] != nil {
		t.Fatalf("fire over cap: %v, stats %v", fire, st)
	}
	if cold["headroom"] != -num(st, "MissingColdResist") || cold["under_cap"] != true {
		t.Fatalf("cold under cap: %v, stats %v", cold, st)
	}
	rank := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "Life", "limit": 1})
	if num(rank["base_headroom"], "Cold") != -num(st, "MissingColdResist") {
		t.Fatalf("base_headroom = %v, stats %v", rank["base_headroom"], st)
	}
}

// TestCompareBuildsMatchesGroupsByContent: removing one socket group shows that group removed and no other, and a
// gem change in a later group names it by its index in each build.
func TestCompareBuildsMatchesGroupsByContent(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "remove_group", "group": 2},
		map[string]any{"op": "remove_gem", "group": 4, "gem": "Melee Splash"})})
	st := call(t, s, "compare_builds", map[string]any{"a": h + "@0", "b": h})["structure"].(map[string]any)
	gems := st["gems_changed"].([]any)
	if len(gems) != 2 {
		t.Fatalf("gems_changed has %d rows, want the removed group and the edited one: %v", len(gems), gems)
	}
	removed, edited := gems[0].(map[string]any), gems[1].(map[string]any)
	if removed["group"] != float64(2) || removed["group_removed"] != true {
		t.Fatalf("removed group row = %v", removed)
	}
	if edited["group"] != float64(4) || edited["group_b"] != float64(3) || fmt.Sprint(edited["removed"]) != "[Melee Splash]" || len(edited["added"].([]any)) != 0 {
		t.Fatalf("edited group row = %v", edited)
	}
}
