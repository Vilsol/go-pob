package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

func perStatMoreList() *ModList {
	list := NewModList()
	list.AddMod(mod.NewFloat("Damage", mod.TypeMore, 10).Tag(mod.PerStat(5, "Dexterity")))
	return list
}

// TestPerStatActorlessStore exercises ModList.More -> evalPerStatTag -> getStat on a store with no actor.
func TestPerStatActorlessStore(t *testing.T) {
	list := perStatMoreList()
	testza.AssertNil(t, list.GetActor(), "standalone ModList must have no actor")

	testza.AssertEqual(t, 1.0, list.More(nil, "Damage"))

	cfg := &ListCfg{SkillStats: map[string]float64{"Dexterity": 100}}
	testza.AssertEqual(t, 3.0, list.More(cfg, "Damage"))
}

// TestPerStatInheritsParentActor pins the actor wiring.
func TestPerStatInheritsParentActor(t *testing.T) {
	parent := NewModDB()
	parent.Actor = stubActor{outputs: map[string]float64{"Dexterity": 100}}

	list := perStatMoreList()
	list.Parent = parent

	testza.AssertNotNil(t, list.GetActor(), "child list must inherit the parent's actor")
	testza.AssertEqual(t, 3.0, list.More(nil, "Damage"))

	child := perStatMoreList()
	child.Parent = list
	testza.AssertEqual(t, 9.0, child.More(nil, "Damage"))
}
