package calculator

import (
	"testing"
)

// TestUnknownBraceTagsDoNotEatTheModLine pins Item.lua:849-876.
func TestUnknownBraceTagsDoNotEatTheModLine(t *testing.T) {
	requireGameData(t, "3.29")

	const base = "Rarity: RARE\nTest Item\nVaal Regalia\n"

	mods := func(t *testing.T, text string) []string {
		t.Helper()
		i := ParseItem(text)
		if i == nil {
			t.Fatal("REFUSING: the item did not parse at all")
		}
		var out []string
		for _, ml := range i.ExplicitModLines {
			for _, m := range ml.ModList {
				out = append(out, m.Name())
			}
			if ml.Extra != "" {
				t.Logf("leftover Extra on %q: %q", ml.Line, ml.Extra)
			}
		}
		return out
	}

	control := mods(t, base+"{range:0.5}+(50-75)% to Fire Resistance\n")
	if !contains(control, "FireResist") {
		t.Fatalf("REFUSING: the control line produced %v, not FireResist; the fixture cannot "+
			"attribute anything", control)
	}

	for _, tag := range []string{
		"{modGroup:FireResistUniqueBodyInt2}",
		"{corruptedRange:1.22}",
	} {
		line := base + tag + "{range:0.5}+(50-75)% to Fire Resistance\n"
		if got := mods(t, line); !contains(got, "FireResist") {
			t.Errorf("with %s the line produced %v, want FireResist; Item.lua:849 strips every "+
				"brace tag with one gsub, so an unknown one must not cost the whole mod", tag, got)
		}
	}

	if got := mods(t, base+"{notARealTag:7}+(50-75)% to Fire Resistance\n"); !contains(got, "FireResist") {
		t.Errorf("an unrecognised tag produced %v, want FireResist; the pin's gsub consumes the "+
			"tag whether or not it means anything", got)
	}

	if got := mods(t, base+"+(50-75)% to Fire Resistance\n"); !contains(got, "FireResist") {
		t.Errorf("a plain line produced %v; the strip must be a no-op when there is no tag", got)
	}
}
