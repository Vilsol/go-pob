package calculator2

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// CalcSetup.lua:543-548, 2186 and CalcActiveSkill.lua:810 (PoB-PoE2): a gem level an item grants keeps the item's
// source in the skill's mod list, so the Gem Level cell's mod table names the item. The rows are
// oracle/poe2_mod_sections.lua's over testdata/gem_global_level.xml.
func TestModSectionGemGlobalLevelNamesItsItem(t *testing.T) {
	raw, err := os.ReadFile("testdata/gem_global_level.xml")
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	if err != nil {
		t.Fatal(err)
	}
	got := env.ModSection(ModSectionData{ModName: []string{"GemGlobalLevel"}, Multi: true, Cfg: "skill"})
	for i := range got {
		got[i].ItemID = 0
	}
	totals := []string{"Total from Item:", "+3 base Gem Global Level"}
	want := []ModSectionRow{
		{Value: "+2 base", Name: "Gem Global Level", Source: "Item", SourceName: "^xFFFF77Gem Clasp, Solar Amulet", Totals: totals},
		{Value: "+1 base", Name: "Gem Global Level", Source: "Item", SourceName: "^xFFFF77Gem Clasp, Solar Amulet", Totals: totals},
	}
	g, _ := json.Marshal(got)
	w, _ := json.Marshal(want)
	if string(g) != string(w) {
		t.Errorf("Gem Global Level mods:\n PoB-PoE2 %s\n ours     %s", w, g)
	}
}
