package mcp

import (
	"math"
	"testing"
)

// TestRankTotemLossAgreesWithItsDPSColumn: a removal that lowers the totem limit loses a totem's share of MainDPS, which
// is per totem; the row's dDPS and worst_d.dDPS count the same lost totems as loss and worst_loss do, and say so.
func TestRankTotemLossAgreesWithItsDPSColumn(t *testing.T) {
	s := testServer(t)
	h := totemBuild(t, s)
	for _, forced := range []bool{false, true} {
		if forced {
			call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "TotemsSummoned", "value": 2})})
		}
		res := call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "remove_nodes", "stat": "MainDPS", "limit": 25, "worst_case": true})
		r := removeRow(t, res, 41970)
		agree := func(col, gain string) {
			d, _ := r[col].(map[string]any)
			got, _ := d["dDPS"].(float64)
			want := -num(r, gain)
			if math.Abs(got-want) > math.Max(1, 0.001*math.Abs(want)) {
				t.Errorf("forced %v: %s.dDPS %v, %s %v: %v", forced, col, got, gain, num(r, gain), r)
			}
		}
		agree("d", "loss")
		if forced {
			if num(r, "worst_loss") == 0 {
				t.Fatalf("forced TotemsSummoned: no worst_loss: %v", r)
			}
			agree("worst_d", "worst_loss")
		}
		if !flagged(r, "totems: ", "per totem") {
			t.Errorf("forced %v: no word that MainDPS is per totem: %v", forced, r["flags"])
		}
	}
}
