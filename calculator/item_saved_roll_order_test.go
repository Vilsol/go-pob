package calculator

import "testing"

// spec/System/TestItemParse_spec.lua "keeps saved rolls on their lines for a unique saved out of stat order"
func TestSavedRollsStayOnTheirLines(t *testing.T) {
	initItemsTabData(t)
	item := ParseItem(`Rarity: UNIQUE
Geofri's Sanctuary
Elegant Ringmail
Implicits: 0
{range:0.2}+(60-70) to maximum Life
{range:0.9}(50-75)% increased Armour and Energy Shield
<ModRange id="1" range="0.2"/>
<ModRange id="2" range="0.9"/>`)
	ranges := map[string]float64{}
	for _, ml := range item.ExplicitModLines {
		if ml.Range != nil {
			ranges[ml.Line] = *ml.Range
		}
	}
	if got := ranges["(50-75)% increased Armour and Energy Shield"]; got != 0.9 {
		t.Errorf("armour line range %v, PoB 0.9 (%v)", got, ranges)
	}
}

// Item.lua:584-590: an in-game "{ ... }" header is not a modifier line.
func TestModifierHeaderIsNotAModLine(t *testing.T) {
	initItemsTabData(t)
	item := ParseItem("Rarity: RARE\nDoom Loop\nCoral Ring\n--------\nItem Level: 80\n--------\n{ Prefix Modifier \"Hale\" (Tier: 9) - Life }\n+(10-19) to maximum Life\n")
	for _, ml := range append(item.ImplicitModLines, item.ExplicitModLines...) {
		if ml.Line == "" {
			t.Errorf("header parsed as an empty modifier: %+v", item.ExplicitModLines)
		}
	}
}

// spec/System/TestItemParse_spec.lua "keeps the roll written in a saved value(min-max) line over its ModRange"
func TestSavedValueRollBeatsModRange(t *testing.T) {
	initItemsTabData(t)
	item := ParseItem(`Rarity: UNIQUE
Crown of the Inward Eye
Prophet Crown
Implicits: 0
{range:0.5}14(9-21)% increased maximum Life, Mana and Global Energy Shield
<ModRange id="1" range="0.5"/>`)
	if len(item.ExplicitModLines) != 1 || item.ExplicitModLines[0].Range == nil || *item.ExplicitModLines[0].Range != 0.416667 {
		t.Errorf("range %+v, PoB 0.416667", item.ExplicitModLines)
	}
}

// spec/System/TestItemParse_spec.lua "keeps saved rolls when a property was saved among the modifier lines"
func TestSurplusModRangesAreIgnored(t *testing.T) {
	initItemsTabData(t)
	item := ParseItem(`Rarity: RARE
Pain Shell
Vaal Regalia
Implicits: 0
Intangibility: 7%
{range:0}+(10-13) to all Attributes
<ModRange id="1" range="0.5"/>
<ModRange id="2" range="0"/>`)
	if len(item.ExplicitModLines) != 1 || item.ExplicitModLines[0].Range == nil || *item.ExplicitModLines[0].Range != 0 {
		t.Errorf("range %+v, PoB 0", item.ExplicitModLines)
	}
}
