package calculator

import (
	"math"
	"sort"

	"github.com/Vilsol/go-pob/calculator/calclib"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

// PerformCalc
//
// Finalises the environment and performs the stat calculations:
// 1. Merges keystone modifiers
// 2. Initialises minion skills
// 3. Initialises the main skill's minion, if present
// 4. Merges flask effects
// 5. Sets conditions and calculates attributes and life/mana pools (doActorAttribsPoolsConditions)
// 6. Calculates reservations
// 7. Sets life/mana reservation (doActorLifeManaReservation)
// 8. Processes buffs and debuffs
// 9. Processes charges and misc buffs (doActorMisc)
// 10. Calculates defence and offence stats (calcs.defence, calcs.offence)
func PerformCalc(env *Environment) {
	/*
		Kept for reference

		avoidCache := avoidCache or false
		modDB := env.modDB
		enemyDB := env.enemyDB
	*/

	// Merge keystone modifiers
	env.KeystonesAdded = make(map[string]interface{}, 0)
	mergeKeystones(env)

	for _, activeSkill := range env.Player.ActiveSkillList {
		activeSkill.SkillModList = moddb.NewModList()
		activeSkill.SkillModList.Parent = activeSkill.BaseSkillModList
		if activeSkill.Minion != nil {
			/*
				TODO // Build minion skills
				activeSkill.minion.modDB = new("ModDB")
				activeSkill.minion.modDB.actor = activeSkill.minion
				calcs.createMinionSkills(env, activeSkill)
				activeSkill.skillPartName = activeSkill.minion.mainSkill.activeEffect.grantedEffect.name
			*/
		}
	}

	env.Player.Output = make(map[string]float64)
	env.Player.OutputTable = make(map[OutTable]map[string]float64)

	env.Enemy.Output = make(map[string]float64)
	env.Enemy.OutputTable = make(map[OutTable]map[string]float64)

	// Kept for reference
	//
	// local output = env.player.output

	/*
		TODO Minions
		env.minion = env.player.mainSkill.minion
		if env.minion {
			// Initialise minion modifier database
			actor.Output["Minion"] = { }
			env.minion.output = actor.Output["Minion"]
			env.minion.modDB.multipliers["Level"] = env.minion.level
			calcs.initModDB(env, env.minion.modDB)
			env.minion.modDB.NewMod("Life", "BASE", math.Floor(env.minion.lifeTable[env.minion.level] * env.minion.minionData.life), "Base")
			if env.minion.minionData.energyShield {
				env.minion.modDB.NewMod("EnergyShield", "BASE", math.Floor(env.data.monsterAllyLifeTable[env.minion.level] * env.minion.minionData.life * env.minion.minionData.energyShield), "Base")
			}
			if env.minion.minionData.armour {
				env.minion.modDB.NewMod("Armour", "BASE", math.Floor((10 + env.minion.level * 2) * env.minion.minionData.armour * 1.038 ^ env.minion.level), "Base")
			}
			env.minion.modDB.NewMod("Evasion", "BASE", round((30 + env.minion.level * 5) * 1.03 ^ env.minion.level), "Base")
			env.minion.modDB.NewMod("Accuracy", "BASE", round((17 + env.minion.level / 2) * (env.minion.minionData.accuracy or 1) * 1.03 ^ env.minion.level), "Base")
			env.minion.modDB.NewMod("CritMultiplier", "BASE", 30, "Base")
			env.minion.modDB.NewMod("CritDegenMultiplier", "BASE", 30, "Base")
			env.minion.modDB.NewMod("FireResist", "BASE", env.minion.minionData.fireResist, "Base")
			env.minion.modDB.NewMod("ColdResist", "BASE", env.minion.minionData.coldResist, "Base")
			env.minion.modDB.NewMod("LightningResist", "BASE", env.minion.minionData.lightningResist, "Base")
			env.minion.modDB.NewMod("ChaosResist", "BASE", env.minion.minionData.chaosResist, "Base")
			env.minion.modDB.NewMod("CritChance", "INC", 200, "Base", { type = "Multiplier", var = "PowerCharge" })
			env.minion.modDB.NewMod("Speed", "INC", 15, "Base", { type = "Multiplier", var = "FrenzyCharge" })
			env.minion.modDB.NewMod("Damage", "MORE", 4, "Base", { type = "Multiplier", var = "FrenzyCharge" })
			env.minion.modDB.NewMod("MovementSpeed", "INC", 5, "Base", { type = "Multiplier", var = "FrenzyCharge" })
			env.minion.modDB.NewMod("PhysicalDamageReduction", "BASE", 15, "Base", { type = "Multiplier", var = "EnduranceCharge" })
			env.minion.modDB.NewMod("ElementalResist", "BASE", 15, "Base", { type = "Multiplier", var = "EnduranceCharge" })
			env.minion.modDB.NewMod("ProjectileCount", "BASE", 1, "Base")
			env.minion.modDB.NewMod("MaximumFortification", "BASE", 20, "Base")
			env.minion.modDB.NewMod("Damage", "MORE", -50, "Base", 0, KeywordFlag.Poison)
			env.minion.modDB.NewMod("Damage", "MORE", -50, "Base", 0, KeywordFlag.Ignite)
			env.minion.modDB.NewMod("SkillData", "LIST", { key = "bleedBasePercent", value = 70/6 }, "Base")
			env.minion.modDB.NewMod("Damage", "MORE", 200, "Base", 0, KeywordFlag.Bleed, { type = "ActorCondition", actor = "enemy", var = "Moving" })
			for _, mod in ipairs(env.minion.minionData.modList) {
				env.minion.modDB.AddMod(mod)
			}
			for _, mod in ipairs(env.player.mainSkill.extraSkillModList) {
				env.minion.modDB.AddMod(mod)
			}
			if env.aegisModList {
				env.minion.itemList["Weapon 3"] = env.player.itemList["Weapon 2"]
				env.minion.modDB.AddList(env.aegisModList)
			}
			if env.theIronMass and env.minion.type == "RaisedSkeleton" {
				env.minion.modDB.AddList(env.theIronMass)
			}
			if env.player.mainSkill.skillData.minionUseBowAndQuiver {
				if env.player.weaponData1.type == "Bow" {
					env.minion.modDB.AddList(env.player.itemList["Weapon 1"].slotModList[1])
				}
				if env.player.itemList["Weapon 2"] and env.player.itemList["Weapon 2"].type == "Quiver" {
					env.minion.modDB.AddList(env.player.itemList["Weapon 2"].modList)
				}
			}
			if env.minion.itemSet or env.minion.uses {
				for slotName, slot in pairs(env.build.itemsTab.slots) {
					if env.minion.uses[slotName] {
						local item
						if env.minion.itemSet {
							if slot.weaponSet == 1 and env.minion.itemSet.useSecondWeaponSet {
								slotName = slotName + " Swap"
							}
							item = env.build.itemsTab.items[env.minion.itemSet[slotName].selItemId]
						} else {
							item = env.player.itemList[slotName]
						}
						if item {
							env.minion.itemList[slotName] = item
							env.minion.modDB.AddList(item.modList or item.slotModList[slot.slotNum])
						}
					}
				}
			}
			if modDB.Flag(nil, "StrengthAddedToMinions") {
				env.minion.modDB.NewMod("Str", "BASE", round(calcLib.val(modDB, "Str")), "Player")
			}
			if modDB.Flag(nil, "HalfStrengthAddedToMinions") {
				env.minion.modDB.NewMod("Str", "BASE", round(calcLib.val(modDB, "Str") * 0.5), "Player")
			}
		}
	*/

	/*
		TODO Aegis
		if env.aegisModList {
			env.player.itemList["Weapon 2"] = nil
		}
	*/

	// AlchemistsGenius
	if env.ModDB.Flag(nil, "AlchemistsGenius") {
		effectMod := 1 + env.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100
		env.ModDB.AddMod(mod.NewFloat("FlaskEffect", mod.TypeIncrease, math.Floor(10*effectMod)).Source("Alchemist's Genius"))
		env.ModDB.AddMod(mod.NewFloat("FlaskChargesGained", mod.TypeIncrease, math.Floor(20*effectMod)).Source("Alchemist's Genius"))
	}

	for _, activeSkill := range env.Player.ActiveSkillList {
		if activeSkill.SkillFlags[SkillFlagBrand] {
			attachLimit := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "BrandsAttachedLimit")
			attached := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:ConfigBrandsAttachedToEnemy")
			activeBrands := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:ConfigActiveBrands")
			actual := min(attachLimit, attached)
			// Cap the number of active brands by the limit, which is 3 by default
			env.ModDB.Multipliers["ActiveBrand"] = min(activeBrands, env.ModDB.Sum(mod.TypeBase, nil, "ActiveBrandLimit"))
			env.ModDB.Multipliers["BrandsAttachedToEnemy"] = max(actual, env.ModDB.Multipliers["BrandsAttachedToEnemy"])
			env.EnemyModDB.Multipliers["BrandsAttached"] = max(actual, env.EnemyModDB.Multipliers["BrandsAttached"])
		}

		// The actual hexes as opposed to hex related skills all have the curse flag. TotemCastsWhenNotDetached is to remove blasphemy
		// Note that this doesn't work for triggers yet, insufficient support
		if activeSkill.SkillFlags[SkillFlagHex] && activeSkill.SkillFlags[SkillFlagCurse] && !activeSkill.SkillTypes[data.SkillTypeTotemCastsWhenNotDetached] {
			hexDoom := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:HexDoomStack")
			maxDoom := activeSkill.SkillModList.Sum(mod.TypeBase, nil, "MaxDoom")
			if maxDoom == 0 {
				maxDoom = 30
			}
			doomEffect := activeSkill.SkillModList.More(nil, "DoomEffect")
			// Update the max doom limit
			env.Player.Output["HexDoomLimit"] = max(maxDoom, env.Player.Output["HexDoomLimit"])
			// Update the Hex Doom to apply
			activeSkill.SkillModList.AddMod(mod.NewFloat("CurseEffect", mod.TypeIncrease, min(hexDoom, maxDoom)*doomEffect).Source("Doom"))
			env.ModDB.Multipliers["HexDoom"] = min(max(hexDoom, env.ModDB.Multipliers["HexDoom"]), env.Player.Output["HexDoomLimit"])
		}

		if activeSkill.SkillData.SupportBonechill {
			if activeSkill.SkillTypes[data.SkillTypeChillingArea] || ((activeSkill.SkillTypes[data.SkillTypeNonHitChill] && !activeSkill.SkillModList.Flag(nil, "CannotChill")) &&
				!(activeSkill.ActiveEffect.GrantedEffect.Raw.GetActiveSkill().DisplayedName == "Summon Skitterbots" && activeSkill.SkillModList.Flag(nil, "SkitterbotsCannotChill"))) {
				env.Player.Output["BonechillDotEffect"] = math.Floor(*data.NonDamagingAilments[data.AilmentChill].Default * (1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, nil, "EnemyChillEffect")/100))
			}
			env.Player.Output["BonechillEffect"] = max(env.Player.Output["BonechillEffect"], env.EnemyModDB.Sum(mod.TypeBase, nil, "BonechillEffect"), env.Player.Output["BonechillDotEffect"])
		}

		// Vaal Lightning Trap
		if activeSkill.ActiveEffect.GrantedEffect.Raw.ID == "Vaal Lightning Trap" || activeSkill.ActiveEffect.GrantedEffect.Raw.ID == "Shock Ground" {
			env.ModDB.AddMod(mod.NewFloat("ShockOverride", mod.TypeBase, activeSkill.SkillModList.Sum(mod.TypeBase, nil, "ShockedGroundEffect")).Source("Shocked Ground").Tag(mod.ActorCondition("enemy", "OnShockedGround")))
		}

		// Summon Skitterbots
		if activeSkill.ActiveEffect.GrantedEffect.Raw.ID == "Summon Skitterbots" {
			if !activeSkill.SkillModList.Flag(nil, "SkitterbotsCannotShock") {
				effect := *data.NonDamagingAilments[data.AilmentShock].Default * (1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, &moddb.ListCfg{
					Source: utils.Ptr(mod.SourceSkill),
				}, "EnemyShockEffect")/100)
				env.ModDB.AddMod(mod.NewFloat("ShockOverride", mod.TypeBase, effect).Source("Summon Skitterbots"))
				env.ModDB.AddMod(mod.NewFlag("Condition:Shocked", true).Source("Summon Skitterbots"))
			}
			if !activeSkill.SkillModList.Flag(nil, "SkitterbotsCannotChill") {
				effect := *data.NonDamagingAilments[data.AilmentChill].Default * (1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, &moddb.ListCfg{
					Source: utils.Ptr(mod.SourceSkill),
				}, "EnemyChillEffect")/100)
				env.ModDB.AddMod(mod.NewFloat("ChillOverride", mod.TypeBase, effect).Source("Summon Skitterbots"))
				env.ModDB.AddMod(mod.NewFlag("Condition:Chilled", true).Source("Summon Skitterbots"))
				if activeSkill.SkillData.SupportBonechill {
					env.Player.Output["BonechillEffect"] = max(env.Player.Output["BonechillEffect"], effect)
				}
			}
		}
		if activeSkill.SkillModList.Flag(nil, "Condition:CanWither") {
			effect := utils.Ternary(activeSkill.Minion != nil, 6, math.Floor(6*(1+env.ModDB.Sum(mod.TypeIncrease, nil, "WitherEffect")/100)))
			env.ModDB.AddMod(mod.NewFloat("WitherEffectStack", mod.TypeMAX, effect))
		}
		/*
			TODO Warcry
			if activeSkill.skillFlags.warcry and not modDB.Flag(nil, "AlreadyGlobalWarcryCooldown") {
				cooldown := calcSkillCooldown(activeSkill.skillModList, activeSkill.skillCfg, activeSkill.skillData)
				warcryList := { }
				numWarcries, sumWarcryCooldown := 0
				for _, activeSkill in ipairs(env.player.activeSkillList) {
					if activeSkill.skillTypes[SkillType.Warcry] {
						warcryList[activeSkill.skillCfg.skillName] = true
					}
				}
				for _, warcry in pairs(warcryList) {
					numWarcries = numWarcries + 1
					sumWarcryCooldown = (sumWarcryCooldown or 0) + cooldown
				}
				env.player.modDB.NewMod("GlobalWarcryCooldown", "BASE", sumWarcryCooldown)
				env.player.modDB.NewMod("GlobalWarcryCount", "BASE", numWarcries)
				modDB.NewMod("AlreadyGlobalWarcryCooldown", "FLAG", true, "Config") // Prevents effect from applying multiple times
			}
		*/
		/*
			TODO Minion
			if activeSkill.minion and activeSkill.minion.minionData and activeSkill.minion.minionData.limit {
				limit := activeSkill.skillModList:Sum(mod.TypeBase, nil, activeSkill.minion.minionData.limit)
				actor.Output[activeSkill.minion.minionData.limit] = max(limit, actor.Output[activeSkill.minion.minionData.limit] or 0)
			}
		*/
		/*
			TODO Buffs
			if env.mode_buffs and activeSkill.skillFlags.warcry {
				extraExertions := activeSkill.skillModList:Sum(mod.TypeBase, nil, "ExtraExertedAttacks") or 0
				full_duration := calcSkillDuration(activeSkill.skillModList, activeSkill.skillCfg, activeSkill.skillData, env, enemyDB)
				cooldownOverride := activeSkill.skillModList:Override(activeSkill.skillCfg, "CooldownRecovery")
				actual_cooldown := cooldownOverride or (activeSkill.skillData.cooldown  + activeSkill.skillModList:Sum(mod.TypeBase, activeSkill.skillCfg, "CooldownRecovery")) / calcLib.mod(activeSkill.skillModList, activeSkill.skillCfg, "CooldownRecovery")
				globalCooldown := modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
				globalCount := modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
				uptime := min(full_duration / actual_cooldown, 1)
				buff_inc := 1 + activeSkill.skillModList:Sum(mod.TypeIncrease, activeSkill.skillCfg, "BuffEffect") / 100
				warcryPowerBonus := math.Floor((modDB.Override(nil, "WarcryPower") or modDB.Sum(mod.TypeBase, nil, "WarcryPower") or 0) / 5)
				if modDB.Flag(nil, "WarcryShareCooldown") {
					uptime = min(full_duration / (actual_cooldown + (globalCooldown - actual_cooldown) / globalCount), 1)
				}
				if modDB.Flag(nil, "Condition:WarcryMaxHit") {
					uptime = 1
				}
				if activeSkill.activeEffect.grantedEffect.name == "Ancestral Cry" and not modDB.Flag(nil, "AncestralActive") {
					ancestralArmour := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "AncestralArmourPer5MP")
					ancestralArmourMax := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "AncestralArmourMax")
					ancestralArmourIncrease := activeSkill.skillModList:Sum(mod.TypeIncrease, env.player.mainSkill.skillCfg, "AncestralArmourMax")
					ancestralStrikeRange := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "AncestralMeleeWeaponRangePer5MP")
					ancestralStrikeRangeMax := math.Floor(6 * buff_inc)
					env.player.modDB.NewMod("NumAncestralExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "AncestralExertedAttacks") + extraExertions)
					ancestralArmourMax = math.Floor(ancestralArmourMax * buff_inc)
					if warcryPowerBonus ~= 0 {
						ancestralArmour = math.Floor(ancestralArmour * warcryPowerBonus * buff_inc) / warcryPowerBonus
						ancestralStrikeRange = math.Floor(ancestralStrikeRange * warcryPowerBonus * buff_inc) / warcryPowerBonus
					} else {
						// Since no buff happens, you don't get the divergent increase.
						ancestralArmourIncrease = 0
					}
					env.player.modDB.NewMod("Armour", "BASE", ancestralArmour * uptime, "Ancestral Cry", { type = "Multiplier", var = "WarcryPower", div = 5, limit = ancestralArmourMax, limitTotal = true })
					env.player.modDB.NewMod("Armour", "INC", ancestralArmourIncrease * uptime, "Ancestral Cry")
					env.player.modDB.NewMod("MeleeWeaponRange", "BASE", ancestralStrikeRange * uptime, "Ancestral Cry", { type = "Multiplier", var = "WarcryPower", div = 5, limit = ancestralStrikeRangeMax, limitTotal = true })
					modDB.NewMod("AncestralActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Enduring Cry" and not modDB.Flag(nil, "EnduringActive") {
					heal_over_1_sec := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "EnduringCryLifeRegen")
					resist_all_per_endurance := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "EnduringCryElementalResist")
					pdr_per_endurance := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "EnduringCryPhysicalDamageReduction")
					env.player.modDB.NewMod("LifeRegen", "BASE", heal_over_1_sec, "Enduring Cry", { type = "Condition", var = "LifeRegenBurstFull" })
					env.player.modDB.NewMod("LifeRegen", "BASE", heal_over_1_sec / actual_cooldown, "Enduring Cry", { type = "Condition", var = "LifeRegenBurstAvg" })
					env.player.modDB.NewMod("ElementalResist", "BASE", math.Floor(resist_all_per_endurance * buff_inc) * uptime, "Enduring Cry", { type = "Multiplier", var = "EnduranceCharge" })
					env.player.modDB.NewMod("PhysicalDamageReduction", "BASE", math.Floor(pdr_per_endurance * buff_inc) * uptime, "Enduring Cry", { type = "Multiplier", var = "EnduranceCharge" })
					modDB.NewMod("EnduringActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Infernal Cry" and not modDB.Flag(nil, "InfernalActive") {
					infernalAshEffect := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "InfernalFireTakenPer5MP")
					env.player.modDB.NewMod("NumInfernalExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "InfernalExertedAttacks") + extraExertions)
					if env.mode_effective {
						env.player.modDB.NewMod("CoveredInAshEffect", "BASE", infernalAshEffect * uptime, { type = "Multiplier", var = "WarcryPower", div = 5 })
					}
					modDB.NewMod("InfernalActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Battlemage's Cry" and not modDB.Flag(nil, "BattlemageActive") {
					battlemageSpellToAttack := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "BattlemageSpellIncreaseApplyToAttackPer5MP")
					battlemageSpellToAttackMax := math.Floor(150 * buff_inc)
					battlemageCritChance := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "BattlemageCritChancePer5MP")
					battlemageCritChanceMax := math.Floor(30 * buff_inc)
					env.player.modDB.NewMod("NumBattlemageExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "BattlemageExertedAttacks") + extraExertions)
					if warcryPowerBonus ~= 0 {
						battlemageCritChance = math.Floor(battlemageCritChance * warcryPowerBonus * buff_inc) / warcryPowerBonus
						battlemageSpellToAttack = math.Floor(battlemageSpellToAttack * warcryPowerBonus * buff_inc) / warcryPowerBonus
						modDB.NewMod("SpellDamageAppliesToAttacks", "FLAG", true)
					}
					env.player.modDB.NewMod("CritChance", "INC", battlemageCritChance * uptime, "Battlemage's Cry", { type = "Multiplier", var = "WarcryPower", div = 5, limit = battlemageCritChanceMax, limitTotal = true })
					env.player.modDB.NewMod("ImprovedSpellDamageAppliesToAttacks", "MAX", battlemageSpellToAttack * uptime, "Battlemage's Cry", { type = "Multiplier", var = "WarcryPower", div = 5, limit = battlemageSpellToAttackMax, limitTotal = true })
					modDB.NewMod("BattlemageActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Intimidating Cry" and not modDB.Flag(nil, "IntimidatingActive") {
					intimidatingOverwhelmEffect := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "IntimidatingPDRPer5MP")
					if warcryPowerBonus ~= 0 {
						intimidatingOverwhelmEffect = math.Floor(intimidatingOverwhelmEffect * warcryPowerBonus * buff_inc) / warcryPowerBonus
					}
					env.player.modDB.NewMod("NumIntimidatingExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "IntimidatingExertedAttacks") + extraExertions)
					env.player.modDB.NewMod("EnemyPhysicalDamageReduction", "BASE", -intimidatingOverwhelmEffect * uptime, "Intimidating Cry Buff", { type = "Multiplier", var = "WarcryPower", div = 5, limit = 6 })
					modDB.NewMod("IntimidatingActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Rallying Cry" and not modDB.Flag(nil, "RallyingActive") {
					env.player.modDB.NewMod("NumRallyingExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "RallyingExertedAttacks") + extraExertions)
					env.player.modDB.NewMod("RallyingExertMoreDamagePerAlly",  "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "RallyingCryExertDamageBonus"))
					rallyingWeaponEffect := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "RallyingCryAllyDamageBonusPer5Power")
					// Rallying cry divergent more effect of buff
					rallyingBonusMoreMultiplier := 1 + (activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "RallyingCryMinionDamageBonusMultiplier") or 0)
					if warcryPowerBonus ~= 0 {
						rallyingWeaponEffect = math.Floor(rallyingWeaponEffect * warcryPowerBonus * buff_inc) / warcryPowerBonus
					}
					// Special handling for the minion side to add the flat damage bonus
					if env.minion {
						// Add all damage types
						dmgTypeList := {"Physical", "Lightning", "Cold", "Fire", "Chaos"}
						for _, damageType in ipairs(dmgTypeList) {
							env.minion.modDB.NewMod(damageType+"Min", "BASE", math.Floor((env.player.weaponData1[damageType+"Min"] or 0) * rallyingBonusMoreMultiplier * rallyingWeaponEffect / 100) * uptime, "Rallying Cry", { type = "Multiplier", actor = "parent", var = "WarcryPower", div = 5, limit = 6.6667})
							env.minion.modDB.NewMod(damageType+"Max", "BASE", math.Floor((env.player.weaponData1[damageType+"Max"] or 0) * rallyingBonusMoreMultiplier * rallyingWeaponEffect / 100) * uptime, "Rallying Cry", { type = "Multiplier", actor = "parent", var = "WarcryPower", div = 5, limit = 6.6667})
						}
					}
					modDB.NewMod("RallyingActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Seismic Cry" and not modDB.Flag(nil, "SeismicActive") {
					seismicStunEffect := activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "SeismicStunThresholdPer5MP")
					if warcryPowerBonus ~= 0 {
						seismicStunEffect = math.Floor(seismicStunEffect * warcryPowerBonus * buff_inc) / warcryPowerBonus
					}
					env.player.modDB.NewMod("NumSeismicExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "SeismicExertedAttacks") + extraExertions)
					env.player.modDB.NewMod("SeismicIncAoEPerExert",  "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "SeismicAoEMultiplier"))
					if env.mode_effective {
						env.player.modDB.NewMod("EnemyStunThreshold", "INC", -seismicStunEffect * uptime, "Seismic Cry Buff", { type = "Multiplier", var = "WarcryPower", div = 5, limit = 6 })
					}
					modDB.NewMod("SeismicActive", "FLAG", true) // Prevents effect from applying multiple times
				}
			}
		*/
		/*
			TODO Triggers
			if activeSkill.skillData.triggeredByBrand and not activeSkill.skillFlags.minion {
				activeSkill.skillData.triggered = true
				spellCount, quality := 0
				for _, skill in ipairs(env.player.activeSkillList) {
					match1 := skill.activeEffect.grantedEffect.fromItem and skill.socketGroup.slot == activeSkill.socketGroup.slot
					match2 := not skill.activeEffect.grantedEffect.fromItem and skill.socketGroup == activeSkill.socketGroup
					if skill.skillData.triggeredByBrand and (match1 or match2) {
						spellCount = spellCount + 1
					}
					if skill.activeEffect.grantedEffect.name == "Arcanist Brand" and (match1 or match2) {
						quality = skill.activeEffect.quality / 2
					}
				}
				addTriggerIncMoreMods(activeSkill, env.player.mainSkill)
				activeSkill.skillModList:NewMod("ArcanistSpellsLinked", "BASE", spellCount, "Skill")
				activeSkill.skillModList:NewMod("BrandActivationFrequency", "INC", quality, "Skill")
			}
			if activeSkill.skillData.triggeredOnDeath and not activeSkill.skillFlags.minion {
				activeSkill.skillData.triggered = true
				for _, value in ipairs(activeSkill.skillModList:Tabulate("INC", env.player.mainSkill.skillCfg, "TriggeredDamage")) {
					activeSkill.skillModList:NewMod("Damage", "INC", value.mod.value, value.mod.source, value.mod.flags, value.mod.keywordFlags, unpack(value.mod))
				}
				for _, value in ipairs(activeSkill.skillModList:Tabulate("MORE", env.player.mainSkill.skillCfg, "TriggeredDamage")) {
					activeSkill.skillModList:NewMod("Damage", "MORE", value.mod.value, value.mod.source, value.mod.flags, value.mod.keywordFlags, unpack(value.mod))
				}
				// Set trigger time to 1 min in ms ( == 6000 ). Technically any large value would do.
				activeSkill.skillData.triggerTime = 60 * 1000
			}
		*/
		/*
			TODO // The Saviour
			if activeSkill.activeEffect.grantedEffect.name == "Reflection" or activeSkill.skillData.triggeredBySaviour {
				activeSkill.infoMessage = "Triggered by a Crit from The Saviour"
				activeSkill.infoTrigger = "Saviour"
			}
		*/
	}

	/*
		TODO Breakdown Module
		breakdown := nil
		if env.mode == "CALCS" {
			// Initialise breakdown module
			breakdown = LoadModule(calcs.breakdownModule, modDB, output, env.player)
			env.player.breakdown = breakdown
			if env.minion {
				env.minion.breakdown = LoadModule(calcs.breakdownModule, env.minion.modDB, env.minion.output, env.minion)
			}
		}
	*/

	/*
		TODO // Special handling of Mageblood
		maxActiveMagicUtilityCount := modDB.Sum(mod.TypeBase, nil, "ActiveMagicUtilityFlasks")
		if maxActiveMagicUtilityCount > 0 {
			curActiveMagicUtilityCount := 0
			for _, slot in pairs(env.build.itemsTab.orderedSlots) {
				slotName := slot.slotName
				item := env.build.itemsTab.items[slot.selItemId]
				if item and item.type == "Flask" {
					mageblood_applies := item.rarity == "MAGIC" and not (item.baseName:match("Life Flask") or
						item.baseName:match("Mana Flask") or item.baseName:match("Hybrid Flask")) and
						curActiveMagicUtilityCount < maxActiveMagicUtilityCount
					if mageblood_applies {
						env.flasks[item] = true
						curActiveMagicUtilityCount = curActiveMagicUtilityCount + 1
					}
				}
			}
		}
	*/

	/*
		TODO // Merge flask modifiers
		if env.mode_combat {
			effectInc := modDB.Sum(mod.TypeIncrease, nil, "FlaskEffect")
			flaskBuffs := { }
			usingFlask := false
			usingLifeFlask := false
			usingManaFlask := false
			for item in pairs(env.flasks) {
				usingFlask = true
				if item.baseName:match("Life Flask") {
					usingLifeFlask = true
				}
				if item.baseName:match("Mana Flask") {
					usingManaFlask = true
				}
				if item.baseName:match("Hybrid Flask") {
					usingLifeFlask = true
					usingManaFlask = true
				}

				flaskEffectInc := item.flaskData.effectInc
				if item.rarity == "MAGIC" and not (usingLifeFlask or usingManaFlask) {
					flaskEffectInc = flaskEffectInc + modDB.Sum(mod.TypeIncrease, nil, "MagicUtilityFlaskEffect")
				}

				// Avert thine eyes, lest they be forever scarred
				// I have no idea how to determine which buff is applied by a given flask,
				// so utility flasks are grouped by base, unique flasks are grouped by name, and magic flasks by their modifiers
				effectMod := 1 + (effectInc + flaskEffectInc) / 100
				if item.buffModList[1] {
					srcList := new("ModList")
					srcList:ScaleAddList(item.buffModList, effectMod)
					mergeBuff(srcList, flaskBuffs, item.baseName)
				}
				if item.modList[1] {
					srcList := new("ModList")
					srcList:ScaleAddList(item.modList, effectMod)
					local key
					if item.rarity == "UNIQUE" {
						key = item.title
					} else {
						key = ""
						for _, mod in ipairs(item.modList) {
							key = key + modLib.formatModParams(mod) + "&"
						}
					}
					mergeBuff(srcList, flaskBuffs, key)
				}
			}
			if not modDB.Flag(nil, "FlasksDoNotApplyToPlayer") {
				modDB.conditions["UsingFlask"] = usingFlask
				modDB.conditions["UsingLifeFlask"] = usingLifeFlask
				modDB.conditions["UsingManaFlask"] = usingManaFlask
				for _, buffModList in pairs(flaskBuffs) {
					modDB.AddList(buffModList)
				}
			}
			if env.minion and modDB.Flag(env.player.mainSkill.skillCfg, "FlasksApplyToMinion") {
				minionModDB := env.minion.modDB
				minionModDB.conditions["UsingFlask"] = usingFlask
				minionModDB.conditions["UsingLifeFlask"] = usingLifeFlask
				minionModDB.conditions["UsingManaFlask"] = usingManaFlask
				for _, buffModList in pairs(flaskBuffs) {
					minionModDB:AddList(buffModList)
				}
			}
		}
	*/

	// Merge keystones again to catch any that were added by flasks
	mergeKeystones(env)

	// Calculate attributes and life/mana pools
	doActorAttribsPoolsConditions(env, env.Player)

	/*
		TODO Calculate minion attributes and life/mana pools
		if env.minion {
			for _, value in ipairs(env.player.mainSkill.skillModList:List(env.player.mainSkill.skillCfg, "MinionModifier")) {
				if not value.type or env.minion.type == value.type {
					env.minion.modDB.AddMod(value.mod)
				}
			}
			for _, name in ipairs(env.minion.modDB.List(nil, "Keystone")) {
				if env.spec.tree.keystoneMap[name] {
					env.minion.modDB.AddList(env.spec.tree.keystoneMap[name].modList)
				}
			}
			doActorAttribsPoolsConditions(env, env.minion)
		}
	*/

	/*
		TODO // Calculate skill life and mana reservations
		env.player.reserved_LifeBase = 0
		env.player.reserved_LifePercent = modDB.Sum(mod.TypeBase, nil, "ExtraLifeReserved")
		env.player.reserved_ManaBase = 0
		env.player.reserved_ManaPercent = 0
		if breakdown {
			breakdown.LifeReserved = { reservations = { } }
			breakdown.ManaReserved = { reservations = { } }
		}
		for _, activeSkill in ipairs(env.player.activeSkillList) {
			if activeSkill.skillTypes[SkillType.HasReservation] and not activeSkill.skillTypes[SkillType.ReservationBecomesCost] {
				skillModList := activeSkill.skillModList
				skillCfg := activeSkill.skillCfg
				mult := skillModList:More(skillCfg, "SupportManaMultiplier")
				pool := { ["Mana"] = { }, ["Life"] = { } }
				pool.Mana.baseFlat = activeSkill.skillData.manaReservationFlat or activeSkill.activeEffect.grantedEffectLevel.manaReservationFlat or 0
				if skillModList:Flag(skillCfg, "ManaCostGainAsReservation") and activeSkill.activeEffect.grantedEffectLevel.cost {
					pool.Mana.baseFlat = skillModList:Sum(mod.TypeBase, skillCfg, "ManaCostBase") + (activeSkill.activeEffect.grantedEffectLevel.cost.Mana or 0)
				}
				pool.Mana.basePercent = activeSkill.skillData.manaReservationPercent or activeSkill.activeEffect.grantedEffectLevel.manaReservationPercent or 0
				pool.Life.baseFlat = activeSkill.skillData.lifeReservationFlat or activeSkill.activeEffect.grantedEffectLevel.lifeReservationFlat or 0
				if skillModList:Flag(skillCfg, "LifeCostGainAsReservation") and activeSkill.activeEffect.grantedEffectLevel.cost {
					pool.Life.baseFlat = skillModList:Sum(mod.TypeBase, skillCfg, "LifeCostBase") + (activeSkill.activeEffect.grantedEffectLevel.cost.Life or 0)
				}
				pool.Life.basePercent = activeSkill.skillData.lifeReservationPercent or activeSkill.activeEffect.grantedEffectLevel.lifeReservationPercent or 0
				if skillModList:Flag(skillCfg, "BloodMagicReserved") {
					pool.Life.baseFlat = pool.Life.baseFlat + pool.Mana.baseFlat
					pool.Mana.baseFlat = 0
					activeSkill.skillData["LifeReservationFlatForced"] = activeSkill.skillData["ManaReservationFlatForced"]
					activeSkill.skillData["ManaReservationFlatForced"] = nil
					pool.Life.basePercent = pool.Life.basePercent + pool.Mana.basePercent
					pool.Mana.basePercent = 0
					activeSkill.skillData["LifeReservationPercentForced"] = activeSkill.skillData["ManaReservationPercentForced"]
					activeSkill.skillData["ManaReservationPercentForced"] = nil
				}
				for name, values in pairs(pool) {
					values.more = skillModList:More(skillCfg, name+"Reserved", "Reserved")
					values.inc = skillModList:Sum(mod.TypeIncrease, skillCfg, name+"Reserved", "Reserved")
					values.efficiency = max(skillModList:Sum(mod.TypeIncrease, skillCfg, name+"ReservationEfficiency", "ReservationEfficiency"), -100)
					// used for Arcane Cloak calculations in ModStore.GetStat
					env.player[name+"Efficiency"] = values.efficiency
					if activeSkill.skillData[name+"ReservationFlatForced"] {
						values.reservedFlat = activeSkill.skillData[name+"ReservationFlatForced"]
					} else {
						baseFlatVal := math.Floor(values.baseFlat * mult)
						values.reservedFlat = 0
						if values.more > 0 and values.inc > -100 and baseFlatVal ~= 0 {
							values.reservedFlat = max(round(baseFlatVal * (100 + values.inc) / 100 * values.more / (1 + values.efficiency / 100), 0), 0)
						}
					}
					if activeSkill.skillData[name+"ReservationPercentForced"] {
						values.reservedPercent = activeSkill.skillData[name+"ReservationPercentForced"]
					} else {
						basePercentVal := values.basePercent * mult
						values.reservedPercent = 0
						if values.more > 0 and values.inc > -100 and basePercentVal ~= 0 {
							values.reservedPercent = max(round(basePercentVal * (100 + values.inc) / 100 * values.more / (1 + values.efficiency / 100), 2), 0)
						}
					}
					if activeSkill.activeMineCount {
						values.reservedFlat = values.reservedFlat * activeSkill.activeMineCount
						values.reservedPercent = values.reservedPercent * activeSkill.activeMineCount
					}
					if values.reservedFlat ~= 0 {
						activeSkill.skillData[name+"ReservedBase"] = values.reservedFlat
						env.player["reserved_"+name+"Base"] = env.player["reserved_"+name+"Base"] + values.reservedFlat
						if breakdown {
							t_insert(breakdown[name+"Reserved"].reservations, {
								skillName = activeSkill.activeEffect.grantedEffect.name,
								base = values.baseFlat,
								mult = mult ~= 1 and ("x "+mult),
								more = values.more ~= 1 and ("x "+values.more),
								inc = values.inc ~= 0 and ("x "+(1 + values.inc / 100)),
								efficiency = values.efficiency ~= 0 and ("x " + 1 / (1 + values.efficiency / 100)),
								total = values.reservedFlat,
							})
						}
					}
					if values.reservedPercent ~= 0 {
						activeSkill.skillData[name+"ReservedPercent"] = values.reservedPercent
						activeSkill.skillData[name+"ReservedBase"] = (activeSkill.skillData[name+"ReservedBase"] or 0) + m_ceil(actor.Output[name] * values.reservedPercent / 100)
						env.player["reserved_"+name+"Percent"] = env.player["reserved_"+name+"Percent"] + values.reservedPercent
						if breakdown {
							t_insert(breakdown[name+"Reserved"].reservations, {
								skillName = activeSkill.activeEffect.grantedEffect.name,
								base = values.basePercent + "%",
								mult = mult ~= 1 and ("x "+mult),
								more = values.more ~= 1 and ("x "+values.more),
								inc = values.inc ~= 0 and ("x "+(1 + values.inc / 100)),
								efficiency = values.efficiency ~= 0 and ("x " + 1 / (1 + values.efficiency / 100)),
								total = values.reservedPercent + "%",
							})
						}
					}
				}
			}
		}
	*/

	// Set the life/mana reservations
	doActorLifeManaReservation(env.Player)
	if env.Minion != nil {
		/*
			TODO Minion
			doActorLifeManaReservation(env.minion)
		*/
	}

	// Process attribute requirements
	reqMult := calclib.Mod(env.ModDB, nil, "GlobalAttributeRequirements")
	attrTable := utils.Ternary(env.ModDB.Flag(nil, "OmniscienceRequirements"), []string{"Omni", "Str", "Dex", "Int"}, []string{"Str", "Dex", "Int"})
	for _, attr := range attrTable {
		breakdownAttr := attr
		if env.ModDB.Flag(nil, "OmniscienceRequirements") {
			breakdownAttr = "Omni"
		}
		/*
			TODO Breakdown
			if breakdown {
				breakdown["Req"+attr] = {
					rowList = { },
					colList = {
						{ label = attr, key = "req" },
						{ label = "Source", key = "source" },
						{ label = "Source Name", key = "sourceName" },
					}
				}
			}
		*/
		out := float64(0)
		for _, reqSource := range env.RequirementsTable {
			attrVal := float64(utils.GetOr(reqSource, attr, 0))
			if attrVal > 0 {
				req := math.Floor(attrVal * reqMult)
				if env.ModDB.Flag(nil, "OmniscienceRequirements") {
					omniReqMult := 1 / (calclib.Mod(env.ModDB, nil, "OmniAttributeRequirements") - 1)
					attributereq := math.Floor(attrVal * reqMult)
					req = math.Floor(attributereq * omniReqMult)
				}
				out = max(out, req)
				/*
					TODO Breakdown
					if breakdown {
						row := {
							req = req > actor.Output[breakdownAttr] and colorCodes.NEGATIVE+req or req,
							reqNum = req,
							source = reqSource.source,
						}
						if reqSource.source == "Item" {
							item := reqSource.sourceItem
							row.sourceName = colorCodes[item.rarity]+item.name
							row.sourceNameTooltip = function(tooltip)
								env.build.itemsTab:AddItemTooltip(tooltip, item, reqSource.sourceSlot)
							}
						} else if reqSource.source == "Gem" {
							row.sourceName = s_format("%s%s ^7%d/%d", reqSource.sourceGem.color, reqSource.sourceGem.nameSpec, reqSource.sourceGem.level, reqSource.sourceGem.quality)
						}
						t_insert(breakdown["Req"+breakdownAttr].rowList, row)
					}
				*/
			}
		}
		if env.ModDB.Flag(nil, "IgnoreAttributeRequirements") {
			out = 0
		}
		env.Player.Output["Req"+attr+"String"] = 0
		if out > (env.Player.Output["Req"+breakdownAttr]) {
			env.Player.Output["Req"+breakdownAttr+"String"] = out
			env.Player.Output["Req"+breakdownAttr] = out
			/*
				TODO Breakdown
				if breakdown {
					actor.Output["Req"+breakdownAttr+"String"] = out > (actor.Output[breakdownAttr] or 0) and colorCodes.NEGATIVE+out or out
				}
			*/
		}
	}
	/*
		TODO Breakdown
		if breakdown and breakdown["ReqOmni"] {
			table.sort(breakdown["ReqOmni"].rowList, function(a, b)
				if a.reqNum ~= b.reqNum {
					return a.reqNum > b.reqNum
				} else if a.source ~= b.source {
					return a.source < b.source
				} else {
					return a.sourceName < b.sourceName
				}
			end)
		}
	*/

	/*
		TODO // Calculate number of active heralds
		if env.mode_buffs {
			heraldList := { }
			for _, activeSkill in ipairs(env.player.activeSkillList) {
				if activeSkill.skillTypes[SkillType.Herald] and not heraldList[activeSkill.skillCfg.skillName] {
					heraldList[activeSkill.skillCfg.skillName] = true
					modDB.multipliers["Herald"] = (modDB.multipliers["Herald"] or 0) + 1
					modDB.conditions["AffectedByHerald"] = true
				}
			}
		}
	*/

	/*
		TODO // Calculate number of active auras affecting self
		if env.mode_buffs {
			auraList := { }
			for _, activeSkill in ipairs(env.player.activeSkillList) {
				if activeSkill.skillTypes[SkillType.Aura] and not activeSkill.skillTypes[SkillType.RemoteMined] and not activeSkill.skillData.auraCannotAffectSelf and not auraList[activeSkill.skillCfg.skillName] {
					auraList[activeSkill.skillCfg.skillName] = true
					modDB.multipliers["AuraAffectingSelf"] = (modDB.multipliers["AuraAffectingSelf"] or 0) + 1
				}
			}
		}
	*/

	/*
		TODO // Deal with Consecrated Ground
		if modDB.Flag(nil, "Condition:OnConsecratedGround") {
			effect := 1 + modDB.Sum(mod.TypeIncrease, nil, "ConsecratedGroundEffect") / 100
			modDB.NewMod("LifeRegenPercent", "BASE", 5 * effect, "Consecrated Ground")
			modDB.NewMod("CurseEffectOnSelf", "INC", -50 * effect, "Consecrated Ground")
		}
	*/

	/*
		TODO // Maximum Mana conversion from Lightning Mastery
		if modDB.Flag(nil, "ManaAppliesToShockEffect") {
			multiplier := (modDB.Max(nil, "ImprovedManaAppliesToShockEffect") or 100) / 100
			for _, value in ipairs(modDB.Tabulate("INC", nil, "Mana")) {
				mod := value.mod
				modifiers := calcLib.getConvertedModTags(mod, multiplier)
				modDB.NewMod("EnemyShockEffect", "INC", math.Floor(mod.value * multiplier), mod.source, mod.flags, mod.keywordFlags, unpack(modifiers))
			}
		}
	*/

	/*
		TODO // Combine buffs/debuffs
		buffs := { }
		env.buffs = buffs
		guards := { }
		minionBuffs := { }
		env.minionBuffs = minionBuffs
		debuffs := { }
		env.debuffs = debuffs
		curses := { }
		minionCurses := {
			limit = 1,
		}
		for spectreId = 1, #env.spec.build.spectreList {
			spectreData := data.minions[env.spec.build.spectreList[spectreId]]
			for modId = 1, #spectreData.modList {
				modData := spectreData.modList[modId]
				if modData.name == "EnemyCurseLimit" {
					minionCurses.limit = modData.value + 1
					break
				}
			}
		}
		affectedByAura := { }
		for _, activeSkill in ipairs(env.player.activeSkillList) {
			skillModList := activeSkill.skillModList
			skillCfg := activeSkill.skillCfg
			for _, buff in ipairs(activeSkill.buffList) {
				//Skip adding buff if reservation exceeds maximum
				for _, value in ipairs({"Mana", "Life"}) {
					if activeSkill.skillData[value+"ReservedBase"] and activeSkill.skillData[value+"ReservedBase"] > env.player.output[value] {
						goto disableAura
					}
				}
				if buff.cond and not skillModList:GetCondition(buff.cond, skillCfg) {
					// Nothing!
				} else if buff.enemyCond and not enemyDB:GetCondition(buff.enemyCond) {
					// Also nothing :/
				} else if buff.type == "Buff" {
					if env.mode_buffs and (not activeSkill.skillFlags.totem or buff.allowTotemBuff) {
						skillCfg := buff.activeSkillBuff and skillCfg
						modStore := buff.activeSkillBuff and skillModList or modDB
					 	if not buff.applyNotPlayer {
							activeSkill.buffSkill = true
							modDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
							srcList := new("ModList")
							inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer") + skillModList:Sum(mod.TypeIncrease, skillCfg, buff.name:gsub(" ", "")+"Effect")
							more := modStore:More(skillCfg, "BuffEffect", "BuffEffectOnSelf")
							srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
							mergeBuff(srcList, buffs, buff.name)
							mergeBuff(buff.unscalableModList, buffs, buff.name)
							if activeSkill.skillData.thisIsNotABuff {
								buffs[buff.name].notBuff = true
							}
						}
						if env.minion and (buff.applyMinions or buff.applyAllies) {
							activeSkill.minionBuffSkill = true
							env.minion.modDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
							srcList := new("ModList")
							inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnMinion") + env.minion.modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
							more := modStore:More(skillCfg, "BuffEffect", "BuffEffectOnMinion") * env.minion.modDB.More(nil, "BuffEffectOnSelf")
							srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
							mergeBuff(srcList, minionBuffs, buff.name)
							mergeBuff(buff.unscalableModList, minionBuffs, buff.name)
						}
					}
				} else if buff.type == "Guard" {
					if env.mode_buffs and (not activeSkill.skillFlags.totem or buff.allowTotemBuff) {
						skillCfg := buff.activeSkillBuff and skillCfg
						modStore := buff.activeSkillBuff and skillModList or modDB
					 	if not buff.applyNotPlayer {
							activeSkill.buffSkill = true
							srcList := new("ModList")
							inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer")
							more := modStore:More(skillCfg, "BuffEffect", "BuffEffectOnSelf")
							srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
							mergeBuff(srcList, guards, buff.name)
							mergeBuff(buff.unscalableModList, guards, buff.name)
						}
					}
				} else if buff.type == "Aura" {
					if env.mode_buffs {
						// Check for extra modifiers to apply to aura skills
						extraAuraModList := { }
						for _, value in ipairs(modDB.List(skillCfg, "ExtraAuraEffect")) {
							add := true
							for _, mod in ipairs(extraAuraModList) {
								if modLib.compareModParams(mod, value.mod) {
									mod.value = mod.value + value.mod.value
									add = false
									break
								}
							}
							if add {
								t_insert(extraAuraModList, copyTable(value.mod, true))
							}
						}
						if not activeSkill.skillData.auraCannotAffectSelf {
							activeSkill.buffSkill = true
							affectedByAura[env.player] = true
							if buff.name:sub(1,4) == "Vaal" {
								modDB.conditions["AffectedBy"+buff.name:sub(6):gsub(" ","")] = true
							}
							modDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
							srcList := new("ModList")
							inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
							more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
							mult := (1 + inc / 100) * more
							srcList:ScaleAddList(buff.modList, mult)
							srcList:ScaleAddList(extraAuraModList, mult)
							mergeBuff(srcList, buffs, buff.name)
						}
						if env.minion and not (modDB.Flag(nil, "SelfAurasCannotAffectAllies") or modDB.Flag(nil, "SelfAurasOnlyAffectYou") or modDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) {
							activeSkill.minionBuffSkill = true
							affectedByAura[env.minion] = true
							env.minion.modDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
							srcList := new("ModList")
							inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect") + env.minion.modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
							more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect") * env.minion.modDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
							mult := (1 + inc / 100) * more
							srcList:ScaleAddList(buff.modList, mult)
							srcList:ScaleAddList(extraAuraModList, mult)
							mergeBuff(srcList, minionBuffs, buff.name)
						}
					}
				} else if buff.type == "Debuff" or buff.type == "AuraDebuff" {
					local stackCount
					if buff.stackVar {
						stackCount = skillModList:Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.stackVar)
						if buff.stackLimit {
							stackCount = min(stackCount, buff.stackLimit)
						} else if buff.stackLimitVar {
							stackCount = min(stackCount, skillModList:Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.stackLimitVar))
						}
					} else {
						stackCount = activeSkill.skillData.stackCount or 1
					}
					if env.mode_effective and stackCount > 0 {
						activeSkill.debuffSkill = true
						modDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
						srcList := new("ModList")
						mult := 1
						if buff.type == "AuraDebuff" {
							mult = 0
							if not modDB.Flag(nil, "SelfAurasOnlyAffectYou") {
								inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
								more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
								mult = (1 + inc / 100) * more
							}
						}
						if buff.type == "Debuff" {
							inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "DebuffEffect")
							more := skillModList:More(skillCfg, "DebuffEffect")
							mult = (1 + inc / 100) * more
						}
						srcList:ScaleAddList(buff.modList, mult * stackCount)
						if activeSkill.skillData.stackCount or buff.stackVar {
							srcList:NewMod("Multiplier:"+buff.name+"Stack", "BASE", stackCount, buff.name)
						}
						mergeBuff(srcList, debuffs, buff.name)
					}
				} else if buff.type == "Curse" or buff.type == "CurseBuff" {
					mark := activeSkill.skillTypes[SkillType.Mark]
					if env.mode_effective and (not enemyDB:Flag(nil, "Hexproof") or modDB.Flag(nil, "CursesIgnoreHexproof")) or mark {
						curse := {
							name = buff.name,
							fromPlayer = true,
							priority = determineCursePriority(buff.name, activeSkill),
							isMark = mark,
							ignoreHexLimit = modDB.Flag(activeSkill.skillCfg, "CursesIgnoreHexLimit") and not mark or false,
							socketedCursesHexLimit = modDB.Flag(activeSkill.skillCfg, "SocketedCursesAdditionalLimit")
						}
						inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "CurseEffect") + enemyDB:Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf")
						if activeSkill.skillTypes[SkillType.Aura] {
							inc = inc + skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect")
						}
						more := skillModList:More(skillCfg, "CurseEffect")
						// This is non-ideal, but the only More for enemy is the boss effect
						if not curse.isMark {
							more = more * enemyDB:More(nil, "CurseEffectOnSelf")
						}
						mult := 0
						if not (modDB.Flag(nil, "SelfAurasOnlyAffectYou") and activeSkill.skillTypes[SkillType.Aura]) then //If your aura only effect you blasphemy does nothing
							mult = (1 + inc / 100) * more
						}
						if buff.type == "Curse" {
							curse.modList = new("ModList")
							curse.modList:ScaleAddList(buff.modList, mult)
						} else {
							// Curse applies a buff; scale by curse effect, then buff effect
							temp := new("ModList")
							temp:ScaleAddList(buff.modList, mult)
							curse.buffModList = new("ModList")
							buffInc := modDB.Sum(mod.TypeIncrease, skillCfg, "BuffEffectOnSelf")
							buffMore := modDB.More(skillCfg, "BuffEffectOnSelf")
							curse.buffModList:ScaleAddList(temp, (1 + buffInc / 100) * buffMore)
							if env.minion {
								curse.minionBuffModList = new("ModList")
								buffInc := env.minion.modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
								buffMore := env.minion.modDB.More(nil, "BuffEffectOnSelf")
								curse.minionBuffModList:ScaleAddList(temp, (1 + buffInc / 100) * buffMore)
							}
						}
						t_insert(curses, curse)
					}
				}
				::disableAura::
			}
			if activeSkill.minion and activeSkill.minion.activeSkillList {
				castingMinion := activeSkill.minion
				for _, activeSkill in ipairs(activeSkill.minion.activeSkillList) {
					skillModList := activeSkill.skillModList
					skillCfg := activeSkill.skillCfg
					for _, buff in ipairs(activeSkill.buffList) {
						if buff.type == "Buff" {
							if env.mode_buffs and activeSkill.skillData.enable {
								skillCfg := buff.activeSkillBuff and skillCfg
								modStore := buff.activeSkillBuff and skillModList or castingMinion.modDB
								if buff.applyAllies {
									modDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
									srcList := new("ModList")
									inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect") + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
									more := modStore:More(skillCfg, "BuffEffect") * modDB.More(nil, "BuffEffectOnSelf")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, buffs, buff.name)
									mergeBuff(buff.unscalableModList, buffs, buff.name)
								}
								if env.minion and (env.minion == castingMinion or buff.applyAllies) {
					 				env.minion.modDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
									srcList := new("ModList")
									inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnSelf")
									more := modStore:More(skillCfg, "BuffEffect", "BuffEffectOnSelf")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, minionBuffs, buff.name)
									mergeBuff(buff.unscalableModList, minionBuffs, buff.name)
								}
							}
						} else if buff.type == "Aura" {
							if env.mode_buffs and activeSkill.skillData.enable {
								if not modDB.Flag(nil, "AlliesAurasCannotAffectSelf") {
									srcList := new("ModList")
									inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect") + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect") * modDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, buffs, buff.name)
								}
								if env.minion and (env.minion ~= activeSkill.minion or not activeSkill.skillData.auraCannotAffectSelf) {
									srcList := new("ModList")
									inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect") + env.minion.modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect") * env.minion.modDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, minionBuffs, buff.name)
								}
							}
						} else if buff.type == "Curse" {
							if env.mode_effective and activeSkill.skillData.enable and (not enemyDB:Flag(nil, "Hexproof") or activeSkill.skillTypes[SkillType.Mark]) {
								curse := {
									name = buff.name,
									priority = determineCursePriority(buff.name, activeSkill),
								}
								inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "CurseEffect") + enemyDB:Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf")
								more := skillModList:More(skillCfg, "CurseEffect") * enemyDB:More(nil, "CurseEffectOnSelf")
								curse.modList = new("ModList")
								curse.modList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
								t_insert(minionCurses, curse)
							}
						} else if buff.type == "Debuff" {
							local stackCount
							if buff.stackVar {
								stackCount = modDB.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.stackVar)
								if buff.stackLimit {
									stackCount = min(stackCount, buff.stackLimit)
								} else if buff.stackLimitVar {
									stackCount = min(stackCount, modDB.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.stackLimitVar))
								}
							} else {
								stackCount = activeSkill.skillData.stackCount or 1
							}
							if env.mode_effective and stackCount > 0 {
								activeSkill.debuffSkill = true
								srcList := new("ModList")
								srcList:ScaleAddList(buff.modList, stackCount)
								if activeSkill.skillData.stackCount {
									srcList:NewMod("Multiplier:"+buff.name+"Stack", "BASE", activeSkill.skillData.stackCount, buff.name)
								}
								mergeBuff(srcList, debuffs, buff.name)
							}
						}
					}
				}
			}
		}
	*/

	/*
		TODO // Limited support for handling buffs originating from Spectres
		for _, activeSkill in ipairs(env.player.activeSkillList) {
			if activeSkill.minion {
				for _, activeMinionSkill in ipairs(activeSkill.minion.activeSkillList) {
					if activeMinionSkill.skillData.enable {
						skillModList := activeMinionSkill.skillModList
						skillCfg := activeMinionSkill.skillCfg
						for _, buff in ipairs(activeMinionSkill.buffList) {
							if buff.type == "Buff" {
								if buff.applyAllies {
									activeMinionSkill.buffSkill = true
									modDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
									srcList := new("ModList")
									inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnPlayer")
									more := skillModList:More(skillCfg, "BuffEffect", "BuffEffectOnPlayer")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, buffs, buff.name)
									mergeBuff(buff.modList, buffs, buff.name)
									if activeMinionSkill.skillData.thisIsNotABuff {
										buffs[buff.name].notBuff = true
									}
								}
								if buff.applyMinions {
									activeMinionSkill.minionBuffSkill = true
									activeSkill.minion.modDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
									srcList := new("ModList")
									inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnMinion")
									more := skillModList:More(skillCfg, "BuffEffect", "BuffEffectOnMinion")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, minionBuffs, buff.name)
									mergeBuff(buff.modList, minionBuffs, buff.name)
									if activeMinionSkill.skillData.thisIsNotABuff {
										buffs[buff.name].notBuff = true
									}
								}
							}
						}
					}
				}
			}
		}
	*/

	/*
		TODO // Check for extra curses
		for dest, modDB in pairs({[curses] = modDB, [minionCurses] = env.minion and env.minion.modDB}) {
			for _, value in ipairs(modDB.List(nil, "ExtraCurse")) {
				gemModList := new("ModList")
				grantedEffect := env.data.skills[value.skillId]
				if grantedEffect {
					calcs.mergeSkillInstanceMods(env, gemModList, {
						grantedEffect = grantedEffect,
						level = value.level,
						quality = 0,
					})
					curseModList := { }
					for _, mod in ipairs(gemModList) {
						for _, tag in ipairs(mod) {
							if tag.type == "GlobalEffect" and tag.effectType == "Curse" {
								t_insert(curseModList, mod)
								break
							}
						}
					}
					if value.applyToPlayer {
						// Sources for curses on the player don't usually respect any kind of limit, so there's little point bothering with slots
						if modDB.Sum(mod.TypeBase, nil, "AvoidCurse") < 100 {
							modDB.conditions["Cursed"] = true
							modDB.multipliers["CurseOnSelf"] = (modDB.multipliers["CurseOnSelf"] or 0) + 1
							modDB.conditions["AffectedBy"+grantedEffect.name:gsub(" ","")] = true
							cfg := { skillName = grantedEffect.name }
							inc := modDB.Sum(mod.TypeIncrease, cfg, "CurseEffectOnSelf") + gemModList:Sum(mod.TypeIncrease, nil, "CurseEffectAgainstPlayer")
							more := modDB.More(cfg, "CurseEffectOnSelf") * gemModList:More(nil, "CurseEffectAgainstPlayer")
							modDB.ScaleAddList(curseModList, (1 + inc / 100) * more)
						}
					} else if not enemyDB:Flag(nil, "Hexproof") or modDB.Flag(nil, "CursesIgnoreHexproof") {
						curse := {
							name = grantedEffect.name,
							fromPlayer = (dest == curses),
							priority = determineCursePriority(grantedEffect.name),
						}
						curse.modList = new("ModList")
						curse.modList:ScaleAddList(curseModList, (1 + enemyDB:Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf") / 100) * enemyDB:More(nil, "CurseEffectOnSelf"))
						t_insert(dest, curse)
					}
				}
			}
		}

		// Set curse limit
		actor.Output["EnemyCurseLimit"] = modDB.Sum(mod.TypeBase, nil, "EnemyCurseLimit")
		curses.limit = actor.Output["EnemyCurseLimit"]
		// Assign curses to slots
		curseSlots := { }
		env.curseSlots = curseSlots
		// Currently assume only 1 mark is possible
		markSlotted := false
		for _, source in ipairs({curses, minionCurses}) {
			for _, curse in ipairs(source) {
				// Calculate curses that ignore hex limit after
				if not curse.ignoreHexLimit and not curse.socketedCursesHexLimit {
					local slot
					skipAddingCurse := false
					// Check if we need to disable a certain curse aura.
					for _, activeSkill in ipairs(env.player.activeSkillList) {
						if (activeSkill.buffList[1] and curse.name == activeSkill.buffList[1].name and activeSkill.skillTypes[SkillType.Aura]) {
							if modDB.Flag(nil, "SelfAurasOnlyAffectYou") {
								skipAddingCurse = true
								break
							}
							for _, value in ipairs({"Mana", "Life"}) {
								if activeSkill.skillData[value+"ReservedBase"] and activeSkill.skillData[value+"ReservedBase"] > env.player.output[value] {
									skipAddingCurse = true
									break
								}
							}
							break
						}
					}
					for i = 1, source.limit {
						// Prevent multiple marks from being considered
						if curse.isMark {
							if markSlotted {
								slot = nil
								break
							}
						}
						if not curseSlots[i] {
							slot = i
							break
						} else if curseSlots[i].name == curse.name {
							if curseSlots[i].priority < curse.priority {
								slot = i
							} else {
								slot = nil
							}
							break
						} else if curseSlots[i].priority < curse.priority {
							slot = i
						}
					}
					if slot {
						if curseSlots[slot] and curseSlots[slot].isMark {
							markSlotted = false
						}
						if skipAddingCurse == false {
							curseSlots[slot] = curse
						}
						if curse.isMark {
							markSlotted = true
						}
					}
				}
			}
		}

		for _, source in ipairs({curses, minionCurses}) {
			for _, curse in ipairs(source) {
				if curse.ignoreHexLimit {
					skipAddingCurse := false
					for i = 1, #curseSlots {
						if curseSlots[i].name == curse.name {
							// if curse is higher priority, replace current curse with it, otherwise if same or lower priority skip it entirely
							if curseSlots[i].priority < curse.priority {
								curseSlots[i] = curse
							}
							skipAddingCurse = true
							break
						}
					}
					if not skipAddingCurse {
						curseSlots[#curseSlots + 1] = curse
					}
				}
				if curse.socketedCursesHexLimit {
					socketedCursesHexLimitValue := modDB.Sum(mod.TypeBase, nil, "SocketedCursesHexLimitValue")
					skipAddingCurse := false
					for i = 1, #curseSlots {
						if curseSlots[i].name == curse.name {
							// if curse is higher priority, replace current curse with it, otherwise if same or lower priority skip it entirely
							if curseSlots[i].priority < curse.priority {
								curseSlots[i] = curse
							}
							skipAddingCurse = true
							break
						}
						if i >= socketedCursesHexLimitValue {
							skipAddingCurse = true
						}
					}
					if not skipAddingCurse {
						curseSlots[#curseSlots + 1] = curse
					}
				}
			}
		}
	*/

	/*
		TODO // Process guard buffs
		guardSlots := { }
		nonVaal := false
		for name, modList in pairs(guards) {
			if name == "Vaal Molten Shell" {
				wipeTable(guardSlots)
				nonVaal = false
				t_insert(guardSlots, { name = name, modList = modList })
				break
			} else if name:match("^Vaal") {
				t_insert(guardSlots, { name = name, modList = modList })
			} else if not nonVaal {
				t_insert(guardSlots, { name = name, modList = modList })
				nonVaal = true
			}
		}
		if nonVaal {
			modDB.conditions["AffectedByNonVaalGuardSkill"] = true
		}
		for _, guard in ipairs(guardSlots) {
			modDB.conditions["AffectedByGuardSkill"] = true
			modDB.conditions["AffectedBy"+guard.name:gsub(" ","")] = true
			mergeBuff(guard.modList, buffs, guard.name)
		}
	*/

	/*
		TODO // Apply buff/debuff modifiers
		for _, modList in pairs(buffs) {
			modDB.AddList(modList)
			if not modList.notBuff {
				modDB.multipliers["BuffOnSelf"] = (modDB.multipliers["BuffOnSelf"] or 0) + 1
			}
			if env.minion {
				for _, value in ipairs(modList:List(env.player.mainSkill.skillCfg, "MinionModifier")) {
					if not value.type or env.minion.type == value.type {
						env.minion.modDB.AddMod(value.mod)
					}
				}
			}
		}
		if env.minion {
			for _, modList in pairs(minionBuffs) {
				env.minion.modDB.AddList(modList)
			}
		}
		for _, modList in pairs(debuffs) {
			enemyDB:AddList(modList)
		}
		modDB.multipliers["CurseOnEnemy"] = #curseSlots
		affectedByCurse := { }
		for _, slot in ipairs(curseSlots) {
			enemyDB.conditions["Cursed"] = true
			if slot.isMark {
				enemyDB.conditions["Marked"] = true
			}
			if slot.fromPlayer {
				affectedByCurse[env.enemy] = true
			}
			if slot.modList {
				enemyDB:AddList(slot.modList)
			}
			if slot.buffModList {
				modDB.AddList(slot.buffModList)
			}
			if slot.minionBuffModList {
				env.minion.modDB.AddList(slot.minionBuffModList)
			}
		}
	*/

	/*
		TODO // Do another pass on the SkillList to catch effects of buffs, if needed
		for _, activeSkill in ipairs(env.player.activeSkillList) {
			if activeSkill.activeEffect.grantedEffect.name == "Blight" and activeSkill.skillPart == 2 {
				rate := (1 / activeSkill.activeEffect.grantedEffect.castTime) * calcLib.mod(activeSkill.skillModList, activeSkill.skillCfg, "Speed") * calcs.actionSpeedMod(env.player)
				duration := calcSkillDuration(activeSkill.skillModList, activeSkill.skillCfg, activeSkill.skillData, env, enemyDB)
				maximum := min((math.Floor(rate * duration) - 1), 19)
				activeSkill.skillModList:NewMod("Multiplier:BlightMaxStages", "BASE", maximum, "Base")
				activeSkill.skillModList:NewMod("Multiplier:BlightStageAfterFirst", "BASE", maximum, "Base")
			}
			if activeSkill.activeEffect.grantedEffect.name == "Penance Brand" and activeSkill.skillPart == 2 {
				rate := 1 / (activeSkill.skillData.repeatFrequency / (1 + env.player.mainSkill.skillModList:Sum(mod.TypeIncrease, env.player.mainSkill.skillCfg, "Speed", "BrandActivationFrequency") / 100) / activeSkill.skillModList:More(activeSkill.skillCfg, "BrandActivationFrequency"))
				duration := calcSkillDuration(activeSkill.skillModList, activeSkill.skillCfg, activeSkill.skillData, env, enemyDB)
				ticks := min((math.Floor(rate * duration) - 1), 19)
				activeSkill.skillModList:NewMod("Multiplier:PenanceBrandMaxStages", "BASE", ticks, "Base")
				activeSkill.skillModList:NewMod("Multiplier:PenanceBrandStageAfterFirst", "BASE", ticks, "Base")
			}
			if activeSkill.activeEffect.grantedEffect.name == "Scorching Ray" and activeSkill.skillPart == 2 {
				rate := (1 / activeSkill.activeEffect.grantedEffect.castTime) * calcLib.mod(activeSkill.skillModList, activeSkill.skillCfg, "Speed") * calcs.actionSpeedMod(env.player)
				duration := calcSkillDuration(activeSkill.skillModList, activeSkill.skillCfg, activeSkill.skillData, env, enemyDB)
				maximum := min((math.Floor(rate * duration) - 1), 7)
				activeSkill.skillModList:NewMod("Multiplier:ScorchingRayMaxStages", "BASE", maximum, "Base")
				activeSkill.skillModList:NewMod("Multiplier:ScorchingRayStageAfterFirst", "BASE", maximum, "Base")
				if maximum >= 7 {
					activeSkill.skillModList:NewMod("Condition:ScorchingRayMaxStages", "FLAG", true, "Config")
					enemyDB:NewMod("FireResist", "BASE", -25, "Scorching Ray", { type = "GlobalEffect", effectType = "Debuff" } )
				}
			}
		}
	*/

	/*
		TODO // Process Triggered Skill and Set Trigger Conditions
		// Cospri's Malice
		if env.player.mainSkill.skillData.triggeredByCospris and not env.player.mainSkill.skillFlags.minion {
			spellCount := {}
			icdr := calcLib.mod(env.player.mainSkill.skillModList, env.player.mainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill in ipairs(env.player.activeSkillList) {
				if skill.skillTypes[SkillType.Melee] and band(skill.skillCfg.flags, bor(ModFlag.Sword, ModFlag.Weapon1H)) > 0 and skill ~= env.player.mainSkill {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredByCospris and env.player.mainSkill.socketGroup.slot == skill.socketGroup.slot {
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = skill.skillData.cooldown / icdr, next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.player.mainSkill.skillData.triggeredByCospris = nil
				env.player.mainSkill.infoMessage = "No Cospri Triggering Skill Found"
				env.player.mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.player.mainSkill.infoTrigger = ""
			} else {
				env.player.mainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed
				dualWield := false

				sourceAPS, dualWield = calcDualWieldImpact(env, sourceAPS, source.skillData.doubleHitsWhenDualWielding)

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown, dualWield)

				// Account for chance to hit/crit
				sourceCritChance := GlobalCache.cachedData["CACHE"][uuid].CritChance
				trigRate = trigRate * sourceCritChance / 100
				if breakdown {
					breakdown.Speed = {
						s_format("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						s_format("x %.2f%% ^8(%s effective crit chance)", sourceCritChance, source.activeEffect.grantedEffect.name),
						s_format("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.player.mainSkill, env.player.mainSkill)
				env.player.mainSkill.skillData.triggerRate = trigRate
				env.player.mainSkill.skillData.triggerSource = source
				env.player.mainSkill.infoMessage = "Cospri Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.player.mainSkill.infoTrigger = "Cospri"
			}
		}

		// Mjolner
		if env.player.mainSkill.skillData.triggeredByMjolner and not env.player.mainSkill.skillFlags.minion {
			spellCount := {}
			icdr := calcLib.mod(env.player.mainSkill.skillModList, env.player.mainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill in ipairs(env.player.activeSkillList) {
				if (skill.skillTypes[SkillType.Damage] or skill.skillTypes[SkillType.Attack]) and band(skill.skillCfg.flags, bor(ModFlag.Mace, ModFlag.Weapon1H)) > 0 and skill ~= env.player.mainSkill {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredByMjolner and env.player.mainSkill.socketGroup.slot == skill.socketGroup.slot {
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = skill.skillData.cooldown / icdr, next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.player.mainSkill.skillData.triggeredByMjolner = nil
				env.player.mainSkill.infoMessage = "No Mjolner Triggering Skill Found"
				env.player.mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.player.mainSkill.infoTrigger = ""
			} else {
				env.player.mainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed
				dualWield := false

				sourceAPS, dualWield = calcDualWieldImpact(env, sourceAPS, source.skillData.doubleHitsWhenDualWielding)

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown, dualWield)

				// Account for chance to hit/crit
				sourceHitChance := GlobalCache.cachedData["CACHE"][uuid].HitChance
				trigRate = trigRate * sourceHitChance / 100
				if breakdown {
					breakdown.Speed = {
						s_format("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						s_format("x %.0f%% ^8(%s hit chance)", sourceHitChance, source.activeEffect.grantedEffect.name),
						s_format("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.player.mainSkill, env.player.mainSkill)
				env.player.mainSkill.skillData.triggerRate = trigRate
				env.player.mainSkill.skillData.triggerSource = source
				env.player.mainSkill.infoMessage = "Mjolner Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.player.mainSkill.infoTrigger = "Mjolner"
			}
		}

		// Mirage Archer Support
		// This creates and populates env.player.mainSkill.mirage table
		if env.player.mainSkill.skillData.triggeredByMirageArcher and not env.player.mainSkill.skillFlags.minion and not env.player.mainSkill.marked {
			usedSkill := nil
			uuid := cacheSkillUUID(env.player.mainSkill)
			calcMode := env.mode == "CALCS" and "CALCS" or "MAIN"

			// cache a new copy of this skill that's affected by Mirage Archer
			if avoidCache {
				usedSkill = env.player.mainSkill
				env.dontCache = true
			} else {
				if not GlobalCache.cachedData[calcMode][uuid] {
					calcs.buildActiveSkill(env, calcMode, env.player.mainSkill, true)
				}

				if GlobalCache.cachedData[calcMode][uuid] and not avoidCache {
					usedSkill = GlobalCache.cachedData[calcMode][uuid].ActiveSkill
				}
			}

			if usedSkill {
				moreDamage :=  usedSkill.skillModList:Sum(mod.TypeBase, usedSkill.skillCfg, "MirageArcherLessDamage")
				moreAttackSpeed := usedSkill.skillModList:Sum(mod.TypeBase, usedSkill.skillCfg, "MirageArcherLessAttackSpeed")
				mirageCount :=  usedSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "MirageArcherMaxCount")

				// Make a copy of this skill so we can add new modifiers to the copy affected by Mirage Archers
				newSkill, newEnv := calcs.copyActiveSkill(env, calcMode, usedSkill)

				// Add new modifiers to new skill (which already has all the old skill's modifiers)
				newSkill.skillModList:NewMod("Damage", "MORE", moreDamage, "Mirage Archer", env.player.mainSkill.ModFlags, env.player.mainSkill.KeywordFlags)
				newSkill.skillModList:NewMod("Speed", "MORE", moreAttackSpeed, "Mirage Archer", env.player.mainSkill.ModFlags, env.player.mainSkill.KeywordFlags)

				env.player.mainSkill.mirage = { }
				env.player.mainSkill.mirage.count = mirageCount
				env.player.mainSkill.mirage.name = usedSkill.activeEffect.grantedEffect.name

				if usedSkill.skillPartName {
					env.player.mainSkill.mirage.skillPart = usedSkill.skillPart
					env.player.mainSkill.mirage.skillPartName = usedSkill.skillPartName
					env.player.mainSkill.mirage.infoMessage2 = usedSkill.activeEffect.grantedEffect.name
				} else {
					env.player.mainSkill.mirage.skillPartName = nil
				}
				env.player.mainSkill.mirage.infoTrigger = "MA"

				// Recalculate the offensive/defensive aspects of the Mirage Archer influence on skill
				newEnv.player.mainSkill = newSkill
				// mark it so we don't recurse infinitely
				newSkill.marked = true
				newEnv.dontCache = true
				calcs.perform(newEnv)

				env.player.mainSkill.infoMessage = tostring(mirageCount) + " Mirage Archers using " + usedSkill.activeEffect.grantedEffect.name

				// Re-link over the output
				env.player.mainSkill.mirage.output = newEnv.player.output

				if newSkill.minion {
					env.player.mainSkill.mirage.minion = {}
					env.player.mainSkill.mirage.minion.output = newEnv.minion.output
				}

				// Make any necessary corrections to output
				env.player.mainSkill.mirage.output.ManaCost = 0

				if newEnv.player.breakdown {
					env.player.mainSkill.mirage.breakdown = newEnv.player.breakdown
					// Make any necessary corrections to breakdown
					env.player.mainSkill.mirage.breakdown.ManaCost = nil
					if newSkill.minion {
						env.player.mainSkill.mirage.minion.breakdown = newEnv.minion.breakdown
					}
				}
			} else {
				env.player.mainSkill.infoMessage2 = "No Mirage Archer active skill found"
			}
		}

		// Kitava's Thirst
		if env.player.mainSkill.skillData.triggeredByManaSpent and not env.player.mainSkill.skillFlags.minion {
			triggerName := "Kitava"
			spellCount := 0
			icdr := calcLib.mod(env.player.mainSkill.skillModList, env.player.mainSkill.skillCfg, "CooldownRecovery")
			reqManaCost := env.player.modDB.Sum(mod.TypeBase, nil, "KitavaRequiredManaCost")
			trigRate := 0
			source := nil
			for _, skill in ipairs(env.player.activeSkillList) {
				if not skill.skillTypes[SkillType.Triggered] and skill ~= env.player.mainSkill and not skill.skillData.triggeredByManaSpent {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate, reqManaCost)
				}
				if skill.skillData.triggeredByManaSpent and env.player.mainSkill.socketGroup.slot == skill.socketGroup.slot {
					spellCount = spellCount + 1
				}
			}

			if not source or spellCount < 1 {
				env.player.mainSkill.skillData.triggeredByManaSpent = nil
				env.player.mainSkill.infoMessage = s_format("No %s Triggering Skill Found", triggerName)
				env.player.mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.player.mainSkill.infoTrigger = ""
			} else {
				env.player.mainSkill.skillData.triggered = true

				actor.Output["ActionTriggerRate"] = getTriggerActionTriggerRate(env.player.mainSkill.skillData.cooldown, env, breakdown)

				// Get action trigger rate
				kitavaCD := getTriggerDefaultCooldown(env.player.mainSkill.supportList, "SupportCastOnManaSpent")

				trigRate = icdr / kitavaCD
				actor.Output["SourceTriggerRate"] = trigRate
				actor.Output["ServerTriggerRate"] = min(actor.Output["SourceTriggerRate"], actor.Output["ActionTriggerRate"])
				if breakdown {
					modActionCooldown := kitavaCD / icdr
					rateCapAdjusted := m_ceil(modActionCooldown * data.misc.ServerTickRate) / data.misc.ServerTickRate
					extraICDRNeeded := m_ceil((modActionCooldown - rateCapAdjusted + data.misc.ServerTickTime) * icdr * 1000)
					breakdown.SimData = {
						s_format("%.2f ^8(base cooldown of kitava's trigger)", kitavaCD),
						s_format("/ %.2f ^8(increased/reduced cooldown recovery)", icdr),
						s_format("= %.4f ^8(final cooldown of trigger)", modActionCooldown),
						s_format(""),
						s_format("%.3f ^8(adjusted for server tick rate)", rateCapAdjusted),
						s_format("^8(extra ICDR of %d%% would reach next breakpoint)", extraICDRNeeded),
						s_format(""),
						s_format("Trigger rate:"),
						s_format("1 / %.3f", rateCapAdjusted),
						s_format("= %.2f ^8per second", 1 / rateCapAdjusted),
					}
					breakdown.ServerTriggerRate = {
						s_format("%.2f ^8(smaller of 'cap' and 'skill' trigger rates)", actor.Output["ServerTriggerRate"]),
					}
				}

				// Account for chance to trigger
				kitavaTriggerChance := env.player.modDB.Sum(mod.TypeBase, nil, "KitavaTriggerChance")
				trigRate = actor.Output["ServerTriggerRate"] * kitavaTriggerChance / 100
				if breakdown {
					breakdown.Speed = {
						s_format("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						s_format("x %.2f%% ^8(kitava's trigger chance)", kitavaTriggerChance),
						s_format("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.player.mainSkill, env.player.mainSkill)
				env.player.mainSkill.skillData.triggerRate = trigRate
				env.player.mainSkill.skillData.triggerSource = source
				env.player.mainSkill.infoMessage = "Kitava's Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.player.mainSkill.infoTrigger = triggerName
			}
		}

		// Crafted Trigger
		if env.player.mainSkill.skillData.triggeredByCraft and not env.player.mainSkill.skillFlags.minion {
			triggerName := "Crafted"
			spellCount := 0
			icdr := calcLib.mod(env.player.mainSkill.skillModList, env.player.mainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill in ipairs(env.player.activeSkillList) {
				if (skill.skillTypes[SkillType.Damage] or skill.skillTypes[SkillType.Attack] or skill.skillTypes[SkillType.Spell]) and skill ~= env.player.mainSkill and not skill.skillData.triggeredByCraft {
					source, trigRate = skill, 0
				}
				if skill.skillData.triggeredByCraft and env.player.mainSkill.socketGroup.slot == skill.socketGroup.slot {
					spellCount = spellCount + 1
				}
				// we just need one source and one linked spell
				if source and spellCount > 0 {
					break
				}
			}
			if not source or spellCount < 1 {
				env.player.mainSkill.skillData.triggeredByCraft = nil
				env.player.mainSkill.infoMessage = s_format("No %s Triggering Skill Found", triggerName)
				env.player.mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.player.mainSkill.infoTrigger = ""
			} else {
				env.player.mainSkill.skillData.triggered = true

				actor.Output["ActionTriggerRate"] = getTriggerActionTriggerRate(env.player.mainSkill.skillData.cooldown, env, breakdown)

				// Get action trigger rate
				craftedCD := getTriggerDefaultCooldown(env.player.mainSkill.supportList, "SupportTriggerSpellOnSkillUse")

				trigRate = icdr / craftedCD
				actor.Output["SourceTriggerRate"] = trigRate
				actor.Output["ServerTriggerRate"] = min(actor.Output["SourceTriggerRate"], actor.Output["ActionTriggerRate"])
				if breakdown {
					modActionCooldown := craftedCD / icdr
					rateCapAdjusted := m_ceil(modActionCooldown * data.misc.ServerTickRate) / data.misc.ServerTickRate
					extraICDRNeeded := m_ceil((modActionCooldown - rateCapAdjusted + data.misc.ServerTickTime) * icdr * 1000)
					breakdown.SimData = {
						s_format("%.2f ^8(base cooldown of crafted trigger)", craftedCD),
						s_format("/ %.2f ^8(increased/reduced cooldown recovery)", icdr),
						s_format("= %.4f ^8(final cooldown of trigger)", modActionCooldown),
						s_format(""),
						s_format("%.3f ^8(adjusted for server tick rate)", rateCapAdjusted),
						s_format("^8(extra ICDR of %d%% would reach next breakpoint)", extraICDRNeeded),
						s_format(""),
						s_format("Trigger rate:"),
						s_format("1 / %.3f", rateCapAdjusted),
						s_format("= %.2f ^8per second", 1 / rateCapAdjusted),
					}
					breakdown.ServerTriggerRate = {
						s_format("%.2f ^8(smaller of 'cap' and 'skill' trigger rates)", actor.Output["ServerTriggerRate"]),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.player.mainSkill, env.player.mainSkill)
				env.player.mainSkill.skillData.triggerRate = actor.Output["ServerTriggerRate"]
				env.player.mainSkill.skillData.triggerSource = source
				env.player.mainSkill.infoMessage = "Weapon-Crafted Triggering Skill Found"
				env.player.mainSkill.infoTrigger = triggerName
				env.player.mainSkill.skillFlags.dontDisplay = true
			}
		}

		// Helmet Focus Trigger
		if env.player.mainSkill.skillData.triggeredByFocus and not env.player.mainSkill.skillFlags.minion {
			triggerName := "Focus"
			spellCount := 0
			icdr := calcLib.mod(env.player.mainSkill.skillModList, env.player.mainSkill.skillCfg, "FocusCooldownRecovery")
			trigRate := 0
			source := env.player.modDB.Flag(nil, "Condition:Focused")
			for _, skill in ipairs(env.player.activeSkillList) {
				if skill.skillData.triggeredByFocus and env.player.mainSkill.socketGroup.slot == skill.socketGroup.slot {
					spellCount = spellCount + 1
				}
			}
			if not source or spellCount < 1 {
				env.player.mainSkill.skillData.triggeredByFocus = nil
				env.player.mainSkill.infoMessage = s_format("No %s Triggering Skill Found", triggerName)
				env.player.mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.player.mainSkill.infoTrigger = ""
			} else {
				env.player.mainSkill.skillData.triggered = true

				actor.Output["ActionTriggerRate"] = getTriggerActionTriggerRate(env.player.mainSkill.skillData.cooldown, env, breakdown, true)

				// Get action trigger rate
				skillFocus := env.data.skills["Focus"]
				focusCD := skillFocus.levels[1].cooldown

				trigRate = icdr / focusCD
				actor.Output["SourceTriggerRate"] = trigRate
				actor.Output["ServerTriggerRate"] = min(actor.Output["SourceTriggerRate"], actor.Output["ActionTriggerRate"])
				if breakdown {
					modActionCooldown := focusCD / icdr
					rateCapAdjusted := m_ceil(modActionCooldown * data.misc.ServerTickRate) / data.misc.ServerTickRate
					breakdown.SimData = {
						s_format("%.2f ^8(base cooldown of focus trigger)", focusCD),
						s_format("/ %.2f ^8(increased/reduced cooldown recovery)", icdr),
						s_format("= %.4f ^8(final cooldown of trigger)", modActionCooldown),
						s_format(""),
						s_format("%.3f ^8(adjusted for server tick rate)", rateCapAdjusted),
						s_format(""),
						s_format("Trigger rate:"),
						s_format("1 / %.3f", rateCapAdjusted),
						s_format("= %.2f ^8per second", 1 / rateCapAdjusted),
					}
					breakdown.ServerTriggerRate = {
						s_format("%.2f ^8(smaller of 'cap' and 'skill' trigger rates)", actor.Output["ServerTriggerRate"]),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.player.mainSkill, env.player.mainSkill)
				env.player.mainSkill.skillData.triggerRate = actor.Output["ServerTriggerRate"]
				env.player.mainSkill.skillData.triggerSource = source
				env.player.mainSkill.infoMessage = "Focus Triggering Skill Found"
				env.player.mainSkill.infoTrigger = triggerName
				env.player.mainSkill.skillFlags.dontDisplay = true
			}
		}

		// Unique Item Trigger
		if env.player.mainSkill.skillData.triggeredByUnique and not env.player.mainSkill.skillFlags.minion {
			uniqueTriggerName := getUniqueItemTriggerName(env.player.mainSkill)
			triggerName := ""
			spellCount := {}
			icdr := calcLib.mod(env.player.mainSkill.skillModList, env.player.mainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill in ipairs(env.player.activeSkillList) {
				cooldownOverride := skill.skillModList:Override(env.player.mainSkill.skillCfg, "CooldownRecovery")
				if uniqueTriggerName == "Poet's Pen" {
					triggerName = "Poet"
					if (skill.skillTypes[SkillType.Damage] or skill.skillTypes[SkillType.Attack]) and band(skill.skillCfg.flags, ModFlag.Wand) > 0 and skill ~= env.player.mainSkill and not skill.skillData.triggeredByUnique {
						source, trigRate = findTriggerSkill(env, skill, source, trigRate)
					}
					if skill.skillData.triggeredByUnique and env.player.mainSkill.socketGroup.slot == skill.socketGroup.slot and skill.skillTypes[SkillType.Spell] {
						t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
					}
				} else if uniqueTriggerName == "Maloney's Mechanism" {
					triggerName = "Maloney"
					if skill.skillTypes[SkillType.Attack] and band(skill.skillCfg.flags, ModFlag.Bow) > 0 and skill ~= env.player.mainSkill and not skill.skillData.triggeredByUnique {
						source, trigRate = findTriggerSkill(env, skill, source, trigRate)
					}
					if skill.skillData.triggeredByUnique and env.player.mainSkill.socketGroup.slot == skill.socketGroup.slot and skill.skillTypes[SkillType.RangedAttack] {
						t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
					}
				} else if uniqueTriggerName == "Asenath's Chant" {
					triggerName = "Asenath"
					if (skill.skillTypes[SkillType.Damage] or skill.skillTypes[SkillType.Attack]) and band(skill.skillCfg.flags, ModFlag.Bow) > 0 and skill ~= env.player.mainSkill and not skill.skillData.triggeredByUnique {
						source, trigRate = findTriggerSkill(env, skill, source, trigRate)
					}
					if skill.skillData.triggeredByUnique and env.player.mainSkill.socketGroup.slot == skill.socketGroup.slot and skill.skillTypes[SkillType.Spell] {
						t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
					}
				} else if uniqueTriggerName == "Queen's Demand" {
					triggerName = "QD"
					if skill.activeEffect.grantedEffect.name == uniqueTriggerName {
						source, trigRate = findTriggerSkill(env, skill, source, trigRate)
					}
					if skill.skillData.triggeredByUnique and env.player.mainSkill.socketGroup.slot == skill.socketGroup.slot {
						t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
					}
				} else {
					ConPrintf("[ERROR]: Unhandled Unique Trigger Name: " + uniqueTriggerName)
				}
			}
			if not source or #spellCount < 1 {
				env.player.mainSkill.skillData.triggeredByUnique = nil
				env.player.mainSkill.infoMessage = s_format("No %s Triggering Skill Found", triggerName)
				env.player.mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.player.mainSkill.infoTrigger = ""
			} else {
				env.player.mainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed
				dualWield := false

				sourceAPS, dualWield = calcDualWieldImpact(env, sourceAPS, source.skillData.doubleHitsWhenDualWielding)

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown, dualWield)

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.player.mainSkill, env.player.mainSkill)

				env.player.mainSkill.skillData.triggerRate = trigRate
				env.player.mainSkill.skillData.triggerSource = source
				env.player.mainSkill.skillData.triggerSourceUUID = cacheSkillUUID(source, env.mode)
				env.player.mainSkill.skillData.triggerUnleash = source.skillModList:Flag(nil, "HasSeals") and source.skillTypes[SkillType.CanRapidFire]
				env.player.mainSkill.infoMessage = env.player.mainSkill.activeEffect.grantedEffect.name + "'s Trigger: " + source.activeEffect.grantedEffect.name
				env.player.mainSkill.infoTrigger = env.player.mainSkill.infoTrigger or triggerName
			}
		}

		// Cast On Critical Strike Support (CoC)
		if env.player.mainSkill.skillData.triggeredByCoC and not env.player.mainSkill.skillFlags.minion {
			spellCount := {}
			icdr := calcLib.mod(env.player.mainSkill.skillModList, env.player.mainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill in ipairs(env.player.activeSkillList) {
				match1 := env.player.mainSkill.activeEffect.grantedEffect.fromItem and skill.socketGroup.slot == env.player.mainSkill.socketGroup.slot
				match2 := (not env.player.mainSkill.activeEffect.grantedEffect.fromItem) and skill.socketGroup == env.player.mainSkill.socketGroup
				if skill.skillTypes[SkillType.Attack] and skill ~= env.player.mainSkill and (match1 or match2) {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredByCoC and (match1 or match2) {
					cooldownOverride := skill.skillModList:Override(env.player.mainSkill.skillCfg, "CooldownRecovery")
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.player.mainSkill.skillData.triggeredByCoC = nil
				env.player.mainSkill.infoMessage = "No CoC Triggering Skill Found"
				env.player.mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.player.mainSkill.infoTrigger = ""
			} else {
				env.player.mainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown)

				// Account for chance to hit/crit
				sourceCritChance := GlobalCache.cachedData["CACHE"][uuid].CritChance
				trigRate = trigRate * sourceCritChance / 100
				trigRate = trigRate * (source.skillData.chanceToTriggerOnCrit or 100) / 100
				if breakdown {
					breakdown.Speed = {
						s_format("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						s_format("x %.2f%% ^8(%s crit chance)", sourceCritChance, source.activeEffect.grantedEffect.name),
						s_format("x %.2f%% ^8(chance to trigger on crit)", source.skillData.chanceToTriggerOnCrit or 100),
						s_format("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.player.mainSkill, env.player.mainSkill)
				env.player.mainSkill.skillData.triggerRate = trigRate
				env.player.mainSkill.skillData.triggerSource = source
				env.player.mainSkill.infoMessage = "CoC Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.player.mainSkill.infoTrigger = "CoC"
			}
		}

		// Cast On Melee Kill Support (CoMK)
		if env.player.mainSkill.skillData.triggeredByMeleeKill and not env.player.mainSkill.skillFlags.minion and modDB.Flag(nil, "Condition:KilledRecently") {
			spellCount := {}
			icdr := calcLib.mod(env.player.mainSkill.skillModList, env.player.mainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill in ipairs(env.player.activeSkillList) {
				match1 := env.player.mainSkill.activeEffect.grantedEffect.fromItem and skill.socketGroup.slot == env.player.mainSkill.socketGroup.slot
				match2 := (not env.player.mainSkill.activeEffect.grantedEffect.fromItem) and skill.socketGroup == env.player.mainSkill.socketGroup
				if skill.skillTypes[SkillType.Attack] and skill.skillTypes[SkillType.Melee] and skill ~= env.player.mainSkill and (match1 or match2) {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredByMeleeKill and (match1 or match2) {
					cooldownOverride := skill.skillModList:Override(env.player.mainSkill.skillCfg, "CooldownRecovery")
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.player.mainSkill.skillData.triggeredByMeleeKill = nil
				env.player.mainSkill.infoMessage = "No CoMK Triggering Skill Found"
				env.player.mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.player.mainSkill.infoTrigger = ""
			} else {
				env.player.mainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown)

				// Account for chance to trigger on Melee Kill
				trigRate = trigRate * source.skillData.chanceToTriggerOnMeleeKill / 100

				if breakdown {
					breakdown.Speed = {
						s_format("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						s_format("x %.2f%% ^8(chance to trigger on melee kill)", source.skillData.chanceToTriggerOnMeleeKill),
						s_format("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.player.mainSkill, env.player.mainSkill)
				env.player.mainSkill.skillData.triggerRate = trigRate
				env.player.mainSkill.skillData.triggerSource = source
				env.player.mainSkill.infoMessage = "CoMK Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.player.mainSkill.infoTrigger = "CoMK"
			}
		}

		// Cast While Channelling
		if env.player.mainSkill.skillData.triggeredWhileChannelling and not env.player.mainSkill.skillFlags.minion {
			spellCount := {}
			trigRate := 0
			source := nil
			for _, skill in ipairs(env.player.activeSkillList) {
				match1 := env.player.mainSkill.activeEffect.grantedEffect.fromItem and skill.socketGroup.slot == env.player.mainSkill.socketGroup.slot
				match2 := (not env.player.mainSkill.activeEffect.grantedEffect.fromItem) and skill.socketGroup == env.player.mainSkill.socketGroup
				if skill.skillTypes[SkillType.Channel] and skill ~= env.player.mainSkill and (match1 or match2) {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredWhileChannelling and (match1 or match2) {
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = skill.skillData.cooldown, next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.player.mainSkill.skillData.triggeredWhileChannelling = nil
				env.player.mainSkill.infoMessage = "No CwC Triggering Skill Found"
				env.player.mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.player.mainSkill.infoTrigger = ""
			} else {
				env.player.mainSkill.skillData.triggered = true

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, nil, spellCount, output, breakdown)

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.player.mainSkill, env.player.mainSkill)
				env.player.mainSkill.skillData.triggerRate = trigRate
				env.player.mainSkill.skillData.triggerSource = source
				env.player.mainSkill.infoMessage = "CwC Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.player.mainSkill.infoTrigger = "CwC"

				env.player.mainSkill.skillFlags.dontDisplay = true
			}
		}

		// Triggered by parent attack
		if env.minion and env.player.mainSkill.minion {
			if env.minion.mainSkill.skillData.triggeredByParentAttack {
				spellCount := {}
				trigRate := 0
				source := nil
				for _, skill in ipairs(env.player.activeSkillList) {
					if skill.skillTypes[SkillType.Attack] and skill ~= env.player.mainSkill {
						source, trigRate = findTriggerSkill(env, skill, source, trigRate)
					}
				}

				icdr := calcLib.mod(env.minion.mainSkill.skillModList, env.minion.mainSkill.skillCfg, "CooldownRecovery")
				t_insert(spellCount, { uuid = cacheSkillUUID(env.minion.mainSkill), cd = env.minion.mainSkill.skillData.cooldown / icdr, next_trig = 0, count = 0 })

				if not source {
					env.minion.mainSkill.skillData.triggeredByParentAttack = nil
					env.minion.mainSkill.infoMessage = "No triggering Skill Found"
					env.minion.mainSkill.infoMessage2 = "DPS reported assuming regular cast"
					env.minion.mainSkill.infoTrigger = ""
				} else {
					env.minion.mainSkill.skillData.triggered = true
					uuid := cacheSkillUUID(source)

					sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed

					// Get action trigger rate
					trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, env.minion.output, env.minion.breakdown, false, true)

					// Account for chance to hit
					sourceHitChance := GlobalCache.cachedData["CACHE"][uuid].HitChance
					trigRate = trigRate * sourceHitChance / 100
					if env.minion.breakdown {
						env.minion.breakdown.Speed = {
							s_format("%.2fs ^8(adjusted trigger rate)", env.minion.output.ServerTriggerRate),
							s_format("x %.2f%% ^8(%s Hit chance)", sourceHitChance, source.activeEffect.grantedEffect.name),
							s_format("= %.2f ^8per second", trigRate),
						}
					}

					// Account for Trigger-related INC/MORE modifiers
					addTriggerIncMoreMods(env.minion.mainSkill, env.minion.mainSkill)
					env.minion.mainSkill.skillData.triggerRate = trigRate
					env.minion.mainSkill.skillData.triggerSource = source
					env.minion.mainSkill.infoMessage = "Triggering Skill: " + source.activeEffect.grantedEffect.name
					env.minion.mainSkill.infoTrigger = "Parent attack"
				}
			}
		}
	*/

	/*
		TODO // Fix the configured impale stacks on the enemy
		// 		If the config is missing (blank), then use the maximum number of stacks
		//		If the config is larger than the maximum number of stacks, replace it with the correct maximum
		maxImpaleStacks := modDB.Sum(mod.TypeBase, nil, "ImpaleStacksMax")
		if not enemyDB:HasMod("BASE", nil, "Multiplier:ImpaleStacks") {
			enemyDB:NewMod("Multiplier:ImpaleStacks", "BASE", maxImpaleStacks, "Config", { type = "Condition", var = "Combat" })
		} else if enemyDB:Sum(mod.TypeBase, nil, "Multiplier:ImpaleStacks") > maxImpaleStacks {
			enemyDB:ReplaceMod("Multiplier:ImpaleStacks", "BASE", maxImpaleStacks, "Config", { type = "Condition", var = "Combat" })
		}
	*/

	/*
		TODO // Calculate maximum and apply the strongest non-damaging ailments
		ailmentData := data.nonDamagingAilment
		ailments := {
			["Chill"] = { condition = "Chilled", mods = function(num)
				mods := {
					modLib.createMod("ActionSpeed", "INC", -num, "Chill", { type = "Condition", var = "Chilled" })
				}
				if actor.Output["BonechillEffect"] {
					t_insert(mods, modLib.createMod("ColdDamageTaken", "INC", actor.Output["BonechillEffect"], "Bonechill", { type = "Limit", limit = actor.Output["MaximumChill"] }, { type = "Condition", var = "Chilled" }))
				}
				return mods
			end },
			["Shock"] = { condition = "Shocked", mods = function(num) return {
				modLib.createMod("DamageTaken", "INC", num, "Shock", { type = "Condition", var = "Shocked" })
			} end },
			["Scorch"] = { condition = "Scorched", mods = function(num) return {
				modLib.createMod("ElementalResist", "BASE", -num, "Scorch", { type = "Condition", var = "Scorched" })
			} end },
			["Brittle"] = { condition = "Brittle", mods = function(num) return {
				modLib.createMod("SelfCritChance", "BASE", num, "Brittle", { type = "Condition", var = "Brittle" })
			} end },
			["Sap"] = { condition = "Sapped", mods = function(num) return {
				modLib.createMod("Damage", "MORE", -num, "Sap", { type = "Condition", var = "Sapped" })
			} end },
		}

		for ailment, val in pairs(ailments) {
			if (enemyDB:Sum(mod.TypeBase, nil, ailment+"Val") > 0
			or modDB.Sum(mod.TypeBase, nil, ailment+"Base", ailment+"Override")
			or (ailment == "Chill" and actor.Output["BonechillEffect"]))
			and not enemyDB:Flag(nil, "Condition:Already"+val.condition) {
				override := 0
				for _, value in ipairs(modDB.Tabulate("BASE", nil, ailment+"Base", ailment+"Override")) {
					mod := value.mod
					effect := mod.value
					if mod.name == ailment+"Override" {
						enemyDB:NewMod("Condition:"+val.condition, "FLAG", true, mod.source)
					}
					if mod.name == ailment+"Base" {
						effect = effect * calcLib.mod(modDB, nil, "Enemy"+ailment+"Effect")
						modDB.NewMod(ailment+"Override", "BASE", effect, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
					}
					override = max(override, effect or 0)
				}
				actor.Output["Maximum"+ailment] = modDB.Override(nil, ailment+"Max") or ailmentData[ailment].max
				actor.Output["Current"+ailment] = math.Floor(min(max(override, enemyDB:Sum(mod.TypeBase, nil, ailment+"Val"), ailment == "Chill" and actor.Output["BonechillEffect"] or 0), actor.Output["Maximum"+ailment]) * (10 ^ ailmentData[ailment].precision)) / (10 ^ ailmentData[ailment].precision)
				for _, mod in ipairs(val.mods(actor.Output["Current"+ailment])) {
					enemyDB:AddMod(mod)
				}
				enemyDB:NewMod("Condition:Already"+val.condition, "FLAG", true, { type = "Condition", var = val.condition } ) // Prevents ailment from applying doubly for minions
			}
		}
	*/

	/*
		TODO // Check for extra auras
		for _, value in ipairs(modDB.List(nil, "ExtraAura")) {
			modList := { value.mod }
			if not value.onlyAllies {
				inc := modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				more := modDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				modDB.ScaleAddList(modList, (1 + inc / 100) * more)
				if not value.notBuff {
					modDB.multipliers["BuffOnSelf"] = (modDB.multipliers["BuffOnSelf"] or 0) + 1
				}
			}
			if env.minion and not modDB.Flag(nil, "SelfAurasCannotAffectAllies") {
				inc := env.minion.modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				more := env.minion.modDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				env.minion.modDB.ScaleAddList(modList, (1 + inc / 100) * more)
			}
		}
	*/

	/*
		TODO // Check for modifiers to apply to actors affected by player auras or curses
		for _, value in ipairs(modDB.List(nil, "AffectedByAuraMod")) {
			for actor in pairs(affectedByAura) {
				actor.modDB.AddMod(value.mod)
			}
		}
		for _, value in ipairs(modDB.List(nil, "AffectedByCurseMod")) {
			for actor in pairs(affectedByCurse) {
				actor.modDB.AddMod(value.mod)
			}
		}
	*/

	// Merge keystones again to catch any that were added by buffs
	mergeKeystones(env)

	/*
		TODO // Special handling for Dancing Dervish
		if modDB.Flag(nil, "DisableWeapons") {
			env.player.weaponData1 = copyTable(env.data.unarmedWeaponData[env.classId])
			modDB.conditions["Unarmed"] = true
			if not env.player.Gloves or env.player.Gloves == None {
				modDB.conditions["Unencumbered"] = true
			}
		} else if env.weaponModList1 {
			modDB.AddList(env.weaponModList1)
		}
	*/

	// Process misc buffs/modifiers
	DoActorMisc(env, env.Player)
	if env.Minion != nil {
		// TODO doActorMisc(env, env.minion)
	}
	DoActorMisc(env, env.Enemy)

	// Totems
	for _, activeSkill := range env.Player.ActiveSkillList {
		if activeSkill.SkillFlags[SkillFlagTotem] {
			limit := env.Player.MainSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
			env.Player.Output["ActiveTotemLimit"] = max(limit, env.Player.Output["ActiveTotemLimit"])
			TotemsSummoned := env.ModDB.Override(nil, "TotemsSummoned")
			if TotemsSummoned != nil {
				env.Player.Output["TotemsSummoned"] = TotemsSummoned.Float()
			} else {
				env.Player.Output["TotemsSummoned"] = 0
			}
			env.EnemyModDB.Multipliers["TotemsSummoned"] = max(env.Player.Output["TotemsSummoned"], env.EnemyModDB.Multipliers["TotemsSummoned"])
		}
	}

	/*
		TODO // Apply exposures
		major, minor := env.spec.treeVersion:match("(%d+)_(%d+)")
		for _, element in ipairs({"Fire", "Cold", "Lightning"}) {
			if tonumber(major) <= 3 and tonumber(minor) <= 15 // Elemental Equilibrium pre-3.16 does not remove Exposure effects
				or not modDB.Flag(nil, "ElementalEquilibrium") // if Elemental Equilibrium isn't active we just process Exposure normally
				or element == "Fire" and not enemyDB:Flag(nil, "Condition:HitByFireDamage")
				or element == "Cold" and not enemyDB:Flag(nil, "Condition:HitByColdDamage")
				or element == "Lightning" and not enemyDB:Flag(nil, "Condition:HitByLightningDamage") {
				min := math.huge
				source := ""
				for _, mod in ipairs(enemyDB:Tabulate("BASE", nil, element+"Exposure")) {
					if mod.value < min {
						min = mod.value
						source = mod.mod.source
					}
				}
				if min ~= math.huge {
					// Modify the magnitude of all exposures
					for _, mod in ipairs(modDB.Tabulate("BASE", nil, "ExtraExposure", "Extra"+element+"Exposure")) {
						min = min + mod.value
					}
					enemyDB:NewMod(element+"Resist", "BASE", min(min, modDB.Override(nil, "ExposureMin")), source)
					modDB.NewMod("Condition:AppliedExposureRecently", "FLAG", true, "")
				}
			}
		}
	*/

	/*
		TODO // Handle consecrated ground effects on enemies
		if enemyDB:Flag(nil, "Condition:OnConsecratedGround") {
			effect := 1 + modDB.Sum(mod.TypeIncrease, nil, "ConsecratedGroundEffect") / 100
			enemyDB:NewMod("DamageTaken", "INC", enemyDB:Sum(mod.TypeIncrease, nil, "DamageTakenConsecratedGround") * effect, "Consecrated Ground")
		}
	*/

	// Defence/offence calculations
	CalculateDefence(env, env.Player)
	CalculateOffence(env, env.Player, env.Player.MainSkill)

	/*
		TODO Minion Defence/offence calculations
		if env.minion {
			calcs.defence(env, env.minion)
			calcs.offence(env, env.minion, env.minion.mainSkill)
		}
	*/

	/*
		TODO Cache Data
		uuid := cacheSkillUUID(env.player.mainSkill)
		if not env.dontCache {
			cacheData(uuid, env)
		}
	*/
}

