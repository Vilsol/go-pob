package mcp

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"
)

// mainDPSDelta is what edit_tree's dry run says a batch of ops does to MainDPS.
func mainDPSDelta(t *testing.T, s *Server, h string, list ...map[string]any) float64 {
	t.Helper()
	dry := call(t, s, "edit_tree", map[string]any{"handle": h, "dry_run": true, "ops": ops(list...)})
	for _, d := range dry["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == mainDPS {
			return row["after"].(float64) - row["before"].(float64)
		}
	}
	return 0
}

func removeRow(t *testing.T, res map[string]any, id float64) map[string]any {
	t.Helper()
	for _, r := range rows(res) {
		if r["id"] == id {
			return r
		}
	}
	t.Fatalf("no row for %v: %v", id, rows(res))
	return nil
}

// TestRankRemoveNodesCountsGrantedSkills: Unwavering Crusade grants the Elemental Relic, whose group PoB builds only in
// a full calculation (CalcSetup.lua:1452-1505); removing the node loses what deallocating it loses.
func TestRankRemoveNodesCountsGrantedSkills(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar", "ascendancy": "Guardian"})["handle"].(string)
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "19641"})})
	want := -mainDPSDelta(t, s, h, map[string]any{"op": "deallocate", "node": "19641"})
	if want <= 0 {
		t.Fatalf("deallocating Unwavering Crusade loses %v MainDPS", want)
	}
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "MainDPS", "limit": 25})
	r := removeRow(t, res, 19641)
	if math.Abs(r["loss"].(float64)-want) > 0.11 {
		t.Fatalf("Unwavering Crusade ranked as losing %v MainDPS, deallocating it loses %v", r["loss"], want)
	}
}

// TestRankRemoveNodesFreesNoPassivePointsForAscendancy: ascendancy points are their own pool, so removing an
// ascendancy node frees no passive points.
func TestRankRemoveNodesFreesNoPassivePointsForAscendancy(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "import_build", map[string]any{"source": fixture("1.xml")})["handle"].(string)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "MainDPS", "limit": 25})
	r := removeRow(t, res, 51101)
	if r["type"] != "ascendancy" || r["points_freed"] != float64(0) || r["ascendancy_points_freed"] != float64(1) {
		t.Fatalf("Nature's Adrenaline row: %v", r)
	}
}

// TestRankRemoveNodesScoresOffenceDefence: a removal's Offence/Defence loss is PoB's combined power of adding the node
// back (CalcsTab.lua:748-758): a DPS fraction and a defence score, both in percent, ranked by their sum per point.
func TestRankRemoveNodesScoresOffenceDefence(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "OffenceDefence", "limit": 25, "watch": []string{"MainDPS"}})
	baseDPS := res["base_stats"].(map[string]any)["MainDPS"].(float64)
	list := rows(res)
	nonzero := 0
	for i, r := range list {
		off, def := num(r, "offence_pct"), num(r, "defence_pct")
		if math.Abs(r["loss"].(float64)-(off+def)) > 1 {
			t.Fatalf("row %d = %v", i, r)
		}
		if i > 0 && r["loss_per_point"].(float64) < list[i-1]["loss_per_point"].(float64) {
			t.Fatalf("rows not sorted by loss_per_point: %v", list)
		}
		after := r["after"].(map[string]any)["MainDPS"].(float64)
		if after > 0 {
			if want := (baseDPS/after - 1) * 100; math.Abs(off-want) > math.Max(0.11, 0.006*math.Abs(want)) {
				t.Fatalf("%v: offence_pct %v, MainDPS falls to %v of %v", r["name"], off, after, baseDPS)
			}
		}
		if r["loss"].(float64) != 0 {
			nonzero++
		}
	}
	if nonzero == 0 {
		t.Fatalf("every Offence/Defence removal scores 0: %v", list)
	}
}

// totemBuild is a Templar whose main skill is a Freezing Pulse Spell Totem, with Ancestral Bond allocated.
func totemBuild(t *testing.T, s *Server) string {
	t.Helper()
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse", "Spell Totem"}}, map[string]any{"op": "set_main", "group": 1})})
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "41970"})})
	return h
}

// TestRankRemoveNodesCountsLostTotems: PoB's totem DPS is per totem, so a removal that lowers the totem limit loses
// that share of every summoned totem's damage (CalcOffence.lua:4451-4453 counts the summoned totems the same way).
func TestRankRemoveNodesCountsLostTotems(t *testing.T) {
	s := testServer(t)
	h := totemBuild(t, s)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "MainDPS", "limit": 25})
	r := removeRow(t, res, 41970)
	also := r["also"].(map[string]any)["ActiveTotemLimit"].([]any)
	before, after := also[0].(float64), also[1].(float64)
	if after >= before {
		t.Fatalf("Ancestral Bond row: %v", r)
	}
	base := res["base"].(float64)
	dps := r["after"].(map[string]any)["MainDPS"].(float64)
	if want := base - dps*after/before; want <= 0 || math.Abs(r["loss"].(float64)-want) > math.Max(1, 0.001*want) {
		t.Fatalf("Ancestral Bond loss %v, the lost totem is worth %v: %v", r["loss"], want, r)
	}
}

