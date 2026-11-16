package mcp

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
)

// TestScanFlasksScoreCandidatesActive: with every flask slot full and switched off, a candidate replaces Flask 1 and
// is scored active, as applying the row leaves it.
func TestScanFlasksScoreCandidatesActive(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	off := make([]map[string]any, 0, len(flaskSlots))
	for _, sl := range flaskSlots {
		off = append(off, map[string]any{"op": "set_flask_active", "slot": sl, "active": false})
	}
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(off...)})
	list := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"flasks"}, "stat": mainDPS}))
	if len(list) == 0 {
		t.Fatal("no flask row gains")
	}
	top := list[0]
	if top["slot"] != "Flask 1" {
		t.Fatalf("top row %v", top)
	}
	if got, _ := committedChange(t, s, h, top["id"].(string), mainDPS); !near(top["gain"].(float64), got) {
		t.Fatalf("%v: row gain %v, committed %v", top["id"], top["gain"], got)
	}
}

// TestScanFlasksRankByOffenceDefence: the flasks part ranks by PoB's Offence/Defence power of the candidate active,
// as rank_candidates scores a flask slot; the other parts still refuse it.
func TestScanFlasksRankByOffenceDefence(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"flasks"}, "stat": "OffenceDefence"})
	list := rows(res)
	if len(list) == 0 {
		t.Fatalf("no rows: %v", res)
	}
	for _, r := range list {
		if g := num(r, "gain"); g <= 0 || math.Abs(g-num(r, "offence_pct")-num(r, "defence_pct")) > 0.11 {
			t.Fatalf("gain is not offence_pct + defence_pct: %v", r)
		}
	}
	callErr(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"uniques"}, "stat": "OffenceDefence"})
}

// TestScanFlasksOffenceDefenceReplacesWeakest: with every flask slot full, OffenceDefence rows replace the flask whose
// removal loses the least Offence/Defence power, here not Flask 1.
func TestScanFlasksOffenceDefenceReplacesWeakest(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	one, two := slotText(t, s, h, "Flask 1"), slotText(t, s, h, "Flask 2")
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "equip_text", "text": two, "slot": "Flask 1"},
		map[string]any{"op": "equip_text", "text": one, "slot": "Flask 2"},
		map[string]any{"op": "set_flask_active", "slot": "Flask 1", "active": true},
		map[string]any{"op": "set_flask_active", "slot": "Flask 2", "active": true})})
	hh, _ := s.get(h)
	b, _ := hh.parseRev(hh.cur)
	env, _ := hh.envAt(hh.cur)
	ps, e := powerStat(env, "OffenceDefence")
	if e != nil {
		t.Fatal(e)
	}
	misc := calculator.GetMiscCalculator(b)
	want, best := "", math.Inf(-1)
	for _, sl := range flaskSlots {
		gain, _, _ := rankGain(ps, "OffenceDefence", misc.Base(), misc.Calc(&calculator.CalcOverride{RepSlotName: sl}))
		if gain > best {
			want, best = sl, gain
		}
	}
	list := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"flasks"}, "stat": "OffenceDefence"}))
	if want == "Flask 1" || len(list) == 0 || list[0]["slot"] != want {
		t.Fatalf("rows %v, want them in %s (the weakest, and not Flask 1 in this build)", list, want)
	}
}

// TestScanFlagsFlaskDrawbacks: Replica Rumi's Concoction's effect lasts a tenth of its base's and it petrifies, which
// PoB does not model; its row says both, in full.
func TestScanFlagsFlaskDrawbacks(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "stat": "TotalEHP", "parts": []string{"flasks"}, "per_slot": 5})
	for _, r := range rows(res) {
		if !strings.HasPrefix(r["name"].(string), "Replica Rumi's Concoction") {
			continue
		}
		flags := fmt.Sprint(r["flags"])
		if !strings.Contains(flags, "uptime: ") || !strings.Contains(flags, "unmodelled: Petrified during Effect") || strings.Contains(flags, "Unsupported mod in") {
			t.Fatalf("Replica Rumi's Concoction flags %v", r["flags"])
		}
		return
	}
	t.Fatalf("no Replica Rumi's Concoction row: %v", rows(res))
}
