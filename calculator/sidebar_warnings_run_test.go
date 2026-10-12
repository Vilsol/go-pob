package calculator

import (
	"reflect"
	"testing"
)

// TestSidebarWarningsRunSteps: Stepping a SidebarWarningsRun gives what SidebarWarnings gives, one calculated skill per Step.
func TestSidebarWarningsRunSteps(t *testing.T) {
	for _, file := range []string{"0201-elementalist-vaal-cold-snap-2.xml", "0011-saboteur-cyclone.xml"} {
		t.Run(file, func(t *testing.T) {
			want := sidebarEnv(t, file).SidebarWarnings()
			r := sidebarEnv(t, file).NewSidebarWarningsRun()
			steps := 0
			for r.Step() {
				steps++
				if !r.loopEnvUsed {
					t.Fatalf("step %d returned before calculating a skill", steps)
				}
			}
			if steps < 2 {
				t.Fatalf("%d steps: the run did not yield between calculated skills", steps)
			}
			if got := r.Result(); !reflect.DeepEqual(got, want) {
				t.Errorf("stepped %+v, one call %+v", got, want)
			}
		})
	}
}

// TestSidebarWarningsReuseFullDPSCosts: The skills calcFullDPS already calculated are not calculated again for the warnings.
func TestSidebarWarningsReuseFullDPSCosts(t *testing.T) {
	for _, file := range []string{"0001-necromancer-unknown.xml", "0008-necromancer-vaal-summon-skeletons.xml"} {
		t.Run(file, func(t *testing.T) {
			env := sidebarEnv(t, file)
			reused := env.fullDPSSkillCosts
			if len(reused) == 0 {
				t.Fatal("calcFullDPS recorded no skill costs")
			}
			served := map[int]bool{}
			for index, skill := range env.Player.ActiveSkillList {
				_, triggerCached := env.triggerCache.entries[cacheSkillUUID(env, skill)]
				if _, ok := reused[index]; ok && skill != env.Player.MainSkill && !triggerCached {
					served[index] = true
				}
			}
			if len(served) == 0 {
				t.Fatal("no recorded skill would be read by SidebarWarnings")
			}

			env.fullDPSSkillCosts = nil
			r := env.NewSidebarWarningsRun()
			for index := range env.Player.ActiveSkillList {
				if !served[index] {
					continue
				}
				want := reused[index]
				skill := env.Player.ActiveSkillList[index]
				costs, life, mana, ok, _ := r.skillCosts(index, skill)
				got := skillCostEntry{costs: costs, lifeReservedPercent: life, manaReservedPercent: mana}
				if !ok || !reflect.DeepEqual(got, want) {
					t.Errorf("skill %d %s: reused %+v, calculated %+v", index, skill.ActiveEffect.GrantedEffect.Name(), want, got)
				}
			}

			calculated := sidebarEnv(t, file)
			calculated.fullDPSSkillCosts = nil
			if want, got := calculated.SidebarWarnings(), sidebarEnv(t, file).SidebarWarnings(); !reflect.DeepEqual(got, want) {
				t.Errorf("warnings with reuse %+v, without %+v", got, want)
			}
		})
	}
}
