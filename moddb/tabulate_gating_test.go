package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestTabulateAppliesConditionGateModDB: a tagged float mod whose condition is false must not appear in Tabulate's result at all.
func TestTabulateAppliesConditionGateModDB(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).Tag(mod.Condition("Fortified")))
	db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 7))

	res := db.Tabulate(mod.TypeIncrease, nil, "Damage")
	testza.AssertEqual(t, 1, len(res))
	testza.AssertEqual(t, 7.0, res[0].Value)

	db.Conditions["Fortified"] = true
	res = db.Tabulate(mod.TypeIncrease, nil, "Damage")
	testza.AssertEqual(t, 2, len(res))
	testza.AssertEqual(t, 40.0, res[0].Value)
	testza.AssertEqual(t, 7.0, res[1].Value)
}

// TestTabulateAppliesConditionGateModList: same predicate, ModList side.
func TestTabulateAppliesConditionGateModList(t *testing.T) {
	list := NewModList()
	list.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).Tag(mod.Condition("Fortified")))
	list.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 7))

	res := list.Tabulate(mod.TypeIncrease, nil, "Damage")
	testza.AssertEqual(t, 1, len(res))
	testza.AssertEqual(t, 7.0, res[0].Value)

	SetCondition(list.ModStore, "Fortified", true)
	res = list.Tabulate(mod.TypeIncrease, nil, "Damage")
	testza.AssertEqual(t, 2, len(res))
}

// TestTabulateScalesTaggedValue: a Multiplier tag scales the value, so Tabulate must report the EVALUATED number, not the stored one.
func TestTabulateScalesTaggedValue(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 3).Tag(mod.Multiplier("Frenzy")))
	db.Multipliers["Frenzy"] = 4

	res := db.Tabulate(mod.TypeIncrease, nil, "Damage")
	testza.AssertEqual(t, 1, len(res))
	testza.AssertEqual(t, 12.0, res[0].Value)
}

// TestTabulateAppliesGlobalLimit mirrors ModDB.lua:230-236.
func TestTabulateAppliesGlobalLimit(t *testing.T) {
	db := NewModDB()
	for i := 0; i < 3; i++ {
		db.AddMod(mod.NewFloat("SkillAuraEffectOnSelf", mod.TypeIncrease, 10).
			Tag(mod.Multiplier("Herald").GlobalLimit(25).GlobalLimitKey("PurposefulHarbinger")))
	}
	db.Multipliers["Herald"] = 1

	res := db.Tabulate(mod.TypeIncrease, nil, "SkillAuraEffectOnSelf")
	testza.AssertEqual(t, 3, len(res))
	testza.AssertEqual(t, 10.0, res[0].Value)
	testza.AssertEqual(t, 10.0, res[1].Value)
	testza.AssertEqual(t, 5.0, res[2].Value)
}

// TestTabulateModListFiltersByName pins ModList.lua:187's `mod.name == modName` guard — Tabulate must not report mods it was not asked about.
func TestTabulateModListFiltersByName(t *testing.T) {
	list := NewModList()
	list.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 10))
	list.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 20))

	res := list.Tabulate(mod.TypeIncrease, nil, "Damage")
	testza.AssertEqual(t, 1, len(res))
	testza.AssertEqual(t, "Damage", res[0].Mod.Name())
}

// TestTabulateEvaluatesParentInChildContext pins that PoB threads `context`.
func TestTabulateEvaluatesParentInChildContext(t *testing.T) {
	parent := NewModDB()
	parent.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 30).Tag(mod.Condition("Fortified")))

	child := NewModDB()
	child.Parent = parent
	child.Conditions["Fortified"] = true

	res := child.Tabulate(mod.TypeIncrease, nil, "Damage")
	testza.AssertEqual(t, 1, len(res))
	testza.AssertEqual(t, 30.0, res[0].Value)
}
