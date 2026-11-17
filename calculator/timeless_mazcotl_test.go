package calculator

import (
	"strings"
	"testing"
)

// TestTimelessMazcotlLabels: PoB 2.68.1 names the third Glorious Vanity conqueror Mazcotl
// (TreeTab.lua:1224, TimelessJewelListControl.lua:86 and 92).
func TestTimelessMazcotlLabels(t *testing.T) {
	if got := TimelessConquerorTypes[1][3]; got != "Mazcotl (Immortal Ambition)" {
		t.Errorf("conqueror label %q", got)
	}
	item := TimelessResultItem(1, 4, 0, 0, 4050, 10, "x", nil)
	for _, want := range []string{"Variant: Mazcotl (Immortal Ambition)\n", "{variant:3}Bathed in the blood of 4050 sacrificed in the name of Mazcotl\n"} {
		if !strings.Contains(item, want) {
			t.Errorf("item lacks %q:\n%s", want, item)
		}
	}
}
