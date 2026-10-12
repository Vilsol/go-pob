package calculator

import "testing"

// TestPlusVersionSupportReplacesItsBase.
func TestPlusVersionSupportReplacesItsBase(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "2275-ladder-_WitchDoctor__1069.I_Hate_PartyPlay.xml")
	if got := env.Player.Output["ProjectileCount"]; got != 5 {
		t.Errorf("ProjectileCount = %v, want 5 (PoB: Greater Multiple Projectiles supersedes Multiple Projectiles)", got)
	}
}
