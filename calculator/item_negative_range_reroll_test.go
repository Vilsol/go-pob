package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestNegativeRangeRerollsAtStoredRange pins Item.lua:2471's test.
func TestNegativeRangeRerollsAtStoredRange(t *testing.T) {
	item := ParseItem("Rarity: RARE\nFixture\nAmethyst Ring\nImplicits: 0\n{tags:mana}{prefix}{custom}{crafted}{range:0}Non-Channelling Skills have +(-7--6) to Total Mana Cost\n")
	if got := sumItemMod(item, "ManaCost", mod.TypeBase); got != -7 {
		t.Errorf("ManaCost BASE = %v, want -7 ({range:0} of -7..-6)", got)
	}
}
