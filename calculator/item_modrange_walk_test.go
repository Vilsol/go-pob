package calculator

import (
	"testing"
)

const defianceOfDestinyRaw = `
			Rarity: UNIQUE
Defiance of Destiny
Paua Amulet
Catalyst: Fertile
CatalystQuality: 20
Item Level: 84
LevelReq: 49
Implicits: 2
{enchant}{crafted}Allocates Force of Darkness
{tags:mana}{range:1}(20-30)% increased Mana Regeneration Rate
Intangibility: 5%
{tags:life}{range:0.3}+(60-80) to maximum Life
{tags:elemental,fire,resistance}{range:0.833}+(10-40)% to Fire Resistance
{tags:elemental,cold,resistance}{range:0.933}+(10-40)% to Cold Resistance
{tags:elemental,lightning,resistance}{range:0.9}+(10-40)% to Lightning Resistance
{tags:life}{range:1}Gain (10-20)% of Missing Unreserved Life before being Hit by an Enemy
			<ModRange id="1" range="1"/>
			<ModRange id="2" range="1"/>
			<ModRange id="3" range="1"/>
			<ModRange id="4" range="0.3"/>
			<ModRange id="5" range="0.833"/>
			<ModRange id="6" range="0.933"/>
			<ModRange id="7" range="0.9"/>
			<ModRange id="8" range="1"/>
`

const kaomsRootsRaw = `
			Rarity: UNIQUE
Kaom&apos;s Roots
Titan Greaves
Armour: 334
ArmourBasePercentile: 1
Variant: Pre 2.6.0
Variant: Current
Selected Variant: 2
Quality: 20
LevelReq: 68
Implicits: 0
+25% movement speed
Has no Sockets
{variant:1}{range:0.5}+(120-150) to maximum Life
{variant:2}{range:0.72}+(150-200) to maximum Life
Cannot be Knocked Back
{variant:2}Action Speed cannot be modified to below Base Value
Unwavering Stance
			<ModRange id="1" range="0.5"/>
			<ModRange id="2" range="0.5"/>
			<ModRange id="3" range="0.5"/>
			<ModRange id="4" range="0.72"/>
			<ModRange id="5" range="0.5"/>
			<ModRange id="6" range="0.5"/>
			<ModRange id="7" range="0.5"/>
`

func rangeOf(t *testing.T, lines []ModLine, idx int, what string) float64 {
	t.Helper()
	if idx < 1 || idx > len(lines) {
		t.Fatalf("REFUSING: %s: index %d outside a list of %d -- the fixture is not exercising what it claims", what, idx, len(lines))
	}
	ml := lines[idx-1]
	if ml.Range == nil {
		t.Fatalf("%s (%q): range is nil", what, ml.Line)
	}
	return *ml.Range
}

// TestModRangeWalkAddressesTheSixListConcatenation pins ItemsTab.lua:1242-1255.
func TestModRangeWalkAddressesTheSixListConcatenation(t *testing.T) {
	t.Run("index shift across enchant and implicit", func(t *testing.T) {
		item := ParseItem(defianceOfDestinyRaw)

		if len(item.EnchantModLines) != 1 || len(item.ImplicitModLines) != 1 || len(item.ExplicitModLines) != 6 {
			t.Fatalf("REFUSING: fixture shape changed: enchant %d implicit %d explicit %d",
				len(item.EnchantModLines), len(item.ImplicitModLines), len(item.ExplicitModLines))
		}
		if len(item.ScourgeModLines) != 0 || len(item.ClassRequirementModLines) != 0 {
			t.Fatalf("REFUSING: fixture gained a scourge/classRequirement line; the concatenation it pins has moved")
		}

		if got := rangeOf(t, item.EnchantModLines, 1, "enchant[1]"); got != 1 {
			t.Errorf("enchant[1] range = %v, want 1 (id1)", got)
		}
		if got := rangeOf(t, item.ImplicitModLines, 1, "implicit[1]"); got != 1 {
			t.Errorf("implicit[1] range = %v, want 1 (id2)", got)
		}

		if got := rangeOf(t, item.ExplicitModLines, 1, "explicit[1] Intangibility"); got != 1 {
			t.Errorf("explicit[1] (Intangibility) range = %v, want 1 (id3 through the shift)", got)
		}
		for _, c := range []struct {
			idx  int
			want float64
			name string
		}{
			{2, 0.3, "maximum Life"},
			{3, 0.833, "Fire Resistance"},
			{4, 0.933, "Cold Resistance"},
			{5, 0.9, "Lightning Resistance"},
			{6, 1, "Missing Unreserved Life"},
		} {
			if got := rangeOf(t, item.ExplicitModLines, c.idx, "explicit"); got != c.want {
				t.Errorf("explicit[%d] (%s) range = %v, want %v", c.idx, c.name, got, c.want)
			}
		}
	})

	t.Run("flat case, no shift", func(t *testing.T) {
		item := ParseItem(kaomsRootsRaw)
		if len(item.EnchantModLines) != 0 || len(item.ImplicitModLines) != 0 || len(item.ExplicitModLines) != 7 {
			t.Fatalf("REFUSING: fixture shape changed: enchant %d implicit %d explicit %d",
				len(item.EnchantModLines), len(item.ImplicitModLines), len(item.ExplicitModLines))
		}
		if got := rangeOf(t, item.ExplicitModLines, 4, "explicit[4]"); got != 0.72 {
			t.Errorf("explicit[4] range = %v, want 0.72 (id4)", got)
		}
		for _, idx := range []int{1, 2, 3, 5, 6, 7} {
			if got := rangeOf(t, item.ExplicitModLines, idx, "explicit"); got != 0.5 {
				t.Errorf("explicit[%d] (%q) range = %v, want 0.5", idx, item.ExplicitModLines[idx-1].Line, got)
			}
		}
	})

	t.Run("attribute order is irrelevant", func(t *testing.T) {
		a := parseModRangeElements(`<ModRange id="3" range="0.25"/>`)
		b := parseModRangeElements(`<ModRange range="0.25" id="3"/>`)
		if len(a) != 1 || len(b) != 1 {
			t.Fatalf("REFUSING: reader returned %d and %d elements, want 1 and 1", len(a), len(b))
		}
		if a[0] != b[0] {
			t.Errorf("attribute order changed the parse: %+v vs %+v", a[0], b[0])
		}
		if a[0].ID != 3 || a[0].Range != 0.25 {
			t.Errorf("got %+v, want {ID:3 Range:0.25}", a[0])
		}
		c := parseModRangeElements(`<ModRange id="2"/>`)
		if len(c) != 1 || c[0].Range != 1 {
			t.Errorf("missing range attribute: got %+v, want Range 1", c)
		}
	})
}
