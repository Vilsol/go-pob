package calculator

import "testing"

// TestParseModDistinguishesNoFormFromNoName covers the nil-vs-{} split ModParser.lua draws and this port collapsed.
func TestParseModDistinguishesNoFormFromNoName(t *testing.T) {
	requireGameData(t, "3.29")

	const noName = "+5 to Zzyzx Nonexistent Stat"
	const noForm = "Zzyzx nonexistent"

	if got := ParseMod(noForm, false); got.ModList != nil {
		t.Fatalf("control: %q returned a %d-mod list; ModParser.lua:6629 returns nil when no form matches", noForm, len(got.ModList))
	}

	entry := ParseMod(noName, false)
	if entry.ModList == nil {
		t.Errorf("%q returned nil; a BASE form matched and only the mod NAME did not, and "+
			"ModParser.lua:6810 returns an empty table for that", noName)
	} else if len(entry.ModList) != 0 {
		t.Errorf("%q returned %d mods, want an empty list", noName, len(entry.ModList))
	}
	if entry.Extra == "" {
		t.Errorf("%q reported no leftover text; the fixture no longer reaches the bail it targets", noName)
	}
}
