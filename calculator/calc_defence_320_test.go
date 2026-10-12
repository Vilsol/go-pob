package calculator

import (
	"math"
	"testing"
)

func defence320Env(t *testing.T, file string) *Environment {
	t.Helper()
	requireGameData(t, "3.20")

	return corpusEnvForLoadedData(t, file)
}

func assertMatches320Oracle(t *testing.T, env *Environment, file, stat string) {
	t.Helper()

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	rec, ok := oracle[file]
	if !ok {
		t.Fatalf("REFUSING: %s has no record in the oracle blob", file)
	}
	if rec.TreeSources < oracleMinTreeSources {
		t.Fatalf("REFUSING: %s has only %d distinct tree sources (<%d): headless PoB failed to "+
			"apply its passive tree, so its output is garbage. Pin a different build.",
			file, rec.TreeSources, oracleMinTreeSources)
	}
	want, ok := rec.Output[stat]
	if !ok {
		t.Fatalf("REFUSING: PoB's 3.20 output for %s has no %s (it reports %d stats)",
			file, stat, len(rec.Output))
	}

	got, ok := env.Player.Output[stat]
	if !ok {
		t.Fatalf("%s: we never write %s at all; PoB reports %.6g", file, stat, want)
	}
	if math.IsNaN(got) || math.IsInf(got, 0) {
		t.Fatalf("%s: %s is %v on our side; PoB reports %.6g", file, stat, got, want)
	}
	if diff := math.Abs(got - want); diff > math.Abs(want)*oracleDumpTolerance {
		t.Fatalf("%s: %s = %.10g, PoB 3.20 reports %.10g (delta %.10g)", file, stat, got, want, diff)
	}
}

const defence320Build = "0040-elementalist-storm-brand.xml"

// TestEnemyDamageMultIsExposed pins CalcDefence.lua:977.
func TestEnemyDamageMultIsExposed(t *testing.T) {
	env := defence320Env(t, defence320Build)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatches320Oracle(t, env, defence320Build, damageType+"EnemyDamageMult")
	}
}

// TestResistAndAfterReductionTakenHitMultiAreExposed pins CalcDefence.lua:1216-1218.
func TestResistAndAfterReductionTakenHitMultiAreExposed(t *testing.T) {
	env := defence320Env(t, defence320Build)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatches320Oracle(t, env, defence320Build, damageType+"ResistTakenHitMulti")
		assertMatches320Oracle(t, env, defence320Build, damageType+"AfterReductionTakenHitMulti")
	}
}

// TestEffectiveAppliedArmourIsExposed pins CalcDefence.lua:1215.
func TestEffectiveAppliedArmourIsExposed(t *testing.T) {
	env := defence320Env(t, defence320Build)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatches320Oracle(t, env, defence320Build, damageType+"EffectiveAppliedArmour")
	}
}

// TestSelfAilmentEffectIsExposed pins CalcDefence.lua:930-932.
func TestSelfAilmentEffectIsExposed(t *testing.T) {
	env := defence320Env(t, defence320Build)
	for _, ailment := range []string{"Bleed", "Poison", "Ignite", "Chill", "Freeze", "Shock", "Scorch", "Brittle", "Sap"} {
		assertMatches320Oracle(t, env, defence320Build, "Self"+ailment+"Effect")
	}
}

// TestDebuffExpirationModifierIsExposed pins CalcDefence.lua:905-906.
func TestDebuffExpirationModifierIsExposed(t *testing.T) {
	env := defence320Env(t, defence320Build)
	assertMatches320Oracle(t, env, defence320Build, "DebuffExpirationModifier")
}

// TestSecondMinimalMaximumHitTakenIsExposed pins CalcDefence.lua:2562-2572 — the second smallest <T>MaximumHitTaken, which is what PoB's sidebar labels "Eff.
func TestSecondMinimalMaximumHitTakenIsExposed(t *testing.T) {
	env := defence320Env(t, defence320Build)
	assertMatches320Oracle(t, env, defence320Build, "SecondMinimalMaximumHitTaken")
}

// TestStunThresholdIsExposed pins CalcDefence.lua:1301-1320.
func TestStunThresholdIsExposed(t *testing.T) {
	env := defence320Env(t, defence320Build)
	assertMatches320Oracle(t, env, defence320Build, "StunThreshold")
}
