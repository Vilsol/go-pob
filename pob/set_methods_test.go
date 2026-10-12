package pob

import (
	"testing"
)

func twoSkillSets() *PathOfBuilding {
	return &PathOfBuilding{Skills: Skills{
		ActiveSkillSet: 3,
		SkillSets: []SkillSet{
			{ID: 3, Skills: []Skill{{Label: "final"}}},
			{ID: 2, Skills: []Skill{{Label: "swap a"}, {Label: "swap b"}}},
		},
	}}
}

func TestSetActiveSkillSetTakesAnID(t *testing.T) {
	b := twoSkillSets()
	b.SetActiveSkillSet(2)
	if b.Skills.ActiveSkillSet != 2 {
		t.Fatalf("ActiveSkillSet = %d, want 2", b.Skills.ActiveSkillSet)
	}
	if got := b.ActiveSkillSetIndex(); got != 1 {
		t.Fatalf("ActiveSkillSetIndex = %d, want 1 (the set with id 2)", got)
	}
}

// TestSetActiveSkillSetUnknownIDFallsBackToTheFirstSet: SkillsTab.lua:1462-1464: an id with no set falls back to the first set in order, not to id 1.
func TestSetActiveSkillSetUnknownIDFallsBackToTheFirstSet(t *testing.T) {
	b := twoSkillSets()
	b.Skills.ActiveSkillSet = 2
	b.SetActiveSkillSet(7)
	if b.Skills.ActiveSkillSet != 3 {
		t.Fatalf("ActiveSkillSet = %d, want 3 (skillSetOrderList[1])", b.Skills.ActiveSkillSet)
	}
}

// TestSetActiveSkillSetCreatesSetOne: SkillsTab.lua:1453-1456: a build with no skill set gets set 1.
func TestSetActiveSkillSetCreatesSetOne(t *testing.T) {
	b := &PathOfBuilding{}
	b.SetActiveSkillSet(4)
	if len(b.Skills.SkillSets) != 1 || b.Skills.SkillSets[0].ID != 1 || b.Skills.ActiveSkillSet != 1 {
		t.Fatalf("got %+v, want one set with id 1, active", b.Skills)
	}
}

// TestSocketGroupMethodsActOnTheActiveSetByID: The socket-group structure methods act on the active set by id.
func TestSocketGroupMethodsActOnTheActiveSetByID(t *testing.T) {
	b := twoSkillSets()
	b.AddNewSocketGroup()
	if n := len(b.Skills.SkillSets[0].Skills); n != 2 {
		t.Fatalf("set 3 has %d groups after AddNewSocketGroup, want 2", n)
	}
	if n := len(b.Skills.SkillSets[1].Skills); n != 2 {
		t.Fatalf("set 2 has %d groups, want it untouched at 2", n)
	}
	b.DeleteSocketGroup(0)
	if n := len(b.Skills.SkillSets[0].Skills); n != 1 {
		t.Fatalf("set 3 has %d groups after DeleteSocketGroup, want 1", n)
	}
	b.DeleteAllSocketGroups()
	if n := len(b.Skills.SkillSets[0].Skills); n != 0 {
		t.Fatalf("set 3 has %d groups after DeleteAllSocketGroups, want 0", n)
	}
	if n := len(b.Skills.SkillSets[1].Skills); n != 2 {
		t.Fatalf("set 2 has %d groups, want it untouched at 2", n)
	}
}

// TestCalcsInputsRoundTrip: The Calcs tab's inputs (CalcsTab.lua:28-30 defaults, :40-44 / 87-91 / 120-125 writes) round-trip by name and type.
func TestCalcsInputsRoundTrip(t *testing.T) {
	b := &PathOfBuilding{}
	if b.CalcsInputString("misc_buffMode", "EFFECTIVE") != "EFFECTIVE" || b.CalcsInputNumber("skill_number", 1) != 1 || b.CalcsInputBool("showMinion") {
		t.Fatal("defaults")
	}
	b.SetCalcsInputString("misc_buffMode", "COMBAT")
	b.SetCalcsInputNumber("skill_number", 3)
	b.SetCalcsInputBool("showMinion", true)
	b.SetCalcsInputString("misc_buffMode", "BUFFED")
	if b.CalcsInputString("misc_buffMode", "") != "BUFFED" || b.CalcsInputNumber("skill_number", 1) != 3 || !b.CalcsInputBool("showMinion") || len(b.Calcs.Inputs) != 3 {
		t.Fatalf("inputs %+v", b.Calcs.Inputs)
	}
}
