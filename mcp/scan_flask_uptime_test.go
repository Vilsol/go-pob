package mcp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
)

// TestScanWorstCountsFlaskUptime: worst_gain scores Replica Rumi's Concoction at its uptime, under half its always-on
// gain, and says so.
func TestScanWorstCountsFlaskUptime(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "stat": "TotalEHP", "parts": []string{"flasks"}, "per_slot": 5})
	for _, r := range rows(res) {
		if !strings.HasPrefix(r["name"].(string), "Replica Rumi's Concoction") {
			continue
		}
		w, ok := r["worst_gain"].(float64)
		if !ok || w >= num(r, "gain")/2 || !strings.Contains(fmt.Sprint(r["flags"]), "uptime: worst_gain counts") {
			t.Fatalf("Replica Rumi's Concoction gain %v worst_gain %v flags %v", r["gain"], r["worst_gain"], r["flags"])
		}
		return
	}
	t.Fatalf("no Replica Rumi's Concoction row: %v %v", resultNames(res), res["empty_parts"])
}

// TestFlaskUptime: Replica Rumi's Concoction lasts a tenth of its base's duration for the base's charges, so it is up a
// tenth of the time; its base and a life flask are up all the time.
func TestFlaskUptime(t *testing.T) {
	testServer(t)
	rumi := ""
	for _, it := range calculator.ItemDB(calculator.ItemDBUnique) {
		if strings.HasPrefix(it.Name, "Replica Rumi's Concoction") {
			rumi, _ = calculator.ItemDBText(calculator.ItemDBUnique, it.Name)
		}
	}
	if rumi == "" {
		t.Fatal("no Replica Rumi's Concoction")
	}
	if u := flaskUptimeOf(calculator.ParseItem(rumi)); u <= 0 || u > 0.2 {
		t.Errorf("Replica Rumi's Concoction uptime %v", u)
	}
	for _, base := range []string{"Granite Flask", "Divine Life Flask"} {
		text, _ := calculator.CraftItemText(1, "", base)
		if u := flaskUptimeOf(calculator.ParseItem(text)); u != 1 {
			t.Errorf("%s uptime %v", base, u)
		}
	}
}
