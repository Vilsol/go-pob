package calculator2

import (
	"math"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/pob2"
)

// The expected values are PoB-PoE2's calcsTab.mainOutput for the same file, loaded headless with
// loadBuildFromXML. It is the tree-only warrior with custom lines that reach every defence-slice stat
// without an item.
func TestDefenceCustomModsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/defence_custom_mods.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{
		"Str": 40, "Dex": 17, "Int": 12, "Life": 547, "Mana": 174, "EnergyShield": 78, "Spirit": 70,
		"FireResist": 5, "ColdResist": 78, "LightningResist": -35, "ChaosResist": 17,
		"Armour": 150, "Evasion": 161, "BlockChance": 13, "DeflectChance": 2, "MovementSpeedMod": 1.15,
		"LifeUnreserved": 410, "ManaUnreserved": 174, "ActionSpeedMod": 1, "EvadeChance": 2,
		"DeflectionRating": 182.2, "FireResistTotal": 5, "ChaosResistTotal": 17, "SpellBlockChance": 0,
	} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9 {
			t.Errorf("%s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
}
