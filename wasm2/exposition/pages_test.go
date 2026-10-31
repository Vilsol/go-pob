package exposition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type pagesOracle struct {
	Groups []struct {
		Label            string `json:"label"`
		Slot             string `json:"slot"`
		Source           string `json:"source"`
		Enabled          bool   `json:"enabled"`
		IncludeInFullDPS bool   `json:"includeInFullDPS"`
		Gems             []struct {
			Name    string  `json:"name"`
			Level   int     `json:"level"`
			Quality float64 `json:"quality"`
			Count   float64 `json:"count"`
			Enabled bool    `json:"enabled"`
			Support bool    `json:"support"`
			Color   string  `json:"color"`
		} `json:"gems"`
	} `json:"groups"`
	UseSecondWeaponSet bool `json:"useSecondWeaponSet"`
	Items              []struct {
		Slot   string     `json:"slot"`
		ID     int        `json:"id"`
		Rarity string     `json:"rarity"`
		Name   []string   `json:"name"`
		Lines  [][]string `json:"lines"`
	} `json:"items"`
}

func calculatedFixture(t *testing.T, name string) *Calculator {
	t.Helper()
	xml, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	return calc
}

func readPagesOracle(t *testing.T, name string) pagesOracle {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", name))
	if err != nil {
		t.Fatal(err)
	}
	var oracle pagesOracle
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatal(err)
	}
	return oracle
}

// The groups are PoB-PoE2's skills tab for the fixture (oracle/poe2_pages.lua, `task calc-sections-poe2`).
func TestSkillGroupsMatchPoB(t *testing.T) {
	oracle := readPagesOracle(t, "poe2_pages_sorceress_items.json")
	got, err := calculatedFixture(t, "sorceress_items.xml").SkillGroups()
	if err != nil {
		t.Fatal(err)
	}
	want := make([]SkillGroupView, len(oracle.Groups))
	for i, g := range oracle.Groups {
		want[i] = SkillGroupView{Label: g.Label, Slot: g.Slot, Source: g.Source, Enabled: g.Enabled, IncludeInFullDPS: g.IncludeInFullDPS, Set1: true, Set2: true}
		for _, gem := range g.Gems {
			want[i].Gems = append(want[i].Gems, SkillGemView{Name: gem.Name, Level: gem.Level, Quality: gem.Quality, Count: gem.Count, Enabled: gem.Enabled, Support: gem.Support, Color: gem.Color})
		}
	}
	if len(want) < 2 {
		t.Fatalf("the oracle lists %d groups; the fixture has two", len(want))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("skill groups:\n ours %+v\n  PoB %+v", got, want)
	}
}

// The items are PoB-PoE2's items tab for the fixture: the shown slots that hold an item, and each item's
// tooltip title and mod lines (oracle/poe2_pages.lua).
func TestEquippedItemsMatchPoB(t *testing.T) {
	oracle := readPagesOracle(t, "poe2_pages_sorceress_items.json")
	got, err := calculatedFixture(t, "sorceress_items.xml").EquippedItems()
	if err != nil {
		t.Fatal(err)
	}
	want := &ItemsView{UseSecondWeaponSet: oracle.UseSecondWeaponSet}
	for _, item := range oracle.Items {
		want.Items = append(want.Items, ItemView{Slot: item.Slot, ID: item.ID, Rarity: item.Rarity, Name: item.Name, Lines: item.Lines})
	}
	if !want.UseSecondWeaponSet || len(want.Items) < 4 {
		t.Fatalf("the oracle shows weapon set 2 %v and %d items; the fixture uses set 2 with four", want.UseSecondWeaponSet, len(want.Items))
	}
	if got.UseSecondWeaponSet != want.UseSecondWeaponSet || len(got.Items) != len(want.Items) {
		t.Fatalf("weapon set 2 %v, %d items; PoB-PoE2 %v, %d", got.UseSecondWeaponSet, len(got.Items), want.UseSecondWeaponSet, len(want.Items))
	}
	for i := range want.Items {
		if !reflect.DeepEqual(got.Items[i], want.Items[i]) {
			t.Errorf("item %d:\n ours %+v\n  PoB %+v", i, got.Items[i], want.Items[i])
		}
	}
}

// Reading the Calcs, Skills and Items pages leaves the saved build as it was; choosing the main group
// changes the mainSocketGroup attribute alone.
func TestPagesLeaveTheSaveUnchanged(t *testing.T) {
	for _, name := range []string{"sorceress_items.xml", "sorceress_spark.xml", "sorceress_three_spells.xml"} {
		calc := calculatedFixture(t, name)
		before, err := calc.Serialize()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := calc.SkillGroups(); err != nil {
			t.Fatal(err)
		}
		if _, err := calc.EquippedItems(); err != nil {
			t.Fatal(err)
		}
		if _, err := calc.CalcsMods(CalcsModQuery{Entries: map[string]CalcsModEntry{"Life": {ModName: []string{"Life"}, ModType: "BASE", ModSource: "Item"}}}); err != nil {
			t.Fatal(err)
		}
		if _, err := calc.Calculate(); err != nil {
			t.Fatal(err)
		}
		after, err := calc.Serialize()
		if err != nil {
			t.Fatal(err)
		}
		if after != before {
			t.Errorf("%s: the pages changed the save:\n%s\n---\n%s", name, before, after)
		}
		if name != "sorceress_items.xml" {
			continue
		}
		calc.SetMainSocketGroup(2)
		moved, err := calc.Serialize()
		if err != nil {
			t.Fatal(err)
		}
		if want := strings.Replace(before, `mainSocketGroup="1"`, `mainSocketGroup="2"`, 1); moved != want || moved == before {
			t.Errorf("%s: choosing the main group wrote more than mainSocketGroup:\n%s", name, moved)
		}
	}
}
