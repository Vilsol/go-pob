package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestGetConditionStoredFalseDoesNotVetoAFlagMod: ModStore.lua:214 is one `or` chain.
//
//	return self.conditions[var] or (self.parent and self.parent:GetCondition(var, cfg, true))
//	       or (not noMod and self:Flag(cfg, conditionName[var]))
func TestGetConditionStoredFalseDoesNotVetoAFlagMod(t *testing.T) {
	db := NewModDB()
	db.Conditions["UsingFlask"] = false
	db.AddMod(mod.NewFlag("Condition:UsingFlask", true).Source("Config"))

	cond, _ := db.GetCondition("UsingFlask", nil, false)
	testza.AssertTrue(t, cond, "a stored false must fall through to the Condition: flag mod")

	db.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 20).Tag(mod.Condition("UsingFlask")))
	testza.AssertEqual(t, 20.0, db.Sum(mod.TypeIncrease, nil, "Speed"))
}

// TestGetConditionParentFalseDoesNotVetoAFlagMod: The same fall-through applies to the PARENT hop.
func TestGetConditionParentFalseDoesNotVetoAFlagMod(t *testing.T) {
	parent := NewModDB()
	parent.Conditions["UsingFlask"] = false

	list := NewModList()
	list.Parent = parent
	list.AddMod(mod.NewFlag("Condition:UsingFlask", true).Source("Config"))
	list.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 20).Tag(mod.Condition("UsingFlask")))

	cond, _ := list.GetCondition("UsingFlask", nil, false)
	testza.AssertTrue(t, cond, "a parent's stored false must fall through to the child's flag mod")
	testza.AssertEqual(t, 20.0, list.Sum(mod.TypeIncrease, nil, "Speed"))
}

// TestGetConditionNoModSkipsTheFlagLookup: noMod still skips the flag hop.
func TestGetConditionNoModSkipsTheFlagLookup(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFlag("Condition:UsingFlask", true).Source("Config"))

	cond, _ := db.GetCondition("UsingFlask", nil, true)
	testza.AssertFalse(t, cond, "noMod must not consult Condition: flag mods")
}

// TestGetConditionStoredTrueAndUnknown: A stored TRUE still answers, and a store that knows nothing still answers false.
func TestGetConditionStoredTrueAndUnknown(t *testing.T) {
	db := NewModDB()
	db.Conditions["Combat"] = true

	cond, _ := db.GetCondition("Combat", nil, false)
	testza.AssertTrue(t, cond)

	cond, _ = db.GetCondition("NeverSet", nil, false)
	testza.AssertFalse(t, cond)
}
