package pob

import (
	"slices"
	"testing"
)

func threeSkillSets() *PathOfBuilding {
	return &PathOfBuilding{Skills: Skills{ActiveSkillSet: 3, SkillSets: []SkillSet{
		{ID: 1, Title: title("Leveling")},
		{ID: 3, Title: title("Maps"), Skills: []Skill{{Label: "Main", Gems: []Gem{{NameSpec: "Fireball", Level: 20}}}}},
		{ID: 4},
	}}}
}

func skillSetIDs(b *PathOfBuilding) []int {
	ids := make([]int, 0, len(b.Skills.SkillSets))
	for _, s := range b.Skills.SkillSets {
		ids = append(ids, s.ID)
	}
	return ids
}

// TestNewSkillSetTakesTheLowestFreeID: SkillsTab:NewSkillSet takes the lowest unused id; the list appends it (SkillSetListControl.lua:48-50, 62-66).
func TestNewSkillSetTakesTheLowestFreeID(t *testing.T) {
	b := threeSkillSets()
	if id := b.NewSkillSet("Bossing"); id != 2 {
		t.Fatalf("new id %d", id)
	}
	if !slices.Equal(skillSetIDs(b), []int{1, 3, 4, 2}) || *b.Skills.SkillSets[3].Title != "Bossing" || len(b.Skills.SkillSets[3].Skills) != 0 {
		t.Fatalf("sets %v", skillSetIDs(b))
	}
	if b.Skills.ActiveSkillSet != 3 {
		t.Fatal("a new set changed the active one")
	}
}

// TestCopySkillSetIsDeep: Copy is a deep copy of every group and gem under a new id and the new name (SkillSetListControl.lua:14-32).
func TestCopySkillSetIsDeep(t *testing.T) {
	b := threeSkillSets()
	id := b.CopySkillSet(3, "Maps copy")
	if id != 2 || !slices.Equal(skillSetIDs(b), []int{1, 3, 4, 2}) {
		t.Fatalf("id %d sets %v", id, skillSetIDs(b))
	}
	c := b.Skills.SkillSets[3]
	if *c.Title != "Maps copy" || len(c.Skills) != 1 || c.Skills[0].Gems[0].NameSpec != "Fireball" {
		t.Fatalf("copy %+v", c)
	}
	c.Skills[0].Gems[0].Level = 1
	c.Skills[0].Label = "x"
	if b.Skills.SkillSets[1].Skills[0].Gems[0].Level != 20 || b.Skills.SkillSets[1].Skills[0].Label != "Main" {
		t.Fatal("the copy shares groups or gems with the original")
	}
}

func TestRenameSkillSet(t *testing.T) {
	b := threeSkillSets()
	b.RenameSkillSet(4, "Delve")
	if *b.Skills.SkillSets[2].Title != "Delve" || *b.Skills.SkillSets[0].Title != "Leveling" {
		t.Fatal("rename hit the wrong set")
	}
}

// TestDeleteSkillSet: OnSelDelete (SkillSetListControl.lua:99-110).
func TestDeleteSkillSet(t *testing.T) {
	b := threeSkillSets()
	b.DeleteSkillSet(3)
	if !slices.Equal(skillSetIDs(b), []int{1, 4}) || b.Skills.ActiveSkillSet != 1 {
		t.Fatalf("sets %v active %d", skillSetIDs(b), b.Skills.ActiveSkillSet)
	}
	b.DeleteSkillSet(1)
	if !slices.Equal(skillSetIDs(b), []int{4}) || b.Skills.ActiveSkillSet != 4 {
		t.Fatalf("sets %v active %d", skillSetIDs(b), b.Skills.ActiveSkillSet)
	}
	b.DeleteSkillSet(4)
	if !slices.Equal(skillSetIDs(b), []int{4}) {
		t.Fatal("the last set was deleted")
	}
	c := threeSkillSets()
	c.DeleteSkillSet(4)
	if !slices.Equal(skillSetIDs(c), []int{1, 3}) || c.Skills.ActiveSkillSet != 3 {
		t.Fatalf("deleting another set moved the active one: %v %d", skillSetIDs(c), c.Skills.ActiveSkillSet)
	}
}

// TestGuessMainSocketGroup: ImportTab.lua:1900-1910: the active set's group with the most gems, the first on a tie, 1-based; 1 when empty.
func TestGuessMainSocketGroup(t *testing.T) {
	b := &PathOfBuilding{Skills: Skills{ActiveSkillSet: 1, SkillSets: []SkillSet{{ID: 1, Skills: []Skill{
		{Gems: []Gem{{}, {}}}, {Gems: []Gem{{}, {}, {}, {}}}, {Gems: []Gem{{}, {}, {}, {}}}, {},
	}}}}}
	if got := b.GuessMainSocketGroup(); got != 2 {
		t.Errorf("guess %d, want 2", got)
	}
	if got := (&PathOfBuilding{}).GuessMainSocketGroup(); got != 1 {
		t.Errorf("no groups: %d", got)
	}
}
