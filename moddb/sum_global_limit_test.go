package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestSumAppliesGlobalLimit: three Purposeful-Harbinger-shaped mods sharing one key must sum to the limit, not to 3x their value.
func TestSumAppliesGlobalLimit(t *testing.T) {
	db := NewModDB()
	for i := 0; i < 3; i++ {
		db.AddMod(mod.NewFloat("SkillAuraEffectOnSelf", mod.TypeIncrease, 10).
			Tag(mod.Multiplier("Herald").GlobalLimit(25).GlobalLimitKey("PurposefulHarbinger")))
	}
	db.Multipliers["Herald"] = 1

	testza.AssertEqual(t, 25.0, db.Sum(mod.TypeIncrease, nil, "SkillAuraEffectOnSelf"))
}

// TestSumGlobalLimitIsPerKey: two different keys keep independent running totals.
func TestSumGlobalLimitIsPerKey(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 40).
		Tag(mod.Multiplier("PowerCharge").GlobalLimit(25).GlobalLimitKey("VastPower")))
	db.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 40).
		Tag(mod.Multiplier("StationarySeconds").GlobalLimit(30).GlobalLimitKey("ExpansiveMight")))
	db.Multipliers["PowerCharge"] = 1
	db.Multipliers["StationarySeconds"] = 1

	testza.AssertEqual(t, 55.0, db.Sum(mod.TypeIncrease, nil, "AreaOfEffect"))
}

// TestSumGlobalLimitOnPerStatTag.
func TestSumGlobalLimitOnPerStatTag(t *testing.T) {
	db := NewModDB()
	for i := 0; i < 2; i++ {
		db.AddMod(mod.NewFloat("ChaosDamage", mod.TypeIncrease, 1).
			Tag(mod.PerStat(10, "Mana").GlobalLimit(75).GlobalLimitKey("DarkIdeation")))
	}
	db.Actor = fixedOutputActor{"Mana": 500}

	testza.AssertEqual(t, 75.0, db.Sum(mod.TypeIncrease, nil, "ChaosDamage"))
}

// TestSumGlobalLimitFrameIsPerStore.
func TestSumGlobalLimitFrameIsPerStore(t *testing.T) {
	parent := NewModDB()
	parent.AddMod(mod.NewFloat("SkillAuraEffectOnSelf", mod.TypeIncrease, 40).
		Tag(mod.Multiplier("Herald").GlobalLimit(25).GlobalLimitKey("PurposefulHarbinger")))

	child := NewModDB()
	child.Parent = parent
	child.AddMod(mod.NewFloat("SkillAuraEffectOnSelf", mod.TypeIncrease, 40).
		Tag(mod.Multiplier("Herald").GlobalLimit(25).GlobalLimitKey("PurposefulHarbinger")))
	child.Multipliers["Herald"] = 1

	testza.AssertEqual(t, 50.0, child.Sum(mod.TypeIncrease, nil, "SkillAuraEffectOnSelf"))
}

// TestSumModListHasNoGlobalLimit pins the asymmetry: ModList.lua:73-89 has no globalLimit block, so the clamp must NOT be copied there.
func TestSumModListHasNoGlobalLimit(t *testing.T) {
	list := NewModList()
	for i := 0; i < 3; i++ {
		list.AddMod(mod.NewFloat("SkillAuraEffectOnSelf", mod.TypeIncrease, 10).
			Tag(mod.Multiplier("Herald").GlobalLimit(25).GlobalLimitKey("PurposefulHarbinger")))
	}
	list.Multipliers["Herald"] = 1

	testza.AssertEqual(t, 30.0, list.Sum(mod.TypeIncrease, nil, "SkillAuraEffectOnSelf"))
}

