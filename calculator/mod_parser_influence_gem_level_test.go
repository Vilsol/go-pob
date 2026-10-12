package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestInfluenceGemLevelRowsParse pins ModParser.lua:3188 and :3190 -- the gem-LEVEL members of the "if N <influence> Items are Equipped" family.
func TestInfluenceGemLevelRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line      string
		keyword   string
		value     float64
		multVar   string
		threshold float64
	}{
		{"+1 to Level of all Chaos Skill Gems if 6 Hunter Items are Equipped", "chaos", 1, "HunterItem", 6},
		{"+1 to Level of all Lightning Skill Gems if 6 Crusader Items are Equipped", "lightning", 1, "CrusaderItem", 6},
		{"+1 to Level of all Cold Skill Gems if 6 Redeemer Items are Equipped", "cold", 1, "RedeemerItem", 6},
		{"+1 to Level of all Physical Skill Gems if 6 Elder Items are Equipped", "physical", 1, "ElderItem", 6},
		{"+1 to Level of all Fire Skill Gems if 6 Warlord Items are Equipped", "fire", 1, "WarlordItem", 6},
		{"+1 to Level of all non-Exceptional Support Gems if 6 Shaper Items are Equipped", "non-exceptional", 1, "ShaperItem", 6},
	} {
		g := onlyGemProperty(t, tc.line)
		if g == nil {
			continue
		}
		if g.Key != "level" || g.Value != tc.value {
			t.Errorf("%q -> key=%q value=%v", tc.line, g.Key, g.Value)
		}
		if g.Keyword == nil || !strings.EqualFold(*g.Keyword, tc.keyword) {
			t.Errorf("%q -> keyword %v, want %q", tc.line, g.Keyword, tc.keyword)
		}
		var got []string
		for _, tag := range ParseMod(tc.line, false).ModList[0].Tags() {
			m, ok := tag.(*mod.MultiplierThresholdTag)
			if !ok {
				continue
			}
			got = append(got, m.Variable)
			if m.TagThreshold == nil || *m.TagThreshold != tc.threshold {
				t.Errorf("%q -> threshold %v, want %v", tc.line, m.TagThreshold, tc.threshold)
			}
		}
		if !contains(got, tc.multVar) {
			t.Errorf("%q -> thresholds on %v, want %q: without it the row applies unconditionally",
				tc.line, got, tc.multVar)
		}
	}
}
