package calculator

import "testing"

// TestItemGrantedNoSupportsGroup: a group for an item skill granted with noSupports takes no supports from its slot.
func TestItemGrantedNoSupportsGroup(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "8556-ladder-yingjieabc_6667."+string([]rune{0x84dd, 0x5973, 0x4eba, 0x53f7})+".xml")
	for k, want := range map[string]float64{"ManaReserved": 1178, "ManaUnreserved": 23} {
		if got := env.Player.Output[k]; got != want {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}
