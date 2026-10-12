package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestUncommentedConfigHandlersEmitTheirMods covers the seven ConfigOptions rows that were either commented out or absent while their apply functions were fully written.
func TestUncommentedConfigHandlersEmitTheirMods(t *testing.T) {
	inOrder := map[string]bool{}
	for _, n := range configOrder {
		inOrder[n] = true
	}

	for _, tc := range []struct {
		option string
		val    interface{}
		want   []string
	}{
		{"conditionTotemsNotSummonedInPastTwoSeconds", true, []string{"Condition:NoSummonedTotemsInPastTwoSeconds"}},
		{"conditionTotemsKilledRecently", true, []string{"Condition:TotemsKilledRecently"}},
		{"conditionTotemsHitRecently", true, []string{"Condition:TotemsHitRecently"}},
		{"conditionTotemsHitSpellRecently", true, []string{"Condition:TotemsHitSpellRecently", "Condition:TotemsHitRecently"}},
		{"conditionUsedBrandRecently", true, []string{"Condition:UsedBrandRecently"}},
		{"multiplierTotemsKilledRecently", 3.0, []string{"Multiplier:EnemyKilledByTotemsRecently", "Condition:TotemsKilledRecently"}},
		{"multiplierShockedEnemyKilledRecently", 4.0, []string{"Multiplier:ShockedEnemyKilledRecently"}},
		{"multiplierShockedNonShockedEnemyRecently", 5.0, []string{"Multiplier:ShockedNonShockedEnemyRecently"}},
	} {
		if !inOrder[tc.option] {
			t.Errorf("%s is not in configOrder, so env.go never dispatches it however good the "+
				"handler is", tc.option)
			continue
		}
		handler, ok := configurations[tc.option]
		if !ok {
			t.Errorf("%s is in configOrder with no handler in `configurations`, so it dispatches "+
				"to nothing", tc.option)
			continue
		}

		modList := moddb.NewModList()
		enemyModList := moddb.NewModList()
		handler(tc.val, modList, enemyModList)
		for _, name := range tc.want {
			if !modListHasNamed(modList, name) {
				t.Errorf("%s emitted no %s", tc.option, name)
			}
		}

		off := moddb.NewModList()
		handler(nil, off, moddb.NewModList())
		for _, name := range tc.want {
			if modListHasNamed(off, name) {
				t.Errorf("%s emitted %s for an UNSET option", tc.option, name)
			}
		}
	}
}

func modListHasNamed(list *moddb.ModList, name string) bool {
	for _, m := range list.Mods() {
		if m.Name() == name {
			return true
		}
	}
	return false
}
