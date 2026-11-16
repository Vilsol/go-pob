package mcp

import "testing"

// TestScanFlaskRowOncePerFlask: under rank_by savings with two dear flasks, a cheaper flask is one row, in the slot it
// saves most in, not one row per slot.
func TestScanFlaskRowOncePerFlask(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = progenesisBook(t)
	h := progenesisBuild(t, s)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "equip_unique", "name": "Vorana's Preparation", "slot": "Flask 2"},
		map[string]any{"op": "set_flask_active", "slot": "Flask 2", "active": true})})
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"flasks"}, "rank_by": "savings", "per_slot": 5})
	seen := map[string]string{}
	for _, r := range rows(res) {
		n := r["name"].(string)
		if was, ok := seen[n]; ok {
			t.Fatalf("%s in %s and %v", n, was, r["slot"])
		}
		seen[n] = r["slot"].(string)
	}
	if len(seen) == 0 {
		t.Fatalf("no rows: %v", res)
	}
}
