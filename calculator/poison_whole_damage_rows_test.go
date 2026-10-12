package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestWholeDamageCanPoisonAndTriggeredPoisonRows pins five specialModList rows that were absent here while the pin parses each of them whole.
func TestWholeDamageCanPoisonAndTriggeredPoisonRows(t *testing.T) {
	elements := []string{"FireCanPoison", "ColdCanPoison", "LightningCanPoison"}

	assertElementFlags := func(t *testing.T, e *ModCacheEntry) {
		t.Helper()
		if e == nil || e.Extra != "" || len(e.ModList) != 3 {
			t.Fatalf("got %v", e)
		}
		for i, want := range elements {
			m := e.ModList[i]
			if m.Name() != want || m.Type() != mod.TypeFlag || !m.Value().Flag() {
				t.Errorf("mod %d = %s %s %v, want %s FLAG true", i, m.Name(), m.Type(), m.Value(), want)
			}
		}
	}

	t.Run("all damage from hits can poison", func(t *testing.T) {
		assertElementFlags(t, ParseMod("All Damage from Hits can Poison", false))
	})

	t.Run("all damage can poison", func(t *testing.T) {
		assertElementFlags(t, ParseMod("All Damage can Poison", false))
	})

	t.Run("all damage with triggered spells can poison", func(t *testing.T) {
		e := ParseMod("All Damage with Triggered Spells can Poison", false)
		assertElementFlags(t, e)
		for _, m := range e.ModList {
			if m.KeywordFlags()&mod.KeywordFlagSpell == 0 {
				t.Errorf("%s keywordFlags = %d, want KeywordFlagSpell set", m.Name(), m.KeywordFlags())
			}
			var st *mod.SkillTypeTag
			for _, tg := range m.Tags() {
				if s, ok := tg.(*mod.SkillTypeTag); ok {
					st = s
				}
			}
			if st == nil || len(st.SkillTypeList) != 1 || st.SkillTypeList[0] != string(data.SkillTypeTriggered) {
				t.Errorf("%s SkillType tag = %+v, want Triggered", m.Name(), st)
			}
		}
	})

	t.Run("triggered spells poison on hit", func(t *testing.T) {
		e := ParseMod("Triggered Spells Poison on Hit", false)
		if e == nil || e.Extra != "" || len(e.ModList) != 1 {
			t.Fatalf("got %v", e)
		}
		m := e.ModList[0]
		if m.Name() != "PoisonChance" || m.Type() != mod.TypeBase || m.Value().Float() != 100 {
			t.Fatalf("got %s %s %v, want PoisonChance BASE 100", m.Name(), m.Type(), m.Value())
		}
		if m.KeywordFlags()&mod.KeywordFlagSpell == 0 {
			t.Errorf("keywordFlags = %d, want KeywordFlagSpell set", m.KeywordFlags())
		}
	})

	t.Run("always poison on hit against cursed enemies", func(t *testing.T) {
		e := ParseMod("Always Poison on Hit against Cursed Enemies", false)
		if e == nil || e.Extra != "" || len(e.ModList) != 1 {
			t.Fatalf("got %v", e)
		}
		m := e.ModList[0]
		if m.Name() != "PoisonChance" || m.Value().Float() != 100 {
			t.Fatalf("got %s %v, want PoisonChance 100", m.Name(), m.Value())
		}
		var ac *mod.ActorConditionTag
		for _, tg := range m.Tags() {
			if c, ok := tg.(*mod.ActorConditionTag); ok {
				ac = c
			}
		}
		if ac == nil || ac.Actor == nil || *ac.Actor != "enemy" ||
			len(ac.VariableList) != 1 || ac.VariableList[0] != "Cursed" {
			t.Errorf("tag = %+v, want enemy/Cursed", ac)
		}
	})
}
