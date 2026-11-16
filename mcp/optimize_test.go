package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func lifeOf(t *testing.T, s *Server, h string) float64 {
	t.Helper()
	return stat(call(t, s, "get_stats", map[string]any{"handle": h, "stats": []string{"Life"}}), "Life")
}

// TestOptimizeExhaustiveIsOptimal scores every combination of a 4-ring x 4-node-subset space through the edit
// tools and checks the exhaustive search's first result is the best of them.
func TestOptimizeExhaustiveIsOptimal(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	rings := []string{"Kalandra's Touch", "Mark of the Elder", "Zana's Ingenuity"}
	nodes := []string{"Heart of Oak", "Blood Siphon"}
	res := assertSize(t, s, "optimize", map[string]any{"handle": h,
		"slots": []any{
			map[string]any{"kind": "item", "slot": "Ring 1", "uniques": rings},
			map[string]any{"kind": "nodes", "candidates": nodes},
		},
		"objective": map[string]any{"stat": "Life"},
	}, capOptimize)
	space := res["space"].(map[string]any)
	if space["method"] != "exhaustive" || space["size"] != float64(16) || res["proven_optimal"] != true {
		t.Fatalf("space = %v proven = %v", space, res["proven_optimal"])
	}
	if used := res["evaluations"].(map[string]any)["used"]; used != float64(16) {
		t.Fatalf("used %v evaluations for 16 combinations", used)
	}

	best := 0.0
	ringChoices := append([]string{""}, rings...)
	subsets := [][]string{{}, {nodes[0]}, {nodes[1]}, nodes}
	for _, ring := range ringChoices {
		for _, sub := range subsets {
			if ring != "" {
				call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_unique", "name": ring, "slot": "Ring 1"})})
			}
			for _, n := range sub {
				call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "allocate", "node": n})})
			}
			best = max(best, lifeOf(t, s, h))
			call(t, s, "revert_build", map[string]any{"handle": h, "to_revision": 0})
		}
	}
	top := rows(res)[0]
	var after float64
	for _, d := range top["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == "Life" {
			after = row["after"].(float64)
		}
	}
	if after != best {
		t.Fatalf("optimize's best Life %v, brute force %v (%v)", after, best, top["choices"])
	}

	applied := call(t, s, "optimize", map[string]any{"handle": h,
		"slots":     []any{map[string]any{"kind": "item", "slot": "Ring 1", "uniques": rings}, map[string]any{"kind": "nodes", "candidates": nodes}},
		"objective": map[string]any{"stat": "Life"}, "apply": true,
	})
	if applied["applied"].(map[string]any)["revision"] == nil || lifeOf(t, s, h) != best {
		t.Fatalf("apply left Life %v, best %v", lifeOf(t, s, h), best)
	}
}

// supportSpace is C(9,5)=126 support sets x 5 ring choices = 630 combinations, one sweep 131 evaluations.
func supportSpace(h string, seed int, budget int) map[string]any {
	return map[string]any{"handle": h, "seed": seed, "budget": budget,
		"slots": []any{
			map[string]any{"kind": "supports", "k": 5, "candidates": []string{"Fortify", "Ruthless", "Close Combat", "Multistrike"}},
			map[string]any{"kind": "item", "slot": "Ring 1", "uniques": []string{"Kalandra's Touch", "Mark of the Elder", "Zana's Ingenuity", "Andvarius"}},
		},
		"objective":   map[string]any{"stat": "MainDPS"},
		"constraints": []any{map[string]any{"stat": "Life", "op": ">=", "value": 0.99, "relative": true}},
	}
}

func TestOptimizeCoordinateAscentIsDeterministicAndBudgeted(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	strip := func(m map[string]any) string {
		delete(m, "elapsed_ms")
		b, _ := json.Marshal(m)
		return string(b)
	}
	first := call(t, s, "optimize", supportSpace(h, 7, 300))
	space := first["space"].(map[string]any)
	if space["method"] != "coordinate_ascent" || space["size"].(float64) <= 300 || first["proven_optimal"] != false {
		t.Fatalf("space = %v", space)
	}
	used := first["evaluations"].(map[string]any)["used"].(float64)
	if used > 300 {
		t.Fatalf("used %v of a 300 budget", used)
	}
	t.Logf("%.0f evaluations in %v ms", used, first["elapsed_ms"])
	if again := call(t, s, "optimize", supportSpace(h, 7, 300)); strip(again) != strip(first) {
		t.Fatalf("seed 7 gave two answers:\n%s\n%s", strip(first), strip(again))
	}
	best := rows(first)[0]
	if best["feasible"] != true || best["score"].(float64) < first["objective"].(map[string]any)["current"].(float64) {
		t.Fatalf("best = %v", best)
	}

	timed := supportSpace(h, 7, 2000)
	timed["max_ms"] = 1
	res := call(t, s, "optimize", timed)
	if res["stopped"] != "time" || res["evaluations"].(map[string]any)["used"].(float64) > 5 {
		t.Fatalf("max_ms 1: stopped %v after %v", res["stopped"], res["evaluations"])
	}
}

