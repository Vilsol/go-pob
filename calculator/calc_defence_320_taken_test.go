package calculator

import "testing"

const takenBuildJuggernaut = "0001-juggernaut-molten-strike.xml"

// TestPhysicalResistIsZero pins CalcDefence.lua:64.
func TestPhysicalResistIsZero(t *testing.T) {
	env := defence320Env(t, takenBuildJuggernaut)
	assertMatches320Oracle(t, env, takenBuildJuggernaut, "PhysicalResist")
	assertMatches320Oracle(t, env, takenBuildJuggernaut, "PhysicalResistTakenHitMulti")
}

// TestBaseDamageReductionFamilyIsExposed pins CalcDefence.lua:136-139.
func TestBaseDamageReductionFamilyIsExposed(t *testing.T) {
	env := defence320Env(t, takenBuildJuggernaut)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatches320Oracle(t, env, takenBuildJuggernaut, "Base"+damageType+"DamageReduction")
		assertMatches320Oracle(t, env, takenBuildJuggernaut, "Base"+damageType+"DamageReductionWhenHit")
	}
}

// TestArmourAppliesToElementalIsAPercentage pins ModParser.lua:1794-1804 through its only consumer, CalcDefence.lua:1165.
func TestArmourAppliesToElementalIsAPercentage(t *testing.T) {
	env := defence320Env(t, takenBuildJuggernaut)
	for _, damageType := range []string{"Fire", "Cold", "Lightning"} {
		assertMatches320Oracle(t, env, takenBuildJuggernaut, damageType+"EffectiveAppliedArmour")
	}
}

const takenBuildOverwhelm = "0287-occultist-frostbite-2.xml"

// TestEnemyOverwhelmIsExposed pins CalcDefence.lua:960, :969-971 and :978 — a config-sourced output the pinned 2.18.1 had no counterpart for.
func TestEnemyOverwhelmIsExposed(t *testing.T) {
	env := defence320Env(t, takenBuildOverwhelm)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatches320Oracle(t, env, takenBuildOverwhelm, damageType+"EnemyOverwhelm")
	}
}

const takenBuildArmourAndFlatDR = "5368-elementalist-summon-chaos-golem.xml"

func TestTakenHitChainUsesReductMult(t *testing.T) {
	env := defence320Env(t, takenBuildArmourAndFlatDR)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatches320Oracle(t, env, takenBuildArmourAndFlatDR, damageType+"DamageReduction")
		assertMatches320Oracle(t, env, takenBuildArmourAndFlatDR, damageType+"BaseTakenHitMult")
		assertMatches320Oracle(t, env, takenBuildArmourAndFlatDR, damageType+"TakenHitMult")
		assertMatches320Oracle(t, env, takenBuildArmourAndFlatDR, damageType+"TakenHit")
	}
}

// TestMaximumHitTakenSolvesTheQuadratic pins CalcDefence.lua:2420-2519.
func TestMaximumHitTakenSolvesTheQuadratic(t *testing.T) {
	env := defence320Env(t, takenBuildArmourAndFlatDR)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatches320Oracle(t, env, takenBuildArmourAndFlatDR, damageType+"MaximumHitTaken")
	}
	assertMatches320Oracle(t, env, takenBuildArmourAndFlatDR, "SecondMinimalMaximumHitTaken")
}

// TestTakenDotMultUsesFlatReduction pins CalcDefence.lua:1125-1127 — the DoT half of the same move.
func TestTakenDotMultUsesFlatReduction(t *testing.T) {
	env := defence320Env(t, takenBuildArmourAndFlatDR)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatches320Oracle(t, env, takenBuildArmourAndFlatDR, damageType+"TakenDotMult")
	}
}
