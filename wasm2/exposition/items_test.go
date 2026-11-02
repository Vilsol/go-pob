package exposition

import (
	"os"
	"strings"
	"testing"
)

// A paste opens the editor, its save equips the item, and the next calculation and the saved build carry it, as
// PoB-PoE2's do (calculator2/testdata/items_edit_pob.jsonl, "paste helmet" and "body armour runes").
func TestItemsEditThroughTheCalculator(t *testing.T) {
	xml, err := os.ReadFile("../../calculator2/testdata/items_edit.xml")
	if err != nil {
		t.Fatal(err)
	}
	pastes, err := os.ReadFile("../../calculator2/testdata/items_edit_pastes.txt")
	if err != nil {
		t.Fatal(err)
	}
	helmet := strings.SplitN(string(pastes), "\n=====\n", 2)[0]
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if err := calc.PasteItem(helmet); err == nil {
		t.Fatal("an edit before the first calculation was accepted")
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if err := calc.PasteItem(helmet); err != nil {
		t.Fatal(err)
	}
	// the editor's item outlives the calculation it was opened over
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	page, err := calc.ItemsPage()
	if err != nil {
		t.Fatal(err)
	}
	if page.Display == nil || page.Display.ID != 0 || page.Display.SocketCount != 2 || len(page.Display.RuneChoices) < 2 {
		t.Fatalf("editor %+v; PoB-PoE2 opens a new item with 2 sockets", page.Display)
	}
	if _, err := calc.AddDisplayItem(""); err != nil {
		t.Fatal(err)
	}
	out, err := calc.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	if out.Output["Life"] != 718 || out.Output["ColdResist"] != -5 {
		t.Errorf("Life %v ColdResist %v, PoB-PoE2 718 -5", out.Output["Life"], out.Output["ColdResist"])
	}
	page, err = calc.ItemsPage()
	if err != nil {
		t.Fatal(err)
	}
	if page.Display != nil {
		t.Error("the editor stayed open after its item was added")
	}
	held := map[string]int{}
	for _, s := range page.Slots {
		held[s.Name] = s.ItemID
	}
	if held["Helmet"] != 4 || held["Body Armour"] != 1 {
		t.Errorf("slots %v; PoB-PoE2 puts item 4 in the Helmet", held)
	}
	if err := calc.EditItem(1); err != nil {
		t.Fatal(err)
	}
	if err := calc.SetDisplayItemSocketCount(2); err != nil {
		t.Fatal(err)
	}
	if err := calc.SetDisplayItemRune(1, "Desert Rune"); err != nil {
		t.Fatal(err)
	}
	if err := calc.SetDisplayItemRune(2, "Glacial Rune"); err != nil {
		t.Fatal(err)
	}
	if err := calc.SetDisplayItemRune(1, "No Such Rune"); err == nil {
		t.Error("a rune the item does not take was accepted")
	}
	if _, err := calc.AddDisplayItem(""); err != nil {
		t.Fatal(err)
	}
	if out, err = calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if out.Output["FireResist"] != -1 || out.Output["ColdResist"] != 9 {
		t.Errorf("FireResist %v ColdResist %v, PoB-PoE2 -1 9", out.Output["FireResist"], out.Output["ColdResist"])
	}
	saved, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(saved, "Rune: Desert Rune\nRune: Glacial Rune") {
		t.Error("the saved build lost the body armour's runes")
	}
	reopened, err := ParseBuild(saved)
	if err != nil {
		t.Fatal(err)
	}
	if out, err = reopened.Calculate(); err != nil {
		t.Fatal(err)
	}
	if out.Output["Life"] != 718 || out.Output["ColdResist"] != 9 {
		t.Errorf("reopened: Life %v ColdResist %v, PoB-PoE2 718 9", out.Output["Life"], out.Output["ColdResist"])
	}
}

// The item sets and the weapon set swap through the calculator, as PoB-PoE2's "new item set" and "weapon set II".
func TestItemSetsThroughTheCalculator(t *testing.T) {
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
	if err := calc.NewItemSet("Second"); err != nil {
		t.Fatal(err)
	}
	out, err := calc.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	if out.Output["Life"] != 557 || out.Output["Armour"] != 0 {
		t.Errorf("an empty set: Life %v Armour %v, PoB-PoE2 557 0", out.Output["Life"], out.Output["Armour"])
	}
	page, err := calc.ItemsPage()
	if err != nil {
		t.Fatal(err)
	}
	if page.ActiveItemSet != 3 || len(page.Sets) != 3 || page.Sets[2].Title != "Second" {
		t.Fatalf("sets %+v active %d; PoB-PoE2 makes set 3 active", page.Sets, page.ActiveItemSet)
	}
	if err := calc.SetUseSecondWeaponSet(true); err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if page, err = calc.ItemsPage(); err != nil {
		t.Fatal(err)
	}
	shown := map[string]bool{}
	for _, s := range page.Slots {
		shown[s.Name] = true
	}
	if !page.UseSecondWeaponSet || !shown["Weapon 1 Swap"] || shown["Weapon 1"] {
		t.Errorf("weapon set II shows %v", shown)
	}
	if err := calc.DeleteItemSet(3); err != nil {
		t.Fatal(err)
	}
	if err := calc.DeleteItemSet(2); err != nil {
		t.Fatal(err)
	}
	if err := calc.DeleteItemSet(1); err == nil {
		t.Error("the last item set was deleted")
	}
}
