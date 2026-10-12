package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

// TestManaRegenPercentDividedBy100 isolates the ManaRegenPercent contribution to mana regeneration.
func TestManaRegenPercentDividedBy100(t *testing.T) {
	modDB := moddb.NewModDB()
	modDB.AddMod(mod.NewFloat("ManaRegen", mod.TypeBase, 10).Source(mod.Source("Test")))
	modDB.AddMod(mod.NewFloat("ManaRegenPercent", mod.TypeBase, 5).Source(mod.Source("Test")))

	actor := &Actor{
		ModDB:  modDB,
		Output: map[string]float64{"Mana": 200},
		Enemy:  &Actor{ModDB: moddb.NewModDB()},
	}
	env := &Environment{
		Build:  &pob.PathOfBuilding{},
		Player: actor,
	}

	CalculateDefence(env, actor)

	want := modDB.Sum(mod.TypeBase, nil, "ManaRegen") + actor.Output["Mana"]*modDB.Sum(mod.TypeBase, nil, "ManaRegenPercent")/100
	testza.AssertEqual(t, want, actor.Output["ManaRegen"])
	testza.AssertEqual(t, 20.0, actor.Output["ManaRegen"])
}
