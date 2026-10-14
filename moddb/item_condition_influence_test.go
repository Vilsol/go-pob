package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestItemConditionShaperElderCond: ModStore.lua:688-697: shaperCond / elderCond match the slot's item.shaper / item.elder (ModParser.lua:1593-1594).
func TestItemConditionShaperElderCond(t *testing.T) {
	build := func(tag *mod.ItemConditionTag, rings map[string]fakeEquippedItem) float64 {
		db := NewModDB()
		db.Actor = &linkedActor{modDB: db, links: map[string]*linkedActor{}, equipped: rings}
		db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 70).Tag(tag))
		return db.Sum(mod.TypeIncrease, nil, "Damage")
	}
	shaper := fakeEquippedItem{name: "Shaper Ring", shaper: true}
	elder := fakeEquippedItem{name: "Elder Ring", elder: true}
	plain := fakeEquippedItem{name: "Plain Ring"}

	testza.AssertEqual(t, 70.0, build(mod.ItemCondition("Ring 2").Shaper(), map[string]fakeEquippedItem{"Ring 1": elder, "Ring 2": shaper}))
	testza.AssertEqual(t, 0.0, build(mod.ItemCondition("Ring 2").Shaper(), map[string]fakeEquippedItem{"Ring 1": shaper, "Ring 2": elder}))
	testza.AssertEqual(t, 0.0, build(mod.ItemCondition("Ring 2").Shaper(), map[string]fakeEquippedItem{"Ring 1": shaper}))
	testza.AssertEqual(t, 70.0, build(mod.ItemCondition("Ring 1").Elder(), map[string]fakeEquippedItem{"Ring 1": elder, "Ring 2": plain}))
	testza.AssertEqual(t, 0.0, build(mod.ItemCondition("Ring 1").Elder(), map[string]fakeEquippedItem{"Ring 1": plain, "Ring 2": elder}))
	testza.AssertEqual(t, 70.0, build(mod.ItemCondition("Ring 1").Elder().Negate(true), map[string]fakeEquippedItem{"Ring 1": plain}))
}
