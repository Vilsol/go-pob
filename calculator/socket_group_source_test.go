package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestSocketGroupSourceNames: SkillsTab.lua:363-379: the explode group names each source.
func TestSocketGroupSourceNames(t *testing.T) {
	initItemsTabData(t)
	text, ok := ItemDBText(ItemDBUnique, "Inpulsa's Broken Heart, Sadist Garb")
	if !ok {
		t.Fatal("no Inpulsa's Broken Heart in the DB")
	}
	item := pob.Gem{SkillID: "EnemyExplode"}
	item.SetExplodeSource("Item:-1:Inpulsa's Broken Heart, Sadist Garb")
	node := pob.Gem{SkillID: "EnemyExplode"}
	node.SetExplodeSource("1731")
	tree := "Tree:1731"
	build := &pob.PathOfBuilding{}
	build.Items.Items = []pob.Item{{ID: 1, Text: text}}
	build.Skills.SkillSets = []pob.SkillSet{{Skills: []pob.Skill{
		{Source: "Explode", Gems: []pob.Gem{item, node}},
		{SourceAttr: &tree},
		{Source: "Item:1:Inpulsa's Broken Heart, Sadist Garb"},
	}}}
	want := []GroupSourceName{{"Inpulsa's Broken Heart, Sadist Garb", "UNIQUE"}, {"Hinekora, Death's Fury", "NORMAL"}}
	if got := SocketGroupSourceNames(build, 0, 0); !slices.Equal(got, want) {
		t.Errorf("explode group: %v, want %v", got, want)
	}
	if got := SocketGroupSourceNames(build, 0, 1); !slices.Equal(got, want[1:]) {
		t.Errorf("node group: %v", got)
	}
	if got := SocketGroupSourceNames(build, 0, 2); got != nil {
		t.Errorf("item group: %v", got)
	}
}
