package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestGemPropertyRowsParseWhole pins the gem-property row family against the pin's own rows.
func TestGemPropertyRowsParseWhole(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse")
	}

	for _, tc := range []struct {
		line    string
		keyword string
		key     string
		value   float64
	}{
		{"+30% to Quality of all Skill Gems", "active_skill", "quality", 30},
		{"+1 to Level of Socketed Vaal Gems", "vaal", "level", 1},
		{"+2 to Level of Socketed Gems", "all", "level", 2},
	} {
		g := onlyGemProperty(t, tc.line)
		if g == nil {
			continue
		}
		if g.Keyword == nil || !strings.EqualFold(*g.Keyword, tc.keyword) || g.Key != tc.key || g.Value != tc.value {
			t.Errorf("CONTROL MOVED: %q -> keyword=%v key=%q value=%v, want %q/%q/%v",
				tc.line, g.Keyword, g.Key, g.Value, tc.keyword, tc.key, tc.value)
		}
	}

	t.Run("coverage", func(t *testing.T) {
		if g := onlyGemProperty(t, "+30% to Quality of all Minion Skill Gems"); g != nil {
			if g.Key != "quality" || g.Value != 30 || !hasKeyword(g, "minion") {
				t.Errorf("+30%% to Quality of all Minion Skill Gems -> %+v", *g)
			}
		}
		if g := onlyGemProperty(t, "-5 to Level of Socketed Non-Vaal Gems"); g != nil {
			if g.Key != "level" || g.Value != -5 || g.Keyword == nil || !strings.EqualFold(*g.Keyword, "non-vaal") {
				t.Errorf("-5 to Level of Socketed Non-Vaal Gems -> %+v", *g)
			}
		}
	})
}

func hasKeyword(g *mod.GemProperty, want string) bool {
	if g.Keyword != nil && strings.EqualFold(*g.Keyword, want) {
		return true
	}
	for _, k := range g.KeywordList {
		if strings.EqualFold(k, want) {
			return true
		}
	}
	return false
}

func onlyGemProperty(t *testing.T, line string) *mod.GemProperty {
	t.Helper()

	e := ParseMod(line, false)
	if e == nil || len(e.ModList) == 0 {
		t.Errorf("DROPPED WHOLE (no mods): %s", line)
		return nil
	}
	if extra := strings.TrimSpace(modExtra(e)); extra != "" {
		t.Errorf("leftover %q, so the line is discarded: %s", extra, line)
		return nil
	}
	if len(e.ModList) != 1 {
		t.Errorf("want 1 mod, got %d: %s", len(e.ModList), line)
		return nil
	}
	m := e.ModList[0]
	if m.Name() != "GemProperty" {
		t.Errorf("want GemProperty, got %q: %s", m.Name(), line)
		return nil
	}
	if m.Value().Type() != mod.ModValueMultiTypeList {
		t.Errorf("value is %v, not a list: %s", m.Value().Type(), line)
		return nil
	}
	g, ok := m.Value().List().(mod.GemProperty)
	if !ok {
		t.Errorf("payload is %T, not GemProperty: %s", m.Value().List(), line)
		return nil
	}
	return &g
}
