package calculator2

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// ModStore's ItemCondition tag reads the actor's itemList: a rarity condition and a modifier search, with
// the body armour "life" search skipping the lines data.itemTagSpecialExclusionPattern lists.
func TestActorItemConditions(t *testing.T) {
	actor := &Actor{ModDB: moddb.NewModDB(), ItemList: map[string]*Item{
		"Body Armour": {Rarity: "NORMAL", ExplicitModLines: []*ModLine{{Line: "+20 to maximum Life when on Low Life"}}},
		"Helmet":      {Rarity: "RARE", ExplicitModLines: []*ModLine{{Line: "+30 to maximum Life"}}},
	}}
	actor.ModDB.Actor = actor
	actor.ModDB.AddMod(mod.NewFloat("Armour", mod.TypeBase, 10).Tag(mod.ItemCondition("Body Armour").Rarity("NORMAL")))
	actor.ModDB.AddMod(mod.NewFloat("Armour", mod.TypeBase, 100).Tag(mod.ItemCondition("Helmet").Rarity("NORMAL")))
	actor.ModDB.AddMod(mod.NewFloat("Evasion", mod.TypeBase, 1).Tag(mod.ItemCondition("Body Armour").Search("Life")))
	actor.ModDB.AddMod(mod.NewFloat("Evasion", mod.TypeBase, 10).Tag(mod.ItemCondition("Helmet").Search("Life")))

	testza.AssertEqual(t, 10.0, actor.ModDB.Sum(mod.TypeBase, nil, "Armour"))
	testza.AssertEqual(t, 10.0, actor.ModDB.Sum(mod.TypeBase, nil, "Evasion"))
}
