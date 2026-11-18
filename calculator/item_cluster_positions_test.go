package calculator

import (
	"strconv"
	"strings"
	"testing"
)

// TestClusterNotablePositions: ItemTools.lua:16-82, ItemsTab.lua:1036-1040. Values from PoB's
// itemLib.getClusterJewelNotablePositions on the same texts.
func TestClusterNotablePositions(t *testing.T) {
	initItemsTabData(t)
	large := func(nodes, sockets string, notables ...string) string {
		lines := make([]string, 0, 8+len(notables))
		lines = append(lines, "Rarity: RARE", "Test Spiral", "Large Cluster Jewel", "Crafted: true", "Implicits: 3", "{crafted}Adds "+nodes+" Passive Skills",
			"{crafted}"+sockets, "{crafted}Added Small Passive Skills grant: 12% increased Fire Damage")
		for _, n := range notables {
			lines = append(lines, "1 Added Passive Skill is "+n)
		}
		return strings.Join(lines, "\n")
	}
	for _, c := range []struct{ name, raw, want string }{
		{"large3", large("8", "2 Added Passive Skills are Jewel Sockets", "Prismatic Heart", "Smoking Remains", "Burning Bright"),
			"Front:Prismatic Heart@2 | Back:Smoking Remains@6 | Front:Burning Bright@10"},
		{"large2", large("12", "2 Added Passive Skills are Jewel Sockets", "Prismatic Heart", "Smoking Remains"),
			"Back:Prismatic Heart@6 | Front:Smoking Remains@10"},
		{"large1sock", large("8", "1 Added Passive Skill is a Jewel Socket", "Prismatic Heart", "Smoking Remains", "Burning Bright"),
			"Back:Prismatic Heart@4 | Back:Smoking Remains@8 | Front:Burning Bright@10"},
		{"megalo3", "Rarity: UNIQUE\nMegalomaniac\nMedium Cluster Jewel\nImplicits: 0\nAdds 4 Passive Skills\nAdded Small Passive Skills grant Nothing\n" +
			"1 Added Passive Skill is Prismatic Heart\n1 Added Passive Skill is Smoking Remains\n1 Added Passive Skill is Burning Bright",
			"Front:Prismatic Heart@3 | Back:Smoking Remains@6 | Front:Burning Bright@9"},
		{"not shown on an uncrafted large cluster", strings.Replace(large("8", "2 Added Passive Skills are Jewel Sockets", "Prismatic Heart"), "Crafted: true\n", "", 1), ""},
	} {
		info, ok := DisplayItem(c.raw)
		var got []string
		for _, p := range info.NotablePositions {
			got = append(got, p.Position+":"+p.Name+"@"+strconv.Itoa(p.Index))
		}
		if !ok || strings.Join(got, " | ") != c.want {
			t.Errorf("%s: %q, PoB %q", c.name, strings.Join(got, " | "), c.want)
		}
	}
}
