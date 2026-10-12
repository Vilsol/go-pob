package calculator

import "testing"

// TestReservationDPS pins CalcOffence.lua:6137-6138: an enemy's life reserved scales CombinedDPS and shows as ReservationDPS.
func TestReservationDPS(t *testing.T) {
	assertBuildStats(t, "9320-ladder-Cavalorth_5118.CavaChef.xml", []string{"ReservationDPS", "CombinedDPS"}, nil)
}
