package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSocketedTriggerAndInfluenceGemLevelRowsParse is the third tranche off the parser-coverage worklist.
func TestSocketedTriggerAndInfluenceGemLevelRowsParse(t *testing.T) {
	t.Run("socketed triggers", func(t *testing.T) {
		for _, tc := range []struct {
			line    string
			skillID string
		}{
			{"Trigger a Socketed Elemental Spell on Block, with a 0.25 second Cooldown", "SupportTriggerElementalSpellOnBlock"},
			{"Trigger Socketed Curse Spell when you Cast a Curse Spell, with a 0.25 second Cooldown", "SupportUniqueCastCurseOnCurse"},
		} {
			e := ParseMod(tc.line, false)
			if e == nil || e.Extra != "" || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) = %v, want exactly 1 mod and no Extra", tc.line, e)
			}
			m := e.ModList[0]
			if m.Name() != "ExtraSupport" {
				t.Fatalf("mod name = %q, want ExtraSupport", m.Name())
			}
			es, ok := m.Value().List().(mod.ExtraSupport)
			if !ok {
				t.Fatalf("value is %#v, want mod.ExtraSupport", m.Value().List())
			}
			if es.SkillID != tc.skillID {
				t.Errorf("skillId = %q, want %q", es.SkillID, tc.skillID)
			}
			if es.Level != 1 {
				t.Errorf("level = %d, want 1", es.Level)
			}
			if len(m.Tags()) != 1 {
				t.Fatalf("got %d tags, want exactly 1 SocketedIn", len(m.Tags()))
			}
			if _, ok := m.Tags()[0].(*mod.SocketedInTag); !ok {
				t.Errorf("tag is %#v, want a SocketedIn", m.Tags()[0])
			}
		}
	})

	t.Run("influence gem levels", func(t *testing.T) {
		for _, tc := range []struct {
			line      string
			keyword   string
			level     float64
			threshold float64
			variable  string
		}{
			{"+1 to Level of all Chaos Skill Gems if 6 Hunter Items are Equipped", "chaos", 1, 6, "HunterItem"},
			{"+1 to Level of all Lightning Skill Gems if 6 Crusader Items are Equipped", "lightning", 1, 6, "CrusaderItem"},
			{"+1 to Level of all non-Exceptional Support Gems if 6 Shaper Items are Equipped", "non-exceptional", 1, 6, "ShaperItem"},
		} {
			e := ParseMod(tc.line, false)
			if e == nil || e.Extra != "" || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) = %v, want exactly 1 mod and no Extra", tc.line, e)
			}
			m := e.ModList[0]
			if m.Name() != "GemProperty" {
				t.Fatalf("mod name = %q, want GemProperty", m.Name())
			}
			gp, ok := m.Value().List().(mod.GemProperty)
			if !ok {
				t.Fatalf("value is %#v, want mod.GemProperty", m.Value().List())
			}
			got := ""
			if gp.Keyword != nil {
				got = *gp.Keyword
			}
			if got != tc.keyword {
				t.Errorf("keyword = %q, want %q", got, tc.keyword)
			}
			if gp.Key != "level" || gp.Value != tc.level {
				t.Errorf("key/value = %q/%v, want level/%v", gp.Key, gp.Value, tc.level)
			}
			if len(m.Tags()) != 1 {
				t.Fatalf("got %d tags, want exactly 1 MultiplierThreshold", len(m.Tags()))
			}
			mt, ok := m.Tags()[0].(*mod.MultiplierThresholdTag)
			if !ok {
				t.Fatalf("tag is %#v, want a MultiplierThreshold", m.Tags()[0])
			}
			if mt.Variable != tc.variable {
				t.Errorf("threshold var = %q, want %q", mt.Variable, tc.variable)
			}
			if mt.TagThreshold == nil || *mt.TagThreshold != tc.threshold {
				t.Errorf("threshold = %v, want %v", mt.TagThreshold, tc.threshold)
			}
		}
	})

	for _, line := range []string{"+1 to Level of all Physical Skill Gems", "+1 to Level of all Spell Skill Gems"} {
		e := ParseMod(line, false)
		if e == nil || len(e.ModList) != 1 || e.Extra != "" {
			t.Errorf("control %q: ParseMod produced %v, want exactly 1 mod and no Extra", line, e)
			continue
		}
		if len(e.ModList[0].Tags()) != 0 {
			t.Errorf("control %q picked up %d tags; it is unconditional", line, len(e.ModList[0].Tags()))
		}
	}
	const bowLine = "Trigger a Socketed Bow Skill when you Attack with a Bow"
	if e := ParseMod(bowLine, false); e == nil || len(e.ModList) != 1 || e.Extra != "" {
		t.Errorf("control %q: ParseMod produced %v, want exactly 1 mod and no Extra", bowLine, e)
	}
}
