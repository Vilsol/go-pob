package calculator2

import (
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob2"
)

// The expected values are PoB-PoE2's for the same file, loaded headless with loadBuildFromXML:
// calcsTab.mainOutput and mainEnv.modDB sums. The tree-only warrior wears a rare ring with a
// repeated line under 30% left-ring bonus effect, Kalandra's Touch under 10% right-ring bonus effect,
// and a Mageblood with a duplicated legacy.
func TestPerformRingBonusMagebloodFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/ring_mageblood.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{
		"Str": 59, "Life": 622, "FireResist": 81, "FireResistTotal": 131, "ColdResist": 6, "LightningResist": 6, "Evasion": 3109,
	} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9 {
			t.Errorf("output %s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
	for stat, want := range map[string]float64{
		"Str": 59, "Life": 592, "FireResist": 75, "FireResistMax": 81, "ElementalResist": 56, "Evasion": 2507,
	} {
		testza.AssertEqual(t, want, env.ModDB.Sum("BASE", nil, stat), "BASE "+stat)
	}
	testza.AssertEqual(t, 24.0, env.ModDB.Sum("INC", nil, "Evasion"), "INC Evasion")
	testza.AssertEqual(t, 5.0, env.ModDB.Sum("INC", nil, "Life"), "INC Life")
}

// As above for the tree-only warrior with custom Onslaught, Adrenaline and "You are Blind" lines.
func TestPerformActorMiscFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/actor_misc.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{"Evasion": 16, "MovementSpeedMod": 1.35, "Life": 489} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9 {
			t.Errorf("output %s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
	for _, c := range []struct {
		modType mod.Type
		name    string
		want    float64
	}{
		{mod.TypeBase, "PhysicalDamageReduction", 10}, {mod.TypeMore, "Accuracy", -20}, {mod.TypeIncrease, "Damage", 100}, {mod.TypeIncrease, "MovementSpeed", 35},
	} {
		testza.AssertEqual(t, c.want, env.ModDB.Sum(c.modType, nil, c.name), string(c.modType)+" "+c.name)
	}
}

// As above for the tree-only warrior wielding a normal Quarterstaff, with custom lines conditioned on
// the weapon-type conditions.
func TestPerformWeaponConditionsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/weapon_conditions.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{"Life": 583, "Armour": 140, "Evasion": 37, "EnergyShield": 0} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9 {
			t.Errorf("output %s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
	var conds []string
	for k, v := range env.ModDB.Conditions {
		if v && (strings.HasPrefix(k, "Using") || strings.Contains(k, "Unarmed") || strings.Contains(k, "Wield")) {
			conds = append(conds, k)
		}
	}
	sort.Strings(conds)
	testza.AssertEqual(t, []string{"UsingMartialWeapon", "UsingMeleeWeapon", "UsingStaff", "UsingTwoHandedWeapon"}, conds)
}

func loadChargesCursesFixture(t *testing.T) *Environment {
	t.Helper()
	raw, err := os.ReadFile("testdata/charges_curses.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)
	return env
}

func assertPoE2Outputs(t *testing.T, env *Environment, want map[string]float64) {
	t.Helper()
	for stat, w := range want {
		if got, ok := env.Player.Output[stat]; !ok || math.Abs(got-w) > 1e-9 {
			t.Errorf("output %s: PoB-PoE2 %v, ours %v (set %v)", stat, w, got, ok)
		}
	}
}

// As above for the tree-only warrior with custom maximum charge, charge duration, Blood Charge, curse and
// mark limit, rarity and attribute requirement lines.
func TestPerformChargesFixture(t *testing.T) {
	env := loadChargesCursesFixture(t)
	assertPoE2Outputs(t, env, map[string]float64{
		"PowerChargesMax": 4, "PowerChargesDuration": 18, "FrenzyChargesMax": 5, "FrenzyChargesDuration": 18,
		"EnduranceChargesMax": 5, "EnduranceChargesDuration": 21, "BloodChargesMax": 7, "BloodCharges": 7,
		"InspirationCharges": 5, "InspirationChargesMax": 5, "PowerCharges": 0, "TotalCharges": 0,
	})
	testza.AssertEqual(t, 7.0, env.ModDB.Multipliers["BloodCharge"])
}

func TestPerformCurseLimitFixture(t *testing.T) {
	env := loadChargesCursesFixture(t)
	assertPoE2Outputs(t, env, map[string]float64{"EnemyCurseLimit": 4, "EnemyMarkLimit": 2})
}

func TestPerformStatOutputsFixture(t *testing.T) {
	env := loadChargesCursesFixture(t)
	assertPoE2Outputs(t, env, map[string]float64{
		"EffectiveLootRarityMod": 1.3, "LootQuantityNormalEnemies": 0, "LootRarityMagicEnemies": 0,
		"GlobalItemAttributeRequirements": 0.9, "GlobalGemAttributeRequirements": 0.9, "GlobalWeaponAttributeRequirements": 0.9,
		"CharmLimit": 0, "CharmEffect": 20, "FlaskEffect": 10, "WarcryPower": 20,
		"FlaskChargeGen": 0.25, "LifeFlaskChargeGen": 0.45, "ManaFlaskChargeGen": 0.25, "UtilityFlaskChargeGen": 0.25,
	})
}

// As above for the tree-only warrior with a Chill maximum override, stacking Shocks and a Shock maximum line.
func TestPerformAilmentLimitsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/ailment_limits.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)
	assertPoE2Outputs(t, env, map[string]float64{
		"MaximumChill": 35, "MaximumShock": 110, "CurrentChill": 0, "CurrentShock": 0, "ShockStackCount": 1,
	})
}

// As above for the tree-only warrior wearing items with attribute requirements.
func TestPerformAttributeRequirementsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/warrior_items.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)
	assertPoE2Outputs(t, env, map[string]float64{
		"ReqStr": 104, "ReqStrString": 104, "ReqDex": 44, "ReqInt": 92,
		"StrRequirementsOnBody Armour": 60, "StrRequirementsOnBoots": 101, "StrRequirementsOnWeapon 1": 104,
		"DexRequirementsOnGloves": 44, "IntRequirementsOnGloves": 44, "IntRequirementsOnWeapon 2": 92,
	})
}

