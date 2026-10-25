package calculator2

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// Expected mods are rows of PoB-PoE2's shipped src/Data/ModCache.lua.
func TestParseModMatchesPoBRows(t *testing.T) {
	type want struct {
		name  string
		typ   mod.Type
		value float64
		flags mod.MFlag
		extra string
	}
	cases := map[string]want{
		"+10% to Chaos Resistance":                               {"ChaosResist", mod.TypeBase, 10, 0, ""},
		"+65 to maximum Life":                                    {"Life", mod.TypeBase, 65, 0, ""},
		"+5% to Block chance":                                    {"BlockChance", mod.TypeBase, 5, 0, ""},
		"10% increased Thorns damage":                            {"Damage", mod.TypeIncrease, 10, ModFlagThorns, ""},
		"+100 to Deflection Rating per 50 missing Energy Shield": {"DeflectionRating", mod.TypeBase, 100, 0, "per 50 missing Energy Shield"},
	}
	for line, w := range cases {
		mods, extra := ParseMod(line)
		if len(mods) != 1 {
			t.Errorf("%q: %d mods, want 1", line, len(mods))
			continue
		}
		m := mods[0]
		if m.Name() != w.name || m.Type() != w.typ || m.Value().Float() != w.value || m.Flags() != w.flags || len(m.Tags()) != 0 {
			t.Errorf("%q: got %s %s %v f%d tags %d, want %s %s %v f%d", line, m.Name(), m.Type(), m.Value().Float(), m.Flags(), len(m.Tags()), w.name, w.typ, w.value, w.flags)
		}
		if strings.TrimSpace(extra) != w.extra {
			t.Errorf("%q: remainder %q, want %q", line, extra, w.extra)
		}
	}

	// PoB caches {nil, line}: no form matches a range text.
	if mods, extra := ParseMod("(2-3)% increased Movement Speed"); mods != nil || strings.TrimSpace(extra) != "(2-3)% increased Movement Speed" {
		t.Errorf("range text: got %d mods, remainder %q", len(mods), extra)
	}
}
