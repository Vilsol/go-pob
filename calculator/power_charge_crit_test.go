package calculator

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestPowerChargeCritUsesTheGeneratedConstant covers CalcSetup.lua:511 and CalcPerform.lua:1152.
//
//	modDB:NewMod("CritChance", "INC", data.characterConstants["critical_strike_chance_+%_per_power_charge"], "Base", { type = "Multiplier", var = "PowerCharge" })
func TestPowerChargeCritUsesTheGeneratedConstant(t *testing.T) {
	testza.AssertEqual(t, float64(50), float64(data.CriticalStrikeChancePlusPercentPerPowerCharge),
		"Misc.lua:78 — re-derive this test if the pin moved")

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true for the rest")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range []string{
		"1564-hierophant-fireball-2.xml",
		"1585-hierophant-fireball-3.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok || rec.TreeSources < oracleMinTreeSources {
				t.Fatalf("REFUSING: %s is not graded", build)
			}
			env := corpusEnvForLoadedData(t, build)

			charges := env.Player.ModDB.Multipliers["PowerCharge"]
			if charges <= 0 {
				t.Fatalf("REFUSING: %s has %v power charges; the constant is unobservable there",
					build, charges)
			}

			var found bool
			for _, m := range env.Player.ModDB.Mods["CritChance"] {
				if m.Type() == mod.TypeIncrease && m.GetSource() == mod.SourceBase {
					found = true
					testza.AssertEqual(t, float64(data.CriticalStrikeChancePlusPercentPerPowerCharge),
						m.Value().Float(), "the Base per-power-charge CritChance mod")
				}
			}
			if !found {
				t.Fatal("REFUSING: no Base INC CritChance mod on the player; the producer moved")
			}

			want := rec.Output["PreEffectiveCritChance"]
			got := env.Player.Output["PreEffectiveCritChance"]
			if !oracleClose(got, want, oracleDumpTolerance) {
				t.Errorf("PreEffectiveCritChance = %v, want %v (CalcSetup.lua:511)", got, want)
			}
		})
	}
}
