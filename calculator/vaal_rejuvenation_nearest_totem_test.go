package calculator

import "testing"

// TestVaalRejuvenationTotemIsTheNearestTotem pins CalcDefence.lua:2493-2498.
func TestVaalRejuvenationTotemIsTheNearestTotem(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "12484-ladder-Tryllekat_6173.Badger_StickMerc")
	out := env.Player.Output

	if got := out["VaalRejuvenationTotemAllyDamageMitigation"]; got != 45 {
		t.Errorf("VaalRejuvenationTotemAllyDamageMitigation = %v, want 45", got)
	}
	if got := out["TotemAllyDamageMitigation"]; got != 0 {
		t.Errorf("TotemAllyDamageMitigation = %v, want 0 (folded into Vaal Rejuvenation)", got)
	}
	if _, ok := out["TotalTotemLife"]; ok {
		t.Errorf("TotalTotemLife is published (%v); :2497 deletes it", out["TotalTotemLife"])
	}
}
