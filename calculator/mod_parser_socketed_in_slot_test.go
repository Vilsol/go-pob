package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSkillsSocketedInSlotAreSupported pins ModParser.lua:3374, `["skills socketed in your (.+) are supported by level (%d+) (.+)"]` -> extraSupport(...).
func TestSkillsSocketedInSlotAreSupported(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("Socketed Gems are Supported by Level 20 Faster Attacks", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the sibling `socketed gems are supported by` row does not parse; " +
			"this test cannot attribute a failure to the missing slot row")
	}

	const line = "Skills Socketed in your Helmet are Supported by level 20 Cold Penetration"
	e := ParseMod(line, false)
	if e == nil {
		t.Fatalf("no entry: %s", line)
	}
	if extra := strings.TrimSpace(modExtra(e)); extra != "" {
		t.Fatalf("leftover %q (line is DROPPED WHOLE): %s", extra, line)
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
		t.Fatalf("no ExtraSupport produced: %s", line)
	}
	if found.Level != 20 {
		t.Errorf("ExtraSupport level = %d, want 20", found.Level)
	}
	var slot string
	for _, m := range e.ModList {
		for _, tg := range m.Tags() {
			if si, ok := tg.(*mod.SocketedInTag); ok {
				slot = si.SlotName
			}
		}
	}
	if slot != "Helmet" {
		t.Errorf("SocketedIn slot = %q, want %q", slot, "Helmet")
	}
}
