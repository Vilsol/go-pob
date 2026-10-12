package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

var nestedModPayloads = []struct {
	name  string
	build func(inner mod.Mod) any
	inner func(payload any) mod.Mod
}{
	{
		"EnemyModifier",
		func(m mod.Mod) any { return mod.EnemyModifier{Mod: m} },
		func(p any) mod.Mod { return p.(mod.EnemyModifier).Mod },
	},
	{
		"ExtraAura",
		func(m mod.Mod) any { return mod.ExtraAura{Mod: m} },
		func(p any) mod.Mod { return p.(mod.ExtraAura).Mod },
	},
	{
		"ExtraAuraEffect",
		func(m mod.Mod) any { return mod.ExtraAuraEffect{Mod: m} },
		func(p any) mod.Mod { return p.(mod.ExtraAuraEffect).Mod },
	},
	{
		"AffectedByAuraMod",
		func(m mod.Mod) any { return mod.AffectedByAuraMod{Mod: m} },
		func(p any) mod.Mod { return p.(mod.AffectedByAuraMod).Mod },
	},
	{
		"AffectedByCurseMod",
		func(m mod.Mod) any { return mod.AffectedByCurseMod{Mod: m} },
		func(p any) mod.Mod { return p.(mod.AffectedByCurseMod).Mod },
	},
	{
		"ExtraSkillMod",
		func(m mod.Mod) any { return mod.ExtraSkillMod{Mod: m} },
		func(p any) mod.Mod { return p.(mod.ExtraSkillMod).Mod },
	},
	{
		"MinionModifier",
		func(m mod.Mod) any { return mod.MinionModifier{Mod: m} },
		func(p any) mod.Mod { return p.(mod.MinionModifier).Mod },
	},
	{
		"GrantReservedPoolAsAura",
		func(m mod.Mod) any { return mod.GrantReservedPoolAsAura{Mod: m} },
		func(p any) mod.Mod { return p.(mod.GrantReservedPoolAsAura).Mod },
	},
}

// TestScaleAddListScalesEveryNestedModPayload pins ModStore.lua:66-67, the SECOND arm of ScaleAddList — the one that reaches INSIDE a table payload.
func TestScaleAddListScalesEveryNestedModPayload(t *testing.T) {
	for _, p := range nestedModPayloads {
		t.Run(p.name, func(t *testing.T) {
			src := NewModList()
			src.AddMod(mod.NewList(p.name, p.build(mod.NewFloat("DamageTaken", mod.TypeIncrease, 10))))

			dest := NewModList()
			dest.ScaleAddList(src, 1.69)

			got := dest.List(nil, p.name)
			testza.AssertEqual(t, 1, len(got))

			testza.AssertEqual(t, float64(16), p.inner(got[0]).Value().Float())

			testza.AssertEqual(t, float64(10), p.inner(src.List(nil, p.name)[0]).Value().Float())
		})
	}
}

// TestScaleAddListRoundsNonIntegerNestedValueToDefaultPrecision.
func TestScaleAddListRoundsNonIntegerNestedValueToDefaultPrecision(t *testing.T) {
	src := NewModList()
	src.AddMod(mod.NewList("EnemyModifier", mod.EnemyModifier{
		Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, 2.5),
	}))

	dest := NewModList()
	dest.ScaleAddList(src, 1.5)

	got := dest.List(nil, "EnemyModifier")
	testza.AssertEqual(t, 1, len(got))
	testza.AssertEqual(t, 3.7, got[0].(mod.EnemyModifier).Mod.Value().Float())
}
