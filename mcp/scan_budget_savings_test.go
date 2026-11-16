package mcp

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mcp/prices"
)

// bookOf is a snapshot of invented prices: every listed category empty unless lines names it, and a 150c divine.
func bookOf(t *testing.T, lines map[string]string) *prices.Book {
	t.Helper()
	responses := map[string]json.RawMessage{}
	for _, c := range []string{"UniqueWeapon", "UniqueArmour", "UniqueAccessory", "UniqueFlask", "UniqueJewel", "BaseType", "ClusterJewel", "SkillGem"} {
		responses[c] = json.RawMessage(`{"lines":[]}`)
	}
	responses["Currency"] = json.RawMessage(`{"lines":[{"currencyTypeName":"Divine Orb","chaosEquivalent":150}]}`)
	for k, v := range lines {
		responses[k] = json.RawMessage(v)
	}
	b, _ := json.Marshal(map[string]any{"league": "Testleague", "fetched_at": "2026-01-01T00:00:00Z", "responses": responses})
	book, err := prices.FromSnapshot(strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	return book
}

// progenesisBook prices Progenesis dear and a few flasks cheap.
func progenesisBook(t *testing.T) *prices.Book {
	return bookOf(t, map[string]string{"UniqueFlask": `{"lines":[
		{"name":"Progenesis","baseType":"Amethyst Flask","itemClass":3,"chaosValue":6000,"listingCount":10},
		{"name":"Atziri's Promise","baseType":"Amethyst Flask","itemClass":3,"chaosValue":2,"listingCount":10},
		{"name":"Bottled Faith","baseType":"Sulphur Flask","itemClass":3,"chaosValue":20,"listingCount":10},
		{"name":"Vorana's Preparation","baseType":"Iron Flask","itemClass":3,"chaosValue":6000,"listingCount":10}]}`})
}

// progenesisBuild is a build whose main skill gains from extra chaos damage, with an active Progenesis in Flask 1.
func progenesisBuild(t *testing.T, s *Server) string {
	t.Helper()
	h := call(t, s, "import_build", map[string]any{"source": fixture("9.xml")})["handle"].(string)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "equip_unique", "name": "Progenesis", "slot": "Flask 1"},
		map[string]any{"op": "set_flask_active", "slot": "Flask 1", "active": true})})
	return h
}

// TestScanSavingsRanksCheaperFlask: rank_by savings keeps only rows that save chaos, with signed costs, Atziri's
// Promise for Progenesis among them, ranked by savings_per_pct.
func TestScanSavingsRanksCheaperFlask(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = progenesisBook(t)
	h := progenesisBuild(t, s)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"flasks"}, "rank_by": "savings", "per_slot": 5})
	got := rows(res)
	if len(got) == 0 {
		t.Fatalf("no savings rows: %v", res)
	}
	for _, r := range got {
		if c, ok := r["cost"].(float64); !ok || c >= 0 {
			t.Fatalf("row without a saving: %v", r)
		}
	}
	atziri := false
	for i, r := range got {
		if r["savings_per_pct"] == nil || r["loss_of"] == nil || (i > 0 && savingsTier(r["savings_per_pct"].(float64)) > savingsTier(got[i-1]["savings_per_pct"].(float64))) {
			t.Fatalf("row %d %v out of savings_per_pct order", i, r)
		}
		atziri = atziri || (strings.HasPrefix(r["name"].(string), "Atziri's Promise") && r["slot"] == "Flask 1" && r["cost"].(float64) == -5998)
	}
	if !atziri {
		t.Fatalf("no Atziri's Promise in Flask 1 at -5998: %v", got)
	}
	if res["cost_budget"].(map[string]any)["savings"] == nil {
		t.Error("summary does not define savings_per_pct")
	}
}

// TestScanGainPerDivPaysForItself: a row that gains and saves chaos is pays_for_itself, not a 1c cost, and ranks
// above every row that costs something.
func TestScanGainPerDivPaysForItself(t *testing.T) {
	rs := []map[string]any{
		{"name": "dear", "gain": 50.0, "gain_per_div": 900.0},
		{"name": "saving small", "gain": 1.0, "pays_for_itself": true},
		{"name": "saving big", "gain": 9.0, "pays_for_itself": true},
	}
	sortPerDiv(rs)
	if rs[0]["name"] != "saving big" || rs[1]["name"] != "saving small" {
		t.Fatalf("order %v", rs)
	}
	if v := perDivOf(1, -6000, 150); !math.IsInf(v, 1) {
		t.Errorf("perDivOf(1, -6000) = %v, want +Inf", v)
	}
	if v := perDivOf(10, 0.5, 150); v != 1500 {
		t.Errorf("perDivOf(10, 0.5) = %v, want 1500 (a cost under 1c counts as 1c)", v)
	}
	cb := &costBudget{div: 150}
	row := map[string]any{"cost": -5998.0}
	cb.putPerDiv(row, 12, "worst_gain", 100, 100, false)
	if row["pays_for_itself"] != true || row["gain_per_div"] != nil || row["gain_per_div_of"] != "worst_gain" {
		t.Errorf("putPerDiv on a saving that gains: %v", row)
	}
}

