package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestTriggerLevelEventRows pins fifteen `trigger level N <skill> <event>` rows and the five `onCrit` ones, all absent here while the pin parses each line whole.
func TestTriggerLevelEventRows(t *testing.T) {
	extraSkill := func(t *testing.T, e *ModCacheEntry, wantLevel int) mod.ExtraSkill {
		t.Helper()
		if e == nil || e.Extra != "" || len(e.ModList) == 0 {
			t.Fatalf("got %v", e)
		}
		m := e.ModList[0]
		if m.Name() != "ExtraSkill" || m.Type() != mod.TypeList {
			t.Fatalf("got %s %s, want ExtraSkill LIST", m.Name(), m.Type())
		}
		es, ok := m.Value().List().(mod.ExtraSkill)
		if !ok {
			t.Fatalf("value = %T, want mod.ExtraSkill", m.Value().List())
		}
		if es.Level != wantLevel || !es.Triggered {
			t.Errorf("level = %d triggered = %v, want %d true", es.Level, es.Triggered, wantLevel)
		}
		return es
	}

	for _, tc := range []struct {
		line  string
		skill string
		level int
	}{
		{"Trigger Level 20 Tawhoa's Chosen when you Attack with a Non-Vaal Slam or Strike Skill near an Enemy", "tawhoa's chosen", 20},
		{"Trigger Level 20 Molten Burst on Melee Hit", "molten burst", 20},
		{"Trigger Level 20 Ward Shatter when your Ward Breaks", "ward shatter", 20},
		{"Trigger level 1 Summon Spirit of Kaom Skill when a Totem dies while a Unique Enemy is in your Presence", "summon spirit of kaom", 1},
		{"Trigger level 1 Summon Spirit of Ikiaho Skill when you use a Travel Skill while a Unique Enemy is in your Presence", "summon spirit of ikiaho", 1},
		{"Trigger level 1 Summon Spirit of Akoya Skill when you reach Maximum Rage while a Unique Enemy is in your Presence", "summon spirit of akoya", 1},
		{"Trigger level 1 Summon Spirit of Maata Skill when you reach Low Life while a Unique Enemy is in your Presence", "summon spirit of maata", 1},
		{"Trigger level 1 Summon Spirit of Tawhanuku Skill when Energy Shield Recharge starts while a Unique Enemy is in your Presence", "summon spirit of tawhanuku", 1},
		{"Trigger level 1 Summon Spirit of Kahuturoa Skill when you take a Critical Strike from a Unique Enemy", "summon spirit of kahuturoa", 1},
		{"Trigger level 1 Summon Spirit of Ahuana Skill when you Suppress Spell Damage from a Unique Enemy", "summon spirit of ahuana", 1},
		{"Trigger level 1 Summon Spirit of Kiloava Skill when you Block Damage from a Unique Enemy", "summon spirit of kiloava", 1},
	} {
		t.Run(tc.skill, func(t *testing.T) {
			es := extraSkill(t, ParseMod(tc.line, false), tc.level)
			if es.SkillName != tc.skill {
				t.Errorf("skillName = %q, want %q", es.SkillName, tc.skill)
			}
		})
	}

	t.Run("on melee critical strike carries triggerOnCrit", func(t *testing.T) {
		e := ParseMod("Trigger Level 20 Starfall on Melee Critical Strike", false)
		es := extraSkill(t, e, 20)
		if es.SkillName != "starfall" {
			t.Errorf("skillName = %q, want starfall", es.SkillName)
		}
		if len(e.ModList) != 2 {
			t.Fatalf("got %d mods, want 2 (ExtraSkill + ExtraSkillMod)", len(e.ModList))
		}
		m := e.ModList[1]
		if m.Name() != "ExtraSkillMod" || m.Type() != mod.TypeList {
			t.Fatalf("second mod = %s %s, want ExtraSkillMod LIST", m.Name(), m.Type())
		}
		esm, ok := m.Value().List().(mod.ExtraSkillMod)
		if !ok {
			t.Fatalf("value = %T, want mod.ExtraSkillMod", m.Value().List())
		}
		sd, ok := esm.Mod.Value().List().(mod.SkillData)
		if !ok || sd.Key != "triggerOnCrit" {
			t.Fatalf("inner = %+v, want SkillData triggerOnCrit", esm.Mod.Value().List())
		}
		var sid *mod.SkillIDTag
		for _, tg := range m.Tags() {
			if x, ok := tg.(*mod.SkillIDTag); ok {
				sid = x
			}
		}
		if sid == nil {
			t.Errorf("no SkillId tag; tags = %v", m.Tags())
		}
	})
}
