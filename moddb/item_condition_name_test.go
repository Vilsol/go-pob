package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestItemConditionNameCondGatesOnTheEquippedItemsName pins ModStore.lua:698-702, the one arm of the ItemCondition evaluator this tree never carried.
func TestItemConditionNameCondGatesOnTheEquippedItemsName(t *testing.T) {
	withWeapon := func(name string) *ModDB {
		db := NewModDB()
		db.Actor = &linkedActor{
			modDB:    db,
			links:    map[string]*linkedActor{},
			equipped: map[string]fakeEquippedItem{"Weapon 1": {name: name}},
		}
		db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).
			Tag(mod.ItemCondition("Weapon 1").Name("The Iron Mass, Gladius")))
		return db
	}

	testza.AssertEqual(t, 40.0, withWeapon("The Iron Mass, Gladius").Sum(mod.TypeIncrease, nil, "Damage"))

	testza.AssertEqual(t, 40.0, withWeapon("the iron mass, gladius").Sum(mod.TypeIncrease, nil, "Damage"))

	testza.AssertEqual(t, 0.0, withWeapon("Starforge, Infernal Sword").Sum(mod.TypeIncrease, nil, "Damage"))

	testza.AssertEqual(t, 0.0, withWeapon("The Iron Mass").Sum(mod.TypeIncrease, nil, "Damage"))
}
