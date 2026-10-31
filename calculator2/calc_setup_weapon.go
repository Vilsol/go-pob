package calculator2

// unarmedWeaponData is data.unarmedWeaponData without its type ("None") and CritChance, which
// characterConstants supplies.
//
//pob2:src/Modules/Data.lua:626-636
var unarmedWeaponData = map[int]map[string]float64{
	0:  {"AttackRate": 1.65, "PhysicalMin": 2, "PhysicalMax": 6},
	1:  {"AttackRate": 1.65, "PhysicalMin": 2, "PhysicalMax": 5},
	2:  {"AttackRate": 1.65, "PhysicalMin": 2, "PhysicalMax": 5},
	6:  {"AttackRate": 1.65, "PhysicalMin": 2, "PhysicalMax": 8},
	7:  {"AttackRate": 1.65, "PhysicalMin": 2, "PhysicalMax": 5},
	8:  {"AttackRate": 1.65, "PhysicalMin": 2, "PhysicalMax": 5},
	9:  {"AttackRate": 1.65, "PhysicalMin": 2, "PhysicalMax": 6},
	10: {"AttackRate": 1.65, "PhysicalMin": 2, "PhysicalMax": 5},
	11: {"AttackRate": 1.65, "PhysicalMin": 2, "PhysicalMax": 6},
}

// weaponData is actor.weaponData<i>'s numbers as CalcSetup sets them: the equipped weapon's
// weaponData[i], else a copy of the class's unarmed data for the main hand, else an empty table.
// Its type and asThoughUsing are Actor.WeaponData<i>Type and WeaponData<i>AsThoughUsing.
// countsAsDualWielding is not carried: the item data has no flag for it.
//
//pob2:block:src/Modules/CalcSetup.lua:2120-2153
func weaponData(env *Environment, actor *Actor, i int) map[string]float64 {
	if actor.weaponDataSet {
		if i == 1 {
			return actor.WeaponData1
		}
		return actor.WeaponData2
	}
	if i == 1 {
		if item := actor.ItemList["Weapon 1"]; item != nil && item.WeaponData[1] != nil {
			return item.WeaponData[1]
		}
		out := unarmedWeaponDataFor(env)
		for k, v := range actor.WeaponData1Facebreaker {
			out[k] = v
		}
		return out
	}
	item := actor.ItemList["Weapon 2"]
	if item == nil {
		if actor.WeaponData2Type == "None" {
			return unarmedWeaponDataFor(env)
		}
		return map[string]float64{}
	}
	if item.WeaponData[2] != nil {
		return item.WeaponData[2]
	}
	return map[string]float64{}
}

func unarmedWeaponDataFor(env *Environment) map[string]float64 {
	out := map[string]float64{"CritChance": characterConstant("unarmed_base_critical_strike_chance") / 100}
	for k, v := range unarmedWeaponData[env.ClassID] {
		out[k] = v
	}
	return out
}
