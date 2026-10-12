package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestWeaponAddedDamageFromPoolParses covers ModParser.lua:5366-5386.
func TestWeaponAddedDamageFromPoolParses(t *testing.T) {
	exactlyOne := func(t *testing.T, line, name string, stat string, percent float64) {
		t.Helper()
		mods, extra := parseMod(line, 1)
		testza.AssertEqual(t, "", extra, "the line must parse completely: "+line)
		if len(mods) != 1 {
			t.Fatalf("%q produced %d mods, want exactly 1", line, len(mods))
		}
		m := mods[0]
		testza.AssertEqual(t, name, m.Name())
		testza.AssertEqual(t, mod.TypeBase, m.Type())
		testza.AssertEqual(t, 1.0, m.Value().ValueFloat, "PoB threads the pool through the tag, so the mod value is 1")

		var ps *mod.PercentStatTag
		var cond *mod.ConditionTag
		var st *mod.SkillTypeTag
		for _, tag := range m.Tags() {
			switch tg := tag.(type) {
			case *mod.PercentStatTag:
				ps = tg
			case *mod.ConditionTag:
				cond = tg
			case *mod.SkillTypeTag:
				st = tg
			}
		}
		if ps == nil || cond == nil || st == nil {
			t.Fatalf("%q: want PercentStat+Condition+SkillType tags, got %d tags (%+v)", line, len(m.Tags()), m.Tags())
		}
		testza.AssertEqual(t, stat, ps.Stat)
		testza.AssertEqual(t, percent, ps.Percent)
		testza.AssertTrue(t, len(cond.VarList) == 1 && cond.VarList[0] == "{Hand}Attack", "must carry the {Hand}Attack condition")
		testza.AssertTrue(t, len(st.SkillTypeList) == 1 && st.SkillTypeList[0] == string(data.SkillTypeAttack), "must be gated to attacks")
	}

	t.Run("player's maximum energy shield", func(t *testing.T) {
		exactlyOne(t, "Attacks with this Weapon have Added Maximum Lightning Damage equal to 13% of Player's Maximum Energy Shield", "LightningMax", "EnergyShield", 13)
	})
	t.Run("your maximum energy shield", func(t *testing.T) {
		exactlyOne(t, "Attacks with this Weapon have Added Maximum Lightning Damage equal to 20% of your Maximum Energy Shield", "LightningMax", "EnergyShield", 20)
	})
	t.Run("your energy shield (upstream typo row)", func(t *testing.T) {
		exactlyOne(t, "Attacks with this Weapon have Added Maximum Lightning Damage equal to 20% of your Energy Shield", "LightningMax", "EnergyShield", 20)
	})

	t.Run("fire damage from player's maximum life", func(t *testing.T) {
		mods, extra := parseMod("Attacks with this Weapon have Added Fire Damage equal to 10% of Player's Maximum Life", 1)
		testza.AssertEqual(t, "", extra, "the line must parse completely")
		if len(mods) != 2 {
			t.Fatalf("produced %d mods, want exactly 2 (FireMin and FireMax)", len(mods))
		}
		testza.AssertEqual(t, "FireMin", mods[0].Name())
		testza.AssertEqual(t, "FireMax", mods[1].Name())
	})

	t.Run("cold damage from maximum energy shield", func(t *testing.T) {
		mods, extra := parseMod("Adds 25% of your Maximum Energy Shield as Cold Damage to Attacks with this Weapon", 1)
		testza.AssertEqual(t, "", extra, "the line must parse completely")
		if len(mods) != 2 {
			t.Fatalf("produced %d mods, want exactly 2 (ColdMin and ColdMax)", len(mods))
		}
		testza.AssertEqual(t, "ColdMin", mods[0].Name())
		testza.AssertEqual(t, "ColdMax", mods[1].Name())
	})
	t.Run("fire damage from maximum mana", func(t *testing.T) {
		mods, extra := parseMod("Adds 25% of your Maximum Mana as Fire Damage to Attacks with this Weapon", 1)
		testza.AssertEqual(t, "", extra, "the line must parse completely")
		if len(mods) != 2 {
			t.Fatalf("produced %d mods, want exactly 2 (FireMin and FireMax)", len(mods))
		}
		testza.AssertEqual(t, "FireMin", mods[0].Name())
		testza.AssertEqual(t, "FireMax", mods[1].Name())
	})
}
