package calculator

import (
	"math"
	"os"
	"testing"

	raw2 "github.com/Vilsol/go-pob/data/raw"
)

// TestMinionSupportLevelArm pins the THIRD arm of CalcActiveSkill.lua:707-709, and the bookkeeping at :660 and :671-680 that feeds it.
//
//	for _, minionType in ipairs(skillEffect.grantedEffect.addMinionList) do
//		if not isValueInArray(minionList, minionType) then
//			t_insert(minionList, minionType)
//		end
//	end
func TestMinionSupportLevelArm(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build      string
		wantArmour float64
	}{
		{"1492-reliquarian-vaal-fireball.xml", 16419},
		{"1593-reliquarian-frostblink.xml", 0},
	} {
		build := tc.build
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			if env.Minion == nil {
				t.Fatal("no minion actor built; PoB builds one for this fixture")
			}

			fromSupport := false
			for _, se := range env.Player.MainSkill.EffectList {
				if se.GrantedEffect == nil || !se.GrantedEffect.Raw.IsSupport {
					continue
				}
				for _, minionType := range raw2.SupportAddMinionLists[se.GrantedEffect.ID()] {
					if minionType == env.Minion.Type {
						fromSupport = true
					}
				}
			}
			if !fromSupport {
				t.Fatalf("premise broken: minion %q is not added by any support on this build, "+
					"so minionSupportLevel is not what sets its level — re-pick the fixture",
					env.Minion.Type)
			}

			if got := env.Minion.Level; got != 72 {
				t.Errorf("minion level = %d, want 72 (Summon Phantasm Support's levelRequirement, "+
					"not the active gem's 70)", got)
			}
			if tc.wantArmour != 0 {
				if got := env.Minion.Output["Armour"]; math.Abs(got-tc.wantArmour) > 0.5 {
					t.Errorf("Minion.Armour = %v, PoB %v = monsterArmourTable[72] "+
						"(monsterArmourTable[70] would be 14047)", got, tc.wantArmour)
				}
			}
		})
	}
}