func TestOptimizeRefusals(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	bad := supportSpace(h, 1, 300)
	bad["objective"] = map[string]any{"stat": "Lief"}
	if e := callErr(t, s, "optimize", bad); e.Code != CodeUnknownStat || e.DidYouMean[0] != "Life" {
		t.Fatalf("unknown objective: %s %v", e.Code, e.DidYouMean)
	}
	bad = supportSpace(h, 1, 300)
	bad["constraints"] = []any{map[string]any{"stat": "Resistance", "op": ">=", "value": 75}}
	if e := callErr(t, s, "optimize", bad); e.Code != CodeUnknownStat {
		t.Fatalf("unknown constraint: %s", e.Code)
	}
	dup := map[string]any{"handle": h, "objective": map[string]any{"stat": "Life"}, "slots": []any{
		map[string]any{"kind": "item", "slot": "Ring 1", "uniques": []string{"Mark of the Elder"}},
		map[string]any{"kind": "item", "slot": "Ring 1", "uniques": []string{"Mark of the Shaper"}},
	}}
	if e := callErr(t, s, "optimize", dup); e.Code != CodeInvalidArgument || e.SlotIndex == nil || *e.SlotIndex != 1 {
		t.Fatalf("duplicate slot: %s %v", e.Code, e.SlotIndex)
	}
	wrong := map[string]any{"handle": h, "objective": map[string]any{"stat": "Life"}, "slots": []any{
		map[string]any{"kind": "item", "slot": "Helmet", "uniques": []string{"Mark of the Elder"}},
	}}
	if e := callErr(t, s, "optimize", wrong); e.Code != CodeInvalidSlot || e.SlotIndex == nil {
		t.Fatalf("ring in helmet slot: %s", e.Code)
	}
}

func allSupports(n int) []string {
	var all []string
	for _, g := range gems() {
		if g.Support && len(all) < n {
			all = append(all, g.Name)
		}
	}
	return all
}

// TestOptimizeSearchesLargeKOfN runs a C(65,5) support space (v1 refused anything above 500) by swap moves and
// returns a best-so-far no worse than the current build.
func TestOptimizeSearchesLargeKOfN(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := assertSize(t, s, "optimize", map[string]any{"handle": h, "budget": 600, "objective": map[string]any{"stat": "MainDPS"},
		"slots": []any{map[string]any{"kind": "supports", "k": 5, "candidates": allSupports(60)}}}, capOptimize)
	space := res["space"].(map[string]any)
	if space["method"] != "coordinate_ascent" || space["size"].(float64) < 1e6 || res["proven_optimal"] != false {
		t.Fatalf("space = %v", space)
	}
	best := rows(res)[0]
	if best["score"].(float64) < res["objective"].(map[string]any)["current"].(float64) {
		t.Fatalf("best %v is below the current build", best)
	}
	if used := res["evaluations"].(map[string]any)["used"].(float64); used > 600 {
		t.Fatalf("used %v of 600", used)
	}

	low := call(t, s, "optimize", map[string]any{"handle": h, "budget": 20, "objective": map[string]any{"stat": "MainDPS"},
		"slots": []any{map[string]any{"kind": "supports", "k": 5, "candidates": allSupports(60)}}})
	if low["stopped"] != "budget" || !strings.Contains(low["hint"].(string), "raise budget to >= ") {
		t.Fatalf("budget below one sweep: stopped %v hint %v", low["stopped"], low["hint"])
	}
}

