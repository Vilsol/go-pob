package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestTotemAuraModsScaleLikePoB: the totem copy of an aura keeps only defence and resist mods, truncates a scaled
// integer, scales a fraction exactly and ignores the unscalable flag. The wants are the pinned Lua loop
// (CalcPerform.lua:2564-2579) run under luajit on the same mods at scale 1.37.
//
//pob:block:src/Modules/CalcPerform.lua:2564-2579
func TestTotemAuraModsScaleLikePoB(t *testing.T) {
	extra := moddb.NewModList()
	extra.AddMod(mod.NewFloat("Armour", mod.TypeBase, 1000))
	buffList := moddb.NewModList()
	buffList.AddMod(mod.NewFloat("EnergyShield", mod.TypeBase, 12.5))
	buffList.AddMod(mod.NewFloat("Life", mod.TypeBase, 50))
	buffList.AddMod(mod.NewFloat("FireResistMax", mod.TypeBase, -3))
	unscalable := moddb.NewModList()
	unscalable.AddMod(mod.NewFloat("ChaosResist", mod.TypeBase, 23).Tag(mod.GlobalEffect("Aura").Unscalable(true)))

	got := map[string]float64{}
	for _, m := range totemAuraModList([]*moddb.ModList{extra, buffList, unscalable}, 1.37).Mods() {
		got[m.Name()] = m.Value().Float()
	}
	want := map[string]float64{"TotemArmour": 1370, "TotemEnergyShield": 17.125, "TotemFireResistMax": -4, "TotemChaosResist": 31}
	if len(got) != len(want) {
		t.Errorf("mods %v, PoB %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, PoB %v", k, got[k], v)
		}
	}
}
