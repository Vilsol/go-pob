package calculator

import (
	"fmt"
	"math"

	"github.com/Vilsol/go-pob/calculator/calclib"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

var resistTypeList = []string{
	"Fire",
	"Cold",
	"Lightning",
	"Chaos",
}

var isElemental = map[string]bool{
	"Fire":      true,
	"Cold":      true,
	"Lightning": true,
}

func CalcArmourReductionF(armour float64, raw float64) float64 {
	if armour == 0 && raw == 0 {
		return 0
	}
	return armour / (armour + raw*5) * 100
}

func CalcArmourReduction(armour float64, raw float64) float64 {
	return math.Round(CalcArmourReductionF(armour, raw))
}

func CalcHitChance(evasion float64, accuracy float64) float64 {
	if accuracy < 0 {
		return 5
	}
	rawChance := accuracy / (accuracy + math.Pow(evasion/5, 0.9)) * 125
	return math.Max(math.Min(math.Round(rawChance), 100), 5)
}

func CalculateDefence(environment *Environment, actor *Actor) {
	modDB := actor.ModDB

	// Action Speed
	actor.Output["ActionSpeedMod"] = CalcActionSpeedMod(actor)

	// Resistances
	actor.Output["DamageReductionMax"] = data.DamageReductionCap
	DamageReductionMax := actor.ModDB.Override(nil, "DamageReductionMax")
	if DamageReductionMax != nil {
		actor.Output["DamageReductionMax"] = DamageReductionMax.Float()
	}

	actor.Output["PhysicalResist"] = math.Min(math.Max(0, actor.ModDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction")), actor.Output["DamageReductionMax"])
	actor.Output["PhysicalResistWhenHit"] = math.Min(math.Max(0, actor.Output["PhysicalResist"]+actor.ModDB.Sum(mod.TypeBase, nil, "PhysicalDamageReductionWhenHit")), actor.Output["DamageReductionMax"])

	// Highest Maximum Elemental Resistance for Melding of the Flesh
	if modDB.Flag(nil, "ElementalResistMaxIsHighestResistMax") {
		highestResistMax := float64(0)
		highestResistMaxType := ""
		for _, elem := range resistTypeList {
			resistMax := utils.Or(modDB.Override(nil, elem+"ResistMax"), min(data.MaxResistCap, modDB.Sum(mod.TypeBase, nil, elem+"ResistMax", utils.Ternary(isElemental[elem], "ElementalResistMax", ""))))
			if resistMax > highestResistMax && isElemental[elem] {
				highestResistMax = resistMax
				highestResistMaxType = elem
			}
		}
		for _, elem := range resistTypeList {
			if isElemental[elem] {
				modDB.AddMod(mod.NewFloat(elem+"ResistMax", mod.TypeOverride, highestResistMax).Source(mod.Source(highestResistMaxType + " Melding of the Flesh")))
			}
		}
	}

	for _, elem := range resistTypeList {
		Min := float64(data.ResistFloor)
		Max := utils.Or(modDB.Override(nil, elem+"ResistMax"), min(data.MaxResistCap, modDB.Sum(mod.TypeBase, nil, elem+"ResistMax", utils.Ternary(isElemental[elem], "ElementalResistMax", ""))))
		totemMax := utils.Or(modDB.Override(nil, "Totem"+elem+"ResistMax"), min(data.MaxResistCap, modDB.Sum(mod.TypeBase, nil, "Totem"+elem+"ResistMax", utils.Ternary(isElemental[elem], "TotemElementalResistMax", ""))))
		total := modDB.Override(nil, elem+"Resist")
		totemTotal := modDB.Override(nil, "Totem"+elem+"Resist")
		if total == nil {
			base := modDB.Sum(mod.TypeBase, nil, elem+"Resist", utils.Ternary(isElemental[elem], "ElementalResist", ""))
			total = mod.NewModValueFloat(base * calclib.Mod(modDB, nil, elem+"Resist", utils.Ternary(isElemental[elem], "ElementalResist", "")))
		}
		if totemTotal == nil {
			base := modDB.Sum(mod.TypeBase, nil, "Totem"+elem+"Resist", utils.Ternary(isElemental[elem], "TotemElementalResist", ""))
			totemTotal = mod.NewModValueFloat(base * calclib.Mod(modDB, nil, "Totem"+elem+"Resist", utils.Ternary(isElemental[elem], "TotemElementalResist", "")))
		}
		final := max(min(total.Float(), Max), Min)
		totemFinal := max(min(totemTotal.Float(), totemMax), Min)
		actor.Output[elem+"Resist"] = final
		actor.Output[elem+"ResistTotal"] = total.Float()
		actor.Output[elem+"ResistOverCap"] = max(0, total.Float()-Max)
		actor.Output[elem+"ResistOver75"] = max(0, final-75)
		actor.Output["Missing"+elem+"Resist"] = max(0, totemMax-final)
		actor.Output["Totem"+elem+"Resist"] = totemFinal
		actor.Output["Totem"+elem+"ResistTotal"] = totemTotal.Float()
		actor.Output["Totem"+elem+"ResistOverCap"] = max(0, totemTotal.Float()-totemMax)
		actor.Output["MissingTotem"+elem+"Resist"] = max(0, totemMax-totemFinal)
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine(elem+"Resist",
				"Min: "+fmt.Sprint(Min)+"%",
				"Max: "+fmt.Sprint(Max)+"%",
				"Total: "+fmt.Sprint(total)+"%",
			)
			actor.Breakdown.AddLine("Totem"+elem+"Resist",
				"Min: "+fmt.Sprint(Min)+"%",
				"Max: "+fmt.Sprint(totemMax)+"%",
				"Total: "+fmt.Sprint(totemTotal)+"%",
			)
		}
	}

	// Block
	actor.Output["BlockChanceMax"] = modDB.Sum(mod.TypeBase, nil, "BlockChanceMax")
	actor.Output["BlockChanceOverCap"] = 0
	actor.Output["SpellBlockChanceOverCap"] = 0
	baseBlockChance := float64(0)

	if actor.ItemList["Weapon 2"] != nil && actor.ItemList["Weapon 2"].ArmourData != nil {
		baseBlockChance = baseBlockChance + actor.ItemList["Weapon 2"].ArmourData.BlockChance
	}
	if actor.ItemList["Weapon 3"] != nil && actor.ItemList["Weapon 3"].ArmourData != nil {
		baseBlockChance = baseBlockChance + actor.ItemList["Weapon 3"].ArmourData.BlockChance
	}
	actor.Output["ShieldBlockChance"] = baseBlockChance
	if modDB.Flag(nil, "MaxBlockIfNotBlockedRecently") {
		actor.Output["BlockChance"] = actor.Output["BlockChanceMax"]
	} else {
		totalBlockChance := (baseBlockChance + modDB.Sum(mod.TypeBase, nil, "BlockChance")) * calclib.Mod(modDB, nil, "BlockChance")
		actor.Output["BlockChance"] = min(totalBlockChance, actor.Output["BlockChanceMax"])
		actor.Output["BlockChanceOverCap"] = max(0, totalBlockChance-actor.Output["BlockChanceMax"])
	}
	actor.Output["ProjectileBlockChance"] = min(actor.Output["BlockChance"]+modDB.Sum(mod.TypeBase, nil, "ProjectileBlockChance")*calclib.Mod(modDB, nil, "BlockChance"), actor.Output["BlockChanceMax"])
	if modDB.Flag(nil, "SpellBlockChanceMaxIsBlockChanceMax") {
		actor.Output["SpellBlockChanceMax"] = actor.Output["BlockChanceMax"]
	} else {
		actor.Output["SpellBlockChanceMax"] = modDB.Sum(mod.TypeBase, nil, "SpellBlockChanceMax")
	}
	if modDB.Flag(nil, "SpellBlockChanceIsBlockChance") {
		actor.Output["SpellBlockChance"] = actor.Output["BlockChance"]
		actor.Output["SpellProjectileBlockChance"] = actor.Output["ProjectileBlockChance"]
		actor.Output["SpellBlockChanceOverCap"] = actor.Output["BlockChanceOverCap"]
	} else {
		totalSpellBlockChance := modDB.Sum(mod.TypeBase, nil, "SpellBlockChance") * calclib.Mod(modDB, nil, "SpellBlockChance")
		actor.Output["SpellBlockChance"] = min(totalSpellBlockChance, actor.Output["SpellBlockChanceMax"])
		actor.Output["SpellBlockChanceOverCap"] = max(0, totalSpellBlockChance-actor.Output["SpellBlockChanceMax"])
		actor.Output["SpellProjectileBlockChance"] = actor.Output["SpellBlockChance"]
	}
	if actor.Breakdown != nil {
		actor.Breakdown.AddLine("BlockChance",
			"Base: "+fmt.Sprint(baseBlockChance)+"%",
			"Max: "+fmt.Sprint(actor.Output["BlockChanceMax"])+"%",
			"Total: "+fmt.Sprint(actor.Output["BlockChance"]+actor.Output["BlockChanceOverCap"])+"%",
		)
		actor.Breakdown.AddLine("SpellBlockChance",
			"Max: "+fmt.Sprint(actor.Output["SpellBlockChanceMax"])+"%",
			"Total: "+fmt.Sprint(actor.Output["SpellBlockChance"]+actor.Output["SpellBlockChanceOverCap"])+"%",
		)
	}

	if modDB.Flag(nil, "CannotBlockAttacks") {
		actor.Output["BlockChance"] = 0
		actor.Output["ProjectileBlockChance"] = 0
	}
	if modDB.Flag(nil, "CannotBlockSpells") {
		actor.Output["SpellBlockChance"] = 0
		actor.Output["SpellProjectileBlockChance"] = 0
	}
	actor.Output["AverageBlockChance"] = (actor.Output["BlockChance"] + actor.Output["ProjectileBlockChance"] + actor.Output["SpellBlockChance"] + actor.Output["SpellProjectileBlockChance"]) / 4
	actor.Output["BlockEffect"] = max(100-modDB.Sum(mod.TypeBase, nil, "BlockEffect"), 0)
	if actor.Output["BlockEffect"] == 0 {
		actor.Output["BlockEffect"] = 100
	} else {
		actor.Output["ShowBlockEffect"] = 1
		actor.Output["DamageTakenOnBlock"] = 100 - actor.Output["BlockEffect"]
	}

	if modDB.Flag(nil, "ArmourAppliesToEnergyShieldRecharge") {
		// Armour to ES Recharge conversion from Armour and Energy Shield Mastery
		multiplier := utils.OrF(modDB.Max(nil, "ImprovedArmourAppliesToEnergyShieldRecharge"), 100) / 100
		for _, value := range modDB.Tabulate("INC", nil, "Armour", "ArmourAndEvasion", "Defences") {
			modDB.AddMod(value.Mod.Clone().SetName("EnergyShieldRecharge").SetValue(mod.NewModValueFloat(math.Floor(value.Mod.Value().Float() * multiplier))).Tag(calclib.GetConvertedModTags(value.Mod, multiplier, false)...))
		}
	}

	// Primary defences: Energy shield, evasion and armour
	{
		ironReflexes := modDB.Flag(nil, "IronReflexes")
		ward := float64(0)
		energyShield := float64(0)
		armour := float64(0)
		evasion := float64(0)
		energyShieldBase := float64(0)
		armourBase := float64(0)
		evasionBase := float64(0)
		wardBase := float64(0)
		gearWard := float64(0)
		gearEnergyShield := float64(0)
		gearArmour := float64(0)
		gearEvasion := float64(0)
		slotCfg := &moddb.ListCfg{}
		for _, slot := range []string{"Helmet", "Body Armour", "Gloves", "Boots", "Weapon 2", "Weapon 3"} {
			itemData := actor.ItemList[slot]
			var armourData *ArmourData
			if itemData != nil {
				armourData = itemData.ArmourData
			}
			if armourData != nil {
				slotCfg.SlotName = slot
				wardBase = armourData.Ward
				if wardBase > 0 {
					actor.Output["WardOn"+slot] = wardBase
					if modDB.Flag(nil, "EnergyShieldToWard") {
						inc := modDB.Sum(mod.TypeIncrease, slotCfg, "Ward", "Defences", "EnergyShield")
						more := modDB.More(slotCfg, "Ward", "Defences")
						ward = ward + wardBase*(1+inc/100)*more
						gearWard = gearWard + wardBase
						if actor.Breakdown != nil {
							actor.Breakdown.AddSlot("Ward", BSlot{
								Base:   wardBase,
								Inc:    utils.Ternary(inc != 0, fmt.Sprintf(" x %.2f", 1+inc/100), ""),
								More:   utils.Ternary(more != 1, fmt.Sprintf(" x %.2f", more), ""),
								Total:  fmt.Sprintf("%.2f", wardBase*(1+inc/100)*more),
								Source: slot,
								Item:   actor.ItemList[slot],
							})
						}
					} else {
						ward = ward + wardBase*calclib.Mod(modDB, slotCfg, "Ward", "Defences")
						gearWard = gearWard + wardBase
						if actor.Breakdown != nil {
							actor.Breakdown.Slot(slot, nil, slotCfg, wardBase, nil, "Ward", "Defences")
						}
					}
				}
				energyShieldBase = armourData.EnergyShield
				if energyShieldBase > 0 {
					actor.Output["EnergyShieldOn"+slot] = energyShieldBase
					if modDB.Flag(nil, "EnergyShieldToWard") {
						more := modDB.More(slotCfg, "EnergyShield", "Defences")
						energyShield = energyShield + energyShieldBase*more
						gearEnergyShield = gearEnergyShield + energyShieldBase
						if actor.Breakdown != nil {
							actor.Breakdown.AddSlot("EnergyShield", BSlot{
								Base:   energyShieldBase,
								More:   utils.Ternary(more != 1, fmt.Sprintf(" x %.2f", more), ""),
								Total:  fmt.Sprintf("%.2f", energyShieldBase*more),
								Source: slot,
								Item:   actor.ItemList[slot],
							})
						}
					} else {
						energyShield = energyShield + energyShieldBase*calclib.Mod(modDB, slotCfg, "EnergyShield", "Defences")
						gearEnergyShield = gearEnergyShield + energyShieldBase
						if actor.Breakdown != nil {
							actor.Breakdown.Slot(slot, nil, slotCfg, energyShieldBase, nil, "EnergyShield", "Defences")
						}
					}
				}
				armourBase = armourData.Armour
				if armourBase > 0 {
					actor.Output["ArmourOn"+slot] = armourBase
					if slot == "Body Armour" && modDB.Flag(nil, "Unbreakable") {
						armourBase = armourBase * 2
					}
					armour = armour + armourBase*calclib.Mod(modDB, slotCfg, "Armour", "ArmourAndEvasion", "Defences")
					gearArmour = gearArmour + armourBase
					if actor.Breakdown != nil {
						actor.Breakdown.Slot(slot, nil, slotCfg, armourBase, nil, "Armour", "ArmourAndEvasion", "Defences")
					}
				}
				evasionBase = armourData.Evasion
				if evasionBase > 0 {
					actor.Output["EvasionOn"+slot] = evasionBase
					if ironReflexes {
						armour = armour + evasionBase*calclib.Mod(modDB, slotCfg, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
						gearArmour = gearArmour + evasionBase
						if actor.Breakdown != nil {
							actor.Breakdown.Slot(slot, nil, slotCfg, evasionBase, nil, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
						}
					} else {
						evasion = evasion + evasionBase*calclib.Mod(modDB, slotCfg, "Evasion", "ArmourAndEvasion", "Defences")
						gearEvasion = gearEvasion + evasionBase
						if actor.Breakdown != nil {
							actor.Breakdown.Slot(slot, nil, slotCfg, evasionBase, nil, "Evasion", "ArmourAndEvasion", "Defences")
						}
					}
				}
			}
		}
		wardBase = modDB.Sum(mod.TypeBase, nil, "Ward")

		if wardBase > 0 {
			if modDB.Flag(nil, "EnergyShieldToWard") {
				inc := modDB.Sum(mod.TypeIncrease, slotCfg, "Ward", "Defences", "EnergyShield")
				more := modDB.More(slotCfg, "Ward", "Defences")
				ward = ward + wardBase*(1+inc/100)*more
				if actor.Breakdown != nil {
					actor.Breakdown.AddSlot("Ward", BSlot{
						Base:   wardBase,
						Inc:    utils.Ternary(inc != 0, fmt.Sprintf(" x %.2f", 1+inc/100), ""),
						More:   utils.Ternary(more != 1, fmt.Sprintf(" x %.2f", more), ""),
						Total:  fmt.Sprintf("%.2f", wardBase*(1+inc/100)*more),
						Source: "Global",
						Item:   actor.ItemList["Global"],
					})
				}
			} else {
				ward = ward + wardBase*calclib.Mod(modDB, nil, "Ward", "Defences")
				if actor.Breakdown != nil {
					actor.Breakdown.Slot("Global", nil, nil, wardBase, nil, "Ward", "Defences")
				}
			}
		}
		energyShieldBase = modDB.Sum(mod.TypeBase, nil, "EnergyShield")
		if energyShieldBase > 0 {
			if modDB.Flag(nil, "EnergyShieldToWard") {
				energyShield = energyShield + energyShieldBase*modDB.More(slotCfg, "EnergyShield", "Defences")
			} else {
				energyShield = energyShield + energyShieldBase*calclib.Mod(modDB, nil, "EnergyShield", "Defences")
			}
			if actor.Breakdown != nil {
				more := modDB.More(slotCfg, "EnergyShield", "Defences")
				actor.Breakdown.AddSlot("EnergyShield", BSlot{
					Base:   energyShieldBase,
					More:   utils.Ternary(more != 1, fmt.Sprintf(" x %.2f", more), ""),
					Total:  fmt.Sprintf("%.2f", energyShieldBase*more),
					Source: "Global",
					Item:   actor.ItemList["Global"],
				})
			}
		}
		armourBase = modDB.Sum(mod.TypeBase, nil, "Armour", "ArmourAndEvasion")
		if armourBase > 0 {
			armour = armour + armourBase*calclib.Mod(modDB, nil, "Armour", "ArmourAndEvasion", "Defences")
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Global", nil, nil, armourBase, nil, "Armour", "ArmourAndEvasion", "Defences")
			}
		}
		evasionBase = modDB.Sum(mod.TypeBase, nil, "Evasion", "ArmourAndEvasion")
		if evasionBase > 0 {
			if ironReflexes {
				armour = armour + evasionBase*calclib.Mod(modDB, nil, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
				if actor.Breakdown != nil {
					actor.Breakdown.Slot("Conversion", utils.Ptr("Evasion to Armour"), nil, evasionBase, nil, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
				}
			} else {
				evasion = evasion + evasionBase*calclib.Mod(modDB, nil, "Evasion", "ArmourAndEvasion", "Defences")
				if actor.Breakdown != nil {
					actor.Breakdown.Slot("Global", nil, nil, evasionBase, nil, "Evasion", "ArmourAndEvasion", "Defences")
				}
			}
		}
		convManaToArmour := modDB.Sum(mod.TypeBase, nil, "ManaConvertToArmour")
		if convManaToArmour > 0 {
			armourBase = 2 * modDB.Sum(mod.TypeBase, nil, "Mana") * convManaToArmour / 100
			total := armourBase * calclib.Mod(modDB, nil, "Mana", "Armour", "ArmourAndEvasion", "Defences")
			armour = armour + total
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Conversion", utils.Ptr("Mana to Armour"), nil, armourBase, utils.Ptr(total), "Armour", "ArmourAndEvasion", "Defences", "Mana")
			}
		}
		convManaToES := modDB.Sum(mod.TypeBase, nil, "ManaGainAsEnergyShield")
		if convManaToES > 0 {
			energyShieldBase = modDB.Sum(mod.TypeBase, nil, "Mana") * convManaToES / 100
			energyShield = energyShield + energyShieldBase*calclib.Mod(modDB, nil, "Mana", "EnergyShield", "Defences")
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Conversion", utils.Ptr("Mana to Energy Shield"), nil, energyShieldBase, nil, "EnergyShield", "Defences", "Mana")
			}
		}
		convLifeToArmour := modDB.Sum(mod.TypeBase, nil, "LifeGainAsArmour")
		if convLifeToArmour > 0 {
			armourBase = modDB.Sum(mod.TypeBase, nil, "Life") * convLifeToArmour / 100
			total := float64(0)
			if modDB.Flag(nil, "ChaosInoculation") {
				total = 1
			} else {
				total = armourBase * calclib.Mod(modDB, nil, "Life", "Armour", "ArmourAndEvasion", "Defences")
			}
			armour = armour + total
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Conversion", utils.Ptr("Life to Armour"), nil, armourBase, utils.Ptr(total), "Armour", "ArmourAndEvasion", "Defences", "Life")
			}
		}
		convLifeToES := modDB.Sum(mod.TypeBase, nil, "LifeConvertToEnergyShield", "LifeGainAsEnergyShield")
		if convLifeToES > 0 {
			energyShieldBase = modDB.Sum(mod.TypeBase, nil, "Life") * convLifeToES / 100
			total := float64(0)
			if modDB.Flag(nil, "ChaosInoculation") {
				total = 1
			} else {
				total = energyShieldBase * calclib.Mod(modDB, nil, "Life", "EnergyShield", "Defences")
			}
			energyShield = energyShield + total
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Conversion", utils.Ptr("Life to Energy Shield"), nil, energyShieldBase, utils.Ptr(total), "EnergyShield", "Defences", "Life")
			}
		}
		convEvasionToArmour := modDB.Sum(mod.TypeBase, nil, "EvasionGainAsArmour")
		if convEvasionToArmour > 0 {
			armourBase = (modDB.Sum(mod.TypeBase, nil, "Evasion") + gearEvasion) * convEvasionToArmour / 100
			total := armourBase * calclib.Mod(modDB, nil, "Evasion", "Armour", "ArmourAndEvasion", "Defences")
			armour = armour + total
			if actor.Breakdown != nil {
				actor.Breakdown.Slot("Conversion", utils.Ptr("Evasion to Armour"), nil, armourBase, utils.Ptr(total), "Armour", "ArmourAndEvasion", "Defences", "Evasion")
			}
		}
		actor.Output["EnergyShield"] = utils.Or(modDB.Override(nil, "EnergyShield"), max(utils.RoundTo(energyShield, 0), 0))
		actor.Output["Armour"] = max(utils.RoundTo(armour, 0), 0)
		actor.Output["ArmourDefense"] = (modDB.Max(nil, "ArmourDefense")) / 100
		actor.Output["RawArmourDefense"] = utils.Ternary(actor.Output["ArmourDefense"] > 0, (1+actor.Output["ArmourDefense"])*100, 0)
		actor.Output["Evasion"] = max(utils.RoundTo(evasion, 0), 0)
		actor.Output["LowestOfArmourAndEvasion"] = min(actor.Output["Armour"], actor.Output["Evasion"])
		actor.Output["Ward"] = max(utils.RoundTo(ward, 0), 0)
		actor.Output["Gear:Ward"] = gearWard
		actor.Output["Gear:EnergyShield"] = gearEnergyShield
		actor.Output["Gear:Armour"] = gearArmour
		actor.Output["Gear:Evasion"] = gearEvasion

		ArmourESRecoveryCap := modDB.Flag(nil, "ArmourESRecoveryCap")
		EvasionESRecoveryCap := modDB.Flag(nil, "EvasionESRecoveryCap")
		CappingES := utils.Ternary(ArmourESRecoveryCap, actor.Output["Armour"] < actor.Output["EnergyShield"], utils.Ternary(EvasionESRecoveryCap, actor.Output["Evasion"] < actor.Output["EnergyShield"], environment.Build.GetBooleanOption("conditionLowEnergyShield")))

		if CappingES {
			actor.Output["CappingES"] = 1
		}

		if actor.Output["CappingES"] != 0 {
			actor.Output["EnergyShieldRecoveryCap"] = utils.Ternary(ArmourESRecoveryCap && EvasionESRecoveryCap, min(actor.Output["Armour"], actor.Output["Evasion"]), utils.Ternary(ArmourESRecoveryCap, actor.Output["Armour"], utils.Ternary(EvasionESRecoveryCap, actor.Output["Evasion"], actor.Output["EnergyShield"])))
			actor.Output["EnergyShieldRecoveryCap"] = utils.Ternary(environment.Build.GetBooleanOption("conditionLowEnergyShield"), min(actor.Output["EnergyShield"]*data.LowPoolThreshold, actor.Output["EnergyShieldRecoveryCap"]), actor.Output["EnergyShieldRecoveryCap"])
		} else {
			actor.Output["EnergyShieldRecoveryCap"] = actor.Output["EnergyShield"]
		}

		if modDB.Flag(nil, "CannotEvade") {
			actor.Output["EvadeChance"] = 0
			actor.Output["MeleeEvadeChance"] = 0
			actor.Output["ProjectileEvadeChance"] = 0
		} else {
			enemyAccuracy := utils.RoundTo(calclib.Val(actor.Enemy.ModDB, "Accuracy"), 0)
			actor.Output["EvadeChance"] = 100 - (CalcHitChance(actor.Output["Evasion"], enemyAccuracy)-modDB.Sum(mod.TypeBase, nil, "EvadeChance"))*calclib.Mod(actor.Enemy.ModDB, nil, "HitChance")
			actor.Output["MeleeEvadeChance"] = max(0, min(data.EvadeChanceCap, actor.Output["EvadeChance"]*calclib.Mod(modDB, nil, "EvadeChance", "MeleeEvadeChance")))
			actor.Output["ProjectileEvadeChance"] = max(0, min(data.EvadeChanceCap, actor.Output["EvadeChance"]*calclib.Mod(modDB, nil, "EvadeChance", "ProjectileEvadeChance")))
			// Condition for displayng evade chance only if melee or projectile evade chance have the same values
			if actor.Output["MeleeEvadeChance"] != actor.Output["ProjectileEvadeChance"] {
				actor.Output["splitEvade"] = 1
			} else {
				actor.Output["EvadeChance"] = actor.Output["MeleeEvadeChance"]
				actor.Output["dontSplitEvade"] = 1
			}
			if actor.Breakdown != nil {
				actor.Breakdown.AddLine("EvadeChance",
					fmt.Sprintf("Enemy level: %d ^8(%s the Configuration tab)", environment.EnemyLevel, utils.Ternary(environment.Build.GetNumberOption("enemyLevel") != 0, "overridden from", "can be overridden in")),
					fmt.Sprintf("Average enemy accuracy: %.2f", enemyAccuracy),
					fmt.Sprintf("Approximate evade chance: %.2f%%", actor.Output["EvadeChance"]),
				)
				actor.Breakdown.AddLine("MeleeEvadeChance",
					fmt.Sprintf("Enemy level: %d ^8(%s the Configuration tab)", environment.EnemyLevel, utils.Ternary(environment.Build.GetNumberOption("enemyLevel") != 0, "overridden from", "can be overridden in")),
					fmt.Sprintf("Average enemy accuracy: %.2f", enemyAccuracy),
					fmt.Sprintf("Approximate melee evade chance: %.2f%%", actor.Output["MeleeEvadeChance"]),
				)
				actor.Breakdown.AddLine("ProjectileEvadeChance",
					fmt.Sprintf("Enemy level: %d ^8(%s the Configuration tab)", environment.EnemyLevel, utils.Ternary(environment.Build.GetNumberOption("enemyLevel") != 0, "overridden from", "can be overridden in")),
					fmt.Sprintf("Average enemy accuracy: %.2f", enemyAccuracy),
					fmt.Sprintf("Approximate projectile evade chance: %.2f%%", actor.Output["ProjectileEvadeChance"]),
				)
			}
		}
	}

	// Dodge
	// Acrobatics Spell Suppression to Spell Dodge Chance conversion.
	if modDB.Flag(nil, "ConvertSpellSuppressionToSpellDodge") {
		SpellSuppressionChance := modDB.Sum(mod.TypeBase, nil, "SpellSuppressionChance")
		modDB.AddMod(mod.NewFloat("SpellDodgeChance", mod.TypeBase, SpellSuppressionChance/2).Source("Acrobatics"))
	}

	totalSpellSuppressionChance := utils.Or(modDB.Override(nil, "SpellSuppressionChance"), modDB.Sum(mod.TypeBase, nil, "SpellSuppressionChance"))

	actor.Output["SpellSuppressionChance"] = min(totalSpellSuppressionChance, data.SuppressionChanceCap)
	actor.Output["SpellSuppressionEffect"] = data.SuppressionEffect + modDB.Sum(mod.TypeBase, nil, "SpellSuppressionEffect")

	if environment.ModeEffective && modDB.Flag(nil, "SpellSuppressionChanceIsUnlucky") {
		actor.Output["SpellSuppressionChance"] = actor.Output["SpellSuppressionChance"] / 100 * actor.Output["SpellSuppressionChance"]
	} else if environment.ModeEffective && modDB.Flag(nil, "SpellSuppressionChanceIsLucky") {
		actor.Output["SpellSuppressionChance"] = (1 - math.Pow(1-actor.Output["SpellSuppressionChance"]/100, 2)) * 100
	}

	actor.Output["SpellSuppressionChanceOverCap"] = max(0, totalSpellSuppressionChance-data.SuppressionChanceCap)

	if actor.ItemList["Weapon 3"] != nil && actor.ItemList["Weapon 3"].ArmourData != nil {
		baseBlockChance = baseBlockChance + actor.ItemList["Weapon 3"].ArmourData.BlockChance
	}
	actor.Output["ShieldBlockChance"] = baseBlockChance
	if modDB.Flag(nil, "MaxBlockIfNotBlockedRecently") {
		actor.Output["BlockChance"] = actor.Output["BlockChanceMax"]
	} else {
		actor.Output["BlockChance"] = min((baseBlockChance+modDB.Sum(mod.TypeBase, nil, "BlockChance"))*calclib.Mod(modDB, nil, "BlockChance"), actor.Output["BlockChanceMax"])
	}
	actor.Output["ProjectileBlockChance"] = min(actor.Output["BlockChance"]+modDB.Sum(mod.TypeBase, nil, "ProjectileBlockChance")*calclib.Mod(modDB, nil, "BlockChance"), actor.Output["BlockChanceMax"])
	if modDB.Flag(nil, "SpellBlockChanceMaxIsBlockChanceMax") {
		actor.Output["SpellBlockChanceMax"] = actor.Output["BlockChanceMax"]
	} else {
		actor.Output["SpellBlockChanceMax"] = modDB.Sum(mod.TypeBase, nil, "SpellBlockChanceMax")
	}
	if modDB.Flag(nil, "SpellBlockChanceIsBlockChance") {
		actor.Output["SpellBlockChance"] = actor.Output["BlockChance"]
		actor.Output["SpellProjectileBlockChance"] = actor.Output["ProjectileBlockChance"]
	} else {
		actor.Output["SpellBlockChance"] = min(modDB.Sum(mod.TypeBase, nil, "SpellBlockChance")*calclib.Mod(modDB, nil, "SpellBlockChance"), actor.Output["SpellBlockChanceMax"])
		actor.Output["SpellProjectileBlockChance"] = actor.Output["SpellBlockChance"]
	}
	if actor.Breakdown != nil {
		actor.Breakdown.Simple(utils.Ptr(baseBlockChance), nil, actor.Output["BlockChance"], "BlockChance")
		actor.Breakdown.Simple(nil, nil, actor.Output["SpellBlockChance"], "SpellBlockChance")
	}
	if modDB.Flag(nil, "CannotBlockAttacks") {
		actor.Output["BlockChance"] = 0
		actor.Output["ProjectileBlockChance"] = 0
	}
	if modDB.Flag(nil, "CannotBlockSpells") {
		actor.Output["SpellBlockChance"] = 0
		actor.Output["SpellProjectileBlockChance"] = 0
	}
	actor.Output["AverageBlockChance"] = (actor.Output["BlockChance"] + actor.Output["ProjectileBlockChance"] + actor.Output["SpellBlockChance"] + actor.Output["SpellProjectileBlockChance"]) / 4
	actor.Output["BlockEffect"] = max(100-modDB.Sum(mod.TypeBase, nil, "BlockEffect"), 0)
	if actor.Output["BlockEffect"] == 0 || actor.Output["BlockEffect"] == 100 {
		actor.Output["BlockEffect"] = 100
	} else {
		actor.Output["ShowBlockEffect"] = 1
		actor.Output["DamageTakenOnBlock"] = 100 - actor.Output["BlockEffect"]
	}
	actor.Output["LifeOnBlock"] = modDB.Sum(mod.TypeBase, nil, "LifeOnBlock")
	actor.Output["ManaOnBlock"] = modDB.Sum(mod.TypeBase, nil, "ManaOnBlock")
	actor.Output["EnergyShieldOnBlock"] = modDB.Sum(mod.TypeBase, nil, "EnergyShieldOnBlock")

	// Dodge
	baseDodgeChance := 0
	totalAttackDodgeChance := modDB.Sum(mod.TypeBase, nil, "AttackDodgeChance")
	totalSpellDodgeChance := modDB.Sum(mod.TypeBase, nil, "SpellDodgeChance")
	attackDodgeChanceMax := data.DodgeChanceCap
	spellDodgeChanceMax := utils.Or(modDB.Override(nil, "SpellDodgeChanceMax"), modDB.Sum(mod.TypeBase, nil, "SpellDodgeChanceMax"))

	actor.Output["AttackDodgeChance"] = min(totalAttackDodgeChance, float64(attackDodgeChanceMax))
	actor.Output["SpellDodgeChance"] = min(totalSpellDodgeChance, spellDodgeChanceMax)
	if environment.ModeEffective && modDB.Flag(nil, "DodgeChanceIsUnlucky") {
		actor.Output["AttackDodgeChance"] = actor.Output["AttackDodgeChance"] / 100 * actor.Output["AttackDodgeChance"]
		actor.Output["SpellDodgeChance"] = actor.Output["SpellDodgeChance"] / 100 * actor.Output["SpellDodgeChance"]
	}
	actor.Output["AttackDodgeChanceOverCap"] = max(0, totalAttackDodgeChance-float64(attackDodgeChanceMax))
	actor.Output["SpellDodgeChanceOverCap"] = max(0, totalSpellDodgeChance-spellDodgeChanceMax)

	if actor.Breakdown != nil {
		actor.Breakdown.AddLine("AttackDodgeChance",
			"Base: "+fmt.Sprint(baseDodgeChance)+"%",
			"Max: "+fmt.Sprint(attackDodgeChanceMax)+"%",
			"Total: "+fmt.Sprint(actor.Output["AttackDodgeChance"]+actor.Output["AttackDodgeChanceOverCap"])+"%",
		)
		actor.Breakdown.AddLine("SpellDodgeChance",
			"Base: "+fmt.Sprint(baseDodgeChance)+"%",
			"Max: "+fmt.Sprint(spellDodgeChanceMax)+"%",
			"Total: "+fmt.Sprint(actor.Output["SpellDodgeChance"]+actor.Output["SpellDodgeChanceOverCap"])+"%",
		)
	}

	// Recovery modifiers
	actor.Output["LifeRecoveryRateMod"] = calclib.Mod(modDB, nil, "LifeRecoveryRate")
	actor.Output["ManaRecoveryRateMod"] = calclib.Mod(modDB, nil, "ManaRecoveryRate")
	actor.Output["EnergyShieldRecoveryRateMod"] = calclib.Mod(modDB, nil, "EnergyShieldRecoveryRate")

	// Leech caps
	actor.Output["MaxLifeLeechInstance"] = actor.Output["Life"] * calclib.Val(modDB, "MaxLifeLeechInstance") / 100
	actor.Output["MaxLifeLeechRatePercent"] = calclib.Val(modDB, "MaxLifeLeechRate")
	actor.Output["MaxLifeLeechRate"] = actor.Output["Life"] * actor.Output["MaxLifeLeechRatePercent"] / 100
	if actor.Breakdown != nil {
		actor.Breakdown.AddLine("MaxLifeLeechRate",
			fmt.Sprintf("%.2f ^8(maximum life)", actor.Output["Life"]),
			fmt.Sprintf("x %.2f%% ^8(percentage of life to maximum leech rate)", actor.Output["MaxLifeLeechRatePercent"]),
			fmt.Sprintf("= %.1f", actor.Output["MaxLifeLeechRate"]),
		)
	}
	actor.Output["MaxEnergyShieldLeechInstance"] = actor.Output["EnergyShield"] * calclib.Val(modDB, "MaxEnergyShieldLeechInstance") / 100
	actor.Output["MaxEnergyShieldLeechRate"] = actor.Output["EnergyShield"] * calclib.Val(modDB, "MaxEnergyShieldLeechRate") / 100
	if actor.Breakdown != nil {
		actor.Breakdown.AddLine("MaxEnergyShieldLeechRate",
			fmt.Sprintf("%.2f ^8(maximum energy shield)", actor.Output["EnergyShield"]),
			fmt.Sprintf("x %.2f%% ^8(percentage of energy shield to maximum leech rate)", calclib.Val(modDB, "MaxEnergyShieldLeechRate")),
			fmt.Sprintf("= %.1f", actor.Output["MaxEnergyShieldLeechRate"]),
		)
	}
	actor.Output["MaxManaLeechInstance"] = actor.Output["Mana"] * calclib.Val(modDB, "MaxManaLeechInstance") / 100
	actor.Output["MaxManaLeechRate"] = actor.Output["Mana"] * calclib.Val(modDB, "MaxManaLeechRate") / 100
	if actor.Breakdown != nil {
		actor.Breakdown.AddLine("MaxManaLeechRate",
			fmt.Sprintf("%.2f ^8(maximum mana)", actor.Output["Mana"]),
			fmt.Sprintf("x %.2f%% ^8(percentage of mana to maximum leech rate)", modDB.Sum(mod.TypeBase, nil, "MaxManaLeechRate")),
			fmt.Sprintf("= %.1f", actor.Output["MaxManaLeechRate"]),
		)
	}

	// Mana, life, energy shield, and rage regen
	if modDB.Flag(nil, "NoManaRegen") {
		actor.Output["ManaRegen"] = 0
	} else {
		base := modDB.Sum(mod.TypeBase, nil, "ManaRegen") + actor.Output["Mana"]*modDB.Sum(mod.TypeBase, nil, "ManaRegenPercent")
		actor.Output["ManaRegenInc"] = modDB.Sum(mod.TypeIncrease, nil, "ManaRegen")
		more := modDB.More(nil, "ManaRegen")
		if modDB.Flag(nil, "ManaRegenToRageRegen") {
			actor.Output["ManaRegenInc"] = 0
		}
		regen := base * (1 + actor.Output["ManaRegenInc"]/100) * more
		regenRate := utils.RoundTo(regen*actor.Output["ManaRecoveryRateMod"], 1)
		degen := modDB.Sum(mod.TypeBase, nil, "ManaDegen")
		actor.Output["ManaRegen"] = regenRate - degen
		if actor.Breakdown != nil {
			actor.Breakdown.MultiChain("ManaRegen", BMultiChain{
				Label: "Mana Regeneration:",
				Base:  fmt.Sprintf("%.1f ^8(base)", base),
				Total: fmt.Sprintf("= %.1f ^8per second", regen),
				Items: []BMultiChainItem{
					{"%.2f ^8(increased/reduced)", 1 + actor.Output["ManaRegenInc"]/100},
					{"%.2f ^8(more/less)", more},
				},
			})
			actor.Breakdown.MultiChain("ManaRegen", BMultiChain{
				Label: "Effective Mana Regeneration:",
				Base:  fmt.Sprintf("%.1f", regen),
				Total: fmt.Sprintf("= %.1f ^8per second", regenRate),
				Items: []BMultiChainItem{
					{"%.2f ^8(recovery rate modifier)", actor.Output["ManaRecoveryRateMod"]},
				},
			})
			if degen != 0 {
				actor.Breakdown.AddLine("ManaRegen", fmt.Sprintf("- %.2f", degen))
				actor.Breakdown.AddLine("ManaRegen", fmt.Sprintf("= %.1f ^8per second", actor.Output["ManaRegen"]))
			}
		}
	}

	if modDB.Flag(nil, "NoLifeRegen") {
		actor.Output["LifeRegen"] = 0
	} else if modDB.Flag(nil, "ZealotsOath") {
		actor.Output["LifeRegen"] = 0
		lifeBase := modDB.Sum(mod.TypeBase, nil, "LifeRegen")
		if lifeBase > 0 {
			modDB.AddMod(mod.NewFloat("EnergyShieldRegen", mod.TypeBase, lifeBase).Source("Zealot's Oath"))
		}
		lifePercent := modDB.Sum(mod.TypeBase, nil, "LifeRegenPercent")
		if lifePercent > 0 {
			modDB.AddMod(mod.NewFloat("EnergyShieldRegenPercent", mod.TypeBase, lifePercent).Source("Zealot's Oath"))
		}
	} else {
		lifeBase := modDB.Sum(mod.TypeBase, nil, "LifeRegen")
		lifePercent := modDB.Sum(mod.TypeBase, nil, "LifeRegenPercent")
		if lifePercent > 0 {
			lifeBase = lifeBase + actor.Output["Life"]*lifePercent/100
		}
		if lifeBase > 0 {
			actor.Output["LifeRegen"] = lifeBase * actor.Output["LifeRecoveryRateMod"] * modDB.More(nil, "LifeRegen") * (1 + modDB.Sum(mod.TypeIncrease, nil, "LifeRegen")/100)
		} else {
			actor.Output["LifeRegen"] = 0
		}

		// Don't add life recovery mod for this
		if actor.Output["LifeRegen"] > 0 && modDB.Flag(nil, "LifeRegenerationRecoversEnergyShield") && actor.Output["EnergyShield"] > 0 {
			modDB.AddMod(mod.NewFloat("EnergyShieldRecovery", mod.TypeBase, lifeBase*modDB.More(nil, "LifeRegen")*(1+modDB.Sum(mod.TypeIncrease, nil, "LifeRegen")/100)).Source("Life Regeneration Recovers Energy Shield"))
		}
	}

	actor.Output["LifeRegen"] = actor.Output["LifeRegen"] - modDB.Sum(mod.TypeBase, nil, "LifeDegen") + modDB.Sum(mod.TypeBase, nil, "LifeRecovery")*actor.Output["LifeRecoveryRateMod"]
	actor.Output["LifeRegenPercent"] = utils.RoundTo(actor.Output["LifeRegen"]/actor.Output["Life"]*100, 1)

	if modDB.Flag(nil, "NoEnergyShieldRegen") {
		actor.Output["EnergyShieldRegen"] = 0 - modDB.Sum(mod.TypeBase, nil, "EnergyShieldDegen")
		actor.Output["EnergyShieldRegenPercent"] = utils.RoundTo(actor.Output["EnergyShieldRegen"]/actor.Output["EnergyShield"]*100, 1)
	} else {
		esBase := modDB.Sum(mod.TypeBase, nil, "EnergyShieldRegen")
		esPercent := modDB.Sum(mod.TypeBase, nil, "EnergyShieldRegenPercent")
		if esPercent > 0 {
			esBase = esBase + actor.Output["EnergyShield"]*esPercent/100
		}
		if esBase > 0 {
			actor.Output["EnergyShieldRegen"] = esBase*actor.Output["EnergyShieldRecoveryRateMod"]*calclib.Mod(modDB, nil, "EnergyShieldRegen") - modDB.Sum(mod.TypeBase, nil, "EnergyShieldDegen")
			actor.Output["EnergyShieldRegenPercent"] = utils.RoundTo(actor.Output["EnergyShieldRegen"]/actor.Output["EnergyShield"]*100, 1)
		} else {
			actor.Output["EnergyShieldRegen"] = 0 - modDB.Sum(mod.TypeBase, nil, "EnergyShieldDegen")
		}
	}

	actor.Output["EnergyShieldRegen"] = actor.Output["EnergyShieldRegen"] + modDB.Sum(mod.TypeBase, nil, "EnergyShieldRecovery")*actor.Output["EnergyShieldRecoveryRateMod"]
	actor.Output["EnergyShieldRegenPercent"] = utils.RoundTo(actor.Output["EnergyShieldRegen"]/actor.Output["EnergyShield"]*100, 1)

	if modDB.Sum(mod.TypeBase, nil, "RageRegen") > 0 {
		modDB.AddMod(mod.NewFlag("Condition:CanGainRage", true).Source("RageRegen"))
		base := modDB.Sum(mod.TypeBase, nil, "RageRegen")
		if modDB.Flag(nil, "ManaRegenToRageRegen") {
			mana := modDB.Sum(mod.TypeIncrease, nil, "ManaRegen")
			modDB.AddMod(mod.NewFloat("RageRegen", mod.TypeIncrease, mana).Source("Mana Regen to Rage Regen"))
		}
		inc := modDB.Sum(mod.TypeIncrease, nil, "RageRegen")
		more := modDB.More(nil, "RageRegen")
		actor.Output["RageRegen"] = base * (1 + inc/100) * more

		if actor.Breakdown != nil {
			actor.Breakdown.MultiChain("RageRegen", BMultiChain{
				Base:  fmt.Sprintf("%.1f ^8(base)", base),
				Total: fmt.Sprintf("= %.1f ^8per second", actor.Output["RageRegen"]),
				Items: []BMultiChainItem{
					{"%.2f ^8(increased/reduced)", 1 + inc/100},
					{"%.2f ^8(more/less)", more},
				},
			})
		}
	}

	// Energy Shield Recharge
	if modDB.Flag(nil, "NoEnergyShieldRecharge") {
		actor.Output["EnergyShieldRecharge"] = 0
	} else {
		inc := modDB.Sum(mod.TypeIncrease, nil, "EnergyShieldRecharge")
		more := modDB.More(nil, "EnergyShieldRecharge")
		if modDB.Flag(nil, "EnergyShieldRechargeAppliesToLife") {
			actor.Output["EnergyShieldRechargeAppliesToLife"] = 1
			recharge := actor.Output["Life"] * data.EnergyShieldRechargeBase * (1 + inc/100) * more
			actor.Output["LifeRecharge"] = utils.RoundTo(recharge*actor.Output["LifeRecoveryRateMod"], 0)
			if actor.Breakdown != nil {
				actor.Breakdown.MultiChain("LifeRecharge", BMultiChain{
					Label: "Recharge rate:",
					Base:  fmt.Sprintf("%.1f ^8(33%% per second)", actor.Output["Life"]*data.EnergyShieldRechargeBase),
					Total: fmt.Sprintf("= %.1f ^8per second", recharge),
					Items: []BMultiChainItem{
						{"%.2f ^8(increased/reduced)", 1 + inc/100},
						{"%.2f ^8(more/less)", more},
					},
				})
				actor.Breakdown.MultiChain("LifeRecharge", BMultiChain{
					Label: "Effective Recharge rate:",
					Base:  fmt.Sprintf("%.1f", recharge),
					Total: fmt.Sprintf("= %.1f ^8per second", actor.Output["LifeRecharge"]),
					Items: []BMultiChainItem{
						{"%.2f ^8(recovery rate modifier)", actor.Output["LifeRecoveryRateMod"]},
					},
				})
			}
		} else {
			actor.Output["EnergyShieldRechargeAppliesToEnergyShield"] = 1
			recharge := actor.Output["EnergyShield"] * data.EnergyShieldRechargeBase * (1 + inc/100) * more
			actor.Output["EnergyShieldRecharge"] = utils.RoundTo(recharge*actor.Output["EnergyShieldRecoveryRateMod"], 0)
			if actor.Breakdown != nil {
				actor.Breakdown.MultiChain("EnergyShieldRecharge", BMultiChain{
					Label: "Recharge rate:",
					Base:  fmt.Sprintf("%.1f ^8(33%% per second)", actor.Output["EnergyShield"]*data.EnergyShieldRechargeBase),
					Total: fmt.Sprintf("= %.1f ^8per second", recharge),
					Items: []BMultiChainItem{
						{"%.2f ^8(increased/reduced)", 1 + inc/100},
						{"%.2f ^8(more/less)", more},
					},
				})
				actor.Breakdown.MultiChain("EnergyShieldRecharge", BMultiChain{
					Label: "Effective Recharge rate:",
					Base:  fmt.Sprintf("%.1f", recharge),
					Total: fmt.Sprintf("= %.1f ^8per second", actor.Output["EnergyShieldRecharge"]),
					Items: []BMultiChainItem{
						{"%.2f ^8(recovery rate modifier)", actor.Output["EnergyShieldRecoveryRateMod"]},
					},
				})
			}
		}
		actor.Output["EnergyShieldRechargeDelay"] = data.EnergyShieldRechargeDelay / (1 + modDB.Sum(mod.TypeIncrease, nil, "EnergyShieldRechargeFaster")/100)
		if actor.Breakdown != nil {
			if actor.Output["EnergyShieldRechargeDelay"] != data.EnergyShieldRechargeDelay {
				actor.Breakdown.AddLine("EnergyShieldRechargeDelay",
					fmt.Sprintf("%ds ^8(base)", data.EnergyShieldRechargeDelay),
					fmt.Sprintf("/ %.2f ^8(faster start)", 1+modDB.Sum(mod.TypeIncrease, nil, "EnergyShieldRechargeFaster")/100),
					fmt.Sprintf("= %.2fs", actor.Output["EnergyShieldRechargeDelay"]),
				)
			}
		}
	}

	// Ward recharge
	actor.Output["WardRechargeDelay"] = data.WardRechargeDelay / (1 + modDB.Sum(mod.TypeIncrease, nil, "WardRechargeFaster")/100)
	if actor.Breakdown != nil {
		if actor.Output["WardRechargeDelay"] != data.WardRechargeDelay {
			actor.Breakdown.AddLine("WardRechargeDelay",
				fmt.Sprintf("%ds ^8(base)", data.WardRechargeDelay),
				fmt.Sprintf("/ %.2f ^8(faster start)", 1+modDB.Sum(mod.TypeIncrease, nil, "WardRechargeFaster")/100),
				fmt.Sprintf("= %.2fs", actor.Output["WardRechargeDelay"]),
			)
		}
	}

	// Miscellaneous: move speed, stun recovery, avoidance
	actor.Output["MovementSpeedMod"] = utils.Or(modDB.Override(nil, "MovementSpeed"), calclib.Mod(modDB, nil, "MovementSpeed"))
	if modDB.Flag(nil, "MovementSpeedCannotBeBelowBase") {
		actor.Output["MovementSpeedMod"] = max(actor.Output["MovementSpeedMod"], 1)
	}
	actor.Output["EffectiveMovementSpeedMod"] = actor.Output["MovementSpeedMod"] * actor.Output["ActionSpeedMod"]

	actor.Output["MovementSpeedMod"] = utils.Or(modDB.Override(nil, "MovementSpeed"), calclib.Mod(modDB, nil, "MovementSpeed"))
	if modDB.Flag(nil, "MovementSpeedCannotBeBelowBase") {
		actor.Output["MovementSpeedMod"] = max(actor.Output["MovementSpeedMod"], 1)
	}
	actor.Output["EffectiveMovementSpeedMod"] = actor.Output["MovementSpeedMod"] * actor.Output["ActionSpeedMod"]
	if actor.Breakdown != nil {
		actor.Breakdown.MultiChain("EffectiveMovementSpeedMod", BMultiChain{
			Total: fmt.Sprintf("= %.2f ^8(effective movement speed modifier)", actor.Output["EffectiveMovementSpeedMod"]),
			Items: []BMultiChainItem{
				{"%.2f ^8(movement speed modifier)", actor.Output["MovementSpeedMod"]},
				{"%.2f ^8(action speed modifier)", actor.Output["ActionSpeedMod"]},
			},
		})
	}

	if actor.Enemy.ModDB.Flag(nil, "Blind") {
		actor.Output["BlindEffectMod"] = calclib.Mod(actor.Enemy.ModDB, nil, "BlindEffect", "BuffEffectOnSelf") * 100
	}

	// recovery on block, needs to be after primary defences
	actor.Output["LifeOnBlock"] = modDB.Sum(mod.TypeBase, nil, "LifeOnBlock")
	actor.Output["ManaOnBlock"] = modDB.Sum(mod.TypeBase, nil, "ManaOnBlock")
	actor.Output["EnergyShieldOnBlock"] = modDB.Sum(mod.TypeBase, nil, "EnergyShieldOnBlock")
	actor.Output["EnergyShieldOnSpellBlock"] = modDB.Sum(mod.TypeBase, nil, "EnergyShieldOnSpellBlock")

	// damage avoidances
	for _, damageType := range data.DamageType("").Values() {
		actor.Output[string("Avoid"+damageType+"DamageChance")] = min(modDB.Sum(mod.TypeBase, nil, string("Avoid"+damageType+"DamageChance")), data.AvoidChanceCap)
	}
	actor.Output["AvoidProjectilesChance"] = min(modDB.Sum(mod.TypeBase, nil, "AvoidProjectilesChance"), data.AvoidChanceCap)

	// other avoidances etc
	stunChance := 100 - min(modDB.Sum(mod.TypeBase, nil, "AvoidStun"), 100)
	if actor.Output["EnergyShield"] > actor.Output["Life"]*2 {
		stunChance = stunChance * 0.5
	}
	actor.Output["StunAvoidChance"] = 100 - stunChance
	if actor.Output["StunAvoidChance"] >= 100 {
		actor.Output["StunDuration"] = 0
		actor.Output["BlockDuration"] = 0
	} else {
		actor.Output["StunDuration"] = 0.35 / (1 + modDB.Sum(mod.TypeIncrease, nil, "StunRecovery")/100)
		actor.Output["BlockDuration"] = 0.35 / (1 + modDB.Sum(mod.TypeIncrease, nil, "StunRecovery", "BlockRecovery")/100)
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("StunDuration",
				"0.35s ^8(base)",
				fmt.Sprintf("/ %.2f ^8(increased/reduced recovery)", 1+modDB.Sum(mod.TypeIncrease, nil, "StunRecovery")/100),
				fmt.Sprintf("= %.2fs", actor.Output["StunDuration"]),
			)
			actor.Breakdown.AddLine("BlockDuration",
				"0.35s ^8(base)",
				fmt.Sprintf("/ %.2f ^8(increased/reduced recovery)", 1+modDB.Sum(mod.TypeIncrease, nil, "StunRecovery", "BlockRecovery")/100),
				fmt.Sprintf("= %.2fs", actor.Output["BlockDuration"]),
			)
		}
	}
	actor.Output["InteruptStunAvoidChance"] = min(modDB.Sum(mod.TypeBase, nil, "AvoidInteruptStun"), 100)
	actor.Output["BlindAvoidChance"] = min(modDB.Sum(mod.TypeBase, nil, "AvoidBlind"), 100)
	for _, ailment := range data.Ailment("").Values() {
		actor.Output[string(ailment+"AvoidChance")] = min(modDB.Sum(mod.TypeBase, nil, string("Avoid"+ailment)), 100)
	}
	actor.Output["CritExtraDamageReduction"] = min(modDB.Sum(mod.TypeBase, nil, "ReduceCritExtraDamage"), 100)
	actor.Output["LightRadiusMod"] = calclib.Mod(modDB, nil, "LightRadius")
	if actor.Breakdown != nil {
		actor.Breakdown.Mod("LightRadiusMod", modDB, nil, "LightRadius")
	}
	actor.Output["CurseEffectOnSelf"] = modDB.More(nil, "CurseEffectOnSelf") * (100 + modDB.Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf"))

	// Ailment duration on self
	actor.Output["SelfBlindDuration"] = modDB.More(nil, "SelfBlindDuration") * (100 + modDB.Sum(mod.TypeIncrease, nil, "SelfBlindDuration"))
	for _, ailment := range data.Ailment("").Values() {
		actor.Output[string("Self"+ailment+"Duration")] = modDB.More(nil, string("Self"+ailment+"Duration")) * (100 + modDB.Sum(mod.TypeIncrease, nil, string("Self"+ailment+"Duration")))
	}
	actor.Output["SelfChillEffect"] = modDB.More(nil, "SelfChillEffect") * (100 + modDB.Sum(mod.TypeIncrease, nil, "SelfChillEffect"))
	actor.Output["SelfShockEffect"] = modDB.More(nil, "SelfShockEffect") * (100 + modDB.Sum(mod.TypeIncrease, nil, "SelfShockEffect"))

	{
		actor.Output["totalEnemyDamage"] = 0
		actor.Output["totalEnemyDamageIn"] = 0
		if actor.Breakdown != nil {
			actor.Breakdown.SetLabel("totalEnemyDamage", "Total damage from the enemy")
			actor.Breakdown.AddCol("totalEnemyDamage",
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "Value", Key: "value"},
				BCol{Label: "Mult", Key: "mult"},
				BCol{Label: "Crit", Key: "crit"},
				BCol{Label: "Final", Key: "final"},
				BCol{Label: "From", Key: "from"},
			)
		}
		enemyCritChance := environment.Build.GetNumberOption("enemyCritChance")
		enemyCritDamage := environment.Build.GetNumberOption("enemyCritDamage")
		actor.Output["EnemyCritEffect"] = 1 + enemyCritChance/100*(enemyCritDamage/100)*(1-actor.Output["CritExtraDamageReduction"]/100)
		for _, damageType := range data.DamageType("").Values() {
			enemyDamageMult := calclib.Mod(actor.Enemy.ModDB, nil, "Damage", string(damageType+"Damage"), utils.Ternary(isElemental[string(damageType)], "ElementalDamage", "")) // missing taunt from allies
			enemyDamage := environment.Build.GetNumberOption(string("enemy" + damageType + "Damage"))
			enemyPen := environment.Build.GetNumberOption(string("enemy" + damageType + "Pen"))
			sourceStr := utils.Ternary(enemyDamage == 0, "Default", "Config")

			actor.Output[string(damageType+"EnemyPen")] = enemyPen
			actor.Output["totalEnemyDamageIn"] = actor.Output["totalEnemyDamageIn"] + enemyDamage
			actor.Output[string(damageType+"EnemyDamage")] = enemyDamage * enemyDamageMult * actor.Output["EnemyCritEffect"]
			actor.Output["totalEnemyDamage"] = actor.Output["totalEnemyDamage"] + actor.Output[string(damageType+"EnemyDamage")]
			if actor.Breakdown != nil {
				actor.Breakdown.AddLine(string(damageType+"EnemyDamage"),
					fmt.Sprintf("from %s: %.2f", sourceStr, enemyDamage),
					fmt.Sprintf("* %.2f (modifiers to enemy damage)", enemyDamageMult),
					fmt.Sprintf("* %.3f (enemy crit effect)", actor.Output["EnemyCritEffect"]),
					fmt.Sprintf("= %.2f", actor.Output[string(damageType+"EnemyDamage")]),
				)
				actor.Breakdown.AddRow("totalEnemyDamage", map[string]string{
					"type":  fmt.Sprintf("%s", damageType),
					"value": fmt.Sprintf("%.2f", enemyDamage),
					"mult":  fmt.Sprintf("%.2f", enemyDamageMult),
					"crit":  fmt.Sprintf("%.2f", actor.Output["EnemyCritEffect"]),
					"final": fmt.Sprintf("%.2f", actor.Output[string(damageType+"EnemyDamage")]),
					"from":  fmt.Sprintf("%s", sourceStr),
				})
			}
		}
	}
	{
		actor.DamageShiftTable = make(map[data.DamageType]map[data.DamageType]float64)
		for _, damageType := range data.DamageType("").Values() {
			// Build damage shift table
			shiftTable := make(map[data.DamageType]float64)
			destTotal := float64(0)
			for _, destType := range data.DamageType("").Values() {
				if destType != damageType {
					shiftTable[destType] = modDB.Sum(mod.TypeBase, nil, string(damageType+"DamageTakenAs"+destType), utils.Ternary(isElemental[string(damageType)], string("ElementalDamageTakenAs"+destType), ""))
					destTotal = destTotal + shiftTable[destType]
				}
			}
			if destTotal > 100 {
				factor := 100 / destTotal
				for destType, portion := range shiftTable {
					shiftTable[destType] = portion * factor
				}
				destTotal = 100
			}
			shiftTable[damageType] = 100 - destTotal
			actor.DamageShiftTable[damageType] = shiftTable

			// add same type damage
			actor.Output[string(damageType+"TakenDamage")] = actor.Output[string(damageType+"EnemyDamage")] * actor.DamageShiftTable[damageType][damageType] / 100
			if actor.Breakdown != nil {
				actor.Breakdown.SetLabel(string(damageType+"TakenDamage"), "Taken")
				actor.Breakdown.AddCol(string(damageType+"TakenDamage"),
					BCol{Label: "Type", Key: "type"},
					BCol{Label: "Value", Key: "value"},
				)
				actor.Breakdown.AddRow(string(damageType+"TakenDamage"), map[string]string{
					"type":  fmt.Sprintf("%s", damageType),
					"value": fmt.Sprintf("%.2f", actor.Output[string(damageType+"TakenDamage")]),
				})
			}
		}
		// converted damage types
		for _, damageType := range data.DamageType("").Values() {
			for _, damageConvertedType := range data.DamageType("").Values() {
				if damageType != damageConvertedType {
					damage := actor.Output[string(damageType+"EnemyDamage")] * actor.DamageShiftTable[damageType][damageConvertedType] / 100
					actor.Output[string(damageConvertedType+"TakenDamage")] = actor.Output[string(damageConvertedType+"TakenDamage")] + damage
					if actor.Breakdown != nil && damage > 0 {
						actor.Breakdown.AddRow(string(damageConvertedType+"TakenDamage"), map[string]string{
							"type":  fmt.Sprintf("%s", damageType),
							"value": fmt.Sprintf("%.2f", damage),
						})
					}
				}
			}
		}
		// total
		actor.Output["totalTakenDamage"] = 0
		if actor.Breakdown != nil {
			actor.Breakdown.SetLabel("totalTakenDamage", "Total damage taken from the enemy after taken as")
			actor.Breakdown.AddCol("totalTakenDamage",
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "Value", Key: "value"},
			)
		}
		for _, damageType := range data.DamageType("").Values() {
			actor.Output["totalTakenDamage"] = actor.Output["totalTakenDamage"] + actor.Output[string(damageType+"TakenDamage")]
			if actor.Breakdown != nil {
				actor.Breakdown.AddRow("totalTakenDamage", map[string]string{
					"type":  fmt.Sprintf("%s", damageType),
					"value": fmt.Sprintf("%.2f", actor.Output[string(damageType+"TakenDamage")]),
				})
			}
		}
	}
	// Damage taken multipliers/Degen calculations
	actor.Output["AnyTakenReflect"] = 0
	damageCategoryConfig := environment.Build.GetStringOption("enemyDamageType")
	if damageCategoryConfig == "" {
		damageCategoryConfig = "Average"
	}
	for _, damageType := range data.DamageType("").Values() {
		baseTakenInc := modDB.Sum(mod.TypeIncrease, nil, "DamageTaken", string(damageType+"DamageTaken"))
		baseTakenMore := modDB.More(nil, "DamageTaken", string(damageType+"DamageTaken"))
		if isElemental[string(damageType)] {
			baseTakenInc = baseTakenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTaken")
			baseTakenMore = baseTakenMore * modDB.More(nil, "ElementalDamageTaken")
		}
		{ // Hit
			takenInc := baseTakenInc + modDB.Sum(mod.TypeIncrease, nil, "DamageTakenWhenHit", string(damageType+"DamageTakenWhenHit"))
			takenMore := baseTakenMore * modDB.More(nil, "DamageTakenWhenHit", string(damageType+"DamageTakenWhenHit"))
			if isElemental[string(damageType)] {
				takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTakenWhenHit")
				takenMore = takenMore * modDB.More(nil, "ElementalDamageTakenWhenHit")
			}
			actor.Output[string(damageType+"TakenHitMult")] = max((1+takenInc/100)*takenMore, 0)

			for _, hitType := range []string{"Attack", "Spell"} {
				baseTakenIncType := takenInc + modDB.Sum(mod.TypeIncrease, nil, hitType+"DamageTaken")
				baseTakenMoreType := takenMore * modDB.More(nil, hitType+"DamageTaken")
				actor.Output[hitType+"TakenHitMult"] = max((1+baseTakenIncType/100)*baseTakenMoreType, 0)
				actor.Output[string(damageType)+hitType+"TakenHitMult"] = actor.Output[hitType+"TakenHitMult"]
			}
			{
				// Reflect
				takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, string(damageType+"ReflectedDamageTaken"))
				takenMore = takenMore * modDB.More(nil, string(damageType+"ReflectedDamageTaken"))
				if isElemental[string(damageType)] {
					takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalReflectedDamageTaken")
					takenMore = takenMore * modDB.More(nil, "ElementalReflectedDamageTaken")
				}
				actor.Output[string(damageType+"TakenReflect")] = max((1+takenInc/100)*takenMore, 0)
				if actor.Output[string(damageType+"TakenReflect")] != actor.Output[string(damageType+"TakenHitMult")] {
					actor.Output["AnyTakenReflect"] = 0 // true // this needs a rework as well
				}
			}
		}
		{ // Dot
			takenInc := baseTakenInc + modDB.Sum(mod.TypeIncrease, nil, "DamageTakenOverTime", string(damageType+"DamageTakenOverTime"))
			takenMore := baseTakenMore * modDB.More(nil, "DamageTakenOverTime", string(damageType+"DamageTakenOverTime"))
			if isElemental[string(damageType)] {
				takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTakenOverTime")
				takenMore = takenMore * modDB.More(nil, "ElementalDamageTakenOverTime")
			}
			resist := utils.Ternary(modDB.Flag(nil, "SelfIgnore"+string(damageType)+"Resistance"), 0, actor.Output[string(damageType+"Resist")])
			if damageType == "Physical" {
				resist = max(resist, 0)
			}
			actor.Output[string(damageType+"TakenDotMult")] = (1 - resist/100) * (1 + takenInc/100) * takenMore
			if actor.Breakdown != nil {
				actor.Breakdown.MultiChain(string(damageType+"TakenDotMult"), BMultiChain{
					Label: "DoT Multiplier:",
					Total: fmt.Sprintf("= %.2f", actor.Output[string(damageType+"TakenDotMult")]),
					Items: []BMultiChainItem{
						{"%.2f ^8(" + utils.Ternary(damageType == "Physical", "physical damage reduction", "resistance") + ")", 1 - resist/100},
						{"%.2f ^8(increased/reduced damage taken)", 1 + takenInc/100},
						{"%.2f ^8(more/less damage taken)", takenMore},
					},
				})
			}
		}
	}
	// Incoming hit damage multipliers
	actor.Output["totalTakenHit"] = 0
	if actor.Breakdown != nil {
		actor.Breakdown.SetLabel("totalTakenHit", "Total damage taken after mitigation")
		actor.Breakdown.AddCol("totalTakenHit",
			BCol{Label: "Type", Key: "type"},
			BCol{Label: "Incoming", Key: "incoming"},
			BCol{Label: "Mult", Key: "mult"},
			BCol{Label: "Value", Key: "value"},
		)
	}
	for _, damageType := range data.DamageType("").Values() {
		// Calculate incoming damage multiplier
		resist := utils.Ternary(modDB.Flag(nil, string("SelfIgnore"+damageType+"Resistance")), 0, utils.OrF(actor.Output[string(damageType+"ResistWhenHit")], actor.Output[string(damageType+"Resist")]))
		enemyPen := utils.Ternary(modDB.Flag(nil, string("SelfIgnore"+damageType+"Resistance")), 0, actor.Output[string(damageType+"EnemyPen")])
		takenFlat := modDB.Sum(mod.TypeBase, nil, "DamageTaken", string(damageType+"DamageTaken"), "DamageTakenWhenHit", string(damageType+"DamageTakenWhenHit"))
		if damageCategoryConfig == "Melee" || damageCategoryConfig == "Projectile" {
			takenFlat = takenFlat + modDB.Sum(mod.TypeBase, nil, "DamageTakenFromAttacks", string(damageType+"DamageTakenFromAttacks"))
		} else if damageCategoryConfig == "Average" {
			takenFlat = takenFlat + modDB.Sum(mod.TypeBase, nil, "DamageTakenFromAttacks", string(damageType+"DamageTakenFromAttacks"))/2
		}
		if damageType == "Physical" || modDB.Flag(nil, string("ArmourAppliesTo"+damageType+"DamageTaken")) {
			damage := actor.Output[string(damageType+"TakenDamage")]
			armourReduct := float64(0)
			portionArmour := float64(100)
			if damageType == "Physical" {
				if !modDB.Flag(nil, "ArmourDoesNotApplyToPhysicalDamageTaken") {
					armourReduct = CalcArmourReduction(actor.Output["Armour"]*(1+actor.Output["ArmourDefense"]), damage)
					armourReduct = max(min(actor.Output["DamageReductionMax"], resist-enemyPen+armourReduct), 0)
					resist = armourReduct
				}
			} else {
				portionArmour = 100 - (resist - enemyPen)
				armourReduct = CalcArmourReduction(actor.Output["Armour"]*(1+actor.Output["ArmourDefense"]), damage*portionArmour/100)
				armourReduct = min(actor.Output["DamageReductionMax"], armourReduct)
				resist = resist + armourReduct*portionArmour/100
			}
			actor.Output[string(damageType+"DamageReduction")] = utils.Ternary(portionArmour < 100, armourReduct*portionArmour/100, armourReduct)
			if actor.Breakdown != nil {
				if portionArmour > 100 {
					actor.Breakdown.AddLine("DamageReduction",
						fmt.Sprintf("Enemy Hit Damage:"),
						fmt.Sprintf("    %.2f ^8(total incoming damage)", damage),
						fmt.Sprintf("    * %.2f ^8(from resistance, applies before armour)", portionArmour/100),
					)
				} else if portionArmour < 100 {
					actor.Breakdown.AddLine("DamageReduction",
						fmt.Sprintf("Enemy Hit Damage: %.2f ^8(total incoming damage)", damage),
						fmt.Sprintf("Portion mitigated by Armour: %.2f%%", portionArmour),
					)
				} else {
					actor.Breakdown.AddLine("DamageReduction",
						fmt.Sprintf("Enemy Hit Damage: %.2f ^8(total incoming damage)", damage),
					)
				}
				actor.Breakdown.AddLine(string(damageType+"DamageReduction"), fmt.Sprintf("Reduction from Armour: %.2f%%", armourReduct))
			}
		}
		takenMult := actor.Output[string(damageType+"TakenHitMult")]
		if damageCategoryConfig == "Melee" || damageCategoryConfig == "Projectile" {
			takenMult = actor.Output[string(damageType+"AttackTakenHitMult")]
		} else if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" {
			takenMult = actor.Output[string(damageType+"SpellTakenHitMult")]
		} else if damageCategoryConfig == "Average" {
			takenMult = (actor.Output[string(damageType+"SpellTakenHitMult")] + actor.Output[string(damageType+"AttackTakenHitMult")]) / 2
		}
		actor.Output[string(damageType+"BaseTakenHitMult")] = (1 - (resist-enemyPen)/100) * takenMult
		takenMultReflect := actor.Output[string(damageType+"TakenReflect")]
		finalReflect := (1 - (resist-enemyPen)/100) * takenMultReflect
		actor.Output[string(damageType+"TakenHit")] = max(actor.Output[string(damageType+"TakenDamage")]*(1-(resist-enemyPen)/100)+takenFlat, 0) * takenMult
		actor.Output[string(damageType+"TakenHitMult")] = utils.Ternary(actor.Output[string(damageType+"TakenDamage")] > 0, actor.Output[string(damageType+"TakenHit")]/actor.Output[string(damageType+"TakenDamage")], 0)
		actor.Output["totalTakenHit"] = actor.Output["totalTakenHit"] + actor.Output[string(damageType+"TakenHit")]
		if actor.Output["AnyTakenReflect"] != 0 {
			actor.Output[string(damageType+"TakenReflectMult")] = finalReflect
		}
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("TakenHitMult",
				fmt.Sprintf("Resistance: %.2f", 1-resist/100),
			)
			if enemyPen > 0 {
				actor.Breakdown.AddLine(string(damageType+"TakenHitMult"), fmt.Sprintf("Enemy Pen: %.2f", enemyPen))
			}
			actor.Breakdown.AddLine(string(damageType+"TakenHitMult"), fmt.Sprintf("+ Flat: %.3f", takenFlat))
			actor.Breakdown.AddLine(string(damageType+"TakenHitMult"), fmt.Sprintf("x Taken: %.3f", takenMult))
			actor.Breakdown.AddLine(string(damageType+"TakenHitMult"), fmt.Sprintf("= %.3f", actor.Output[string(damageType+"TakenHitMult")]))
			actor.Breakdown.AddLine("TakenHit",
				fmt.Sprintf("Final %s Damage taken:", damageType),
				fmt.Sprintf("%.1f incoming damage", actor.Output[string(damageType+"TakenDamage")]),
				fmt.Sprintf("x %.3f damage mult", actor.Output[string(damageType+"TakenHitMult")]),
				fmt.Sprintf("= %.1f", actor.Output[string(damageType+"TakenHit")]),
			)
			actor.Breakdown.AddRow("totalTakenHit", map[string]string{
				"type":     fmt.Sprintf("%s", damageType),
				"incoming": fmt.Sprintf("%.1f incoming damage", actor.Output[string(damageType+"TakenDamage")]),
				"mult":     fmt.Sprintf("x %.3f damage mult", actor.Output[string(damageType+"TakenHitMult")]),
				"value":    fmt.Sprintf("%.2f", actor.Output[string(damageType+"TakenHit")]),
			})
			if actor.Output["AnyTakenReflect"] != 0 {
				actor.Breakdown.AddLine("TakenReflectMult",
					fmt.Sprintf("Resistance: %.3f", 1-resist/100),
				)
				if enemyPen > 0 {
					actor.Breakdown.AddLine(string(damageType+"TakenReflectMult"), fmt.Sprintf("Enemy Pen: %.2f", enemyPen))
				}
				actor.Breakdown.AddLine(string(damageType+"TakenReflectMult"), fmt.Sprintf("Taken: %.3f", takenMultReflect))
				actor.Breakdown.AddLine(string(damageType+"TakenReflectMult"), fmt.Sprintf("= %.3f", finalReflect))
			}
		}
	}
	// Life Recoverable
	actor.Output["LifeRecoverable"] = actor.Output["LifeUnreserved"]
	if environment.Build.GetBooleanOption("conditionLowLife") {
		actor.Output["LifeRecoverable"] = min(actor.Output["Life"]*data.LowPoolThreshold, actor.Output["LifeUnreserved"])
		if actor.Output["LifeRecoverable"] < actor.Output["LifeUnreserved"] {
			actor.Output["CappingLife"] = 1
		}
	}
	// Prevented life loss (Petrified Blood)
	{
		actor.Output["preventedLifeLoss"] = modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfPrevented")
		portionLife := float64(1)
		if !environment.Build.GetBooleanOption("conditionLowLife") {
			// portion of life that is lowlife
			portionLife = min(actor.Output["Life"]*data.LowPoolThreshold/actor.Output["LifeRecoverable"], 1)
			actor.Output["preventedLifeLoss"] = actor.Output["preventedLifeLoss"] * portionLife
		}
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("preventedLifeLoss",
				fmt.Sprintf("Total life protected:"),
			)
			if portionLife != 1 {
				actor.Breakdown.AddLine("preventedLifeLoss", fmt.Sprintf("%.2f ^8(initial portion taken from petrified blood)", actor.Output["preventedLifeLoss"]/portionLife/100))
				actor.Breakdown.AddLine("preventedLifeLoss", fmt.Sprintf("* %.2f ^8(portion of life on low life)", portionLife))
				actor.Breakdown.AddLine("preventedLifeLoss", fmt.Sprintf("= %.2f ^8(final portion taken from petrified blood)", actor.Output["preventedLifeLoss"]/100))
				actor.Breakdown.AddLine("preventedLifeLoss", fmt.Sprintf(""))
			} else {
				actor.Breakdown.AddLine("preventedLifeLoss", fmt.Sprintf("%.2f ^8(portion taken from petrified blood)", actor.Output["preventedLifeLoss"]/100))
			}
			actor.Breakdown.AddLine("preventedLifeLoss", fmt.Sprintf("%.2f ^8(portion taken from life)", 1-actor.Output["preventedLifeLoss"]/100))
		}
	}
	// Energy Shield bypass
	actor.Output["AnyBypass"] = 0
	actor.Output["MinimumBypass"] = 100
	for _, damageType := range data.DamageType("").Values() {
		if modDB.Flag(nil, "UnblockedDamageDoesBypassES") {
			actor.Output[string(damageType+"EnergyShieldBypass")] = 100
			actor.Output["AnyBypass"] = 1
		} else {
			actor.Output[string(damageType+"EnergyShieldBypass")] = modDB.Sum(mod.TypeBase, nil, string(damageType+"EnergyShieldBypass"))
			if actor.Output[string(damageType+"EnergyShieldBypass")] != 0 {
				actor.Output["AnyBypass"] = 1
			}
			if damageType == "Chaos" {
				if !modDB.Flag(nil, "ChaosNotBypassEnergyShield") {
					actor.Output[string(damageType+"EnergyShieldBypass")] = actor.Output[string(damageType+"EnergyShieldBypass")] + 100
				} else {
					actor.Output["AnyBypass"] = 1
				}
			}
		}
		actor.Output[string(damageType+"EnergyShieldBypass")] = max(min(actor.Output[string(damageType+"EnergyShieldBypass")], 100), 0)
		actor.Output["MinimumBypass"] = min(actor.Output["MinimumBypass"], actor.Output[string(damageType+"EnergyShieldBypass")])
	}

	actor.Output["ehpSectionAnySpecificTypes"] = 0
	// Mind over Matter
	actor.Output["OnlySharedMindOverMatter"] = 0
	actor.Output["AnySpecificMindOverMatter"] = 0
	actor.Output["sharedMindOverMatter"] = min(modDB.Sum(mod.TypeBase, nil, "DamageTakenFromManaBeforeLife"), 100)
	if actor.Output["sharedMindOverMatter"] > 0 {
		actor.Output["OnlySharedMindOverMatter"] = 1
		sourcePool := max(actor.Output["ManaUnreserved"], 0)
		manatext := "unreserved mana"
		if modDB.Flag(nil, "EnergyShieldProtectsMana") && actor.Output["MinimumBypass"] < 100 {
			manatext = manatext + " + non-bypassed energy shield"
			if actor.Output["MinimumBypass"] > 0 {
				manaProtected := actor.Output["EnergyShieldRecoveryCap"] / (1 - actor.Output["MinimumBypass"]/100) * (actor.Output["MinimumBypass"] / 100)
				sourcePool = max(sourcePool-manaProtected, 0) + min(sourcePool, manaProtected)/(actor.Output["MinimumBypass"]/100)
			} else {
				sourcePool = sourcePool + actor.Output["EnergyShieldRecoveryCap"]
			}
		}
		poolProtected := sourcePool / (actor.Output["sharedMindOverMatter"] / 100) * (1 - actor.Output["sharedMindOverMatter"]/100)
		if actor.Output["sharedMindOverMatter"] >= 100 {
			poolProtected = math.MaxFloat64
			actor.Output["sharedManaEffectiveLife"] = actor.Output["LifeRecoverable"] + sourcePool
		} else {
			actor.Output["sharedManaEffectiveLife"] = max(actor.Output["LifeRecoverable"]-poolProtected, 0) + min(actor.Output["LifeRecoverable"], poolProtected)/(1-actor.Output["sharedMindOverMatter"]/100)
		}
		if actor.Breakdown != nil {
			if actor.Output["sharedMindOverMatter"] != 0 {
				actor.Breakdown.AddLine("sharedMindOverMatter",
					fmt.Sprintf("Total life protected:"),
					fmt.Sprintf("%.2f ^8(%s)", sourcePool, manatext),
					fmt.Sprintf("/ %.2f ^8(portion taken from mana)", actor.Output["sharedMindOverMatter"]/100),
					fmt.Sprintf("x %.2f ^8(portion taken from life)", 1-actor.Output["sharedMindOverMatter"]/100),
					fmt.Sprintf("= %.2f", poolProtected),
					fmt.Sprintf("Effective life: %.2f", actor.Output["sharedManaEffectiveLife"]),
				)
			}
		}
	} else {
		actor.Output["sharedManaEffectiveLife"] = actor.Output["LifeRecoverable"]
	}
	for _, damageType := range data.DamageType("").Values() {
		actor.Output[string(damageType+"MindOverMatter")] = min(modDB.Sum(mod.TypeBase, nil, string(damageType+"DamageTakenFromManaBeforeLife")), 100-actor.Output["sharedMindOverMatter"])
		if actor.Output[string(damageType+"MindOverMatter")] > 0 || (actor.Output[string(damageType+"EnergyShieldBypass")] > actor.Output["MinimumBypass"] && actor.Output["sharedMindOverMatter"] > 0) {
			MindOverMatter := actor.Output[string(damageType+"MindOverMatter")] + actor.Output["sharedMindOverMatter"]
			actor.Output["ehpSectionAnySpecificTypes"] = 1
			actor.Output["AnySpecificMindOverMatter"] = 1
			actor.Output["OnlySharedMindOverMatter"] = 0
			sourcePool := max(actor.Output["ManaUnreserved"], 0)
			manatext := "unreserved mana"
			if modDB.Flag(nil, "EnergyShieldProtectsMana") && actor.Output[string(damageType+"EnergyShieldBypass")] < 100 {
				manatext = manatext + " + non-bypassed energy shield"
				if actor.Output[string(damageType+"EnergyShieldBypass")] > 0 {
					manaProtected := actor.Output["EnergyShieldRecoveryCap"] / (1 - actor.Output[string(damageType+"EnergyShieldBypass")]/100) * (actor.Output[string(damageType+"EnergyShieldBypass")] / 100)
					sourcePool = max(sourcePool-manaProtected, 0) + min(sourcePool, manaProtected)/(actor.Output[string(damageType+"EnergyShieldBypass")]/100)
				} else {
					sourcePool = sourcePool + actor.Output["EnergyShieldRecoveryCap"]
				}
			}
			poolProtected := sourcePool / (MindOverMatter / 100) * (1 - MindOverMatter/100)
			if MindOverMatter >= 100 {
				poolProtected = math.MaxFloat64
				actor.Output[string(damageType+"ManaEffectiveLife")] = actor.Output["LifeRecoverable"] + sourcePool
			} else {
				actor.Output[string(damageType+"ManaEffectiveLife")] = max(actor.Output["LifeRecoverable"]-poolProtected, 0) + min(actor.Output["LifeRecoverable"], poolProtected)/(1-MindOverMatter/100)
			}
			if actor.Breakdown != nil {
				if actor.Output[string(damageType+"MindOverMatter")] != 0 {
					actor.Breakdown.AddLine("MindOverMatter",
						fmt.Sprintf("Total life protected:"),
						fmt.Sprintf("%.2f ^8(%s)", sourcePool, manatext),
						fmt.Sprintf("/ %.2f ^8(portion taken from mana)", MindOverMatter/100),
						fmt.Sprintf("x %.2f ^8(portion taken from life)", 1-MindOverMatter/100),
						fmt.Sprintf("= %.2f", poolProtected),
						fmt.Sprintf("Effective life: %.2f", actor.Output[string(damageType+"ManaEffectiveLife")]),
					)
				}
			}
		} else {
			actor.Output[string(damageType+"ManaEffectiveLife")] = actor.Output["sharedManaEffectiveLife"]
		}
	}
	// Guard
	actor.Output["AnyGuard"] = 0
	actor.Output["sharedGuardAbsorbRate"] = min(modDB.Sum(mod.TypeBase, nil, "GuardAbsorbRate"), 100)
	if actor.Output["sharedGuardAbsorbRate"] > 0 {
		actor.Output["OnlySharedGuard"] = 1
		actor.Output["sharedGuardAbsorb"] = calclib.Val(modDB, "GuardAbsorbLimit")
		lifeProtected := actor.Output["sharedGuardAbsorb"] / (actor.Output["sharedGuardAbsorbRate"] / 100) * (1 - actor.Output["sharedGuardAbsorbRate"]/100)
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("sharedGuardAbsorb",
				fmt.Sprintf("Total life protected:"),
				fmt.Sprintf("%.2f ^8(guard limit)", actor.Output["sharedGuardAbsorb"]),
				fmt.Sprintf("/ %.2f ^8(portion taken from guard)", actor.Output["sharedGuardAbsorbRate"]/100),
				fmt.Sprintf("x %.2f ^8(portion taken from life and energy shield)", 1-actor.Output["sharedGuardAbsorbRate"]/100),
				fmt.Sprintf("= %.2f", lifeProtected),
			)
		}
	}
	for _, damageType := range data.DamageType("").Values() {
		actor.Output[string(damageType+"GuardAbsorbRate")] = min(modDB.Sum(mod.TypeBase, nil, string(damageType+"GuardAbsorbRate")), 100)
		if actor.Output[string(damageType+"GuardAbsorbRate")] > 0 {
			actor.Output["ehpSectionAnySpecificTypes"] = 1
			actor.Output["AnyGuard"] = 1
			actor.Output["OnlySharedGuard"] = 0
			actor.Output[string(damageType+"GuardAbsorb")] = calclib.Val(modDB, string(damageType+"GuardAbsorbLimit"))
			lifeProtected := actor.Output[string(damageType+"GuardAbsorb")] / (actor.Output[string(damageType+"GuardAbsorbRate")] / 100) * (1 - actor.Output[string(damageType+"GuardAbsorbRate")]/100)
			if actor.Breakdown != nil {
				actor.Breakdown.AddLine("GuardAbsorb",
					fmt.Sprintf("Total life protected:"),
					fmt.Sprintf("%.2f ^8(guard limit)", actor.Output[string(damageType+"GuardAbsorb")]),
					fmt.Sprintf("/ %.2f ^8(portion taken from guard)", actor.Output[string(damageType+"GuardAbsorbRate")]/100),
					fmt.Sprintf("x %.2f ^8(portion taken from life and energy shield)", 1-actor.Output[string(damageType+"GuardAbsorbRate")]/100),
					fmt.Sprintf("= %.2f", lifeProtected),
				)
			}
		}
	}
	// aegis
	actor.Output["AnyAegis"] = 0
	actor.Output["sharedAegis"] = modDB.Max(nil, "AegisValue")
	actor.Output["sharedElementalAegis"] = modDB.Max(nil, "ElementalAegisValue")
	if actor.Output["sharedAegis"] > 0 {
		actor.Output["AnyAegis"] = 1
	}
	if actor.Output["sharedElementalAegis"] > 0 {
		actor.Output["ehpSectionAnySpecificTypes"] = 1
		actor.Output["AnyAegis"] = 1
	}
	for _, damageType := range data.DamageType("").Values() {
		aegisValue := modDB.Max(nil, string(damageType+"AegisValue"))
		if aegisValue > 0 {
			actor.Output["ehpSectionAnySpecificTypes"] = 1
			actor.Output["AnyAegis"] = 1
			actor.Output[string(damageType+"Aegis")] = aegisValue
		} else {
			actor.Output[string(damageType+"Aegis")] = 0
		}
		if isElemental[string(damageType)] {
			actor.Output[string(damageType+"AegisDisplay")] = actor.Output[string(damageType+"Aegis")] + actor.Output["sharedElementalAegis"]
		}
	}
	// frost shield
	{
		actor.Output["FrostShieldLife"] = modDB.Sum(mod.TypeBase, nil, "FrostGlobeHealth")
		actor.Output["FrostShieldDamageMitigation"] = modDB.Sum(mod.TypeBase, nil, "FrostGlobeDamageMitigation")

		lifeProtected := actor.Output["FrostShieldLife"] / (actor.Output["FrostShieldDamageMitigation"] / 100) * (1 - actor.Output["FrostShieldDamageMitigation"]/100)
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("FrostShieldLife",
				fmt.Sprintf("Total life protected:"),
				fmt.Sprintf("%.2f ^8(frost shield limit)", actor.Output["FrostShieldLife"]),
				fmt.Sprintf("/ %.2f ^8(portion taken from frost shield)", actor.Output["FrostShieldDamageMitigation"]/100),
				fmt.Sprintf("x %.2f ^8(portion taken from life and energy shield)", 1-actor.Output["FrostShieldDamageMitigation"]/100),
				fmt.Sprintf("= %.2f", lifeProtected),
			)
		}
	}
	// total pool
	for _, damageType := range data.DamageType("").Values() {
		actor.Output[string(damageType+"TotalPool")] = actor.Output[string(damageType+"ManaEffectiveLife")]
		manatext := "Mana"
		if actor.Output[string(damageType+"EnergyShieldBypass")] < 100 {
			if modDB.Flag(nil, "EnergyShieldProtectsMana") {
				manatext = manatext + " and non-bypassed Energy Shield"
			} else {
				if actor.Output[string(damageType+"EnergyShieldBypass")] > 0 {
					poolProtected := actor.Output["EnergyShieldRecoveryCap"] / (1 - actor.Output[string(damageType+"EnergyShieldBypass")]/100) * (actor.Output[string(damageType+"EnergyShieldBypass")] / 100)
					actor.Output[string(damageType+"TotalPool")] = max(actor.Output[string(damageType+"TotalPool")]-poolProtected, 0) + min(actor.Output[string(damageType+"TotalPool")], poolProtected)/(actor.Output[string(damageType+"EnergyShieldBypass")]/100)
				} else {
					actor.Output[string(damageType+"TotalPool")] = actor.Output[string(damageType+"TotalPool")] + actor.Output["EnergyShieldRecoveryCap"]
				}
			}
		}
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("TotalPool",
				fmt.Sprintf("Life: %.2f", actor.Output["LifeRecoverable"]),
			)
			if actor.Output[string(damageType+"ManaEffectiveLife")] != actor.Output["LifeRecoverable"] {
				actor.Breakdown.AddLine(string(damageType+"TotalPool"), fmt.Sprintf("%s through MoM: %.2f", manatext, actor.Output[string(damageType+"ManaEffectiveLife")]-actor.Output["LifeRecoverable"]))
			}
			if (!modDB.Flag(nil, "EnergyShieldProtectsMana")) && actor.Output[string(damageType+"EnergyShieldBypass")] < 100 {
				actor.Breakdown.AddLine(string(damageType+"TotalPool"), fmt.Sprintf("Non-bypassed Energy Shield: %.2f", actor.Output[string(damageType+"TotalPool")]-actor.Output[string(damageType+"ManaEffectiveLife")]))
			}
			actor.Breakdown.AddLine(string(damageType+"TotalPool"), fmt.Sprintf("TotalPool: %.2f", actor.Output[string(damageType+"TotalPool")]))
		}
	}

	// helper function that iteratively reduces pools until life hits 0 to determine the number of hits it would take with given damage to die
	var numberOfHitsToDie func(DamageIn map[string]float64) int
	numberOfHitsToDie = func(DamageIn map[string]float64) int {
		numHits := 0
		DamageIn["cycles"] = DamageIn["cycles"]
		if DamageIn["cycles"] == 0 {
			DamageIn["cycles"] = 1
		}

		// check damage in isnt 0 and that ward doesnt mitigate all damage
		for _, damageType := range data.DamageType("").Values() {
			numHits = numHits + int(DamageIn[string(damageType)])
		}
		if numHits == 0 {
			return math.MaxInt
		} else if modDB.Flag(nil, "WardNotBreak") && actor.Output["Ward"] > 0 && float64(numHits) < actor.Output["Ward"] {
			return math.MaxInt
		} else {
			numHits = 0
		}

		life := actor.Output["LifeRecoverable"]
		mana := actor.Output["ManaUnreserved"]
		energyShield := actor.Output["EnergyShieldRecoveryCap"]
		ward := actor.Output["Ward"]
		restoreWard := utils.Ternary(modDB.Flag(nil, "WardNotBreak"), ward, 0)
		// dont apply non-perma ward for speed up calcs as it wont zero it correctly per hit
		if (!modDB.Flag(nil, "WardNotBreak")) && DamageIn["cycles"] > 1 {
			ward = 0
			restoreWard = 0
		}
		frostShield := actor.Output["FrostShieldLife"]
		aegis := make(map[string]float64)
		aegis["shared"] = actor.Output["sharedAegis"]
		aegis["sharedElemental"] = actor.Output["sharedElementalAegis"]
		guard := make(map[string]float64)
		guard["shared"] = actor.Output["sharedGuardAbsorb"]
		for _, damageType := range data.DamageType("").Values() {
			aegis[string(damageType)] = actor.Output[string(damageType+"Aegis")]
			guard[string(damageType)] = actor.Output[string(damageType+"GuardAbsorb")]
			if DamageIn[string(damageType+"EnergyShieldBypass")] == 0 {
				DamageIn[string(damageType+"EnergyShieldBypass")] = actor.Output[string(damageType+"EnergyShieldBypass")]
			}

		}
		DamageIn["LifeLossBelowHalfLost"] = DamageIn["LifeLossBelowHalfLost"]
		DamageIn["WardBypass"] = utils.OrF(DamageIn["WardBypass"], modDB.Sum(mod.TypeBase, nil, "WardBypass"))

		itterationMultiplier := float64(1)
		maxHits := float64(data.EhpCalcMaxHitsToCalc) / DamageIn["cycles"]
		for life > 0 && float64(numHits) < maxHits {
			numHits = numHits + int(itterationMultiplier)
			Damage := make(map[data.DamageType]float64)
			for _, damageType := range data.DamageType("").Values() {
				Damage[damageType] = DamageIn[string(damageType)] * itterationMultiplier
			}
			if DamageIn["GainWhenHit"] != 0 && (itterationMultiplier > 1 || DamageIn["cycles"] > 1) {
				gainMult := itterationMultiplier * DamageIn["cycles"]
				life = min(life+DamageIn["LifeWhenHit"]*(gainMult-1), gainMult*(actor.Output["LifeRecoverable"]))
				mana = min(mana+DamageIn["ManaWhenHit"]*(gainMult-1), gainMult*(actor.Output["ManaUnreserved"]))
				energyShield = min(energyShield+DamageIn["EnergyShieldWhenHit"]*(gainMult-1), gainMult*actor.Output["EnergyShieldRecoveryCap"])
			}
			for _, damageType := range data.DamageType("").Values() {
				if Damage[damageType] > 0 {
					if frostShield > 0 {
						tempDamage := min(Damage[damageType]*actor.Output["FrostShieldDamageMitigation"]/100/itterationMultiplier, frostShield)
						frostShield = frostShield - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if aegis[string(damageType)] > 0 {
						tempDamage := min(Damage[damageType], aegis[string(damageType)])
						aegis[string(damageType)] = aegis[string(damageType)] - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if isElemental[string(damageType)] && aegis["sharedElemental"] > 0 {
						tempDamage := min(Damage[damageType], aegis["sharedElemental"])
						aegis["sharedElemental"] = aegis["sharedElemental"] - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if aegis["shared"] > 0 {
						tempDamage := min(Damage[damageType], aegis["shared"])
						aegis["shared"] = aegis["shared"] - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if guard[string(damageType)] > 0 {
						tempDamage := min(Damage[damageType]*actor.Output[string(damageType+"GuardAbsorbRate")]/100/itterationMultiplier, guard[string(damageType)])
						guard[string(damageType)] = guard[string(damageType)] - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if guard["shared"] > 0 {
						tempDamage := min(Damage[damageType]*actor.Output["sharedGuardAbsorbRate"]/100/itterationMultiplier, guard["shared"])
						guard["shared"] = guard["shared"] - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if ward > 0 {
						tempDamage := min(Damage[damageType]*(1-DamageIn["WardBypass"]/100), ward)
						ward = ward - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if energyShield > 0 && (!modDB.Flag(nil, "EnergyShieldProtectsMana")) && DamageIn[string(damageType+"EnergyShieldBypass")] < 100 {
						tempDamage := min(Damage[damageType]*(1-DamageIn[string(damageType+"EnergyShieldBypass")]/100), energyShield)
						energyShield = energyShield - tempDamage
						Damage[damageType] = Damage[damageType] - tempDamage
					}
					if (actor.Output["sharedMindOverMatter"] + actor.Output[string(damageType+"MindOverMatter")]) > 0 {
						MoMDamage := Damage[damageType] * min(actor.Output["sharedMindOverMatter"]+actor.Output[string(damageType+"MindOverMatter")], 100) / 100
						if modDB.Flag(nil, "EnergyShieldProtectsMana") && energyShield > 0 && DamageIn[string(damageType+"EnergyShieldBypass")] < 100 {
							tempDamage := min(MoMDamage*(1-DamageIn[string(damageType+"EnergyShieldBypass")]/100), energyShield)
							energyShield = energyShield - tempDamage
							MoMDamage = MoMDamage - tempDamage
							tempDamage2 := min(MoMDamage, mana)
							mana = mana - tempDamage2
							Damage[damageType] = Damage[damageType] - tempDamage - tempDamage2
						} else if mana > 0 {
							tempDamage := min(MoMDamage, mana)
							mana = mana - tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
					}
					if actor.Output["preventedLifeLoss"] > 0 {
						if DamageIn["LifeLossBelowHalfLost"] > 0 {
							actor.Output["LifeLossBelowHalfLost"] = actor.Output["LifeLossBelowHalfLost"] + Damage[damageType]*actor.Output["preventedLifeLoss"]/100
						}
						Damage[damageType] = Damage[damageType] * (1 - actor.Output["preventedLifeLoss"]/100)
					}
					life = life - Damage[damageType]
				}
			}
			if modDB.Flag(nil, "WardNotBreak") {
				ward = restoreWard
			} else if ward > 0 {
				ward = 0
			}
			if DamageIn["GainWhenHit"] != 0 && life > 0 {
				life = min(life+DamageIn["LifeWhenHit"], actor.Output["LifeRecoverable"])
				mana = min(mana+DamageIn["ManaWhenHit"], actor.Output["ManaUnreserved"])
				energyShield = min(energyShield+DamageIn["EnergyShieldWhenHit"], actor.Output["EnergyShieldRecoveryCap"])
			}
			itterationMultiplier = 1
			// To speed it up, run recurivly but speed up
			maxDepth := float64(data.EhpCalcMaxDepth)
			speedUp := float64(data.EhpCalcSpeedUp)
			if DamageIn["cyclesRan"] == 0 && life > 0 && DamageIn["cycles"] < maxDepth {
				DamageDown := make(map[string]float64)
				for _, damageType := range data.DamageType("").Values() {
					DamageDown[string(damageType)] = DamageIn[string(damageType)] * speedUp
				}
				DamageDown["cycles"] = DamageIn["cycles"] * speedUp
				itterationMultiplier = max((float64(numberOfHitsToDie(DamageDown)-1))*speedUp-1, 1)
				DamageIn["cyclesRan"] = 1
			}
		}
		if float64(numHits) >= maxHits {
			return math.MaxInt
		}
		return numHits
	}
	// number of damaging hits needed to be taken to die
	{
		DamageIn := make(map[string]float64)
		for _, damageType := range data.DamageType("").Values() {
			DamageIn[string(damageType)] = actor.Output[string(damageType+"TakenHit")]
		}
		actor.Output["NumberOfDamagingHits"] = float64(numberOfHitsToDie(DamageIn))
	}

	{
		DamageIn := make(map[string]float64)
		BlockChance := float64(0)
		blockEffect := float64(1)
		suppressChance := float64(0)
		suppressionEffect := float64(1)
		ExtraAvoidChance := float64(0)
		averageAvoidChance := float64(0)
		worstOf := environment.Build.GetNumberOption("EHPUnluckyWorstOf")
		if worstOf == 0 {
			worstOf = 1
		}
		// block effect
		if damageCategoryConfig == "Melee" {
			BlockChance = actor.Output["BlockChance"] / 100
		} else {
			BlockChance = actor.Output[damageCategoryConfig+"BlockChance"] / 100
		}
		// unlucky config to lower the value of block, dodge, evade etc for ehp
		if worstOf > 1 {
			BlockChance = BlockChance * BlockChance
			if worstOf == 4 {
				BlockChance = BlockChance * BlockChance
			}
		}
		blockEffect = (1 - BlockChance*actor.Output["BlockEffect"]/100)
		if !environment.Build.GetBooleanOption("DisableEHPGainOnBlock") {
			DamageIn["LifeWhenHit"] = actor.Output["LifeOnBlock"] * BlockChance
			DamageIn["ManaWhenHit"] = actor.Output["ManaOnBlock"] * BlockChance
			DamageIn["EnergyShieldWhenHit"] = actor.Output["EnergyShieldOnBlock"] * BlockChance
			if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" {
				DamageIn["EnergyShieldWhenHit"] = DamageIn["EnergyShieldWhenHit"] + actor.Output["EnergyShieldOnSpellBlock"]*BlockChance
			} else if damageCategoryConfig == "Average" {
				DamageIn["EnergyShieldWhenHit"] = DamageIn["EnergyShieldWhenHit"] + actor.Output["EnergyShieldOnSpellBlock"]/2*BlockChance
			}
		}
		// suppression
		if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" || damageCategoryConfig == "Average" {
			suppressChance = actor.Output["SpellSuppressionChance"] / 100
		}
		// unlucky config to lower the value of block, dodge, evade etc for ehp
		if worstOf > 1 {
			suppressChance = suppressChance * suppressChance
			if worstOf == 4 {
				suppressChance = suppressChance * suppressChance
			}
		}
		if damageCategoryConfig == "Average" {
			suppressChance = suppressChance / 2
		}
		suppressionEffect = 1 - suppressChance*actor.Output["SpellSuppressionEffect"]/100
		// extra avoid chance
		if damageCategoryConfig == "Projectile" || damageCategoryConfig == "SpellProjectile" {
			ExtraAvoidChance = ExtraAvoidChance + actor.Output["AvoidProjectilesChance"]
		} else if damageCategoryConfig == "Average" {
			ExtraAvoidChance = ExtraAvoidChance + actor.Output["AvoidProjectilesChance"]/2
		}
		// gain when hit (currently just gain on block)
		if !environment.Build.GetBooleanOption("DisableEHPGainOnBlock") {
			if DamageIn["LifeWhenHit"] != 0 || DamageIn["ManaWhenHit"] != 0 || DamageIn["EnergyShieldWhenHit"] != 0 {
				DamageIn["GainWhenHit"] = 1
			}
		}
		for _, damageType := range data.DamageType("").Values() {
			// Emperor's Vigilance (this needs to fail with divine flesh as it cant override it, hence the check for high bypass)
			if modDB.Flag(nil, "BlockedDamageDoesntBypassES") && actor.Output[string(damageType+"EnergyShieldBypass")] < 100 && damageType != "Chaos" {
				DamageIn[string(damageType+"EnergyShieldBypass")] = actor.Output[string(damageType+"EnergyShieldBypass")] * (1 - BlockChance)
			}
			AvoidChance := min(actor.Output[string("Avoid"+damageType+"DamageChance")]+ExtraAvoidChance, data.AvoidChanceCap)
			// unlucky config to lower the value of block, dodge, evade etc for ehp
			if worstOf > 1 {
				AvoidChance = AvoidChance / 100 * AvoidChance
				if worstOf == 4 {
					AvoidChance = AvoidChance / 100 * AvoidChance
				}
			}
			averageAvoidChance = averageAvoidChance + AvoidChance
			DamageIn[string(damageType)] = actor.Output[string(damageType+"TakenHit")] * (blockEffect * suppressionEffect * (1 - AvoidChance/100))
		}
		// petrified blood degen initialisation
		if actor.Output["preventedLifeLoss"] > 0 {
			actor.Output["LifeLossBelowHalfLost"] = 0
			DamageIn["LifeLossBelowHalfLost"] = modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfLost") / 100
		}
		actor.Output["NumberOfMitigatedDamagingHits"] = float64(numberOfHitsToDie(DamageIn))
		averageAvoidChance = averageAvoidChance / 5
		actor.Output["ConfiguredDamageChance"] = 100 * (blockEffect * suppressionEffect * (1 - averageAvoidChance/100))
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("ConfiguredDamageChance",
				fmt.Sprintf("%.2f ^8(chance for block to fail)", 1-BlockChance),
			)
			if actor.Output["ShowBlockEffect"] != 0 {
				actor.Breakdown.AddLine("ConfiguredDamageChance", fmt.Sprintf("x %.2f ^8(block effect)", actor.Output["BlockEffect"]/100))
			}
			if suppressionEffect > 0 {
				actor.Breakdown.AddLine("ConfiguredDamageChance", fmt.Sprintf("x %.3f ^8(suppression effect)", suppressionEffect))
			}
			if averageAvoidChance > 0 {
				actor.Breakdown.AddLine("ConfiguredDamageChance", fmt.Sprintf("x %.2f ^8(chance for avoidance to fail)", 1-averageAvoidChance/100))
			}
			actor.Breakdown.AddLine("ConfiguredDamageChance", fmt.Sprintf("= %.1f%% ^8(of damage taken from a%s hit)", actor.Output["ConfiguredDamageChance"], utils.Ternary(damageCategoryConfig == "Average", "n ", " ")+damageCategoryConfig))
		}
	}
	// chance to not be hit
	{
		worstOf := environment.Build.GetNumberOption("EHPUnluckyWorstOf")
		if worstOf == 0 {
			worstOf = 1
		}
		actor.Output["MeleeNotHitChance"] = 100 - (1-actor.Output["MeleeEvadeChance"]/100)*(1-actor.Output["AttackDodgeChance"]/100)*100
		actor.Output["ProjectileNotHitChance"] = 100 - (1-actor.Output["ProjectileEvadeChance"]/100)*(1-actor.Output["AttackDodgeChance"]/100)*100
		actor.Output["SpellNotHitChance"] = 100 - (1-actor.Output["SpellDodgeChance"]/100)*100
		actor.Output["SpellProjectileNotHitChance"] = actor.Output["SpellNotHitChance"]
		actor.Output["AverageNotHitChance"] = (actor.Output["MeleeNotHitChance"] + actor.Output["ProjectileNotHitChance"] + actor.Output["SpellNotHitChance"] + actor.Output["SpellProjectileNotHitChance"]) / 4
		actor.Output["ConfiguredNotHitChance"] = actor.Output[damageCategoryConfig+"NotHitChance"]
		// unlucky config to lower the value of block, dodge, evade etc for ehp
		if worstOf > 1 {
			actor.Output["ConfiguredNotHitChance"] = actor.Output["ConfiguredNotHitChance"] / 100 * actor.Output["ConfiguredNotHitChance"]
			if worstOf == 4 {
				actor.Output["ConfiguredNotHitChance"] = actor.Output["ConfiguredNotHitChance"] / 100 * actor.Output["ConfiguredNotHitChance"]
			}
		}
		actor.Output["TotalNumberOfHits"] = actor.Output["NumberOfMitigatedDamagingHits"] / (1 - actor.Output["ConfiguredNotHitChance"]/100)
		if actor.Breakdown != nil {
			if damageCategoryConfig == "Melee" || damageCategoryConfig == "Projectile" {
				actor.Breakdown.AddLine("ConfiguredNotHitChance", fmt.Sprintf("%.2f ^8(chance for evasion to fail)", 1-actor.Output[damageCategoryConfig+"EvadeChance"]/100))
				actor.Breakdown.AddLine("ConfiguredNotHitChance", fmt.Sprintf("x %.2f ^8(chance for dodge to fail)", 1-actor.Output["AttackDodgeChance"]/100))
			} else if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" {
				actor.Breakdown.AddLine("ConfiguredNotHitChance", fmt.Sprintf("%.2f ^8(chance for dodge to fail)", 1-actor.Output["SpellDodgeChance"]/100))
			} else if damageCategoryConfig == "Average" {
				actor.Breakdown.AddLine("ConfiguredNotHitChance", fmt.Sprintf("%.2f ^8(chance for evasion to fail, only applies to the attack portion)", 1-(actor.Output["MeleeEvadeChance"]+actor.Output["ProjectileEvadeChance"])/2/100))
				actor.Breakdown.AddLine("ConfiguredNotHitChance", fmt.Sprintf("x%.2f ^8(chance for dodge to fail)", 1-(actor.Output["AttackDodgeChance"]+actor.Output["SpellDodgeChance"])/2/100))
			}
			if worstOf > 1 {
				actor.Breakdown.AddLine("ConfiguredNotHitChance", fmt.Sprintf("unlucky worst of %.2f", worstOf))
			}
			actor.Breakdown.AddLine("ConfiguredNotHitChance", fmt.Sprintf("= %.2f%% ^8(chance to be hit by a%s hit)", 100-actor.Output["ConfiguredNotHitChance"], utils.Ternary(damageCategoryConfig == "Average", "n ", " ")+damageCategoryConfig))
			actor.Breakdown.AddLine("TotalNumberOfHits",
				fmt.Sprintf("%.2f ^8(Number of mitigated hits)", actor.Output["NumberOfMitigatedDamagingHits"]),
				fmt.Sprintf("/ %.2f ^8(Chance to even be hit)", 1-actor.Output["ConfiguredNotHitChance"]/100),
				fmt.Sprintf("= %.2f ^8(total average number of hits you can take)", actor.Output["TotalNumberOfHits"]),
			)
		}
	}

	// effective hit pool
	actor.Output["TotalEHP"] = actor.Output["TotalNumberOfHits"] * actor.Output["totalEnemyDamageIn"]
	if actor.Breakdown != nil {
		actor.Breakdown.AddLine("TotalEHP",
			fmt.Sprintf("%.2f ^8(total average number of hits you can take)", actor.Output["TotalNumberOfHits"]),
			fmt.Sprintf("x %.2f ^8(total incoming damage)", actor.Output["totalEnemyDamageIn"]),
			fmt.Sprintf("= %.2f ^8(total damage you can take)", actor.Output["TotalEHP"]),
		)
	}
	// survival time
	{
		enemySkillTime := utils.OrF(environment.Build.GetNumberOption("enemySpeed"), 700)
		enemyActionSpeed := CalcActionSpeedMod(actor.Enemy)
		enemySkillTime = enemySkillTime / 1000 / enemyActionSpeed
		actor.Output["EHPsurvivalTime"] = actor.Output["TotalNumberOfHits"] * enemySkillTime
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("EHPsurvivalTime",
				fmt.Sprintf("%.2f ^8(total average number of hits you can take)", actor.Output["TotalNumberOfHits"]),
				fmt.Sprintf("x %.2f ^8enemy attack/cast time", enemySkillTime),
				fmt.Sprintf("= %.2f seconds ^8(total time it would take to die)", actor.Output["EHPsurvivalTime"]),
			)
		}
	}

	// petrified blood "degen"
	if actor.Output["preventedLifeLoss"] > 0 {
		LifeLossBelowHalfLost := modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfLost") / 100
		actor.Output["LifeLossBelowHalfLostMax"] = actor.Output["LifeLossBelowHalfLost"] * LifeLossBelowHalfLost / 4
		actor.Output["LifeLossBelowHalfLostAvg"] = actor.Output["LifeLossBelowHalfLost"] * LifeLossBelowHalfLost / (actor.Output["EHPsurvivalTime"] + 4)
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("LifeLossBelowHalfLostMax",
				fmt.Sprintf("%.2f ^8(total damage prevented by petrified blood)", actor.Output["LifeLossBelowHalfLost"]),
				fmt.Sprintf("* %.2f ^8(percent of damage taken)", LifeLossBelowHalfLost),
				fmt.Sprintf("/ %d ^8(over 4 seconds)", 4),
				fmt.Sprintf("= %.2f per second", actor.Output["LifeLossBelowHalfLostMax"]),
			)
			actor.Breakdown.AddLine("LifeLossBelowHalfLostAvg",
				fmt.Sprintf("%.2f ^8(total damage prevented by petrified blood)", actor.Output["LifeLossBelowHalfLost"]),
				fmt.Sprintf("* %.2f ^8(percent of damage taken)", LifeLossBelowHalfLost),
				fmt.Sprintf("/ %.2f ^8(total time of the degen (survival time + 4))", actor.Output["EHPsurvivalTime"]+4),
				fmt.Sprintf("= %.2f per second", actor.Output["LifeLossBelowHalfLostAvg"]),
			)
		}
	}

	// effective health pool vs dots
	for _, damageType := range data.DamageType("").Values() {
		actor.Output[string(damageType+"DotEHP")] = actor.Output[string(damageType+"TotalPool")] / actor.Output[string(damageType+"TakenDotMult")]
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine(string(damageType+"DotEHP"),
				fmt.Sprintf("Total Pool: %.2f", actor.Output[string(damageType+"TotalPool")]),
				fmt.Sprintf("Dot Damage Taken modifier: %.2f", actor.Output[string(damageType+"TakenDotMult")]),
				fmt.Sprintf("Total Effective Dot Pool: %.2f", actor.Output[string(damageType+"DotEHP")]),
			)
		}
	}

	// Degens
	for _, damageType := range data.DamageType("").Values() {
		baseVal := modDB.Sum(mod.TypeBase, nil, string(damageType+"Degen"))
		if baseVal > 0 {
			total := baseVal * actor.Output[string(damageType+"TakenDotMult")]
			actor.Output[string(damageType+"Degen")] = total
			actor.Output["TotalDegen"] = (actor.Output["TotalDegen"]) + total
			if actor.Breakdown != nil {
				actor.Breakdown.AddCol("TotalDegen",
					BCol{Label: "Type", Key: "type"},
					BCol{Label: "Base", Key: "base"},
					BCol{Label: "Multiplier", Key: "mult"},
					BCol{Label: "Total", Key: "total"},
				)
				actor.Breakdown.AddRow("TotalDegen", map[string]string{
					"type":  string(damageType),
					"base":  fmt.Sprintf("%.1f", baseVal),
					"mult":  fmt.Sprintf("x %.2f", actor.Output[string(damageType+"TakenDotMult")]),
					"total": fmt.Sprintf("%.1f", total),
				})
				actor.Breakdown.AddCol(string(damageType+"Degen"),
					BCol{Label: "Type", Key: "type"},
					BCol{Label: "Base", Key: "base"},
					BCol{Label: "Multiplier", Key: "mult"},
					BCol{Label: "Total", Key: "total"},
				)
				actor.Breakdown.AddRow(string(damageType+"Degen"), map[string]string{
					"type":  string(damageType),
					"base":  fmt.Sprintf("%.1f", baseVal),
					"mult":  fmt.Sprintf("x %.2f", actor.Output[string(damageType+"TakenDotMult")]),
					"total": fmt.Sprintf("%.1f", total),
				})
			}
		}
	}
	if actor.Output["TotalDegen"] != 0 {
		actor.Output["NetLifeRegen"] = actor.Output["LifeRegen"]
		actor.Output["NetManaRegen"] = actor.Output["ManaRegen"]
		actor.Output["NetEnergyShieldRegen"] = actor.Output["EnergyShieldRegen"]
		totalLifeDegen := float64(0)
		totalManaDegen := float64(0)
		totalEnergyShieldDegen := float64(0)
		if actor.Breakdown != nil {
			actor.Breakdown.SetLabel("NetLifeRegen", "Total Life Degen")
			actor.Breakdown.AddCol("NetLifeRegen",
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "Degen", Key: "degen"},
			)
			actor.Breakdown.SetLabel("NetManaRegen", "Total Mana Degen")
			actor.Breakdown.AddCol("NetManaRegen",
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "Degen", Key: "degen"},
			)
			actor.Breakdown.SetLabel("NetEnergyShieldRegen", "Total Energy Shield Degen")
			actor.Breakdown.AddCol("NetEnergyShieldRegen",
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "Degen", Key: "degen"},
			)
		}
		for _, damageType := range data.DamageType("").Values() {
			if actor.Output[string(damageType+"Degen")] != 0 {
				energyShieldDegen := float64(0)
				lifeDegen := float64(0)
				manaDegen := float64(0)
				takenFromMana := actor.Output[string(damageType+"MindOverMatter")] + actor.Output["sharedMindOverMatter"]
				if actor.Output["EnergyShieldRegen"] > 0 {
					if modDB.Flag(nil, "EnergyShieldProtectsMana") {
						lifeDegen = actor.Output[string(damageType+"Degen")] * (1 - takenFromMana/100)
						energyShieldDegen = actor.Output[string(damageType+"Degen")] * (1 - actor.Output[string(damageType+"EnergyShieldBypass")]/100) * (takenFromMana / 100)
					} else {
						lifeDegen = actor.Output[string(damageType+"Degen")] * (actor.Output[string(damageType+"EnergyShieldBypass")] / 100) * (1 - takenFromMana/100)
						energyShieldDegen = actor.Output[string(damageType+"Degen")] * (1 - actor.Output[string(damageType+"EnergyShieldBypass")]/100)
					}
					manaDegen = actor.Output[string(damageType+"Degen")] * (actor.Output[string(damageType+"EnergyShieldBypass")] / 100) * (takenFromMana / 100)
				} else {
					lifeDegen = actor.Output[string(damageType+"Degen")] * (1 - takenFromMana/100)
					manaDegen = actor.Output[string(damageType+"Degen")] * (takenFromMana / 100)
				}
				totalLifeDegen = totalLifeDegen + lifeDegen
				totalManaDegen = totalManaDegen + manaDegen
				totalEnergyShieldDegen = totalEnergyShieldDegen + energyShieldDegen
				if actor.Breakdown != nil {
					actor.Breakdown.AddRow("NetLifeRegen", map[string]string{
						"type":  fmt.Sprintf("%s", damageType),
						"degen": fmt.Sprintf("%.2f", lifeDegen),
					})
					actor.Breakdown.AddRow("NetManaRegen", map[string]string{
						"type":  fmt.Sprintf("%s", damageType),
						"degen": fmt.Sprintf("%.2f", manaDegen),
					})
					actor.Breakdown.AddRow("NetEnergyShieldRegen", map[string]string{
						"type":  fmt.Sprintf("%s", damageType),
						"degen": fmt.Sprintf("%.2f", energyShieldDegen),
					})
				}
			}
		}
		actor.Output["NetLifeRegen"] = actor.Output["NetLifeRegen"] - totalLifeDegen
		actor.Output["NetManaRegen"] = actor.Output["NetManaRegen"] - totalManaDegen
		actor.Output["NetEnergyShieldRegen"] = actor.Output["NetEnergyShieldRegen"] - totalEnergyShieldDegen
		actor.Output["TotalNetRegen"] = actor.Output["NetLifeRegen"] + actor.Output["NetManaRegen"] + actor.Output["NetEnergyShieldRegen"]
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine("NetLifeRegen", fmt.Sprintf("%.1f ^8(total life regen)", actor.Output["LifeRegen"]))
			actor.Breakdown.AddLine("NetLifeRegen", fmt.Sprintf("- %.1f ^8(total life degen)", totalLifeDegen))
			actor.Breakdown.AddLine("NetLifeRegen", fmt.Sprintf("= %.1f", actor.Output["NetLifeRegen"]))
			actor.Breakdown.AddLine("NetManaRegen", fmt.Sprintf("%.1f ^8(total mana regen)", actor.Output["ManaRegen"]))
			actor.Breakdown.AddLine("NetManaRegen", fmt.Sprintf("- %.1f ^8(total mana degen)", totalManaDegen))
			actor.Breakdown.AddLine("NetManaRegen", fmt.Sprintf("= %.1f", actor.Output["NetManaRegen"]))
			actor.Breakdown.AddLine("NetEnergyShieldRegen", fmt.Sprintf("%.1f ^8(total energy shield regen)", actor.Output["EnergyShieldRegen"]))
			actor.Breakdown.AddLine("NetEnergyShieldRegen", fmt.Sprintf("- %.1f ^8(total energy shield degen)", totalEnergyShieldDegen))
			actor.Breakdown.AddLine("NetEnergyShieldRegen", fmt.Sprintf("= %.1f", actor.Output["NetEnergyShieldRegen"]))
			actor.Breakdown.AddLine("TotalNetRegen",
				fmt.Sprintf("Net Life Regen: %.1f", actor.Output["NetLifeRegen"]),
				fmt.Sprintf("+ Net Mana Regen: %.1f", actor.Output["NetManaRegen"]),
				fmt.Sprintf("+ Net Energy Shield Regen: %.1f", actor.Output["NetEnergyShieldRegen"]),
				fmt.Sprintf("= Total Net Regen: %.1f", actor.Output["TotalNetRegen"]),
			)
		}
	}
	// maximum hit taken
	// this is not done yet, using old max hit taken
	// fix total pools, as they arnt used anymore
	for _, damageType := range data.DamageType("").Values() {
		// base + petrified blood
		if actor.Output["preventedLifeLoss"] > 0 {
			actor.Output[string(damageType+"TotalPool")] = actor.Output[string(damageType+"TotalPool")] / (1 - actor.Output["preventedLifeLoss"]/100)
		}
		// ward
		wardBypass := modDB.Sum(mod.TypeBase, nil, "WardBypass")
		if wardBypass > 0 {
			poolProtected := actor.Output["Ward"] / (1 - wardBypass/100) * (wardBypass / 100)
			sourcePool := actor.Output[string(damageType+"TotalPool")]
			sourcePool = max(sourcePool-poolProtected, 0) + min(sourcePool, poolProtected)/(wardBypass/100)
			actor.Output[string(damageType+"TotalPool")] = sourcePool
		} else {
			actor.Output[string(damageType+"TotalPool")] = actor.Output[string(damageType+"TotalPool")] + actor.Output["Ward"]
		}
		// aegis
		actor.Output[string(damageType+"TotalHitPool")] = actor.Output[string(damageType+"TotalPool")] + actor.Output[string(damageType+"Aegis")] + actor.Output[string(damageType+"sharedAegis")] + utils.Ternary(isElemental[string(damageType)], actor.Output[string(damageType+"sharedElementalAegis")], 0)
		// guardskill
		GuardAbsorbRate := actor.Output["sharedGuardAbsorbRate"] + actor.Output[string(damageType+"GuardAbsorbRate")]
		if GuardAbsorbRate > 0 {
			GuardAbsorb := actor.Output["sharedGuardAbsorb"] + actor.Output[string(damageType+"GuardAbsorb")]
			if GuardAbsorbRate >= 100 {
				actor.Output[string(damageType+"TotalHitPool")] = actor.Output[string(damageType+"TotalHitPool")] + GuardAbsorb
			} else {
				poolProtected := GuardAbsorb / (GuardAbsorbRate / 100) * (1 - GuardAbsorbRate/100)
				actor.Output[string(damageType+"TotalHitPool")] = max(actor.Output[string(damageType+"TotalHitPool")]-poolProtected, 0) + min(actor.Output[string(damageType+"TotalHitPool")], poolProtected)/(1-GuardAbsorbRate/100)
			}
		}
		// frost shield
		if actor.Output["FrostShieldLife"] > 0 {
			poolProtected := actor.Output["FrostShieldLife"] / (actor.Output["FrostShieldDamageMitigation"] / 100) * (1 - actor.Output["FrostShieldDamageMitigation"]/100)
			actor.Output[string(damageType+"TotalHitPool")] = max(actor.Output[string(damageType+"TotalHitPool")]-poolProtected, 0) + min(actor.Output[string(damageType+"TotalHitPool")], poolProtected)/(1-actor.Output["FrostShieldDamageMitigation"]/100)
		}
	}
	for _, damageType := range data.DamageType("").Values() {
		if actor.Breakdown != nil {
			actor.Breakdown.SetLabel(string(damageType+"MaximumHitTaken"), "Maximum Hit Taken (uses lowest value)")
			actor.Breakdown.AddCol(string(damageType+"MaximumHitTaken"),
				BCol{Label: "Type", Key: "type"},
				BCol{Label: "TotalPool", Key: "pool"},
				BCol{Label: "Taken", Key: "taken"},
				BCol{Label: "Final", Key: "final"},
			)
		}
		actor.Output[string(damageType+"MaximumHitTaken")] = math.MaxFloat64
		for _, damageConvertedType := range data.DamageType("").Values() {
			if actor.DamageShiftTable[damageType][damageConvertedType] > 0 {
				hitTaken := actor.Output[string(damageConvertedType+"TotalHitPool")] / (actor.DamageShiftTable[damageType][damageConvertedType] / 100) / actor.Output[string(damageConvertedType+"BaseTakenHitMult")]
				if hitTaken < actor.Output[string(damageType+"MaximumHitTaken")] {
					actor.Output[string(damageType+"MaximumHitTaken")] = hitTaken
				}
				if actor.Breakdown != nil {
					actor.Breakdown.AddRow(string(damageType+"MaximumHitTaken"), map[string]string{
						"type":  fmt.Sprintf("%.2f%% as %s", actor.DamageShiftTable[damageType][damageConvertedType], damageConvertedType),
						"pool":  fmt.Sprintf("x %.2f", actor.Output[string(damageConvertedType+"TotalHitPool")]),
						"taken": fmt.Sprintf("/ %.2f", actor.Output[string(damageConvertedType+"BaseTakenHitMult")]),
						"final": fmt.Sprintf("x %.0f", hitTaken),
					})
				}
			}
		}
		if actor.Breakdown != nil {
			actor.Breakdown.AddLine(string(damageType+"MaximumHitTaken"), fmt.Sprintf("Total Pool: %.2f", actor.Output[string(damageType+"TotalHitPool")]))
			actor.Breakdown.AddLine(string(damageType+"MaximumHitTaken"), fmt.Sprintf("Taken Mult: %.2f", actor.Output[string(damageType+"TotalHitPool")]/actor.Output[string(damageType+"MaximumHitTaken")]))
			actor.Breakdown.AddLine(string(damageType+"MaximumHitTaken"), fmt.Sprintf("Maximum hit you can take: %.0f", actor.Output[string(damageType+"MaximumHitTaken")]))
		}
	}
}
