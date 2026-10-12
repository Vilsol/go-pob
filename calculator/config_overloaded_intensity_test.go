package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestOverloadedIntensityConfigDispatches pins the `OverloadedIntensity` config row (ConfigOptions.lua:506-508), which had no handler at all.
func TestOverloadedIntensityConfigDispatches(t *testing.T) {
	assertConfigDispatchable(t, "OverloadedIntensity")
	assertConfigListMembership(t, "OverloadedIntensity")
	assertModSet(t, "OverloadedIntensity", float64(2), "Multiplier:OverloadedIntensity|BASE")

	for _, tc := range []struct {
		val  interface{}
		want float64
	}{
		{float64(2), 2},
		{float64(3), 3},
		{float64(4), 3},
		{float64(9), 3},
	} {
		h, ok := configurations["OverloadedIntensity"]
		if !ok {
			t.Fatalf("no handler named OverloadedIntensity")
		}
		modList, enemyModList := moddb.NewModList(), moddb.NewModList()
		h(tc.val, modList, enemyModList)

		var got float64
		var seen int
		for _, m := range modList.Mods() {
			if m.Name() == "Multiplier:OverloadedIntensity" && m.Type() == mod.TypeBase {
				got = m.Value().Float()
				seen++
			}
		}
		if seen != 1 {
			t.Fatalf("val=%v: wrote %d Multiplier:OverloadedIntensity BASE mods, want exactly 1", tc.val, seen)
		}
		if got != tc.want {
			t.Errorf("val=%v: Multiplier:OverloadedIntensity = %v, want %v", tc.val, got, tc.want)
		}
	}
}

func assertConfigListMembership(t *testing.T, name string) {
	t.Helper()
	if _, ok := configCountDefaults[name]; ok {
		t.Errorf("%s is in configCountDefaults, but upstream declares no defaultState for it", name)
	}
	if _, ok := configIndexDefaults[name]; ok {
		t.Errorf("%s is in configIndexDefaults, but upstream declares no defaultState for it", name)
	}
	if _, ok := configDefaultPlaceholders[name]; ok {
		t.Errorf("%s is in configDefaultPlaceholders, but upstream declares no defaultPlaceholderState for it", name)
	}
	for _, n := range configDefaults {
		if n == name {
			t.Errorf("%s is in configDefaults, but upstream declares no defaultState for it", name)
		}
	}
	if isConfigPlaceholderVar(name) {
		t.Errorf("%s is in configPlaceholderVars, but upstream declares no defaultPlaceholderState for it", name)
	}
}
