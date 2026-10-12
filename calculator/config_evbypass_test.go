package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestEVBypassConfigDispatches pins ConfigOptions.lua:245-247.
func TestEVBypassConfigDispatches(t *testing.T) {
	if !slices.Contains(configOrder, "EVBypass") {
		t.Fatal("EVBypass missing from configOrder: the handler would never run")
	}
	apply, ok := configurations["EVBypass"]
	if !ok {
		t.Fatal("EVBypass has no handler in configurations")
	}

	on := moddb.NewModList()
	apply(true, on, moddb.NewModList())
	if !on.Flag(nil, "Condition:EVBypass") {
		t.Error("checked EVBypass must add Condition:EVBypass")
	}

	off := moddb.NewModList()
	apply(false, off, moddb.NewModList())
	if off.Flag(nil, "Condition:EVBypass") {
		t.Error("unchecked EVBypass must add nothing")
	}
}

// TestCommentedOutConfigHandlersDispatch covers seven more names that sat in configOrder with their handlers commented out.
func TestCommentedOutConfigHandlersDispatch(t *testing.T) {
	for _, tc := range []struct {
		name  string
		val   interface{}
		enemy bool
		flag  string
		mult  string
		want  float64
	}{
		{name: "conditionPoisoned", val: true, flag: "Condition:Poisoned"},
		{name: "multiplierSkillUsedRecently", val: 3.0, flag: "Condition:UsedSkillRecently", mult: "Multiplier:SkillUsedRecently", want: 3},
		{name: "conditionUsedTravelSkillRecently", val: true, flag: "Condition:UsedMovementSkillRecently"},
		{name: "conditionUsedVaalSkillRecently", val: true, flag: "Condition:UsedVaalSkillRecently"},
		{name: "conditionHaveManaStorm", val: true, flag: "Condition:SacrificeManaForLightning"},
		{name: "multiplierRuptureStacks", val: 2.0, enemy: true, mult: "Multiplier:RuptureStack", want: 2},
		{name: "conditionNearLinkedTarget", val: true, enemy: true, flag: "Condition:NearLinkedTarget"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !slices.Contains(configOrder, tc.name) {
				t.Fatal("missing from configOrder: the handler would never run")
			}
			apply, ok := configurations[tc.name]
			if !ok {
				t.Fatal("no handler in configurations")
			}
			self, enemy := moddb.NewModList(), moddb.NewModList()
			apply(tc.val, self, enemy)
			target := self
			if tc.enemy {
				target = enemy
			}
			if len(self.Mods())+len(enemy.Mods()) == 0 {
				t.Fatal("handler emitted nothing")
			}
			found := map[string]float64{}
			for _, m := range target.Mods() {
				found[m.Name()] = m.Value().Float()
			}
			if tc.flag != "" {
				if _, ok := found[tc.flag]; !ok {
					t.Errorf("want %s on the %s list, got %v", tc.flag, map[bool]string{true: "enemy", false: "player"}[tc.enemy], found)
				}
			}
			if tc.mult != "" && found[tc.mult] != tc.want {
				t.Errorf("want %s = %v, got %v", tc.mult, tc.want, found[tc.mult])
			}
		})
	}
}
