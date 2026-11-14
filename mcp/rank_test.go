package mcp

import (
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
	best := supports[0]
	dry := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "add_gem", "group": 4, "gem": best["name"]})})
	for _, d := range dry["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == mainDPS && row["after"].(float64)-row["before"].(float64) != best["gain_added"].(float64) {
			t.Fatalf("%v ranked +%v MainDPS, adding it gives %v", best["name"], best["gain_added"], row)
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
		st := call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"Life", "TotalEHP", "Armour", "FireResistTotal", "FireResistMax", "ChaosResistTotal", "ChaosResistMax"}})
		call(t, s, "revert_build", map[string]any{"handle": h, "to_revision": 0})
		return st["stats"].(map[string]any)
	}
	check := func(kind string, row map[string]any, st map[string]any, keys ...string) {
		t.Helper()
		for _, k := range keys {
			if num(row["after"], k) != num(st, k) {
				t.Fatalf("%s %v: row says %s %v, committed %v", kind, row["name"], k, num(row["after"], k), num(st, k))
			}
		}
		hr := row["headroom"].(map[string]any)
		if len(hr) != 4 || hr["Fire"] != round(num(st, "FireResistTotal")-num(st, "FireResistMax")) ||
			hr["Chaos"] != round(num(st, "ChaosResistTotal")-num(st, "ChaosResistMax")) {
			t.Fatalf("%s %v: headroom %v, committed %v", kind, row["name"], hr, st)
		}
	}

	res := assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques", "slot": "Ring 1", "stat": "Life", "watch": []string{"Armour"}}, capRank)
	if res["base_headroom"] == nil || num(res["base_stats"], "Life") == 0 {
		t.Fatalf("no base view: %v", res)
	}
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
