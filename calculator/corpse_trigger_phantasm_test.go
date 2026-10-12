package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestConsumeCorpseTriggerResolvesPhantasmBranch pins ModParser.lua:3295's `skill == "summon phantasm skill"` branch, which was dead here.
func TestConsumeCorpseTriggerResolvesPhantasmBranch(t *testing.T) {
	for _, tc := range []struct {
		line    string
		skillID string
		level   int
	}{
		{"Trigger Level 20 Summon Phantasm Skill when you Consume a Corpse", "TriggeredSummonGhostOnKill", 20},
		{"trigger level 20 summon phantasm skill when you consume a corpse", "TriggeredSummonGhostOnKill", 20},
		{"Trigger Level 10 Bone Offering when you Consume a Corpse", "BoneOffering", 10},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v, want 1 mod", tc.line, e)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			es, ok := e.ModList[0].Value().List().(mod.ExtraSkill)
			if !ok {
				t.Fatalf("value is %T, want mod.ExtraSkill", e.ModList[0].Value().List())
			}
			if es.SkillID != tc.skillID {
				t.Errorf("SkillID = %q, want %q", es.SkillID, tc.skillID)
			}
			if es.Level != tc.level {
				t.Errorf("Level = %d, want %d", es.Level, tc.level)
			}
			if !es.Triggered {
				t.Error("Triggered = false, want true")
			}
		})
	}
}
