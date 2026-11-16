package mcp

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func rows(res map[string]any) []map[string]any {
	list := res["results"].([]any)
	out := make([]map[string]any, 0, len(list))
	for _, r := range list {
		out = append(out, r.(map[string]any))
	}
	return out
}

func TestRankCandidatesNodesByPerPoint(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "Life", "max_points": 3}, capRank)
	list := rows(res)
	if len(list) == 0 || len(list) > 8 || res["base"].(float64) <= 0 {
		t.Fatalf("nodes = %v", res)
	}
	for i, r := range list {
		if r["points"].(float64) > 3 || r["gain"].(float64) <= 0 || len(r["path"].([]any)) != int(r["points"].(float64)) {
			t.Fatalf("row %d = %v", i, r)
		}
		if i > 0 && r["per_point"].(float64) > list[i-1]["per_point"].(float64) {
			t.Fatalf("rows not sorted by per_point: %v", list)
		}
	}
	top := list[0]
	dry := call(t, s, "edit_tree", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "allocate", "node": top["id"]})})
	gained := 0.0
	for _, d := range dry["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == "Life" {
			gained = row["after"].(float64) - row["before"].(float64)
		}
	}
	if gained != top["gain"].(float64) {
		t.Fatalf("ranked %v gains %v Life, allocating it gains %v", top["name"], top["gain"], gained)
	}

	kept := call(t, s, "import_build", map[string]any{"source": fixture("3.xml"), "convert_tree": false})["handle"]
	if e := callErr(t, s, "rank_candidates", map[string]any{"handle": kept, "kind": "nodes"}); e.Code != CodeTreeNotLatest {
		t.Fatalf("unconverted tree: %s", e.Code)
	}
	if e := callErr(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "Lief"}); e.Code != CodeUnknownStat || len(e.DidYouMean) == 0 {
		t.Fatalf("unknown stat: %s %v", e.Code, e.DidYouMean)
	}
}

func TestRankCandidatesRemoveSupportsAndUniques(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	removal := rows(assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "Life"}, capRank))
	if len(removal) == 0 || removal[0]["points_freed"].(float64) < 1 {
		t.Fatalf("remove_nodes = %v", removal)
	}
	for i := 1; i < len(removal); i++ {
		if removal[i]["loss_per_point"].(float64) < removal[i-1]["loss_per_point"].(float64) {
			t.Fatalf("remove_nodes not sorted: %v", removal)
		}
	}

	supports := rows(assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "supports"}, capRank))
	if len(supports) == 0 || supports[0]["swaps_out"] == nil {
		t.Fatalf("supports = %v", supports)
	}
	// group 4 fills its six sockets, so a support can only be swapped in
	best := supports[0]
	gems := call(t, s, "get_build", map[string]any{"handle": h, "section": "skills"})["groups"].([]any)[3].(map[string]any)["gems"].([]any)
	list := make([]string, 0, len(gems))
	for _, g := range gems {
		n := g.(map[string]any)["name"].(string)
		if n == best["swaps_out"] {
			n = best["name"].(string)
		}
		list = append(list, n)
	}
	dry := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "set_gems", "group": 4, "gems": list})})
	for _, d := range dry["delta"].([]any) {
		// before and after are rounded apart from the gain, so they may differ from it by one
		if row := d.(map[string]any); row["stat"] == mainDPS && math.Abs(row["after"].(float64)-row["before"].(float64)-best["gain_swapped"].(float64)) > 1 {
			t.Fatalf("%v ranked +%v MainDPS, swapping it in gives %v", best["name"], best["gain_swapped"], row)
		}
	}

	uniques := rows(assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques", "slot": "Ring 1", "stat": "Life"}, capRank))
	if len(uniques) == 0 {
		t.Fatal("no ring uniques ranked")
	}
	if e := callErr(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques"}); e.Code != CodeInvalidArgument {
		t.Fatalf("uniques without slot: %s", e.Code)
	}
}

func num(m any, key string) float64 {
	v, _ := m.(map[string]any)[key].(float64)
	return v
}

