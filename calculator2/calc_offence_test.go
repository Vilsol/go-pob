package calculator2

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/pob2"
)

func offenceFixtureEnv(t *testing.T, path string) *Environment {
	t.Helper()
	raw, err := os.ReadFile(path)
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)
	return env
}

// The expected values are PoB-PoE2's output for the same files, loaded headless. offence_setup.xml is
// a level 13, 20% quality Spark with Fork under "20% increased Area of Effect", "30% increased Skill
// Effect Duration", "15% increased Mana Cost of Skills", "Projectiles Pierce an additional Target" and
// "+1 to Limit for Traps".
func TestOffenceSetupFixture(t *testing.T) {
	env := offenceFixtureEnv(t, "testdata/offence_setup.xml")
	testza.AssertEqual(t, "Spark", env.Player.MainSkill.ActiveEffect.GrantedEffect.Name)
	assertPoE2Outputs(t, env, map[string]float64{
		"ActiveMineLimit": 15, "ActiveTrapLimit": 15, "ArmourBreakPerHit": 0,
		"Chain": 0, "ChainMax": 0, "ChainMaxString": 0, "ChainRemaining": 0,
		"Duration": 2.607, "DurationMod": 1.3,
		"ForkCountMax": 1, "ForkCountString": 1, "ForkRemaining": 1, "ForkedCount": 0,
		"LifeCost": 0, "ManaCost": 33, "ManaCostRaw": 34.155, "ManaCostPayablePool": 183, "RageCost": 0,
		"PierceCount": 1, "PierceCountString": 1, "PiercedCount": 0,
		"ProjectileCount": 7, "ProjectileSpeedMod": 1.3, "TwoAdditionalProjectiles": 0,
		"SplitCount": 0, "SplitCountString": 0, "TerrainChain": 0,
		"Repeats": 1, "TotemDuration": 0, "TotemDurationMod": 1,
	})
}

// skill_modlist.xml is TestBuildActiveSkillModListFixture's Falling Thunder.
func TestOffenceSetupAreaFixture(t *testing.T) {
	env := offenceFixtureEnv(t, "testdata/skill_modlist.xml")
	assertPoE2Outputs(t, env, map[string]float64{
		"ActiveMineLimit": 15, "ActiveTrapLimit": 15,
		"AreaOfEffectMod": 1, "AreaOfEffectRadius": 40, "AreaOfEffectRadiusMetres": 4,
		"DurationMod": 1, "ManaCost": 78, "ManaCostRaw": 78, "ManaCostPayablePool": 183,
		"Repeats": 1, "TotemDuration": 0, "TotemDurationMod": 1,
	})
}

// The hit half of the same Spark: 50% enemy lightning resistance halves the stored hit.
func TestOffenceHitFixture(t *testing.T) {
	env := offenceFixtureEnv(t, "testdata/offence_setup.xml")
	assertPoE2Outputs(t, env, map[string]float64{
		"TotalDPS": 38.15, "CombinedDPS": 38.15, "AverageHit": 26.705, "AverageDamage": 26.705,
		"Speed": 1.4285714285714, "CastRate": 1.4285714285714, "Time": 0.7,
		"CritChance": 9, "PreEffectiveCritChance": 9, "CritMultiplier": 2, "CritEffect": 1.09,
		"HitChance": 100, "AccuracyHitChance": 100, "Accuracy": 246, "DpsMultiplier": 1,
		"LightningMinBase": 5, "LightningMaxBase": 93, "LightningSummedMinBase": 5, "LightningSummedMaxBase": 93,
		"LightningHitAverage": 24.5, "LightningCritAverage": 49, "LightningStoredCombinedAvg": 53.41,
		"LightningStoredHitMax": 93, "TotalMin": 2.5, "TotalMax": 46.5,
		"ScaledDamageEffect": 1, "allMult": 1, "CullMultiplier": 1, "TotalDotDPS": 0,
	})
}

