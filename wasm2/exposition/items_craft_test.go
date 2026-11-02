package exposition

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator2"
)

// Create in the Craft Item popup opens the base's item in the editor, as PoB-PoE2's does
// (calculator2/testdata/item_craft_pob.jsonl, the Doom Grip row).
func TestCraftDisplayItem(t *testing.T) {
	xml, err := os.ReadFile("../../calculator2/testdata/items_edit.xml")
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
	if err := calc.CraftDisplayItem(3, "Doom Grip", "No Such Base"); err == nil {
		t.Fatal("an unknown base was crafted")
	}
	if err := calc.CraftDisplayItem(3, "Doom Grip", "Adherent Cuffs"); err != nil {
		t.Fatal(err)
	}
	page, err := calc.ItemsPage()
	if err != nil {
		t.Fatal(err)
	}
	if page.Display == nil || page.Display.ID != 0 || page.Display.Rarity != "RARE" || page.Display.SocketCount != 3 ||
		!strings.HasPrefix(page.Display.Text, "Doom Grip\nAdherent Cuffs\n") {
		t.Fatalf("editor %+v; PoB-PoE2 opens a rare Doom Grip with 3 sockets", page.Display)
	}
	groups := CraftBases()
	if len(groups) == 0 || groups[0].Type != "Amulet" || groups[0].Bases[0] != "Absent Amulet" {
		t.Fatalf("craft bases start %+v; PoB-PoE2's start with Amulet: Absent Amulet", groups[:1])
	}
}

// Ctrl+click on a unique database row adds it and equips it in its primary slot, the weapon swap set's while that set
// is in use (ItemDBControl.lua:348-373); a double click opens it in the editor.
func TestAddDBItem(t *testing.T) {
	xml, err := os.ReadFile("../../calculator2/testdata/items_edit.xml")
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
	rows, err := calc.ItemDBRows("UNIQUE", calculator2.ItemDBFilter{Slot: 10, Search: "headhunter"})
	if err != nil || len(rows) != 1 || rows[0].Name != "Headhunter, Heavy Belt" || rows[0].PrimarySlot != "Belt" {
		t.Fatalf("rows %+v err %v; one Headhunter belt", rows, err)
	}
	slotOf := func(id int) string {
		page, err := calc.ItemsPage()
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range page.Slots {
			if s.ItemID == id {
				return s.Name
			}
		}
		return ""
	}
	id, err := calc.AddDBItem("UNIQUE", rows[0].Name, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := slotOf(id); got != "Belt" {
		t.Errorf("Headhunter equipped in %q", got)
	}
	if err := calc.SetUseSecondWeaponSet(true); err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	id, err = calc.AddDBItem("UNIQUE", "Lifesprig, Attuned Wand", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := slotOf(id); got != "Weapon 1 Swap" {
		t.Errorf("a wand added in weapon set II went to %q", got)
	}
	id, err = calc.AddDBItem("UNIQUE", "Lifesprig, Attuned Wand", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := slotOf(id); got != "" {
		t.Errorf("Add without equip equipped it in %q", got)
	}
	if err := calc.OpenDBItem("UNIQUE", "Lifesprig, Attuned Wand"); err != nil {
		t.Fatal(err)
	}
	page, err := calc.ItemsPage()
	if err != nil {
		t.Fatal(err)
	}
	if page.Display == nil || page.Display.Rarity != "UNIQUE" || !strings.HasPrefix(page.Display.Text, "Lifesprig\nAttuned Wand\n") {
		t.Errorf("editor %+v", page.Display)
	}
}

// The editor's item's stat differences are one block per slot it fits, as PoB-PoE2's tooltip shows them
// (calculator2/testdata/item_compare_pob.jsonl.gz, "paste helmet": +42 Total Life in Helmet).
func TestDisplayItemCompare(t *testing.T) {
	xml, err := os.ReadFile("../../calculator2/testdata/items_edit.xml")
	if err != nil {
		t.Fatal(err)
	}
	pastes, err := os.ReadFile("../../calculator2/testdata/items_edit_pastes.txt")
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
	if err := calc.PasteItem(strings.SplitN(string(pastes), "\n=====\n", 2)[0]); err != nil {
		t.Fatal(err)
	}
	v, err := calc.DisplayItemCompare()
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Blocks) != 1 || v.Blocks[0].Slot != "Helmet" || v.Blocks[0].Header != "Equipping this item in Helmet will give you:" {
		t.Fatalf("blocks %+v", v.Blocks)
	}
	if got := v.Blocks[0].Compare.Output["Life"] - v.Base.Output["Life"]; got != 42 {
		t.Errorf("Life +%v, PoB-PoE2 +42", got)
	}
}
