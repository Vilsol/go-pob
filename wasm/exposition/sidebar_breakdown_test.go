package exposition

import (
	"testing"

	"github.com/Vilsol/go-pob/calculator"
)

// TestSidebarStatBreakdownKey: Build.lua:1872-1908: which sidebar stats have a breakdown to show, and under which key.
func TestSidebarStatBreakdownKey(t *testing.T) {
	initItemData(t)
	pass := NewSidebarPass(corpusBuild(t, "0280-elementalist-fireball.xml"))
	cases := []struct {
		name                      string
		stat, childStat, override string
		hasOverride, modNames     bool
		wantKey                   string
		wantOK                    bool
	}{
		{name: "the stat's own breakdown", stat: "Life", modNames: true, wantKey: "Life", wantOK: true},
		{name: "an override with lines", stat: "AreaOfEffectRadiusMetres", override: "AreaOfEffectRadius", hasOverride: true, wantKey: "AreaOfEffectRadius", wantOK: true},
		{name: "an override with no lines", stat: "Str", override: "Str", hasOverride: true},
		{name: "an empty override with modNames", stat: "EffectiveSpellSuppressionChance", hasOverride: true, modNames: true, wantKey: "", wantOK: true},
		{name: "an override without a breakdown", stat: "HitChance", override: "HitChance", hasOverride: true},
		{name: "no breakdown for the stat", stat: "ActiveMinionLimit"},
		{name: "a child stat the hand does not have", stat: "MainHand", childStat: "Accuracy"},
	}
	for _, c := range cases {
		key, ok := pass.StatBreakdownKey(false, c.stat, c.childStat, c.override, c.hasOverride, c.modNames)
		if key != c.wantKey || ok != c.wantOK {
			t.Errorf("%s: got (%q, %v), want (%q, %v)", c.name, key, ok, c.wantKey, c.wantOK)
		}
	}
	if e := pass.Entry(false, "Life"); e == nil || len(e.Lines) == 0 {
		t.Errorf("Entry(Life) has no lines")
	}
	if len(pass.ModSection(false, calculator.ModSectionData{ModName: []string{"Life"}, ModType: "BASE"})) == 0 {
		t.Errorf("ModSection(Life BASE) is empty")
	}
}
