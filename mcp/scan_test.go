package mcp

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
)

// committedChange applies a scan row by its id with stat watched, reverts it, and returns the stat's change and the
// resist block of the committed edit.
func committedChange(t *testing.T, s *Server, h, id, stat string) (float64, map[string]any) {
	t.Helper()
	res := call(t, s, "scan_build", map[string]any{"handle": h, "apply": id, "watch": []string{stat}})
	defer call(t, s, "revert_build", map[string]any{"handle": h, "to_revision": 0})
	if u, ok := res["unchanged"].(map[string]any); ok {
		if _, ok := u[stat]; ok {
			return 0, res["resists"].(map[string]any)
		}
	}
	for _, r := range res["delta"].([]any) {
		row := r.(map[string]any)
		if row["stat"] == stat {
			return row["after"].(float64) - row["before"].(float64), res["resists"].(map[string]any)
		}
	}
	t.Fatalf("%s: %s missing from the committed delta: %v", id, stat, res)
	return 0, nil
}

func near(a, b float64) bool {
	return math.Abs(a-b) <= math.Max(1, 0.005*math.Abs(b))
}

// TestScanBuildRowsMatchCommittedEdits applies the top row of every part and checks the row's gain and resist gap
// against the build with that change committed.
func TestScanBuildRowsMatchCommittedEdits(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	for _, tc := range []struct {
		part, stat string
		slots      []string
	}{
		{"uniques", "Life", []string{"Ring 1", "Belt"}},
		{"flasks", mainDPS, nil},
		{"anoints", mainDPS, nil},
		{"affixes", "Life", nil},
	} {
		args := map[string]any{"handle": h, "parts": []string{tc.part}, "stat": tc.stat}
		if tc.slots != nil {
			args["slots"] = tc.slots
		}
		res := assertSize(t, s, "scan_build", args, capScan)
		list := rows(res)
		if len(list) == 0 {
			t.Fatalf("%s: no rows: %v", tc.part, res)
		}
		top := list[0]
		if top["part"] != tc.part || top["gain"].(float64) <= 0 {
			t.Fatalf("%s: top row %v", tc.part, top)
		}
		got, resists := committedChange(t, s, h, top["id"].(string), tc.stat)
		if !near(top["gain"].(float64), got) {
			t.Fatalf("%s %v: row gain %v, committed %v", tc.part, top["id"], top["gain"], got)
		}
		gap, _ := top["resist_gap"].(map[string]any)
		for _, el := range []string{"Fire", "Cold", "Lightning"} {
			hr := resists[el].(map[string]any)["headroom"].(float64)
			if g, ok := gap[el].(float64); ok != (hr < 0 && hr < res["base_headroom"].(map[string]any)[el].(float64)) || (ok && g != hr) {
				t.Fatalf("%s %v: resist_gap %v, committed %s headroom %v", tc.part, top["id"], gap, el, hr)
			}
		}
	}
}

// TestScanBuildKeepResistsDropsGaps: Life uniques for the rings and belt include swaps that open a resist gap, each
// saying what refills it; keep_resists drops exactly those.
func TestScanBuildKeepResistsDropsGaps(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	args := map[string]any{"handle": h, "parts": []string{"uniques"}, "stat": "Life", "slots": []string{"Ring 1", "Ring 2", "Belt"}, "per_slot": 5}
	gaps := 0
	for _, r := range rows(call(t, s, "scan_build", args)) {
		if r["resist_gap"] != nil {
			gaps++
			if r["refill"] == nil {
				t.Fatalf("row opens a gap without a refill: %v", r)
			}
		}
	}
	if gaps == 0 {
		t.Fatal("no row opens a resist gap; the fixture no longer exercises keep_resists")
	}
	args["keep_resists"] = true
	kept := rows(call(t, s, "scan_build", args))
	if len(kept) == 0 {
		t.Fatal("keep_resists dropped every row")
	}
	for _, r := range kept {
		if r["resist_gap"] != nil {
			t.Fatalf("keep_resists kept %v", r)
		}
	}
}

