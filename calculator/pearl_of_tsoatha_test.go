package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

// TestExtraSupportAppliesToGrantedSkillsRows pins ModParser.lua:2073-2104 and :3419-3420 (upstream #10236).
func TestExtraSupportAppliesToGrantedSkillsRows(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line, slot string
		applies    bool
	}{
		{"Skills Socketed in your Helmet are Supported by Level 20 Elemental Focus", "Helmet", true},
		{"Skills granted by your Passive Tree are Supported by Level 20 Elemental Focus", "Passive Tree", true},
		{"Socketed Gems are Supported by Level 20 Elemental Focus", "{SlotName}", false},
	} {
		entry := ParseMod(tc.line, false)
		if strings.TrimSpace(entry.Extra) != "" || len(entry.ModList) == 0 {
			t.Errorf("%q: extra %q, %d mods", tc.line, entry.Extra, len(entry.ModList))
			continue
		}
		m := entry.ModList[0]
		support, ok := m.Value().List().(mod.ExtraSupport)
		if m.Name() != "ExtraSupport" || !ok {
			t.Errorf("%q: got %s %T", tc.line, m.Name(), m.Value().List())
			continue
		}
		if support.AppliesToGrantedSkills != tc.applies {
			t.Errorf("%q: AppliesToGrantedSkills = %v, want %v", tc.line, support.AppliesToGrantedSkills, tc.applies)
		}
		slot := ""
		for _, tag := range m.Tags() {
			if s, ok := tag.(*mod.SocketedInTag); ok {
				slot = s.SlotName
			}
		}
		if slot != tc.slot {
			t.Errorf("%q: SocketedIn %q, want %q", tc.line, slot, tc.slot)
		}
	}
}

// TestPassiveTreeSkillsTakeTreeSupports pins CalcSetup.lua:1507-1508 and :1551-1557.
func TestPassiveTreeSkillsTakeTreeSupports(t *testing.T) {
	env := warcryFixtureEnv(t, "1606-chieftain-treeskill-4.xml")
	found := false
	for _, as := range env.Player.ActiveSkillList {
		group, ok := as.SocketGroup.(*pob.Skill)
		if !ok || group.Source != "Tree:10916" {
			continue
		}
		found = true
		supported := false
		for _, effect := range as.EffectList {
			if effect.GrantedEffect.Raw.IsSupport && strings.Contains(effect.GrantedEffect.Raw.ID, "IgniteProliferation") {
				supported = true
			}
		}
		if !supported {
			t.Errorf("%s: no Ignite Proliferation support from the tree", as.ActiveEffect.GrantedEffect.ID())
		}
	}
	if !found {
		t.Fatal("CONTROL FAILED: no skill from the Tree:10916 group")
	}
}

// TestItemExtraSupportsTakeTheBest pins CalcSetup.lua:1639-1649 (upstream #10236).
func TestItemExtraSupportsTakeTheBest(t *testing.T) {
	env := warcryFixtureEnv(t, "0331-saboteur-vaal-fireball.xml")
	got := make([]string, 0, len(env.Player.MainSkill.SupportList))
	for _, s := range env.Player.MainSkill.SupportList {
		got = append(got, s.GrantedEffect.Raw.ID)
	}
	if strings.Join(got, ",") != "SupportGreaterVolley,SupportCombustion" {
		t.Errorf("Fireball's supports %v, PoB's Greater Volley, Combustion", got)
	}
}
