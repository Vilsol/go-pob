package calculator

import (
	"testing"
)

// TestSnipeSocketedRowMatchesAndEmitsNothing: ModParser.lua:3212-3213 carries this row as an EMPTY table.
func TestSnipeSocketedRowMatchesAndEmitsNothing(t *testing.T) {
	const line = "Socketed Non-Channelling Bow Skills are Triggered by Snipe"

	e := ParseMod(line, false)
	if e == nil {
		t.Fatalf("ParseMod(%q) returned nil — the row must still MATCH; without it the line is "+
			"unparsed and processModLine discards it entirely", line)
	}
	if e.Extra != "" {
		t.Errorf("leftover Extra %q — the line no longer parses whole", e.Extra)
	}
	if len(e.ModList) != 0 {
		t.Errorf("ParseMod(%q) emitted %d mods, want 0: PoB's row is an empty table and "+
			"TriggeredBySnipe comes from ChannelledSnipeSupport's own mods, so anything emitted "+
			"here is a duplicate producer", line, len(e.ModList))
	}
}
