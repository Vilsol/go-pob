package calculator

import (
	"os"
	"testing"
)

// TestEnergyShieldRechargeOverride pins CalcDefence.lua:1331: a clone's EnergyShieldRecharge OVERRIDE 0 (Minions.lua:465) replaces the recharge base.
func TestEnergyShieldRechargeOverride(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	for _, name := range []string{"0653-scion-none-mirror-arrow-of-bombarding-clones.xml", "2089-ladder-Aragos_7557.RzultoMordoDrugo.xml"} {
		t.Run(name, func(t *testing.T) {
			rec, ok := oracle[name]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", name)
			}
			want, ok := rec.Minion["EnergyShieldRecharge"]
			if !ok || rec.Minion["EnergyShield"] == 0 {
				t.Fatalf("REFUSING: %s has no minion energy shield to recharge", name)
			}
			env := corpusEnvForLoadedData(t, name)
			if env.Minion == nil {
				t.Fatalf("REFUSING: we built no minion for %s", name)
			}
			if got := env.Minion.Output["EnergyShieldRecharge"]; !oracleClose(got, want, oracleDumpTolerance) {
				t.Errorf("Minion.EnergyShieldRecharge = %g, PoB %g", got, want)
			}
		})
	}
}
