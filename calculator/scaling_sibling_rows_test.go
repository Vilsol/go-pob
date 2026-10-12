package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestScalingSiblingRowsAreComplete pins three parser FAMILIES where upstream writes an `increased` row and a `reduced` row side by side and we carried exactly one of each pair.
func TestScalingSiblingRowsAreComplete(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line      string
		wantName  string
		wantVal   float64
		wantTags  int
		wantInner float64
	}{
		{"Your Offerings have 20% increased Effect on you", "ExtraSkillMod", 0, 1, 20},
		{"Corpses you Spawn have 5% reduced Maximum Life", "CorpseLife", -5, 0, 0},
		{"Focus has 8% reduced Cooldown Recovery Rate", "FocusCooldownRecovery", -8, 1, 0},
		{"Your Offerings have 20% reduced Effect on you", "ExtraSkillMod", 0, 1, -20},
		{"Corpses you Spawn have 5% increased Maximum Life", "CorpseLife", 5, 0, 0},
		{"Focus has 8% increased Cooldown Recovery Rate", "FocusCooldownRecovery", 8, 1, 0},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e.Extra != "" {
				t.Fatalf("leftover %q -- an unparsed remainder DISCARDS THE WHOLE LINE", e.Extra)
			}
			if len(e.ModList) != 1 {
				t.Fatalf("got %d mods, want 1: %+v", len(e.ModList), e.ModList)
			}
			m := e.ModList[0]
			if m.Name() != tc.wantName {
				t.Fatalf("name = %q, want %q", m.Name(), tc.wantName)
			}
			if len(m.Tags()) != tc.wantTags {
				t.Fatalf("tags = %d, want %d", len(m.Tags()), tc.wantTags)
			}
			if tc.wantName == "ExtraSkillMod" {
				inner, ok := m.Value().List().(mod.ExtraSkillMod)
				if !ok {
					t.Fatalf("ExtraSkillMod payload is %T", m.Value().List())
				}
				if got := inner.Mod.Value().Float(); got != tc.wantInner {
					t.Fatalf("inner BuffEffectOnPlayer = %v, want %v", got, tc.wantInner)
				}
				if inner.Mod.Name() != "BuffEffectOnPlayer" {
					t.Fatalf("inner name = %q, want BuffEffectOnPlayer", inner.Mod.Name())
				}
				return
			}
			if m.Type() != mod.TypeIncrease {
				t.Fatalf("type = %v, want INC", m.Type())
			}
			if got := m.Value().Float(); got != tc.wantVal {
				t.Fatalf("value = %v, want %v", got, tc.wantVal)
			}
		})
	}
}
