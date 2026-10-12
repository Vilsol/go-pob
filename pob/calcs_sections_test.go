package pob

import "testing"

func TestCalcsSectionCollapsedRoundTrip(t *testing.T) {
	b := &PathOfBuilding{}
	if _, ok := b.CalcsSectionCollapsed("HitDamage", "SkillHitDamage"); ok {
		t.Fatal("a fresh build saves no section")
	}
	b.SetCalcsSectionCollapsed("HitDamage", "SkillHitDamage", true)
	b.SetCalcsSectionCollapsed("Life", "Life", false)
	b.SetCalcsSectionCollapsed("HitDamage", "SkillHitDamage", false)
	if c, ok := b.CalcsSectionCollapsed("HitDamage", "SkillHitDamage"); !ok || c {
		t.Errorf("HitDamage: %v %v", c, ok)
	}
	if len(b.Calcs.Sections) != 2 {
		t.Errorf("sections %v", b.Calcs.Sections)
	}
}

func TestStatDifferencesDefaultOnAndSaved(t *testing.T) {
	b := &PathOfBuilding{}
	if !b.StatDifferencesShown() {
		t.Fatal("a build without the attribute must show stat differences (ItemsTab.lua:93)")
	}
	b.SetShowStatDifferences(false)
	if b.StatDifferencesShown() || b.Items.ShowStatDifferences == nil || *b.Items.ShowStatDifferences {
		t.Error("Ctrl+D did not save the toggle")
	}
}
