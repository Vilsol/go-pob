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
