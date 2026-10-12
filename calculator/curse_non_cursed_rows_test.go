package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestCurseNonCursedEnemiesRows pins ModParser.lua:3352 and the empty-SkillID defect it exposed in its two siblings (:3340 and :3348).
func TestCurseNonCursedEnemiesRows(t *testing.T) {
	curseSkill := func(t *testing.T, line string) mod.ExtraSkill {
		t.Helper()
		e := ParseMod(line, false)
		if e == nil || e.Extra != "" || len(e.ModList) == 0 {
			t.Fatalf("got %v", e)
		}
		m := e.ModList[0]
		if m.Name() != "ExtraSkill" || m.Type() != mod.TypeList {
			t.Fatalf("got %s %s, want ExtraSkill LIST", m.Name(), m.Type())
		}
		es, ok := m.Value().List().(mod.ExtraSkill)
		if !ok {
			t.Fatalf("value = %T, want mod.ExtraSkill", m.Value().List())
		}
		if es.Level != 1 || !es.NoSupports || !es.Triggered {
			t.Errorf("got level %d noSupports %v triggered %v, want 1 true true", es.Level, es.NoSupports, es.Triggered)
		}
		return es
	}

	t.Run("curse non-cursed enemies resolves a real hex to a skill id", func(t *testing.T) {
		requireGameData(t, "3.29")
		es := curseSkill(t, "10% chance to Curse Non-Cursed Enemies with Enfeeble on Hit")
		if es.SkillID == "" {
			t.Fatal("SkillID is empty; item.go:910 drops the mod and the curse is never granted")
		}
	})

	t.Run("the sibling with increased effect resolves it too", func(t *testing.T) {
		requireGameData(t, "3.29")
		es := curseSkill(t, "25% chance to Curse Non-Cursed Enemies with Enfeeble on Hit, with 20% increased effect")
		if es.SkillID == "" {
			t.Fatal("SkillID is empty; item.go:910 drops the mod and the curse is never granted")
		}
	})

	t.Run("an unresolvable name keeps the empty id upstream also produces", func(t *testing.T) {
		requireGameData(t, "3.29")
		es := curseSkill(t, "10% chance to Curse non-Cursed Enemies with a random Hex on Hit")
		if es.SkillID != "" {
			t.Errorf("SkillID = %q, want empty -- upstream cannot resolve this name either", es.SkillID)
		}
	})
}
