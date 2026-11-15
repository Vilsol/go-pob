package pob

import (
	"encoding/xml"
	"testing"
)

// TestSkillLoadDefaults: SkillsTab.lua:447 and :498-501 default the attributes a saved Skill or Gem leaves out.
func TestSkillLoadDefaults(t *testing.T) {
	var s Skills
	in := `<Skills><SkillSet id="1">` +
		`<Skill><Gem nameSpec="Bare"/><Gem nameSpec="Off" enabled="false" enableGlobal1="false" enableGlobal2="true"/></Skill>` +
		`<Skill active="true"><Gem nameSpec="X"/></Skill>` +
		`<Skill enabled="true"><Gem nameSpec="Y"/></Skill>` +
		`</SkillSet></Skills>`
	if err := xml.Unmarshal([]byte(in), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	groups := s.SkillSets[0].Skills
	bare, off := groups[0].Gems[0], groups[0].Gems[1]
	if !bare.Enabled || !bare.EnableGlobal1 || bare.EnableGlobal2 {
		t.Errorf("bare gem: enabled %v enableGlobal1 %v enableGlobal2 %v, want true true false", bare.Enabled, bare.EnableGlobal1, bare.EnableGlobal2)
	}
	if off.Enabled || off.EnableGlobal1 || !off.EnableGlobal2 {
		t.Errorf("explicit gem: enabled %v enableGlobal1 %v enableGlobal2 %v, want false false true", off.Enabled, off.EnableGlobal1, off.EnableGlobal2)
	}
	if groups[0].Enabled || !groups[1].Enabled || !groups[2].Enabled {
		t.Errorf("group enabled: %v %v %v, want false true true", groups[0].Enabled, groups[1].Enabled, groups[2].Enabled)
	}
}
