package exposition

import (
	"slices"
	"strings"
	"testing"
)

func TestCopyCompareSpec(t *testing.T) {
	primary := calculatedFixture(t, "sorceress_spark.xml")
	compared := calculatedFixture(t, "tree_specs.xml")
	if err := primary.CopyCompareSpec(nil, false); err == nil {
		t.Fatal("copying from no build succeeded")
	}
	want, err := compared.specAt(compared.activeSpecIndex())
	mustOK(t, err)

	mustOK(t, primary.CopyCompareSpec(compared, false))
	specs := primary.TreeSpecs()
	if got := specs.Labels; len(got) != 2 || got[1] != "Radius (Compared)" {
		t.Fatalf("tree list %q, want the compared spec added as \"Radius (Compared)\"", got)
	}
	if specs.Active != 1 {
		t.Fatalf("active spec %d, want the primary's own still active", specs.Active)
	}
	got, err := primary.specAt(2)
	mustOK(t, err)
	if got.Save().Nodes != want.Save().Nodes || got.CurClassName != want.CurClassName {
		t.Fatalf("copied spec %s %q, want %s %q", got.CurClassName, got.Save().Nodes, want.CurClassName, want.Save().Nodes)
	}

	mustOK(t, primary.CopyCompareSpec(compared, true))
	if specs := primary.TreeSpecs(); specs.Active != 3 {
		t.Fatalf("active spec %d after copy and use, want 3", specs.Active)
	}
	out := recalc(t, primary)
	if out.ClassName != "Warrior" {
		t.Fatalf("class %q after using the copied spec, want Warrior", out.ClassName)
	}
}

func TestCopyCompareItem(t *testing.T) {
	primary := calculatedFixture(t, "sorceress_spark.xml")
	compared := calculatedFixture(t, "warrior_dual_mace.xml")
	before := len(primary.items.Items)

	if _, err := primary.CopyCompareItem(nil, "Weapon 1", false); err == nil {
		t.Fatal("copying from no build succeeded")
	}
	if ok, err := primary.CopyCompareItem(compared, "Helmet", true); err != nil || ok {
		t.Fatalf("an empty slot copied (%v, %v)", ok, err)
	}
	ok, err := primary.CopyCompareItem(compared, "Weapon 1", false)
	mustOK(t, err)
	if !ok || len(primary.items.Items) != before+1 {
		t.Fatalf("copied %v, %d items, want one more than %d", ok, len(primary.items.Items), before)
	}
	if id := primary.items.Slots["Weapon 1"].SelItemID; id != 0 {
		t.Fatalf("Weapon 1 holds item %d after a copy without use", id)
	}

	mustOK(t, func() error { _, err := primary.CopyCompareItem(compared, "Weapon 1", true); return err }())
	recalc(t, primary)
	equipped, err := primary.EquippedItems()
	mustOK(t, err)
	theirs, err := compared.EquippedItems()
	mustOK(t, err)
	at := func(items []ItemView, slot string) []string {
		i := slices.IndexFunc(items, func(v ItemView) bool { return v.Slot == slot })
		if i < 0 {
			return nil
		}
		return items[i].Name
	}
	if got, want := at(equipped.Items, "Weapon 1"), at(theirs.Items, "Weapon 1"); want == nil || strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("Weapon 1 holds %q, want the compared build's %q", got, want)
	}
}
