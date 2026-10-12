package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestHexMasterParsesAsAKeystone is the assertion that actually goes RED for the missing row.
func TestHexMasterParsesAsAKeystone(t *testing.T) {
	for _, name := range []string{
		"Hex Master",
		"Eldritch Battery", "Elemental Overload", "Magebane", "Point Blank", "Unwavering Stance",
	} {
		entry := ParseMod(name, false)
		if entry == nil || entry.ModList == nil || len(entry.ModList) != 2 {
			got := 0
			extra := ""
			if entry != nil {
				got = len(entry.ModList)
				extra = entry.Extra
			}
			t.Errorf("%q: got %d mods (extra %q), want Keystone/LIST + Condition:Have<Name>. "+
				"ModParser.lua:5878-5879 registers every data.keystones name as its own "+
				"specialModList pattern; a name missing from data.Keystones never gets registered.",
				name, got, extra)
			continue
		}
		if m := entry.ModList[0]; m.Name() != "Keystone" || m.Type() != mod.TypeList {
			t.Errorf("%q: got %s/%s, want Keystone/LIST", name, m.Name(), m.Type())
		}
		if m := entry.ModList[1]; m.Name() != "Condition:Have"+keystoneConditionSuffix(name) {
			t.Errorf("%q: second mod is %q, want %q", name, m.Name(),
				"Condition:Have"+keystoneConditionSuffix(name))
		}
	}
}
