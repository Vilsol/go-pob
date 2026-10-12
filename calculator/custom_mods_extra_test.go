package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestCustomModsDropLinesWithLeftoverExtra pins ConfigTab.lua:1116 and :1135.
func TestCustomModsDropLinesWithLeftoverExtra(t *testing.T) {
	requireGameData(t, "3.29")

	apply := func(text string) int {
		list := moddb.NewModList()
		configurations["customMods"](text, list, moddb.NewModList())
		return len(list.List(nil, "ExtraAura"))
	}

	const line = "You and nearby allies gain 328 additional Energy Shield"

	if got := apply(line); got != 1 {
		t.Fatalf("CONTROL FAILED: %q produced %d ExtraAura mods, want 1 — this test cannot "+
			"distinguish a working guard from a line the parser never understood", line, got)
	}
	if got := apply(line + "XXXdisabled"); got != 0 {
		t.Errorf("a line with leftover Extra produced %d ExtraAura mods, want 0 "+
			"(ConfigTab.lua:1116 requires `mods and not extra`)", got)
	}
}