// TestScanBudgetForms: "+<n>div" caps what a change adds on top of the build; an absolute cap the build already breaks
// says so with the current price; fit_budget passes over savings with a big_loss.
func TestScanBudgetForms(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = progenesisBook(t)
	h := progenesisBuild(t, s)
	args := map[string]any{"handle": h, "parts": []string{"flasks"}, "cost_budget": "+1div"}
	sum := call(t, s, "scan_build", args)["cost_budget"].(map[string]any)
	build := sum["build_chaos"].(float64)
	if sum["budget_chaos"] != build+150 || sum["added_chaos"] != 150.0 || sum["over_budget"] != nil {
		t.Fatalf("+1div summary %v", sum)
	}

	args["cost_budget"] = build - 1000
	res := call(t, s, "scan_build", args)
	over, _ := res["cost_budget"].(map[string]any)["over_budget"].(string)
	if !strings.Contains(over, "already costs") || !strings.Contains(over, "savings") || !strings.Contains(res["hint"].(string), "already costs") {
		t.Fatalf("over an absolute cap: %v / %v", over, res["hint"])
	}

	// every flask that saves the cap's excess drops Progenesis's TotalEHP by a fifth or more: big_loss, passed over
	args["fit_budget"] = true
	res = call(t, s, "scan_build", args)
	plan := res["fit_budget"].(map[string]any)
	note, _ := plan["note"].(string)
	if skipped, _ := plan["skipped"].(map[string]any); plan["fits"] != false || len(plan["picks"].([]any)) != 0 || !strings.Contains(note, "can't fit") || skipped["big_loss"] == nil {
		t.Fatalf("fit_budget %v", plan)
	}
	if top := rows(res)[0]; !strings.Contains(fmt.Sprint(top["big_loss"]), "TotalEHP") {
		t.Fatalf("top savings row %v", top)
	}
	if e := callErr(t, s, "scan_build", map[string]any{"handle": h, "cost_budget": "+5div", "fit_budget": true}); e.Code != CodeInvalidArgument {
		t.Errorf("fit_budget with a relative cap: %s %s", e.Code, e.Message)
	}
}

// linksBook prices the bench's link currency and the build's support gems.
func linksBook(t *testing.T) *prices.Book {
	return bookOf(t, map[string]string{
		"Currency": `{"lines":[{"currencyTypeName":"Divine Orb","chaosEquivalent":150},{"currencyTypeName":"Orb of Fusing","chaosEquivalent":0.2}]}`,
		"SkillGem": `{"lines":[
			{"name":"Controlled Destruction Support","gemLevel":20,"gemQuality":20,"chaosValue":5,"listingCount":10},
			{"name":"Spell Echo Support","gemLevel":20,"gemQuality":20,"chaosValue":7,"listingCount":10},
			{"name":"Added Cold Damage Support","gemLevel":20,"gemQuality":20,"chaosValue":3,"listingCount":10},
			{"name":"Elemental Focus Support","gemLevel":20,"gemQuality":20,"chaosValue":4,"listingCount":10},
			{"name":"Hypothermia Support","gemLevel":20,"gemQuality":20,"chaosValue":2,"listingCount":10},
			{"name":"Freezing Pulse","gemLevel":20,"gemQuality":20,"chaosValue":1,"listingCount":10}]}`,
	})
}

// TestScanLinksSavings: a six-link rare body armour taken to five links saves the bench's link cost difference and the
// dropped support's price; fit_budget picks one to fit a cap a chaos under the build; applying the row unlinks the
// sixth socket and removes the support.
func TestScanLinksSavings(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = linksBook(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	body := "Rarity: RARE\nDoom Shell\nAstral Plate\nItem Level: 84\nSockets: B-B-B-B-B-B\nImplicits: 1\n+12% to all Elemental Resistances\n+90 to maximum Life"
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_text", "slot": "Body Armour", "text": body})})
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "slot": "Body Armour", "gems": []string{"Freezing Pulse", "Controlled Destruction", "Spell Echo", "Added Cold Damage", "Elemental Focus", "Hypothermia"}, "level": 20, "quality": 20},
		map[string]any{"op": "set_main", "group": 1})})
	res := call(t, s, "scan_build", map[string]any{"handle": h, "rank_by": "savings", "parts": []string{"links"}, "per_slot": 5})
	got := rows(res)
	if len(got) != 5 {
		t.Fatalf("links rows %v (empty: %v)", got, res["empty_parts"])
	}
	bench6, _ := benchLinkChaos(s.opts.Prices, 6)
	bench5, _ := benchLinkChaos(s.opts.Prices, 5)
	for _, r := range got {
		if r["part"] != "links" || r["cost"].(float64) >= -(bench6-bench5) {
			t.Fatalf("row %v, want a saving over the bench's %v", r, bench6-bench5)
		}
	}
	top := got[0]
	build := res["cost_budget"].(map[string]any)["build_chaos"].(float64)
	plan := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"links"}, "per_slot": 5, "cost_budget": build - 1, "fit_budget": true})["fit_budget"].(map[string]any)
	if picks, _ := plan["picks"].([]any); plan["fits"] != true || len(picks) != 1 || plan["loss_pct"] == nil {
		t.Fatalf("fit_budget %v", plan)
	}
	applied := call(t, s, "scan_build", map[string]any{"handle": h, "apply": top["id"]})
	if applied["revision"] == nil {
		t.Fatalf("apply %v: %v", top["id"], applied)
	}
	b := call(t, s, "get_build", map[string]any{"handle": h, "section": "items"})
	if txt := fmt.Sprint(b); !strings.Contains(txt, "B-B-B-B-B B") {
		t.Errorf("applied item is not five-linked: %v", txt)
	}
}

