package calculator

import (
	"testing"
)

// TestNormaliseQualityMatchesThePin pins Item.lua:1644-1652.
//
//	elseif not uniqueID and not corrupted       ... and not split and not mirrored
func TestNormaliseQualityMatchesThePin(t *testing.T) {
	requireGameData(t, "3.29")

	const armourBase = "Rarity: NORMAL\nTest Plate\nGlorious Plate\n"

	t.Run("no quality line defaults to 0, not 20", func(t *testing.T) {
		i := ParseItem(armourBase)
		if i == nil || i.Base == nil {
			t.Skip("fixture base not present in the loaded data")
		}
		if i.Quality == nil || *i.Quality != 0 {
			t.Errorf("quality = %d, want 0 (Item.lua:1647 sets 0 unconditionally; v2.18.1 set 20 "+
				"for a non-corrupted item and that is what this port did)", derefQ(i.Quality))
		}
	})

	t.Run("a quality below 20 is still raised to 20", func(t *testing.T) {
		i := ParseItem(armourBase)
		if i == nil || i.Base == nil {
			t.Skip("fixture base not present in the loaded data")
		}
		q := 5
		i.Quality = &q
		i.NormaliseQuality()
		if derefQ(i.Quality) != 20 {
			t.Errorf("quality = %d, want 20 -- the elseif arm is unchanged and must not regress",
				derefQ(i.Quality))
		}
	})

	t.Run("split and mirrored block the raise", func(t *testing.T) {
		for _, set := range []func(*Item){
			func(it *Item) { it.Split = true },
			func(it *Item) { it.Mirrored = true },
		} {
			i := ParseItem(armourBase)
			if i == nil || i.Base == nil {
				t.Skip("fixture base not present in the loaded data")
			}
			q := 5
			i.Quality = &q
			set(i)
			i.NormaliseQuality()
			if derefQ(i.Quality) != 5 {
				t.Errorf("quality = %d, want 5 (Item.lua:1648 excludes split and mirrored items "+
					"from the raise; v2.18.1 excluded neither)", derefQ(i.Quality))
			}
		}
	})
}

func derefQ(p *int) int {
	if p == nil {
		return -1
	}
	return *p
}
