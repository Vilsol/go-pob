package mcp

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mcp/prices"
)

// budgetBook is a snapshot whose prices are invented: an expensive unique flask and a cheap one that keeps most of its
// gain, and two levels of one gem.
func budgetBook(t *testing.T, currency bool) *prices.Book {
	t.Helper()
	responses := map[string]string{
		"UniqueWeapon":    `{"lines":[]}`,
		"UniqueArmour":    `{"lines":[]}`,
		"UniqueAccessory": `{"lines":[]}`,
		"UniqueJewel":     `{"lines":[]}`,
		"BaseType":        `{"lines":[]}`,
		"ClusterJewel":    `{"lines":[]}`,
		"UniqueFlask": `{"lines":[
			{"name":"Vorana's Preparation","baseType":"Iron Flask","itemClass":3,"chaosValue":6000,"listingCount":10},
			{"name":"Bottled Faith","baseType":"Sulphur Flask","itemClass":3,"chaosValue":20,"listingCount":10}]}`,
		"SkillGem": `{"lines":[
			{"name":"Controlled Destruction Support","gemLevel":20,"gemQuality":20,"chaosValue":5,"listingCount":10},
			{"name":"Controlled Destruction Support","gemLevel":21,"gemQuality":20,"corrupted":true,"chaosValue":900,"listingCount":10}]}`,
	}
	if currency {
		responses["Currency"] = `{"lines":[{"currencyTypeName":"Divine Orb","chaosEquivalent":150}]}`
	}
	snap := map[string]any{"league": "Testleague", "fetched_at": "2026-01-01T00:00:00Z", "responses": map[string]json.RawMessage{}}
	for k, v := range responses {
		snap["responses"].(map[string]json.RawMessage)[k] = json.RawMessage(v)
	}
	b, _ := json.Marshal(snap)
	book, err := prices.FromSnapshot(strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	return book
}

func resultNames(res map[string]any) []string {
	rs := res["results"].([]any)
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.(map[string]any)["name"].(string))
	}
	return out
}

func resultNamed(t *testing.T, res map[string]any, prefix string) map[string]any {
	t.Helper()
	for _, r := range res["results"].([]any) {
		if strings.HasPrefix(r.(map[string]any)["name"].(string), prefix) {
			return r.(map[string]any)
		}
	}
	t.Fatalf("no row %s in %v", prefix, resultNames(res))
	return nil
}

func TestScanCostBudget(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = budgetBook(t, true)
	h := importFixture(t, s)
	args := map[string]any{"handle": h, "parts": []string{"flasks"}, "per_slot": 5}

	// a cheaper unique that keeps most of the gain ranks first by gain per divine
	args["rank_by"] = "gain_per_div"
	res := call(t, s, "scan_build", args)
	names := resultNames(res)
	at := func(prefix string) int {
		return slices.IndexFunc(names, func(n string) bool { return strings.HasPrefix(n, prefix) })
	}
	if at("Bottled Faith") < 0 || at("Bottled Faith") > at("Vorana's Preparation") || at("Vessel of Vinktar") >= 0 {
		t.Fatalf("rank_by gain_per_div = %v, want Bottled Faith above Vorana's Preparation and unknown prices dropped", names)
	}
	cheap, dear := resultNamed(t, res, "Bottled Faith"), resultNamed(t, res, "Vorana's")
	if cheap["gain"].(float64) >= dear["gain"].(float64) || cheap["gain_per_div"].(float64) <= dear["gain_per_div"].(float64) {
		t.Fatalf("Bottled Faith %v vs Vorana's %v", cheap, dear)
	}
	summary := res["cost_budget"].(map[string]any)
	if summary["dropped_unknown_price"].(float64) == 0 {
		t.Errorf("unknown-priced flasks not counted: %v", summary)
	}
	build := summary["build_chaos"].(float64)

	// a budget under the dear flask drops it and says so
	delete(args, "rank_by")
	args["cost_budget"] = build + 100
	res = call(t, s, "scan_build", args)
	if names := resultNames(res); slices.ContainsFunc(names, func(n string) bool { return strings.HasPrefix(n, "Vorana's") }) || !slices.ContainsFunc(names, func(n string) bool { return strings.HasPrefix(n, "Bottled Faith") }) {
		t.Fatalf("cost_budget %v = %v, want Bottled Faith without Vorana's Preparation", build+100, names)
	}
	if n := res["cost_budget"].(map[string]any)["dropped_over_budget"].(float64); n != 1 {
		t.Errorf("dropped_over_budget = %v, want 1", n)
	}
	args["cost_budget"] = "1000div"
	if r := resultNamed(t, call(t, s, "scan_build", args), "Vorana's"); r["cost"] == nil {
		t.Errorf("Vorana's Preparation carries no cost: %v", r)
	}

	// allow_unknown_price keeps the unpriced rows, named
	args["allow_unknown_price"] = true
	res = call(t, s, "scan_build", args)
	if r := resultNamed(t, res, "Vessel of Vinktar"); r["cost_unknown"] == nil {
		t.Errorf("Vessel of Vinktar carries no cost_unknown: %v", r)
	}
}