// Calculate life/mana reservation
func doActorLifeManaReservation(actor *Actor) {
	for _, pool := range []string{"Life", "Mana"} {
		Max := actor.Output[pool]
		reserved := float64(0)
		if Max > 0 {
			reserved = actor.Output["reserved_"+pool+"Base"] + math.Ceil(actor.Output["reserved_"+pool+"Percent"]/100)
			actor.Output[pool+"Reserved"] = min(reserved, Max)
			actor.Output[pool+"ReservedPercent"] = min(reserved/Max*100, 100)
			actor.Output[pool+"Unreserved"] = Max - reserved
			actor.Output[pool+"UnreservedPercent"] = (Max - reserved) / Max * 100
			if (Max-reserved)/Max <= data.LowPoolThreshold {
				actor.ModDB.Conditions["Low"+pool] = true
			}
		}
		for _, value := range actor.ModDB.List(nil, "GrantReserved"+pool+"AsAura") {
			auraMod := value.(*mod.ModValueMulti).List().(mod.GrantReservedPoolAsAura).Mod.Clone()
			auraMod.Value().SetFloat(math.Floor(auraMod.Value().Float() * min(reserved, Max)))
			actor.ModDB.AddMod(mod.NewList("ExtraAura", mod.ExtraAura{
				Mod: auraMod,
			}))
		}
	}
}

