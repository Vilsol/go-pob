package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

// TestLifeRegenRecoversEnergyShieldIsNotGatedOnPositiveRegen pins CalcDefence.lua:663.
//
//	if output.LifeRegen and modDB:Flag(nil, "LifeRegenerationRecoversEnergyShield") and output.EnergyShield > 0 then
//		modDB:NewMod("EnergyShieldRecovery", "BASE", lifeBase * ..., "Life Regeneration Recovers Energy Shield")
//	end
func TestLifeRegenRecoversEnergyShieldIsNotGatedOnPositiveRegen(t *testing.T) {
	modDB := moddb.NewModDB()
	modDB.AddMod(mod.NewFloat("LifeRegen", mod.TypeBase, -5).Source(mod.Source("Test")))
	modDB.AddMod(mod.NewFloat("EnergyShield", mod.TypeBase, 100).Source(mod.Source("Test")))
	modDB.AddMod(mod.NewFlag("LifeRegenerationRecoversEnergyShield", true).Source(mod.Source("Test")))

	actor := &Actor{
		ModDB:  modDB,
		Output: map[string]float64{"Life": 100},
		Enemy:  &Actor{ModDB: moddb.NewModDB()},
	}
	env := &Environment{
		Build:  &pob.PathOfBuilding{},
		Player: actor,
	}

	CalculateDefence(env, actor)

	testza.AssertTrue(t, actor.Output["EnergyShield"] > 0,
		"the test's own premise: :663's third conjunct needs a nonzero energy shield pool, "+
			"but EnergyShield came out %v", actor.Output["EnergyShield"])

	testza.AssertEqual(t, -5.0, modDB.Sum(mod.TypeBase, nil, "EnergyShieldRecovery"),
		"CalcDefence.lua:663 adds the mod whenever the flag is set and EnergyShield > 0; "+
			"the `output.LifeRegen` conjunct is always true")
}
