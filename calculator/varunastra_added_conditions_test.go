package calculator

import (
	"os"
	"testing"
)

// TestVarunastraWritesAddedUsingFlags pins CalcPerform.lua:156-165 and :194-203.
func TestVarunastraWritesAddedUsingFlags(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	t.Run("varunastra-marks-every-added-condition", func(t *testing.T) {
		for _, build := range []string{
			"2456-ladder-DubsMan_7649.Ekkicrush.xml",
			"5784-ladder-Polihraf_5433.Straponten.xml",
			"7838-ladder-nahatorihito_7396.CotA_nahatorihito.xml",
		} {
			t.Run(build, func(t *testing.T) {
				env := corpusEnvForLoadedData(t, build)
				actor := env.Player
				if (actor.WeaponData1 == nil || !actor.WeaponData1.CountsAsAll1H) &&
					(actor.WeaponData2 == nil || !actor.WeaponData2.CountsAsAll1H) {
					t.Fatalf("REFUSING: %s carries no countsAsAll1H weapon, so it no longer "+
						"exercises this arm", build)
				}
				for _, cond := range []string{
					"UsingAxe", "UsingSword", "UsingDagger", "UsingMace", "UsingClaw",
				} {
					if !actor.ModDB.Conditions[cond] {
						t.Errorf("%s: %s is not set — the all-one-handed block did not run",
							build, cond)
					}
					added, present := actor.AddedUsingCondition(cond)
					if !present {
						t.Errorf("%s: Added%s is absent; ModStore.lua:595's `~= nil` test can "+
							"never fire and the arm is a no-op", build, cond)
						continue
					}
					if !added {
						t.Errorf("%s: Added%s is false — the condition is being reported as "+
							"genuinely satisfied by another weapon", build, cond)
					}
				}
			})
		}
	})

	t.Run("ordinary-weapon-reports-absent", func(t *testing.T) {
		for _, build := range []string{
			"0027-saboteur-spark-of-the-nova.xml",
			"0682-gladiator-herald-of-agony.xml",
		} {
			t.Run(build, func(t *testing.T) {
				env := corpusEnvForLoadedData(t, build)
				actor := env.Player
				if (actor.WeaponData1 != nil && actor.WeaponData1.CountsAsAll1H) ||
					(actor.WeaponData2 != nil && actor.WeaponData2.CountsAsAll1H) {
					t.Fatalf("REFUSING: %s carries a countsAsAll1H weapon, so it cannot be the "+
						"negative control", build)
				}
				for _, cond := range []string{
					"UsingAxe", "UsingSword", "UsingDagger", "UsingMace", "UsingClaw",
				} {
					if _, present := actor.AddedUsingCondition(cond); present {
						t.Errorf("%s: Added%s is PRESENT on a character with no all-one-handed "+
							"weapon; every negated %s tag would take the Varunastra arm",
							build, cond, cond)
					}
				}
			})
		}
	})
}