func doActorAttribsPoolsConditions(env *Environment, actor *Actor) {
	/*
		local modDB = actor.modDB
		local output = actor.output
		local breakdown = actor.breakdown
		local condList = modDB.conditions
	*/
	/*
		TODO // Set conditions
		if (actor.itemList["Weapon 2"] and actor.itemList["Weapon 2"].type == "Shield") or (actor == env.player and env.aegisModList) {
			condList["UsingShield"] = true
		}
		if not actor.itemList["Weapon 2"] {
			condList["OffHandIsEmpty"] = true
		}
		if actor.weaponData1.type == "None" {
			condList["Unarmed"] = true
			if not actor.itemList["Weapon 2"] and not actor.itemList["Gloves"] {
				condList["Unencumbered"] = true
			}
		} else {
			info := env.data.weaponTypeInfo[actor.weaponData1.type]
			condList["Using"+info.flag] = true
			if actor.weaponData1.countsAsAll1H {
				condList["UsingAxe"] = true
				condList["UsingSword"] = true
				condList["UsingDagger"] = true
				condList["UsingMace"] = true
				condList["UsingClaw"] = true
				// GGG stated that a single Varunastra satisfied requirement for wielding two different weapons
				condList["WieldingDifferentWeaponTypes"] = true
			}
			if info.melee {
				condList["UsingMeleeWeapon"] = true
			}
			if info.oneHand {
				condList["UsingOneHandedWeapon"] = true
			} else {
				condList["UsingTwoHandedWeapon"] = true
			}
		}
		if actor.weaponData2.type {
			info := env.data.weaponTypeInfo[actor.weaponData2.type]
			condList["Using"+info.flag] = true
			if actor.weaponData2.countsAsAll1H {
				condList["UsingAxe"] = true
				condList["UsingSword"] = true
				condList["UsingDagger"] = true
				condList["UsingMace"] = true
				condList["UsingClaw"] = true
				// GGG stated that a single Varunastra satisfied requirement for wielding two different weapons
				condList["WieldingDifferentWeaponTypes"] = true
			}
			if info.melee {
				condList["UsingMeleeWeapon"] = true
			}
			if info.oneHand {
				condList["UsingOneHandedWeapon"] = true
			} else {
				condList["UsingTwoHandedWeapon"] = true
			}
		}
		if actor.weaponData1.type and actor.weaponData2.type {
			condList["DualWielding"] = true
			if (actor.weaponData1.type == "Claw" or actor.weaponData1.countsAsAll1H) and (actor.weaponData2.type == "Claw" or actor.weaponData2.countsAsAll1H) {
				condList["DualWieldingClaws"] = true
			}
			if (actor.weaponData1.type == "Dagger" or actor.weaponData1.countsAsAll1H) and (actor.weaponData2.type == "Dagger" or actor.weaponData2.countsAsAll1H) {
				condList["DualWieldingDaggers"] = true
			}
			if (env.data.weaponTypeInfo[actor.weaponData1.type].label or actor.weaponData1.type) ~= (env.data.weaponTypeInfo[actor.weaponData2.type].label or actor.weaponData2.type) {
				info1 := env.data.weaponTypeInfo[actor.weaponData1.type]
				info2 := env.data.weaponTypeInfo[actor.weaponData2.type]
				if info1.oneHand and info2.oneHand {
					condList["WieldingDifferentWeaponTypes"] = true
				}
			}
		}
		if env.mode_combat {
			if not modDB.Flag(nil, "NeverCrit") {
				condList["CritInPast8Sec"] = true
			}
			if not actor.mainSkill.skillData.triggered and not actor.mainSkill.skillFlags.trap and not actor.mainSkill.skillFlags.mine and not actor.mainSkill.skillFlags.totem {
				if actor.mainSkill.skillFlags.attack {
					condList["AttackedRecently"] = true
				} else if actor.mainSkill.skillFlags.spell {
					condList["CastSpellRecently"] = true
				}
				if actor.mainSkill.skillTypes[SkillType.Movement] {
					condList["UsedMovementSkillRecently"] = true
				}
				if actor.mainSkill.skillFlags.minion {
					condList["UsedMinionSkillRecently"] = true
				}
				if actor.mainSkill.skillTypes[SkillType.Vaal] {
					condList["UsedVaalSkillRecently"] = true
				}
				if actor.mainSkill.skillTypes[SkillType.Channel] {
					condList["Channelling"] = true
				}
			}
			if actor.mainSkill.skillFlags.hit and not actor.mainSkill.skillFlags.trap and not actor.mainSkill.skillFlags.mine and not actor.mainSkill.skillFlags.totem {
				condList["HitRecently"] = true
			}
			if actor.mainSkill.skillFlags.totem {
				condList["HaveTotem"] = true
				condList["SummonedTotemRecently"] = true
			}
			if actor.mainSkill.skillFlags.mine {
				condList["DetonatedMinesRecently"] = true
			}
			if modDB.Sum(mod.TypeBase, nil, "EnemyScorchChance") > 0 or modDB.Flag(nil, "CritAlwaysAltAilments") and not modDB.Flag(nil, "NeverCrit") {
				condList["CanInflictScorch"] = true
			}
			if modDB.Sum(mod.TypeBase, nil, "EnemyBrittleChance") > 0 or modDB.Flag(nil, "CritAlwaysAltAilments") and not modDB.Flag(nil, "NeverCrit") {
				condList["CanInflictBrittle"] = true
			}
			if modDB.Sum(mod.TypeBase, nil, "EnemySapChance") > 0 or modDB.Flag(nil, "CritAlwaysAltAilments") and not modDB.Flag(nil, "NeverCrit") {
				condList["CanInflictSap"] = true
			}
		}
		if env.mode_effective {
			if env.player.mainSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "FireExposureChance") > 0 or modDB.Sum(mod.TypeBase, nil, "FireExposureChance") > 0 {
				condList["CanApplyFireExposure"] = true
			}
			if env.player.mainSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "ColdExposureChance") > 0 or modDB.Sum(mod.TypeBase, nil, "ColdExposureChance") > 0 {
				condList["CanApplyColdExposure"] = true
			}
			if env.player.mainSkill.skillModList:Sum(mod.TypeBase, env.player.mainSkill.skillCfg, "LightningExposureChance") > 0 or modDB.Sum(mod.TypeBase, nil, "LightningExposureChance") > 0 {
				condList["CanApplyLightningExposure"] = true
			}
		}
	*/

	calculateAttributes := func() {
		for p := 1; p <= 2; p++ {
			for _, stat := range []string{"Str", "Dex", "Int"} {
				actor.Output[stat] = math.Max(math.Round(CalcVal(actor.ModDB, stat, nil)), 0)
				/*
					TODO Breakdown
					if breakdown {
						breakdown[stat] = breakdown.simple(nil, nil, actor.Output[stat], stat)
					}
				*/
			}

			stats := []float64{actor.Output["Str"], actor.Output["Dex"], actor.Output["Int"]}
			sort.Float64s(stats)
			actor.Output["LowestAttribute"] = stats[0]
			actor.ModDB.Conditions["TwoHighestAttributesEqual"] = stats[1] == stats[2]

			actor.ModDB.Conditions["DexHigherThanInt"] = actor.Output["Dex"] > actor.Output["Int"]
			actor.ModDB.Conditions["StrHigherThanDex"] = actor.Output["Str"] > actor.Output["Dex"]
			actor.ModDB.Conditions["IntHigherThanStr"] = actor.Output["Int"] > actor.Output["Str"]
			actor.ModDB.Conditions["StrHigherThanInt"] = actor.Output["Str"] > actor.Output["Int"]
		}
	}
	/*
		TODO calculateOmniscience
		calculateOmniscience := function (convert)
			classStats := env.spec.tree.characterData and env.spec.tree.characterData[env.classId] or env.spec.tree.classes[env.classId]

			for pass = 1, 2 do // Calculate twice because of circular dependency (X attribute higher than Y attribute)
				if pass ~= 1 {
					for _, stat in pairs({"Str","Dex","Int"}) {
						base := classStats["base_"+stat:lower()]
						actor.Output[stat] = min(round(calcLib.val(modDB, stat)), base)
						if breakdown {
							breakdown[stat] = breakdown.simple(nil, nil, actor.Output[stat], stat)
						}

						modDB.NewMod("Omni", "BASE", (modDB.Sum(mod.TypeBase, nil, stat) - base), stat+" conversion Omniscience")
						modDB.NewMod("Omni", "INC", modDB.Sum(mod.TypeIncrease, nil, stat), "Omniscience")
						modDB.NewMod("Omni", "MORE", modDB.Sum("MORE", nil, stat), "Omniscience")
					}
				}

				if pass ~= 2 {
					// Subtract out double and triple dips
					conversion := { }
					reduction := { }
					for _, type in pairs({"BASE", "INC", "MORE"}) {
						conversion[type] = { }
						for _, stat in pairs({"StrDex", "StrInt", "DexInt", "All"}) {
							conversion[type][stat] = modDB.Sum(type, nil, stat) or 0
						}
						reduction[type] = conversion[type].StrDex + conversion[type].StrInt + conversion[type].DexInt + 2*conversion[type].All
					}
					modDB.NewMod("Omni", "BASE", -reduction["BASE"], "Reduction from Double/Triple Dipped attributes to Omniscience")
					modDB.NewMod("Omni", "INC", -reduction["INC"], "Reduction from Double/Triple Dipped attributes to Omniscience")
					modDB.NewMod("Omni", "MORE", -reduction["MORE"], "Reduction from Double/Triple Dipped attributes to Omniscience")
				}

				for _, stat in pairs({"Str","Dex","Int"}) {
					base := classStats["base_"+stat:lower()]
					actor.Output[stat] = base
				}

				actor.Output["Omni"] = max(round(calcLib.val(modDB, "Omni")), 0)
				if breakdown {
					breakdown["Omni"] = breakdown.simple(nil, nil, actor.Output["Omni"], "Omni")
				}

		  stats := { actor.Output["Str"], actor.Output["Dex"], actor.Output["Int"] }
		  table.sort(stats)
		  actor.Output["LowestAttribute"] = stats[1]
		  condList["TwoHighestAttributesEqual"] = stats[2] == stats[3]

				actor.Output["LowestAttribute"] = min(actor.Output["Str"], actor.Output["Dex"], actor.Output["Int"])
				condList["DexHigherThanInt"] = actor.Output["Dex"] > actor.Output["Int"]
				condList["StrHigherThanDex"] = actor.Output["Str"] > actor.Output["Dex"]
				condList["IntHigherThanStr"] = actor.Output["Int"] > actor.Output["Str"]
				condList["StrHigherThanInt"] = actor.Output["Str"] > actor.Output["Int"]
			}
		}
	*/

	if actor.ModDB.Flag(nil, "Omniscience") {
		// TODO calculateOmniscience
		// calculateOmniscience()
	} else {
		calculateAttributes()
	}

	/*
		TODO // Calculate total attributes
		actor.Output["TotalAttr"] = actor.Output["Str"] + actor.Output["Dex"] + actor.Output["Int"]
	*/
	/*
		TODO // Special case for Devotion
		actor.Output["Devotion"] = modDB.Sum(mod.TypeBase, nil, "Devotion")
	*/

	// Add attribute bonuses
	if !env.ModDB.Flag(nil, "NoAttributeBonuses") {
		if !env.ModDB.Flag(nil, "NoStrengthAttributeBonuses") {
			if !env.ModDB.Flag(nil, "NoStrBonusToLife") {
				env.ModDB.AddMod(mod.NewFloat("Life", mod.TypeBase, math.Floor(actor.Output["Str"]/2)).Source("Strength"))
			}
			strDmgBonusRatioOverride := env.ModDB.Sum(mod.TypeBase, nil, "StrDmgBonusRatioOverride")
			if strDmgBonusRatioOverride > 0 {
				actor.StrDmgBonus = math.Floor((actor.Output["Str"] + env.ModDB.Sum(mod.TypeBase, nil, "DexIntToMeleeBonus")) * strDmgBonusRatioOverride)
			} else {
				actor.StrDmgBonus = math.Floor((actor.Output["Str"] + env.ModDB.Sum(mod.TypeBase, nil, "DexIntToMeleeBonus")) / 5)
			}
			env.ModDB.AddMod(mod.NewFloat("PhysicalDamage", mod.TypeIncrease, actor.StrDmgBonus).Source("Strength").Flag(mod.MFlagMelee))
		}

		if !env.ModDB.Flag(nil, "NoDexterityAttributeBonuses") {
			accuracyMult := data.AccuracyPerDexBase
			DexAccBonusOverride := env.ModDB.Override(nil, "DexAccBonusOverride")
			if DexAccBonusOverride != nil {
				accuracyMult = DexAccBonusOverride.Float()
			}

			env.ModDB.AddMod(mod.NewFloat("Accuracy", mod.TypeBase, actor.Output["Dex"]*accuracyMult).Source("Dexterity"))
			if !env.ModDB.Flag(nil, "NoDexBonusToEvasion") {
				env.ModDB.AddMod(mod.NewFloat("Evasion", mod.TypeIncrease, math.Floor(actor.Output["Dex"]/5)).Source("Dexterity"))
			}
		}

		if !env.ModDB.Flag(nil, "NoIntelligenceAttributeBonuses") {
			if !env.ModDB.Flag(nil, "NoIntBonusToMana") {
				env.ModDB.AddMod(mod.NewFloat("Mana", mod.TypeBase, math.Floor(actor.Output["Int"]/2)).Source("Intelligence"))
			}

			if !env.ModDB.Flag(nil, "NoIntBonusToES") {
				env.ModDB.AddMod(mod.NewFloat("EnergyShield", mod.TypeIncrease, math.Floor(actor.Output["Int"]/5)).Source("Intelligence"))
			}
		}
	}

	/*
		TODO // Check shrine buffs, must be done before life pool calculated for massive shrine
		for _, value in ipairs(modDB.List(nil, "ShrineBuff")) {
			modDB.ScaleAddList({ value.mod }, calcLib.mod(modDB, nil, "BuffEffectOnSelf", "ShrineBuffEffect"))
		}

		actor.Output["ChaosInoculation"] = modDB.Flag(nil, "ChaosInoculation")
	*/

	// Life/mana pools
	if actor.Output["ChaosInoculation"] > 0 {
		actor.Output["Life"] = 1
		actor.ModDB.Conditions["FullLife"] = true
	} else {
		base := actor.ModDB.Sum(mod.TypeBase, nil, "Life")
		inc := actor.ModDB.Sum(mod.TypeIncrease, nil, "Life")
		more := actor.ModDB.More(nil, "Life")
		conv := actor.ModDB.Sum(mod.TypeBase, nil, "LifeConvertToEnergyShield")
		actor.Output["Life"] = max(utils.RoundTo(base*(1+inc/100)*more*(1-conv/100), 0), 1)
		/*
			TODO Breakdown
			if breakdown {
				if inc ~= 0 or more ~= 1 or conv ~= 0 {
					breakdown.Life = { }
					breakdown.Life[1] = s_format("%g ^8(base)", base)
					if inc ~= 0 {
						t_insert(breakdown.Life, s_format("x %.2f ^8(increased/reduced)", 1 + inc/100))
					}
					if more ~= 1 {
						t_insert(breakdown.Life, s_format("x %.2f ^8(more/less)", more))
					}
					if conv ~= 0 {
						t_insert(breakdown.Life, s_format("x %.2f ^8(converted to Energy Shield)", 1 - conv/100))
					}
					t_insert(breakdown.Life, s_format("= %g", actor.Output["Life"]))
				}
			}
		*/
	}
	manaConv := actor.ModDB.Sum(mod.TypeBase, nil, "ManaConvertToArmour")
	actor.Output["Mana"] = utils.RoundTo(calclib.Val(actor.ModDB, "Mana")*(1-manaConv/100), 0)
	/*
		TODO Breakdown
		base := actor.ModDB.Sum(mod.TypeBase, nil, "Mana")
		inc := actor.ModDB.Sum(mod.TypeIncrease, nil, "Mana")
		more := actor.ModDB.More(nil, "Mana")
		if breakdown {
			if inc ~= 0 or more ~= 1 or manaConv ~= 0 {
				breakdown.Mana = { }
				breakdown.Mana[1] = s_format("%g ^8(base)", base)
				if inc ~= 0 {
					t_insert(breakdown.Mana, s_format("x %.2f ^8(increased/reduced)", 1 + inc/100))
				}
				if more ~= 1 {
					t_insert(breakdown.Mana, s_format("x %.2f ^8(more/less)", more))
				}
				if manaConv ~= 0 {
					t_insert(breakdown.Mana, s_format("x %.2f ^8(converted to Armour)", 1 - manaConv/100))
				}
				t_insert(breakdown.Mana, s_format("= %g", actor.Output["Mana"]))
			}
		}
	*/
	actor.Output["LowestOfMaximumLifeAndMaximumMana"] = min(actor.Output["Life"], actor.Output["Mana"])
}

