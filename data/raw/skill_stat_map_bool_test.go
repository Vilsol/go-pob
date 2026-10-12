package raw

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSkillStatMapBooleansAreNotZero pins oracle/skill_stat_map_emit.py's handling of a Lua boolean skillData value.
func TestSkillStatMapBooleansAreNotZero(t *testing.T) {
	wantTrue := map[string][]string{
		"modifiers_to_skill_effect_duration_also_affect_soul_prevention_duration": {"skillEffectAppliesToSoulGainPrevention"},
		"modifiers_to_buff_effect_duration_also_affect_soul_prevention_duration":  {"skillEffectAppliesToSoulGainPrevention"},
	}

	for statKey, dataKeys := range wantTrue {
		entry, ok := SkillStatMap[statKey]
		if !ok {
			t.Fatalf("REFUSING: %q is absent from SkillStatMap — this test cannot answer. "+
				"If PoB removed the stat, drop the row; do not weaken the assertion.", statKey)
		}
		for _, dataKey := range dataKeys {
			var found bool
			for _, m := range entry.Mods {
				mv := m.Value()
				if mv == nil || mv.ValueList == nil {
					continue
				}
				sd, isSD := mv.ValueList.(mod.SkillData)
				if !isSD || sd.Key != dataKey {
					continue
				}
				found = true
				if sd.Value != float64(1) {
					t.Errorf("%s -> skillData %q emitted Value=%v, want 1.\n"+
						"PoB declares it `true`; utils.Set turns anything but 1 into false, so this "+
						"row is silently disabled. Fix oracle/skill_stat_map_emit.py and run "+
						"`task data-regen` — do not hand-edit the generated file.",
						statKey, dataKey, sd.Value)
				}
			}
			if !found {
				t.Errorf("%s carries no SkillData mod for %q at all", statKey, dataKey)
			}
		}
	}
}
