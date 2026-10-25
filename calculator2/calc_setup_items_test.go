package calculator2

import (
	"os"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/pob2"
)

// The expected values are PoB-PoE2's for the same file, loaded headless with loadBuildFromXML:
// calcsTab.mainOutput, mainEnv.modDB:Sum("BASE"), its item multipliers and conditions, and
// mainEnv.player.itemList. The build equips a weapon swap that is not active, an item in Ring 3
// without the extra ring slot, an active flask and charm, a jewel in an allocated tree socket, one in an unallocated socket, and one in the Sinister
// socket its amulet allocates.
func TestBuildOutputItemsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/warrior_items.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{"Str": 107, "Dex": 32, "Int": 46, "Life": 1690, "Mana": 472, "Spirit": 170} {
		testza.AssertEqual(t, want, env.Player.Output[stat], stat)
	}
	for stat, want := range map[string]float64{
		"Str": 107, "Dex": 32, "Int": 46, "Life": 1470, "Mana": 472, "Spirit": 170, "Armour": 40, "Evasion": 7,
		"EnergyShield": 0, "FireResist": -7, "ColdResist": -36, "LightningResist": -12, "ChaosResist": 0, "CharmLimit": 2,
	} {
		testza.AssertEqual(t, want, env.ModDB.Sum("BASE", nil, stat), stat)
	}
	for name, want := range map[string]float64{
		"AmuletItem": 1, "BeltItem": 1, "BodyArmourItem": 1, "BootsItem": 1, "CorruptedItem": 1, "GlovesItem": 1,
		"LifeFlask1MaxCharges": 75, "LifeFlaskRecovery": 1435.2, "MagicItem": 2, "NonCorruptedItem": 8,
		"NonIdolAugmentsInEquipment": 3, "NormalItem": 1, "OneHandMaceItem": 1, "RareItem": 6, "RingItem": 2,
		"RunesSocketedInAmulet": 0, "RunesSocketedInBelt": 0, "RunesSocketedInBody Armour": 2, "RunesSocketedInBoots": 0,
		"RunesSocketedInGloves": 0, "RunesSocketedInJewel 26725": 0, "RunesSocketedInJewel 62152": 0, "RunesSocketedInRing 1": 0, "RunesSocketedInRing 2": 0,
		"RunesSocketedInWeapon 1": 1, "RunesSocketedInWeapon 2": 0, "SceptreItem": 1,
	} {
		got, ok := env.ModDB.Multipliers[name]
		testza.AssertTrue(t, ok, name)
		testza.AssertEqual(t, want, got, name)
	}
	for _, name := range []string{
		"MagicItemInAmulet", "MagicItemInBelt", "NormalItemInWeapon 2", "RareItemInBody Armour", "RareItemInBoots",
		"RareItemInGloves", "RareItemInRing 1", "RareItemInRing 2", "RareItemInWeapon 1",
	} {
		testza.AssertTrue(t, env.ModDB.Conditions[name], name)
	}
	slots := map[string]int{}
	for name, item := range env.Player.ItemList {
		slots[name] = item.ID
	}
	testza.AssertEqual(t, map[string]int{
		"Amulet": 12, "Belt": 13, "Body Armour": 1, "Boots": 10, "Gloves": 3, "Jewel 26725": 8, "Jewel 62152": 16, "Ring 1": 5, "Ring 2": 14,
		"Weapon 1": 4, "Weapon 2": 11,
	}, slots)
}

// As above for a tree-only build wearing one amulet, with custom "increased bonuses gained from
// equipped rings and amulets" and rune tattoo lines, two character runes and one unknown rune name.
func TestBuildOutputAmuletBonusFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/warrior_amulet.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)
	for stat, want := range map[string]float64{"Str": 89, "Dex": 17, "Int": 12, "Life": 660, "Mana": 174, "Spirit": 70} {
		testza.AssertEqual(t, want, env.Player.Output[stat], stat)
	}
	testza.AssertEqual(t, -32.0, env.ModDB.Sum("BASE", nil, "FireResist"))
}

// PoB-PoE2's mainEnv.enemyLevel for the tree-only fixture (no enemyIsBoss input, so the "Pinnacle"
// default) and for the same file with enemyIsBoss saved as "none".
func TestBuildOutputEnemyLevel(t *testing.T) {
	raw, err := os.ReadFile("testdata/warrior_tree_only.xml")
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	for xml, want := range map[string]int{
		string(raw): 82,
		strings.Replace(string(raw), `<ConfigSet id="1" title="Default">`, `<ConfigSet id="1" title="Default"><Input name="enemyIsBoss" string="none"/>`, 1): 30,
	} {
		build, err := pob2.ParseBuild([]byte(xml))
		testza.AssertNoError(t, err)
		env, err := BuildOutput(build, Options{Tree: tree})
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, want, env.EnemyLevel)
	}
}