// TestRankRemoveNodesCountsLostTotemsOverConfig: a summoned-totems config above the limit still loses the totem a
// lower limit takes away; no more totems than the limit can be out.
func TestRankRemoveNodesCountsLostTotemsOverConfig(t *testing.T) {
	s := testServer(t)
	h := totemBuild(t, s)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "TotemsSummoned", "value": 5})})
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "MainDPS", "limit": 25})
	r := removeRow(t, res, 41970)
	also := r["also"].(map[string]any)["ActiveTotemLimit"].([]any)
	before, after := also[0].(float64), also[1].(float64)
	base := res["base"].(float64)
	dps := r["after"].(map[string]any)["MainDPS"].(float64)
	if want := base - dps*after/before; want <= 0 || math.Abs(r["loss"].(float64)-want) > math.Max(1, 0.001*want) {
		t.Fatalf("Ancestral Bond loss %v with 5 totems configured, the lost totem is worth %v: %v", r["loss"], want, r)
	}
}

// TestRankRemoveNodesKeepScoresWhatTheBuildHas: with Servant of Decay worn, the fast calculation leaves out the Void
// Spawns' share of TotalEHP, so a relative TotalEHP keep scored from it dropped every removal; a removal of a node
// that gives no defence keeps TotalEHP and is kept, its after value the build's.
func TestRankRemoveNodesKeepScoresWhatTheBuildHas(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_unique", "name": "Servant of Decay", "slot": "Body Armour"})})
	base := call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"TotalEHP"}})
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "limit": 25,
		"keep": []any{map[string]any{"stat": "TotalEHP", "op": ">=", "value": 0.999, "relative": true}}})
	got := rows(res)
	if len(got) == 0 {
		t.Fatalf("every removal dropped (%v of %v); base %v", res["dropped"], res["considered"], base)
	}
}

// TestRankRemoveNodesFullDPSCountsTheLimit: ranking FullDPS with Full DPS counting the spell totem twice, removing
// Ancestral Bond takes the limit to 1; PoB's Full DPS keeps the stored count and loses nothing, the loss counts the
// totem the limit takes away, and a flag says so.
func TestRankRemoveNodesFullDPSCountsTheLimit(t *testing.T) {
	s := testServer(t)
	h := totemBuild(t, s)
	xml := call(t, s, "export_build", map[string]any{"handle": h, "format": "xml"})["xml"].(string)
	xml = strings.Replace(xml, "<Skill ", `<Skill includeInFullDPS="true" `, 1)
	xml = strings.Replace(xml, `count="1"`, `count="2"`, 1)
	h = call(t, s, "import_build", map[string]any{"source": xml})["handle"].(string)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "FullDPS", "limit": 25})
	if res["base"].(float64) <= 0 {
		t.Fatalf("no FullDPS: %v", res)
	}
	r := removeRow(t, res, 41970)
	if loss := r["loss"].(float64); loss < 0.4*res["base"].(float64) || !strings.Contains(fmt.Sprint(r["flags"]), "loss counts") {
		t.Fatalf("Ancestral Bond row %v, base %v", r, res["base"])
	}
}

// TestRankRemoveNodesFullCalculation: removing a cluster jewel's socket loses the cluster's life, as deallocating it
// does; the fast calculation keeps the cluster's nodes.
func TestRankRemoveNodesFullCalculation(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "import_build", map[string]any{"source": fixture("7.xml")})["handle"].(string)
	dry := call(t, s, "edit_tree", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "deallocate", "node": "7960"})})
	want := 0.0
	for _, d := range dry["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == "Life" {
			want = row["before"].(float64) - row["after"].(float64)
		}
	}
	hd, _ := s.get(h)
	b, _ := hd.parseRev(hd.cur)
	env, _ := hd.envAt(hd.cur)
	ps, _ := powerStat(env, "Life")
	got, _, _, _ := rankRemoveNodes(b, env, "Life", ps, nil, time.Now(), nil)
	i := slices.IndexFunc(got, func(r map[string]any) bool { return r["id"] == int64(7960) })
	if i < 0 || want <= 0 || got[i]["loss"].(float64) != want {
		t.Fatalf("Large Jewel Socket row %d of %d, deallocating it loses %v", i, len(got), want)
	}
	if got[i]["loss"].(float64) != want {
		t.Fatalf("Large Jewel Socket loss %v, deallocating it loses %v", got[i]["loss"], want)
	}
}

// TestRankRemoveNodesUnusable: a removal that leaves the auras reserving more mana than the build has is unusable and
// ranks after the removals the character can use.
func TestRankRemoveNodesUnusable(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": "7594"})})
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Hatred"}, "level": 20},
		map[string]any{"op": "add_group", "gems": []string{"Anger"}, "level": 20},
		map[string]any{"op": "add_group", "gems": []string{"Clarity"}, "level": 1})})
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "Life", "limit": 25})
	got := rows(res)
	r := removeRow(t, res, 7594)
	at := slices.IndexFunc(got, func(x map[string]any) bool { return x["id"] == r["id"] })
	if u, _ := r["unusable"].(string); !strings.Contains(u, "Mana") || slices.ContainsFunc(got[at:], func(x map[string]any) bool { return x["unusable"] == nil && x["big_loss"] == nil }) {
		t.Fatalf("Reservation Efficiency row %v; rows %v", r, got)
	}
}
