package calculator

import "testing"

// TestGlobalTriggerFlagMatchesPoB pins skillFlags.globalTrigger on the main skill: the focus handler sets it
// (CalcTriggers.lua:232), cast while channelling sets it (:399), and the generic handler sets it when the source rate
// is not used (:738). The wants are the pinned PoB's dumped skillFlags.
//
//pob:block:src/Modules/CalcTriggers.lua:232,src/Modules/CalcTriggers.lua:399,src/Modules/CalcTriggers.lua:734-739
func TestGlobalTriggerFlagMatchesPoB(t *testing.T) {
	for _, tc := range []struct {
		build string
		want  bool
	}{
		{"4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne", true},
		{"0011-saboteur-cyclone", true},
		{"0134-chieftain-detonate-dead", true},
		{"0379-luminary-convocation", true},
		{"0027-saboteur-spark-of-the-nova", false},
	} {
		env := corpusEnvForLoadedData(t, tc.build)
		ms := env.Player.MainSkill
		if !ms.SkillFlags[SkillFlagTriggered] {
			t.Fatalf("premise broken: %s's main skill is not triggered", tc.build)
		}
		if got := ms.SkillFlags[SkillFlagGlobalTrigger]; got != tc.want {
			t.Errorf("%s: globalTrigger = %v, PoB %v", tc.build, got, tc.want)
		}
	}
}