// As above, for the base damage and local accuracy of the same warrior's weapon.
func TestPerformWeaponBaseStatsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/warrior_items.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)
	assertPoE2Outputs(t, env, map[string]float64{
		"AccuracyOnWeapon 1": 150, "PhysicalMinOnWeapon 1": 157, "PhysicalMaxOnWeapon 1": 231,
		"LightningMinOnWeapon 1": 5, "LightningMaxOnWeapon 1": 50,
	})
	if _, ok := env.Player.Output["FireMinOnWeapon 1"]; ok {
		t.Error("FireMinOnWeapon 1 is set; PoB-PoE2 leaves it unset")
	}
}

// applyEnemyModifiers copies each EnemyModifier's mod into the enemy's modDB once per cache, with the
// outer mod's source when the inner one has none.
func TestApplyEnemyModifiers(t *testing.T) {
	player := &Actor{ModDB: moddb.NewModDB()}
	enemy := &Actor{ModDB: moddb.NewModDB()}
	player.Enemy, enemy.Enemy = enemy, player
	player.ModDB.AddMod(mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Blinded", true)}).Source("Item:Ring"))

	applyEnemyModifiers(player, true)
	applyEnemyModifiers(player, false)
	testza.AssertTrue(t, enemy.ModDB.Flag(nil, "Condition:Blinded"))
	testza.AssertLen(t, enemy.ModDB.Mods["Condition:Blinded"], 1)
	testza.AssertEqual(t, mod.Source("Item:Ring"), enemy.ModDB.Mods["Condition:Blinded"][0].GetSource())

	applyEnemyModifiers(player, true)
	testza.AssertLen(t, enemy.ModDB.Mods["Condition:Blinded"], 2)
}
