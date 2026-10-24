package calculator

import "testing"

const (
	takenBuildJuggernaut      = "5059-ladder-JohnyCena777_6914.JohnyCAgain.xml"
	takenBuildElementalArmour = "1101-reliquarian-automation.xml"
)

// TestPhysicalResistIsZero pins CalcDefence.lua:64.
func TestPhysicalResistIsZero(t *testing.T) {
	env := defenceEnv(t, takenBuildJuggernaut)
	assertMatchesOracle(t, env, takenBuildJuggernaut, "PhysicalResist")
	assertMatchesOracle(t, env, takenBuildJuggernaut, "PhysicalResistTakenHitMulti")
}

// TestBaseDamageReductionFamilyIsExposed pins CalcDefence.lua:136-139.
func TestBaseDamageReductionFamilyIsExposed(t *testing.T) {
	env := defenceEnv(t, takenBuildJuggernaut)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatchesOracle(t, env, takenBuildJuggernaut, "Base"+damageType+"DamageReduction")
		assertMatchesOracle(t, env, takenBuildJuggernaut, "Base"+damageType+"DamageReductionWhenHit")
	}
}

// TestArmourAppliesToElementalIsAPercentage pins ModParser.lua:1794-1804 through its only consumer, CalcDefence.lua:1165.
func TestArmourAppliesToElementalIsAPercentage(t *testing.T) {
	env := defenceEnv(t, takenBuildElementalArmour)
	for _, damageType := range []string{"Fire", "Cold", "Lightning"} {
		assertMatchesOracle(t, env, takenBuildElementalArmour, damageType+"EffectiveAppliedArmour")
	}
}

const takenBuildOverwhelm = "0869-gladiator-animate-guardian-2.xml"

// TestEnemyOverwhelmIsExposed pins CalcDefence.lua:960, :969-971 and :978 — a config-sourced output the pinned 2.18.1 had no counterpart for.
func TestEnemyOverwhelmIsExposed(t *testing.T) {
	env := defenceEnv(t, takenBuildOverwhelm)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatchesOracle(t, env, takenBuildOverwhelm, damageType+"EnemyOverwhelm")
	}
}

const takenBuildArmourAndFlatDR = "4756-ladder-Furych_6028.Furych_Gangsta.xml"

func TestTakenHitChainUsesReductMult(t *testing.T) {
	env := defenceEnv(t, takenBuildArmourAndFlatDR)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatchesOracle(t, env, takenBuildArmourAndFlatDR, damageType+"DamageReduction")
		assertMatchesOracle(t, env, takenBuildArmourAndFlatDR, damageType+"BaseTakenHitMult")
		assertMatchesOracle(t, env, takenBuildArmourAndFlatDR, damageType+"TakenHitMult")
		assertMatchesOracle(t, env, takenBuildArmourAndFlatDR, damageType+"TakenHit")
	}
}

// TestMaximumHitTakenSolvesTheQuadratic pins CalcDefence.lua:2420-2519.
func TestMaximumHitTakenSolvesTheQuadratic(t *testing.T) {
	env := defenceEnv(t, takenBuildArmourAndFlatDR)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatchesOracle(t, env, takenBuildArmourAndFlatDR, damageType+"MaximumHitTaken")
	}
	assertMatchesOracle(t, env, takenBuildArmourAndFlatDR, "SecondMinimalMaximumHitTaken")
}

// TestTakenDotMultUsesFlatReduction pins CalcDefence.lua:1125-1127 — the DoT half of the same move.
func TestTakenDotMultUsesFlatReduction(t *testing.T) {
	env := defenceEnv(t, takenBuildArmourAndFlatDR)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatchesOracle(t, env, takenBuildArmourAndFlatDR, damageType+"TakenDotMult")
	}
}