// TestRankRowsCarryWatchedStatsAndHeadroom checks each row's after-stats and resistance headroom against the build
// with that candidate committed.
func TestRankRowsCarryWatchedStatsAndHeadroom(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	committed := func(ops []any, tool string) map[string]any {
		call(t, s, tool, map[string]any{"handle": h, "ops": ops})
		st := call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"Life", "TotalEHP", "Armour", "FireResistOverCap", "MissingFireResist", "ChaosResistOverCap", "MissingChaosResist"}})
		call(t, s, "revert_build", map[string]any{"handle": h, "to_revision": 0})
		return st["stats"].(map[string]any)
	}
	var base any
	check := func(kind string, row map[string]any, st map[string]any, keys ...string) {
		t.Helper()
		for _, k := range keys {
			if num(row["after"], k) != num(st, k) {
				t.Fatalf("%s %v: row says %s %v, committed %v", kind, row["name"], k, num(row["after"], k), num(st, k))
			}
		}
		// a row leaves out headroom equal to the base build's
		hr, _ := row["headroom"].(map[string]any)
		if hr == nil {
			hr = base.(map[string]any)
		} else if reflect.DeepEqual(hr, base) {
			t.Fatalf("%s %v: headroom repeats base_headroom", kind, row["name"])
		}
		if len(hr) != 4 || hr["Fire"] != round(num(st, "FireResistOverCap")-num(st, "MissingFireResist")) ||
			hr["Chaos"] != round(num(st, "ChaosResistOverCap")-num(st, "MissingChaosResist")) {
			t.Fatalf("%s %v: headroom %v, committed %v", kind, row["name"], hr, st)
		}
	}

	res := assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques", "slot": "Ring 1", "stat": "Life", "watch": []string{"Armour"}}, capRank)
	if res["base_headroom"] == nil || num(res["base_stats"], "Life") == 0 {
		t.Fatalf("no base view: %v", res)
	}
	base = res["base_headroom"]
	top := rows(res)[0]
	check("uniques", top, committed(ops(map[string]any{"op": "equip_unique", "name": top["name"], "slot": "Ring 1"}), "edit_items"), "Life", "TotalEHP", "Armour")

	node := rows(assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "Life", "max_points": 3}, capRank))[0]
	check("nodes", node, committed(ops(map[string]any{"op": "allocate", "node": node["id"]}), "edit_tree"), "Life", "TotalEHP")

	gone := rows(assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "Life"}, capRank))[0]
	check("remove_nodes", gone, committed(ops(map[string]any{"op": "deallocate", "node": gone["id"]}), "edit_tree"), "Life", "TotalEHP")
}

// TestRankUniquesSkipsEquippedAndCountsSockets equips the best-ranked ring in the other ring slot, after which it
// must no longer be ranked; body armour rows carry socket and link counts.
func TestRankUniquesSkipsEquippedAndCountsSockets(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	ranking := func() []map[string]any {
		return rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques", "slot": "Ring 1", "stat": "Life", "limit": 25}))
	}
	best := ranking()[0]["name"].(string)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_unique", "name": best, "slot": "Ring 2"})})
	for _, r := range ranking() {
		if r["name"] == best {
			t.Fatalf("%s is equipped in Ring 2 but still ranked for Ring 1", best)
		}
	}
	body := rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques", "slot": "Body Armour", "stat": "Life", "limit": 25}))
	sawSix := false
	for _, r := range body {
		if r["sockets"] == nil || r["links"].(float64) > r["sockets"].(float64) {
			t.Fatalf("row %v has no socket count", r)
		}
		sawSix = sawSix || r["sockets"] == float64(6)
	}
	if !sawSix {
		t.Fatalf("no body armour with 6 sockets in %d rows", len(body))
	}
}

