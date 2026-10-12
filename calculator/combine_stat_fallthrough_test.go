package calculator

import (
	"os"
	"testing"
)

// TestCombineStatFallthroughLeavesKeyAbsent pins CalcOffence.lua:2062 and :2093.
func TestCombineStatFallthroughLeavesKeyAbsent(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	absent := []string{"CausticGroundDPS", "BleedDPS", "PoisonDPS", "PoisonDamage", "IgniteDPS", "ImpaleModifier"}
	exercised := map[string]int{}
	for _, build := range []string{
		"0005-occultist-vaal-double-strike.xml",
		"0016-gladiator-dual-strike-of-ambidexterity.xml",
		"0017-deadeye-spectral-throw-of-materialising.xml",
		"0021-assassin-blade-flurry.xml",
		"0044-assassin-charged-dash.xml",
	} {
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			for _, k := range absent {
				if hasOutputKey(env.Player.OutputTable[OutTableMainHand], k) ||
					hasOutputKey(env.Player.OutputTable[OutTableOffHand], k) {
					continue
				}
				exercised[k]++
				if _, present := env.Player.Output[k]; present {
					t.Errorf("%s is PRESENT in the global output with value %v; upstream assigns nil "+
						"when neither hand has it, so the key must be ABSENT", k, env.Player.Output[k])
				}
			}
		})
	}
	for _, k := range absent {
		if exercised[k] == 0 {
			t.Errorf("REFUSING: %s was produced by a hand on every fixture build, so the fallthrough "+
				"was never exercised for it and this test says nothing about it. Replace the fixture.", k)
		}
	}

	t.Run("control", func(t *testing.T) {
		env := corpusEnvForLoadedData(t, "0050-occultist-viper-strike-of-the-mamba.xml")
		for _, k := range []string{"CausticGroundDPS", "PoisonDPS", "PoisonDamage"} {
			v, present := env.Player.Output[k]
			if !present || v == 0 {
				t.Errorf("control: %s = %v (present=%v), want present and nonzero — the fallthrough "+
					"fix has deleted a key a hand really produced", k, v, present)
			}
		}
	})
}

func hasOutputKey(m map[string]float64, k string) bool {
	_, ok := m[k]
	return ok
}
