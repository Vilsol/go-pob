package calculator

import "testing"

// TestDefaultStateSeedsEveryVar is the `defaultState` twin of TestDefaultPlaceholderStateSeedsEveryVar.
func TestDefaultStateSeedsEveryVar(t *testing.T) {
	checks := map[string]string{
		"targetBrandedEnemy":                   "310",
		"ConcPathBypassCD":                     "361",
		"FlickerStrikeBypassCD":                "447",
		"raiseSpectreEnableBuffs":              "577",
		"raiseSpectreEnableCurses":             "580",
		"summonElementalRelicEnableAngerAura":  "667",
		"summonElementalRelicEnableHatredAura": "670",
		"summonElementalRelicEnableWrathAura":  "673",
		"VigilantStrikeBypassCD":               "720",
		"conditionChampionIntimidate":          "1475",
		"conditionSacrificeMinion":             "1555",
	}
	counts := map[string]float64{
		"bloodsoakedBannerStages":      1,
		"conditionCorruptingCryStages": 1,
		"touchedDebuffsCount":          10,
		"maniaDebuffsCount":            15,
	}

	for name := range checks {
		if !isConfigDefault(name) {
			t.Errorf("%s declares defaultState upstream and is not in configDefaults — its apply never runs", name)
		}
	}
	for name, want := range counts {
		got, ok := configCountDefaults[name]
		if !ok {
			t.Errorf("%s declares a numeric defaultState upstream and is not in configCountDefaults", name)
			continue
		}
		if got != want {
			t.Errorf("configCountDefaults[%s] = %v, want %v", name, got, want)
		}
	}

	for name := range checks {
		assertConfigDispatchable(t, name)
	}
	for name := range counts {
		assertConfigDispatchable(t, name)
	}
}

// TestDefaultIndexSeedsEveryDispatchedVar is the same check for `defaultIndex`, which seeds a STRING from the option's `list` rather than a boolean.
func TestDefaultIndexSeedsEveryDispatchedVar(t *testing.T) {
	want := map[string]string{
		"resourceGainMode":  "AVERAGE",
		"plagueBearerState": "INC",
		"enemySizePreset":   "Medium",
		"repeatMode":        "AVERAGE",
		"enemyIsBoss":       "Pinnacle",
	}
	for name, val := range want {
		got, ok := configIndexDefaults[name]
		if !ok {
			t.Errorf("%s declares defaultIndex upstream and is not in configIndexDefaults — its apply never runs on a build that omits the input", name)
			continue
		}
		if got != val {
			t.Errorf("configIndexDefaults[%s] = %q, want %q", name, got, val)
		}
		assertConfigDispatchable(t, name)
	}
}

func assertConfigDispatchable(t *testing.T, name string) {
	t.Helper()
	if _, ok := configurations[name]; !ok {
		t.Errorf("%s has no handler in `configurations`", name)
	}
	for _, n := range configOrder {
		if n == name {
			return
		}
	}
	t.Errorf("%s is absent from configOrder — env.go:53 iterates the order and would never reach it", name)
}
