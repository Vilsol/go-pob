package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestShrineBuffLinesParseToConditionsOnly pins the two Blunderbore lines to the shape the pinned PoB gives them: a bare CONDITION, and nothing else.
func TestShrineBuffLinesParseToConditionsOnly(t *testing.T) {
	for _, tc := range []struct {
		line string
		cond string
	}{
		{"You have Lesser Brutal Shrine Buff", "Condition:LesserBrutalShrine"},
		{"You have Lesser Massive Shrine Buff", "Condition:LesserMassiveShrine"},
	} {
		t.Run(tc.cond, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || e.Extra != "" {
				t.Fatalf("line did not parse whole: %v", e)
			}
			if len(e.ModList) != 1 {
				t.Fatalf("got %d mods, want exactly 1: %v", len(e.ModList), e.ModList)
			}
			m := e.ModList[0]
			if m.Name() != tc.cond || m.Type() != mod.TypeFlag {
				t.Fatalf("got %s %s, want %s FLAG", m.Name(), m.Type(), tc.cond)
			}
			if !m.Value().Flag() {
				t.Errorf("value = %v, want true", m.Value())
			}
		})
	}

	t.Run("no EnemyKnockbackChance from either line", func(t *testing.T) {
		for _, line := range []string{
			"You have Lesser Brutal Shrine Buff",
			"You have Lesser Massive Shrine Buff",
		} {
			for _, m := range ParseMod(line, false).ModList {
				if m.Name() == "EnemyKnockbackChance" {
					t.Errorf("%q emits %s %s %v; CalcOffence.lua:4142 sums BASE, so upstream's "+
						"INC 100 from CalcPerform.lua:354 must be the ONLY source",
						line, m.Name(), m.Type(), m.Value())
				}
			}
		}
	})
}