// offence_ailments.xml is the level 13, 20% quality Spark under "Gain 10% of Damage as Extra Physical
// Damage", "Gain 10% of Damage as Extra Fire Damage", "25% chance to Poison on Hit" and "20% chance to
// inflict Bleeding on Hit", against an enemy with no armour: ignite, bleed, poison, shock and heavy
// stun buildup from one hit.
func TestOffenceAilmentsFixture(t *testing.T) {
	env := offenceFixtureEnv(t, "testdata/offence_ailments.xml")
	assertPoE2Outputs(t, env, map[string]float64{
		"EnemyAilmentThreshold": 71303, "chillMinimumThreshold": 713.03, "ChillChanceOnHit": 0,
		"BleedChance": 20, "BleedChancePerHit": 20, "BleedDPS": 0.96376440034885, "BleedDamage": 4.8188220017442, "BleedDuration": 5, "BleedPhysicalMin": 1, "BleedPhysicalMax": 9, "BleedRollAverage": 58.823529411765, "BleedStackPotential": 1.4285714285714, "BleedStacksMax": 1, "BleedEffMult": 1, "BleedMagnitudeEffect": 1,
		"PoisonChance": 25, "PoisonDPS": 0.77857142857143, "PoisonDamage": 1.5571428571429, "PoisonDuration": 2, "PoisonStackPotential": 0.71428571428571, "PoisonPhysicalMax": 9,
		"IgniteChanceOnHit": 0.0014024655344095, "IgniteChanceOnCrit": 0.002804931068819, "IgniteChance": 0.0015286874325063, "IgniteDPS": 5.0889463677144e-05, "IgniteDamage": 0.00020355785470858, "IgniteDuration": 4, "IgniteEffMult": 0.5, "IgniteFireMin": 1, "IgniteFireMax": 9, "IgniteStackPotential": 8.7353567571791e-05,
		"ShockChanceOnHit": 0.017180202796516, "ShockChanceOnCrit": 0.034360405593033, "ShockChance": 0.018726421048203, "ShockDuration": 8, "ShockEffectMod": 1, "ShockLightningMin": 5, "ShockLightningMax": 93,
		"HeavyStunBuildupAvg": 0.00022901995220435, "HeavyStunFireMin": 1, "HeavyStunFireMax": 2.5, "HeavyStunLightningMin": 5, "HeavyStunLightningMax": 24.5, "HeavyStunPhysicalMin": 1.5, "HeavyStunPhysicalMax": 7.5, "FreezeBuildupAvg": 0, "PinBuildupAvg": 0, "ElectrocuteBuildupAvg": 0,
		"EnemyStunDuration": 0.35, "EnemyStunThresholdMod": 1,
		"TotalDot": 0, "TotalDotInstance": 0, "TotalDotDPS": 1.742386718384, "WithBleedDPS": 50.79233582892, "WithIgniteDPS": 49.828622318035, "WithPoisonDPS": 50.607142857143,
		"TotalDPS": 49.828571428571, "CombinedDPS": 51.570958146955, "ManaPerSecondCost": 38.571428571429,
	})
}

//pob2:block:src/Data/Skills/other.lua:21152
func TestPreDamageExplodeSource(t *testing.T) {
	item := &Item{ID: 3, Name: "Foo"}
	item.modSourceName = "Item:3:Foo"
	itemSkill := preDamageSkill{&ActiveSkill{ActiveEffect: &GemEffect{SrcInstance: &GemInstance{ExplodeSource: item}}}}
	modSource, ok, _ := itemSkill.ExplodeSource()
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, "Item:3:Foo", modSource)

	nodeSkill := preDamageSkill{&ActiveSkill{ActiveEffect: &GemEffect{SrcInstance: &GemInstance{ExplodeSource: &tree2.SpecNode{Node: &tree2.Node{ID: 4271}}}}}}
	_, ok, id := nodeSkill.ExplodeSource()
	testza.AssertFalse(t, ok)
	testza.AssertEqual(t, "4271", id)
}
