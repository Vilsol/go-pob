package exposition

import (
	"encoding/json"
	"fmt"
	"html"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
)

// spec/System/TestImport_spec.lua "keeps Wickerman items separate through import and save/load"
func TestImportWickermanItemsMatchPoB(t *testing.T) {
	if raw.LatestVersion != "3.30" {
		t.Skipf("NOT-RUN: Viscera bases exist from 3.30 game data; the process loaded %s (POE_VERSION=3.30)", raw.LatestVersion)
	}
	initItemData(t)
	examples := []struct{ typeLine, base, slot, implicit, inventoryID string }{
		{"Ravenous Skull", "Ravenous Skull", "Head", "Every 3 seconds, Consume a Corpse within 5 metres to Recover 4% of Life", ""},
		{"Lesser Warden's Arm", "Warden's Arm", "Arm", "+43 to maximum Life", ""},
		{"Greater Sirus' Head", "Sirus' Head", "Head", "+10 to Strength", ""},
		{"Lesser Blackened Lungs", "Blackened Lungs", "Body", "12% increased Warcry Speed", "VisceraBody"},
		{"Lesser Pocked Leg", "Pocked Leg", "Leg", "20% increased Armour from Equipped Helmet and Gloves", "VisceraLeg"},
		{"Artificial Appendage", "Artificial Appendage", "Arm", "25% increased Effect of Chill", ""},
		{"Artificial Appendage", "Artificial Appendage", "Arm", "18% increased Area of Effect for Attacks", ""},
		{"Artificial Appendage", "Artificial Appendage", "Arm", "Minions have 30% increased Critical Strike Chance", ""},
	}
	items := make([]map[string]any, 0, len(examples))
	for i, e := range examples {
		inv := e.inventoryID
		if inv == "" {
			inv = map[bool]string{true: "VisceraHead", false: "VisceraArm"}[e.slot == "Head"]
		}
		items = append(items, map[string]any{"id": fmt.Sprintf("synthetic-wickerman-%d", i+1), "name": "", "typeLine": e.typeLine, "baseType": e.typeLine,
			"frameType": 15, "ilvl": 83, "inventoryId": inv, "implicitMods": []map[string]string{{"description": e.implicit}}})
	}
	char, err := json.Marshal(map[string]any{"character": map[string]any{"level": 80, "equipment": []any{}, "passives": map[string]any{}, "wickerman": items}})
	if err != nil {
		t.Fatal(err)
	}
	build := builds.EmptyBuild()
	if _, err := ImportItemsAndSkills(build, string(char), true, false, false); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportItemsAndSkills(build, string(char), false, false, false); err != nil {
		t.Fatal(err)
	}
	saved, err := builds.SerializeBuild(build)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := builds.ParseBuild(saved)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Items.Items) != len(examples) {
		t.Fatalf("%d items after save/load, PoB %d", len(loaded.Items.Items), len(examples))
	}
	for i, e := range examples {
		item := calculator.ParseItem(html.UnescapeString(loaded.Items.Items[i].Text))
		var line string
		if len(item.ImplicitModLines) > 0 {
			line = item.ImplicitModLines[0].Line
		}
		if item.UniqueID != fmt.Sprintf("synthetic-wickerman-%d", i+1) || item.BaseName != e.base || item.Rarity != calculator.RarityNormal ||
			item.Type != "Viscera" || line != e.implicit {
			t.Errorf("item %d: id %q base %q rarity %s type %q implicit %q; PoB %q NORMAL Viscera %q", i+1, item.UniqueID, item.BaseName, item.Rarity, item.Type, line, e.base, e.implicit)
		}
	}
	equipped := map[string]int{}
	for _, s := range loaded.Items.ItemSets[0].Slots {
		equipped[s.Name] = s.ItemID
	}
	for _, slot := range []string{"Head", "Body", "Arm", "Leg"} {
		if equipped[slot] == 0 {
			t.Errorf("nothing equipped in the Wickerman %s slot: %v", slot, equipped)
		}
	}
}
