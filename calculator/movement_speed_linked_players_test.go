package calculator

import "testing"

// TestMovementSpeedEqualHighestLinkedPlayers pins CalcDefence.lua:1504: with the flag, the party's exported MovementSpeedMod replaces the player's own.
func TestMovementSpeedEqualHighestLinkedPlayers(t *testing.T) {
	env := (&Calculator{PoB: loadCorpusBuild(t, "15832-ascendant-static-strike.xml")}).BuildOutput(OutputModeMain)
	if got, want := env.Player.Output["MovementSpeedMod"], 2.71; got != want {
		t.Errorf("MovementSpeedMod = %v, want %v", got, want)
	}
}
