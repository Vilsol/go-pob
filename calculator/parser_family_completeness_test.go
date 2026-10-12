package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestWitheredFamilyIsComplete: The CLASS-COMPLETENESS shape.
func TestWitheredFamilyIsComplete(t *testing.T) {
	for _, tc := range []struct {
		name      string
		line      string
		enemyCond string
	}{
		{"ModParser.lua:4151", "Melee Weapon Hits Inflict 3 Withered Debuffs for 2 seconds", ""},
		{"ModParser.lua:4157", "Chaos Skills inflict up to 15 Withered Debuffs on Hit for 6 seconds", ""},
		{"ModParser.lua:4154", "25% chance to inflict Withered for 2 seconds on Hit against Cursed Enemies", "Cursed"},
		{"neighbour ModParser.lua:4150", "25% chance to inflict Withered for 2 seconds on Hit", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mods, extra := parseMod(tc.line, 1)
			testza.AssertEqual(t, "", extra, "%q must parse completely", tc.line)
			testza.AssertEqual(t, 1, len(mods), "%q", tc.line)
			if len(mods) != 1 {
				return
			}
			testza.AssertEqual(t, "Condition:CanWither", mods[0].Name(), "%q", tc.line)
			testza.AssertEqual(t, mod.TypeFlag, mods[0].Type(), "%q", tc.line)
			testza.AssertTrue(t, mods[0].Value().Flag(), "%q", tc.line)

			if tc.enemyCond == "" {
				testza.AssertEqual(t, 0, len(mods[0].Tags()),
					"%q must be the BARE flag — a stray tag here is an over-restriction", tc.line)
				return
			}
			testza.AssertEqual(t, 1, len(mods[0].Tags()),
				"%q must carry exactly its enemy condition — without it Wither is unconditional", tc.line)
			for _, tag := range mods[0].Tags() {
				cond, isCond := tag.(*mod.ActorConditionTag)
				testza.AssertTrue(t, isCond, "%q: tag %T is not an ActorCondition", tc.line, tag)
				if isCond {
					testza.AssertEqual(t, "enemy", *cond.Actor, "%q", tc.line)
					testza.AssertEqual(t, []string{tc.enemyCond}, cond.VariableList, "%q", tc.line)
				}
			}
		})
	}
}

// TestAddedDamageEqualToStatFamilyIsComplete: ModParser.lua:5391 and :5395 sat BETWEEN two ported neighbours.
func TestAddedDamageEqualToStatFamilyIsComplete(t *testing.T) {
	t.Run("ModParser.lua:5395", func(t *testing.T) {
		mods, extra := parseMod("Spells deal added Chaos Damage equal to 18% of your maximum Life", 1)
		testza.AssertEqual(t, "", extra)
		testza.AssertEqual(t, 2, len(mods))
		if len(mods) != 2 {
			return
		}
		for i, want := range []string{"ChaosMin", "ChaosMax"} {
			m := mods[i]
			testza.AssertEqual(t, want, m.Name())
			testza.AssertEqual(t, mod.TypeBase, m.Type())
			testza.AssertEqual(t, float64(1), m.Value().Float(),
				"the VALUE is 1 and the PercentStat tag carries the 18 — a port that put 18 here "+
					"would be off by the whole of the character's Life")
			testza.AssertEqual(t, 2, len(m.Tags()), "PercentStat(Life, 18) and SkillType(Spell)")

			var sawPercent, sawSpell bool
			for _, tag := range m.Tags() {
				switch tg := tag.(type) {
				case *mod.PercentStatTag:
					sawPercent = true
					testza.AssertEqual(t, "Life", tg.Stat)
					testza.AssertEqual(t, float64(18), tg.Percent)
				case *mod.SkillTypeTag:
					sawSpell = true
					testza.AssertEqual(t, []string{"Spell"}, tg.SkillTypeList)
					testza.AssertFalse(t, tg.Negative)
				}
			}
			testza.AssertTrue(t, sawPercent, "mod %d has no PercentStat tag", i)
			testza.AssertTrue(t, sawSpell,
				"mod %d has no SkillType tag — ungated it adds chaos damage to attacks too", i)
		}
	})

	t.Run("ModParser.lua:5391", func(t *testing.T) {
		mods, extra := parseMod("While you have Unbroken Ward, your next non-channelling Attack "+
			"you Use yourself breaks your Ward to gain Added Cold Damage equal to 25% of Ward", 1)
		testza.AssertEqual(t, "", extra)
		testza.AssertEqual(t, 2, len(mods))
		if len(mods) != 2 {
			return
		}
		for i, want := range []string{"ColdMin", "ColdMax"} {
			m := mods[i]
			testza.AssertEqual(t, want, m.Name())
			testza.AssertEqual(t, float64(1), m.Value().Float())
			testza.AssertEqual(t, 5, len(m.Tags()), "mod %d", i)

			var negChannel, posAttack, negWardNotBreak, unbroken bool
			for _, tag := range m.Tags() {
				switch tg := tag.(type) {
				case *mod.SkillTypeTag:
					if len(tg.SkillTypeList) == 1 && tg.SkillTypeList[0] == "Channel" && tg.Negative {
						negChannel = true
					}
					if len(tg.SkillTypeList) == 1 && tg.SkillTypeList[0] == "Attack" && !tg.Negative {
						posAttack = true
					}
				case *mod.ConditionTag:
					if len(tg.VarList) == 1 && tg.VarList[0] == "WardNotBreak" && tg.Negative {
						negWardNotBreak = true
					}
					if len(tg.VarList) == 1 && tg.VarList[0] == "UnbrokenWard" && !tg.Negative {
						unbroken = true
					}
				}
			}
			testza.AssertTrue(t, negChannel, "mod %d: SkillType Channel must be NEGATED", i)
			testza.AssertTrue(t, posAttack, "mod %d: SkillType Attack must be positive", i)
			testza.AssertTrue(t, negWardNotBreak, "mod %d: Condition WardNotBreak must be NEGATED", i)
			testza.AssertTrue(t, unbroken, "mod %d: Condition UnbrokenWard must be positive", i)
		}
	})

	t.Run("neighbour ModParser.lua:5387", func(t *testing.T) {
		mods, extra := parseMod("Gain Added Chaos Damage equal to 20% of Ward", 1)
		testza.AssertEqual(t, "", extra)
		testza.AssertEqual(t, 2, len(mods))
		if len(mods) != 2 {
			return
		}
		for i, want := range []string{"ChaosMin", "ChaosMax"} {
			testza.AssertEqual(t, want, mods[i].Name())
			testza.AssertEqual(t, 1, len(mods[i].Tags()),
				"the ward row takes ONE tag; a new row that swallowed it would show up here")
		}
	})
}
