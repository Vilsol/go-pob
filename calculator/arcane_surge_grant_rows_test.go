package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestUniqueEnemyHitAndTotemSpellHitTagRows pins ModParser.lua:1323 and :1782, two modTagList SUFFIXES that were absent here.
func TestUniqueEnemyHitAndTotemSpellHitTagRows(t *testing.T) {
	t.Run("when you hit a unique enemy", func(t *testing.T) {
		e := ParseMod("10% chance to gain Arcane Surge when you Hit a Unique enemy", false)
		if e == nil || len(e.ModList) != 1 || e.Extra != "" {
			t.Fatalf("got %v", e)
		}
		if e.ModList[0].Name() != "Condition:ArcaneSurge" {
			t.Fatalf("name = %q", e.ModList[0].Name())
		}
		var ac *mod.ActorConditionTag
		for _, tg := range e.ModList[0].Tags() {
			if c, ok := tg.(*mod.ActorConditionTag); ok {
				ac = c
			}
		}
		if ac == nil {
			t.Fatalf("no ActorConditionTag; tags = %v", e.ModList[0].Tags())
		}
		if ac.Actor == nil || *ac.Actor != "enemy" || len(ac.VariableList) != 1 || ac.VariableList[0] != "RareOrUnique" {
			t.Errorf("tag = %+v, want enemy/RareOrUnique", ac)
		}
	})

	t.Run("when you or your totems hit an enemy with a spell", func(t *testing.T) {
		e := ParseMod("Gain Arcane Surge when you or your Totems Hit an Enemy with a Spell", false)
		if e == nil || len(e.ModList) != 1 || e.Extra != "" {
			t.Fatalf("got %v", e)
		}
		if e.ModList[0].Name() != "Condition:ArcaneSurge" {
			t.Fatalf("name = %q", e.ModList[0].Name())
		}
		var ct *mod.ConditionTag
		for _, tg := range e.ModList[0].Tags() {
			if c, ok := tg.(*mod.ConditionTag); ok {
				ct = c
			}
		}
		if ct == nil {
			t.Fatalf("no ConditionTag; tags = %v", e.ModList[0].Tags())
		}
		if len(ct.VarList) != 2 || ct.VarList[0] != "HitSpellRecently" || ct.VarList[1] != "TotemsHitSpellRecently" {
			t.Errorf("varList = %v, want [HitSpellRecently TotemsHitSpellRecently]", ct.VarList)
		}
	})
}