// TestScanBuildFlaskSlots: with every flask slot filled, a row names the flask it replaces; with one emptied, rows go
// in that slot (inactive before), and the committed flask is active and gains what the row says.
func TestScanBuildFlaskSlots(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	full := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"flasks"}}))
	if len(full) == 0 || full[0]["replaces"] == nil {
		t.Fatalf("full flask slots: %v", full)
	}
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_flask_active", "slot": "Flask 2", "active": false}, map[string]any{"op": "unequip", "slot": "Flask 2"})})
	list := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"flasks"}}))
	if len(list) == 0 || list[0]["slot"] != "Flask 2" || list[0]["replaces"] != nil {
		t.Fatalf("empty Flask 2: %v", list)
	}
	res := call(t, s, "scan_build", map[string]any{"handle": h, "apply": list[0]["id"], "watch": []string{mainDPS}})
	var got float64
	for _, r := range res["delta"].([]any) {
		if row := r.(map[string]any); row["stat"] == mainDPS {
			got = row["after"].(float64) - row["before"].(float64)
		}
	}
	if !near(list[0]["gain"].(float64), got) {
		t.Fatalf("%v: row gain %v, committed %v", list[0]["id"], list[0]["gain"], got)
	}
	hh, _ := s.get(h)
	b, _ := hh.parseRev(hh.cur)
	for _, eq := range equippedItems(b) {
		if eq.slot == "Flask 2" && !eq.active {
			t.Fatalf("Flask 2 not active after applying %v", list[0]["id"])
		}
	}
}

// TestAffixCountsAndGroups: a rare's explicit lines are matched to prefix and suffix, and a group it already has is
// not offered again.
func TestAffixCountsAndGroups(t *testing.T) {
	_ = testServer(t)
	text := "Rarity: RARE\nTest Hood\nSilken Hood\nItem Level: 84\nImplicits: 0\n+90 to maximum Life\n+40% to Fire Resistance\n+40% to Cold Resistance\n"
	item := calculator.ParseItem(text)
	a := countAffixes(item)
	if a.pre != 1 || a.suf != 2 || a.exact {
		t.Fatalf("counts %d/%d exact=%v", a.pre, a.suf, a.exact)
	}
	cands := affixCandidates(text, item, a, defaultRoll)
	if len(cands) == 0 {
		t.Fatal("no candidates")
	}
	for _, c := range cands {
		if a.groups[c.group] {
			t.Fatalf("offered a group the item has: %v", c.label)
		}
	}
}

// TestScanBuildReservation: per-skill reservations add up to the build's reserved mana, and the top source for more
// unreserved mana gains what its committed edit gains.
func TestScanBuildReservation(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := assertSize(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"reservation"}, "mana_needed": 100}, capScan)
	info := res["reservation"].(map[string]any)
	sum := 0.0
	for _, sk := range info["skills"].([]any) {
		sum += num(sk, "mana")
	}
	// PoB ceils the summed percentage once (CalcPerform.lua:562); each skill's share is ceiled on its own
	reserved := info["Mana"].(float64) - info[manaUnreserved].(float64)
	if sum == 0 || math.Abs(sum-reserved) > float64(len(info["skills"].([]any))) {
		t.Fatalf("skills reserve %v, build %v", sum, reserved)
	}
	if reserved > info["Mana"].(float64) && info["over_reserved"] == nil {
		t.Fatalf("over-reserved build not flagged: %v", info)
	}
	list := rows(res)
	if len(list) == 0 || info["node_plan"] == nil {
		t.Fatalf("no mana sources: %v", res)
	}
	got, _ := committedChange(t, s, h, list[0]["id"].(string), manaUnreserved)
	if !near(list[0]["gain"].(float64), got) {
		t.Fatalf("%v: row gain %v, committed %v", list[0]["id"], list[0]["gain"], got)
	}
}

