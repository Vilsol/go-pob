package calculator

import (
	"testing"
)

// TestInitialNodeModDBMergesKeystones: keystones are merged into initialNodeModDB.
func TestInitialNodeModDBMergesKeystones(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0278-reliquarian-vaal-arc-5.xml")
	for k, want := range map[string]float64{"Gear:Armour": 1254, "Gear:EnergyShield": 36} {
		if got := env.Player.Output[k]; got != want {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}
