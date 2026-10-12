package calculator

import (
	"testing"
)

// TestBuffEffectModSiblings pins CalcOffence.lua:1206-1217.
func TestBuffEffectModSiblings(t *testing.T) {
	t.Run("a herald's buff effect", func(t *testing.T) {
		assertBuildStats(t, "1180-elementalist-herald-of-thunder-2.xml",
			[]string{"HeraldBuffEffectMod"}, nil)
	})

	t.Run("a link skill's effect", func(t *testing.T) {
		assertBuildStats(t, "0427-luminary-flame-link-2.xml",
			[]string{"LinkEffectMod"}, nil)
	})
}
