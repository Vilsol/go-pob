package calculator

import (
	"strings"
	"testing"
)

// Expected values come from Item.lua's parser and BuildRaw (1871-1956) on the same raw text.
func TestItemLineFlagsAgainstPoB(t *testing.T) {
	requireGameData(t, "3.30")
	cases := []struct {
		raw           string
		str, dex, int float64
		lines         []string
	}{
		{
			raw: "Rarity: RARE\nTest\nPaua Amulet\nCatalyst: Intrinsic\nCatalystQuality: 20\nImplicits: 0\n{tags:attribute}{unscalable}+20 to Strength\n{tags:attribute}+20 to Dexterity - Unscalable Value\n{tags:attribute}+20 to Intelligence",
			str: 20, dex: 20, int: 24,
			lines: []string{"{tags:attribute}{unscalable}+20 to Strength", "{tags:attribute}{unscalable}+20 to Dexterity", "{tags:attribute}+20 to Intelligence"},
		},
		{
			raw: "Rarity: RARE\nTest\nPaua Amulet\nVariant: A\nVariant: B\nSelected Variant: 1\nImplicits: 1\n{enchant}+10 to Strength\n{exarch}+5 to Dexterity\n{eater}+6 to Dexterity\n{synthesis}+7 to Dexterity\n{variant:1}{tags:attribute}+8 to Intelligence\n{unscalable}+9 to Intelligence",
			str: 10, dex: 18, int: 17,
			lines: []string{"{enchant}{crafted}+10 to Strength", "{exarch}+5 to Dexterity", "{eater}+6 to Dexterity", "{synthesis}+7 to Dexterity", "{tags:attribute}{variant:1}+8 to Intelligence", "{unscalable}+9 to Intelligence"},
		},
	}
	for n, c := range cases {
		item := ParseItem(c.raw)
		sums := itemModSummary(item)
		if sums["Str"] != c.str || sums["Dex"] != c.dex || sums["Int"] != c.int {
			t.Errorf("case %d: Str %v Dex %v Int %v, want %v %v %v", n, sums["Str"], sums["Dex"], sums["Int"], c.str, c.dex, c.int)
		}
		raw := "\n" + buildRawText(item) + "\n"
		for _, l := range c.lines {
			if !strings.Contains(raw, "\n"+l+"\n") {
				t.Errorf("case %d: raw lacks %q:\n%s", n, l, raw)
			}
		}
	}
}

// Item.lua:1692-1706 carry the replaced line's variant, version and group lists onto the mutated line.
func TestMutateModKeepsSelectionLists(t *testing.T) {
	requireGameData(t, "3.30")
	for _, dbItem := range ItemDB(ItemDBUnique) {
		name := dbItem.Name
		text, _ := ItemDBText(ItemDBUnique, name)
		item := ParseItem(text)
		for k, ml := range item.ExplicitModLines {
			if ml.NewModID == "" || !item.checkModLineVariant(ml) {
				continue
			}
			item.ExplicitModLines[k].VersionList = map[int]bool{1: true}
			item.ExplicitModLines[k].VariantGroupList = map[int]bool{2: true}
			item.UsesVariantGroups = false
			if !item.mutateMod(ml.ModID, ml.NewModID, true) {
				t.Fatalf("REFUSING: %s did not mutate", name)
			}
			got := item.ExplicitModLines[k]
			if !got.VersionList[1] || !got.VariantGroupList[2] || !got.Mutated {
				t.Errorf("%s: mutated line lists version %v group %v", name, got.VersionList, got.VariantGroupList)
			}
			return
		}
	}
	t.Fatal("REFUSING: no unique has a Foulborn line")
}

// Item.lua:1810-1819 and 1854-1861 write the defences and the cluster jewel skill and node count back.
func TestBuildRawDefencesAndClusterAgainstPoB(t *testing.T) {
	requireGameData(t, "3.30")
	for _, c := range []struct{ raw, want string }{
		{"Rarity: RARE\nTest\nVaal Regalia\nEnergy Shield: 400\nEnergyShieldBasePercentile: 0.5\nImplicits: 0\n+50 to maximum Energy Shield",
			"Rarity: RARE\nTest\nVaal Regalia\nEnergy Shield: 234\nEnergyShieldBasePercentile: 0.5\n"},
		{"Rarity: RARE\nTest\nLarge Cluster Jewel\nImplicits: 3\nAdds 8 Passive Skills\n2 Added Passive Skills are Jewel Sockets\nAdded Small Passive Skills grant: 12% increased Fire Damage\n1 Added Passive Skill is Prismatic Heart",
			"Rarity: RARE\nTest\nLarge Cluster Jewel\nCluster Jewel Skill: affliction_fire_damage\nCluster Jewel Node Count: 8\nImplicits: 3\n"},
	} {
		if got := buildRawText(ParseItem(c.raw)); !strings.HasPrefix(got, c.want) {
			t.Errorf("raw starts\n%s\nPoB\n%s", got, c.want)
		}
	}
}
