package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
)

// TestSkillStatMapsMultiplierThresholdHasASource pins the ONE invariant evalMultiplierThresholdTag.
func TestSkillStatMapsMultiplierThresholdHasASource(t *testing.T) {
	for id, statMap := range raw.SkillStatMaps {
		for stat, entry := range statMap {
			for _, m := range entry.Mods {
				for _, tag := range m.Tags() {
					mt, ok := tag.(*mod.MultiplierThresholdTag)
					if !ok {
						continue
					}
					if mt.TagThreshold == nil && mt.ThresholdVariable == nil {
						t.Errorf("%s / %s: MultiplierThreshold(%q) carries neither Threshold "+
							"nor ThresholdVar; evalMultiplierThresholdTag will nil-deref",
							id, stat, mt.Variable)
					}
				}
			}
		}
	}
}

// TestSkillStatMapsCircleOfPowerThresholdVar pins the single row that found the defect.
func TestSkillStatMapsCircleOfPowerThresholdVar(t *testing.T) {
	const (
		id   = "SigilOfPower"
		stat = "circle_of_power_enemy_damage_+%_final_at_max_stages"
	)

	entry := raw.SkillStatMaps[id][stat]
	if entry == nil {
		t.Fatalf("REFUSING: %s / %s is absent from SkillStatMaps. This is the only "+
			"thresholdVar producer in the pinned PoB; if it is gone the pin moved.", id, stat)
	}

	var found *mod.MultiplierThresholdTag
	for _, m := range entry.Mods {
		for _, tag := range m.Tags() {
			if mt, ok := tag.(*mod.MultiplierThresholdTag); ok {
				found = mt
			}
		}
	}
	if found == nil {
		t.Fatalf("REFUSING: %s / %s carries no MultiplierThreshold tag at all", id, stat)
	}

	if found.ThresholdVariable == nil {
		t.Fatalf("%s / %s: ThresholdVariable is nil, want %q", id, stat, "SigilOfPowerMaxStages")
	}
	if *found.ThresholdVariable != "SigilOfPowerMaxStages" {
		t.Errorf("%s / %s: ThresholdVariable = %q, want %q",
			id, stat, *found.ThresholdVariable, "SigilOfPowerMaxStages")
	}
	if found.TagActor != "enemy" {
		t.Errorf("%s / %s: Actor = %q, want %q", id, stat, found.TagActor, "enemy")
	}
}
