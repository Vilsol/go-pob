package calculator2

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestConversionTableSkillThenGlobal is 30% skill conversion of physical to fire with 75% of all damage converted to
// fire; the expected rows are PoB-PoE2's (CalcOffence.lua 2432-2512 under luajit). Summing the additions in another
// order lands on 82.4999... and rounds to 82% instead.
func TestConversionTableSkillThenGlobal(t *testing.T) {
	skillModList := moddb.NewModList()
	skillModList.AddMod(MOD("SkillPhysicalDamageConvertToFire", mod.TypeBase, 30.0))
	skillModList.AddMod(MOD("DamageConvertToFire", mod.TypeBase, 75.0))
	state := &offenceHitState{}
	offenceConversionTables(state, skillModList, nil)

	want := map[string]struct{ fire, mult float64 }{
		"Physical":  {0.83, 0.17000000000000004},
		"Lightning": {0.75, 0.25},
		"Cold":      {0.75, 0.25},
		"Fire":      {0.75, 0.25},
		"Chaos":     {0.75, 0.25},
	}
	for _, damageType := range dmgTypeList {
		row := state.conversionTable[damageType]
		w := want[damageType]
		if math.Abs(row.to["Fire"]-w.fire) > 1e-12 || math.Abs(row.mult-w.mult) > 1e-12 {
			t.Errorf("%s: to Fire %v mult %v, PoB %v mult %v", damageType, row.to["Fire"], row.mult, w.fire, w.mult)
		}
	}
}
