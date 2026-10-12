package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestStarsAreAlignedLinesParseWhole pins all four of the amulet's lines: ModParser.lua:2704-2719, :3185-3186 and :5442-5445.
func TestStarsAreAlignedLinesParseWhole(t *testing.T) {
	for _, tc := range []struct {
		line     string
		wantMods int
		check    func(t *testing.T, e *ModCacheEntry)
	}{
		{
			line: "+3 to Level of all Elemental Skill Gems if the stars are aligned", wantMods: 1,
			check: func(t *testing.T, e *ModCacheEntry) {
				assertGemLevel(t, e, 3, []string{"elemental", "grants_active_skill"})
			},
		},
		{
			line: "+3 to Level of all Elemental Support Gems if the stars are aligned", wantMods: 1,
			check: func(t *testing.T, e *ModCacheEntry) {
				assertGemLevel(t, e, 3, []string{"elemental", "support"})
			},
		},
		{
			line: "The stars are aligned if you have 6 Influence types among other Equipped Items", wantMods: 1,
			check: func(t *testing.T, e *ModCacheEntry) {
				m := e.ModList[0]
				if m.Name() != "Condition:StarsAreAligned" {
					t.Fatalf("got %s, want Condition:StarsAreAligned", m.Name())
				}
				if len(m.Tags()) != 6 {
					t.Fatalf("got %d tags, want 6 -- one influence type has been dropped", len(m.Tags()))
				}
				for _, tag := range m.Tags() {
					mt, ok := tag.(*mod.MultiplierThresholdTag)
					if !ok {
						t.Fatalf("tag %#v is not a MultiplierThreshold", tag)
					}
					if mt.TagThreshold == nil || *mt.TagThreshold != 2 {
						t.Fatalf("%s threshold is %v, want 2", mt.Variable, mt.TagThreshold)
					}
				}
			},
		},
		{
			line: "You have Elemental Conflux if the stars are aligned", wantMods: 14,
			check: func(t *testing.T, e *ModCacheEntry) {
				for _, m := range e.ModList {
					if len(m.Tags()) != 1 {
						t.Fatalf("%s carries %d tags, want 1", m.Name(), len(m.Tags()))
					}
					c, ok := m.Tags()[0].(*mod.ConditionTag)
					if !ok || len(c.VarList) != 1 || c.VarList[0] != "StarsAreAligned" {
						t.Fatalf("%s tag is %#v, want Condition:StarsAreAligned", m.Name(), m.Tags()[0])
					}
				}
			},
		},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e.Extra != "" {
				t.Fatalf("unparsed remainder %q -- the WHOLE line is discarded", e.Extra)
			}
			if len(e.ModList) != tc.wantMods {
				t.Fatalf("got %d mods, want %d", len(e.ModList), tc.wantMods)
			}
			tc.check(t, e)
		})
	}
}

func assertGemLevel(t *testing.T, e *ModCacheEntry, want float64, keywords []string) {
	t.Helper()
	m := e.ModList[0]
	if m.Name() != "GemProperty" {
		t.Fatalf("got %s, want GemProperty", m.Name())
	}
	prop, ok := m.Value().List().(mod.GemProperty)
	if !ok {
		t.Fatalf("list value is %#v, want a GemProperty", m.Value().List())
	}
	if prop.Key != "level" || prop.Value != want {
		t.Fatalf("got %s=%v, want level=%v", prop.Key, prop.Value, want)
	}
	if len(prop.KeywordList) != len(keywords) {
		t.Fatalf("got keywords %v, want %v", prop.KeywordList, keywords)
	}
	for i := range keywords {
		if prop.KeywordList[i] != keywords[i] {
			t.Fatalf("got keywords %v, want %v", prop.KeywordList, keywords)
		}
	}
	if len(m.Tags()) != 1 {
		t.Fatalf("got %d tags, want 1 (Condition:StarsAreAligned)", len(m.Tags()))
	}
	c, ok := m.Tags()[0].(*mod.ConditionTag)
	if !ok || len(c.VarList) != 1 || c.VarList[0] != "StarsAreAligned" {
		t.Fatalf("tag is %#v, want Condition:StarsAreAligned", m.Tags()[0])
	}
}
