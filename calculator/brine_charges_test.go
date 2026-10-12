package calculator

import "testing"

// TestBrineChargesFromEnduranceCharges: with CanGainBrineCharges, used endurance charges also fill brine charges and set Multiplier:BrineCharge.
func TestBrineChargesFromEnduranceCharges(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0699-juggernaut-consecrated-path-of-endurance-2.xml")
	if got := env.Player.Output["BrineCharges"]; got != 18 {
		t.Errorf("BrineCharges = %v, want 18 (PoB)", got)
	}
}
