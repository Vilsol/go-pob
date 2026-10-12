package calculator

import (
	"math"
	"os"
	"testing"
)

// TestTotemDefensiveOutputsAreProduced pins CalcOffence.lua:1407-1409, three consecutive output keys that this tree simply did not write.
//
//	output.TotemEnergyShield = skillModList:Sum("BASE", skillCfg, "TotemEnergyShield")
//	output.TotemBlockChance  = skillModList:Sum("BASE", skillCfg, "TotemBlockChance")
//	output.TotemArmour       = skillModList:Sum("BASE", skillCfg, "TotemArmour")
func TestTotemDefensiveOutputsAreProduced(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build                     string
		armour, energyShield, blk float64
	}{
		{"12598-ladder-Vepar90_3537.VeparAllflame_II.xml", 2068, 295, 0},
		{"0567-hierophant-storm-burst-of-repulsion-4.xml", 0, 0, 25},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			for _, k := range []struct {
				name string
				want float64
			}{
				{"TotemArmour", tc.armour},
				{"TotemEnergyShield", tc.energyShield},
				{"TotemBlockChance", tc.blk},
			} {
				got := env.Player.Output[k.name]
				if math.Abs(got-k.want) > 1e-6 {
					t.Errorf("%s = %v, want %v (PoB) — CalcOffence.lua:1407-1409", k.name, got, k.want)
				}
			}
		})
	}
}
