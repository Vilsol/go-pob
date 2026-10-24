package calculator

import (
	"math"
	"testing"
)

func defenceEnv(t *testing.T, file string) *Environment {
	t.Helper()
	requireGameData(t, "3.29")

	return corpusEnvForLoadedData(t, file)
}

func assertMatchesOracle(t *testing.T, env *Environment, file, stat string) {
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
		t.Fatalf("REFUSING: PoB's output for %s has no %s (it reports %d stats)",
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
		t.Fatalf("%s: %s = %.10g, PoB reports %.10g (delta %.10g)", file, stat, got, want, diff)
	}
}

const (
	defenceTakenHitMultiBuild    = "2649-ladder-dgsgsgs_6561.PathofExileisreallyfun.xml"
	defenceAppliedArmourBuild    = "1101-reliquarian-automation.xml"
	defenceSelfAilmentBuild      = "0167-slayer-cyclone.xml"
	defenceDebuffExpirationBuild = "3271-ladder-SoftlyAndQuietly_5252.Softly_Yolo.xml"
)

// TestResistAndAfterReductionTakenHitMultiAreExposed pins CalcDefence.lua:1216-1218.
func TestResistAndAfterReductionTakenHitMultiAreExposed(t *testing.T) {
	env := defenceEnv(t, defenceTakenHitMultiBuild)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatchesOracle(t, env, defenceTakenHitMultiBuild, damageType+"ResistTakenHitMulti")
		assertMatchesOracle(t, env, defenceTakenHitMultiBuild, damageType+"AfterReductionTakenHitMulti")
	}
}

// TestEffectiveAppliedArmourIsExposed pins CalcDefence.lua:1215.
func TestEffectiveAppliedArmourIsExposed(t *testing.T) {
	env := defenceEnv(t, defenceAppliedArmourBuild)
	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		assertMatchesOracle(t, env, defenceAppliedArmourBuild, damageType+"EffectiveAppliedArmour")
	}
}

// TestSelfAilmentEffectIsExposed pins CalcDefence.lua:930-932.
func TestSelfAilmentEffectIsExposed(t *testing.T) {
	env := defenceEnv(t, defenceSelfAilmentBuild)
	for _, ailment := range []string{"Bleed", "Poison", "Ignite", "Chill", "Freeze", "Shock", "Scorch", "Brittle", "Sap"} {
		assertMatchesOracle(t, env, defenceSelfAilmentBuild, "Self"+ailment+"Effect")
	}
}

// TestDebuffExpirationModifierIsExposed pins CalcDefence.lua:905-906.
func TestDebuffExpirationModifierIsExposed(t *testing.T) {
	env := defenceEnv(t, defenceDebuffExpirationBuild)
	assertMatchesOracle(t, env, defenceDebuffExpirationBuild, "DebuffExpirationModifier")
}
