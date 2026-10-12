package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestParseRyslathaLifeFlaskCharges: ModParser.lua:5695-5698 (Pantheon: Soul of Ryslatha support).
func TestParseRyslathaLifeFlaskCharges(t *testing.T) {
	requireGameData(t, "3.29")
	entry := ParseMod("Life Flasks gain 3 Charges every 3 seconds if you haven't used a Life Flask Recently", false)
	if entry == nil || entry.Extra != "" || len(entry.ModList) != 1 {
		t.Fatalf("parse: %+v", entry)
	}
	m := entry.ModList[0]
	if m.Name() != "LifeFlaskChargesGenerated" || m.Type() != mod.TypeBase || m.Value().Float() != 1 {
		t.Errorf("mod %s %s %v, PoB LifeFlaskChargesGenerated BASE 1", m.Name(), m.Type(), m.Value())
	}
	tags := m.Tags()
	if len(tags) != 1 {
		t.Fatalf("%d tags, PoB 1", len(tags))
	}
	if c, ok := tags[0].(*mod.ConditionTag); !ok || !c.Negative || len(c.VarList) != 1 || c.VarList[0] != "UsingLifeFlask" {
		t.Errorf("tag %+v, PoB Condition not UsingLifeFlask", tags[0])
	}
}
