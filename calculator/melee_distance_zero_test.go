package calculator

import (
	"testing"
)

// TestMeleeDistanceZeroIsADistance: meleeDistance 0 is a set skillDist.
func TestMeleeDistanceZeroIsADistance(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0990-juggernaut-static-strike-2.xml")
	if want, got := 17912408.4748, env.Player.Output["TotalDPS"]; !oracleClose(got, want, oracleDumpTolerance) {
		t.Errorf("TotalDPS = %v, want %v (PoB)", got, want)
	}
}
