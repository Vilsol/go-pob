package calculator

import (
	"slices"
	"strings"
	"testing"
)

// TestBaseEnchantAppend: Item.lua:1632-1657, ItemsTab.lua:1267-1328. Values from PoB parsing the same texts and
// loading the same build item.
func TestBaseEnchantAppend(t *testing.T) {
	initItemsTabData(t)
	lines := func(l []ModLine) []string {
		out := make([]string, 0, len(l))
		for _, ml := range l {
			s := ml.Line
			if ml.AddedFromBase {
				s += "[base]"
			}
			if ml.Crafted {
				s += "[crafted]"
			}
			s += "{" + strings.Join(ml.ModTags, ",") + "}"
			if ml.Range != nil {
				s += "@" + luaFixed(*ml.Range, 2)
			}
			out = append(out, s)
		}
		return out
	}
	for _, c := range []struct {
		raw            string
		enchant, other []string
	}{
		{"Rarity: RARE\nTest Charm\nChieftain Talisman\nImplicits: 0\n+20 to maximum Life",
			[]string{"16% increased Area of Effect[base][crafted]{}"}, nil},
		{"Rarity: RARE\nTest Maw\nBlack Maw Talisman\nImplicits: 1\nHas 1 Socket\n+20 to maximum Life",
			[]string{}, []string{"Has 1 Socket{}@0.50"}},
		{"Rarity: RARE\nTest Maw\nBlack Maw Talisman\nImplicits: 0\n+20 to maximum Life",
			[]string{"Has 1 Socket[base][crafted]{}"}, nil},
		{"Rarity: RARE\nTest Ape\nApe Talisman\nImplicits: 0\n+20 to maximum Life",
			[]string{"+1 to Minimum Endurance, Frenzy and Power Charges[base][crafted]{endurance_charge,power_charge,frenzy_charge}"}, nil},
		{"Rarity: RARE\nTest Charm\nChieftain Talisman\nImplicits: 0\n+(10-30) to maximum Life\n+(10-30) to Strength\n<ModRange id=\"2\" range=\"0.25\"/>",
			[]string{"16% increased Area of Effect[base][crafted]{}"}, nil},
	} {
		item := ParseItem(c.raw)
		if got := lines(item.EnchantModLines); !slices.Equal(got, c.enchant) {
			t.Errorf("%q enchants %q, PoB %q", c.raw, got, c.enchant)
		}
		if c.other != nil {
			if got := lines(item.ImplicitModLines); !slices.Equal(got, c.other) {
				t.Errorf("%q implicits %q, PoB %q", c.raw, got, c.other)
			}
		}
	}
	item := ParseItem("Rarity: RARE\nTest Charm\nChieftain Talisman\nImplicits: 0\n+(10-30) to maximum Life\n+(10-30) to Strength\n<ModRange id=\"2\" range=\"0.25\"/>")
	if got := lines(item.ExplicitModLines); !slices.Equal(got, []string{"+(10-30) to maximum Life{}@0.50", "+(10-30) to Strength{}@0.25"}) {
		t.Errorf("mod ranges land on %q, PoB on Strength", got)
	}
}
