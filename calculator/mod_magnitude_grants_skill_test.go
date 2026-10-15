package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestModMagnitudeSkipsSkillGrantingLines pins Item.lua:1417-1428 / :1449.
func TestModMagnitudeSkipsSkillGrantingLines(t *testing.T) {
	requireGameData(t, "3.29")

	item := &Item{}
	item.ParseRaw(`Rarity: RARE
Plague Knuckle
Helical Ring
Crafted: true
Implicits: 1
50% increased Suffix Modifier magnitudes
{suffix}Grants Level 20 Aspect of !!UNPARSEABLE!! Skill
{suffix}Grants Level 20 Aspect of the Spider Skill`)

	found, extraSkill := 0, false
	for _, ml := range item.ExplicitModLines {
		if !strings.Contains(ml.Line, "Grants Level") {
			continue
		}
		found++
		if ml.ValueScalar != 0 && ml.ValueScalar != 1 {
			t.Errorf("%q: valueScalar = %v, want unscaled", ml.Line, ml.ValueScalar)
		}
		for _, m := range ml.ModList {
			if m.Name() != "ExtraSkill" {
				continue
			}
			extraSkill = true
			if lvl := m.Value().List().(mod.ExtraSkill).Level; lvl != 20 {
				t.Errorf("%q: ExtraSkill level = %d, want 20", ml.Line, lvl)
			}
		}
		if strings.Contains(ml.Line, "UNPARSEABLE") && ml.Extra == "" {
			t.Errorf("%q: extra is empty, want the unparsed line", ml.Line)
		}
	}
	if found != 2 || !extraSkill {
		t.Fatalf("found %d granting lines, ExtraSkill %v; want 2 and true", found, extraSkill)
	}
}
