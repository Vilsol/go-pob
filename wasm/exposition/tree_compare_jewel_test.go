package exposition

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

// TestCompareSpecJewelTooltipReadsTheComparedTree: 0008's active tree holds item 5 in socket 2311.
func TestCompareSpecJewelTooltipReadsTheComparedTree(t *testing.T) {
	initItemData(t)
	build := corpusBuild(t, "0008-necromancer-vaal-summon-skeletons.xml")
	active := build.ActiveSpecIndex()
	if active < 0 || !slices.Contains(build.Tree.Specs[active].Sockets, pob.Socket{NodeID: 2311, ItemID: 5}) {
		t.Fatal("REFUSING: 0008's active tree does not hold item 5 in 2311")
	}
	var text string
	for _, it := range build.Items.Items {
		if it.ID == 5 {
			text = it.Text
		}
	}
	if text == "" {
		t.Fatal("REFUSING: no item 5")
	}
	want := calculator.AddItemTooltip(calculator.ParseItem(text), calculator.ItemTooltipContext{ShowFlavourText: showFlavourText, CharacterLevel: build.Build.Level, ClassName: string(build.Build.ClassName)})
	got, ok := CompareSpecJewelTooltip(build, active+1, 2311, calculator.ItemTooltipStats{})
	if !ok || !slices.Equal(got.Title, want.Title) || len(got.Lines) != len(want.Lines) {
		t.Fatalf("tree %d socket 2311 = %v %v, want item 5's %v", active+1, ok, got.Title, want.Title)
	}
	empty := build.Tree.Specs[active]
	empty.Sockets = []pob.Socket{}
	build.Tree.Specs = append(build.Tree.Specs, empty)
	if _, ok := CompareSpecJewelTooltip(build, len(build.Tree.Specs), 2311, calculator.ItemTooltipStats{}); ok {
		t.Error("the added tree has no jewel in 2311")
	}
	for _, id := range []int{0, len(build.Tree.Specs) + 1} {
		if _, ok := CompareSpecJewelTooltip(build, id, 2311, calculator.ItemTooltipStats{}); ok {
			t.Errorf("no tree %d", id)
		}
	}
}
