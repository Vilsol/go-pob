package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// Expected mods are the pinned PoB's live modLib.parseMod output for ModParser.lua:3778-3781.
func TestShockYourselfWhenYouFocusRow(t *testing.T) {
	line := "Shock yourself for 6 Seconds when you Focus"
	entry := ParseMod(line, false)
	if strings.TrimSpace(entry.Extra) != "" || len(entry.ModList) != 2 {
		t.Fatalf("%q: extra %q, %d mods, want 2", line, entry.Extra, len(entry.ModList))
	}
	for i, w := range []struct {
		name  string
		typ   mod.Type
		value float64
		flag  bool
	}{
		{"ShockBase", mod.TypeBase, 15, false},
		{"Condition:Shocked", mod.TypeFlag, 0, true},
	} {
		m := entry.ModList[i]
		if m.Name() != w.name || m.Type() != w.typ || m.Value().Float() != w.value || m.Value().Flag() != w.flag || m.Flags() != 0 || m.KeywordFlags() != 0 {
			t.Errorf("mod %d: got %s %s %v f%d k%d, want %s %s %v", i, m.Name(), m.Type(), m.Value().Float(), m.Flags(), m.KeywordFlags(), w.name, w.typ, w.value)
		}
		tags := m.Tags()
		if len(tags) != 1 {
			t.Errorf("mod %d: %d tags, want 1", i, len(tags))
			continue
		}
		if c, ok := tags[0].(*mod.ConditionTag); !ok || strings.Join(c.VarList, ",") != "Focused" || c.Negative {
			t.Errorf("mod %d: tags %#v, want Condition Focused", i, tags)
		}
	}
}
