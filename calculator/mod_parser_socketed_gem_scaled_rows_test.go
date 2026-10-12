package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSocketedGemScaledLevelRowsParse pins the two socketed-gem level rows whose value is scaled by a Multiplier tag and which had no counterpart here at all.
func TestSocketedGemScaledLevelRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line  string
		value float64
		tag   string
	}{
		{"+1 to Level of Socketed Skill Gems per 25 Player Levels", 1, "Level"},
		{"-2 to level of Socketed Skill Gems per Socketed Gem", -2, "SocketedGemsIn{SlotName}"},
	} {
		g := onlyGemProperty(t, tc.line)
		if g == nil {
			continue
		}
		if g.Key != "level" || g.Value != tc.value {
			t.Errorf("%q -> key=%q value=%v, want level/%v", tc.line, g.Key, g.Value, tc.value)
			continue
		}
		if !hasKeyword(g, "active_skill") {
			t.Errorf("%q -> keyword %v/%v, want the active-skill keyword", tc.line, g.Keyword, g.KeywordList)
		}
		e := ParseMod(tc.line, false)
		var mults []string
		for _, tag := range e.ModList[0].Tags() {
			if m, ok := tag.(*mod.MultiplierTag); ok {
				mults = append(mults, m.VariableList...)
			}
		}
		if !contains(mults, tc.tag) {
			t.Errorf("%q -> multipliers %v, want %q: without it the value never scales",
				tc.line, mults, tc.tag)
		}
	}
}
