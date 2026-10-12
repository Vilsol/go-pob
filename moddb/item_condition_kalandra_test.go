package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestItemConditionKalandraRedirect pins ModStore.lua:658-659.
func TestItemConditionKalandraRedirect(t *testing.T) {
	const kalandra = "Kalandra's Touch, Ring"
	build := func(tag *mod.ItemConditionTag, rings map[string]fakeEquippedItem) *ModDB {
		db := NewModDB()
		db.Actor = &linkedActor{modDB: db, links: map[string]*linkedActor{}, equipped: rings}
		db.AddMod(mod.NewFloat("EnergyShield", mod.TypeIncrease, 20).Tag(tag))
		return db
	}
	evasion := fakeEquippedItem{modifiers: []string{"+125 to Evasion Rating"}, name: "Honour Loop, Two-Stone Ring"}
	plain := fakeEquippedItem{modifiers: []string{"+44 to maximum Energy Shield"}, name: "Doom Band, Two-Stone Ring"}
	kal := fakeEquippedItem{modifiers: []string{"Reflects opposite Ring"}, name: kalandra}

	both := func() *mod.ItemConditionTag { return mod.ItemCondition("Ring").Search("evasion").WithBothSlots() }

	testza.AssertEqual(t, 20.0, build(both(), map[string]fakeEquippedItem{"Ring 1": evasion, "Ring 2": kal}).
		Sum(mod.TypeIncrease, nil, "EnergyShield"))
	testza.AssertEqual(t, 20.0, build(both(), map[string]fakeEquippedItem{"Ring 1": kal, "Ring 2": evasion}).
		Sum(mod.TypeIncrease, nil, "EnergyShield"))
	testza.AssertEqual(t, 0.0, build(both(), map[string]fakeEquippedItem{"Ring 1": plain, "Ring 2": kal}).
		Sum(mod.TypeIncrease, nil, "EnergyShield"))
	testza.AssertEqual(t, 0.0, build(both(), map[string]fakeEquippedItem{"Ring 1": evasion, "Ring 2": plain}).
		Sum(mod.TypeIncrease, nil, "EnergyShield"))

	single := func() *mod.ItemConditionTag { return mod.ItemCondition("Ring 2").Search("evasion") }
	testza.AssertEqual(t, 20.0, build(single(), map[string]fakeEquippedItem{"Ring 1": evasion, "Ring 2": kal}).
		Sum(mod.TypeIncrease, nil, "EnergyShield"))
	testza.AssertEqual(t, 0.0, build(single(), map[string]fakeEquippedItem{"Ring 1": plain, "Ring 2": kal}).
		Sum(mod.TypeIncrease, nil, "EnergyShield"))
	testza.AssertEqual(t, 0.0, build(single(), map[string]fakeEquippedItem{"Ring 2": kal}).
		Sum(mod.TypeIncrease, nil, "EnergyShield"))
}
