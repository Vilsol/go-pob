package calculator

import (
	"testing"
)

// TestClawDamageAppliesToUnarmedTabulatesHits: Rigwald's Curse tabulates claw Damage with Claw|Hit flags.
func TestClawDamageAppliesToUnarmedTabulatesHits(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1329-gladiator-cyclone.xml")
	for k, want := range map[string]float64{"ImpaleDPS": 24565373.916712, "TotalDPS": 17528928.168686} {
		if got := env.Player.Output[k]; !oracleClose(got, want, oracleDumpTolerance) {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}
