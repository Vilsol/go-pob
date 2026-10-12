package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestStormBurstBaseCarriesBothArms pins act_int.lua:18036-18048.
func TestStormBurstBaseCarriesBothArms(t *testing.T) {
	fn := skillFuncsFor("StormBurst")["preDamageFunc"]
	if fn == nil {
		t.Fatal("REFUSING: StormBurst has no preDamageFunc registered")
	}

	newSkill := func(part int) (*ActiveSkill, *moddb.ModList) {
		list := moddb.NewModList()
		return &ActiveSkill{
			SkillPart:    part,
			SkillModList: list,
			SkillCfg:     &moddb.ListCfg{},
			SkillData: &SkillData{
				Duration:                2.0,
				RepeatFrequency:         0.15,
				RemainingDurationDamage: 30,
			},
		}, list
	}
	damageMore := func(list *moddb.ModList) (float64, bool) {
		for _, m := range list.Mods() {
			if m.Name() == "Damage" && m.Type() == mod.TypeMore {
				return m.Value().Float(), true
			}
		}
		return 0, false
	}

	t.Run("part 2 multiplies into an existing dpsMultiplier", func(t *testing.T) {
		skill, _ := newSkill(2)
		skill.SkillData.DpsMultiplier = 3
		fn(skill, map[string]float64{"DurationMod": 1}, nil)

		if skill.SkillData.DpsMultiplier != 42 {
			t.Fatalf("dpsMultiplier = %v, want 42 — :18042 is `(dpsMultiplier or 1) * (1 + floor(...))`; "+
				"assigning discards a multiplier the skill arrived with", skill.SkillData.DpsMultiplier)
		}
	})

	t.Run("part 2 treats an unset dpsMultiplier as 1", func(t *testing.T) {
		skill, _ := newSkill(2)
		fn(skill, map[string]float64{"DurationMod": 1}, nil)
		if skill.SkillData.DpsMultiplier != 14 {
			t.Fatalf("dpsMultiplier = %v, want 14 — the `or 1` arm", skill.SkillData.DpsMultiplier)
		}
	})

	t.Run("part 3 adds the remaining-zap Damage MORE", func(t *testing.T) {
		skill, list := newSkill(3)
		fn(skill, map[string]float64{"DurationMod": 1}, nil)

		got, ok := damageMore(list)
		if !ok {
			t.Fatal("part 3 added no Damage MORE — :18046 is the arm this tree registered only on " +
				"StormBurstAltX, where the same body sits at part 2 because the variant has two parts")
		}
		if got != 290 {
			t.Fatalf("Damage MORE = %v, want 290 — max(remainingDurationDamage*jumps - 100, 0)", got)
		}
	})

	t.Run("control: the arms are exclusive", func(t *testing.T) {
		skill, list := newSkill(3)
		fn(skill, map[string]float64{"DurationMod": 1}, nil)
		if skill.SkillData.DpsMultiplier != 0 {
			t.Fatalf("part 3 wrote dpsMultiplier = %v, want untouched", skill.SkillData.DpsMultiplier)
		}
		skill2, list2 := newSkill(2)
		fn(skill2, map[string]float64{"DurationMod": 1}, nil)
		if _, ok := damageMore(list2); ok {
			t.Fatal("part 2 added a Damage MORE; that arm is part 3's")
		}
		if _, ok := damageMore(list); !ok {
			t.Fatal("control is inert: part 3 added no mod, so the part-2 assertion proves nothing")
		}
	})

	t.Run("control: part 1 writes nothing", func(t *testing.T) {
		skill, list := newSkill(1)
		fn(skill, map[string]float64{"DurationMod": 1}, nil)
		if _, ok := damageMore(list); ok || skill.SkillData.DpsMultiplier != 0 {
			t.Fatalf("part 1 was touched: dpsMultiplier=%v", skill.SkillData.DpsMultiplier)
		}
	})
}