// TestOptimizeBuildCost: BuildCost bounds a search as a constraint and is an objective to minimize.
func TestOptimizeBuildCost(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = budgetBook(t, true)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse", "Controlled Destruction"}, "level": 20, "quality": 20}, map[string]any{"op": "set_main", "group": 1})})
	slots := []any{map[string]any{"kind": "gem", "group": 1, "gem": "Controlled Destruction", "levels": []int{20, 21}, "qualities": []int{20}, "corrupted": true}}
	choice := func(r map[string]any) string { return fmt.Sprint(r["choices"].([]any)[0].(map[string]any)["choice"]) }
	args := map[string]any{"handle": h, "slots": slots, "objective": map[string]any{"stat": "MainDPS"}, "top": 5}
	if c := choice(rows(call(t, s, "optimize", args))[0]); !strings.Contains(c, "level 21") {
		t.Fatalf("unconstrained best %v, want level 21", c)
	}
	args["constraints"] = []any{map[string]any{"stat": "BuildCost", "op": "<=", "value": 100}}
	got := rows(call(t, s, "optimize", args))
	if top := got[0]; !strings.Contains(choice(top), "level 20") || top["feasible"] != true {
		t.Fatalf("BuildCost <= 100: top %v", top)
	}
	for _, r := range got {
		if strings.Contains(choice(r), "level 21") && (r["feasible"] != false || r["cost"] != 895.0) {
			t.Fatalf("level 21 under BuildCost <= 100: %v", r)
		}
	}
	args["objective"] = map[string]any{"stat": "BuildCost", "goal": "maximize"}
	delete(args, "constraints")
	if top := rows(call(t, s, "optimize", args))[0]; !strings.Contains(choice(top), "level 21") {
		t.Fatalf("maximize BuildCost: top %v", top)
	}
	if e := callErr(t, s, "optimize", map[string]any{"handle": h, "slots": slots, "objective": map[string]any{"stat": "MainDPS"}, "rank_by": "savings"}); e.Code != CodeInvalidArgument || !strings.Contains(e.Message, "BuildCost") {
		t.Errorf("optimize rank_by savings: %s %s", e.Code, e.Message)
	}
}

// TestScanAnointCostUnknown: an anoint swap on a priced amulet needs oils, which have no price: its row names them in
// cost_unknown and shows no cost, not the amulet's unchanged price as 0.
func TestScanAnointCostUnknown(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = bookOf(t, map[string]string{"UniqueAccessory": `{"lines":[
		{"name":"Impresence","baseType":"Onyx Amulet","itemClass":3,"chaosValue":300,"listingCount":10}]}`})
	h := importFixture(t, s)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"anoints"}, "cost_budget": "1000div", "allow_unknown_price": true})
	got := rows(res)
	if len(got) == 0 {
		t.Fatalf("no anoint rows: %v", res["empty_parts"])
	}
	for _, r := range got {
		unknown := fmt.Sprint(r["cost_unknown"])
		if r["cost"] != nil || !strings.Contains(unknown, "oils for "+r["name"].(string)) {
			t.Fatalf("anoint row %v", r)
		}
	}
}

// TestScanUniquesCutByCount: under rank_by gain_per_div, a slot's uniques are cut by count, best gain first, not by
// time, so a scan returns the same rows however long it ran.
func TestScanUniquesCutByCount(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = bookOf(t, nil)
	h := importFixture(t, s)
	was := uniquesPerDivRows
	uniquesPerDivRows = 1
	t.Cleanup(func() { uniquesPerDivRows = was })
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"uniques"}, "slots": []string{"Helmet", "Amulet"}, "rank_by": "gain_per_div", "allow_unknown_price": true, "per_slot": 5})
	per := map[string]int{}
	for _, r := range rows(res) {
		per[r["slot"].(string)]++
	}
	if len(per) == 0 || per["Helmet"] > 1 || per["Amulet"] > 1 || res["partial"] != nil {
		t.Fatalf("rows per slot %v, partial %v", per, res["partial"])
	}
}
