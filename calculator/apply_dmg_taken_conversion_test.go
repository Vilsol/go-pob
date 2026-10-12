package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestApplyDmgTakenConversion pins calcs.applyDmgTakenConversion, CalcDefence.lua:83-142.
func TestApplyDmgTakenConversion(t *testing.T) {
	const eps = 1e-9

	out := func(kv map[string]float64) map[string]float64 {
		base := map[string]float64{"DamageReductionMax": 90}
		for k, v := range kv {
			base[k] = v
		}
		return base
	}

	t.Run("source keeps the unshifted remainder and elemental INC hits only the destination", func(t *testing.T) {
		list := moddb.NewModList()
		list.AddMod(mod.NewFloat("PhysicalDamageTakenAsFire", mod.TypeBase, 30))
		list.AddMod(mod.NewFloat("ElementalDamageTaken", mod.TypeIncrease, 50))

		_, total := ApplyDmgTakenConversion(list, out(map[string]float64{
			"BasePhysicalDamageReductionWhenHit": 10,
			"FireResist":                         40,
		}), nil, data.DamageTypePhysical, 1000)

		if math.Abs(total-900) > eps {
			t.Errorf("total = %v, want 900 (Physical 630 + Fire 270)", total)
		}
	})

	t.Run("an over-100%% shift clamps the source share at zero", func(t *testing.T) {
		list := moddb.NewModList()
		list.AddMod(mod.NewFloat("PhysicalDamageTakenAsFire", mod.TypeBase, 130))

		_, total := ApplyDmgTakenConversion(list, out(nil), nil, data.DamageTypePhysical, 1000)

		if math.Abs(total-1300) > eps {
			t.Errorf("total = %v, want 1300 (Fire only; the Physical share is clamped to 0)", total)
		}
	})

	t.Run("the elemental shift alias keys on the SOURCE type", func(t *testing.T) {
		list := moddb.NewModList()
		list.AddMod(mod.NewFloat("ElementalDamageTakenAsChaos", mod.TypeBase, 25))
		list.AddMod(mod.NewFloat("ArmourAppliesToChaosDamageTaken", mod.TypeBase, 100))

		_, total := ApplyDmgTakenConversion(list, out(map[string]float64{
			"Armour": 10000,
		}), nil, data.DamageTypeFire, 1000)

		want := 750 + 250*(1-89.0/100)
		if math.Abs(total-want) > eps {
			t.Errorf("total = %v, want %v", total, want)
		}
	})

	t.Run("a non-elemental source does NOT read the elemental shift alias", func(t *testing.T) {
		list := moddb.NewModList()
		list.AddMod(mod.NewFloat("ElementalDamageTakenAsChaos", mod.TypeBase, 25))

		_, total := ApplyDmgTakenConversion(list, out(nil), nil, data.DamageTypePhysical, 1000)

		if math.Abs(total-1000) > eps {
			t.Errorf("total = %v, want 1000 (Physical is not elemental; the alias must not apply)", total)
		}
	})

	t.Run("DamageReductionMax caps armour and flat reduction together", func(t *testing.T) {
		list := moddb.NewModList()
		list.AddMod(mod.NewFloat("ArmourAppliesToPhysicalDamageTaken", mod.TypeBase, 100))

		_, total := ApplyDmgTakenConversion(list, out(map[string]float64{
			"Armour":                             1000000,
			"BasePhysicalDamageReductionWhenHit": 30,
		}), nil, data.DamageTypePhysical, 1000)

		want := 1000 * (1 - 90.0/100)
		if math.Abs(total-want) > eps {
			t.Errorf("total = %v, want %v (capped at DamageReductionMax, not 100-98-30)", total, want)
		}
	})

	t.Run("SelfIgnore flags zero their term rather than falling through", func(t *testing.T) {
		list := moddb.NewModList()
		list.AddMod(mod.NewFlag("SelfIgnorePhysicalResistance", true))
		list.AddMod(mod.NewFlag("SelfIgnoreBasePhysicalDamageReduction", true))

		_, total := ApplyDmgTakenConversion(list, out(map[string]float64{
			"PhysicalResist":                     50,
			"BasePhysicalDamageReductionWhenHit": 40,
		}), nil, data.DamageTypePhysical, 1000)

		if math.Abs(total-1000) > eps {
			t.Errorf("total = %v, want 1000 (both terms ignored)", total)
		}
	})

	t.Run("the reduction read takes the WhenHit member, which the pin always writes", func(t *testing.T) {
		list := moddb.NewModList()

		_, total := ApplyDmgTakenConversion(list, out(map[string]float64{
			"BasePhysicalDamageReductionWhenHit": 0,
			"BasePhysicalDamageReduction":        60,
		}), nil, data.DamageTypePhysical, 1000)

		if math.Abs(total-1000) > eps {
			t.Errorf("total = %v, want 1000 (WhenHit is 0 and wins)", total)
		}
	})
}
