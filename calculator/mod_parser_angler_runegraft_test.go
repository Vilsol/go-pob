package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAnglerRunegraftRow pins ModParser.lua:3553 (Runegraft of the Angler).
func TestAnglerRunegraftRow(t *testing.T) {
	entry := ParseMod("All hits are Critical Strikes while holding a Fishing Rod", false)
	if strings.TrimSpace(entry.Extra) != "" || len(entry.ModList) != 1 {
		t.Fatalf("extra %q, %d mods", entry.Extra, len(entry.ModList))
	}
	m := entry.ModList[0]
	if m.Name() != "CritChance" || m.Type() != mod.TypeOverride || m.Value().Float() != 100 {
		t.Errorf("got %s %s %v", m.Name(), m.Type(), m.Value().Float())
	}
	if tags := m.Tags(); len(tags) != 1 {
		t.Errorf("tags %#v, want one Condition:UsingFishing", tags)
	} else if c, ok := tags[0].(*mod.ConditionTag); !ok || len(c.VarList) != 1 || c.VarList[0] != "UsingFishing" {
		t.Errorf("tag %#v, want Condition:UsingFishing", tags[0])
	}
}
