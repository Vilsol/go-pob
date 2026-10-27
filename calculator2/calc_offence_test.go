package calculator2

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

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
