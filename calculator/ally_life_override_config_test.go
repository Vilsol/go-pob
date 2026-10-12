package calculator

import "testing"

var allyLifeOverrideConfigs = []string{
	"TotalMinionLife",
	"TotalSpectreLife",
	"TotalTotemLife",
	"TotalRadianceSentinelLife",
	"TotalVoidSpawnLife",
	"TotalStoneGolemLife",
	"TotalVaalRejuvenationTotemLife",
}

// TestAllyLifeOverrideConfigsDispatch pins the block as a whole.
func TestAllyLifeOverrideConfigsDispatch(t *testing.T) {
	for _, name := range allyLifeOverrideConfigs {
		assertConfigDispatchable(t, name)
	}
}

// TestAllyLifeOverrideConfigsAreNotSeeded is the NEGATIVE half, and it is the half that matters for safety rather than coverage.
func TestAllyLifeOverrideConfigsAreNotSeeded(t *testing.T) {
	if !isConfigDefault("targetBrandedEnemy") {
		t.Fatalf("REFUSING: the control name is not a config default, so isConfigDefault is not " +
			"answering and the negatives below are vacuous")
	}

	for _, name := range allyLifeOverrideConfigs {
		if isConfigDefault(name) {
			t.Errorf("%s is in configDefaults; upstream declares no defaultState for it, and a "+
				"seeded handler writes OVERRIDE 0 on builds that set no override", name)
		}
		if isConfigPlaceholderVar(name) {
			t.Errorf("%s is in configPlaceholderVars; upstream declares no defaultPlaceholderState "+
				"for it", name)
		}
		if _, ok := configIndexDefaults[name]; ok {
			t.Errorf("%s is in configIndexDefaults; it is an integer override, not an index", name)
		}
		if _, ok := configCountDefaults[name]; ok {
			t.Errorf("%s is in configCountDefaults; upstream seeds no count for it", name)
		}
		if _, ok := configDefaultPlaceholders[name]; ok {
			t.Errorf("%s is in configDefaultPlaceholders; upstream declares no placeholder for it", name)
		}
	}
}
