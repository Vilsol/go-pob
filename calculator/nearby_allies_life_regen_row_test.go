package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestNearbyAlliesLifeRegenRowIsAnExtraAura pins `ModParser.lua:4168`, which this tree got wrong in TWO independent ways at once.
//
//	["and nearby allies regenerate (%d+) life per second"] = function(num)
func TestNearbyAlliesLifeRegenRowIsAnExtraAura(t *testing.T) {
	const line = "and nearby Allies Regenerate 400 Life per second"

	e := ParseMod(line, false)
	if e.Extra != "" {
		t.Fatalf("unparsed remainder %q — the line must parse WHOLE: %s", e.Extra, line)
	}
	if len(e.ModList) != 1 {
		t.Fatalf("got %d mods, want 1: %s", len(e.ModList), line)
	}

	m := e.ModList[0]

	if m.Name() != "ExtraAura" {
		t.Errorf("mod name %q, want %q — a bare LifeRegen reaches only the player, so the "+
			"\"and nearby allies\" half of the line does nothing", m.Name(), "ExtraAura")
	}
	if m.Type() != mod.TypeList {
		t.Errorf("mod type %q, want %q", m.Type(), mod.TypeList)
	}

	if got := len(m.Tags()); got != 1 {
		t.Fatalf("got %d tags, want 1 — a tag-less row applies UNCONDITIONALLY: %v", got, m.Tags())
	}

	ct, ok := m.Tags()[0].(*mod.ConditionTag)
	if !ok {
		t.Fatalf("tag 0 is %T, want *mod.ConditionTag", m.Tags()[0])
	}

	if len(ct.VarList) != 1 || ct.VarList[0] != "KilledPoisonedLast2Seconds" {
		t.Errorf("condition var %v, want [KilledPoisonedLast2Seconds] — the misspelling "+
			"`KilledPosionedLast2Seconds` has ZERO occurrences in the pin, and configs.go now sets "+
			"the correct spelling, so a misspelled consumer never matches", ct.VarList)
	}
}

// TestNearbyAlliesRowDoesNotSwallowItsNeighbour is the control, and it is not commentary.
func TestNearbyAlliesRowDoesNotSwallowItsNeighbour(t *testing.T) {
	for _, c := range []struct {
		line string
		name string
		tags int
	}{
		{"Regenerate 50 Life over 1 second for each Spell you Cast", "LifeRegen", 1},
		{"10% increased Life Regeneration rate", "LifeRegen", 0},
	} {
		e := ParseMod(c.line, false)
		if e.Extra != "" {
			t.Errorf("unparsed remainder %q: %s", e.Extra, c.line)
			continue
		}
		if len(e.ModList) != 1 {
			t.Errorf("got %d mods, want 1: %s", len(e.ModList), c.line)
			continue
		}
		m := e.ModList[0]
		if m.Name() != c.name {
			t.Errorf("mod name %q, want %q — the allies row may have swallowed this line: %s",
				m.Name(), c.name, c.line)
		}
		if got := len(m.Tags()); got != c.tags {
			t.Errorf("got %d tags, want %d: %s", got, c.tags, c.line)
		}
	}
}
