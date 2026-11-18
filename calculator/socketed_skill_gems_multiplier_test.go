package calculator

import (
	"math"
	"testing"
)

// TestSocketedSkillGemsInScalesSocketedDamage: CalcSetup.lua:1368-1390 counts the non-support gems that fit in an
// item's sockets into SocketedSkillGemsIn<slot>, which "socketed gems deal N% more damage per socketed skill gem"
// reads. The want is the pinned PoB's TotalDPS with that line added to the body armour.
//
//pob:block:src/Modules/CalcSetup.lua:1368-1390
func TestSocketedSkillGemsInScalesSocketedDamage(t *testing.T) {
	env := corpusEnvWithEdit(t, "0006-elementalist-storm-burst",
		"10% increased maximum Life\n+32% to Chaos Resistance\n",
		"10% increased maximum Life\nSocketed Gems deal 20% more damage per Socketed Skill Gem\n+32% to Chaos Resistance\n")
	if got := env.ItemModDB.Multipliers["SocketedSkillGemsInBody Armour"]; got != 1 {
		t.Errorf("SocketedSkillGemsInBody Armour = %v, PoB 1", got)
	}
	if got, want := env.Player.Output["TotalDPS"], 174901.0356288; math.Abs(got-want) > 1e-6*want {
		t.Errorf("TotalDPS = %v, PoB %v", got, want)
	}
}
