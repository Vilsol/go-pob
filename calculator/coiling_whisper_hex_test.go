package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestTargetsAreUnaffectedByYourHexesParses pins ModParser.lua:4068.
func TestTargetsAreUnaffectedByYourHexesParses(t *testing.T) {
	const line = "Targets are Unaffected by your Hexes"

	entry := ParseMod(line, false)
	if entry.Extra != "" {
		t.Fatalf("ModParser.lua:4068 — %q left %q unparsed", line, entry.Extra)
	}
	if len(entry.ModList) != 1 {
		t.Fatalf("ModParser.lua:4068 — %q produced %d mods, want 1", line, len(entry.ModList))
	}

	m := entry.ModList[0]
	if m.Name() != "CurseEffect" {
		t.Fatalf("mod name = %s, want CurseEffect", m.Name())
	}
	if m.Type() != mod.TypeMore {
		t.Fatalf("mod type = %v, want MORE", m.Type())
	}
	if v := m.Value().Float(); v != -100 {
		t.Fatalf("mod value = %v, want -100", v)
	}
	if len(m.Tags()) != 1 {
		t.Fatalf("got %d tags, want 1 (SkillType Hex)", len(m.Tags()))
	}
	st, ok := m.Tags()[0].(*mod.SkillTypeTag)
	if !ok {
		t.Fatalf("tag is %#v, want a SkillType", m.Tags()[0])
	}
	if len(st.SkillTypeList) != 1 || st.SkillTypeList[0] != string(data.SkillTypeHex) {
		t.Fatalf("skillTypeList = %v, want [%s]", st.SkillTypeList, string(data.SkillTypeHex))
	}
	if st.Negative {
		t.Fatal("the SkillType tag is negated; upstream's is not, so this would zero everything EXCEPT hexes")
	}
}
