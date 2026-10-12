package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestMapAffixCheckRowsApply pins ConfigOptions.lua:833-841 through mapAffixDropDownFunction (ConfigOptions.lua:117-130) onto the check rows of Data/ModMap.lua.
func TestMapAffixCheckRowsApply(t *testing.T) {
	for _, name := range []string{"MapPrefix1", "MapPrefix2", "MapPrefix3", "MapPrefix4", "MapSuffix1", "MapSuffix2", "MapSuffix3", "MapSuffix4"} {
		if !slices.Contains(configOrder, name) {
			t.Fatalf("%s missing from configOrder: the handler would never run", name)
		}
		if _, ok := configurations[name]; !ok {
			t.Fatalf("%s has no handler in configurations", name)
		}
	}
	apply := configurations["MapSuffix2"]

	self, enemy := moddb.NewModList(), moddb.NewModList()
	apply("of Congealment", self, enemy)
	for _, flag := range []string{"CannotLeechLifeFromSelf", "CannotLeechManaFromSelf", "CannotLeechEnergyShieldFromSelf"} {
		if !enemy.Flag(nil, flag) {
			t.Errorf("of Congealment must set %s on the enemy", flag)
		}
	}

	self, enemy = moddb.NewModList(), moddb.NewModList()
	apply("of Stasis", self, enemy)
	if !self.Flag(nil, "NoLifeRegen") || !self.Flag(nil, "NoManaRegen") || !self.Flag(nil, "NoEnergyShieldRegen") {
		t.Error("of Stasis must set NoLifeRegen, NoManaRegen and NoEnergyShieldRegen on the player")
	}

	self, enemy = moddb.NewModList(), moddb.NewModList()
	apply("NONE", self, enemy)
	if self.Flag(nil, "NoLifeRegen") || enemy.Flag(nil, "CannotLeechLifeFromSelf") {
		t.Error("NONE must apply nothing")
	}
}