// TestOptimizeRefusesOnlyWhenOneSweepExceedsTheCap checks the refusal's shrink suggestion is achievable: the pool
// size it names runs.
func TestOptimizeRefusesOnlyWhenOneSweepExceedsTheCap(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	all := allSupports(400)
	slot := func(n int) map[string]any {
		return map[string]any{"handle": h, "budget": 5, "objective": map[string]any{"stat": "MainDPS"},
			"slots": []any{map[string]any{"kind": "supports", "k": 10, "keep_current": false, "candidates": all[:n]}}}
	}
	e := callErr(t, s, "optimize", slot(len(all)))
	if e.Code != CodeSpaceTooLarge || len(e.Suggestions) != 1 {
		t.Fatalf("k=10 of %d: %s %v", len(all), e.Code, e.Suggestions)
	}
	var keep int
	if _, err := fmt.Sscanf(e.Suggestions[0][strings.Index(e.Suggestions[0], "keep at most"):], "keep at most %d", &keep); err != nil {
		t.Fatalf("suggestion %q names no pool size: %v", e.Suggestions[0], err)
	}
	if keep <= 10 || keep >= len(all) {
		t.Fatalf("suggested pool %d", keep)
	}
	call(t, s, "optimize", slot(keep))
	if e := callErr(t, s, "optimize", slot(keep+1)); e.Code != CodeSpaceTooLarge {
		t.Fatalf("pool %d: %s", keep+1, e.Code)
	}
}

// TestOptimizePointsAddedChargesMasteriesAndRemovals checks points_added equals what committing the pick costs: a
// mastery is a point (as once its effect is set), and a removal frees its dependents' points too.
func TestOptimizePointsAddedChargesMasteriesAndRemovals(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	used := func() float64 {
		return call(t, s, "get_build", map[string]any{"handle": h})["points"].(map[string]any)["used"].(float64)
	}
	before := used()
	res := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "Life"},
		"slots":       []any{map[string]any{"kind": "nodes", "candidates": []string{"12382"}}},
		"constraints": []any{map[string]any{"stat": "points_added", "op": ">=", "value": 1}}})
	top := rows(res)[0]
	if add := top["choices"].([]any)[0].(map[string]any)["choice"].(map[string]any)["add"].([]any); len(add) != 1 {
		t.Fatalf("mastery not picked: %v", top)
	}
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_mastery", "node": 12382, "effect": "+30 to maximum Life"})})
	if got := used() - before; got != top["points_added"].(float64) || got < 1 {
		t.Fatalf("optimize charged %v points for the mastery, committing it cost %v", top["points_added"], got)
	}
	call(t, s, "revert_build", map[string]any{"handle": h, "to_revision": 0})

	removal := rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "Life"}))
	var victim map[string]any
	for _, r := range removal {
		if r["points_freed"].(float64) >= 2 {
			victim = r
			break
		}
	}
	if victim == nil {
		t.Fatal("no removal frees 2+ points")
	}
	res = call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "Life"}, "apply": true,
		"slots":       []any{map[string]any{"kind": "nodes", "remove": []string{fmt.Sprint(victim["id"])}}},
		"constraints": []any{map[string]any{"stat": "points_added", "op": "<=", "value": -1}}})
	top = rows(res)[0]
	if top["points_added"].(float64) != -victim["points_freed"].(float64) || res["applied"] == nil {
		t.Fatalf("removing %v (frees %v): %v", victim["name"], victim["points_freed"], top)
	}
	if got := used() - before; got != top["points_added"].(float64) {
		t.Fatalf("optimize reported %v points, applying it moved used by %v", top["points_added"], got)
	}
}