// TestScanBuildClusterNotables: a socketed cluster jewel's notable swap gains what the committed swap gains, and every
// offered notable is one the jewel's skill rolls.
func TestScanBuildClusterNotables(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "import_build", map[string]any{"source": fixture("7.xml")})["handle"].(string)
	res := assertSize(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"clusters"}}, capScan)
	list := rows(res)
	if len(list) == 0 {
		t.Fatalf("no cluster rows: %v", res)
	}
	top := list[0]
	if !strings.HasPrefix(top["slot"].(string), "Jewel ") || top["replaces"] == nil {
		t.Fatalf("top row %v", top)
	}
	got, _ := committedChange(t, s, h, top["id"].(string), mainDPS)
	if !near(top["gain"].(float64), got) {
		t.Fatalf("%v: row gain %v, committed %v", top["id"], top["gain"], got)
	}
}

// TestScanBuildClusterRowsReplaceNotables: a jewel socket line is not a notable to swap out.
func TestScanBuildClusterRowsReplaceNotables(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "import_build", map[string]any{"source": fixture("7.xml")})["handle"].(string)
	for _, r := range rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"clusters"}, "per_slot": 5})) {
		if _, ok := data.ClusterJewels.NotableSortOrder[r["replaces"].(string)]; !ok {
			t.Fatalf("row replaces %q, not a notable: %v", r["replaces"], r)
		}
	}
}

// TestScanBuildUniquesFollowRankOrder: per slot, scan rows are the uniques ranking's rows that gain, ranked by their
// worst-case gain, so a unique whose cost its flags name (overfilled sockets, a dropped off-hand, a lost totem) or that
// is unusable or has big_loss stays after clean ones.
func TestScanBuildUniquesFollowRankOrder(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	for _, slot := range gearSlots {
		hh, _ := s.get(h)
		b, _ := hh.parseRev(hh.cur)
		ranked, _, _, _ := rankUniques(b, "Life", rankIn{Slot: slot}, time.Now(), math.MaxInt, nil)
		dirty := map[any]bool{}
		for _, r := range ranked {
			if r["gain"].(float64) > 0 && len(dirty) < 5 {
				dirty[r["name"]] = r[dirtyKey] != nil
			}
		}
		lastDirty, last := false, math.Inf(1)
		for _, r := range rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"uniques"}, "slots": []string{slot}, "per_slot": 5, "stat": "Life"})) {
			d, ok := dirty[r["name"]]
			d = d || r["unusable"] != nil || r["big_loss"] != nil
			score, scored := r["worst_gain"].(float64)
			if !scored {
				score = r["gain"].(float64)
			}
			if d != lastDirty {
				last = math.Inf(1)
			}
			if !ok || (lastDirty && !d) || score > last {
				t.Fatalf("%s: %v (dirty %v) out of the ranking's gaining rows %v or out of order", slot, r["name"], d, dirty)
			}
			lastDirty, last = d, score
		}
	}
}

// TestScanReservationRanksByManaInTheWorstCase: reservation rows gain unreserved mana, so a forced condition's worst
// case (a DPS figure) neither ranks nor drops them: they stay in order of the mana they free.
func TestScanReservationRanksByManaInTheWorstCase(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar", "level": 90})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Fireball"}}, map[string]any{"op": "set_main", "group": 1},
		map[string]any{"op": "add_group", "gems": []string{"Clarity"}})})
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "buffOnslaught", "value": true})})
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"reservation"}, "mana_needed": 50})
	got := rows(res)
	if len(got) < 2 || res["worst_case"] == nil {
		t.Fatalf("want several rows scored in the worst case: %v", res)
	}
	for i, r := range got {
		if r["worst_gain"] != nil {
			t.Fatalf("%s carries worst_gain %v, a DPS figure, beside its mana gain %v", r["name"], r["worst_gain"], r["gain"])
		}
		if i > 0 && r["gain"].(float64) > got[i-1]["gain"].(float64) {
			t.Fatalf("rows out of mana order: %v", got)
		}
	}
}
