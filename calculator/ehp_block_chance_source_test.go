package calculator

import (
	"os"
	"testing"
)

// TestEHPBlockChanceReadsTheEffectiveOutputs pins CalcDefence.lua:2748-2755.
//
//	local BlockChance = output.EffectiveBlockChance / 100
//	if damageCategoryConfig ~= "Melee" and damageCategoryConfig ~= "Untyped" then
//	    BlockChance = output["Effective"..damageCategoryConfig.."BlockChance"] / 100
//	end
//	if enemyDB:Flag(nil, "CannotBeBlocked") then BlockChance = 0 end
func TestEHPBlockChanceReadsTheEffectiveOutputs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build string
		want  float64
		why   string
	}{
		{"0702-necromancer-summon-holy-relic-of-conviction-9.xml", 9.99424,
			"3.1250 cluster (227 builds); was 31.232"},
		{"0555-elementalist-corrupting-fever-6.xml", 11.245,
			"2.9791 cluster (125 builds); was 33.5"},
		{"0611-necromancer-summon-holy-relic-of-conviction-8.xml", 8.82,
			"3.3333 cluster (70 builds); was 29.4"},

		{"0003-guardian-unknown.xml", 28, "CONTROL: high-block build already correct"},
		{"0001-necromancer-unknown.xml", 54.5, "CONTROL: must not move"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output["ConfiguredDamageChance"]
			if !ok {
				t.Fatal("ConfiguredDamageChance ABSENT; CalcDefence.lua:2839 writes it")
			}
			if d := got - tc.want; d > 0.005*tc.want || d < -0.005*tc.want {
				t.Errorf("ConfiguredDamageChance = %v, want %v (3.29 oracle) — %s", got, tc.want, tc.why)
			}
		})
	}
}