// TestOptimizeCopiesLeaveTheBaseIntact scores every state of a mixed space on a build copy and on a fresh parse:
// the values must agree and the base build must serialize unchanged.
func TestOptimizeCopiesLeaveTheBaseIntact(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	hd := s.handles[h]
	base, e := hd.parseRev(0)
	if e != nil {
		t.Fatal(e)
	}
	slots, e := buildSlots(base, []optSlotIn{
		{Kind: "item", Slot: "Ring 1", Uniques: []string{"Mark of the Elder"}},
		{Kind: "supports", Candidates: []string{"Ruthless"}},
		{Kind: "nodes", Candidates: []string{"Heart of Oak"}, Remove: []string{fmt.Sprint(rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "Life"}))[0]["id"])}},
	})
	if e != nil {
		t.Fatal(e)
	}
	xml, _ := serialize(base)
	lists := [][][]int{slots[0].enumerate(), slots[1].enumerate(), slots[2].enumerate()}
	n := 0
	for _, a := range lists[0] {
		for _, b := range lists[1] {
			for _, c := range lists[2] {
				st := state{a, b, c}
				cp := cloneBuild(base)
				fresh, _ := (&handle{revs: []*revision{{xml: xml}}}).parseRev(0)
				for i, sl := range slots {
					sl.apply(cp, st[i])
					sl.apply(fresh, st[i])
				}
				x, y := calcEnv(cp), calcEnv(fresh)
				for _, k := range []string{"Life", mainDPS, "FireResist"} {
					if vx, vy := statOf(x, k), statOf(y, k); vx != vy {
						t.Fatalf("%v: %s copy %v, parse %v", st.key(), k, vx, vy)
					}
				}
				n++
			}
		}
	}
	if after, _ := serialize(base); string(after) != string(xml) || n != len(lists[0])*len(lists[1])*len(lists[2]) || n < 8 {
		t.Fatalf("base changed after %d copies (or wrong count)", n)
	}
}

// TestOptimizePicksMasteryEffects scores a mastery as (node, effect) pairs, as edit_tree commits it: the best result
// holding the mastery names an effect, and committing it through edit_tree gives the points and Life optimize
// reported.
func TestOptimizePicksMasteryEffects(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	usedBefore := call(t, s, "get_build", map[string]any{"handle": h})["points"].(map[string]any)["used"].(float64)
	res := call(t, s, "optimize", map[string]any{"handle": h, "objective": map[string]any{"stat": "Life"}, "top": 5,
		"slots":       []any{map[string]any{"kind": "nodes", "candidates": []string{"12382", "Heart of Oak"}, "k": 2}},
		"constraints": []any{map[string]any{"stat": "points_added", "op": "<=", "value": 2}}})
	var top map[string]any
	var add []any
	for _, r := range rows(res) {
		a := r["choices"].([]any)[0].(map[string]any)["choice"].(map[string]any)["add"].([]any)
		if strings.Contains(fmt.Sprint(a), "Mastery (") {
			top, add = r, a
			break
		}
	}
	if top == nil {
		t.Fatalf("no result holds the mastery: %v", rows(res))
	}
	var effect string
	var others []map[string]any
	for _, a := range add {
		name, eff, ok := strings.Cut(a.(string), ": ")
		if ok && strings.Contains(name, "Mastery") {
			if effect != "" {
				t.Fatalf("two effects of one mastery picked: %v", add)
			}
			effect = eff
			continue
		}
		id := name[strings.LastIndex(name, "(")+1 : len(name)-1]
		others = append(others, map[string]any{"op": "allocate", "node": id})
	}
	if effect == "" {
		t.Fatalf("best Life pick %v (effect %q)", add, effect)
	}
	var lifeAfter float64
	for _, d := range top["delta"].([]any) {
		if row := d.(map[string]any); row["stat"] == "Life" {
			lifeAfter = row["after"].(float64)
		}
	}
	base, _ := s.handles[h].parseRev(0)
	sl, e := nodesSlot(base, optSlotIn{Kind: "nodes", Candidates: []string{"12382", "Heart of Oak"}, K: ptrInt(2)})
	if e != nil {
		t.Fatal(e)
	}
	for _, sel := range sl.enumerate() {
		if len(sel) == 2 && sl.group[sel[0]] >= 0 && sl.group[sel[0]] == sl.group[sel[1]] {
			t.Fatalf("a state takes two effects of one mastery: %v", sl.label(sel))
		}
	}
	for _, nb := range sl.neighbours([]int{0}) {
		if !sl.compatible(nb) {
			t.Fatalf("a move takes two effects of one mastery: %v", sl.label(nb))
		}
	}

	batch := append(others, map[string]any{"op": "set_mastery", "node": 12382, "effect": effect})
	call(t, s, "edit_tree", map[string]any{"handle": h, "ops": batch})
	b := call(t, s, "get_build", map[string]any{"handle": h})
	if used := b["points"].(map[string]any)["used"].(float64) - usedBefore; used != top["points_added"].(float64) || lifeOf(t, s, h) != lifeAfter {
		t.Fatalf("optimize: %v points, Life %v; edit_tree: %v points, Life %v", top["points_added"], lifeAfter, used, lifeOf(t, s, h))
	}
}

func ptrInt(v int) *int { return &v }
