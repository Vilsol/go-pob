package calculator2

import (
	"os"
	"sort"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/pob2"
)

// The expected values are PoB-PoE2's for the same file, loaded headless: mainOutput, and the
// mods its final modDB holds from the jewel socket and the two passives in the jewel's radius.
// The fixture is the items fixture with a Time-Lost jewel in socket 26725 (Large radius) and two
// more passives allocated inside that radius, a small one and a notable.
func TestBuildOutputRadiusJewelFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/radius_jewel.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{"Str": 114, "Dex": 32, "Int": 46, "Life": 1662, "Mana": 472, "Armour": 1557} {
		testza.AssertEqual(t, want, env.Player.Output[stat], stat)
	}

	var got []string
	for _, mods := range env.ModDB.Mods {
		for _, m := range mods {
			switch m.GetSource() {
			case "Tree:26725", "Tree:53921", "Tree:58838":
				got = append(got, radiusModKey(m.Name(), string(m.Type()), m.Value().Float()))
			}
		}
	}
	sort.Strings(got)
	testza.AssertEqual(t, []string{
		"AilmentThreshold|INC|36",
		"Armour|INC|4",
		"JewelNotablePassiveSkillEffect|INC|20",
		"Life|INC|7",
		"Str|BASE|7",
		"StunThreshold|INC|12",
		"StunThreshold|INC|36",
	}, got)
}
