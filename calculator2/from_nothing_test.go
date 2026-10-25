package calculator2

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob2"
)

func allocatedIDs(t *testing.T, raw []byte) []int {
	t.Helper()
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)
	ids := make([]int, 0, len(env.Spec.AllocNodes))
	for id := range env.Spec.AllocNodes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// The expected sets are PoB-PoE2's spec.allocNodes for the same file loaded headless: the From Nothing
// jewel keeps Momentum, Bannerman and a Life Leech node in Vaal Pact's radius allocated although
// they are not connected to the tree; without the socketed jewel PoB prunes them.
func TestFromNothingKeepsUnconnectedNodes(t *testing.T) {
	raw, err := os.ReadFile("testdata/from_nothing.xml")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, []int{
		1207, 1913, 3936, 4665, 5710, 6839, 13397, 16725, 22928, 23570, 26725, 27373, 33556, 36709, 38323,
		38646, 39274, 39581, 41031, 46325, 47175, 54232, 54811, 55473, 57703, 61534, 62455, 63393, 63579,
	}, allocatedIDs(t, raw))

	unsocketed := strings.Replace(string(raw), `<Socket itemId="1" nodeId="26725"/>`, "", 1)
	testza.AssertNotEqual(t, string(raw), unsocketed)
	testza.AssertEqual(t, []int{
		1207, 1913, 3936, 4665, 5710, 6839, 13397, 16725, 22928, 23570, 26725, 27373, 33556, 36709, 38323,
		38646, 39581, 41031, 46325, 47175, 54232, 54811, 55473, 57703, 61534, 63393,
	}, allocatedIDs(t, []byte(unsocketed)))
}

// PoB-PoE2's ModCache.lua holds the parse of this line with lowercase captures; parseMod scans the
// lowercased line, so an uncached spelling of it captures the same.
func TestFromNothingLineCapturesLowercase(t *testing.T) {
	mods, extra := ParseMod("Passives in Radius of Vaal Pact can be Allocated without being connected to your tree")
	testza.AssertEqual(t, "", extra)
	testza.AssertEqual(t, 2, len(mods))
	testza.AssertEqual(t, mod.JewelData{Key: "fromNothingKeystone", Value: "vaal pact"}, mods[0].Value().List())
	testza.AssertEqual(t, LuaTable{"key": "vaal pact", "value": true}, mods[1].Value().List())
}
