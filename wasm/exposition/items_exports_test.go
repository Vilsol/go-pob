package exposition

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
)

// TestItemSetTooltips: ItemsTab.lua:4138-4147: each of 0339's sets gets its own lines, one per filled slot, "^7<label>: <colour><name>".
func TestItemSetTooltips(t *testing.T) {
	initItemData(t)
	build := corpusBuild(t, "0339-reliquarian-shield-crush-of-the-chieftain-2.xml")
	tips := ItemSetTooltips(build)
	if len(tips) != len(build.Items.ItemSets) || len(tips) < 2 {
		t.Fatalf("%d tooltips for %d sets", len(tips), len(build.Items.ItemSets))
	}
	for k, tip := range tips {
		if tip.ID != build.Items.ItemSets[k].ID || len(tip.Lines) == 0 {
			t.Fatalf("set %d: id %q, %d lines", k, tip.ID, len(tip.Lines))
		}
		for _, l := range tip.Lines {
			if !strings.HasPrefix(l, "^7") || !strings.Contains(l, ": ^x") {
				t.Fatalf("set %s: %q", tip.ID, l)
			}
		}
	}
	if strings.Join(tips[0].Lines, "\n") == strings.Join(tips[1].Lines, "\n") {
		t.Fatal("two sets draw the same tooltip")
	}
}

// TestShowFlavourTextOption: Main.lua:1020-1024 / Tooltip.lua:227-262: with the option off a unique's tooltip has no flavour text, and on it has.
func TestShowFlavourTextOption(t *testing.T) {
	initItemData(t)
	defer SetShowFlavourText(true)
	lines := func() int {
		tip, ok := ItemDBTooltip(nil, "UNIQUE", "Kaom's Heart, Glorious Plate", calculator.ItemTooltipStats{})
		if !ok {
			t.Fatal("no Kaom's Heart in the unique DB")
		}
		return len(tip.Lines)
	}
	on := lines()
	SetShowFlavourText(false)
	if off := lines(); off >= on {
		t.Fatalf("flavour text off: %d lines, on: %d", off, on)
	}
}