func mergeKeystones(env *Environment) {
	/*
		TODO mergeKeystones
		modDB := env.modDB

		for _, name in ipairs(modDB.List(nil, "Keystone")) {
			if not env.keystonesAdded[name] and env.spec.tree.keystoneMap[name] {
				env.keystonesAdded[name] = true
				modDB.AddList(env.spec.tree.keystoneMap[name].modList)
			}
		}
	*/
}

func CalcActionSpeedMod(actor *Actor) float64 {
	actionSpeedMod := 1 + (math.Max(-data.TemporalChainsEffectCap, actor.ModDB.Sum(mod.TypeIncrease, nil, "TemporalChainsActionSpeed"))+actor.ModDB.Sum(mod.TypeIncrease, nil, "ActionSpeed"))/100
	if actor.ModDB.Flag(nil, "ActionSpeedCannotBeBelowBase") {
		actionSpeedMod = math.Max(1, actionSpeedMod)
	}
	return actionSpeedMod
}

func DoActorMisc(env *Environment, actor *Actor) {
	modDB := actor.ModDB

	// Calculate current and maximum charges
	actor.Output["PowerChargesMin"] = modDB.Sum(mod.TypeBase, nil, "PowerChargesMin")
	actor.Output["PowerChargesMax"] = modDB.Sum(mod.TypeBase, nil, "PowerChargesMax")
	actor.Output["FrenzyChargesMin"] = modDB.Sum(mod.TypeBase, nil, "FrenzyChargesMin")
	actor.Output["FrenzyChargesMax"] = utils.Ternary(modDB.Flag(nil, "MaximumFrenzyChargesIsMaximumPowerCharges"), actor.Output["PowerChargesMax"], modDB.Sum(mod.TypeBase, nil, "FrenzyChargesMax"))
	actor.Output["EnduranceChargesMin"] = modDB.Sum(mod.TypeBase, nil, "EnduranceChargesMin")
	actor.Output["EnduranceChargesMax"] = utils.Ternary(modDB.Flag(nil, "MaximumEnduranceChargesIsMaximumFrenzyCharges"), actor.Output["FrenzyChargesMax"], modDB.Sum(mod.TypeBase, nil, "EnduranceChargesMax"))
	actor.Output["SiphoningChargesMax"] = modDB.Sum(mod.TypeBase, nil, "SiphoningChargesMax")
	actor.Output["ChallengerChargesMax"] = modDB.Sum(mod.TypeBase, nil, "ChallengerChargesMax")
	actor.Output["BlitzChargesMax"] = modDB.Sum(mod.TypeBase, nil, "BlitzChargesMax")
	actor.Output["InspirationChargesMax"] = modDB.Sum(mod.TypeBase, nil, "InspirationChargesMax")
	actor.Output["CrabBarriersMax"] = modDB.Sum(mod.TypeBase, nil, "CrabBarriersMax")
	actor.Output["BrutalChargesMin"] = utils.Ternary(modDB.Flag(nil, "MinimumEnduranceChargesEqualsMinimumBrutalCharges"), actor.Output["EnduranceChargesMin"], 0)
	actor.Output["BrutalChargesMax"] = utils.Ternary(modDB.Flag(nil, "MaximumEnduranceChargesEqualsMaximumBrutalCharges"), actor.Output["EnduranceChargesMax"], 0)
	actor.Output["AbsorptionChargesMin"] = utils.Ternary(modDB.Flag(nil, "MinimumPowerChargesEqualsMinimumAbsorptionCharges"), actor.Output["PowerChargesMin"], 0)
	actor.Output["AbsorptionChargesMax"] = utils.Ternary(modDB.Flag(nil, "MaximumPowerChargesEqualsMaximumAbsorptionCharges"), actor.Output["PowerChargesMax"], 0)
	actor.Output["AfflictionChargesMin"] = utils.Ternary(modDB.Flag(nil, "MinimumFrenzyChargesEqualsMinimumAfflictionCharges"), actor.Output["FrenzyChargesMin"], 0)
	actor.Output["AfflictionChargesMax"] = utils.Ternary(modDB.Flag(nil, "MaximumFrenzyChargesEqualsMaximumAfflictionCharges"), actor.Output["FrenzyChargesMax"], 0)
	actor.Output["BloodChargesMax"] = modDB.Sum(mod.TypeBase, nil, "BloodChargesMax")

	// Initialize Charges
	actor.Output["PowerCharges"] = 0
	actor.Output["FrenzyCharges"] = 0
	actor.Output["EnduranceCharges"] = 0
	actor.Output["SiphoningCharges"] = 0
	actor.Output["ChallengerCharges"] = 0
	actor.Output["BlitzCharges"] = 0
	actor.Output["InspirationCharges"] = 0
	actor.Output["GhostShrouds"] = 0
	actor.Output["BrutalCharges"] = 0
	actor.Output["AbsorptionCharges"] = 0
	actor.Output["AfflictionCharges"] = 0
	actor.Output["BloodCharges"] = 0

	/*
		TODO // Conditionally over-write Charge values
		if modDB.Flag(nil, "UsePowerCharges") {
			actor.Output["PowerCharges"] = modDB.Override(nil, "PowerCharges") or actor.Output["PowerChargesMax"]
		}
		if modDB.Flag(nil, "PowerChargesConvertToAbsorptionCharges") {
			// we max with possible Power Charge Override from Config since Absorption Charges won't have their own config entry
			// and are converted from Power Charges
			actor.Output["AbsorptionCharges"] = max(actor.Output["PowerCharges"], min(actor.Output["AbsorptionChargesMax"], actor.Output["AbsorptionChargesMin"]))
			actor.Output["PowerCharges"] = 0
		} else {
			actor.Output["PowerCharges"] = max(actor.Output["PowerCharges"], min(actor.Output["PowerChargesMax"], actor.Output["PowerChargesMin"]))
		}
		actor.Output["RemovablePowerCharges"] = max(actor.Output["PowerCharges"] - actor.Output["PowerChargesMin"], 0)
		if modDB.Flag(nil, "UseFrenzyCharges") {
			actor.Output["FrenzyCharges"] = modDB.Override(nil, "FrenzyCharges") or actor.Output["FrenzyChargesMax"]
		}
		if modDB.Flag(nil, "FrenzyChargesConvertToAfflictionCharges") {
			// we max with possible Power Charge Override from Config since Absorption Charges won't have their own config entry
			// and are converted from Power Charges
			actor.Output["AfflictionCharges"] = max(actor.Output["FrenzyCharges"], min(actor.Output["AfflictionChargesMax"], actor.Output["AfflictionChargesMin"]))
			actor.Output["FrenzyCharges"] = 0
		} else {
			actor.Output["FrenzyCharges"] = max(actor.Output["FrenzyCharges"], min(actor.Output["FrenzyChargesMax"], actor.Output["FrenzyChargesMin"]))
		}
		actor.Output["RemovableFrenzyCharges"] = max(actor.Output["FrenzyCharges"] - actor.Output["FrenzyChargesMin"], 0)
		if modDB.Flag(nil, "UseEnduranceCharges") {
			actor.Output["EnduranceCharges"] = modDB.Override(nil, "EnduranceCharges") or actor.Output["EnduranceChargesMax"]
		}
		if modDB.Flag(nil, "EnduranceChargesConvertToBrutalCharges") {
			// we max with possible Endurance Charge Override from Config since Brutal Charges won't have their own config entry
			// and are converted from Endurance Charges
			actor.Output["BrutalCharges"] = max(actor.Output["EnduranceCharges"], min(actor.Output["BrutalChargesMax"], actor.Output["BrutalChargesMin"]))
			actor.Output["EnduranceCharges"] = 0
		} else {
			actor.Output["EnduranceCharges"] = max(actor.Output["EnduranceCharges"], min(actor.Output["EnduranceChargesMax"], actor.Output["EnduranceChargesMin"]))
		}
		actor.Output["RemovableEnduranceCharges"] = max(actor.Output["EnduranceCharges"] - actor.Output["EnduranceChargesMin"], 0)
		if modDB.Flag(nil, "UseSiphoningCharges") {
			actor.Output["SiphoningCharges"] = modDB.Override(nil, "SiphoningCharges") or actor.Output["SiphoningChargesMax"]
		}
		if modDB.Flag(nil, "UseChallengerCharges") {
			actor.Output["ChallengerCharges"] = modDB.Override(nil, "ChallengerCharges") or actor.Output["ChallengerChargesMax"]
		}
		if modDB.Flag(nil, "UseBlitzCharges") {
			actor.Output["BlitzCharges"] = modDB.Override(nil, "BlitzCharges") or actor.Output["BlitzChargesMax"]
		}
		if not env.player.mainSkill.minion {
			actor.Output["InspirationCharges"] = modDB.Override(nil, "InspirationCharges") or actor.Output["InspirationChargesMax"]
		}
		if modDB.Flag(nil, "UseGhostShrouds") {
			actor.Output["GhostShrouds"] = modDB.Override(nil, "GhostShrouds") or 3
		}
		if modDB.Flag(nil, "CryWolfMinimumPower") and modDB.Sum(mod.TypeBase, nil, "WarcryPower") < 10 {
			modDB.NewMod("WarcryPower", "OVERRIDE", 10, "Minimum Warcry Power from CryWolf")
		}
		if modDB.Flag(nil, "WarcryInfinitePower") {
			modDB.NewMod("WarcryPower", "OVERRIDE", 999999, "Warcries have infinite power")
		}
		actor.Output["BloodCharges"] = min(modDB.Override(nil, "BloodCharges") or actor.Output["BloodChargesMax"], actor.Output["BloodChargesMax"])

		actor.Output["WarcryPower"] = modDB.Override(nil, "WarcryPower") or modDB.Sum(mod.TypeBase, nil, "WarcryPower") or 0
		actor.Output["CrabBarriers"] = min(modDB.Override(nil, "CrabBarriers") or actor.Output["CrabBarriersMax"], actor.Output["CrabBarriersMax"])
		actor.Output["TotalCharges"] = actor.Output["PowerCharges"] + actor.Output["FrenzyCharges"] + actor.Output["EnduranceCharges"]
		modDB.multipliers["WarcryPower"] = actor.Output["WarcryPower"]
		modDB.multipliers["PowerCharge"] = actor.Output["PowerCharges"]
		modDB.multipliers["PowerChargeMax"] = actor.Output["PowerChargesMax"]
		modDB.multipliers["RemovablePowerCharge"] = actor.Output["RemovablePowerCharges"]
		modDB.multipliers["FrenzyCharge"] = actor.Output["FrenzyCharges"]
		modDB.multipliers["RemovableFrenzyCharge"] = actor.Output["RemovableFrenzyCharges"]
		modDB.multipliers["EnduranceCharge"] = actor.Output["EnduranceCharges"]
		modDB.multipliers["RemovableEnduranceCharge"] = actor.Output["RemovableEnduranceCharges"]
		modDB.multipliers["TotalCharges"] = actor.Output["TotalCharges"]
		modDB.multipliers["SiphoningCharge"] = actor.Output["SiphoningCharges"]
		modDB.multipliers["ChallengerCharge"] = actor.Output["ChallengerCharges"]
		modDB.multipliers["BlitzCharge"] = actor.Output["BlitzCharges"]
		modDB.multipliers["InspirationCharge"] = actor.Output["InspirationCharges"]
		modDB.multipliers["GhostShroud"] = actor.Output["GhostShrouds"]
		modDB.multipliers["CrabBarrier"] = actor.Output["CrabBarriers"]
		modDB.multipliers["BrutalCharge"] = actor.Output["BrutalCharges"]
		modDB.multipliers["AbsorptionCharge"] = actor.Output["AbsorptionCharges"]
		modDB.multipliers["AfflictionCharge"] = actor.Output["AfflictionCharges"]
		modDB.multipliers["BloodCharge"] = actor.Output["BloodCharges"]
	*/
	/*
		TODO // Process enemy modifiers
		for _, value in ipairs(modDB.List(nil, "EnemyModifier")) {
			enemyDB:AddMod(value.mod)
		}
	*/

	// Add misc buffs/debuffs
	if env.ModeCombat {
		if env.Player.MainSkill.BaseSkillModList.Flag(nil, "Cruelty") {
			modDB.Multipliers["Cruelty"] = utils.Or(modDB.Override(nil, "Cruelty"), 40)
		}
		// Fortify from a mod, or minions getting stacks from Kingmaker
		if modDB.Flag(nil, "Fortified") || modDB.Sum(mod.TypeBase, nil, "Multiplier:Fortification") > 0 {
			maxStacks := utils.Or(modDB.Override(nil, "MaximumFortification"), modDB.Sum(mod.TypeBase, nil, "MaximumFortification"))
			stacks := utils.Or(modDB.Override(nil, "FortificationStacks"), maxStacks)
			actor.Output["FortificationStacks"] = stacks
			if !modDB.Flag(nil, "Condition:NoFortificationMitigation") {
				effectScale := 1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100
				effect := math.Floor(effectScale * stacks)
				modDB.AddMod(mod.NewFloat("DamageTakenWhenHit", mod.TypeMore, -effect).Source("Fortification"))
			}
			if stacks >= maxStacks {
				modDB.AddMod(mod.NewFlag("Condition:HaveMaximumFortification", true))
			}
			modDB.Multipliers["BuffOnSelf"] = (modDB.Multipliers["BuffOnSelf"]) + 1
		}

		if modDB.Flag(nil, "Onslaught") {
			effect := math.Floor(20 * (1 + modDB.Sum(mod.TypeIncrease, nil, "OnslaughtEffect", "BuffEffectOnSelf")/100))
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, effect).Source("Onslaught"))
			modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, effect).Source("Onslaught"))
		}

		if modDB.Flag(nil, "Fanaticism") && actor.MainSkill != nil && actor.MainSkill.SkillFlags[SkillFlagSelfCast] {
			effect := math.Floor(75 * (1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeMore, effect).Source("Fanaticism").Flag(mod.MFlagCast))
			modDB.AddMod(mod.NewFloat("Cost", mod.TypeIncrease, -effect).Source("Fanaticism").Flag(mod.MFlagCast))
			modDB.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeIncrease, effect).Source("Fanaticism").Flag(mod.MFlagCast))
		}
		if modDB.Flag(nil, "UnholyMight") {
			effect := math.Floor(30 * (1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(mod.NewFloat("PhysicalDamageGainAsChaos", mod.TypeBase, effect).Source("Unholy Might"))
		}
		if modDB.Flag(nil, "Tailwind") {
			effect := math.Floor(8 * (1 + modDB.Sum(mod.TypeIncrease, nil, "TailwindEffectOnSelf", "BuffEffectOnSelf")/100))
			modDB.AddMod(mod.NewFloat("ActionSpeed", mod.TypeIncrease, effect).Source("Tailwind"))
		}
		if modDB.Flag(nil, "Adrenaline") {
			effectMod := 1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100
			modDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, math.Floor(100*effectMod)).Source("Adrenaline"))
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, math.Floor(25*effectMod)).Source("Adrenaline"))
			modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, math.Floor(25*effectMod)).Source("Adrenaline"))
			modDB.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, math.Floor(10*effectMod)).Source("Adrenaline"))
		}
		if modDB.Flag(nil, "Convergence") {
			effect := math.Floor(30 * (1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(mod.NewFloat("ElementalDamage", mod.TypeMore, effect).Source("Convergence"))
		}
		if modDB.Flag(nil, "HerEmbrace") {
			modDB.Conditions["HerEmbrace"] = true
			modDB.AddMod(mod.NewFloat("AvoidStun", mod.TypeBase, 100).Source("Her Embrace"))
			modDB.AddMod(mod.NewFloat("PhysicalDamageGainAsFire", mod.TypeBase, 123).Source("Her Embrace").Flag(mod.MFlagSword))
			modDB.AddMod(mod.NewFloat("AvoidFreeze", mod.TypeBase, 100).Source("Her Embrace"))
			modDB.AddMod(mod.NewFloat("AvoidChill", mod.TypeBase, 100).Source("Her Embrace"))
			modDB.AddMod(mod.NewFloat("AvoidIgnite", mod.TypeBase, 100).Source("Her Embrace"))
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 20).Source("Her Embrace"))
			modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, 20).Source("Her Embrace"))
		}
		if modDB.Flag(nil, "Condition:PhantasmalMight") {
			modDB.Multipliers["BuffOnSelf"] = (modDB.Multipliers["BuffOnSelf"]) + (utils.OrDefault(actor.Output["ActivePhantasmLimit"], 1)) - 1 // slight hack to not double count the initial buff
		}
		if modDB.Flag(nil, "Elusive") {
			maxSkillInc := modDB.Max(&moddb.ListCfg{
				Source: utils.Ptr(mod.SourceSkill),
			}, "ElusiveEffect")

			inc := modDB.Sum(mod.TypeIncrease, nil, "ElusiveEffect", "BuffEffectOnSelf")
			if actor.MainSkill.SkillModList.Flag(nil, "SupportedByNightblade") {
				inc = inc + modDB.Sum(mod.TypeIncrease, nil, "NightbladeSupportedElusiveEffect")
			}
			inc = inc + maxSkillInc
			actor.Output["ElusiveEffectMod"] = (1 + inc/100) * modDB.More(nil, "ElusiveEffect", "BuffEffectOnSelf") * 100
			// if we want the max skill to not be noted as its own breakdown table entry, comment out below
			modDB.AddMod(mod.NewFloat("ElusiveEffect", mod.TypeIncrease, maxSkillInc).Source("Max Skill Effect"))
			// Override elusive effect if set.
			if modDB.Override(nil, "ElusiveEffect").Float() > 0 {
				actor.Output["ElusiveEffectMod"] = min(modDB.Override(nil, "ElusiveEffect").Float(), actor.Output["ElusiveEffectMod"])
			}
			effect := actor.Output["ElusiveEffectMod"] / 100
			modDB.Conditions["Elusive"] = true
			modDB.AddMod(mod.NewFloat("AvoidPhysicalDamageChance", mod.TypeBase, math.Floor(15*effect)).Source("Elusive"))
			modDB.AddMod(mod.NewFloat("AvoidLightningDamageChance", mod.TypeBase, math.Floor(15*effect)).Source("Elusive"))
			modDB.AddMod(mod.NewFloat("AvoidColdDamageChance", mod.TypeBase, math.Floor(15*effect)).Source("Elusive"))
			modDB.AddMod(mod.NewFloat("AvoidFireDamageChance", mod.TypeBase, math.Floor(15*effect)).Source("Elusive"))
			modDB.AddMod(mod.NewFloat("AvoidChaosDamageChance", mod.TypeBase, math.Floor(15*effect)).Source("Elusive"))
			modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, math.Floor(30*effect)).Source("Elusive"))
		}
		WitherEffectStack := modDB.Max(nil, "WitherEffectStack")
		if WitherEffectStack > 0 {
			modDB.AddMod(mod.NewFlag("Condition:CanWither", true).Source("Config"))
			modDB.AddMod(mod.NewFloat("ChaosDamageTaken", mod.TypeIncrease, WitherEffectStack).Source("Withered").Tag(mod.Multiplier("WitheredStack").Limit(15)))
		}
		if modDB.Flag(nil, "Blind") {
			if !modDB.Flag(nil, "IgnoreBlindHitChance") {
				effect := 1 + modDB.Sum(mod.TypeIncrease, nil, "BlindEffect", "BuffEffectOnSelf")/100
				// Override Blind effect if set.
				if modDB.Override(nil, "BlindEffect").Float() > 0 {
					effect = min(modDB.Override(nil, "BlindEffect").Float()/100, effect)
				}
				modDB.AddMod(mod.NewFloat("Accuracy", mod.TypeMore, math.Floor(-20*effect)).Source("Blind"))
				modDB.AddMod(mod.NewFloat("Evasion", mod.TypeMore, math.Floor(-20*effect)).Source("Blind"))
			}
		}
		if modDB.Flag(nil, "Chill") {
			chillValue := utils.Or(modDB.Override(nil, "ChillVal"), *data.NonDamagingAilments[data.AilmentChill].Default)

			chillSelf := utils.Ternary(modDB.Flag(nil, "Condition:ChilledSelf"), modDB.Sum(mod.TypeIncrease, nil, "EnemyChillEffect")/100, 0)
			totalChillSelfEffect := calclib.Mod(modDB, nil, "SelfChillEffect") + chillSelf

			effect := min(max(math.Floor(chillValue*totalChillSelfEffect), 0), utils.Or(modDB.Override(nil, "ChillMax"), data.NonDamagingAilments[data.AilmentChill].Max))

			modDB.AddMod(mod.NewFloat("ActionSpeed", mod.TypeIncrease, effect*utils.Ternary[float64](modDB.Flag(nil, "SelfChillEffectIsReversed"), 1, -1)).Source("Chill"))
		}
		if modDB.Flag(nil, "Freeze") {
			effect := max(math.Floor(70*calclib.Mod(modDB, nil, "SelfChillEffect")), 0)
			modDB.AddMod(mod.NewFloat("ActionSpeed", mod.TypeIncrease, -effect).Source("Freeze"))
		}
		if modDB.Flag(nil, "CanLeechLifeOnFullLife") {
			modDB.Conditions["Leeching"] = true
			modDB.Conditions["LeechingLife"] = true
			env.Build.SetConfigOption(pob.Input{
				Name:    "conditionLeeching",
				Boolean: utils.Ptr(true),
			})
		}
		if modDB.Flag(nil, "CanLeechLifeOnFullEnergyShield") {
			modDB.Conditions["Leeching"] = true
			modDB.Conditions["LeechingEnergyShield"] = true
			env.Build.SetConfigOption(pob.Input{
				Name:    "conditionLeeching",
				Boolean: utils.Ptr(true),
			})
		}
		if modDB.Flag(nil, "Condition:InfusionActive") {
			effect := 1 + modDB.Sum(mod.TypeIncrease, nil, "InfusionEffect", "BuffEffectOnSelf")/100
			if modDB.Flag(nil, "Condition:HavePhysicalInfusion") {
				modDB.Conditions["PhysicalInfusion"] = true
				modDB.Conditions["Infusion"] = true
				modDB.AddMod(mod.NewFloat("PhysicalDamage", mod.TypeMore, 10*effect).Source("Infusion"))
			}
			if modDB.Flag(nil, "Condition:HaveFireInfusion") {
				modDB.Conditions["FireInfusion"] = true
				modDB.Conditions["Infusion"] = true
				modDB.AddMod(mod.NewFloat("FireDamage", mod.TypeMore, 10*effect).Source("Infusion"))
			}
			if modDB.Flag(nil, "Condition:HaveColdInfusion") {
				modDB.Conditions["ColdInfusion"] = true
				modDB.Conditions["Infusion"] = true
				modDB.AddMod(mod.NewFloat("ColdDamage", mod.TypeMore, 10*effect).Source("Infusion"))
			}
			if modDB.Flag(nil, "Condition:HaveLightningInfusion") {
				modDB.Conditions["LightningInfusion"] = true
				modDB.Conditions["Infusion"] = true
				modDB.AddMod(mod.NewFloat("LightningDamage", mod.TypeMore, 10*effect).Source("Infusion"))
			}
			if modDB.Flag(nil, "Condition:HaveChaosInfusion") {
				modDB.Conditions["ChaosInfusion"] = true
				modDB.Conditions["Infusion"] = true
				modDB.AddMod(mod.NewFloat("ChaosDamage", mod.TypeMore, 10*effect).Source("Infusion"))
			}
		}
		if modDB.Flag(nil, "Condition:CanGainRage") || modDB.Sum(mod.TypeBase, nil, "RageRegen") > 0 {
			actor.Output["MaximumRage"] = modDB.Sum(mod.TypeBase, nil, "MaximumRage")
			modDB.Multipliers["MaxRageVortexSacrifice"] = actor.Output["MaximumRage"] / 4
			modDB.AddMod(mod.NewFloat("Multiplier:Rage", mod.TypeBase, 1).Source("Base").Tag(mod.Multiplier("RageStack").Limit(actor.Output["MaximumRage"])))
		}
		if modDB.Sum(mod.TypeBase, nil, "CoveredInAshEffect") > 0 {
			effect := modDB.Sum(mod.TypeBase, nil, "CoveredInAshEffect")
			modDB.AddMod(mod.NewFloat("FireDamageTaken", mod.TypeIncrease, min(effect, 20)).Source("Covered in Ash"))
		}
		if modDB.Sum(mod.TypeBase, nil, "CoveredInFrostEffect") > 0 {
			effect := modDB.Sum(mod.TypeBase, nil, "CoveredInFrostEffect")
			modDB.AddMod(mod.NewFloat("ColdDamageTaken", mod.TypeIncrease, min(effect, 20)).Source("Covered in Frost"))
		}
		if modDB.Flag(nil, "HasMalediction") {
			modDB.AddMod(mod.NewFloat("DamageTaken", mod.TypeIncrease, 10).Source("Malediction"))
			modDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, -10).Source("Malediction"))
		}
	}
}
