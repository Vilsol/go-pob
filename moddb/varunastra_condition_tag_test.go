package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

type allOneHActor struct {
	added map[string]bool
}

func (a *allOneHActor) GetOutput(string) (float64, bool) { return 0, false }

func (a *allOneHActor) AddedUsingCondition(v string) (bool, bool) {
	val, ok := a.added[v]
	return val, ok
}

// TestConditionTagVarunastraArm pins ModStore.lua:592-613.
func TestConditionTagVarunastraArm(t *testing.T) {
	build := func(actor Actor, conds map[string]bool) *ModDB {
		db := NewModDB()
		db.Actor = actor
		for k, v := range conds {
			db.Conditions[k] = v
		}
		db.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 20).
			Tag(mod.Condition("UsingSword").Neg(true)))
		return db
	}

	t.Run("added-by-varunastra-still-applies", func(t *testing.T) {
		db := build(&allOneHActor{added: map[string]bool{"UsingSword": true}},
			map[string]bool{"UsingSword": true})
		testza.AssertEqual(t, 20.0, db.Sum(mod.TypeIncrease, nil, "Speed"),
			"a UsingSword that only exists because Varunastra added it must not veto a "+
				"negated tag")
	})

	t.Run("genuinely-using-a-sword-vetoes", func(t *testing.T) {
		db := build(&allOneHActor{added: map[string]bool{"UsingSword": false}},
			map[string]bool{"UsingSword": true})
		testza.AssertEqual(t, 0.0, db.Sum(mod.TypeIncrease, nil, "Speed"),
			"a character wielding a real sword alongside the Varunastra must still fail a "+
				"negated UsingSword tag")
	})

	t.Run("no-all-one-handed-weapon-uses-the-ordinary-arm", func(t *testing.T) {
		db := build(&allOneHActor{}, map[string]bool{"UsingSword": true})
		testza.AssertEqual(t, 0.0, db.Sum(mod.TypeIncrease, nil, "Speed"),
			"with no all-one-handed weapon the arm must not fire at all")
	})

	t.Run("positive-tag-never-takes-the-arm", func(t *testing.T) {
		db := NewModDB()
		db.Actor = &allOneHActor{added: map[string]bool{"UsingSword": true}}
		db.Conditions["UsingSword"] = true
		db.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 20).Tag(mod.Condition("UsingSword")))
		testza.AssertEqual(t, 20.0, db.Sum(mod.TypeIncrease, nil, "Speed"),
			"the Varunastra arm is gated on tag.neg and must leave a positive tag alone")
	})
}
