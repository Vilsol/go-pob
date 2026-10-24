package main

import "testing"

// TestEmitTagSkillNameCarriesEveryEvaluatedField pins the SkillName arm of ssmRun.emitTag against the fields moddb actually evaluates.
func TestEmitTagSkillNameCarriesEveryEvaluatedField(t *testing.T) {
	t.Parallel()
	r := &ssmRun{}
	for _, tc := range []struct {
		name string
		tag  map[string]any
		want string
	}{
		{"plain", map[string]any{"type": "SkillName", "skillName": "Cyclone"},
			`mod.SkillName("Cyclone")`},
		{"neg", map[string]any{"type": "SkillName", "skillName": "Cyclone", "neg": true},
			`mod.SkillName("Cyclone").Neg(true)`},
		{"transfigured", map[string]any{"type": "SkillName", "skillName": "Summon Stone Golem", "includeTransfigured": true},
			`mod.SkillName("Summon Stone Golem").Transfigured(true)`},
		{"summonSkill", map[string]any{"type": "SkillName", "skillName": "Raise Spectre", "summonSkill": true},
			`mod.SkillName("Raise Spectre").SummonSkill(true)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := r.emitTag(tc.tag)
			if err != nil {
				t.Fatalf("emitTag(%v) errored: %v", tc.tag, err)
			}
			if got != tc.want {
				t.Fatalf("emitTag(%v)\n  got  %s\n  want %s", tc.tag, got, tc.want)
			}
		})
	}
}
