package raw

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestWarcryStatMapsCarryDivAndLimit pins the nine `effectType = "Warcry"` mods that carry a `div`/`limit` pair in the pinned PoB's skill data.
func TestWarcryStatMapsCarryDivAndLimit(t *testing.T) {
	want := map[string]map[string][2]float64{
		"AncestralCry": {
			"ancestral_cry_elemental_resist_%_per_5_power_up_to_cap":          {5, 30},
			"ancestral_cry_maximum_elemental_resist_%_per_10_power_up_to_cap": {10, 30},
		},
		"BattlemagesCry":  {"divine_cry_additional_base_critical_strike_chance_per_5_power_up_to_cap": {5, 25}},
		"EnduringCry":     {"enduring_cry_life_regeneration_rate_per_minute_%_per_5_power_up_to_cap": {5, 25}},
		"InfernalCry":     {"infernal_cry_physical_damage_%_to_add_as_fire_per_5_power_up_to_cap": {5, 25}},
		"IntimidatingCry": {"intimidating_cry_movement_speed_+%_per_5_power_up_to_cap": {5, 30}},
		"SeismicCry": {
			"seismic_cry_stun_threshold_+%_per_5_power_up_to_cap": {5, 25},
			"seismic_cry_armour_+%_final_per_5_power_up_to_cap":   {5, 25},
		},
		"VengefulCry": {"rage_warcry_gain_X_rage_per_minute_per_5_monster_power_max_25_power": {5, 25}},
	}

	found := 0
	for skill, stats := range want {
		perSkill := SkillStatMaps[skill]
		if perSkill == nil {
			t.Errorf("REFUSING: no per-skill stat map for %q; the fixture has moved", skill)
			continue
		}
		for stat, spec := range stats {
			entry := perSkill[stat]
			if entry == nil {
				t.Errorf("%s: no stat-map entry for %q", skill, stat)
				continue
			}
			var tag *mod.GlobalEffectTag
			for _, m := range entry.Mods {
				for _, raw := range m.Tags() {
					get, ok := raw.(*mod.GlobalEffectTag)
					if ok && len(get.GlobalEffectList) > 0 && get.GlobalEffectList[0] == "Warcry" && get.DivSet {
						tag = get
					}
				}
			}
			if tag == nil {
				t.Errorf("%s / %s: no GlobalEffect(\"Warcry\") mod carrying a div (entry has %d "+
					"mods); the emitter is still dropping it", skill, stat, len(entry.Mods))
				continue
			}
			found++
			if !tag.DivSet || tag.DivTag != spec[0] {
				t.Errorf("%s / %s: div = %v (set %v), want %v", skill, stat, tag.DivTag, tag.DivSet, spec[0])
			}
			if !tag.LimitSet || tag.LimitTag != spec[1] {
				t.Errorf("%s / %s: limit = %v (set %v), want %v", skill, stat, tag.LimitTag, tag.LimitSet, spec[1])
			}
		}
	}
	if found != 9 {
		t.Errorf("resolved %d of the pin's 9 div/limit warcry mods", found)
	}
}
