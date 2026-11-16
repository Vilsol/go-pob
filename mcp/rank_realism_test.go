package mcp

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Vilsol/go-pob/calculator"
)

// unforced forks h with every forced combat condition the ranking named unset.
func unforced(t *testing.T, s *Server, h string, res map[string]any) string {
	t.Helper()
	f := call(t, s, "new_build", map[string]any{"class": "Marauder", "from_handle": h})["handle"].(string)
	forced := res["forced_conditions"].([]any)
	list := make([]map[string]any, 0, len(forced))
	for _, c := range forced {
		list = append(list, map[string]any{"op": "unset_config", "var": c})
	}
	call(t, s, "edit_build", map[string]any{"handle": f, "ops": ops(list...)})
	return f
}

func dryDelta(t *testing.T, s *Server, tool, h, key string, op map[string]any) float64 {
	t.Helper()
	dry := call(t, s, tool, map[string]any{"handle": h, "dry_run": true, "watch": []string{key}, "ops": ops(op)})
	for _, d := range dry["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == key {
			return row["after"].(float64) - row["before"].(float64)
		}
	}
	return 0
}

func flagged(r map[string]any, prefix, name string) bool {
	flags, _ := r["flags"].([]any)
	return slices.ContainsFunc(flags, func(f any) bool {
		s := fmt.Sprint(f)
		return strings.HasPrefix(s, prefix) && strings.Contains(s, name)
	})
}

// TestRankFlagsGainsFromForcedConditions: the fixture assumes the enemy is Bleeding, so a mastery that only works
// against Bleeding enemies gains on the assumption; its row names the condition, and worst_case re-scores it with
// every forced condition off, as unsetting them and allocating it gives.
func TestRankFlagsGainsFromForcedConditions(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "nodes", "stat": "MainDPS", "node_types": []string{"mastery"}, "worst_case": true, "limit": 25}, 3*capRank)
	var row map[string]any
	for _, r := range rows(res) {
		if r["id"] == float64(40170) {
			row = r
		}
	}
	if row == nil || !flagged(row, "config: ", "conditionEnemyBleeding") {
		t.Fatalf("Bleeding Mastery row: %v", row)
	}
	f := unforced(t, s, h, res)
	want := dryDelta(t, s, "edit_tree", f, mainDPS, map[string]any{"op": "set_mastery", "node": "40170", "effect": fmt.Sprint(int64(row["effect"].(float64)))})
	if wg := row["worst_gain"].(float64); math.Abs(wg-want) > math.Max(1, 0.001*math.Abs(want)) || wg > row["gain"].(float64)/2 {
		t.Fatalf("worst_gain %v, gain %v; with the conditions off the mastery gains %v", wg, row["gain"], want)
	}
}

// TestRankFlagsLossesFromForcedConditions: with Fortified assumed, Rampart's removal costs EHP only through the
// assumption; the row names it and its worst-case loss is what removing it costs with the conditions off.
func TestRankFlagsLossesFromForcedConditions(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "TotalEHP", "limit": 25, "worst_case": true,
		"keep": []any{map[string]any{"stat": "Armour", "op": "<=", "value": 0.9, "relative": true}}})
	row := removeRow(t, res, 1340)
	if row["loss"].(float64) <= 0 || !flagged(row, "config: ", "buffFortification") {
		t.Fatalf("Rampart row: %v", row)
	}
	f := unforced(t, s, h, res)
	want := -dryDelta(t, s, "edit_tree", f, "TotalEHP", map[string]any{"op": "deallocate", "node": "1340"})
	if math.Abs(row["worst_loss"].(float64)-want) > 1 {
		t.Fatalf("worst_loss %v; with the conditions off removing Rampart loses %v", row["worst_loss"], want)
	}
}

// TestRankRowsCarryChangeColumns: every row carries the default change columns, each what applying the row gives.
func TestRankRowsCarryChangeColumns(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := assertSize(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques", "slot": "Ring 1", "stat": "MainDPS"}, capRank)
	list := rows(res)
	if len(list) == 0 {
		t.Fatal("no rows")
	}
	for _, r := range list[:min(3, len(list))] {
		d, ok := r["d"].(map[string]any)
		if !ok {
			t.Fatalf("row without change columns: %v", r)
		}
		for col, key := range map[string]string{"dDPS": mainDPS, "dEHP": "TotalEHP", "dLife": "Life"} {
			want := dryDelta(t, s, "edit_items", h, key, map[string]any{"op": "equip_unique", "name": r["name"], "slot": "Ring 1"})
			if got, _ := d[col].(float64); math.Abs(got-want) > math.Max(1, 0.001*math.Abs(want)) {
				t.Fatalf("%v: %s %v, equipping it moves %s by %v", r["name"], col, got, key, want)
			}
		}
	}
}

// TestRankUniquesRanksCleanGainsFirst: a two-handed weapon in Weapon 1 drops the fixture's off-hand and the gems it
// sockets; such a row says so and ranks after every row without that cost, though its raw gain is the largest.
func TestRankUniquesRanksCleanGainsFirst(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	list := rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques", "slot": "Weapon 1", "stat": "MainDPS", "limit": 25}))
	dirty := 0
	for _, r := range list {
		if flagged(r, "removes Weapon 2", "") || flagged(r, "sockets: ", "") {
			dirty++
		} else if dirty > 0 {
			t.Fatalf("%v ranks after %d rows that drop the off-hand or overfill sockets: %v", r["name"], dirty, list)
		}
	}
	if len(list) == 0 || dirty == len(list) {
		t.Fatalf("no clean row: %v", list)
	}
	dry := call(t, s, "edit_items", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "equip_unique", "name": "Replica Wings of Entropy", "slot": "Weapon 1"})})
	for _, d := range dry["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == mainDPS && row["after"].(float64)-row["before"].(float64) <= list[0]["gain"].(float64) {
			t.Fatalf("the two-hander gains %v, no more than the top row %v", row["after"].(float64)-row["before"].(float64), list[0])
		}
	}
}

// TestWorstGainNotAboveGain: with the enemy configured as shocked for 30%, Vessel of Vinktar's shock adds nothing in
// PoB; with the condition questioned the flask shocks the enemy itself, a larger share of the questioned build than its
// gain is of the build. Its worst_gain takes the gain's share, flagged.
func TestWorstGainNotAboveGain(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "conditionEnemyShocked", "value": true},
		map[string]any{"op": "set_config", "var": "conditionShockEffect", "value": 30})})
	hd, _ := s.get(h)
	b, _ := hd.parseRev(hd.cur)
	env, _ := hd.envAt(hd.cur)
	text, _ := calculator.ItemDBText(calculator.ItemDBUnique, "Vessel of Vinktar, Topaz Flask")
	ch := flaskChange(text, "Flask 1")
	gain := statOf(evalChange(b, ch), mainDPS) - statOf(env, mainDPS)
	row := map[string]any{"gain": gain, changeKey: ch}
	rz := newRealism(b, env, mainDPS, time.Now().Add(time.Minute))
	rz.bases = true
	rz.annotate([]map[string]any{row}, 1, true)
	w, _ := row["worst_gain"].(float64)
	wb, _ := row[worstBaseKey].(map[string]float64)
	if wb["stat"] <= 0 || w/wb["stat"] > gain/statOf(env, mainDPS)+0.001 {
		t.Fatalf("worst_gain %v of %v, gain %v of %v, flags %v", w, wb["stat"], gain, statOf(env, mainDPS), row["flags"])
	}
}
