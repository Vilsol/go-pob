package calculator2

import (
	"math"

	"github.com/Vilsol/go-pob/mod"
)

//pob2:src/Modules/CalcDefence.lua:74-129
func doActorLifeManaSpirit(actor *Actor) {
	modDB := actor.ModDB
	output := actor.Output
	condList := modDB.Conditions

	lowLifePerc := modDB.Sum(mod.TypeBase, nil, "LowLifePercentage")
	if lowLifePerc > 0 {
		output["LowLifePercentage"] = 100.0 * lowLifePerc
	} else {
		output["LowLifePercentage"] = 100.0 * miscConstant("LowPoolThreshold")
	}
	fullLifePerc := modDB.Sum(mod.TypeBase, nil, "FullLifePercentage")
	if fullLifePerc > 0 {
		output["FullLifePercentage"] = 100.0 * fullLifePerc
	} else {
		output["FullLifePercentage"] = 100.0
	}

	chaosInoculation := modDB.Flag(nil, "ChaosInoculation")

	for _, res := range []string{"Life", "Mana", "Spirit"} {
		base := modDB.Sum(mod.TypeBase, nil, res)
		extra := modDB.Sum(mod.TypeBase, nil, "Extra"+res)
		total := modDB.Sum(mod.TypeBase, nil, res+"Total")
		inc := modDB.Sum(mod.TypeIncrease, nil, res)
		more := modDB.More(nil, res)
		conv := math.Min(modDB.Sum(mod.TypeBase, nil, res+"ConvertToEnergyShield", res+"ConvertToArmour", res+"ConvertToEvasion"), 100)
		// output[res.."HasOverride"] is a boolean, which this float output table does not carry.
		if override := modDB.Override(nil, res); override != nil {
			output[res] = override.Float()
			continue
		}
		output[res] = math.Max(round((base*(1-conv/100)+extra)*(1+inc/100)*more+total), 1)
	}
	if chaosInoculation {
		output["Life"] = 1
		condList["FullLife"] = true
	}
	output["LowestOfMaximumLifeAndMaximumMana"] = math.Min(output["Life"], output["Mana"])
}
