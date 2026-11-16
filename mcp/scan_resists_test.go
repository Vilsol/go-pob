package mcp

import (
	"reflect"
	"testing"
)

func resOpt(group string, bench bool, values ...resVec) resOption {
	o := resOption{group: group, bench: bench}
	for _, v := range values {
		o.tiers = append(o.tiers, resTier{src: "SUFFIX", id: group, v: v})
	}
	return o
}

// TestSolveResistsFewestThenLowestTier: the plan uses the fewest suffixes, never two mods of one group on an item nor
// more than its open suffixes or bench room, and takes each mod down to the lowest tier that still caps.
func TestSolveResistsFewestThenLowestTier(t *testing.T) {
	fire := []resVec{{46}, {40}, {30}, {20}}
	cold := []resVec{{0, 46}, {0, 40}, {0, 30}, {0, 20}}
	light := []resVec{{0, 0, 46}, {0, 0, 40}, {0, 0, 30}, {0, 0, 20}}
	items := []resItem{
		{slot: "Ring 1", open: 1, benchRoom: 1, opts: []resOption{resOpt("FireResistance", false, fire...), resOpt("AllResistances", false, resVec{16, 16, 16}, resVec{12, 12, 12})}},
		{slot: "Boots", open: 2, benchRoom: 0, opts: []resOption{resOpt("FireResistance", false, fire...), resOpt("ColdResistance", false, cold...)}},
		{slot: "Belt", open: 1, benchRoom: 1, opts: []resOption{resOpt("LightningResistance", false, light...), resOpt("ColdResistance", true, resVec{0, 30})}},
	}
	checkPlan(t, items, resVec{40, 35, 25}, 3)
	// the first item is tried first: a plan that ignored the per-item group or open-suffix rule would put both there
	checkPlan(t, []resItem{
		{slot: "Boots", open: 2, opts: []resOption{resOpt("FireResistance", false, fire...)}},
		{slot: "Ring 1", open: 1, opts: []resOption{resOpt("FireResistance", false, fire...)}},
	}, resVec{80}, 2)
	checkPlan(t, []resItem{
		{slot: "Ring 1", open: 1, opts: []resOption{resOpt("FireResistance", false, fire...), resOpt("ColdResistance", false, cold...)}},
		{slot: "Belt", open: 1, opts: []resOption{resOpt("FireResistance", false, fire...), resOpt("ColdResistance", false, cold...)}},
	}, resVec{40, 40}, 2)
	if _, ok := solveResists(items, resVec{200, 200, 200}); ok {
		t.Fatal("a plan caps 200 short in every element from four open suffixes")
	}
}

func checkPlan(t *testing.T, items []resItem, need resVec, size int) {
	t.Helper()
	picks, ok := solveResists(items, need)
	if !ok || len(picks) != size {
		t.Fatalf("plan %v ok=%v, want %d suffixes", picks, ok, size)
	}
	var got resVec
	used := map[int]int{}
	groups := map[[2]any]bool{}
	for _, p := range picks {
		it := items[p.item]
		o := it.opts[p.opt]
		used[p.item]++
		if used[p.item] > it.open || groups[[2]any{p.item, o.group}] {
			t.Fatalf("plan %v overfills %s", picks, it.slot)
		}
		groups[[2]any{p.item, o.group}] = true
		for e, x := range o.tiers[p.tier].v {
			got[e] += x
		}
	}
	for e := range need {
		if got[e] < need[e] {
			t.Fatalf("plan %v gives %v, short of %v", picks, got, need)
		}
	}
	for k, p := range picks {
		if p.tier+1 < len(items[p.item].opts[p.opt].tiers) {
			picks[k].tier++
			var lower resVec
			for _, q := range picks {
				for e, x := range items[q.item].opts[q.opt].tiers[q.tier].v {
					lower[e] += x
				}
			}
			picks[k].tier--
			if lower[0] >= need[0] && lower[1] >= need[1] && lower[2] >= need[2] {
				t.Fatalf("plan %v: pick %d caps one tier lower too", picks, k)
			}
		}
	}
}

// TestScanResistPlanCapsCommitted: with a resistance-less helmet leaving resistances under their caps, the resists part
// plans suffixes that cap them, short_by what is left after it, and applying its id caps them in the committed build.
func TestScanResistPlanCapsCommitted(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_text", "text": "Rarity: RARE\nTest Mask\nTorturer's Mask\nItem Level: 86\nImplicits: 0\n+90 to maximum Life", "slot": "Helmet"})})
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"resists"}})
	plan := res["resist_plan"].(map[string]any)
	if plan["id"] == nil || plan["suffixes"].(float64) < 1 {
		t.Fatalf("no plan: %v", plan)
	}
	if short, _ := plan["short_by"].(map[string]any); len(short) != 0 || len(plan["needed"].(map[string]any)) == 0 {
		t.Fatalf("a plan that caps them leaves short_by %v, needed %v", plan["short_by"], plan["needed"])
	}
	after := plan["headroom_after"].(map[string]any)
	applied := call(t, s, "scan_build", map[string]any{"handle": h, "apply": plan["id"]})
	for _, el := range []string{"Fire", "Cold", "Lightning"} {
		hr := applied["resists"].(map[string]any)[el].(map[string]any)["headroom"].(float64)
		if after[el].(float64) < 0 || hr < 0 || hr != after[el].(float64) {
			t.Fatalf("%s: plan headroom %v, committed %v", el, after[el], hr)
		}
	}
}

// TestRefillCountsTheRowsGap: a row that opens a resistance gap says which suffixes restore it, without the open
// suffixes of the slot a unique replaces.
func TestRefillCountsTheRowsGap(t *testing.T) {
	fire := resOpt("FireResistance", false, resVec{46})
	fire.tiers[0].lines = []string{"+(46-48)% to Fire Resistance"}
	items := []resItem{
		{slot: "Ring 1", open: 1, opts: []resOption{fire}},
		{slot: "Belt", open: 1, opts: []resOption{fire}},
	}
	room := map[string]float64{"Fire": 0, "Cold": 0, "Lightning": 0}
	row := map[string]any{"slot": "Helmet", "part": "uniques", "resist_gap": map[string]float64{"Fire": -60}}
	want := []string{"Ring 1: +47% to Fire Resistance (T1)", "Belt: +47% to Fire Resistance (T1)"}
	if got, ok := refill(items, room, row, defaultRoll).(map[string]any); !ok || got["suffixes"] != 2 || !reflect.DeepEqual(got["mods"], want) {
		t.Fatalf("refill %v, want 2 suffixes %q", got, want)
	}
	row["slot"] = "Belt"
	if got, ok := refill(items, room, row, defaultRoll).(string); !ok {
		t.Fatalf("refill %v counts the replaced Belt's suffix", got)
	}
}
