package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestPreviouslyUnhandledConfigsDispatch covers twelve ConfigOptions.lua options that corpus XMLs set.
func TestPreviouslyUnhandledConfigsDispatch(t *testing.T) {
	for _, tc := range []struct {
		name  string
		val   interface{}
		enemy bool
		flag  string
		sum   string
		ovr   string
		want  float64
	}{
		{name: "Unbound", val: true, flag: "Condition:Unbound"},
		{name: "freshMeatBuffs", val: true, flag: "Condition:FreshMeatActive"},
		{name: "Disgorged", val: true, flag: "Condition:Disgorged"},
		{name: "MomentumStacks", val: 7.0, sum: "Multiplier:MomentumStacks", want: 7},
		{name: "overrideSpiritCharges", val: 0.0, ovr: "SpiritCharges", want: 0},
		{name: "conditionOnBrineGround", val: true, flag: "Condition:OnBrineGround"},
		{name: "conditionStunnedRecently", val: true, flag: "Condition:StunnedRecently"},
		{name: "conditionEnergyShieldRechargePastTwoSec", val: true, flag: "Condition:EnergyShieldRechargePastTwoSec"},
		{name: "DetonatedMinesRecently", val: true, flag: "Condition:DetonatedMinesRecently"},
		{name: "conditionNonPoisonedOnly", val: true, flag: "Condition:NonPoisonedOnly"},
		{name: "overrideIgniteStackPotential", val: 3.0, ovr: "IgniteStackPotentialOverride", want: 3},
		{name: "enemyMaxResist", val: true, enemy: true, flag: "DoNotChangeMaxResFromConfig"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !slices.Contains(configOrder, tc.name) {
				t.Fatal("missing from configOrder: the handler would never run")
			}
			apply, ok := configurations[tc.name]
			if !ok {
				t.Fatal("no handler in configurations")
			}
			player, enemy := moddb.NewModList(), moddb.NewModList()
			apply(tc.val, player, enemy)
			list := player
			if tc.enemy {
				list = enemy
			}
			var found bool
			for _, m := range list.Mods() {
				switch {
				case tc.flag != "" && m.Name() == tc.flag && m.Type() == mod.TypeFlag:
					found = true
				case tc.sum != "" && m.Name() == tc.sum && m.Type() == mod.TypeBase && m.Value().ValueFloat == tc.want:
					found = true
				case tc.ovr != "" && m.Name() == tc.ovr && m.Type() == mod.TypeOverride && m.Value().ValueFloat == tc.want:
					found = true
				}
			}
			if !found {
				t.Errorf("%s did not add the pin's mod; got %v", tc.name, list.Mods())
			}
		})
	}
}
