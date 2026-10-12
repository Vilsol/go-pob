package calculator

import "testing"

// TestKalandraMirrorsOtherRingAtOwnSlot: Kalandra's Touch copies otherRing.slotModList[slot.slotNum], its own slot (CalcSetup.lua:1090).
func TestKalandraMirrorsOtherRingAtOwnSlot(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "6012-ladder-Scccxd_6309.Ya_Doldayob.xml")
	if got := env.Player.Output["Life"]; got != 4131 {
		t.Errorf("Life = %v, want 4131 (PoB)", got)
	}
}
