package calculator

import (
	"math"
	"os"
	"testing"
)

// TestCurseAuraSlotsBeyondManaPool pins CalcPerform.lua:3265-3272 on an edited build PoB dumped: a Blasphemy
// curse reserving more than the player's mana is still slotted.
func TestCurseAuraSlotsBeyondManaPool(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.30")
	const anchor = "{crafted}+22% to Fire and Lightning Resistances\n"
	env := corpusEnvWithEdit(t, "0059-occultist-viper-strike", anchor, anchor+"-2000 to maximum Mana\n")
	if env.Player.Output["Mana"] >= 0 {
		t.Fatalf("PREMISE BROKEN: mana %v is not below zero", env.Player.Output["Mana"])
	}
	// PoB's dump of this edit.
	const want = 1.0757392307718e+07
	if got := env.Player.Output["TotalPoisonDPS"]; math.Abs(got-want) > 1e-6*want {
		t.Errorf("TotalPoisonDPS = %v, PoB %v", got, want)
	}
}
