package calculator2

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/pob2"
)

// The expected values are PoB-PoE2's calcsTab.mainOutput for the same file, loaded headless with
// loadBuildFromXML. The build has no items or skills, and unticks the one default quest whose line
// ("5% increased Maximum Mana") the shipped ModCache does not hold; its custom lines are cached ones.
func TestBuildOutputTreeOnlyFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/warrior_tree_only.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	testza.AssertEqual(t, 20, len(env.AllocNodes))
	for stat, want := range map[string]float64{"Str": 40, "Dex": 17, "Int": 12, "Life": 547, "Mana": 174, "Spirit": 70} {
		testza.AssertEqual(t, want, env.Player.Output[stat], stat)
	}
}
