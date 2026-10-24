package main

import "testing"

// TestSbmEmitTagSkillName pins the SkillName arm of sbmEmitTag, which was ABSENT.
func TestSbmEmitTagSkillName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		tag  map[string]any
		want string
	}{
		{"vaal cleave", map[string]any{
			"type": "SkillName", "skillName": "Cleave", "includeTransfigured": true,
		}, `mod.SkillName("Cleave").Transfigured(true)`},
		{"reaper consume", map[string]any{
			"type": "SkillName", "skillName": "Summon Reaper", "includeTransfigured": true,
		}, `mod.SkillName("Summon Reaper").Transfigured(true)`},
		{"neg neighbour", map[string]any{
			"type": "SkillName", "skillName": "Cleave", "neg": true,
		}, `mod.SkillName("Cleave").Neg(true)`},
		{"summonSkill neighbour", map[string]any{
			"type": "SkillName", "skillName": "Raise Spectre", "summonSkill": true,
		}, `mod.SkillName("Raise Spectre").SummonSkill(true)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sbmEmitTag(tc.tag, nil, nil)
			if err != nil {
				t.Fatalf("sbmEmitTag(%v) errored: %v", tc.tag, err)
			}
			if got != tc.want {
				t.Fatalf("sbmEmitTag(%v)\n  got  %s\n  want %s", tc.tag, got, tc.want)
			}
		})
	}
}