// TestRankMasteryRowsUseBestEffect ranks masteries: one row per mastery, carrying its best effect, whose gain and
// after-stats match allocating the mastery with that effect.
func TestRankMasteryRowsUseBestEffect(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "Life", "node_types": []string{"mastery"}, "max_points": 4, "limit": 25})
	list := rows(res)
	if len(list) == 0 {
		t.Fatalf("no masteries ranked: %v", res)
	}
	seen := map[float64]bool{}
	for _, r := range list {
		if seen[r["id"].(float64)] {
			t.Fatalf("mastery %v ranked twice: %v", r["id"], list)
		}
		seen[r["id"].(float64)] = true
		if r["effect"] == nil {
			t.Fatalf("mastery row without effect: %v", r)
		}
	}
	top := list[0]
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_mastery", "node": top["id"], "effect": fmt.Sprint(top["effect"])})})
	life := stat(call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"Life"}}), "Life")
	if num(top["after"], "Life") != life || res["base"].(float64)+top["gain"].(float64) != life {
		t.Fatalf("%v: ranked gain %v after %v from %v, committing it gives Life %v", top["name"], top["gain"], num(top["after"], "Life"), res["base"], life)
	}
}

// TestRankRemoveNodesKeepsConstraints ranks removals that keep Life within 1% of the current value, most points
// freed first, and says how many removals it dropped for breaking it.
func TestRankRemoveNodesKeepsConstraints(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	life := stat(call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"Life"}}), "Life")
	res := assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "MainDPS",
		"keep": []any{map[string]any{"stat": "Life", "op": ">=", "value": 0.99, "relative": true}}}, capRank)
	list := rows(res)
	if len(list) == 0 || num(res, "dropped") == 0 {
		t.Fatalf("remove_nodes keep = %v", res)
	}
	for i, r := range list {
		if num(r["after"], "Life") < 0.99*life {
			t.Fatalf("row %d breaks Life >= %v: %v", i, 0.99*life, r)
		}
		if i > 0 && r["points_freed"].(float64) > list[i-1]["points_freed"].(float64) {
			t.Fatalf("rows not sorted by points_freed: %v", list)
		}
	}
	if e := callErr(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes",
		"keep": []any{map[string]any{"stat": "Lief", "op": ">=", "value": 1}}}); e.Code != CodeUnknownStat {
		t.Fatalf("unknown keep stat: %s", e.Code)
	}
}

// TestSupportSwapsAgreeWithSetGems ranks supports for a group of quality 20 supports: a swap is evaluated as set_gems
// makes it, the new gem keeping the swapped one's quality (Fire Penetration's quality adds fire damage), and optimize
// applies its pick the same way.
func TestSupportSwapsAgreeWithSetGems(t *testing.T) {
	s := testServer(t)
	h := qualitySupports(t, s)
	ranked := rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "supports", "group": 1}))
	var best map[string]any
	for _, r := range ranked {
		if r["name"] == "Fire Penetration" && r["gain_swapped"] != nil {
			best = r
			break
		}
	}
	if best == nil {
		t.Fatalf("Fire Penetration is not ranked as a swap: %v", ranked)
	}
	list := make([]string, 0, 4)
	for _, n := range []string{"Fireball", "Spell Echo", "Elemental Focus", "Controlled Destruction"} {
		if n == best["swaps_out"] {
			n = best["name"].(string)
		}
		list = append(list, n)
	}
	dry := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "set_gems", "group": 1, "gems": list})})
	found := false
	for _, d := range dry["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == mainDPS {
			found = true
			if math.Abs(row["after"].(float64)-row["before"].(float64)-best["gain_swapped"].(float64)) > 1 {
				t.Fatalf("%v for %v ranked %+v MainDPS, set_gems gives %v", best["name"], best["swaps_out"], best["gain_swapped"], row)
			}
		}
	}
	if !found {
		t.Fatalf("no MainDPS delta: %v", dry["delta"])
	}

	call(t, s, "optimize", map[string]any{"handle": h, "apply": true, "objective": map[string]any{"stat": mainDPS},
		"slots": []any{map[string]any{"kind": "supports", "group": 1, "replace": []string{best["swaps_out"].(string)}, "keep_current": false,
			"candidates": []string{best["name"].(string)}}}})
	if g := groupGems(t, s, h, 1)[best["name"].(string)]; g == nil || g["quality"] != 20.0 {
		t.Fatalf("optimize put %v in place of a quality 20 %v: %v", best["name"], best["swaps_out"], g)
	}
}
