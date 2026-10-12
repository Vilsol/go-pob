package calculator

import (
	"os"
	"path/filepath"
	"testing"
)

// TestESPerSecondCostIsStableUnderEldritchBattery pins the conjunct at CalcOffence.lua:5796 that disables the `PerSecondHasCost` skip while Eldritch Battery is redirecting Mana onto ES.
func TestESPerSecondCostIsStableUnderEldritchBattery(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const (
		reevaluations = 20
		wantBuilds    = 1
	)

	const knownCarrier = "6101-ladder-Sirmapalot_2198.UnLongNeckSeen.xml"

	_, names := corpusNamesForLoadedData(t)
	if _, err := os.Stat(filepath.Join(corpusDirForLoadedData(t), knownCarrier)); err == nil {
		names = append([]string{knownCarrier}, names...)
	}

	var checked int
	for _, name := range names {
		env := corpusEnvForLoadedData(t, name)
		if env == nil || env.Player == nil {
			continue
		}
		if !env.ModDB.Flag(nil, "EnergyShieldProtectsMana") {
			continue
		}
		if env.Player.Output["ManaCost"] <= 0 || env.Player.Output["ESCost"] <= 0 {
			continue
		}
		want, ok := env.Player.Output["ESPerSecondCost"]
		if !ok || want == 0 {
			continue
		}

		for i := 0; i < reevaluations; i++ {
			again := corpusEnvForLoadedData(t, name)
			if again == nil || again.Player == nil {
				t.Fatalf("%s: rebuilt to a nil environment on iteration %d", name, i)
			}
			if got := again.Player.Output["ESPerSecondCost"]; got != want {
				t.Fatalf("%s: ESPerSecondCost is %v on the first evaluation and %v on iteration %d.\n"+
					"  The per-second cost loop ranges a MAP and its skip guard reads a different key "+
					"than its write, so Mana's redirect onto ES suppresses the ES iteration only when "+
					"the range happens to yield Mana first. CalcOffence.lua:5796 disables that skip "+
					"under Eldritch Battery; the conjunct must be present.", name, want, got, i)
			}
		}

		checked++
		if checked >= wantBuilds {
			break
		}
	}

	if checked == 0 {
		t.Fatal("REFUSING: no build in the corpus carried Eldritch Battery with BOTH a Mana and an " +
			"ES upfront cost, so this test never entered the branch it exists to pin")
	}
	t.Logf("re-evaluated %d Eldritch Battery builds %d times each with a stable ESPerSecondCost",
		checked, reevaluations)
}
