package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestInitModDBDebuffGlobalEffects: CalcSetup.lua:68-71 tag Debilitated and Malignant Madness as Debuff global effects.
func TestInitModDBDebuffGlobalEffects(t *testing.T) {
	db := moddb.NewModDB()
	initModDBMods(db)
	want := map[string]string{"Debilitated": "Debilitated", "MalignantMadness": "Malignant Madness"}
	seen := 0
	for _, name := range []string{"Damage", "MovementSpeed", "ActionSpeed"} {
		for _, m := range db.Mods[name] {
			var cond string
			var effect *mod.GlobalEffectTag
			for _, tag := range m.Tags() {
				switch tg := tag.(type) {
				case *mod.ConditionTag:
					cond = tg.VarList[0]
				case *mod.GlobalEffectTag:
					effect = tg
				}
			}
			effectName, ok := want[cond]
			if !ok {
				continue
			}
			seen++
			if effect == nil || len(effect.GlobalEffectList) == 0 || effect.GlobalEffectList[0] != "Debuff" || effect.NameTag != effectName {
				t.Errorf("%s %s: global effect %+v, PoB Debuff %q", name, cond, effect, effectName)
			}
		}
	}
	if seen != 4 {
		t.Fatalf("%d Debilitated / Malignant Madness mods, want 4", seen)
	}
}
