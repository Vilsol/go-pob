package mcp

import (
	"testing"
)

// TestScanSockets: body armour whose sockets are all blue under green and red gems gets one row recolouring each socket
// to its gem's colour, links kept, gaining what the committed recolour gains; after it, and with white sockets, there
// is nothing to recolour.
func TestScanSockets(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	scan := func(sockets string) ([]map[string]any, map[string]any) {
		t.Helper()
		if sockets != "" {
			call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Astral Plate", "slot": "Body Armour", "sockets": sockets})})
		}
		res := assertSize(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"sockets"}, "slots": []string{"Body Armour"}}, capScan)
		return rows(res), res
	}

	list, _ := scan("B-B-B B-B-B")
	if len(list) != 1 {
		t.Fatalf("want one row, got %v", list)
	}
	r := list[0]
	if r["id"] != "sockets|Body Armour" || r["was"] != "B-B-B B-B-B" || r["sockets"] != "G-G-R R-R-R" || r["recoloured"] != 6.0 {
		t.Fatalf("row: %v", r)
	}
	gain := r["gain"].(float64)
	if gain <= 0 {
		t.Fatalf("matching sockets gain nothing: %v", r)
	}

	res := call(t, s, "scan_build", map[string]any{"handle": h, "apply": r["id"], "watch": []string{mainDPS}})
	var got float64
	for _, d := range res["delta"].([]any) {
		if m := d.(map[string]any); m["stat"] == mainDPS {
			got = m["after"].(float64) - m["before"].(float64)
		}
	}
	if !near(got, gain) {
		t.Fatalf("committed recolour gains %v, the row %v", got, gain)
	}
	if list, res := scan(""); len(list) != 0 || res["empty_parts"].(map[string]any)["sockets"] == nil {
		t.Fatalf("recoloured item still recolours: %v", res)
	}
	if list, _ := scan("W-W-W-W-W-W"); len(list) != 0 {
		t.Fatalf("white sockets recoloured: %v", list)
	}
}
