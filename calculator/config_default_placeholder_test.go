package calculator

import "testing"

// TestDefaultPlaceholderStateSeedsEveryVar pins ConfigTab.lua:1236.
func TestDefaultPlaceholderStateSeedsEveryVar(t *testing.T) {
	general := map[string]float64{
		"sigilOfPowerStages":                          1,
		"configSpectralTigerCount":                    5,
		"multiplierWitheredStackCountSelf":            15,
		"meleeDistance":                               15,
		"projectileDistance":                          40,
		"ScorchStacks":                                1,
		"ShockStacks":                                 1,
		"multiplierEnemyHallowingFlame":               1,
		"multiplierHallowingFlameStacksRemovedByAlly": 1,
	}

	for name, want := range general {
		got, ok := configDefaultPlaceholders[name]
		if !ok {
			t.Errorf("%s: absent from configDefaultPlaceholders — ConfigTab.lua:1236 seeds it "+
				"unconditionally, so PoB has it on every build", name)
			continue
		}
		if got != want {
			t.Errorf("%s = %g, ConfigOptions.lua declares %g", name, got, want)
		}
	}

	for _, name := range []string{"enemyDamageRollRange", "enemySpeed", "enemyCritChance", "enemyCritDamage", "enemyPhysicalDamage"} {
		if _, ok := configDefaultPlaceholders[name]; !ok {
			t.Errorf("%s: absent from configDefaultPlaceholders", name)
		}
	}
}
