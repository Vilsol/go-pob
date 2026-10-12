package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestDualWieldChanceRecombinesPerHand: dualWieldChance is CalcTriggers.lua:737-747 / :762-772.
func TestDualWieldChanceRecombinesPerHand(t *testing.T) {
	env := &Environment{Player: &Actor{
		WeaponData1: &SkillData{Type: "Claw"},
		WeaponData2: &SkillData{Type: "Claw"},
	}}
	flagged := &ActiveSkill{SkillData: &SkillData{DoubleHitsWhenDualWielding: true}}

	testza.AssertEqual(t, float64(90), dualWieldChance(env, flagged, 65, 80, 50),
		"the recombination must be both + main*offMiss + mainMiss*off, and must ignore the "+
			"combined figure it is handed")

	testza.AssertEqual(t, float64(100), dualWieldChance(env, flagged, 100, 100, 100))
	testza.AssertEqual(t, float64(50), dualWieldChance(env, flagged, 50, 0, 50))
}

// TestDualWieldChanceFallsThroughWithoutBothConditions: Both guards, each alone, must fall through to the combined value — the else arm at :748 / :773.
func TestDualWieldChanceFallsThroughWithoutBothConditions(t *testing.T) {
	dual := &Actor{WeaponData1: &SkillData{Type: "Claw"}, WeaponData2: &SkillData{Type: "Claw"}}
	flagged := &ActiveSkill{SkillData: &SkillData{DoubleHitsWhenDualWielding: true}}
	unflagged := &ActiveSkill{SkillData: &SkillData{}}

	oneHand := &Actor{WeaponData1: &SkillData{Type: "Claw"}, WeaponData2: &SkillData{}}
	testza.AssertEqual(t, float64(65),
		dualWieldChance(&Environment{Player: oneHand}, flagged, 65, 80, 50),
		"`env.player.weaponData2.type` is a property of the PLAYER; without it upstream takes "+
			"the else arm even though the skill carries the flag")

	testza.AssertEqual(t, float64(65),
		dualWieldChance(&Environment{Player: dual}, unflagged, 65, 80, 50),
		"without skillData.doubleHitsWhenDualWielding the hands split one instance and the "+
			"combined figure is already correct")
}
