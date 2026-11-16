package mcp

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
)

// TestFoulbornUniques: a Foulborn unique is named as PoB titles it, "Foulborn <unique>", wherever a unique is named.
func TestFoulbornUniques(t *testing.T) {
	offline(t)
	s := testServer(t)
	h := importFixture(t, s)
	// Kikazaru has one Foulborn mutation, its life regeneration per level into mana per level
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_unique", "name": "Foulborn Kikazaru", "slot": "Ring 1"})})
	text := slotText(t, s, h, "Ring 1")
	if it := calculator.ParseItem(text); it.Title != "Foulborn Kikazaru" || !strings.Contains(text, "{mutated}+2 Maximum Mana per Level") {
		t.Fatalf("Foulborn Kikazaru:\n%s", text)
	}
	// Rathpith Globe has two: the name must pick one
	e := callErr(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_unique", "name": "Foulborn Rathpith Globe", "slot": "Weapon 2"})})
	if e.Code != CodeAmbiguous || len(e.Candidates) != 2 {
		t.Fatalf("two mutations: %s %v", e.Code, e.Candidates)
	}
	pick := e.Candidates[1].(string)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_unique", "name": pick, "slot": "Weapon 2"})})
	if text := slotText(t, s, h, "Weapon 2"); !strings.Contains(text, "{mutated}"+pick[strings.Index(pick, ": ")+2:]) {
		t.Fatalf("%s:\n%s", pick, text)
	}
	res := call(t, s, "search_data", map[string]any{"kind": "unique", "query": "Foulborn Kikazaru"})
	rows := res["results"].([]any)
	if len(rows) == 0 || rows[0].(map[string]any)["name"] != "Kikazaru, Topaz Ring" || rows[0].(map[string]any)["foulborn"] != true {
		t.Fatalf("search: %v", rows)
	}
	book := priceSnapshot(t, true)
	if p := PriceItem(book, calculator.PeekItem(text)); p.Chaos == nil || *p.Chaos != 70 {
		t.Errorf("Foulborn Kikazaru priced %v %q, want its own listing's 70", deref(p.Chaos), p.Detail)
	}
	if !slices.Contains(uniqueNames(), "Kikazaru, Topaz Ring") {
		t.Fatal("no Kikazaru")
	}
}

// TestFoulbornSearchAgreesWithEquip: search marks a unique Foulborn exactly when it has a line with a Foulborn
// counterpart (PoB's ItemDBControl), and every one it marks equips. Astramentis mutates into a line that rolls to 0 at
// the default range, Timeclasp mutates a line that rolls to 0 (both hidden from the display); Might of the Meek's
// mutation is no mod line in PoB.
func TestFoulbornSearchAgreesWithEquip(t *testing.T) {
	offline(t)
	s := testServer(t)
	h := importFixture(t, s)
	for _, c := range []struct {
		query, slot, mutated string
		foulborn             bool
	}{
		{"Astramentis", "Amulet", "{mutated}+(-13-13)% to Chaos Resistance", true},
		{"Timeclasp", "Ring 1", "{mutated}Debuffs on you expire (-20-20)% slower", true},
		{"Might of the Meek", "", "", false},
	} {
		rows := call(t, s, "search_data", map[string]any{"kind": "unique", "query": "Foulborn " + c.query})["results"].([]any)
		var flagged bool
		for _, r := range rows {
			if m := r.(map[string]any); strings.HasPrefix(m["name"].(string), c.query+",") {
				flagged = m["foulborn"] == true
			}
		}
		if flagged != c.foulborn {
			t.Errorf("%s: search foulborn %v, want %v", c.query, flagged, c.foulborn)
		}
		if !c.foulborn {
			continue
		}
		_, e := callRaw(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_unique", "name": "Foulborn " + c.query, "slot": c.slot})})
		if e != nil {
			t.Errorf("%s: search marks it Foulborn but equip fails: %s", c.query, e.Message)
			continue
		}
		if text := slotText(t, s, h, c.slot); !strings.Contains(text, c.mutated) || strings.Contains(text, "{range:1}") {
			t.Errorf("%s:\n%s", c.query, text)
		}
	}
}