func TestScanCostBudgetUnavailable(t *testing.T) {
	offline(t)
	s := testServer(t)
	h := importFixture(t, s)
	args := map[string]any{"handle": h, "parts": []string{"flasks"}, "cost_budget": "5div"}
	if e := callErr(t, s, "scan_build", args); e.Code != CodePricesUnavailable {
		t.Errorf("no prices: %s %s", e.Code, e.Message)
	}
	s.opts.Prices = budgetBook(t, false)
	if e := callErr(t, s, "scan_build", args); e.Code != CodePricesUnavailable || !strings.Contains(e.Message, "Currency") {
		t.Errorf("no currency: %s %s", e.Code, e.Message)
	}
	// without a budget nothing is priced
	book := budgetBook(t, true)
	s.opts.Prices = book
	call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"flasks"}})
	if _, at := book.Info(); !at.IsZero() {
		t.Error("scan_build without cost_budget loaded prices")
	}
}

// TestOptimizeCostBudgetGem: a 21/20 gem gains over 20/20, but a budget that cannot pay for the corrupted copy keeps
// 20/20 on top and counts the state it drops.
func TestOptimizeCostBudgetGem(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = budgetBook(t, true)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse", "Controlled Destruction"}, "level": 20, "quality": 20}, map[string]any{"op": "set_main", "group": 1})})
	args := map[string]any{"handle": h, "objective": map[string]any{"stat": "MainDPS"}, "top": 5,
		"slots": []any{map[string]any{"kind": "gem", "group": 1, "gem": "Controlled Destruction", "levels": []int{20, 21}, "qualities": []int{20}, "corrupted": true}}}
	choice := func(r map[string]any) string { return fmt.Sprint(r["choices"].([]any)[0].(map[string]any)["choice"]) }
	if c := choice(rows(call(t, s, "optimize", args))[0]); !strings.Contains(c, "level 21") {
		t.Fatalf("without a budget the best is %v, want level 21", c)
	}
	args["rank_by"] = "gain"
	args["cost_budget"] = 100
	res := call(t, s, "optimize", args)
	got := rows(res)
	for _, r := range got {
		if strings.Contains(choice(r), "level 21") {
			t.Fatalf("21/20 kept under a 100c budget: %v", got)
		}
	}
	if top := got[0]; !strings.Contains(choice(top), "level 20") || top["cost"] != 0.0 {
		t.Fatalf("top under budget %v", top)
	}
	if n := res["cost_budget"].(map[string]any)["dropped_over_budget"].(float64); n != 1 {
		t.Errorf("dropped_over_budget = %v, want 1", n)
	}
	args["cost_budget"] = "7div"
	if c := choice(rows(call(t, s, "optimize", args))[0]); !strings.Contains(c, "level 21") {
		t.Fatalf("7div (1050c) best is %v, want level 21", c)
	}
}
