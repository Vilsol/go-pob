package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

func explodeTestBuild(groups ...pob.Skill) *pob.PathOfBuilding {
	build := &pob.PathOfBuilding{}
	build.Skills.SkillSets = []pob.SkillSet{{Skills: groups}}
	return build
}

func explodeGroup(gems ...pob.Gem) pob.Skill {
	source := "Explode"
	return pob.Skill{Label: "On Kill Monster Explosion", Enabled: true, SourceAttr: &source, Source: source, Gems: gems}
}

// TestProcessExplodeSocketGroup pins CalcSetup.lua:1405-1460 for the explode group: removed with no source, appended when absent, one EnemyExplode gem per source.
func TestProcessExplodeSocketGroup(t *testing.T) {
	plain := pob.Skill{Label: "main", Enabled: true, Gems: []pob.Gem{{SkillID: "Fireball", Level: 20, Enabled: true}}}
	one := []ExplodeSource{{NodeID: "1731"}}
	two := []ExplodeSource{{ModSource: "Item:-1:A"}, {ModSource: "Item:-1:B"}}

	build := explodeTestBuild(explodeGroup(pob.Gem{SkillID: "EnemyExplode", Level: 1, Enabled: true}), plain)
	processExplodeSocketGroup(build, nil)
	if groups := build.Skills.SkillSets[0].Skills; len(groups) != 1 || groups[0].Label != "main" {
		t.Errorf("no source: groups %+v, want the explode group removed", groups)
	}

	build = explodeTestBuild(plain)
	processExplodeSocketGroup(build, two)
	groups := build.Skills.SkillSets[0].Skills
	if len(groups) != 2 || groups[1].Source != "Explode" || groups[1].SourceAttr == nil || !groups[1].Enabled {
		t.Fatalf("absent group: groups %+v, want one appended", groups)
	}
	if len(groups[1].Gems) != 2 || groups[1].Gems[0].SkillID != "EnemyExplode" || groups[1].Gems[1].Level != 1 {
		t.Errorf("appended gems %+v, want two level-1 EnemyExplode", groups[1].Gems)
	}

	build = explodeTestBuild(explodeGroup(pob.Gem{SkillID: "EnemyExplode", Level: 1, Enabled: true}))
	processExplodeSocketGroup(build, two)
	if gems := build.Skills.SkillSets[0].Skills[0].Gems; len(gems) != 2 {
		t.Errorf("stale group: %d gems, want one per source", len(gems))
	}

	build = explodeTestBuild(explodeGroup(pob.Gem{SkillID: "EnemyExplode", Level: 1, Enabled: true, SkillPart: 3}))
	processExplodeSocketGroup(build, one)
	if gems := build.Skills.SkillSets[0].Skills[0].Gems; len(gems) != 1 || gems[0].SkillPart != 0 || gems[0].ExplodeSource() != "1731" {
		t.Errorf("XML-loaded gem: gems %+v, want it rebuilt (a saved gem carries no explodeSource)", gems)
	}

	build.Skills.SkillSets[0].Skills[0].Gems[0].SkillPart = 3
	processExplodeSocketGroup(build, one)
	if gems := build.Skills.SkillSets[0].Skills[0].Gems; len(gems) != 1 || gems[0].SkillPart != 3 {
		t.Errorf("gem created by an earlier pass: gems %+v, want it reused", gems)
	}
}

// TestEnemyExplodeMainSkill grades the pseudo skill against the oracle's Player table on four builds whose main socket group is the explode group.
func TestEnemyExplodeMainSkill(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	cases := []struct {
		build                                 string
		averageHit, hitChance, combinedToLife float64
	}{
		{"1347-elementalist-treeskill-3.xml", 87364.27272, 24, 153.5359130206},
		{"0267-chieftain-treeskill.xml", 179567.388, 10, 1053.6736461367},
		{"0318-elementalist-treeskill.xml", 9653.93, 100, 496.06092480787},
		{"15419-variant-0267.on-kill-explosion-part2.xml", 179567.388, 10, 1053.6736461367},
	}
	for _, c := range cases {
		env := corpusEnvForLoadedData(t, c.build)
		out := env.Player.Output
		for _, check := range []struct {
			key  string
			want float64
		}{
			{"AverageHit", c.averageHit},
			{"HitChance", c.hitChance},
			{"ExplodeChance", c.hitChance},
			{"CombinedAvgToMonsterLife", c.combinedToLife},
		} {
			if got := out[check.key]; math.Abs(got-check.want)/check.want > 1e-6 {
				t.Errorf("%s: %s = %v, PoB %v", c.build, check.key, got, check.want)
			}
		}
		if got := out["Speed"]; got != 0 {
			t.Errorf("%s: Speed = %v, PoB 0 (castTime = 0 is declared)", c.build, got)
		}
	}

	env := corpusEnvForLoadedData(t, "0803-ascendant-ethereal-knives.xml")
	if ms := env.Player.MainSkill; ms == nil || ms.ActiveEffect.GrantedEffect.Name() != "Ethereal Knives" {
		name := "<nil>"
		if ms != nil {
			name = ms.ActiveEffect.GrantedEffect.Name()
		}
		t.Errorf("0803: main skill %s, PoB Ethereal Knives", name)
	}
}
