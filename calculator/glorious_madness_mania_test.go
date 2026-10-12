package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestGloriousMadnessManiaParses pins ModParser.lua:3064-3067 (Delirious bloodline).
func TestGloriousMadnessManiaParses(t *testing.T) {
	e := ParseMod("While affected by Glorious Madness, inflict Mania on nearby Enemies every second", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 2 {
		t.Fatalf("ParseMod = %v, want 2 mods and no Extra", e)
	}
	if m := e.ModList[0]; m.Name() != "Condition:CanInflictMania" || m.Type() != mod.TypeFlag {
		t.Errorf("mod 0 = %s %s, want FLAG Condition:CanInflictMania", m.Type(), m.Name())
	}
	m := e.ModList[1]
	em, ok := m.Value().List().(mod.EnemyModifier)
	if m.Name() != "EnemyModifier" || !ok || em.Mod.Name() != "Condition:AfflictedByMania" {
		t.Errorf("mod 1 = %s %v, want EnemyModifier LIST { flag Condition:AfflictedByMania }", m.Name(), m.Value())
	}
	for i, m := range e.ModList {
		c, ok := m.Tags()[0].(*mod.ConditionTag)
		if len(m.Tags()) != 1 || !ok || len(c.VarList) != 1 || c.VarList[0] != "AffectedByGloriousMadness" {
			t.Errorf("mod %d tags = %v, want Condition AffectedByGloriousMadness", i, m.Tags())
		}
	}
}
