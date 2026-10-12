package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestLancingSteelBaseMatchesThePin pins act_dex.lua:10231-10238, the BASE Lancing Steel's preDamageFunc.
func TestLancingSteelBaseMatchesThePin(t *testing.T) {
	for _, tc := range []struct {
		id     string
		source string
	}{
		{"LancingSteel", "Skill:LancingSteel"},
		{"LancingSteelAltX", "Skill:LancingSteelAltX"},
	} {
		t.Run(tc.id, func(t *testing.T) {
			fn := skillFuncsFor(tc.id)["preDamageFunc"]
			if fn == nil {
				t.Fatalf("REFUSING: %s has no preDamageFunc registered", tc.id)
			}

			list := moddb.NewModList()
			list.AddMod(mod.NewFloat("LancingSteelSubsequentDamage", mod.TypeMore, 25))

			skill := &ActiveSkill{
				SkillPart:    2,
				SkillModList: list,
				SkillCfg:     &moddb.ListCfg{},
				SkillData:    &SkillData{},
			}
			fn(skill, map[string]float64{"ProjectileCount": 5}, nil)

			if skill.SkillData.DpsMultiplier != 5 {
				t.Fatalf("dpsMultiplier = %v, want 5 — :10238 assigns output.ProjectileCount outright; "+
					"a `1 + 0.4*(n-1)` shape is a constant upstream does not have",
					skill.SkillData.DpsMultiplier)
			}

			want := 20.0
			var got float64
			var found bool
			for _, m := range list.Mods() {
				if m.Name() == "Damage" && string(m.GetSource()) == tc.source && m.Type() == mod.TypeMore {
					got, found = m.Value().Float(), true
				}
			}
			if !found {
				t.Fatalf("no Damage MORE from %q — the pin's NewMod at the end of the callback is the "+
					"half a dpsMultiplier of the right magnitude would hide", tc.source)
			}
			if math.Abs(got-want) > 1e-9 {
				t.Fatalf("Damage MORE = %v, want %v — (More-1)*100*(n-1)/n", got, want)
			}
		})
	}

	t.Run("control: part 1 writes nothing", func(t *testing.T) {
		list := moddb.NewModList()
		skill := &ActiveSkill{
			SkillPart: 1, SkillModList: list, SkillCfg: &moddb.ListCfg{}, SkillData: &SkillData{},
		}
		skillFuncsFor("LancingSteel")["preDamageFunc"](skill, map[string]float64{"ProjectileCount": 5}, nil)
		damage := 0
		for _, m := range list.Mods() {
			if m.Name() == "Damage" {
				damage++
			}
		}
		if skill.SkillData.DpsMultiplier != 0 || damage != 0 {
			t.Fatalf("part 1 was touched: dpsMultiplier=%v damageMods=%d",
				skill.SkillData.DpsMultiplier, damage)
		}
	})
}