// TestSumEvaluatesParentInChildContext: ModDB.lua:118 passes `context`, so the parent's condition-gated mod resolves against the child's conditions.
func TestSumEvaluatesParentInChildContext(t *testing.T) {
	parent := NewModDB()
	parent.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 30).Tag(mod.Condition("Fortified")))

	child := NewModDB()
	child.Parent = parent
	child.Conditions["Fortified"] = true

	testza.AssertEqual(t, 30.0, child.Sum(mod.TypeIncrease, nil, "Damage"))
}

// TestSumEvaluatesParentMultiplierInChildContext: the same threading applies to scaling tags — the parent's mod must scale by the CHILD's multiplier.
func TestSumEvaluatesParentMultiplierInChildContext(t *testing.T) {
	parent := NewModDB()
	parent.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 3).Tag(mod.Multiplier("Frenzy")))

	child := NewModDB()
	child.Parent = parent
	child.Multipliers["Frenzy"] = 4

	testza.AssertEqual(t, 12.0, child.Sum(mod.TypeIncrease, nil, "Damage"))
}

// TestMoreEvaluatesParentInChildContext pins ModDB.lua:141.
func TestMoreEvaluatesParentInChildContext(t *testing.T) {
	parent := NewModDB()
	parent.AddMod(mod.NewFloat("Damage", mod.TypeMore, 50).Tag(mod.Condition("Fortified")))

	child := NewModDB()
	child.Parent = parent
	child.Conditions["Fortified"] = true

	testza.AssertEqual(t, 1.5, child.More(nil, "Damage"))
}

// TestFlagEvaluatesParentInChildContext pins ModDB.lua:165.
func TestFlagEvaluatesParentInChildContext(t *testing.T) {
	parent := NewModDB()
	parent.AddMod(mod.NewFlag("CannotBeStunned", true).Tag(mod.Condition("Fortified")))

	child := NewModDB()
	child.Parent = parent

	testza.AssertEqual(t, false, child.Flag(nil, "CannotBeStunned"))

	child.Conditions["Fortified"] = true
	testza.AssertEqual(t, true, child.Flag(nil, "CannotBeStunned"))
}

// TestOverrideEvaluatesParentInChildContext pins ModDB.lua:189.
func TestOverrideEvaluatesParentInChildContext(t *testing.T) {
	parent := NewModDB()
	parent.AddMod(mod.NewFloat("Speed", mod.TypeOverride, 7).Tag(mod.Condition("Fortified")))

	child := NewModDB()
	child.Parent = parent
	child.Conditions["Fortified"] = true

	value := child.Override(nil, "Speed")
	if value == nil {
		t.Fatalf("Override returned nil; parent mod was evaluated outside the child context")
	}
	testza.AssertEqual(t, 7.0, value.Float())
}

// TestListEvaluatesParentInChildContext pins ModDB.lua:214.
func TestListEvaluatesParentInChildContext(t *testing.T) {
	parent := NewModDB()
	parent.AddMod(mod.NewList("ExtraSkill", "Vaal Haste").Tag(mod.Condition("Fortified")))

	child := NewModDB()
	child.Parent = parent

	testza.AssertEqual(t, 0, len(child.List(nil, "ExtraSkill")))

	child.Conditions["Fortified"] = true
	res := child.List(nil, "ExtraSkill")
	testza.AssertEqual(t, 1, len(res))
	testza.AssertEqual(t, "Vaal Haste", res[0])
}

// TestSumModListEvaluatesParentInChildContext: ModList.lua:86 threads `context` too.
func TestSumModListEvaluatesParentInChildContext(t *testing.T) {
	parent := NewModList()
	parent.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 30).Tag(mod.Condition("Fortified")))

	child := NewModList()
	child.Parent = parent
	child.Conditions["Fortified"] = true

	testza.AssertEqual(t, 30.0, child.Sum(mod.TypeIncrease, nil, "Damage"))
}

type fixedOutputActor map[string]float64

func (a fixedOutputActor) GetOutput(stat string) (float64, bool) {
	v, ok := a[stat]
	return v, ok
}
