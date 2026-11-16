package mcp

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
)

// TestRankRemoveNodesReportsLimitChanges: removing Ancestral Bond gains damage on this build but drops the totem limit,
// which the row must say.
func TestRankRemoveNodesReportsLimitChanges(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "import_build", map[string]any{"source": fixture("1.xml")})["handle"].(string)
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "41970"})})
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "MainDPS", "limit": 3})
	for _, r := range rows(res) {
		if r["id"] != float64(41970) {
			continue
		}
		if fmt.Sprint(r["also"]) != "map[ActiveTotemLimit:[2 1]]" {
			t.Fatalf("Ancestral Bond row: %v", r)
		}
		return
	}
	t.Fatalf("no Ancestral Bond row: %v", rows(res))
}

// TestRankSupportsOnAFullGroupRanksBestSwaps: group 6 fills its six-socket body armour, so a candidate can only replace
// a support; its row is the best of those swaps, as set_gems makes them, and offers no seventh-socket gain.
func TestRankSupportsOnAFullGroupRanksBestSwaps(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "import_build", map[string]any{"source": fixture("1.xml")})["handle"].(string)
	ranked := rows(assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "supports", "group": 6, "limit": 2}, capRank))
	gems := call(t, s, "get_build", map[string]any{"handle": h, "section": "skills"})["groups"].([]any)[5].(map[string]any)["gems"].([]any)
	names := make([]string, 0, len(gems))
	for _, g := range gems {
		names = append(names, g.(map[string]any)["name"].(string))
	}
	for _, r := range ranked {
		if r["gain_added"] != nil || !strings.Contains(fmt.Sprint(r["note"]), "full") {
			t.Fatalf("row on a full group: %v", r)
		}
		best, out := math.Inf(-1), ""
		for i := 1; i < len(names); i++ {
			list := slices.Clone(names)
			list[i] = r["name"].(string)
			dry := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "set_gems", "group": 6, "gems": list})})
			for _, d := range dry["delta"].([]any) {
				if row := d.(map[string]any); row["stat"] == mainDPS {
					if g := row["after"].(float64) - row["before"].(float64); g > best {
						best, out = g, names[i]
					}
				}
			}
		}
		if out != r["swaps_out"] || math.Abs(best-r["gain_swapped"].(float64)) > 1 {
			t.Fatalf("%v: ranked %v for %v, best set_gems swap %v for %v", r["name"], r["gain_swapped"], r["swaps_out"], best, out)
		}
	}
}

// TestRankOffenceDefenceRowsCarryBothPowers: PoB's Offence/Defence power is a DPS fraction and a defence score
// (CalcsTab:CalculateCombinedOffDefStat); a row shows both in percent and ranks by their sum per point.
func TestRankOffenceDefenceRowsCarryBothPowers(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "OffenceDefence", "max_points": 1, "watch": []string{"MainDPS"}})
	baseDPS := res["base_stats"].(map[string]any)["MainDPS"].(float64)
	list := rows(res)
	if len(list) == 0 {
		t.Fatal("no rows")
	}
	for i, r := range list {
		off, def := num(r, "offence_pct"), num(r, "defence_pct")
		if r["per_point"].(float64) <= 0 || math.Abs(r["gain"].(float64)-(off+def)) > 0.11 {
			t.Fatalf("row %d = %v", i, r)
		}
		if i > 0 && r["per_point"].(float64) > list[i-1]["per_point"].(float64) {
			t.Fatalf("rows not sorted by per_point: %v", list)
		}
		after := r["after"].(map[string]any)["MainDPS"].(float64)
		if want := (after/baseDPS - 1) * 100; math.Abs(off-want) > 0.11 {
			t.Fatalf("%v: offence_pct %v, MainDPS moves %.2f%%", r["name"], off, want)
		}
	}
}

// TestRankNodesOffersMasteryEffectSwitches: an allocated mastery can take another effect for no points; the ranking
// lists the switches that gain, each worth what set_mastery gives.
func TestRankNodesOffersMasteryEffectSwitches(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "MainDPS", "max_points": 1}, capRank)
	switches, _ := res["mastery_switches"].([]any)
	if len(switches) == 0 {
		t.Fatalf("no mastery switches: %v", keys(res))
	}
	for _, sw := range switches {
		r := sw.(map[string]any)
		dry := call(t, s, "edit_tree", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "set_mastery", "node": r["id"], "effect": fmt.Sprint(int64(r["effect"].(float64)))})})
		gained := 0.0
		for _, d := range dry["delta"].([]any) {
			if row := d.(map[string]any); row["stat"] == mainDPS {
				gained = row["after"].(float64) - row["before"].(float64)
			}
		}
		if gained <= 0 || math.Abs(gained-r["gain"].(float64)) > 1 {
			t.Fatalf("switch %v ranked +%v MainDPS, set_mastery gives %v", r, r["gain"], gained)
		}
	}
}

// TestRankSupportsWithAFreeSocketRanksAdds: group 6 with one support taken out leaves a free socket in its six-socket
// body armour, so rows rank by adding the support, the change their after values and columns show.
func TestRankSupportsWithAFreeSocketRanksAdds(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "import_build", map[string]any{"source": fixture("1.xml")})["handle"].(string)
	gems := call(t, s, "get_build", map[string]any{"handle": h, "section": "skills"})["groups"].([]any)[5].(map[string]any)["gems"].([]any)
	last := gems[len(gems)-1].(map[string]any)["name"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "remove_gem", "group": 6, "gem": last})})
	ranked := rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "supports", "group": 6, "limit": 5}))
	prev := math.Inf(1)
	for _, r := range ranked {
		added, _ := r["gain_added"].(float64)
		d, _ := r["d"].(map[string]any)
		dd, _ := d["dDPS"].(float64)
		if r["gain_added"] == nil || added > prev || math.Abs(dd-added) > 1 {
			t.Fatalf("row %v after %v: want rows by gain_added, their columns the add's", r, prev)
		}
		prev = added
	}
}
