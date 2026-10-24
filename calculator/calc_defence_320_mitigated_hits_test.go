package calculator

import (
	"testing"
)

// TestMitigatedHitsEqualUnmitigatedAtFullDamageChance: CalcDefence.lua:2884 does NOT always run the hit search.
//
//	output["NumberOfMitigatedDamagingHits"] = (output["ConfiguredDamageChance"] ~= 100
//		or DamageIn["TrackRecoupable"] or DamageIn["TrackLifeLossOverTime"] or DamageIn.GainWhenHit)
//		and numberOfHitsToDie(DamageIn) or output["NumberOfDamagingHits"]
func TestMitigatedHitsEqualUnmitigatedAtFullDamageChance(t *testing.T) {
	for _, tc := range []struct {
		build    string
		fallback bool
	}{
		{"3065-ladder-Keyzoner_4232.Curse_ofCWS.xml", true},
		{"0264-juggernaut-earthshatter.xml", false},
	} {
		env := defenceEnv(t, tc.build)
		oracle, err := oracleForLoadedData()
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		out := oracle[tc.build].Output
		if out["ConfiguredDamageChance"] != 100 {
			t.Fatalf("REFUSING: PoB's ConfiguredDamageChance for %s is %v, not 100 — wrong fixture",
				tc.build, out["ConfiguredDamageChance"])
		}
		if took := out["NumberOfMitigatedDamagingHits"] == out["NumberOfDamagingHits"]; took != tc.fallback {
			t.Fatalf("REFUSING: PoB's fallback on %s is %v, the fixture expects %v", tc.build, took, tc.fallback)
		}
		for _, stat := range []string{"NumberOfDamagingHits", "NumberOfMitigatedDamagingHits"} {
			if got := env.Player.Output[stat]; !oracleClose(got, out[stat], oracleTightTolerance) {
				t.Errorf("%s: %s = %.10g, PoB %.10g", tc.build, stat, got, out[stat])
			}
		}
	}
}
