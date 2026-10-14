package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

// TestEvadeChanceBasedOnWardReadsWard: CalcDefence.lua:1078-1086: with EvadeChanceBasedOnWard the evade chance reads Ward x percent instead of Evasion.
func TestEvadeChanceBasedOnWardReadsWard(t *testing.T) {
	t.Parallel()
	build := parseCorpusBuild(t, "0011-saboteur-cyclone.xml")
	before := GetMiscCalculator(build).Base().Player.Output
	if before["EvadeChance"] <= 0 || before["Ward"] != 0 {
		t.Fatalf("REFUSING: EvadeChance %v, Ward %v; the test needs evasion-based evade chance and no Ward", before["EvadeChance"], before["Ward"])
	}
	build.SetConfigOption(pob.Input{Name: "customMods", String: utils.Ptr("Chance to Evade Hits is based off of 50% of your Ward instead of your Evasion Rating")})
	after := GetMiscCalculator(build).Base().Player.Output
	if after["EvadeChance"] >= before["EvadeChance"] || after["MeleeEvadeChance"] != 0 {
		t.Fatalf("EvadeChance %v -> %v, MeleeEvadeChance %v; want the Ward-based (zero) evasion", before["EvadeChance"], after["EvadeChance"], after["MeleeEvadeChance"])
	}
}
