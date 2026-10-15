package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

func forbiddenJewels(build *pob.PathOfBuilding, title string) map[int]string {
	out := map[int]string{}
	for _, it := range build.Items.Items {
		if item := parseItemGlobal(it.Text); item.Title == title {
			out[it.ID] = selectedVariantName(item)
		}
	}
	return out
}

// TestForbiddenJewelCounterpart: ItemsTab.lua:1728-1801 (#10235).
func TestForbiddenJewelCounterpart(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	DeleteAllItems(build)
	count := func() int { return len(build.Items.Items) }
	before := count()
	flameID, ok := AddDBItem(build, ItemDBUnique, "Forbidden Flame, Crimson Jewel", false, false)
	if !ok {
		t.Fatal("no Forbidden Flame in the unique DB")
	}
	flame := forbiddenJewels(build, "Forbidden Flame")[flameID]
	flesh := forbiddenJewels(build, "Forbidden Flesh")
	if count() != before+2 || len(flesh) != 1 || flame == "" {
		t.Fatalf("items %d -> %d, Flesh %v, Flame variant %q", before, count(), flesh, flame)
	}
	for _, v := range flesh {
		if v != flame {
			t.Fatalf("Flesh variant %q, want the Flame's %q", v, flame)
		}
	}
	if _, ok := AddDBItem(build, ItemDBUnique, "Forbidden Flame, Crimson Jewel", false, false); !ok || count() != before+3 {
		t.Fatalf("a second Flame of the same notable: items %d, want %d", count(), before+3)
	}

	text, _ := buildItemText(build, flameID)
	edited := ParseItem(text)
	other := ""
	for k, name := range edited.VariantList {
		if name != flame && name != "" {
			other = name
			edited.Variant = new(k + 1)
			break
		}
	}
	if !GetMiscCalculator(build).ReplaceItem(flameID, buildRawText(edited)) {
		t.Fatal("ReplaceItem found no item")
	}
	flesh = forbiddenJewels(build, "Forbidden Flesh")
	if count() != before+3 || len(flesh) != 1 {
		t.Fatalf("after the edit: items %d, Flesh %v", count(), flesh)
	}
	for _, v := range flesh {
		if v != other {
			t.Fatalf("Flesh variant %q after the edit, want %q", v, other)
		}
	}

	before = count()
	if _, ok := AddDBItem(build, ItemDBUnique, "Watcher's Eye, Prismatic Jewel", false, false); !ok || count() != before+1 {
		t.Fatalf("a jewel with no counterpart: items %d -> %d", before, count())
	}
}
