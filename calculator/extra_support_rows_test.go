package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSocketedTriggerExtraSupportRows pins two ExtraSupport rows the pin parses whole and this tree did not.
func TestSocketedTriggerExtraSupportRows(t *testing.T) {
	extraSupport := func(t *testing.T, e *ModCacheEntry, wantID string) {
		t.Helper()
		if e == nil || e.Extra != "" || len(e.ModList) != 1 {
			t.Fatalf("got %v", e)
		}
		m := e.ModList[0]
		if m.Name() != "ExtraSupport" || m.Type() != mod.TypeList {
			t.Fatalf("got %s %s, want ExtraSupport LIST", m.Name(), m.Type())
		}
		es, ok := m.Value().List().(mod.ExtraSupport)
		if !ok {
			t.Fatalf("value = %T, want mod.ExtraSupport", m.Value().List())
		}
		if es.SkillID != wantID || es.Level != 1 {
			t.Errorf("got %s level %d, want %s level 1", es.SkillID, es.Level, wantID)
		}
		var si *mod.SocketedInTag
		for _, tg := range m.Tags() {
			if x, ok := tg.(*mod.SocketedInTag); ok {
				si = x
			}
		}
		if si == nil {
			t.Errorf("no SocketedIn tag; tags = %v", m.Tags())
		}
	}

	t.Run("trigger a socketed spell on freeze", func(t *testing.T) {
		extraSupport(t, ParseMod("Trigger a Socketed Spell when a Hit from this Weapon Freezes a Target, with a 0.25 second Cooldown", false),
			"SupportTriggerSpellOnBowAttackFreezeHit")
	})

	t.Run("socketed hexes on enemies that trigger your traps", func(t *testing.T) {
		extraSupport(t, ParseMod("You cannot Cast Socketed Hex Curse Skills Inflict Socketed Hexes on Enemies that trigger your Traps", false),
			"SupportCurseOnTrapTriggered")
	})
}
