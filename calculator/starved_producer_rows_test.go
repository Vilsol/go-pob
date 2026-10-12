package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestStarvedProducerRowsParse: Six rows whose CONSUMERS were already correct in this tree and whose producers did not exist.
func TestStarvedProducerRowsParse(t *testing.T) {
	t.Run("ModParser.lua:4223 exposure at maximum rage", func(t *testing.T) {
		mods, extra := parseMod("Nearby Enemies have Fire Exposure while at maximum Rage", 1)
		testza.AssertEqual(t, "", extra)
		testza.AssertEqual(t, 1, len(mods))
		if len(mods) != 1 {
			return
		}
		testza.AssertEqual(t, "EnemyModifier", mods[0].Name())
		testza.AssertEqual(t, 2, len(mods[0].Tags()), "Effective AND HaveMaximumRage")
		inner, ok := mods[0].Value().List().(mod.EnemyModifier)
		testza.AssertTrue(t, ok)
		if ok {
			testza.AssertEqual(t, "FireExposure", inner.Mod.Name())
			testza.AssertEqual(t, float64(-10), inner.Mod.Value().Float(),
				"exposure is NEGATIVE resistance; a positive 10 would be a buff")
		}
	})

	t.Run("ModParser.lua:5471 rage grants spell damage", func(t *testing.T) {
		mods, extra := parseMod("Rage grants Spell Damage instead of Attack Damage", 1)
		testza.AssertEqual(t, "", extra)
		testza.AssertEqual(t, 1, len(mods))
		if len(mods) == 1 {
			testza.AssertEqual(t, "Condition:RageSpellDamage", mods[0].Name(),
				"calc_perform.go:7497 reads this exact literal, which is CalcPerform.lua:891's")
			testza.AssertEqual(t, mod.TypeFlag, mods[0].Type())
		}
	})

	t.Run("ModParser.lua:5273 spells cost life", func(t *testing.T) {
		mods, extra := parseMod("Spells Cost +8% of Life", 1)
		testza.AssertEqual(t, "", extra)
		testza.AssertEqual(t, 1, len(mods))
		if len(mods) != 1 {
			return
		}
		testza.AssertEqual(t, "LifeCostBase", mods[0].Name())
		testza.AssertEqual(t, float64(1), mods[0].Value().Float(),
			"the VALUE is 1; the 8 lives on the PercentStat tag")
		var sawFloor, sawSpell bool
		for _, tag := range mods[0].Tags() {
			switch tg := tag.(type) {
			case *mod.PercentStatTag:
				testza.AssertEqual(t, "Life", tg.Stat)
				testza.AssertEqual(t, float64(8), tg.Percent)
				sawFloor = tg.TagFloor
			case *mod.SkillTypeTag:
				sawSpell = len(tg.SkillTypeList) == 1 && tg.SkillTypeList[0] == "Spell"
			}
		}
		testza.AssertTrue(t, sawFloor, "ModStore.lua:506-508 FLOORS the percent-of-stat product")
		testza.AssertTrue(t, sawSpell, "without SkillType(Spell) attacks would cost life too")
	})

	t.Run("ModParser.lua:3659 same-ailment damage", func(t *testing.T) {
		mods, extra := parseMod("50% increased Damage with Damaging Ailments you inflict "+
			"while you are affected by the same Ailment", 1)
		testza.AssertEqual(t, "", extra)
		testza.AssertEqual(t, 3, len(mods))
		if len(mods) != 3 {
			return
		}
		for i, want := range []struct {
			kw         mod.KeywordFlag
			enemyCond  string
			playerCond string
		}{
			{mod.KeywordFlagBleed, "Bleeding", "Bleeding"},
			{mod.KeywordFlagIgnite, "Ignited", "Ignited"},
			{mod.KeywordFlagPoison, "Poisoned", "Poisoned"},
		} {
			m := mods[i]
			testza.AssertEqual(t, "Damage", m.Name())
			testza.AssertEqual(t, mod.TypeIncrease, m.Type())
			testza.AssertEqual(t, float64(50), m.Value().Float())
			testza.AssertEqual(t, want.kw, m.KeywordFlags(), "mod %d keyword flag", i)
			testza.AssertEqual(t, 2, len(m.Tags()),
				"mod %d needs BOTH actors: the enemy carrying the ailment and you carrying it", i)
			var enemy, player bool
			for _, tag := range m.Tags() {
				switch tg := tag.(type) {
				case *mod.ActorConditionTag:
					enemy = tg.Actor != nil && *tg.Actor == "enemy" &&
						len(tg.VariableList) == 1 && tg.VariableList[0] == want.enemyCond
				case *mod.ConditionTag:
					player = len(tg.VarList) == 1 && tg.VarList[0] == want.playerCond
				}
			}
			testza.AssertTrue(t, enemy, "mod %d has no enemy ActorCondition", i)
			testza.AssertTrue(t, player, "mod %d has no player Condition", i)
		}
	})

	t.Run("ModParser.lua:2436 eclipse solaris", func(t *testing.T) {
		mods, extra := parseMod("Adds 2 to 5 Fire Damage to Attacks for every 1% "+
			"your Light Radius is above base value", 1)
		testza.AssertEqual(t, "", extra)
		testza.AssertEqual(t, 2, len(mods))
		if len(mods) != 2 {
			return
		}
		testza.AssertEqual(t, "FireMin", mods[0].Name())
		testza.AssertEqual(t, float64(2), mods[0].Value().Float())
		testza.AssertEqual(t, "FireMax", mods[1].Name())
		testza.AssertEqual(t, float64(5), mods[1].Value().Float())
		for i := range mods {
			testza.AssertEqual(t, mod.MFlagAttack, mods[i].Flags(), "mod %d must be attack-only", i)
			testza.AssertEqual(t, 1, len(mods[i].Tags()))
			for _, tag := range mods[i].Tags() {
				ps, ok := tag.(*mod.PerStatTag)
				testza.AssertTrue(t, ok, "mod %d tag %T is not a PerStat", i, tag)
				if ok {
					testza.AssertEqual(t, []string{"LightRadiusInc"}, ps.StatList)
					testza.AssertEqual(t, float64(1), ps.Divide)
				}
			}
		}
	})

	t.Run("ModParser.lua:2318 count as moving", func(t *testing.T) {
		mods, extra := parseMod("You and Enemies in your Presence count as moving "+
			"while affected by Elemental Ailments", 1)
		testza.AssertEqual(t, "", extra)
		testza.AssertEqual(t, 2, len(mods), "one mod per ACTOR — the enemy's and yours")
		if len(mods) != 2 {
			return
		}
		testza.AssertEqual(t, "EnemyModifier", mods[0].Name())
		testza.AssertEqual(t, "Condition:Moving", mods[1].Name())
		testza.AssertEqual(t, 1, len(mods[1].Tags()))
		for _, tag := range mods[1].Tags() {
			cond, ok := tag.(*mod.ConditionTag)
			testza.AssertTrue(t, ok)
			if ok {
				testza.AssertEqual(t, []string{"Frozen", "Chilled", "Shocked", "Ignited",
					"Scorched", "Brittle", "Sapped"}, cond.VarList)
			}
		}
	})

	t.Run("neighbour ModParser.lua:4226", func(t *testing.T) {
		mods, extra := parseMod("Nearby Enemies have Fire Exposure while you are affected by Herald of Ash", 1)
		testza.AssertEqual(t, "", extra)
		testza.AssertEqual(t, 1, len(mods))
		if len(mods) == 1 {
			testza.AssertEqual(t, "EnemyModifier", mods[0].Name())
			testza.AssertEqual(t, 2, len(mods[0].Tags()))
		}
	})
}
