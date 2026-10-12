package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSkillsFromEquippedSlotAreSupported pins ModParser.lua:3373, `["skills from equipped (.+) are supported by level (%d+) (.+)"]` -> extraSupport(...).
func TestSkillsFromEquippedSlotAreSupported(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("Skills Socketed in your Helmet are Supported by level 20 Cold Penetration", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the sibling `skills socketed in your <slot>` row does not parse; " +
			"this test cannot attribute a failure to the missing equipped-slot row")
	}

	for _, tc := range []struct {
		line  string
		level int
		slot  string
	}{
		{"Skills from Equipped Body Armour are Supported by Level 20 Fist of War", 20, "Body Armour"},
		{"Skills from Equipped Body Armour are Supported by Level 30 Ancestral Call", 30, "Body Armour"},
		{"Skills from Equipped Main Hand Weapon are supported by level 1 Cruelty", 1, "Weapon 1"},
	} {
		e := ParseMod(tc.line, false)
		if e == nil {
			t.Errorf("no entry: %s", tc.line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q (line is DROPPED WHOLE): %s", extra, tc.line)
			continue
		}
		var found *mod.ExtraSupport
		for _, m := range e.ModList {
			if m.Name() != "ExtraSupport" {
				continue
			}
			if es, ok := m.Value().ValueList.(mod.ExtraSupport); ok {
				found = &es
			}
		}
		if found == nil {
			t.Errorf("no ExtraSupport produced: %s", tc.line)
			continue
		}
		if found.Level != tc.level {
			t.Errorf("ExtraSupport level = %d, want %d: %s", found.Level, tc.level, tc.line)
		}
		var slot string
		for _, m := range e.ModList {
			for _, tg := range m.Tags() {
				if si, ok := tg.(*mod.SocketedInTag); ok {
					slot = si.SlotName
				}
			}
		}
		if slot != tc.slot {
			t.Errorf("SocketedIn slot = %q, want %q: %s", slot, tc.slot, tc.line)
		}
	}
}
