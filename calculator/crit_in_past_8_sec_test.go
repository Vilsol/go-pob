package calculator

import (
	"os"
	"testing"
)

// TestElementalOverloadNeedsRealCritChance: `CritInPast8Sec` is what gates Elemental Overload's "40% more Elemental Damage".
//
//	CalcOffence.lua:3184-3186   if output.CritChance ~= 0 then
//	                            end
func TestElementalOverloadNeedsRealCritChance(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	check := func(t *testing.T, build string, wantCritZero bool) {
		t.Helper()
		rec, ok := oracle[build+".xml"]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob — an absent build cannot fail this "+
				"test, so it must not pass it either", build)
		}
		want, has := rec.Output["AverageHit"]
		if !has || want == 0 {
			t.Fatalf("REFUSING: the oracle carries no nonzero AverageHit for %s", build)
		}
		env := corpusEnvForLoadedData(t, build)
		if env == nil || env.Player == nil {
			t.Fatalf("REFUSING: %s produced no player environment", build)
		}
		if gotZero := env.Player.Output["CritChance"] == 0; gotZero != wantCritZero {
			t.Fatalf("REFUSING: %s has CritChance zero = %v, want %v — this build no longer "+
				"exercises the case it was chosen for", build, gotZero, wantCritZero)
		}
		got := env.Player.Output["AverageHit"]
		if !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("AverageHit = %g, want %g (ratio %.4f)", got, want, got/want)
		}
	}

	t.Run("crit-zero", func(t *testing.T) {
		for _, b := range []string{
			"1180-elementalist-herald-of-thunder-2",
			"1387-elementalist-herald-of-thunder-3",
			"2264-ladder-WztGjf_1792.Ariye",
		} {
			t.Run(b, func(t *testing.T) { t.Parallel(); check(t, b, true) })
		}
	})

	t.Run("crit-nonzero-control", func(t *testing.T) {
		for _, b := range []string{
			"0106-elementalist-lightning-tendrils-of-escalation",
			"0154-elementalist-winter-orb-7",
			"0602-juggernaut-discharge",
		} {
			t.Run(b, func(t *testing.T) { t.Parallel(); check(t, b, false) })
		}
	})
}
