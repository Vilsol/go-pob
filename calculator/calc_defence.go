package calculator

import (
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
	/*
		local enemyDB = actor.enemy.modDB
		local output = actor.output
		local breakdown = actor.breakdown

		local condList = modDB.conditions
	*/

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
		/*
			TODO Breakdown
			if breakdown != nil {
				breakdown[elem+"Resist"] = {
					"Min: "+min+"%",
					"Max: "+max+"%",
					"Total: "+total+"%",
				}
				breakdown["Totem"+elem+"Resist"] = {
					"Min: "+min+"%",
					"Max: "+totemMax+"%",
					"Total: "+totemTotal+"%",
				}
			}
		*/
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
	/*
		TODO Breakdown
		if breakdown != nil {
			breakdown.BlockChance = {
				"Base: "+baseBlockChance+"%",
				"Max: "+actor.Output["BlockChanceMax"]+"%",
				"Total: "+actor.Output["BlockChance"]+actor.Output["BlockChanceOverCap"]+"%",
			}
			breakdown.SpellBlockChance = {
				"Max: "+actor.Output["SpellBlockChanceMax"]+"%",
				"Total: "+actor.Output["SpellBlockChance"]+actor.Output["SpellBlockChanceOverCap"]+"%",
			}
		}
	*/

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
		/*
			// TODO Armour to ES Recharge conversion from Armour and Energy Shield Mastery
			multiplier := (modDB.Max(nil, "ImprovedArmourAppliesToEnergyShieldRecharge") or 100) / 100
			for _, value in ipairs(modDB.Tabulate("INC", nil, "Armour", "ArmourAndEvasion", "Defences")) {
				mod := value.mod
				modifiers := calcLib.getConvertedModTags(mod, multiplier)
				modDB.NewMod("EnergyShieldRecharge", "INC", m_floor(mod.value * multiplier), mod.source, mod.flags, mod.keywordFlags, unpack(modifiers))
			}
		*/
	}

	// Primary defences: Energy shield, evasion and armour
	{
		ironReflexes := modDB.Flag(nil, "IronReflexes")
		ward := float64(0)
		energyShield := float64(0)
		armour := float64(0)
		evasion := float64(0)
		/*
			TODO Breakdown
			if breakdown != nil {
				breakdown.Ward = { slots = { } }
				breakdown.EnergyShield = { slots = { } }
				breakdown.Armour = { slots = { } }
				breakdown.Evasion = { slots = { } }
			}
		*/
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
						/*
							TODO Breakdown
							if breakdown != nil {
								t_insert(breakdown["Ward"].slots, {
									base = wardBase,
									inc = (inc ~= 0) and s_format(" x %.2f", 1 + inc/100),
									more = (more ~= 1) and s_format(" x %.2f", more),
									total = s_format("%.2f", wardBase * (1 + inc / 100) * more),
									source = slot,
									item = actor.itemList[slot],
								})
							}
						*/
					} else {
						ward = ward + wardBase*calclib.Mod(modDB, slotCfg, "Ward", "Defences")
						gearWard = gearWard + wardBase
						/*
							TODO Breakdown
							if breakdown != nil {
								breakdown.slot(slot, nil, slotCfg, wardBase, nil, "Ward", "Defences")
							}
						*/
					}
				}
				energyShieldBase = armourData.EnergyShield
				if energyShieldBase > 0 {
					actor.Output["EnergyShieldOn"+slot] = energyShieldBase
					if modDB.Flag(nil, "EnergyShieldToWard") {
						more := modDB.More(slotCfg, "EnergyShield", "Defences")
						energyShield = energyShield + energyShieldBase*more
						gearEnergyShield = gearEnergyShield + energyShieldBase
						/*
							TODO Breakdown
							if breakdown != nil {
								t_insert(breakdown["EnergyShield"].slots, {
									base = energyShieldBase,
									more = (more ~= 1) and s_format(" x %.2f", more),
									total = s_format("%.2f", energyShieldBase * more),
									source = slot,
									item = actor.itemList[slot],
								})
							}
						*/
					} else {
						energyShield = energyShield + energyShieldBase*calclib.Mod(modDB, slotCfg, "EnergyShield", "Defences")
						gearEnergyShield = gearEnergyShield + energyShieldBase
						/*
							TODO Breakdown
							if breakdown != nil {
								breakdown.slot(slot, nil, slotCfg, energyShieldBase, nil, "EnergyShield", "Defences")
							}
						*/
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
					/*
						TODO Breakdown
						if breakdown != nil {
							breakdown.slot(slot, nil, slotCfg, armourBase, nil, "Armour", "ArmourAndEvasion", "Defences")
						}
					*/
				}
				evasionBase = armourData.Evasion
				if evasionBase > 0 {
					actor.Output["EvasionOn"+slot] = evasionBase
					if ironReflexes {
						armour = armour + evasionBase*calclib.Mod(modDB, slotCfg, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
						gearArmour = gearArmour + evasionBase
						/*
							TODO Breakdown
							if breakdown != nil {
								breakdown.slot(slot, nil, slotCfg, evasionBase, nil, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
							}
						*/
					} else {
						evasion = evasion + evasionBase*calclib.Mod(modDB, slotCfg, "Evasion", "ArmourAndEvasion", "Defences")
						gearEvasion = gearEvasion + evasionBase
						/*
							TODO Breakdown
							if breakdown != nil {
								breakdown.slot(slot, nil, slotCfg, evasionBase, nil, "Evasion", "ArmourAndEvasion", "Defences")
							}
						*/
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
				/*
					TODO Breakdown
					if breakdown != nil {
						t_insert(breakdown["Ward"].slots, {
							base = wardBase,
							inc = (inc ~= 0) and s_format(" x %.2f", 1 + inc/100),
							more = (more ~= 1) and s_format(" x %.2f", more),
							total = s_format("%.2f", wardBase * (1 + inc / 100) * more),
							source = "Global",
							item = actor.itemList["Global"],
						})
					}
				*/
			} else {
				ward = ward + wardBase*calclib.Mod(modDB, nil, "Ward", "Defences")
				/*
					TODO Breakdown
					if breakdown != nil {
						breakdown.slot("Global", nil, nil, wardBase, nil, "Ward", "Defences")
					}
				*/
			}
		}
		energyShieldBase = modDB.Sum(mod.TypeBase, nil, "EnergyShield")
		if energyShieldBase > 0 {
			if modDB.Flag(nil, "EnergyShieldToWard") {
				energyShield = energyShield + energyShieldBase*modDB.More(slotCfg, "EnergyShield", "Defences")
			} else {
				energyShield = energyShield + energyShieldBase*calclib.Mod(modDB, nil, "EnergyShield", "Defences")
			}
			/*
				TODO Breakdown
				if breakdown != nil {
					more := modDB.More(slotCfg, "EnergyShield", "Defences")
					t_insert(breakdown["EnergyShield"].slots, {
						base = energyShieldBase,
						more = (more ~= 1) and s_format(" x %.2f", more),
						total = s_format("%.2f", energyShieldBase * more),
						source = "Global",
						item = actor.itemList["Global"],
					})
				}
			*/
		}
		armourBase = modDB.Sum(mod.TypeBase, nil, "Armour", "ArmourAndEvasion")
		if armourBase > 0 {
			armour = armour + armourBase*calclib.Mod(modDB, nil, "Armour", "ArmourAndEvasion", "Defences")
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.slot("Global", nil, nil, armourBase, nil, "Armour", "ArmourAndEvasion", "Defences")
				}
			*/
		}
		evasionBase = modDB.Sum(mod.TypeBase, nil, "Evasion", "ArmourAndEvasion")
		if evasionBase > 0 {
			if ironReflexes {
				armour = armour + evasionBase*calclib.Mod(modDB, nil, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
				/*
					TODO Breakdown
					if breakdown != nil {
						breakdown.slot("Conversion", "Evasion to Armour", nil, evasionBase, nil, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
					}
				*/
			} else {
				evasion = evasion + evasionBase*calclib.Mod(modDB, nil, "Evasion", "ArmourAndEvasion", "Defences")
				/*
					TODO Breakdown
					if breakdown != nil {
						breakdown.slot("Global", nil, nil, evasionBase, nil, "Evasion", "ArmourAndEvasion", "Defences")
					}
				*/
			}
		}
		convManaToArmour := modDB.Sum(mod.TypeBase, nil, "ManaConvertToArmour")
		if convManaToArmour > 0 {
			armourBase = 2 * modDB.Sum(mod.TypeBase, nil, "Mana") * convManaToArmour / 100
			total := armourBase * calclib.Mod(modDB, nil, "Mana", "Armour", "ArmourAndEvasion", "Defences")
			armour = armour + total
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.slot("Conversion", "Mana to Armour", nil, armourBase, total, "Armour", "ArmourAndEvasion", "Defences", "Mana")
				}
			*/
		}
		convManaToES := modDB.Sum(mod.TypeBase, nil, "ManaGainAsEnergyShield")
		if convManaToES > 0 {
			energyShieldBase = modDB.Sum(mod.TypeBase, nil, "Mana") * convManaToES / 100
			energyShield = energyShield + energyShieldBase*calclib.Mod(modDB, nil, "Mana", "EnergyShield", "Defences")
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.slot("Conversion", "Mana to Energy Shield", nil, energyShieldBase, nil, "EnergyShield", "Defences", "Mana")
				}
			*/
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
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.slot("Conversion", "Life to Armour", nil, armourBase, total, "Armour", "ArmourAndEvasion", "Defences", "Life")
				}
			*/
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
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.slot("Conversion", "Life to Energy Shield", nil, energyShieldBase, total, "EnergyShield", "Defences", "Life")
				}
			*/
		}
		convEvasionToArmour := modDB.Sum(mod.TypeBase, nil, "EvasionGainAsArmour")
		if convEvasionToArmour > 0 {
			armourBase = (modDB.Sum(mod.TypeBase, nil, "Evasion") + gearEvasion) * convEvasionToArmour / 100
			total := armourBase * calclib.Mod(modDB, nil, "Evasion", "Armour", "ArmourAndEvasion", "Defences")
			armour = armour + total
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.slot("Conversion", "Evasion to Armour", nil, armourBase, total, "Armour", "ArmourAndEvasion", "Defences", "Evasion")
				}
			*/
		}
		actor.Output["EnergyShield"] = utils.Or(modDB.Override(nil, "EnergyShield"), max(utils.RoundTo(energyShield, 0), 0))
		actor.Output["Armour"] = max(utils.RoundTo(armour, 0), 0)
		actor.Output["ArmourDefense"] = (modDB.Max(nil, "ArmourDefense")) / 100
		actor.Output["RawArmourDefense"] = utils.Ternary(actor.Output["ArmourDefense"] > 0, ((1 + actor.Output["ArmourDefense"]) * 100), 0)
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
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.EvadeChance = {
						s_format("Enemy level: %d ^8(%s the Configuration tab)", env.enemyLevel, env.configInput.enemyLevel and "overridden from" or "can be overridden in"),
						s_format("Average enemy accuracy: %d", enemyAccuracy),
						s_format("Approximate evade chance: %d%%", actor.Output["EvadeChance"]),
					}
					breakdown.MeleeEvadeChance = {
						s_format("Enemy level: %d ^8(%s the Configuration tab)", env.enemyLevel, env.configInput.enemyLevel and "overridden from" or "can be overridden in"),
						s_format("Average enemy accuracy: %d", enemyAccuracy),
						s_format("Approximate melee evade chance: %d%%", actor.Output["MeleeEvadeChance"]),
					}
					breakdown.ProjectileEvadeChance = {
						s_format("Enemy level: %d ^8(%s the Configuration tab)", env.enemyLevel, env.configInput.enemyLevel and "overridden from" or "can be overridden in"),
						s_format("Average enemy accuracy: %d", enemyAccuracy),
						s_format("Approximate projectile evade chance: %d%%", actor.Output["ProjectileEvadeChance"]),
					}
				}
			*/
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
	/*
		TODO Breakdown
		if breakdown != nil {
			breakdown.BlockChance = breakdown.simple(baseBlockChance, nil, actor.Output["BlockChance"], "BlockChance")
			breakdown.SpellBlockChance = breakdown.simple(0, nil, actor.Output["SpellBlockChance"], "SpellBlockChance")
		}
	*/
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
	//baseDodgeChance := 0
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

	/*
		TODO Breakdown
		if breakdown != nil {
			breakdown.AttackDodgeChance = {
				"Base: "+baseDodgeChance+"%",
				"Max: "+attackDodgeChanceMax+"%",
				"Total: "+actor.Output["AttackDodgeChance"]+actor.Output["AttackDodgeChanceOverCap"]+"%",
			}
			breakdown.SpellDodgeChance = {
				"Base: "+baseDodgeChance+"%",
				"Max: "+spellDodgeChanceMax+"%",
				"Total: "+actor.Output["SpellDodgeChance"]+actor.Output["SpellDodgeChanceOverCap"]+"%",
			}
		}
	*/

	// Recovery modifiers
	actor.Output["LifeRecoveryRateMod"] = calclib.Mod(modDB, nil, "LifeRecoveryRate")
	actor.Output["ManaRecoveryRateMod"] = calclib.Mod(modDB, nil, "ManaRecoveryRate")
	actor.Output["EnergyShieldRecoveryRateMod"] = calclib.Mod(modDB, nil, "EnergyShieldRecoveryRate")

	// Leech caps
	actor.Output["MaxLifeLeechInstance"] = actor.Output["Life"] * calclib.Val(modDB, "MaxLifeLeechInstance") / 100
	actor.Output["MaxLifeLeechRatePercent"] = calclib.Val(modDB, "MaxLifeLeechRate")
	actor.Output["MaxLifeLeechRate"] = actor.Output["Life"] * actor.Output["MaxLifeLeechRatePercent"] / 100
	/*
		TODO Breakdown
		if breakdown != nil {
			breakdown.MaxLifeLeechRate = {
				s_format("%d ^8(maximum life)", actor.Output["Life"]),
				s_format("x %d%% ^8(percentage of life to maximum leech rate)", actor.Output["MaxLifeLeechRatePercent"]),
				s_format("= %.1f", actor.Output["MaxLifeLeechRate"])
			}
		}
	*/
	actor.Output["MaxEnergyShieldLeechInstance"] = actor.Output["EnergyShield"] * calclib.Val(modDB, "MaxEnergyShieldLeechInstance") / 100
	actor.Output["MaxEnergyShieldLeechRate"] = actor.Output["EnergyShield"] * calclib.Val(modDB, "MaxEnergyShieldLeechRate") / 100
	/*
		TODO Breakdown
		if breakdown != nil {
			breakdown.MaxEnergyShieldLeechRate = {
				s_format("%d ^8(maximum energy shield)", actor.Output["EnergyShield"]),
				s_format("x %d%% ^8(percentage of energy shield to maximum leech rate)", calcLib.val(modDB, "MaxEnergyShieldLeechRate")),
				s_format("= %.1f", actor.Output["MaxEnergyShieldLeechRate"])
			}
		}
	*/
	actor.Output["MaxManaLeechInstance"] = actor.Output["Mana"] * calclib.Val(modDB, "MaxManaLeechInstance") / 100
	actor.Output["MaxManaLeechRate"] = actor.Output["Mana"] * calclib.Val(modDB, "MaxManaLeechRate") / 100
	/*
		TODO Breakdown
		if breakdown != nil {
			breakdown.MaxManaLeechRate = {
				s_format("%d ^8(maximum mana)", actor.Output["Mana"]),
				s_format("x %d%% ^8(percentage of mana to maximum leech rate)", modDB.Sum(mod.TypeBase, nil, "MaxManaLeechRate")),
				s_format("= %.1f", actor.Output["MaxManaLeechRate"])
			}
		}
	*/

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
		// TODO Breakdown
		/*
			if breakdown != nil {
				breakdown.ManaRegen = { }
				breakdown.multiChain(breakdown.ManaRegen, {
					label = "Mana Regeneration:",
					base = s_format("%.1f ^8(base)", base),
					{ "%.2f ^8(increased/reduced)", 1 + actor.Output["ManaRegenInc"]/100 },
					{ "%.2f ^8(more/less)", more },
					total = s_format("= %.1f ^8per second", regen),
				})
				breakdown.multiChain(breakdown.ManaRegen, {
					label = "Effective Mana Regeneration:",
					base = s_format("%.1f", regen),
					{ "%.2f ^8(recovery rate modifier)", actor.Output["ManaRecoveryRateMod"] },
					total = s_format("= %.1f ^8per second", regenRate),
				})
				if degen ~= 0 {
					t_insert(breakdown.ManaRegen, s_format("- %d", degen))
					t_insert(breakdown.ManaRegen, s_format("= %.1f ^8per second", actor.Output["ManaRegen"]))
				}
			}
		*/
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

		/*
			TODO Breakdown
			if breakdown != nil {
				breakdown.RageRegen = { }
				breakdown.multiChain(breakdown.RageRegen, {
					base = s_format("%.1f ^8(base)", base),
					{ "%.2f ^8(increased/reduced)", 1 + inc/100 },
					{ "%.2f ^8(more/less)", more },
					total = s_format("= %.1f ^8per second", actor.Output["RageRegen"]),
				})
			}
		*/
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
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.LifeRecharge = { }
					breakdown.multiChain(breakdown.LifeRecharge, {
						label = "Recharge rate:",
						base = s_format("%.1f ^8(33%% per second)", actor.Output["Life"] * data.misc.EnergyShieldRechargeBase),
						{ "%.2f ^8(increased/reduced)", 1 + inc/100 },
						{ "%.2f ^8(more/less)", more },
						total = s_format("= %.1f ^8per second", recharge),
					})
					breakdown.multiChain(breakdown.LifeRecharge, {
						label = "Effective Recharge rate:",
						base = s_format("%.1f", recharge),
						{ "%.2f ^8(recovery rate modifier)", actor.Output["LifeRecoveryRateMod"] },
						total = s_format("= %.1f ^8per second", actor.Output["LifeRecharge"]),
					})
				}
			*/
		} else {
			actor.Output["EnergyShieldRechargeAppliesToEnergyShield"] = 1
			recharge := actor.Output["EnergyShield"] * data.EnergyShieldRechargeBase * (1 + inc/100) * more
			actor.Output["EnergyShieldRecharge"] = utils.RoundTo(recharge*actor.Output["EnergyShieldRecoveryRateMod"], 0)
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.EnergyShieldRecharge = { }
					breakdown.multiChain(breakdown.EnergyShieldRecharge, {
						label = "Recharge rate:",
						base = s_format("%.1f ^8(33%% per second)", actor.Output["EnergyShield"] * data.misc.EnergyShieldRechargeBase),
						{ "%.2f ^8(increased/reduced)", 1 + inc/100 },
						{ "%.2f ^8(more/less)", more },
						total = s_format("= %.1f ^8per second", recharge),
					})
					breakdown.multiChain(breakdown.EnergyShieldRecharge, {
						label = "Effective Recharge rate:",
						base = s_format("%.1f", recharge),
						{ "%.2f ^8(recovery rate modifier)", actor.Output["EnergyShieldRecoveryRateMod"] },
						total = s_format("= %.1f ^8per second", actor.Output["EnergyShieldRecharge"]),
					})
				}
			*/
		}
		actor.Output["EnergyShieldRechargeDelay"] = data.EnergyShieldRechargeDelay / (1 + modDB.Sum(mod.TypeIncrease, nil, "EnergyShieldRechargeFaster")/100)
		/*
			if breakdown != nil {
				if actor.Output["EnergyShieldRechargeDelay"] ~= data.misc.EnergyShieldRechargeDelay {
					breakdown.EnergyShieldRechargeDelay = {
						s_format("%.2fs ^8(base)", data.misc.EnergyShieldRechargeDelay),
						s_format("/ %.2f ^8(faster start)", 1 + modDB.Sum(mod.TypeIncrease, nil, "EnergyShieldRechargeFaster") / 100),
						s_format("= %.2fs", actor.Output["EnergyShieldRechargeDelay"])
					}
				}
			}
		*/
	}

	// Ward recharge
	actor.Output["WardRechargeDelay"] = data.WardRechargeDelay / (1 + modDB.Sum(mod.TypeIncrease, nil, "WardRechargeFaster")/100)
	/*
		TODO Breakdown
		if breakdown != nil {
			if actor.Output["WardRechargeDelay"] ~= data.misc.WardRechargeDelay {
				breakdown.WardRechargeDelay = {
					s_format("%.2fs ^8(base)", data.misc.WardRechargeDelay),
					s_format("/ %.2f ^8(faster start)", 1 + modDB.Sum(mod.TypeIncrease, nil, "WardRechargeFaster") / 100),
					s_format("= %.2fs", actor.Output["WardRechargeDelay"])
				}
			}
		}
	*/

	// Miscellaneous: move speed, stun recovery, avoidance
	actor.Output["MovementSpeedMod"] = utils.Or(modDB.Override(nil, "MovementSpeed"), calclib.Mod(modDB, nil, "MovementSpeed"))
	if modDB.Flag(nil, "MovementSpeedCannotBeBelowBase") {
		actor.Output["MovementSpeedMod"] = max(actor.Output["MovementSpeedMod"], 1)
	}
	actor.Output["EffectiveMovementSpeedMod"] = actor.Output["MovementSpeedMod"] * actor.Output["ActionSpeedMod"]

	/*
		TODO Breakdown
		actor.Output["MovementSpeedMod"] = modDB.Override(nil, "MovementSpeed") or calcLib.mod(modDB, nil, "MovementSpeed")
		if modDB.Flag(nil, "MovementSpeedCannotBeBelowBase") {
			actor.Output["MovementSpeedMod"] = max(actor.Output["MovementSpeedMod"], 1)
		}
		actor.Output["EffectiveMovementSpeedMod"] = actor.Output["MovementSpeedMod"] * actor.Output["ActionSpeedMod"]
		if breakdown != nil {
			breakdown.EffectiveMovementSpeedMod = { }
			breakdown.multiChain(breakdown.EffectiveMovementSpeedMod, {
				{ "%.2f ^8(movement speed modifier)", actor.Output["MovementSpeedMod"] },
				{ "%.2f ^8(action speed modifier)", actor.Output["ActionSpeedMod"] },
				total = s_format("= %.2f ^8(effective movement speed modifier)", actor.Output["EffectiveMovementSpeedMod"])
			})
		}
	*/

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
		/*
			TODO Breakdown
			if breakdown != nil {
				breakdown.StunDuration = {
					"0.35s ^8(base)",
						s_format("/ %.2f ^8(increased/reduced recovery)", 1 + modDB.Sum(mod.TypeIncrease, nil, "StunRecovery") / 100),
						s_format("= %.2fs", actor.Output["StunDuration"])
				}
				breakdown.BlockDuration = {
					"0.35s ^8(base)",
						s_format("/ %.2f ^8(increased/reduced recovery)", 1 + modDB.Sum(mod.TypeIncrease, nil, "StunRecovery", "BlockRecovery") / 100),
						s_format("= %.2fs", actor.Output["BlockDuration"])
				}
			}
		*/
	}
	actor.Output["InteruptStunAvoidChance"] = min(modDB.Sum(mod.TypeBase, nil, "AvoidInteruptStun"), 100)
	actor.Output["BlindAvoidChance"] = min(modDB.Sum(mod.TypeBase, nil, "AvoidBlind"), 100)
	for _, ailment := range data.Ailment("").Values() {
		actor.Output[string(ailment+"AvoidChance")] = min(modDB.Sum(mod.TypeBase, nil, string("Avoid"+ailment)), 100)
	}
	actor.Output["CritExtraDamageReduction"] = min(modDB.Sum(mod.TypeBase, nil, "ReduceCritExtraDamage"), 100)
	actor.Output["LightRadiusMod"] = calclib.Mod(modDB, nil, "LightRadius")
	/*
		TODO Breakdown
		if breakdown != nil {
			breakdown.LightRadiusMod = breakdown.mod(modDB, nil, "LightRadius")
		}
	*/
	actor.Output["CurseEffectOnSelf"] = modDB.More(nil, "CurseEffectOnSelf") * (100 + modDB.Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf"))

	// Ailment duration on self
	actor.Output["SelfBlindDuration"] = modDB.More(nil, "SelfBlindDuration") * (100 + modDB.Sum(mod.TypeIncrease, nil, "SelfBlindDuration"))
	for _, ailment := range data.Ailment("").Values() {
		actor.Output[string("Self"+ailment+"Duration")] = modDB.More(nil, string("Self"+ailment+"Duration")) * (100 + modDB.Sum(mod.TypeIncrease, nil, string("Self"+ailment+"Duration")))
	}
	actor.Output["SelfChillEffect"] = modDB.More(nil, "SelfChillEffect") * (100 + modDB.Sum(mod.TypeIncrease, nil, "SelfChillEffect"))
	actor.Output["SelfShockEffect"] = modDB.More(nil, "SelfShockEffect") * (100 + modDB.Sum(mod.TypeIncrease, nil, "SelfShockEffect"))

	/*
		TODO --Enemy damage input and modifications
		{
			actor.Output["totalEnemyDamage"] = 0
			actor.Output["totalEnemyDamageIn"] = 0
			if breakdown != nil {
				breakdown["totalEnemyDamage"] = {
					label = "Total damage from the enemy",
					rowList = { },
					colList = {
						{ label = "Type", key = "type" },
						{ label = "Value", key = "value" },
						{ label = "Mult", key = "mult" },
						{ label = "Crit", key = "crit" },
						{ label = "Final", key = "final" },
						{ label = "From", key = "from" },
					},
				}
			}
			enemyCritChance := env.configInput["enemyCritChance"] or env.configPlaceholder["enemyCritChance"] or 0
			enemyCritDamage := env.configInput["enemyCritDamage"] or env.configPlaceholder["enemyCritDamage"] or 0
			actor.Output["EnemyCritEffect"] = 1 + enemyCritChance / 100 * (enemyCritDamage / 100) * (1 - actor.Output["CritExtraDamageReduction"] / 100)
			for _, damageType in ipairs(dmgTypeList) {
				enemyDamageMult := calcLib.mod(enemyDB, nil, "Damage", damageType+"Damage", isElemental[damageType] and "ElementalDamage" or nil) --missing taunt from allies
				enemyDamage := env.configInput["enemy"+damageType+"Damage"]
				enemyPen := env.configInput["enemy"+damageType+"Pen"]
				sourceStr := enemyDamage == nil and "Default" or "Config"

				if enemyDamage == nil and env.configPlaceholder["enemy"+damageType+"Damage"] {
					enemyDamage = env.configPlaceholder["enemy"+damageType+"Damage"]
				}
				if enemyPen == nil and env.configPlaceholder["enemy"+damageType+"Pen"] {
					enemyPen = env.configPlaceholder["enemy"+damageType+"Pen"]
				}
				enemyDamage = enemyDamage or 0
				actor.Output[damageType+"EnemyPen"] = enemyPen or 0
				actor.Output["totalEnemyDamageIn"] = actor.Output["totalEnemyDamageIn"] + enemyDamage
				actor.Output[damageType+"EnemyDamage"] = enemyDamage * enemyDamageMult * actor.Output["EnemyCritEffect"]
				actor.Output["totalEnemyDamage"] = actor.Output["totalEnemyDamage"] + actor.Output[damageType+"EnemyDamage"]
				if breakdown != nil {
					breakdown[damageType+"EnemyDamage"] = {
					s_format("from %s: %d", sourceStr, enemyDamage),
					s_format("* %.2f (modifiers to enemy damage)", enemyDamageMult),
					s_format("* %.3f (enemy crit effect)", actor.Output["EnemyCritEffect"]),
					s_format("= %d", actor.Output[damageType+"EnemyDamage"]),
					}
					t_insert(breakdown["totalEnemyDamage"].rowList, {
						type = s_format("%s", damageType),
						value = s_format("%d", enemyDamage),
						mult = s_format("%.2f", enemyDamageMult),
						crit = s_format("%.2f", actor.Output["EnemyCritEffect"]),
						final = s_format("%d", actor.Output[damageType+"EnemyDamage"]),
						from = s_format("%s", sourceStr),
					})
				}
			}
		}
	*/
	/*
		TODO --Damage Taken as
		{
			actor.damageShiftTable = wipeTable(actor.damageShiftTable)
			for _, damageType in ipairs(dmgTypeList) {
				-- Build damage shift table
				shiftTable := { }
				destTotal := 0
				for _, destType in ipairs(dmgTypeList) {
					if destType ~= damageType {
						shiftTable[destType] = modDB.Sum(mod.TypeBase, nil, damageType+"DamageTakenAs"+destType, isElemental[damageType] and "ElementalDamageTakenAs"+destType or nil)
						destTotal = destTotal + shiftTable[destType]
					}
				}
				if destTotal > 100 {
					factor := 100 / destTotal
					for destType, portion in pairs(shiftTable) {
						shiftTable[destType] = portion * factor
					}
					destTotal = 100
				}
				shiftTable[damageType] = 100 - destTotal
				actor.damageShiftTable[damageType] = shiftTable

				--add same type damage
				actor.Output[damageType+"TakenDamage"] = actor.Output[damageType+"EnemyDamage"] * actor.damageShiftTable[damageType][damageType] / 100
				if breakdown != nil {
					breakdown[damageType+"TakenDamage"] = {
						label = "Taken",
						rowList = { },
						colList = {
							{ label = "Type", key = "type" },
							{ label = "Value", key = "value" },
						},
					}
					t_insert(breakdown[damageType+"TakenDamage"].rowList, {
						type = s_format("%s", damageType),
						value = s_format("%d", actor.Output[damageType+"TakenDamage"]),
					})
				}
			}
			--converted damage types
			for _, damageType in ipairs(dmgTypeList) {
				for _, damageConvertedType in ipairs(dmgTypeList) {
					if damageType ~= damageConvertedType {
						damage := actor.Output[damageType+"EnemyDamage"] * actor.damageShiftTable[damageType][damageConvertedType] / 100
						actor.Output[damageConvertedType+"TakenDamage"] = actor.Output[damageConvertedType+"TakenDamage"] + damage
						if breakdown and damage > 0 {
							t_insert(breakdown[damageConvertedType+"TakenDamage"].rowList, {
								type = s_format("%s", damageType),
								value = s_format("%d", damage),
							})
						}
					}
				}
			}
			--total
			actor.Output["totalTakenDamage"] = 0
			if breakdown != nil {
				breakdown["totalTakenDamage"] = {
					label = "Total damage taken from the enemy after taken as",
					rowList = { },
					colList = {
						{ label = "Type", key = "type" },
						{ label = "Value", key = "value" },
					},
				}
			}
			for _, damageType in ipairs(dmgTypeList) {
				actor.Output["totalTakenDamage"] = actor.Output["totalTakenDamage"] + actor.Output[damageType+"TakenDamage"]
				if breakdown != nil {
					t_insert(breakdown["totalTakenDamage"].rowList, {
						type = s_format("%s", damageType),
						value = s_format("%d", actor.Output[damageType+"TakenDamage"]),
					})
				}
			}
		}
	*/
	/*
		TODO -- Damage taken multipliers/Degen calculations
		actor.Output["AnyTakenReflect"] = false
		damageCategoryConfig := env.configInput.enemyDamageType or "Average"
		for _, damageType in ipairs(dmgTypeList) {
			baseTakenInc := modDB.Sum(mod.TypeIncrease, nil, "DamageTaken", damageType+"DamageTaken")
			baseTakenMore := modDB.More(nil, "DamageTaken", damageType+"DamageTaken")
			if isElemental[damageType] {
				baseTakenInc = baseTakenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTaken")
				baseTakenMore = baseTakenMore * modDB.More(nil, "ElementalDamageTaken")
			}
			do	-- Hit
				takenInc := baseTakenInc + modDB.Sum(mod.TypeIncrease, nil, "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
				takenMore := baseTakenMore * modDB.More(nil, "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
				if isElemental[damageType] {
					takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTakenWhenHit")
					takenMore = takenMore * modDB.More(nil, "ElementalDamageTakenWhenHit")
				}
				actor.Output[damageType+"TakenHitMult"] = max((1 + takenInc / 100) * takenMore, 0)

				for _, hitType in ipairs(hitSourceList) {
					baseTakenIncType := takenInc + modDB.Sum(mod.TypeIncrease, nil, hitType+"DamageTaken")
					baseTakenMoreType := takenMore * modDB.More(nil, hitType+"DamageTaken")
					actor.Output[hitType+"TakenHitMult"] = max((1 + baseTakenIncType / 100) * baseTakenMoreType, 0)
					actor.Output[damageType+hitType+"TakenHitMult"] = actor.Output[hitType+"TakenHitMult"]
				}
				{
					-- Reflect
					takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, damageType+"ReflectedDamageTaken")
					takenMore = takenMore * modDB.More(nil, damageType+"ReflectedDamageTaken")
					if isElemental[damageType] {
						takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalReflectedDamageTaken")
						takenMore = takenMore * modDB.More(nil, "ElementalReflectedDamageTaken")
					}
					actor.Output[damageType+"TakenReflect"] = max((1 + takenInc / 100) * takenMore, 0)
					if actor.Output[damageType+"TakenReflect"] ~= actor.Output[damageType+"TakenHitMult"] {
						actor.Output["AnyTakenReflect"] = false --true --this needs a rework as well
					}
				}
			}
			do	-- Dot
				takenInc := baseTakenInc + modDB.Sum(mod.TypeIncrease, nil, "DamageTakenOverTime", damageType+"DamageTakenOverTime")
				takenMore := baseTakenMore * modDB.More(nil, "DamageTakenOverTime", damageType+"DamageTakenOverTime")
				if isElemental[damageType] {
					takenInc = takenInc + modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTakenOverTime")
					takenMore = takenMore * modDB.More(nil, "ElementalDamageTakenOverTime")
				}
				resist := modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance") and 0 or actor.Output[damageType+"Resist"]
				if damageType == "Physical" {
					resist = max(resist, 0)
				}
				actor.Output[damageType+"TakenDotMult"] = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
				if breakdown != nil {
					breakdown[damageType+"TakenDotMult"] = { }
					breakdown.multiChain(breakdown[damageType+"TakenDotMult"], {
						label = "DoT Multiplier:",
						{ "%.2f ^8(%s)", (1 - resist / 100), damageType == "Physical" and "physical damage reduction" or "resistance" },
						{ "%.2f ^8(increased/reduced damage taken)", (1 + takenInc / 100) },
						{ "%.2f ^8(more/less damage taken)", takenMore },
						total = s_format("= %.2f", actor.Output[damageType+"TakenDotMult"]),
					})
				}
			}
		}
	*/
	/*
		TODO -- Incoming hit damage multipliers
		actor.Output["totalTakenHit"] = 0
		if breakdown != nil {
			breakdown["totalTakenHit"] = {
				label = "Total damage taken after mitigation",
				rowList = { },
				colList = {
					{ label = "Type", key = "type" },
					{ label = "Incoming", key = "incoming" },
					{ label = "Mult", key = "mult" },
					{ label = "Value", key = "value" },
				},
			}
		}
		for _, damageType in ipairs(dmgTypeList) {
			-- Calculate incoming damage multiplier
			resist := modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance") and 0 or actor.Output[damageType+"ResistWhenHit"] or actor.Output[damageType+"Resist"]
			enemyPen := modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance") and 0 or actor.Output[damageType+"EnemyPen"]
			takenFlat := modDB.Sum(mod.TypeBase, nil, "DamageTaken", damageType+"DamageTaken", "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
			if damageCategoryConfig == "Melee" or damageCategoryConfig == "Projectile" {
				takenFlat = takenFlat + modDB.Sum(mod.TypeBase, nil, "DamageTakenFromAttacks", damageType+"DamageTakenFromAttacks")
			} else if damageCategoryConfig == "Average" {
				takenFlat = takenFlat + modDB.Sum(mod.TypeBase, nil, "DamageTakenFromAttacks", damageType+"DamageTakenFromAttacks") / 2
			}
			if damageType == "Physical" or modDB.Flag(nil, "ArmourAppliesTo"+damageType+"DamageTaken") {
				damage := actor.Output[damageType+"TakenDamage"]
				armourReduct := 0
				portionArmour := 100
				if damageType == "Physical" {
					if not modDB.Flag(nil, "ArmourDoesNotApplyToPhysicalDamageTaken") {
						armourReduct = calcs.armourReduction(actor.Output["Armour"] * (1 + actor.Output["ArmourDefense"]), damage)
						armourReduct = max(min(actor.Output["DamageReductionMax"], resist - enemyPen + armourReduct), 0)
						resist = armourReduct
					}
				} else {
					portionArmour = 100 - (resist - enemyPen)
					armourReduct = calcs.armourReduction(actor.Output["Armour"] * (1 + actor.Output["ArmourDefense"]), damage * portionArmour / 100)
					armourReduct = min(actor.Output["DamageReductionMax"], armourReduct)
					resist = resist + armourReduct * portionArmour / 100
				}
				actor.Output[damageType+"DamageReduction"] = portionArmour < 100 and armourReduct * portionArmour / 100 or armourReduct
				if breakdown != nil {
					if portionArmour > 100 {
						breakdown[damageType+"DamageReduction"] = {
							s_format("Enemy Hit Damage:"),
							s_format("    %d ^8(total incoming damage)", damage),
							s_format("    * %.2f ^8(from resistance, applies before armour)", (portionArmour / 100)),
						}
					} else if portionArmour < 100 {
						breakdown[damageType+"DamageReduction"] = {
							s_format("Enemy Hit Damage: %d ^8(total incoming damage)", damage),
							s_format("Portion mitigated by Armour: %d%%", portionArmour),
						}
					} else {
						breakdown[damageType+"DamageReduction"] = {
							s_format("Enemy Hit Damage: %d ^8(total incoming damage)", damage),
						}
					}
					t_insert(breakdown[damageType+"DamageReduction"], s_format("Reduction from Armour: %d%%", armourReduct))
				}
			}
			takenMult := actor.Output[damageType+"TakenHitMult"]
			if damageCategoryConfig == "Melee" or damageCategoryConfig == "Projectile" {
				takenMult = actor.Output[damageType+"AttackTakenHitMult"]
			} else if damageCategoryConfig == "Spell" or damageCategoryConfig == "SpellProjectile" {
				takenMult = actor.Output[damageType+"SpellTakenHitMult"]
			} else if damageCategoryConfig == "Average" {
				takenMult = (actor.Output[damageType+"SpellTakenHitMult"] + actor.Output[damageType+"AttackTakenHitMult"]) / 2
			}
			actor.Output[damageType+"BaseTakenHitMult"] = (1 - (resist - enemyPen) / 100) * takenMult
			takenMultReflect := actor.Output[damageType+"TakenReflect"]
			finalReflect := (1 - (resist - enemyPen) / 100) * takenMultReflect
			actor.Output[damageType+"TakenHit"] = max(actor.Output[damageType+"TakenDamage"] * (1 - (resist - enemyPen) / 100) + takenFlat, 0) * takenMult
			actor.Output[damageType+"TakenHitMult"] = (actor.Output[damageType+"TakenDamage"] > 0) and (actor.Output[damageType+"TakenHit"] / actor.Output[damageType+"TakenDamage"]) or 0
			actor.Output["totalTakenHit"] = actor.Output["totalTakenHit"] + actor.Output[damageType+"TakenHit"]
			if actor.Output["AnyTakenReflect"] {
				actor.Output[damageType+"TakenReflectMult"] = finalReflect
			}
			if breakdown != nil {
				breakdown[damageType+"TakenHitMult"] = {
					s_format("Resistance: %.2f", 1 - resist / 100),
				}
				if enemyPen > 0 {
					t_insert(breakdown[damageType+"TakenHitMult"], s_format("Enemy Pen: %.2f", enemyPen))
				}
				t_insert(breakdown[damageType+"TakenHitMult"], s_format("+ Flat: %.3f", takenFlat))
				t_insert(breakdown[damageType+"TakenHitMult"], s_format("x Taken: %.3f", takenMult))
				t_insert(breakdown[damageType+"TakenHitMult"], s_format("= %.3f", actor.Output[damageType+"TakenHitMult"]))
				breakdown[damageType+"TakenHit"] = {
					s_format("Final %s Damage taken:", damageType),
					s_format("%.1f incoming damage", actor.Output[damageType+"TakenDamage"]),
					s_format("x %.3f damage mult", actor.Output[damageType+"TakenHitMult"]),
					s_format("= %.1f", actor.Output[damageType+"TakenHit"]),
				}
				t_insert(breakdown["totalTakenHit"].rowList, {
					type = s_format("%s", damageType),
					incoming = s_format("%.1f incoming damage", actor.Output[damageType+"TakenDamage"]),
					mult = s_format("x %.3f damage mult", actor.Output[damageType+"TakenHitMult"] ),
					value = s_format("%d", actor.Output[damageType+"TakenHit"]),
				})
				if actor.Output["AnyTakenReflect"] {
					breakdown[damageType+"TakenReflectMult"] = {
						s_format("Resistance: %.3f", 1 - resist / 100),
					}
					if enemyPen > 0 {
						t_insert(breakdown[damageType+"TakenReflectMult"], s_format("Enemy Pen: %.2f", enemyPen))
					}
					t_insert(breakdown[damageType+"TakenReflectMult"], s_format("Taken: %.3f", takenMultReflect))
					t_insert(breakdown[damageType+"TakenReflectMult"], s_format("= %.3f", finalReflect))
				}
			}
		}
	*/
	// Life Recoverable
	actor.Output["LifeRecoverable"] = actor.Output["LifeUnreserved"]
	if environment.Build.GetBooleanOption("conditionLowLife") {
		actor.Output["LifeRecoverable"] = min(actor.Output["Life"]*data.LowPoolThreshold, actor.Output["LifeUnreserved"])
		if actor.Output["LifeRecoverable"] < actor.Output["LifeUnreserved"] {
			actor.Output["CappingLife"] = 1
		}
	}
	/*
		TODO -- Prevented life loss (Petrified Blood)
		{
			actor.Output["preventedLifeLoss"] = modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfPrevented")
			portionLife := 1
			if not env.configInput["conditionLowLife"] {
				--portion of life that is lowlife
				portionLife = min(actor.Output["Life"] * data.misc.LowPoolThreshold / actor.Output["LifeRecoverable"], 1)
				actor.Output["preventedLifeLoss"] = actor.Output["preventedLifeLoss"] * portionLife
			}
			if breakdown != nil {
				breakdown["preventedLifeLoss"] = {
					s_format("Total life protected:"),
				}
				if portionLife ~= 1 {
					t_insert(breakdown["preventedLifeLoss"], s_format("%.2f ^8(initial portion taken from petrified blood)", actor.Output["preventedLifeLoss"] / portionLife / 100))
					t_insert(breakdown["preventedLifeLoss"], s_format("* %.2f ^8(portion of life on low life)", portionLife))
					t_insert(breakdown["preventedLifeLoss"], s_format("= %.2f ^8(final portion taken from petrified blood)", actor.Output["preventedLifeLoss"] / 100))
					t_insert(breakdown["preventedLifeLoss"], s_format(""))
				} else {
					t_insert(breakdown["preventedLifeLoss"], s_format("%.2f ^8(portion taken from petrified blood)", actor.Output["preventedLifeLoss"] / 100))
				}
				t_insert(breakdown["preventedLifeLoss"], s_format("%.2f ^8(portion taken from life)", 1 - actor.Output["preventedLifeLoss"] / 100))
			}
		}
	*/
	/*
		TODO -- Energy Shield bypass
		actor.Output["AnyBypass"] = false
		actor.Output["MinimumBypass"] = 100
		for _, damageType in ipairs(dmgTypeList) {
			if modDB.Flag(nil, "UnblockedDamageDoesBypassES") {
				actor.Output[damageType+"EnergyShieldBypass"] = 100
				actor.Output["AnyBypass"] = true
			} else {
				actor.Output[damageType+"EnergyShieldBypass"] = modDB.Sum(mod.TypeBase, nil, damageType+"EnergyShieldBypass") or 0
				if actor.Output[damageType+"EnergyShieldBypass"] ~= 0 {
					actor.Output["AnyBypass"] = true
				}
				if damageType == "Chaos" {
					if not modDB.Flag(nil, "ChaosNotBypassEnergyShield") {
						actor.Output[damageType+"EnergyShieldBypass"] = actor.Output[damageType+"EnergyShieldBypass"] + 100
					} else {
						actor.Output["AnyBypass"] = true
					}
				}
			}
			actor.Output[damageType+"EnergyShieldBypass"] = max(min(actor.Output[damageType+"EnergyShieldBypass"], 100), 0)
			actor.Output["MinimumBypass"] = min(actor.Output["MinimumBypass"], actor.Output[damageType+"EnergyShieldBypass"])
		}

		actor.Output["ehpSectionAnySpecificTypes"] = false
	*/
	/*
		TODO -- Mind over Matter
		actor.Output["OnlySharedMindOverMatter"] = false
		actor.Output["AnySpecificMindOverMatter"] = false
		actor.Output["sharedMindOverMatter"] = min(modDB.Sum(mod.TypeBase, nil, "DamageTakenFromManaBeforeLife"), 100)
		if actor.Output["sharedMindOverMatter"] > 0 {
			actor.Output["OnlySharedMindOverMatter"] = true
			sourcePool := max(actor.Output["ManaUnreserved"] or 0, 0)
			manatext := "unreserved mana"
			if modDB.Flag(nil, "EnergyShieldProtectsMana") and actor.Output["MinimumBypass"] < 100 {
				manatext = manatext+" + non-bypassed energy shield"
				if actor.Output["MinimumBypass"] > 0 {
					manaProtected := actor.Output["EnergyShieldRecoveryCap"] / (1 - actor.Output["MinimumBypass"] / 100) * (actor.Output["MinimumBypass"] / 100)
					sourcePool = max(sourcePool - manaProtected, 0) + min(sourcePool, manaProtected) / (actor.Output["MinimumBypass"] / 100)
				} else {
					sourcePool = sourcePool + actor.Output["EnergyShieldRecoveryCap"]
				}
			}
			poolProtected := sourcePool / (actor.Output["sharedMindOverMatter"] / 100) * (1 - actor.Output["sharedMindOverMatter"] / 100)
			if actor.Output["sharedMindOverMatter"] >= 100 {
				poolProtected = m_huge
				actor.Output["sharedManaEffectiveLife"] = actor.Output["LifeRecoverable"] + sourcePool
			} else {
				actor.Output["sharedManaEffectiveLife"] = max(actor.Output["LifeRecoverable"] - poolProtected, 0) + min(actor.Output["LifeRecoverable"], poolProtected) / (1 - actor.Output["sharedMindOverMatter"] / 100)
			}
			if breakdown != nil {
				if actor.Output["sharedMindOverMatter"] {
					breakdown["sharedMindOverMatter"] = {
						s_format("Total life protected:"),
						s_format("%d ^8(%s)", sourcePool, manatext),
						s_format("/ %.2f ^8(portion taken from mana)", actor.Output["sharedMindOverMatter"] / 100),
						s_format("x %.2f ^8(portion taken from life)", 1 - actor.Output["sharedMindOverMatter"] / 100),
						s_format("= %d", poolProtected),
						s_format("Effective life: %d", actor.Output["sharedManaEffectiveLife"])
					}
				}
			}
		} else {
			actor.Output["sharedManaEffectiveLife"] = actor.Output["LifeRecoverable"]
		}
		for _, damageType in ipairs(dmgTypeList) {
			actor.Output[damageType+"MindOverMatter"] = min(modDB.Sum(mod.TypeBase, nil, damageType+"DamageTakenFromManaBeforeLife"), 100 - actor.Output["sharedMindOverMatter"])
			if actor.Output[damageType+"MindOverMatter"] > 0 or (actor.Output[damageType+"EnergyShieldBypass"] > actor.Output["MinimumBypass"] and actor.Output["sharedMindOverMatter"] > 0) {
				MindOverMatter := actor.Output[damageType+"MindOverMatter"] + actor.Output["sharedMindOverMatter"]
				actor.Output["ehpSectionAnySpecificTypes"] = true
				actor.Output["AnySpecificMindOverMatter"] = true
				actor.Output["OnlySharedMindOverMatter"] = false
				sourcePool := max(actor.Output["ManaUnreserved"] or 0, 0)
				manatext := "unreserved mana"
				if modDB.Flag(nil, "EnergyShieldProtectsMana") and actor.Output[damageType+"EnergyShieldBypass"] < 100 {
					manatext = manatext+" + non-bypassed energy shield"
					if actor.Output[damageType+"EnergyShieldBypass"] > 0 {
						manaProtected := actor.Output["EnergyShieldRecoveryCap"] / (1 - actor.Output[damageType+"EnergyShieldBypass"] / 100) * (actor.Output[damageType+"EnergyShieldBypass"] / 100)
						sourcePool = max(sourcePool - manaProtected, 0) + min(sourcePool, manaProtected) / (actor.Output[damageType+"EnergyShieldBypass"] / 100)
					} else {
						sourcePool = sourcePool + actor.Output["EnergyShieldRecoveryCap"]
					}
				}
				poolProtected := sourcePool / (MindOverMatter / 100) * (1 - MindOverMatter / 100)
				if MindOverMatter >= 100 {
					poolProtected = m_huge
					actor.Output[damageType+"ManaEffectiveLife"] = actor.Output["LifeRecoverable"] + sourcePool
				} else {
					actor.Output[damageType+"ManaEffectiveLife"] = max(actor.Output["LifeRecoverable"] - poolProtected, 0) + min(actor.Output["LifeRecoverable"], poolProtected) / (1 - MindOverMatter / 100)
				}
				if breakdown != nil {
					if actor.Output[damageType+"MindOverMatter"] {
						breakdown[damageType+"MindOverMatter"] = {
							s_format("Total life protected:"),
							s_format("%d ^8(%s)", sourcePool, manatext),
							s_format("/ %.2f ^8(portion taken from mana)", MindOverMatter / 100),
							s_format("x %.2f ^8(portion taken from life)", 1 - MindOverMatter / 100),
							s_format("= %d", poolProtected),
							s_format("Effective life: %d", actor.Output[damageType+"ManaEffectiveLife"])
						}
					}
				}
			} else {
				actor.Output[damageType+"ManaEffectiveLife"] = actor.Output["sharedManaEffectiveLife"]
			}
		}
	*/
	/*
		TODO -- Guard
		actor.Output["AnyGuard"] = false
		actor.Output["sharedGuardAbsorbRate"] = min(modDB.Sum(mod.TypeBase, nil, "GuardAbsorbRate"), 100)
		if actor.Output["sharedGuardAbsorbRate"] > 0 {
			actor.Output["OnlySharedGuard"] = true
			actor.Output["sharedGuardAbsorb"] = calcLib.val(modDB, "GuardAbsorbLimit")
			lifeProtected := actor.Output["sharedGuardAbsorb"] / (actor.Output["sharedGuardAbsorbRate"] / 100) * (1 - actor.Output["sharedGuardAbsorbRate"] / 100)
			if breakdown != nil {
				breakdown["sharedGuardAbsorb"] = {
					s_format("Total life protected:"),
					s_format("%d ^8(guard limit)", actor.Output["sharedGuardAbsorb"]),
					s_format("/ %.2f ^8(portion taken from guard)", actor.Output["sharedGuardAbsorbRate"] / 100),
					s_format("x %.2f ^8(portion taken from life and energy shield)", 1 - actor.Output["sharedGuardAbsorbRate"] / 100),
					s_format("= %d", lifeProtected)
				}
			}
		}
		for _, damageType in ipairs(dmgTypeList) {
			actor.Output[damageType+"GuardAbsorbRate"] = min(modDB.Sum(mod.TypeBase, nil, damageType+"GuardAbsorbRate"), 100)
			if actor.Output[damageType+"GuardAbsorbRate"] > 0 {
				actor.Output["ehpSectionAnySpecificTypes"] = true
				actor.Output["AnyGuard"] = true
				actor.Output["OnlySharedGuard"] = false
				actor.Output[damageType+"GuardAbsorb"] = calcLib.val(modDB, damageType+"GuardAbsorbLimit")
				lifeProtected := actor.Output[damageType+"GuardAbsorb"] / (actor.Output[damageType+"GuardAbsorbRate"] / 100) * (1 - actor.Output[damageType+"GuardAbsorbRate"] / 100)
				if breakdown != nil {
					breakdown[damageType+"GuardAbsorb"] = {
						s_format("Total life protected:"),
						s_format("%d ^8(guard limit)", actor.Output[damageType+"GuardAbsorb"]),
						s_format("/ %.2f ^8(portion taken from guard)", actor.Output[damageType+"GuardAbsorbRate"] / 100),
						s_format("x %.2f ^8(portion taken from life and energy shield)", 1 - actor.Output[damageType+"GuardAbsorbRate"] / 100),
						s_format("= %d", lifeProtected),
					}
				}
			}
		}
	*/
	/*
		TODO --aegis
		actor.Output["AnyAegis"] = false
		actor.Output["sharedAegis"] = modDB.Max(nil, "AegisValue") or 0
		actor.Output["sharedElementalAegis"] = modDB.Max(nil, "ElementalAegisValue") or 0
		if actor.Output["sharedAegis"] > 0 {
			actor.Output["AnyAegis"] = true
		}
		if actor.Output["sharedElementalAegis"] > 0 {
			actor.Output["ehpSectionAnySpecificTypes"] = true
			actor.Output["AnyAegis"] = true
		}
		for _, damageType in ipairs(dmgTypeList) {
			aegisValue := modDB.Max(nil, damageType+"AegisValue") or 0
			if aegisValue > 0 {
				actor.Output["ehpSectionAnySpecificTypes"] = true
				actor.Output["AnyAegis"] = true
				actor.Output[damageType+"Aegis"] = aegisValue
			} else {
				actor.Output[damageType+"Aegis"] = 0
			}
			if isElemental[damageType] {
				actor.Output[damageType+"AegisDisplay"] = actor.Output[damageType+"Aegis"] + actor.Output["sharedElementalAegis"]
			}
		}
	*/
	/*
		TODO --frost shield
		{
			actor.Output["FrostShieldLife"] = modDB.Sum(mod.TypeBase, nil, "FrostGlobeHealth")
			actor.Output["FrostShieldDamageMitigation"] = modDB.Sum(mod.TypeBase, nil, "FrostGlobeDamageMitigation")

			lifeProtected := actor.Output["FrostShieldLife"] / (actor.Output["FrostShieldDamageMitigation"] / 100) * (1 - actor.Output["FrostShieldDamageMitigation"] / 100)
			if breakdown != nil {
				breakdown["FrostShieldLife"] = {
					s_format("Total life protected:"),
					s_format("%d ^8(frost shield limit)", actor.Output["FrostShieldLife"]),
					s_format("/ %.2f ^8(portion taken from frost shield)", actor.Output["FrostShieldDamageMitigation"] / 100),
					s_format("x %.2f ^8(portion taken from life and energy shield)", 1 - actor.Output["FrostShieldDamageMitigation"] / 100),
					s_format("= %d", lifeProtected),
				}
			}
		}
	*/
	/*
		TODO --total pool
		for _, damageType in ipairs(dmgTypeList) {
			actor.Output[damageType+"TotalPool"] = actor.Output[damageType+"ManaEffectiveLife"]
			manatext := "Mana"
			if actor.Output[damageType+"EnergyShieldBypass"] < 100 {
				if modDB.Flag(nil, "EnergyShieldProtectsMana") {
					manatext = manatext+" and non-bypassed Energy Shield"
				} else {
					if actor.Output[damageType+"EnergyShieldBypass"] > 0 {
						poolProtected := actor.Output["EnergyShieldRecoveryCap"] / (1 - actor.Output[damageType+"EnergyShieldBypass"] / 100) * (actor.Output[damageType+"EnergyShieldBypass"] / 100)
						actor.Output[damageType+"TotalPool"] = max(actor.Output[damageType+"TotalPool"] - poolProtected, 0) + min(actor.Output[damageType+"TotalPool"], poolProtected) / (actor.Output[damageType+"EnergyShieldBypass"] / 100)
					} else {
						actor.Output[damageType+"TotalPool"] = actor.Output[damageType+"TotalPool"] + actor.Output["EnergyShieldRecoveryCap"]
					}
				}
			}
			if breakdown != nil {
				breakdown[damageType+"TotalPool"] = {
					s_format("Life: %d", actor.Output["LifeRecoverable"])
				}
				if actor.Output[damageType+"ManaEffectiveLife"] ~= actor.Output["LifeRecoverable"] {
					t_insert(breakdown[damageType+"TotalPool"], s_format("%s through MoM: %d", manatext, actor.Output[damageType+"ManaEffectiveLife"] - actor.Output["LifeRecoverable"]))
				}
				if (not modDB.Flag(nil, "EnergyShieldProtectsMana")) and actor.Output[damageType+"EnergyShieldBypass"] < 100 {
					t_insert(breakdown[damageType+"TotalPool"], s_format("Non-bypassed Energy Shield: %d", actor.Output[damageType+"TotalPool"] - actor.Output[damageType+"ManaEffectiveLife"]))
				}
				t_insert(breakdown[damageType+"TotalPool"], s_format("TotalPool: %d", actor.Output[damageType+"TotalPool"]))
			}
		}
	*/
	/*
		TODO -- helper function that iteratively reduces pools until life hits 0 to determine the number of hits it would take with given damage to die
		function numberOfHitsToDie(DamageIn)
			numHits := 0
			DamageIn["cycles"] = DamageIn["cycles"] or 1

			--check damage in isnt 0 and that ward doesnt mitigate all damage
			for _, damageType in ipairs(dmgTypeList) {
				numHits = numHits + DamageIn[damageType]
			}
			if numHits == 0 {
				return m_huge
			} else if modDB.Flag(nil, "WardNotBreak") and actor.Output["Ward"] > 0 and  numHits < actor.Output["Ward"] {
				return m_huge
			} else {
				numHits = 0
			}

			life := actor.Output["LifeRecoverable"] or 0
			mana := actor.Output["ManaUnreserved"] or 0
			energyShield := actor.Output["EnergyShieldRecoveryCap"]
			ward := actor.Output["Ward"] or 0
			restoreWard := modDB.Flag(nil, "WardNotBreak") and ward or 0
			-- dont apply non-perma ward for speed up calcs as it wont zero it correctly per hit
			if (not modDB.Flag(nil, "WardNotBreak")) and DamageIn["cycles"] > 1 {
				ward = 0
				restoreWard = 0
			}
			frostShield := actor.Output["FrostShieldLife"] or 0
			aegis := {}
			aegis["shared"] = actor.Output["sharedAegis"] or 0
			aegis["sharedElemental"] = actor.Output["sharedElementalAegis"] or 0
			guard := {}
			guard["shared"] = actor.Output["sharedGuardAbsorb"] or 0
			for _, damageType in ipairs(dmgTypeList) {
				aegis[damageType] = actor.Output[damageType+"Aegis"] or 0
				guard[damageType] = actor.Output[damageType+"GuardAbsorb"] or 0
				if not DamageIn[damageType+"EnergyShieldBypass"] {
					DamageIn[damageType+"EnergyShieldBypass"] = actor.Output[damageType+"EnergyShieldBypass"] or 0
				}

			}
			DamageIn["LifeLossBelowHalfLost"] = DamageIn["LifeLossBelowHalfLost"] or 0
			DamageIn["WardBypass"] = DamageIn["WardBypass"] or modDB.Sum(mod.TypeBase, nil, "WardBypass") or 0

			itterationMultiplier := 1
			maxHits := data.misc.ehpCalcMaxHitsToCalc
			maxHits = maxHits / DamageIn["cycles"]
			while life > 0 and numHits < maxHits {
				numHits = numHits + itterationMultiplier
				Damage := {}
				for _, damageType in ipairs(dmgTypeList) {
					Damage[damageType] = DamageIn[damageType] * itterationMultiplier
				}
				if DamageIn.GainWhenHit and (itterationMultiplier > 1 or DamageIn["cycles"] > 1) {
					gainMult := itterationMultiplier * DamageIn["cycles"]
					life = min(life + DamageIn.LifeWhenHit * (gainMult - 1), gainMult * (actor.Output["LifeRecoverable"] or 0))
					mana = min(mana + DamageIn.ManaWhenHit * (gainMult - 1), gainMult * (actor.Output["ManaUnreserved"] or 0))
					energyShield = min(energyShield + DamageIn.EnergyShieldWhenHit * (gainMult - 1), gainMult * actor.Output["EnergyShieldRecoveryCap"])
				}
				for _, damageType in ipairs(dmgTypeList) {
					if Damage[damageType] > 0 {
						if frostShield > 0 {
							tempDamage := min(Damage[damageType] * actor.Output["FrostShieldDamageMitigation"] / 100 / itterationMultiplier, frostShield)
							frostShield = frostShield - tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
						if aegis[damageType] > 0 {
							tempDamage := min(Damage[damageType], aegis[damageType])
							aegis[damageType] = aegis[damageType] - tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
						if isElemental[damageType] and aegis["sharedElemental"] > 0 {
							tempDamage := min(Damage[damageType], aegis["sharedElemental"])
							aegis["sharedElemental"] = aegis["sharedElemental"] - tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
						if aegis["shared"] > 0 {
							tempDamage := min(Damage[damageType], aegis["shared"])
							aegis["shared"] = aegis["shared"] - tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
						if guard[damageType] > 0 {
							tempDamage := min(Damage[damageType] * actor.Output[damageType+"GuardAbsorbRate"] / 100 / itterationMultiplier, guard[damageType])
							guard[damageType] = guard[damageType] - tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
						if guard["shared"] > 0 {
							tempDamage := min(Damage[damageType] * actor.Output["sharedGuardAbsorbRate"] / 100 / itterationMultiplier, guard["shared"])
							guard["shared"] = guard["shared"] - tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
						if ward > 0 {
							tempDamage := min(Damage[damageType] * (1 - DamageIn["WardBypass"] / 100), ward)
							ward = ward - tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
						if energyShield > 0 and (not modDB.Flag(nil, "EnergyShieldProtectsMana")) and DamageIn[damageType+"EnergyShieldBypass"] < 100 {
							tempDamage := min(Damage[damageType] * (1 - DamageIn[damageType+"EnergyShieldBypass"] / 100), energyShield)
							energyShield = energyShield - tempDamage
							Damage[damageType] = Damage[damageType] - tempDamage
						}
						if (actor.Output["sharedMindOverMatter"] + actor.Output[damageType+"MindOverMatter"]) > 0 {
							MoMDamage := Damage[damageType] * min(actor.Output["sharedMindOverMatter"] + actor.Output[damageType+"MindOverMatter"], 100) / 100
							if modDB.Flag(nil, "EnergyShieldProtectsMana") and energyShield > 0 and DamageIn[damageType+"EnergyShieldBypass"] < 100 {
								tempDamage := min(MoMDamage * (1 - DamageIn[damageType+"EnergyShieldBypass"] / 100), energyShield)
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
								actor.Output["LifeLossBelowHalfLost"] = actor.Output["LifeLossBelowHalfLost"] + Damage[damageType] * actor.Output["preventedLifeLoss"] / 100
							}
							Damage[damageType] = Damage[damageType] * (1 - actor.Output["preventedLifeLoss"] / 100)
						}
						life = life - Damage[damageType]
					}
				}
				if modDB.Flag(nil, "WardNotBreak") {
					ward = restoreWard
				} else if ward > 0 {
					ward = 0
				}
				if DamageIn.GainWhenHit and life > 0 {
					life = min(life + DamageIn.LifeWhenHit, actor.Output["LifeRecoverable"] or 0)
					mana = min(mana + DamageIn.ManaWhenHit, actor.Output["ManaUnreserved"] or 0)
					energyShield = min(energyShield + DamageIn.EnergyShieldWhenHit, actor.Output["EnergyShieldRecoveryCap"])
				}
				itterationMultiplier = 1
				--To speed it up, run recurivly but speed up
				maxDepth := data.misc.ehpCalcMaxDepth
				speedUp := data.misc.ehpCalcSpeedUp
				DamageIn["cyclesRan"] = DamageIn["cyclesRan"] or false
				if not DamageIn["cyclesRan"] and life > 0 and DamageIn["cycles"] < maxDepth {
					Damage = {}
					for _, damageType in ipairs(dmgTypeList) {
						Damage[damageType] = DamageIn[damageType] * speedUp
					}
					Damage["cycles"] = DamageIn["cycles"] * speedUp
					itterationMultiplier = max((numberOfHitsToDie(Damage) - 1) * speedUp - 1, 1)
					DamageIn["cyclesRan"] = true
				}
			}
			if numHits >= maxHits {
				return m_huge
			}
			return numHits
		}
	*/
	/*
		TODO --number of damaging hits needed to be taken to die
		{
			DamageIn := {}
			for _, damageType in ipairs(dmgTypeList) {
				DamageIn[damageType] = actor.Output[damageType+"TakenHit"]
			}
			actor.Output["NumberOfDamagingHits"] = numberOfHitsToDie(DamageIn)
		}


		{
			DamageIn := {}
			BlockChance := 0
			blockEffect := 1
			suppressChance := 0
			suppressionEffect := 1
			ExtraAvoidChance := 0
			averageAvoidChance := 0
			worstOf := env.configInput.EHPUnluckyWorstOf or 1
			--block effect
			if damageCategoryConfig == "Melee" {
				BlockChance = actor.Output["BlockChance"] / 100
			} else {
				BlockChance = actor.Output[damageCategoryConfig+"BlockChance"] / 100
			}
			--unlucky config to lower the value of block, dodge, evade etc for ehp
			if worstOf > 1 {
				BlockChance = BlockChance * BlockChance
				if worstOf == 4 {
					BlockChance = BlockChance * BlockChance
				}
			}
			blockEffect = (1 - BlockChance * actor.Output["BlockEffect"] / 100)
			if not env.configInput.DisableEHPGainOnBlock {
				DamageIn.LifeWhenHit = actor.Output["LifeOnBlock"] * BlockChance
				DamageIn.ManaWhenHit = actor.Output["ManaOnBlock"] * BlockChance
				DamageIn.EnergyShieldWhenHit = actor.Output["EnergyShieldOnBlock"] * BlockChance
				if damageCategoryConfig == "Spell" or damageCategoryConfig == "SpellProjectile" {
					DamageIn.EnergyShieldWhenHit = DamageIn.EnergyShieldWhenHit + actor.Output["EnergyShieldOnSpellBlock"] * BlockChance
				} else if damageCategoryConfig == "Average" {
					DamageIn.EnergyShieldWhenHit = DamageIn.EnergyShieldWhenHit + actor.Output["EnergyShieldOnSpellBlock"] / 2 * BlockChance
				}
			}
			-- suppression
			if damageCategoryConfig == "Spell" or damageCategoryConfig == "SpellProjectile" or damageCategoryConfig == "Average" {
				suppressChance = actor.Output["SpellSuppressionChance"] / 100
			}
			--unlucky config to lower the value of block, dodge, evade etc for ehp
			if worstOf > 1 {
				suppressChance = suppressChance * suppressChance
				if worstOf == 4 {
					suppressChance = suppressChance * suppressChance
				}
			}
			if damageCategoryConfig == "Average" {
				suppressChance = suppressChance / 2
			}
			suppressionEffect = 1 - suppressChance * actor.Output["SpellSuppressionEffect"] / 100
			--extra avoid chance
			if damageCategoryConfig == "Projectile" or damageCategoryConfig == "SpellProjectile" {
				ExtraAvoidChance = ExtraAvoidChance + actor.Output["AvoidProjectilesChance"]
			} else if damageCategoryConfig == "Average" {
				ExtraAvoidChance = ExtraAvoidChance + actor.Output["AvoidProjectilesChance"] / 2
			}
			--gain when hit (currently just gain on block)
			if not env.configInput.DisableEHPGainOnBlock {
				if DamageIn.LifeWhenHit ~= 0 or DamageIn.ManaWhenHit ~= 0 or DamageIn.EnergyShieldWhenHit ~= 0 {
					DamageIn.GainWhenHit = true
				}
			}
			for _, damageType in ipairs(dmgTypeList) {
				 -- Emperor's Vigilance (this needs to fail with divine flesh as it cant override it, hence the check for high bypass)
				if modDB.Flag(nil, "BlockedDamageDoesntBypassES")and actor.Output[damageType+"EnergyShieldBypass"] < 100 and damageType ~= "Chaos"  {
					DamageIn[damageType+"EnergyShieldBypass"] = actor.Output[damageType+"EnergyShieldBypass"] * (1 - BlockChance)
				}
				AvoidChance := min(actor.Output["Avoid"+damageType+"DamageChance"] + ExtraAvoidChance, data.misc.AvoidChanceCap)
				--unlucky config to lower the value of block, dodge, evade etc for ehp
				if worstOf > 1 {
					AvoidChance = AvoidChance / 100 * AvoidChance
					if worstOf == 4 {
						AvoidChance = AvoidChance / 100 * AvoidChance
					}
				}
				averageAvoidChance = averageAvoidChance + AvoidChance
				DamageIn[damageType] = actor.Output[damageType+"TakenHit"] * (blockEffect * suppressionEffect * (1 - AvoidChance / 100))
			}
			--petrified blood degen initialisation
			if actor.Output["preventedLifeLoss"] > 0 {
				actor.Output["LifeLossBelowHalfLost"] = 0
				DamageIn["LifeLossBelowHalfLost"] = modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfLost") / 100
			}
			actor.Output["NumberOfMitigatedDamagingHits"] = numberOfHitsToDie(DamageIn)
			averageAvoidChance = averageAvoidChance / 5
			actor.Output["ConfiguredDamageChance"] = 100 * (blockEffect * suppressionEffect * (1 - averageAvoidChance / 100))
			if breakdown != nil {
				breakdown["ConfiguredDamageChance"] = {
					s_format("%.2f ^8(chance for block to fail)", 1 - BlockChance)
				}
				if actor.Output["ShowBlockEffect"] {
					t_insert(breakdown["ConfiguredDamageChance"], s_format("x %.2f ^8(block effect)", actor.Output["BlockEffect"] / 100))
				}
				if suppressionEffect > 0 {
					t_insert(breakdown["ConfiguredDamageChance"], s_format("x %.3f ^8(suppression effect)", suppressionEffect))
				}
				if averageAvoidChance > 0 {
					t_insert(breakdown["ConfiguredDamageChance"], s_format("x %.2f ^8(chance for avoidance to fail)", 1 - averageAvoidChance / 100))
				}
				t_insert(breakdown["ConfiguredDamageChance"], s_format("= %.1f%% ^8(of damage taken from a%s hit)", actor.Output["ConfiguredDamageChance"], (damageCategoryConfig == "Average" and "n " or " ")+damageCategoryConfig))
			}
		}
	*/
	/*
		TODO --chance to not be hit
		{
			worstOf := env.configInput.EHPUnluckyWorstOf or 1
			actor.Output["MeleeNotHitChance"] = 100 - (1 - actor.Output["MeleeEvadeChance"] / 100) * (1 - actor.Output["AttackDodgeChance"] / 100) * 100
			actor.Output["ProjectileNotHitChance"] = 100 - (1 - actor.Output["ProjectileEvadeChance"] / 100) * (1 - actor.Output["AttackDodgeChance"] / 100) * 100
			actor.Output["SpellNotHitChance"] = 100 - (1 - actor.Output["SpellDodgeChance"] / 100) * 100
			actor.Output["SpellProjectileNotHitChance"] = actor.Output["SpellNotHitChance"]
			actor.Output["AverageNotHitChance"] = (actor.Output["MeleeNotHitChance"] + actor.Output["ProjectileNotHitChance"] + actor.Output["SpellNotHitChance"] + actor.Output["SpellProjectileNotHitChance"]) / 4
			actor.Output["ConfiguredNotHitChance"] = actor.Output[damageCategoryConfig+"NotHitChance"]
			--unlucky config to lower the value of block, dodge, evade etc for ehp
			if worstOf > 1 {
				actor.Output["ConfiguredNotHitChance"] = actor.Output["ConfiguredNotHitChance"] / 100 * actor.Output["ConfiguredNotHitChance"]
				if worstOf == 4 {
					actor.Output["ConfiguredNotHitChance"] = actor.Output["ConfiguredNotHitChance"] / 100 * actor.Output["ConfiguredNotHitChance"]
				}
			}
			actor.Output["TotalNumberOfHits"] = actor.Output["NumberOfMitigatedDamagingHits"] / (1 - actor.Output["ConfiguredNotHitChance"] / 100)
			if breakdown != nil {
				breakdown.ConfiguredNotHitChance = { }
				if damageCategoryConfig == "Melee" or damageCategoryConfig == "Projectile" {
					t_insert(breakdown["ConfiguredNotHitChance"], s_format("%.2f ^8(chance for evasion to fail)", 1 - actor.Output[damageCategoryConfig+"EvadeChance"] / 100))
					t_insert(breakdown["ConfiguredNotHitChance"], s_format("x %.2f ^8(chance for dodge to fail)", 1 - actor.Output["AttackDodgeChance"] / 100))
				} else if damageCategoryConfig == "Spell" or damageCategoryConfig == "SpellProjectile" {
					t_insert(breakdown["ConfiguredNotHitChance"], s_format("%.2f ^8(chance for dodge to fail)", 1 - actor.Output["SpellDodgeChance"] / 100))
				} else if damageCategoryConfig == "Average" {
					t_insert(breakdown["ConfiguredNotHitChance"], s_format("%.2f ^8(chance for evasion to fail, only applies to the attack portion)", 1 - (actor.Output["MeleeEvadeChance"] + actor.Output["ProjectileEvadeChance"]) / 2 / 100))
					t_insert(breakdown["ConfiguredNotHitChance"], s_format("x%.2f ^8(chance for dodge to fail)", 1 - (actor.Output["AttackDodgeChance"] + actor.Output["SpellDodgeChance"]) / 2 / 100))
				}
				if worstOf > 1 {
					t_insert(breakdown["ConfiguredNotHitChance"], s_format("unlucky worst of %d", worstOf))
				}
				t_insert(breakdown["ConfiguredNotHitChance"], s_format("= %d%% ^8(chance to be hit by a%s hit)", 100 - actor.Output["ConfiguredNotHitChance"], (damageCategoryConfig == "Average" and "n " or " ")+damageCategoryConfig))
				breakdown["TotalNumberOfHits"] = {
					s_format("%.2f ^8(Number of mitigated hits)", actor.Output["NumberOfMitigatedDamagingHits"]),
					s_format("/ %.2f ^8(Chance to even be hit)", 1 - actor.Output["ConfiguredNotHitChance"] / 100),
					s_format("= %.2f ^8(total average number of hits you can take)", actor.Output["TotalNumberOfHits"]),
				}
			}
		}
	*/

	// effective hit pool
	actor.Output["TotalEHP"] = actor.Output["TotalNumberOfHits"] * actor.Output["totalEnemyDamageIn"]
	/*
		TODO Breakdown
		if breakdown != nil {
			breakdown["TotalEHP"] = {
				s_format("%.2f ^8(total average number of hits you can take)", actor.Output["TotalNumberOfHits"]),
				s_format("x %d ^8(total incoming damage)", actor.Output["totalEnemyDamageIn"]),
				s_format("= %d ^8(total damage you can take)", actor.Output["TotalEHP"]),
			}
		}
	*/
	/*
		TODO --survival time
		{
			enemySkillTime := env.configInput.enemySpeed or env.configPlaceholder.enemySpeed or 700
			enemyActionSpeed := calcs.actionSpeedMod(actor.enemy)
			enemySkillTime = enemySkillTime / 1000 / enemyActionSpeed
			actor.Output["EHPsurvivalTime"] = actor.Output["TotalNumberOfHits"] * enemySkillTime
			if breakdown != nil {
				breakdown["EHPsurvivalTime"] = {
					s_format("%.2f ^8(total average number of hits you can take)", actor.Output["TotalNumberOfHits"]),
					s_format("x %.2f ^8enemy attack/cast time", enemySkillTime),
					s_format("= %.2f seconds ^8(total time it would take to die)", actor.Output["EHPsurvivalTime"]),
				}
			}
		}
	*/

	// petrified blood "degen"
	if actor.Output["preventedLifeLoss"] > 0 {
		LifeLossBelowHalfLost := modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfLost") / 100
		actor.Output["LifeLossBelowHalfLostMax"] = actor.Output["LifeLossBelowHalfLost"] * LifeLossBelowHalfLost / 4
		actor.Output["LifeLossBelowHalfLostAvg"] = actor.Output["LifeLossBelowHalfLost"] * LifeLossBelowHalfLost / (actor.Output["EHPsurvivalTime"] + 4)
		/*
			TODO Breakdown
			if breakdown != nil {
				breakdown["LifeLossBelowHalfLostMax"] = {
					s_format("%d ^8(total damage prevented by petrified blood)", actor.Output["LifeLossBelowHalfLost"]),
						s_format("* %.2f ^8(percent of damage taken)", LifeLossBelowHalfLost),
						s_format("/ %.2f ^8(over 4 seconds)", 4),
						s_format("= %.2f per second", actor.Output["LifeLossBelowHalfLostMax"]),
				}
				breakdown["LifeLossBelowHalfLostAvg"] = {
					s_format("%d ^8(total damage prevented by petrified blood)", actor.Output["LifeLossBelowHalfLost"]),
						s_format("* %.2f ^8(percent of damage taken)", LifeLossBelowHalfLost),
						s_format("/ %.2f ^8(total time of the degen (survival time + 4))", (actor.Output["EHPsurvivalTime"] + 4)),
						s_format("= %.2f per second", actor.Output["LifeLossBelowHalfLostAvg"]),
				}
			}
		*/
	}

	// effective health pool vs dots
	for _, damageType := range data.DamageType("").Values() {
		actor.Output[string(damageType+"DotEHP")] = actor.Output[string(damageType+"TotalPool")] / actor.Output[string(damageType+"TakenDotMult")]
		/*
			TODO Breakdown
			if breakdown != nil {
				breakdown[damageType+"DotEHP"] = {
					s_format("Total Pool: %d", actor.Output[damageType+"TotalPool"]),
						s_format("Dot Damage Taken modifier: %.2f", actor.Output[damageType+"TakenDotMult"]),
						s_format("Total Effective Dot Pool: %d", actor.Output[damageType+"DotEHP"]),
				}
			}
		*/
	}

	/*
		TODO -- Degens
		for _, damageType in ipairs(dmgTypeList) {
			baseVal := modDB.Sum(mod.TypeBase, nil, damageType+"Degen")
			if baseVal > 0 {
				total := baseVal * actor.Output[damageType+"TakenDotMult"]
				actor.Output[damageType+"Degen"] = total
				actor.Output["TotalDegen"] = (actor.Output["TotalDegen"] or 0) + total
				if breakdown != nil {
					breakdown.TotalDegen = breakdown.TotalDegen or {
						rowList = { },
						colList = {
							{ label = "Type", key = "type" },
							{ label = "Base", key = "base" },
							{ label = "Multiplier", key = "mult" },
							{ label = "Total", key = "total" },
						}
					}
					t_insert(breakdown.TotalDegen.rowList, {
						type = damageType,
						base = s_format("%.1f", baseVal),
						mult = s_format("x %.2f", actor.Output[damageType+"TakenDotMult"]),
						total = s_format("%.1f", total),
					})
					breakdown[damageType+"Degen"] = {
						rowList = { },
						colList = {
							{ label = "Type", key = "type" },
							{ label = "Base", key = "base" },
							{ label = "Multiplier", key = "mult" },
							{ label = "Total", key = "total" },
						}
					}
					t_insert(breakdown[damageType+"Degen"].rowList, {
						type = damageType,
						base = s_format("%.1f", baseVal),
						mult = s_format("x %.2f", actor.Output[damageType+"TakenDotMult"]),
						total = s_format("%.1f", total),
					})
				}
			}
		}
		if actor.Output["TotalDegen"] {
			actor.Output["NetLifeRegen"] = actor.Output["LifeRegen"]
			actor.Output["NetManaRegen"] = actor.Output["ManaRegen"]
			actor.Output["NetEnergyShieldRegen"] = actor.Output["EnergyShieldRegen"]
			totalLifeDegen := 0
			totalManaDegen := 0
			totalEnergyShieldDegen := 0
			if breakdown != nil {
				breakdown.NetLifeRegen = {
						label = "Total Life Degen",
						rowList = { },
						colList = {
							{ label = "Type", key = "type" },
							{ label = "Degen", key = "degen" },
						},
					}
				breakdown.NetManaRegen = {
						label = "Total Mana Degen",
						rowList = { },
						colList = {
							{ label = "Type", key = "type" },
							{ label = "Degen", key = "degen" },
						},
					}
				breakdown.NetEnergyShieldRegen = {
						label = "Total Energy Shield Degen",
						rowList = { },
						colList = {
							{ label = "Type", key = "type" },
							{ label = "Degen", key = "degen" },
						},
					}
			}
			for _, damageType in ipairs(dmgTypeList) {
				if actor.Output[damageType+"Degen"] {
					energyShieldDegen := 0
					lifeDegen := 0
					manaDegen := 0
					takenFromMana := actor.Output[damageType+"MindOverMatter"] + actor.Output["sharedMindOverMatter"]
					if actor.Output["EnergyShieldRegen"] > 0 {
						if modDB.Flag(nil, "EnergyShieldProtectsMana") {
							lifeDegen = actor.Output[damageType+"Degen"] * (1 - takenFromMana / 100)
							energyShieldDegen = actor.Output[damageType+"Degen"] * (1 - actor.Output[damageType+"EnergyShieldBypass"] / 100) * (takenFromMana / 100)
						} else {
							lifeDegen = actor.Output[damageType+"Degen"] * (actor.Output[damageType+"EnergyShieldBypass"] / 100) * (1 - takenFromMana / 100)
							energyShieldDegen = actor.Output[damageType+"Degen"] * (1 - actor.Output[damageType+"EnergyShieldBypass"] / 100)
						}
						manaDegen = actor.Output[damageType+"Degen"] * (actor.Output[damageType+"EnergyShieldBypass"] / 100) * (takenFromMana / 100)
					} else {
						lifeDegen = actor.Output[damageType+"Degen"] * (1 - takenFromMana / 100)
						manaDegen = actor.Output[damageType+"Degen"] * (takenFromMana / 100)
					}
					totalLifeDegen = totalLifeDegen + lifeDegen
					totalManaDegen = totalManaDegen + manaDegen
					totalEnergyShieldDegen = totalEnergyShieldDegen + energyShieldDegen
					if breakdown != nil {
						t_insert(breakdown.NetLifeRegen.rowList, {
							type = s_format("%s", damageType),
							degen = s_format("%.2f", lifeDegen),
						})
						t_insert(breakdown.NetManaRegen.rowList, {
							type = s_format("%s", damageType),
							degen = s_format("%.2f", manaDegen),
						})
						t_insert(breakdown.NetEnergyShieldRegen.rowList, {
							type = s_format("%s", damageType),
							degen = s_format("%.2f", energyShieldDegen),
						})
					}
				}
			}
			actor.Output["NetLifeRegen"] = actor.Output["NetLifeRegen"] - totalLifeDegen
			actor.Output["NetManaRegen"] = actor.Output["NetManaRegen"] - totalManaDegen
			actor.Output["NetEnergyShieldRegen"] = actor.Output["NetEnergyShieldRegen"] - totalEnergyShieldDegen
			actor.Output["TotalNetRegen"] = actor.Output["NetLifeRegen"] + actor.Output["NetManaRegen"] + actor.Output["NetEnergyShieldRegen"]
			if breakdown != nil {
				t_insert(breakdown.NetLifeRegen, s_format("%.1f ^8(total life regen)", actor.Output["LifeRegen"]))
				t_insert(breakdown.NetLifeRegen, s_format("- %.1f ^8(total life degen)", totalLifeDegen))
				t_insert(breakdown.NetLifeRegen, s_format("= %.1f", actor.Output["NetLifeRegen"]))
				t_insert(breakdown.NetManaRegen, s_format("%.1f ^8(total mana regen)", actor.Output["ManaRegen"]))
				t_insert(breakdown.NetManaRegen, s_format("- %.1f ^8(total mana degen)", totalManaDegen))
				t_insert(breakdown.NetManaRegen, s_format("= %.1f", actor.Output["NetManaRegen"]))
				t_insert(breakdown.NetEnergyShieldRegen, s_format("%.1f ^8(total energy shield regen)", actor.Output["EnergyShieldRegen"]))
				t_insert(breakdown.NetEnergyShieldRegen, s_format("- %.1f ^8(total energy shield degen)", totalEnergyShieldDegen))
				t_insert(breakdown.NetEnergyShieldRegen, s_format("= %.1f", actor.Output["NetEnergyShieldRegen"]))
				breakdown.TotalNetRegen = {
					s_format("Net Life Regen: %.1f", actor.Output["NetLifeRegen"]),
					s_format("+ Net Mana Regen: %.1f", actor.Output["NetManaRegen"]),
					s_format("+ Net Energy Shield Regen: %.1f", actor.Output["NetEnergyShieldRegen"]),
					s_format("= Total Net Regen: %.1f", actor.Output["TotalNetRegen"])
				}
			}
		}
	*/
	/*
		TODO --maximum hit taken
		-- this is not done yet, using old max hit taken
		--fix total pools, as they arnt used anymore
		for _, damageType in ipairs(dmgTypeList) {
			--base + petrified blood
			if actor.Output["preventedLifeLoss"] > 0 {
				actor.Output[damageType+"TotalPool"] =  actor.Output[damageType+"TotalPool"] / (1 - actor.Output["preventedLifeLoss"] / 100)
			}
			--ward
			wardBypass := modDB.Sum(mod.TypeBase, nil, "WardBypass") or 0
			if wardBypass > 0 {
				poolProtected := actor.Output["Ward"] / (1 - wardBypass / 100) * (wardBypass / 100)
				sourcePool := actor.Output[damageType+"TotalPool"]
				sourcePool = max(sourcePool - poolProtected, 0) + min(sourcePool, poolProtected) / (wardBypass / 100)
				actor.Output[damageType+"TotalPool"] = sourcePool
			} else {
				actor.Output[damageType+"TotalPool"] = actor.Output[damageType+"TotalPool"] + actor.Output["Ward"] or 0
			}
			--aegis
			actor.Output[damageType+"TotalHitPool"] = actor.Output[damageType+"TotalPool"] + actor.Output[damageType+"Aegis"] or 0 + actor.Output[damageType+"sharedAegis"] or 0 + isElemental[damageType] and actor.Output[damageType+"sharedElementalAegis"] or 0
			--guardskill
			GuardAbsorbRate := actor.Output["sharedGuardAbsorbRate"] or 0 + actor.Output[damageType+"GuardAbsorbRate"] or 0
			if GuardAbsorbRate > 0 {
				GuardAbsorb := actor.Output["sharedGuardAbsorb"] or 0 + actor.Output[damageType+"GuardAbsorb"] or 0
				if GuardAbsorbRate >= 100 {
					actor.Output[damageType+"TotalHitPool"] = actor.Output[damageType+"TotalHitPool"] + GuardAbsorb
				} else {
					poolProtected := GuardAbsorb / (GuardAbsorbRate / 100) * (1 - GuardAbsorbRate / 100)
					actor.Output[damageType+"TotalHitPool"] = max(actor.Output[damageType+"TotalHitPool"] - poolProtected, 0) + min(actor.Output[damageType+"TotalHitPool"], poolProtected) / (1 - GuardAbsorbRate / 100)
				}
			}
			--frost shield
			if actor.Output["FrostShieldLife"] > 0 {
				poolProtected := actor.Output["FrostShieldLife"] / (actor.Output["FrostShieldDamageMitigation"] / 100) * (1 - actor.Output["FrostShieldDamageMitigation"] / 100)
				actor.Output[damageType+"TotalHitPool"] = max(actor.Output[damageType+"TotalHitPool"] - poolProtected, 0) + min(actor.Output[damageType+"TotalHitPool"], poolProtected) / (1 - actor.Output["FrostShieldDamageMitigation"] / 100)
			}
		}
		for _, damageType in ipairs(dmgTypeList) {
			if breakdown != nil {
				breakdown[damageType+"MaximumHitTaken"] = {
					label = "Maximum Hit Taken (uses lowest value)",
					rowList = { },
					colList = {
						{ label = "Type", key = "type" },
						{ label = "TotalPool", key = "pool" },
						{ label = "Taken", key = "taken" },
						{ label = "Final", key = "final" },
					},
				}
			}
			actor.Output[damageType+"MaximumHitTaken"] = m_huge
			for _, damageConvertedType in ipairs(dmgTypeList) {
				if actor.damageShiftTable[damageType][damageConvertedType] > 0 {
					hitTaken := actor.Output[damageConvertedType+"TotalHitPool"] / (actor.damageShiftTable[damageType][damageConvertedType] / 100) / actor.Output[damageConvertedType+"BaseTakenHitMult"]
					if hitTaken < actor.Output[damageType+"MaximumHitTaken"] {
						actor.Output[damageType+"MaximumHitTaken"] = hitTaken
					}
					if breakdown != nil {
						t_insert(breakdown[damageType+"MaximumHitTaken"].rowList, {
							type = s_format("%d%% as %s", actor.damageShiftTable[damageType][damageConvertedType], damageConvertedType),
							pool = s_format("x %d", actor.Output[damageConvertedType+"TotalHitPool"]),
							taken = s_format("/ %.2f", actor.Output[damageConvertedType+"BaseTakenHitMult"]),
							final = s_format("x %.0f", hitTaken),
						})
					}
				}
			}
			if breakdown != nil {
				 t_insert(breakdown[damageType+"MaximumHitTaken"], s_format("Total Pool: %d", actor.Output[damageType+"TotalHitPool"]))
				 t_insert(breakdown[damageType+"MaximumHitTaken"], s_format("Taken Mult: %.2f",  actor.Output[damageType+"TotalHitPool"] / actor.Output[damageType+"MaximumHitTaken"]))
				 t_insert(breakdown[damageType+"MaximumHitTaken"], s_format("Maximum hit you can take: %.0f", actor.Output[damageType+"MaximumHitTaken"]))
			}
		}
	*/
}
