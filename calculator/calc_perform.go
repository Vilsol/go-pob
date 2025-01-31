package calculator

import (
	"fmt"
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
			// Build minion skills
			activeSkill.Minion.ModDB = moddb.NewModDB()
			activeSkill.Minion.ModDB.Actor = activeSkill.Minion
			CalcCreateMinionSkills(env, activeSkill)
			activeSkill.SkillPartName = activeSkill.Minion.MainSkill.ActiveEffect.GrantedEffect.Raw.ID
		}
	}

	env.Player.Output = make(map[string]float64)
	env.Player.OutputTable = make(map[OutTable]map[string]float64)
	env.Player.OutputStrings = make(map[string]string)

	env.Enemy.Output = make(map[string]float64)
	env.Enemy.OutputTable = make(map[OutTable]map[string]float64)
	env.Enemy.OutputStrings = make(map[string]string)

	env.Minion = env.Player.MainSkill.Minion
	if env.Minion != nil {
		// Initialise minion modifier database
		env.Player.OutputTable["Minion"] = make(map[string]float64)
		env.Minion.Output = env.Player.OutputTable["Minion"]
		env.Minion.ModDB.Multipliers["Level"] = float64(env.Minion.Level)
		initModDB(env, env.Minion.ModDB)
		env.Minion.ModDB.AddMod(mod.NewFloat("Life", mod.TypeBase, math.Floor(float64(env.Minion.LifeTable[env.Minion.Level])*env.Minion.MinionData.Life)).Source("Base"))
		if env.Minion.MinionData.EnergyShield != 0 {
			env.Minion.ModDB.AddMod(mod.NewFloat("EnergyShield", mod.TypeBase, math.Floor(data.MonsterAllyLifeTable[env.Minion.Level]*env.Minion.MinionData.Life*env.Minion.MinionData.EnergyShield)).Source("Base"))
		}
		if env.Minion.MinionData.Armour != 0 {
			env.Minion.ModDB.AddMod(mod.NewFloat("Armour", mod.TypeBase, math.Floor((10+float64(env.Minion.Level)*2)*env.Minion.MinionData.Armour*math.Pow(1.038, float64(env.Minion.Level)))).Source("Base"))
		}
		env.Minion.ModDB.AddMod(mod.NewFloat("Evasion", mod.TypeBase, utils.RoundTo((30+float64(env.Minion.Level)*5)*math.Pow(1.03, float64(env.Minion.Level)), 0)).Source("Base"))
		env.Minion.ModDB.AddMod(mod.NewFloat("Accuracy", mod.TypeBase, utils.RoundTo((17+float64(env.Minion.Level)/2)*(env.Minion.MinionData.Accuracy)*math.Pow(1.03, float64(env.Minion.Level)), 0)).Source("Base"))
		env.Minion.ModDB.AddMod(mod.NewFloat("CritMultiplier", mod.TypeBase, 30).Source("Base"))
		env.Minion.ModDB.AddMod(mod.NewFloat("CritDegenMultiplier", mod.TypeBase, 30).Source("Base"))
		env.Minion.ModDB.AddMod(mod.NewFloat("FireResist", mod.TypeBase, env.Minion.MinionData.FireResist).Source("Base"))
		env.Minion.ModDB.AddMod(mod.NewFloat("ColdResist", mod.TypeBase, env.Minion.MinionData.ColdResist).Source("Base"))
		env.Minion.ModDB.AddMod(mod.NewFloat("LightningResist", mod.TypeBase, env.Minion.MinionData.LightningResist).Source("Base"))
		env.Minion.ModDB.AddMod(mod.NewFloat("ChaosResist", mod.TypeBase, env.Minion.MinionData.ChaosResist).Source("Base"))
		env.Minion.ModDB.AddMod(mod.NewFloat("CritChance", mod.TypeIncrease, 200).Source("Base").Tag(mod.Multiplier("PowerCharge")))
		env.Minion.ModDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 15).Source("Base").Tag(mod.Multiplier("FrenzyCharge")))
		env.Minion.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, 4).Source("Base").Tag(mod.Multiplier("FrenzyCharge")))
		env.Minion.ModDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, 5).Source("Base").Tag(mod.Multiplier("FrenzyCharge")))
		env.Minion.ModDB.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, 15).Source("Base").Tag(mod.Multiplier("EnduranceCharge")))
		env.Minion.ModDB.AddMod(mod.NewFloat("ElementalResist", mod.TypeBase, 15).Source("Base").Tag(mod.Multiplier("EnduranceCharge")))
		env.Minion.ModDB.AddMod(mod.NewFloat("ProjectileCount", mod.TypeBase, 1).Source("Base"))
		env.Minion.ModDB.AddMod(mod.NewFloat("MaximumFortification", mod.TypeBase, 20).Source("Base"))
		env.Minion.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, -50).Source("Base").KeywordFlag(mod.KeywordFlagPoison))
		env.Minion.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, -50).Source("Base").KeywordFlag(mod.KeywordFlagIgnite))
		env.Minion.ModDB.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "bleedBasePercent", Value: 70 / 6}).Source("Base"))
		env.Minion.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, 200).Source("Base").KeywordFlag(mod.KeywordFlagBleed).Tag(mod.ActorCondition("enemy", "Moving")))
		for _, Mod := range env.Minion.MinionData.ModList {
			env.Minion.ModDB.AddMod(Mod)
		}
		for _, Mod := range env.Player.MainSkill.ExtraSkillModList {
			env.Minion.ModDB.AddMod(Mod)
		}
		if env.AegisModList != nil {
			env.Minion.ItemList["Weapon 3"] = env.Player.ItemList["Weapon 2"]
			env.Minion.ModDB.AddList(env.AegisModList)
		}
		if env.TheIronMass != nil && env.Minion.Type == "RaisedSkeleton" {
			env.Minion.ModDB.AddList(env.TheIronMass)
		}
		if env.Player.MainSkill.SkillData.MinionUseBowAndQuiver {
			if env.Player.WeaponData1.Type == "Bow" {
				env.Minion.ModDB.AddList(env.Player.ItemList["Weapon 1"].SlotModList[1])
			}
			if env.Player.ItemList["Weapon 2"] != nil && env.Player.ItemList["Weapon 2"].Type == "Quiver" {
				env.Minion.ModDB.AddList(env.Player.ItemList["Weapon 2"].ModList)
			}
		}
		/*
			TODO Items
			if env.Minion.ItemSet != nil || env.Minion.Uses != nil {
				for slotName, slot := range env.Build.ItemsTab.Slots {
					if env.Minion.Uses[slotName] {
						var item *ItemData
						if env.Minion.ItemSet != nil {
							if slot.WeaponSet == 1 && env.Minion.ItemSet.useSecondWeaponSet {
								slotName = slotName + " Swap"
							}
							item = env.Build.ItemsTab.Items[env.Minion.ItemSet[slotName].SelItemId]
						} else {
							item = env.Player.ItemList[slotName]
						}
						if item != nil {
							env.Minion.ItemList[slotName] = item
							env.Minion.ModDB.AddList(item.ModList || item.SlotModList[slot.SlotNum])
						}
					}
				}
			}
		*/
		if env.ModDB.Flag(nil, "StrengthAddedToMinions") {
			env.Minion.ModDB.AddMod(mod.NewFloat("Str", mod.TypeBase, utils.RoundTo(calclib.Val(env.ModDB, "Str"), 0)).Source("Player"))
		}
		if env.ModDB.Flag(nil, "HalfStrengthAddedToMinions") {
			env.Minion.ModDB.AddMod(mod.NewFloat("Str", mod.TypeBase, utils.RoundTo(calclib.Val(env.ModDB, "Str")*0.5, 0)).Source("Player"))
		}
	}

	if env.AegisModList != nil {
		env.Player.ItemList["Weapon 2"] = nil
	}

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
			if activeSkill.skillFlags.warcry and not env.ModDB.Flag(nil, "AlreadyGlobalWarcryCooldown") {
				cooldown := calcSkillCooldown(activeSkill.skillModList, activeSkill.skillCfg, activeSkill.skillData)
				warcryList := { }
				numWarcries, sumWarcryCooldown := 0
				for _, activeSkill := range (env.Player.activeSkillList) {
					if activeSkill.skillTypes[SkillType.Warcry] {
						warcryList[activeSkill.skillCfg.skillName] = true
					}
				}
				for _, warcry := range (warcryList) {
					numWarcries = numWarcries + 1
					sumWarcryCooldown = (sumWarcryCooldown or 0) + cooldown
				}
				env.Player.modDB.NewMod("GlobalWarcryCooldown", "BASE", sumWarcryCooldown)
				env.Player.modDB.NewMod("GlobalWarcryCount", "BASE", numWarcries)
				env.ModDB.NewMod("AlreadyGlobalWarcryCooldown", "FLAG", true, "Config") // Prevents effect from applying multiple times
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
				globalCooldown := env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
				globalCount := env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
				uptime := min(full_duration / actual_cooldown, 1)
				buff_inc := 1 + activeSkill.skillModList:Sum(mod.TypeIncrease, activeSkill.skillCfg, "BuffEffect") / 100
				warcryPowerBonus := math.Floor((env.ModDB.Override(nil, "WarcryPower") or env.ModDB.Sum(mod.TypeBase, nil, "WarcryPower") or 0) / 5)
				if env.ModDB.Flag(nil, "WarcryShareCooldown") {
					uptime = min(full_duration / (actual_cooldown + (globalCooldown - actual_cooldown) / globalCount), 1)
				}
				if env.ModDB.Flag(nil, "Condition:WarcryMaxHit") {
					uptime = 1
				}
				if activeSkill.activeEffect.grantedEffect.name == "Ancestral Cry" and not env.ModDB.Flag(nil, "AncestralActive") {
					ancestralArmour := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "AncestralArmourPer5MP")
					ancestralArmourMax := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "AncestralArmourMax")
					ancestralArmourIncrease := activeSkill.skillModList:Sum(mod.TypeIncrease, env.Player.MainSkill.skillCfg, "AncestralArmourMax")
					ancestralStrikeRange := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "AncestralMeleeWeaponRangePer5MP")
					ancestralStrikeRangeMax := math.Floor(6 * buff_inc)
					env.Player.modDB.NewMod("NumAncestralExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "AncestralExertedAttacks") + extraExertions)
					ancestralArmourMax = math.Floor(ancestralArmourMax * buff_inc)
					if warcryPowerBonus != 0 {
						ancestralArmour = math.Floor(ancestralArmour * warcryPowerBonus * buff_inc) / warcryPowerBonus
						ancestralStrikeRange = math.Floor(ancestralStrikeRange * warcryPowerBonus * buff_inc) / warcryPowerBonus
					} else {
						// Since no buff happens, you don't get the divergent increase.
						ancestralArmourIncrease = 0
					}
					env.Player.modDB.NewMod("Armour", "BASE", ancestralArmour * uptime, "Ancestral Cry", { type = "Multiplier", var = "WarcryPower", div = 5, limit = ancestralArmourMax, limitTotal = true })
					env.Player.modDB.NewMod("Armour", "INC", ancestralArmourIncrease * uptime, "Ancestral Cry")
					env.Player.modDB.NewMod("MeleeWeaponRange", "BASE", ancestralStrikeRange * uptime, "Ancestral Cry", { type = "Multiplier", var = "WarcryPower", div = 5, limit = ancestralStrikeRangeMax, limitTotal = true })
					env.ModDB.NewMod("AncestralActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Enduring Cry" and not env.ModDB.Flag(nil, "EnduringActive") {
					heal_over_1_sec := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "EnduringCryLifeRegen")
					resist_all_per_endurance := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "EnduringCryElementalResist")
					pdr_per_endurance := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "EnduringCryPhysicalDamageReduction")
					env.Player.modDB.NewMod("LifeRegen", "BASE", heal_over_1_sec, "Enduring Cry", { type = "Condition", var = "LifeRegenBurstFull" })
					env.Player.modDB.NewMod("LifeRegen", "BASE", heal_over_1_sec / actual_cooldown, "Enduring Cry", { type = "Condition", var = "LifeRegenBurstAvg" })
					env.Player.modDB.NewMod("ElementalResist", "BASE", math.Floor(resist_all_per_endurance * buff_inc) * uptime, "Enduring Cry", { type = "Multiplier", var = "EnduranceCharge" })
					env.Player.modDB.NewMod("PhysicalDamageReduction", "BASE", math.Floor(pdr_per_endurance * buff_inc) * uptime, "Enduring Cry", { type = "Multiplier", var = "EnduranceCharge" })
					env.ModDB.NewMod("EnduringActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Infernal Cry" and not env.ModDB.Flag(nil, "InfernalActive") {
					infernalAshEffect := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "InfernalFireTakenPer5MP")
					env.Player.modDB.NewMod("NumInfernalExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "InfernalExertedAttacks") + extraExertions)
					if env.mode_effective {
						env.Player.modDB.NewMod("CoveredInAshEffect", "BASE", infernalAshEffect * uptime, { type = "Multiplier", var = "WarcryPower", div = 5 })
					}
					env.ModDB.NewMod("InfernalActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Battlemage's Cry" and not env.ModDB.Flag(nil, "BattlemageActive") {
					battlemageSpellToAttack := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "BattlemageSpellIncreaseApplyToAttackPer5MP")
					battlemageSpellToAttackMax := math.Floor(150 * buff_inc)
					battlemageCritChance := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "BattlemageCritChancePer5MP")
					battlemageCritChanceMax := math.Floor(30 * buff_inc)
					env.Player.modDB.NewMod("NumBattlemageExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "BattlemageExertedAttacks") + extraExertions)
					if warcryPowerBonus != 0 {
						battlemageCritChance = math.Floor(battlemageCritChance * warcryPowerBonus * buff_inc) / warcryPowerBonus
						battlemageSpellToAttack = math.Floor(battlemageSpellToAttack * warcryPowerBonus * buff_inc) / warcryPowerBonus
						env.ModDB.NewMod("SpellDamageAppliesToAttacks", "FLAG", true)
					}
					env.Player.modDB.NewMod("CritChance", "INC", battlemageCritChance * uptime, "Battlemage's Cry", { type = "Multiplier", var = "WarcryPower", div = 5, limit = battlemageCritChanceMax, limitTotal = true })
					env.Player.modDB.NewMod("ImprovedSpellDamageAppliesToAttacks", "MAX", battlemageSpellToAttack * uptime, "Battlemage's Cry", { type = "Multiplier", var = "WarcryPower", div = 5, limit = battlemageSpellToAttackMax, limitTotal = true })
					env.ModDB.NewMod("BattlemageActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Intimidating Cry" and not env.ModDB.Flag(nil, "IntimidatingActive") {
					intimidatingOverwhelmEffect := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "IntimidatingPDRPer5MP")
					if warcryPowerBonus != 0 {
						intimidatingOverwhelmEffect = math.Floor(intimidatingOverwhelmEffect * warcryPowerBonus * buff_inc) / warcryPowerBonus
					}
					env.Player.modDB.NewMod("NumIntimidatingExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "IntimidatingExertedAttacks") + extraExertions)
					env.Player.modDB.NewMod("EnemyPhysicalDamageReduction", "BASE", -intimidatingOverwhelmEffect * uptime, "Intimidating Cry Buff", { type = "Multiplier", var = "WarcryPower", div = 5, limit = 6 })
					env.ModDB.NewMod("IntimidatingActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Rallying Cry" and not env.ModDB.Flag(nil, "RallyingActive") {
					env.Player.modDB.NewMod("NumRallyingExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "RallyingExertedAttacks") + extraExertions)
					env.Player.modDB.NewMod("RallyingExertMoreDamagePerAlly",  "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "RallyingCryExertDamageBonus"))
					rallyingWeaponEffect := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "RallyingCryAllyDamageBonusPer5Power")
					// Rallying cry divergent more effect of buff
					rallyingBonusMoreMultiplier := 1 + (activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "RallyingCryMinionDamageBonusMultiplier") or 0)
					if warcryPowerBonus != 0 {
						rallyingWeaponEffect = math.Floor(rallyingWeaponEffect * warcryPowerBonus * buff_inc) / warcryPowerBonus
					}
					// Special handling for the minion side to add the flat damage bonus
					if env.Minion {
						// Add all damage types
						dmgTypeList := {"Physical", "Lightning", "Cold", "Fire", "Chaos"}
						for _, damageType := range (dmgTypeList) {
							env.Minion.ModDB.NewMod(damageType+"Min", "BASE", math.Floor((env.Player.weaponData1[damageType+"Min"] or 0) * rallyingBonusMoreMultiplier * rallyingWeaponEffect / 100) * uptime, "Rallying Cry", { type = "Multiplier", actor = "parent", var = "WarcryPower", div = 5, limit = 6.6667})
							env.Minion.ModDB.NewMod(damageType+"Max", "BASE", math.Floor((env.Player.weaponData1[damageType+"Max"] or 0) * rallyingBonusMoreMultiplier * rallyingWeaponEffect / 100) * uptime, "Rallying Cry", { type = "Multiplier", actor = "parent", var = "WarcryPower", div = 5, limit = 6.6667})
						}
					}
					env.ModDB.NewMod("RallyingActive", "FLAG", true) // Prevents effect from applying multiple times
				} else if activeSkill.activeEffect.grantedEffect.name == "Seismic Cry" and not env.ModDB.Flag(nil, "SeismicActive") {
					seismicStunEffect := activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "SeismicStunThresholdPer5MP")
					if warcryPowerBonus != 0 {
						seismicStunEffect = math.Floor(seismicStunEffect * warcryPowerBonus * buff_inc) / warcryPowerBonus
					}
					env.Player.modDB.NewMod("NumSeismicExerts", "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "SeismicExertedAttacks") + extraExertions)
					env.Player.modDB.NewMod("SeismicIncAoEPerExert",  "BASE", activeSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "SeismicAoEMultiplier"))
					if env.mode_effective {
						env.Player.modDB.NewMod("EnemyStunThreshold", "INC", -seismicStunEffect * uptime, "Seismic Cry Buff", { type = "Multiplier", var = "WarcryPower", div = 5, limit = 6 })
					}
					env.ModDB.NewMod("SeismicActive", "FLAG", true) // Prevents effect from applying multiple times
				}
			}
		*/
		/*
			TODO Triggers
			if activeSkill.skillData.triggeredByBrand and not activeSkill.skillFlags.minion {
				activeSkill.skillData.triggered = true
				spellCount, quality := 0
				for _, skill := range (env.Player.activeSkillList) {
					match1 := skill.activeEffect.grantedEffect.fromItem and skill.socketGroup.slot == activeSkill.socketGroup.slot
					match2 := not skill.activeEffect.grantedEffect.fromItem and skill.socketGroup == activeSkill.socketGroup
					if skill.skillData.triggeredByBrand and (match1 or match2) {
						spellCount = spellCount + 1
					}
					if skill.activeEffect.grantedEffect.name == "Arcanist Brand" and (match1 or match2) {
						quality = skill.activeEffect.quality / 2
					}
				}
				addTriggerIncMoreMods(activeSkill, env.Player.MainSkill)
				activeSkill.skillModList:NewMod("ArcanistSpellsLinked", "BASE", spellCount, "Skill")
				activeSkill.skillModList:NewMod("BrandActivationFrequency", "INC", quality, "Skill")
			}
			if activeSkill.skillData.triggeredOnDeath and not activeSkill.skillFlags.minion {
				activeSkill.skillData.triggered = true
				for _, value := range (activeSkill.skillModList:Tabulate("INC", env.Player.MainSkill.skillCfg, "TriggeredDamage")) {
					activeSkill.skillModList:NewMod("Damage", "INC", value.mod.value, value.mod.source, value.mod.flags, value.mod.keywordFlags, unpack(value.mod))
				}
				for _, value := range (activeSkill.skillModList:Tabulate("MORE", env.Player.MainSkill.skillCfg, "TriggeredDamage")) {
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

	var breakdown = NewBreakdown(env.ModDB, env.Player.Output, env.Player)
	env.Player.Breakdown = breakdown
	if env.Minion != nil {
		env.Minion.Breakdown = NewBreakdown(env.Minion.ModDB, env.Minion.Output, env.Minion.Actor)
	}

	/*
		TODO // Special handling of Mageblood
		maxActiveMagicUtilityCount := env.ModDB.Sum(mod.TypeBase, nil, "ActiveMagicUtilityFlasks")
		if maxActiveMagicUtilityCount > 0 {
			curActiveMagicUtilityCount := 0
			for _, slot := range (env.build.itemsTab.orderedSlots) {
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
			effectInc := env.ModDB.Sum(mod.TypeIncrease, nil, "FlaskEffect")
			flaskBuffs := { }
			usingFlask := false
			usingLifeFlask := false
			usingManaFlask := false
			for item := range (env.flasks) {
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
					flaskEffectInc = flaskEffectInc + env.ModDB.Sum(mod.TypeIncrease, nil, "MagicUtilityFlaskEffect")
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
						for _, mod := range (item.modList) {
							key = key + modLib.formatModParams(mod) + "&"
						}
					}
					mergeBuff(srcList, flaskBuffs, key)
				}
			}
			if not env.ModDB.Flag(nil, "FlasksDoNotApplyToPlayer") {
				env.ModDB.conditions["UsingFlask"] = usingFlask
				env.ModDB.conditions["UsingLifeFlask"] = usingLifeFlask
				env.ModDB.conditions["UsingManaFlask"] = usingManaFlask
				for _, buffModList := range (flaskBuffs) {
					env.ModDB.AddList(buffModList)
				}
			}
			if env.Minion and env.ModDB.Flag(env.Player.MainSkill.skillCfg, "FlasksApplyToMinion") {
				minionModDB := env.Minion.ModDB
				minionModDB.conditions["UsingFlask"] = usingFlask
				minionModDB.conditions["UsingLifeFlask"] = usingLifeFlask
				minionModDB.conditions["UsingManaFlask"] = usingManaFlask
				for _, buffModList := range (flaskBuffs) {
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
		if env.Minion {
			for _, value := range (env.Player.MainSkill.skillModList:List(env.Player.MainSkill.skillCfg, "MinionModifier")) {
				if not value.type or env.Minion.type == value.type {
					env.Minion.ModDB.AddMod(value.mod)
				}
			}
			for _, name := range (env.Minion.ModDB.List(nil, "Keystone")) {
				if env.spec.tree.keystoneMap[name] {
					env.Minion.ModDB.AddList(env.spec.tree.keystoneMap[name].modList)
				}
			}
			doActorAttribsPoolsConditions(env, env.Minion)
		}
	*/

	// Calculate skill life and mana reservations
	env.Player.Reserved_LifeBase = 0
	env.Player.Reserved_LifePercent = env.ModDB.Sum(mod.TypeBase, nil, "ExtraLifeReserved")
	env.Player.Reserved_ManaBase = 0
	env.Player.Reserved_ManaPercent = 0
	for _, activeSkill := range env.Player.ActiveSkillList {
		if activeSkill.SkillTypes[data.SkillTypeHasReservation] && !activeSkill.SkillTypes[data.SkillTypeReservationBecomesCost] {
			skillModList := activeSkill.SkillModList
			skillCfg := activeSkill.SkillCfg
			mult := skillModList.More(skillCfg, "SupportManaMultiplier")
			pool := map[string]map[string]float64{
				"Mana": {},
				"Life": {},
			}
			pool["Mana"]["baseFlat"] = utils.OrF(activeSkill.SkillData.ManaReservationFlat, utils.OrNil(activeSkill.ActiveEffect.GrantedEffectLevel.ManaReservationFlat, 0))
			if skillModList.Flag(skillCfg, "ManaCostGainAsReservation") && activeSkill.ActiveEffect.GrantedEffectLevel.Cost != nil {
				pool["Mana"]["baseFlat"] = skillModList.Sum(mod.TypeBase, skillCfg, "ManaCostBase") + float64(activeSkill.ActiveEffect.GrantedEffectLevel.Cost["Mana"])
			}
			pool["Mana"]["basePercent"] = utils.OrF(activeSkill.SkillData.ManaReservationPercent, utils.OrNil(activeSkill.ActiveEffect.GrantedEffectLevel.ManaReservationPercent, 0))
			pool["Life"]["baseFlat"] = utils.OrF(activeSkill.SkillData.LifeReservationFlat, utils.OrNil(activeSkill.ActiveEffect.GrantedEffectLevel.LifeReservationFlat, 0))
			if skillModList.Flag(skillCfg, "LifeCostGainAsReservation") && activeSkill.ActiveEffect.GrantedEffectLevel.Cost != nil {
				pool["Life"]["baseFlat"] = skillModList.Sum(mod.TypeBase, skillCfg, "LifeCostBase") + float64(activeSkill.ActiveEffect.GrantedEffectLevel.Cost["Life"])
			}
			pool["Life"]["basePercent"] = utils.OrF(activeSkill.SkillData.LifeReservationPercent, utils.OrNil(activeSkill.ActiveEffect.GrantedEffectLevel.LifeReservationPercent, 0))
			if skillModList.Flag(skillCfg, "BloodMagicReserved") {
				pool["Life"]["baseFlat"] = pool["Life"]["baseFlat"] + pool["Mana"]["baseFlat"]
				pool["Mana"]["baseFlat"] = 0
				activeSkill.SkillData.LifeReservationFlatForced = activeSkill.SkillData.ManaReservationFlatForced
				activeSkill.SkillData.ManaReservationFlatForced = nil
				pool["Life"]["basePercent"] = pool["Life"]["basePercent"] + pool["Mana"]["basePercent"]
				pool["Mana"]["basePercent"] = 0
				activeSkill.SkillData.LifeReservationPercentForced = activeSkill.SkillData.ManaReservationPercentForced
				activeSkill.SkillData.ManaReservationPercentForced = nil
			}
			for name, values := range pool {
				values["more"] = skillModList.More(skillCfg, name+"Reserved", "Reserved")
				values["inc"] = skillModList.Sum(mod.TypeIncrease, skillCfg, name+"Reserved", "Reserved")
				values["efficiency"] = max(skillModList.Sum(mod.TypeIncrease, skillCfg, name+"ReservationEfficiency", "ReservationEfficiency"), -100)
				// used for Arcane Cloak calculations in ModStore.GetStat
				env.Player.Output[name+"Efficiency"] = values["efficiency"]
				if utils.GetOr[*float64](activeSkill.SkillData, name+"ReservationFlatForced", nil) != nil {
					values["reservedFlat"] = *utils.GetOr[*float64](activeSkill.SkillData, name+"ReservationFlatForced", nil)
				} else {
					baseFlatVal := math.Floor(values["baseFlat"] * mult)
					values["reservedFlat"] = 0
					if values["more"] > 0 && values["inc"] > -100 && baseFlatVal != 0 {
						values["reservedFlat"] = max(utils.RoundTo(baseFlatVal*(100+values["inc"])/100*values["more"]/(1+values["efficiency"]/100), 0), 0)
					}
				}
				if utils.GetOr[*float64](activeSkill.SkillData, name+"ReservationPercentForced", nil) != nil {
					values["reservedPercent"] = *utils.GetOr[*float64](activeSkill.SkillData, name+"ReservationPercentForced", nil)
				} else {
					basePercentVal := values["basePercent"] * mult
					values["reservedPercent"] = 0
					if values["more"] > 0 && values["inc"] > -100 && basePercentVal != 0 {
						values["reservedPercent"] = max(utils.RoundTo(basePercentVal*(100+values["inc"])/100*values["more"]/(1+values["efficiency"]/100), 2), 0)
					}
				}
				if activeSkill.ActiveMineCount != 0 {
					values["reservedFlat"] = values["reservedFlat"] * activeSkill.ActiveMineCount
					values["reservedPercent"] = values["reservedPercent"] * activeSkill.ActiveMineCount
				}
				if values["reservedFlat"] != 0 {
					utils.Set(activeSkill.SkillData, name+"ReservedBase", values["reservedFlat"])
					env.Player.Output["reserved_"+name+"Base"] = env.Player.Output["reserved_"+name+"Base"] + values["reservedFlat"]
					if breakdown != nil {
						breakdown.Reservation(name+"Reserved", BReservation{
							SkillName:  activeSkill.ActiveEffect.GrantedEffect.Raw.ID,
							Base:       fmt.Sprint(values["baseFlat"]),
							Mult:       utils.Ternary(mult != 1, utils.Ptr("x "+fmt.Sprint(mult)), nil),
							More:       utils.Ternary(values["more"] != 1, utils.Ptr("x "+fmt.Sprint(values["more"])), nil),
							Inc:        utils.Ternary(values["inc"] != 0, utils.Ptr("x "+fmt.Sprint(1+values["inc"]/100)), nil),
							Efficiency: utils.Ternary(values["efficiency"] != 0, utils.Ptr("x "+fmt.Sprint(1/(1+values["efficiency"]/100))), nil),
							Total:      fmt.Sprint(values["reservedFlat"]),
						})
					}
				}
				if values["reservedPercent"] != 0 {
					utils.Set(activeSkill.SkillData, name+"ReservedPercent", values["reservedPercent"])
					utils.Set(activeSkill.SkillData, name+"ReservedBase", (utils.GetOr(activeSkill.SkillData, name+"ReservedBase", float64(0)))+math.Ceil(env.Player.Output[name]*values["reservedPercent"]/100))
					env.Player.Output["reserved_"+name+"Percent"] = env.Player.Output["reserved_"+name+"Percent"] + values["reservedPercent"]
					if breakdown != nil {
						breakdown.Reservation(name+"Reserved", BReservation{
							SkillName:  activeSkill.ActiveEffect.GrantedEffect.Raw.ID,
							Base:       fmt.Sprint(values["basePercent"]) + "%",
							Mult:       utils.Ternary(mult != 1, utils.Ptr("x "+fmt.Sprint(mult)), nil),
							More:       utils.Ternary(values["more"] != 1, utils.Ptr("x "+fmt.Sprint(values["more"])), nil),
							Inc:        utils.Ternary(values["inc"] != 0, utils.Ptr("x "+fmt.Sprint(1+values["inc"]/100)), nil),
							Efficiency: utils.Ternary(values["efficiency"] != 0, utils.Ptr("x "+fmt.Sprint(1/(1+values["efficiency"]/100))), nil),
							Total:      fmt.Sprint(values["reservedPercent"]) + "%",
						})
					}
				}
			}
		}
	}

	// Set the life/mana reservations
	doActorLifeManaReservation(env.Player)
	if env.Minion != nil {
		doActorLifeManaReservation(env.Minion.Actor)
	}

	// Process attribute requirements
	reqMult := calclib.Mod(env.ModDB, nil, "GlobalAttributeRequirements")
	attrTable := utils.Ternary(env.ModDB.Flag(nil, "OmniscienceRequirements"), []string{"Omni", "Str", "Dex", "Int"}, []string{"Str", "Dex", "Int"})
	for _, attr := range attrTable {
		breakdownAttr := attr
		if env.ModDB.Flag(nil, "OmniscienceRequirements") {
			breakdownAttr = "Omni"
		}
		if breakdown != nil {
			breakdown.AddCol("Req"+attr,
				BCol{Label: attr, Key: "req"},
				BCol{Label: "Source", Key: "source"},
				BCol{Label: "Source Name", Key: "sourceName"},
			)
		}
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
				if breakdown != nil {
					row := map[string]string{
						// TODO Colors.Negative
						"req":    utils.Ternary(req > env.Player.Output[breakdownAttr], "^#DD0022"+fmt.Sprint(req), fmt.Sprint(req)),
						"reqNum": fmt.Sprint(req),
						"source": reqSource.Source,
					}
					if reqSource.Source == "Item" {
						row["sourceName"] = fmt.Sprint(reqSource.SourceItem)
						// TODO Func
						//item := reqSource.SourceItem
						//row.SourceName = colorCodes[item.Rarity] + item.Name
						//row.SourceNameTooltip = function(tooltip)
						//	env.build.itemsTab.AddItemTooltip(tooltip, item, reqSource.sourceSlot)
						//}
					} else if reqSource.Source == "Gem" {
						// TODO reqSource.sourceGem.color
						row["sourceName"] = fmt.Sprintf("%s %d/%d", reqSource.SourceGem.NameSpec, reqSource.SourceGem.Level, reqSource.SourceGem.Quality)
					}
					breakdown.AddRow("Req"+breakdownAttr, row)
				}
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
				if breakdown != nil {
					actor.Output["Req"+breakdownAttr+"String"] = out > (actor.Output[breakdownAttr] or 0) and colorCodes.NEGATIVE+out or out
				}
			*/
		}
	}
	/*
		TODO Breakdown
		if breakdown and breakdown["ReqOmni"] {
			table.sort(breakdown["ReqOmni"].rowList, function(a, b)
				if a.reqNum != b.reqNum {
					return a.reqNum > b.reqNum
				} else if a.source != b.source {
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
			for _, activeSkill := range (env.Player.activeSkillList) {
				if activeSkill.skillTypes[SkillType.Herald] and not heraldList[activeSkill.skillCfg.skillName] {
					heraldList[activeSkill.skillCfg.skillName] = true
					env.ModDB.multipliers["Herald"] = (env.ModDB.multipliers["Herald"] or 0) + 1
					env.ModDB.conditions["AffectedByHerald"] = true
				}
			}
		}
	*/

	/*
		TODO // Calculate number of active auras affecting self
		if env.mode_buffs {
			auraList := { }
			for _, activeSkill := range (env.Player.activeSkillList) {
				if activeSkill.skillTypes[SkillType.Aura] and not activeSkill.skillTypes[SkillType.RemoteMined] and not activeSkill.skillData.auraCannotAffectSelf and not auraList[activeSkill.skillCfg.skillName] {
					auraList[activeSkill.skillCfg.skillName] = true
					env.ModDB.multipliers["AuraAffectingSelf"] = (env.ModDB.multipliers["AuraAffectingSelf"] or 0) + 1
				}
			}
		}
	*/

	/*
		TODO // Deal with Consecrated Ground
		if env.ModDB.Flag(nil, "Condition:OnConsecratedGround") {
			effect := 1 + env.ModDB.Sum(mod.TypeIncrease, nil, "ConsecratedGroundEffect") / 100
			env.ModDB.NewMod("LifeRegenPercent", "BASE", 5 * effect, "Consecrated Ground")
			env.ModDB.NewMod("CurseEffectOnSelf", "INC", -50 * effect, "Consecrated Ground")
		}
	*/

	/*
		TODO // Maximum Mana conversion from Lightning Mastery
		if env.ModDB.Flag(nil, "ManaAppliesToShockEffect") {
			multiplier := (env.ModDB.Max(nil, "ImprovedManaAppliesToShockEffect") or 100) / 100
			for _, value := range (env.ModDB.Tabulate("INC", nil, "Mana")) {
				mod := value.mod
				modifiers := calcLib.getConvertedModTags(mod, multiplier)
				env.ModDB.NewMod("EnemyShockEffect", "INC", math.Floor(mod.value * multiplier), mod.source, mod.flags, mod.keywordFlags, unpack(modifiers))
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
		for _, activeSkill := range (env.Player.activeSkillList) {
			skillModList := activeSkill.skillModList
			skillCfg := activeSkill.skillCfg
			for _, buff := range (activeSkill.buffList) {
				//Skip adding buff if reservation exceeds maximum
				for _, value := range ({"Mana", "Life"}) {
					if activeSkill.skillData[value+"ReservedBase"] and activeSkill.skillData[value+"ReservedBase"] > env.Player.output[value] {
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
							env.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
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
						if env.Minion and (buff.applyMinions or buff.applyAllies) {
							activeSkill.minionBuffSkill = true
							env.Minion.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
							srcList := new("ModList")
							inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnMinion") + env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
							more := modStore:More(skillCfg, "BuffEffect", "BuffEffectOnMinion") * env.Minion.ModDB.More(nil, "BuffEffectOnSelf")
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
						for _, value := range (env.ModDB.List(skillCfg, "ExtraAuraEffect")) {
							add := true
							for _, mod := range (extraAuraModList) {
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
							affectedByAura[env.Player] = true
							if buff.name:sub(1,4) == "Vaal" {
								env.ModDB.conditions["AffectedBy"+buff.name:sub(6):gsub(" ","")] = true
							}
							env.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
							srcList := new("ModList")
							inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
							more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
							mult := (1 + inc / 100) * more
							srcList:ScaleAddList(buff.modList, mult)
							srcList:ScaleAddList(extraAuraModList, mult)
							mergeBuff(srcList, buffs, buff.name)
						}
						if env.Minion and not (env.ModDB.Flag(nil, "SelfAurasCannotAffectAllies") or env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") or env.ModDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) {
							activeSkill.minionBuffSkill = true
							affectedByAura[env.Minion] = true
							env.Minion.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
							srcList := new("ModList")
							inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect") + env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
							more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect") * env.Minion.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
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
						env.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
						srcList := new("ModList")
						mult := 1
						if buff.type == "AuraDebuff" {
							mult = 0
							if not env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") {
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
					if env.mode_effective and (not enemyDB:Flag(nil, "Hexproof") or env.ModDB.Flag(nil, "CursesIgnoreHexproof")) or mark {
						curse := {
							name = buff.name,
							fromPlayer = true,
							priority = determineCursePriority(buff.name, activeSkill),
							isMark = mark,
							ignoreHexLimit = env.ModDB.Flag(activeSkill.skillCfg, "CursesIgnoreHexLimit") and not mark or false,
							socketedCursesHexLimit = env.ModDB.Flag(activeSkill.skillCfg, "SocketedCursesAdditionalLimit")
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
						if not (env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") and activeSkill.skillTypes[SkillType.Aura]) then //If your aura only effect you blasphemy does nothing
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
							buffInc := env.ModDB.Sum(mod.TypeIncrease, skillCfg, "BuffEffectOnSelf")
							buffMore := env.ModDB.More(skillCfg, "BuffEffectOnSelf")
							curse.buffModList:ScaleAddList(temp, (1 + buffInc / 100) * buffMore)
							if env.Minion {
								curse.minionBuffModList = new("ModList")
								buffInc := env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
								buffMore := env.Minion.ModDB.More(nil, "BuffEffectOnSelf")
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
				for _, activeSkill := range (activeSkill.minion.activeSkillList) {
					skillModList := activeSkill.skillModList
					skillCfg := activeSkill.skillCfg
					for _, buff := range (activeSkill.buffList) {
						if buff.type == "Buff" {
							if env.mode_buffs and activeSkill.skillData.enable {
								skillCfg := buff.activeSkillBuff and skillCfg
								modStore := buff.activeSkillBuff and skillModList or castingMinion.modDB
								if buff.applyAllies {
									env.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
									srcList := new("ModList")
									inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect") + env.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
									more := modStore:More(skillCfg, "BuffEffect") * env.ModDB.More(nil, "BuffEffectOnSelf")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, buffs, buff.name)
									mergeBuff(buff.unscalableModList, buffs, buff.name)
								}
								if env.Minion and (env.Minion == castingMinion or buff.applyAllies) {
					 				env.Minion.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
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
								if not env.ModDB.Flag(nil, "AlliesAurasCannotAffectSelf") {
									srcList := new("ModList")
									inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect") + env.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect") * env.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, buffs, buff.name)
								}
								if env.Minion and (env.Minion != activeSkill.minion or not activeSkill.skillData.auraCannotAffectSelf) {
									srcList := new("ModList")
									inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect") + env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect") * env.Minion.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
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
								stackCount = env.ModDB.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.stackVar)
								if buff.stackLimit {
									stackCount = min(stackCount, buff.stackLimit)
								} else if buff.stackLimitVar {
									stackCount = min(stackCount, env.ModDB.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.stackLimitVar))
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
		for _, activeSkill := range (env.Player.activeSkillList) {
			if activeSkill.minion {
				for _, activeMinionSkill := range (activeSkill.minion.activeSkillList) {
					if activeMinionSkill.skillData.enable {
						skillModList := activeMinionSkill.skillModList
						skillCfg := activeMinionSkill.skillCfg
						for _, buff := range (activeMinionSkill.buffList) {
							if buff.type == "Buff" {
								if buff.applyAllies {
									activeMinionSkill.buffSkill = true
									env.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
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
		for dest, modDB := range ({[curses] = modDB, [minionCurses] = env.Minion and env.Minion.ModDB}) {
			for _, value := range (env.ModDB.List(nil, "ExtraCurse")) {
				gemModList := new("ModList")
				grantedEffect := env.data.skills[value.skillId]
				if grantedEffect {
					calcs.mergeSkillInstanceMods(env, gemModList, {
						grantedEffect = grantedEffect,
						level = value.level,
						quality = 0,
					})
					curseModList := { }
					for _, mod := range (gemModList) {
						for _, tag := range (mod) {
							if tag.type == "GlobalEffect" and tag.effectType == "Curse" {
								t_insert(curseModList, mod)
								break
							}
						}
					}
					if value.applyToPlayer {
						// Sources for curses on the player don't usually respect any kind of limit, so there's little point bothering with slots
						if env.ModDB.Sum(mod.TypeBase, nil, "AvoidCurse") < 100 {
							env.ModDB.conditions["Cursed"] = true
							env.ModDB.multipliers["CurseOnSelf"] = (env.ModDB.multipliers["CurseOnSelf"] or 0) + 1
							env.ModDB.conditions["AffectedBy"+grantedEffect.name:gsub(" ","")] = true
							cfg := { skillName = grantedEffect.name }
							inc := env.ModDB.Sum(mod.TypeIncrease, cfg, "CurseEffectOnSelf") + gemModList:Sum(mod.TypeIncrease, nil, "CurseEffectAgainstPlayer")
							more := env.ModDB.More(cfg, "CurseEffectOnSelf") * gemModList:More(nil, "CurseEffectAgainstPlayer")
							env.ModDB.ScaleAddList(curseModList, (1 + inc / 100) * more)
						}
					} else if not enemyDB:Flag(nil, "Hexproof") or env.ModDB.Flag(nil, "CursesIgnoreHexproof") {
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
	*/
	/*
		TODO // Set curse limit
		actor.Output["EnemyCurseLimit"] = env.ModDB.Sum(mod.TypeBase, nil, "EnemyCurseLimit")
		curses.limit = actor.Output["EnemyCurseLimit"]
		// Assign curses to slots
		curseSlots := { }
		env.curseSlots = curseSlots
		// Currently assume only 1 mark is possible
		markSlotted := false
		for _, source := range ({curses, minionCurses}) {
			for _, curse := range (source) {
				// Calculate curses that ignore hex limit after
				if not curse.ignoreHexLimit and not curse.socketedCursesHexLimit {
					local slot
					skipAddingCurse := false
					// Check if we need to disable a certain curse aura.
					for _, activeSkill := range (env.Player.activeSkillList) {
						if (activeSkill.buffList[1] and curse.name == activeSkill.buffList[1].name and activeSkill.skillTypes[SkillType.Aura]) {
							if env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") {
								skipAddingCurse = true
								break
							}
							for _, value := range ({"Mana", "Life"}) {
								if activeSkill.skillData[value+"ReservedBase"] and activeSkill.skillData[value+"ReservedBase"] > env.Player.output[value] {
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

		for _, source := range ({curses, minionCurses}) {
			for _, curse := range (source) {
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
					socketedCursesHexLimitValue := env.ModDB.Sum(mod.TypeBase, nil, "SocketedCursesHexLimitValue")
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
		for name, modList := range (guards) {
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
			env.ModDB.conditions["AffectedByNonVaalGuardSkill"] = true
		}
		for _, guard := range (guardSlots) {
			env.ModDB.conditions["AffectedByGuardSkill"] = true
			env.ModDB.conditions["AffectedBy"+guard.name:gsub(" ","")] = true
			mergeBuff(guard.modList, buffs, guard.name)
		}
	*/

	/*
		TODO // Apply buff/debuff modifiers
		for _, modList := range (buffs) {
			env.ModDB.AddList(modList)
			if not modList.notBuff {
				env.ModDB.multipliers["BuffOnSelf"] = (env.ModDB.multipliers["BuffOnSelf"] or 0) + 1
			}
			if env.Minion {
				for _, value := range (modList:List(env.Player.MainSkill.skillCfg, "MinionModifier")) {
					if not value.type or env.Minion.type == value.type {
						env.Minion.ModDB.AddMod(value.mod)
					}
				}
			}
		}
		if env.Minion {
			for _, modList := range (minionBuffs) {
				env.Minion.ModDB.AddList(modList)
			}
		}
		for _, modList := range (debuffs) {
			enemyDB:AddList(modList)
		}
		env.ModDB.multipliers["CurseOnEnemy"] = #curseSlots
		affectedByCurse := { }
		for _, slot := range (curseSlots) {
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
				env.ModDB.AddList(slot.buffModList)
			}
			if slot.minionBuffModList {
				env.Minion.ModDB.AddList(slot.minionBuffModList)
			}
		}
	*/

	/*
		TODO // Do another pass on the SkillList to catch effects of buffs, if needed
		for _, activeSkill := range (env.Player.activeSkillList) {
			if activeSkill.activeEffect.grantedEffect.name == "Blight" and activeSkill.skillPart == 2 {
				rate := (1 / activeSkill.activeEffect.grantedEffect.castTime) * calcLib.mod(activeSkill.skillModList, activeSkill.skillCfg, "Speed") * calcs.actionSpeedMod(env.Player)
				duration := calcSkillDuration(activeSkill.skillModList, activeSkill.skillCfg, activeSkill.skillData, env, enemyDB)
				maximum := min((math.Floor(rate * duration) - 1), 19)
				activeSkill.skillModList:NewMod("Multiplier:BlightMaxStages", "BASE", maximum, "Base")
				activeSkill.skillModList:NewMod("Multiplier:BlightStageAfterFirst", "BASE", maximum, "Base")
			}
			if activeSkill.activeEffect.grantedEffect.name == "Penance Brand" and activeSkill.skillPart == 2 {
				rate := 1 / (activeSkill.skillData.repeatFrequency / (1 + env.Player.MainSkill.skillModList:Sum(mod.TypeIncrease, env.Player.MainSkill.skillCfg, "Speed", "BrandActivationFrequency") / 100) / activeSkill.skillModList:More(activeSkill.skillCfg, "BrandActivationFrequency"))
				duration := calcSkillDuration(activeSkill.skillModList, activeSkill.skillCfg, activeSkill.skillData, env, enemyDB)
				ticks := min((math.Floor(rate * duration) - 1), 19)
				activeSkill.skillModList:NewMod("Multiplier:PenanceBrandMaxStages", "BASE", ticks, "Base")
				activeSkill.skillModList:NewMod("Multiplier:PenanceBrandStageAfterFirst", "BASE", ticks, "Base")
			}
			if activeSkill.activeEffect.grantedEffect.name == "Scorching Ray" and activeSkill.skillPart == 2 {
				rate := (1 / activeSkill.activeEffect.grantedEffect.castTime) * calcLib.mod(activeSkill.skillModList, activeSkill.skillCfg, "Speed") * calcs.actionSpeedMod(env.Player)
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
		if env.Player.MainSkill.skillData.triggeredByCospris and not env.Player.MainSkill.skillFlags.minion {
			spellCount := {}
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				if skill.skillTypes[SkillType.Melee] and band(skill.skillCfg.flags, bor(ModFlag.Sword, ModFlag.Weapon1H)) > 0 and skill != env.Player.MainSkill {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredByCospris and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot {
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = skill.skillData.cooldown / icdr, next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByCospris = nil
				env.Player.MainSkill.infoMessage = "No Cospri Triggering Skill Found"
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed
				dualWield := false

				sourceAPS, dualWield = calcDualWieldImpact(env, sourceAPS, source.skillData.doubleHitsWhenDualWielding)

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown, dualWield)

				// Account for chance to hit/crit
				sourceCritChance := GlobalCache.cachedData["CACHE"][uuid].CritChance
				trigRate = trigRate * sourceCritChance / 100
				if breakdown != nil {
					breakdown.Speed = {
						fmt.Sprintf("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						fmt.Sprintf("x %.2f%% ^8(%s effective crit chance)", sourceCritChance, source.activeEffect.grantedEffect.name),
						fmt.Sprintf("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = trigRate
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "Cospri Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.Player.MainSkill.infoTrigger = "Cospri"
			}
		}
	*/
	/*
		TODO // Mjolner
		if env.Player.MainSkill.skillData.triggeredByMjolner and not env.Player.MainSkill.skillFlags.minion {
			spellCount := {}
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				if (skill.skillTypes[SkillType.Damage] or skill.skillTypes[SkillType.Attack]) and band(skill.skillCfg.flags, bor(ModFlag.Mace, ModFlag.Weapon1H)) > 0 and skill != env.Player.MainSkill {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredByMjolner and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot {
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = skill.skillData.cooldown / icdr, next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByMjolner = nil
				env.Player.MainSkill.infoMessage = "No Mjolner Triggering Skill Found"
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed
				dualWield := false

				sourceAPS, dualWield = calcDualWieldImpact(env, sourceAPS, source.skillData.doubleHitsWhenDualWielding)

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown, dualWield)

				// Account for chance to hit/crit
				sourceHitChance := GlobalCache.cachedData["CACHE"][uuid].HitChance
				trigRate = trigRate * sourceHitChance / 100
				if breakdown != nil {
					breakdown.Speed = {
						fmt.Sprintf("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						fmt.Sprintf("x %.0f%% ^8(%s hit chance)", sourceHitChance, source.activeEffect.grantedEffect.name),
						fmt.Sprintf("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = trigRate
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "Mjolner Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.Player.MainSkill.infoTrigger = "Mjolner"
			}
		}
	*/
	/*
		TODO // Mirage Archer Support
		// This creates and populates env.Player.MainSkill.mirage table
		if env.Player.MainSkill.skillData.triggeredByMirageArcher and not env.Player.MainSkill.skillFlags.minion and not env.Player.MainSkill.marked {
			usedSkill := nil
			uuid := cacheSkillUUID(env.Player.MainSkill)
			calcMode := env.mode == "CALCS" and "CALCS" or "MAIN"

			// cache a new copy of this skill that's affected by Mirage Archer
			if avoidCache {
				usedSkill = env.Player.MainSkill
				env.dontCache = true
			} else {
				if not GlobalCache.cachedData[calcMode][uuid] {
					calcs.buildActiveSkill(env, calcMode, env.Player.MainSkill, true)
				}

				if GlobalCache.cachedData[calcMode][uuid] and not avoidCache {
					usedSkill = GlobalCache.cachedData[calcMode][uuid].ActiveSkill
				}
			}

			if usedSkill {
				moreDamage :=  usedSkill.skillModList:Sum(mod.TypeBase, usedSkill.skillCfg, "MirageArcherLessDamage")
				moreAttackSpeed := usedSkill.skillModList:Sum(mod.TypeBase, usedSkill.skillCfg, "MirageArcherLessAttackSpeed")
				mirageCount :=  usedSkill.skillModList:Sum(mod.TypeBase, env.Player.MainSkill.skillCfg, "MirageArcherMaxCount")

				// Make a copy of this skill so we can add new modifiers to the copy affected by Mirage Archers
				newSkill, newEnv := calcs.copyActiveSkill(env, calcMode, usedSkill)

				// Add new modifiers to new skill (which already has all the old skill's modifiers)
				newSkill.skillModList:NewMod("Damage", "MORE", moreDamage, "Mirage Archer", env.Player.MainSkill.ModFlags, env.Player.MainSkill.KeywordFlags)
				newSkill.skillModList:NewMod("Speed", "MORE", moreAttackSpeed, "Mirage Archer", env.Player.MainSkill.ModFlags, env.Player.MainSkill.KeywordFlags)

				env.Player.MainSkill.mirage = { }
				env.Player.MainSkill.mirage.count = mirageCount
				env.Player.MainSkill.mirage.name = usedSkill.activeEffect.grantedEffect.name

				if usedSkill.skillPartName {
					env.Player.MainSkill.mirage.skillPart = usedSkill.skillPart
					env.Player.MainSkill.mirage.skillPartName = usedSkill.skillPartName
					env.Player.MainSkill.mirage.infoMessage2 = usedSkill.activeEffect.grantedEffect.name
				} else {
					env.Player.MainSkill.mirage.skillPartName = nil
				}
				env.Player.MainSkill.mirage.infoTrigger = "MA"

				// Recalculate the offensive/defensive aspects of the Mirage Archer influence on skill
				newEnv.player.mainSkill = newSkill
				// mark it so we don't recurse infinitely
				newSkill.marked = true
				newEnv.dontCache = true
				calcs.perform(newEnv)

				env.Player.MainSkill.infoMessage = tostring(mirageCount) + " Mirage Archers using " + usedSkill.activeEffect.grantedEffect.name

				// Re-link over the output
				env.Player.MainSkill.mirage.output = newEnv.player.output

				if newSkill.minion {
					env.Player.MainSkill.mirage.minion = {}
					env.Player.MainSkill.mirage.minion.output = newEnv.minion.output
				}

				// Make any necessary corrections to output
				env.Player.MainSkill.mirage.output.ManaCost = 0

				if newEnv.player.breakdown {
					env.Player.MainSkill.mirage.breakdown = newEnv.player.breakdown
					// Make any necessary corrections to breakdown
					env.Player.MainSkill.mirage.breakdown.ManaCost = nil
					if newSkill.minion {
						env.Player.MainSkill.mirage.minion.breakdown = newEnv.minion.breakdown
					}
				}
			} else {
				env.Player.MainSkill.infoMessage2 = "No Mirage Archer active skill found"
			}
		}
	*/
	/*
		TODO // Kitava's Thirst
		if env.Player.MainSkill.skillData.triggeredByManaSpent and not env.Player.MainSkill.skillFlags.minion {
			triggerName := "Kitava"
			spellCount := 0
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "CooldownRecovery")
			reqManaCost := env.Player.modDB.Sum(mod.TypeBase, nil, "KitavaRequiredManaCost")
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				if not skill.skillTypes[SkillType.Triggered] and skill != env.Player.MainSkill and not skill.skillData.triggeredByManaSpent {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate, reqManaCost)
				}
				if skill.skillData.triggeredByManaSpent and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot {
					spellCount = spellCount + 1
				}
			}

			if not source or spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByManaSpent = nil
				env.Player.MainSkill.infoMessage = fmt.Sprintf("No %s Triggering Skill Found", triggerName)
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true

				actor.Output["ActionTriggerRate"] = getTriggerActionTriggerRate(env.Player.MainSkill.skillData.cooldown, env, breakdown)

				// Get action trigger rate
				kitavaCD := getTriggerDefaultCooldown(env.Player.MainSkill.supportList, "SupportCastOnManaSpent")

				trigRate = icdr / kitavaCD
				actor.Output["SourceTriggerRate"] = trigRate
				actor.Output["ServerTriggerRate"] = min(actor.Output["SourceTriggerRate"], actor.Output["ActionTriggerRate"])
				if breakdown != nil {
					modActionCooldown := kitavaCD / icdr
					rateCapAdjusted := m_ceil(modActionCooldown * data.misc.ServerTickRate) / data.misc.ServerTickRate
					extraICDRNeeded := m_ceil((modActionCooldown - rateCapAdjusted + data.misc.ServerTickTime) * icdr * 1000)
					breakdown.SimData = {
						fmt.Sprintf("%.2f ^8(base cooldown of kitava's trigger)", kitavaCD),
						fmt.Sprintf("/ %.2f ^8(increased/reduced cooldown recovery)", icdr),
						fmt.Sprintf("= %.4f ^8(final cooldown of trigger)", modActionCooldown),
						fmt.Sprintf(""),
						fmt.Sprintf("%.3f ^8(adjusted for server tick rate)", rateCapAdjusted),
						fmt.Sprintf("^8(extra ICDR of %d%% would reach next breakpoint)", extraICDRNeeded),
						fmt.Sprintf(""),
						fmt.Sprintf("Trigger rate:"),
						fmt.Sprintf("1 / %.3f", rateCapAdjusted),
						fmt.Sprintf("= %.2f ^8per second", 1 / rateCapAdjusted),
					}
					breakdown.ServerTriggerRate = {
						fmt.Sprintf("%.2f ^8(smaller of 'cap' and 'skill' trigger rates)", actor.Output["ServerTriggerRate"]),
					}
				}

				// Account for chance to trigger
				kitavaTriggerChance := env.Player.modDB.Sum(mod.TypeBase, nil, "KitavaTriggerChance")
				trigRate = actor.Output["ServerTriggerRate"] * kitavaTriggerChance / 100
				if breakdown != nil {
					breakdown.Speed = {
						fmt.Sprintf("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						fmt.Sprintf("x %.2f%% ^8(kitava's trigger chance)", kitavaTriggerChance),
						fmt.Sprintf("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = trigRate
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "Kitava's Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.Player.MainSkill.infoTrigger = triggerName
			}
		}
	*/
	/*
		TODO // Crafted Trigger
		if env.Player.MainSkill.skillData.triggeredByCraft and not env.Player.MainSkill.skillFlags.minion {
			triggerName := "Crafted"
			spellCount := 0
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				if (skill.skillTypes[SkillType.Damage] or skill.skillTypes[SkillType.Attack] or skill.skillTypes[SkillType.Spell]) and skill != env.Player.MainSkill and not skill.skillData.triggeredByCraft {
					source, trigRate = skill, 0
				}
				if skill.skillData.triggeredByCraft and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot {
					spellCount = spellCount + 1
				}
				// we just need one source and one linked spell
				if source and spellCount > 0 {
					break
				}
			}
			if not source or spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByCraft = nil
				env.Player.MainSkill.infoMessage = fmt.Sprintf("No %s Triggering Skill Found", triggerName)
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true

				actor.Output["ActionTriggerRate"] = getTriggerActionTriggerRate(env.Player.MainSkill.skillData.cooldown, env, breakdown)

				// Get action trigger rate
				craftedCD := getTriggerDefaultCooldown(env.Player.MainSkill.supportList, "SupportTriggerSpellOnSkillUse")

				trigRate = icdr / craftedCD
				actor.Output["SourceTriggerRate"] = trigRate
				actor.Output["ServerTriggerRate"] = min(actor.Output["SourceTriggerRate"], actor.Output["ActionTriggerRate"])
				if breakdown != nil {
					modActionCooldown := craftedCD / icdr
					rateCapAdjusted := m_ceil(modActionCooldown * data.misc.ServerTickRate) / data.misc.ServerTickRate
					extraICDRNeeded := m_ceil((modActionCooldown - rateCapAdjusted + data.misc.ServerTickTime) * icdr * 1000)
					breakdown.SimData = {
						fmt.Sprintf("%.2f ^8(base cooldown of crafted trigger)", craftedCD),
						fmt.Sprintf("/ %.2f ^8(increased/reduced cooldown recovery)", icdr),
						fmt.Sprintf("= %.4f ^8(final cooldown of trigger)", modActionCooldown),
						fmt.Sprintf(""),
						fmt.Sprintf("%.3f ^8(adjusted for server tick rate)", rateCapAdjusted),
						fmt.Sprintf("^8(extra ICDR of %d%% would reach next breakpoint)", extraICDRNeeded),
						fmt.Sprintf(""),
						fmt.Sprintf("Trigger rate:"),
						fmt.Sprintf("1 / %.3f", rateCapAdjusted),
						fmt.Sprintf("= %.2f ^8per second", 1 / rateCapAdjusted),
					}
					breakdown.ServerTriggerRate = {
						fmt.Sprintf("%.2f ^8(smaller of 'cap' and 'skill' trigger rates)", actor.Output["ServerTriggerRate"]),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = actor.Output["ServerTriggerRate"]
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "Weapon-Crafted Triggering Skill Found"
				env.Player.MainSkill.infoTrigger = triggerName
				env.Player.MainSkill.skillFlags.dontDisplay = true
			}
		}
	*/
	/*
		TODO // Helmet Focus Trigger
		if env.Player.MainSkill.skillData.triggeredByFocus and not env.Player.MainSkill.skillFlags.minion {
			triggerName := "Focus"
			spellCount := 0
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "FocusCooldownRecovery")
			trigRate := 0
			source := env.Player.modDB.Flag(nil, "Condition:Focused")
			for _, skill := range (env.Player.activeSkillList) {
				if skill.skillData.triggeredByFocus and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot {
					spellCount = spellCount + 1
				}
			}
			if not source or spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByFocus = nil
				env.Player.MainSkill.infoMessage = fmt.Sprintf("No %s Triggering Skill Found", triggerName)
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true

				actor.Output["ActionTriggerRate"] = getTriggerActionTriggerRate(env.Player.MainSkill.skillData.cooldown, env, breakdown, true)

				// Get action trigger rate
				skillFocus := env.data.skills["Focus"]
				focusCD := skillFocus.levels[1].cooldown

				trigRate = icdr / focusCD
				actor.Output["SourceTriggerRate"] = trigRate
				actor.Output["ServerTriggerRate"] = min(actor.Output["SourceTriggerRate"], actor.Output["ActionTriggerRate"])
				if breakdown != nil {
					modActionCooldown := focusCD / icdr
					rateCapAdjusted := m_ceil(modActionCooldown * data.misc.ServerTickRate) / data.misc.ServerTickRate
					breakdown.SimData = {
						fmt.Sprintf("%.2f ^8(base cooldown of focus trigger)", focusCD),
						fmt.Sprintf("/ %.2f ^8(increased/reduced cooldown recovery)", icdr),
						fmt.Sprintf("= %.4f ^8(final cooldown of trigger)", modActionCooldown),
						fmt.Sprintf(""),
						fmt.Sprintf("%.3f ^8(adjusted for server tick rate)", rateCapAdjusted),
						fmt.Sprintf(""),
						fmt.Sprintf("Trigger rate:"),
						fmt.Sprintf("1 / %.3f", rateCapAdjusted),
						fmt.Sprintf("= %.2f ^8per second", 1 / rateCapAdjusted),
					}
					breakdown.ServerTriggerRate = {
						fmt.Sprintf("%.2f ^8(smaller of 'cap' and 'skill' trigger rates)", actor.Output["ServerTriggerRate"]),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = actor.Output["ServerTriggerRate"]
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "Focus Triggering Skill Found"
				env.Player.MainSkill.infoTrigger = triggerName
				env.Player.MainSkill.skillFlags.dontDisplay = true
			}
		}
	*/
	/*
		TODO // Unique Item Trigger
		if env.Player.MainSkill.skillData.triggeredByUnique and not env.Player.MainSkill.skillFlags.minion {
			uniqueTriggerName := getUniqueItemTriggerName(env.Player.MainSkill)
			triggerName := ""
			spellCount := {}
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				cooldownOverride := skill.skillModList:Override(env.Player.MainSkill.skillCfg, "CooldownRecovery")
				if uniqueTriggerName == "Poet's Pen" {
					triggerName = "Poet"
					if (skill.skillTypes[SkillType.Damage] or skill.skillTypes[SkillType.Attack]) and band(skill.skillCfg.flags, ModFlag.Wand) > 0 and skill != env.Player.MainSkill and not skill.skillData.triggeredByUnique {
						source, trigRate = findTriggerSkill(env, skill, source, trigRate)
					}
					if skill.skillData.triggeredByUnique and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot and skill.skillTypes[SkillType.Spell] {
						t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
					}
				} else if uniqueTriggerName == "Maloney's Mechanism" {
					triggerName = "Maloney"
					if skill.skillTypes[SkillType.Attack] and band(skill.skillCfg.flags, ModFlag.Bow) > 0 and skill != env.Player.MainSkill and not skill.skillData.triggeredByUnique {
						source, trigRate = findTriggerSkill(env, skill, source, trigRate)
					}
					if skill.skillData.triggeredByUnique and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot and skill.skillTypes[SkillType.RangedAttack] {
						t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
					}
				} else if uniqueTriggerName == "Asenath's Chant" {
					triggerName = "Asenath"
					if (skill.skillTypes[SkillType.Damage] or skill.skillTypes[SkillType.Attack]) and band(skill.skillCfg.flags, ModFlag.Bow) > 0 and skill != env.Player.MainSkill and not skill.skillData.triggeredByUnique {
						source, trigRate = findTriggerSkill(env, skill, source, trigRate)
					}
					if skill.skillData.triggeredByUnique and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot and skill.skillTypes[SkillType.Spell] {
						t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
					}
				} else if uniqueTriggerName == "Queen's Demand" {
					triggerName = "QD"
					if skill.activeEffect.grantedEffect.name == uniqueTriggerName {
						source, trigRate = findTriggerSkill(env, skill, source, trigRate)
					}
					if skill.skillData.triggeredByUnique and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot {
						t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
					}
				} else {
					ConPrintf("[ERROR]: Unhandled Unique Trigger Name: " + uniqueTriggerName)
				}
			}
			if not source or #spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByUnique = nil
				env.Player.MainSkill.infoMessage = fmt.Sprintf("No %s Triggering Skill Found", triggerName)
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed
				dualWield := false

				sourceAPS, dualWield = calcDualWieldImpact(env, sourceAPS, source.skillData.doubleHitsWhenDualWielding)

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown, dualWield)

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)

				env.Player.MainSkill.skillData.triggerRate = trigRate
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.skillData.triggerSourceUUID = cacheSkillUUID(source, env.mode)
				env.Player.MainSkill.skillData.triggerUnleash = source.skillModList:Flag(nil, "HasSeals") and source.skillTypes[SkillType.CanRapidFire]
				env.Player.MainSkill.infoMessage = env.Player.MainSkill.activeEffect.grantedEffect.name + "'s Trigger: " + source.activeEffect.grantedEffect.name
				env.Player.MainSkill.infoTrigger = env.Player.MainSkill.infoTrigger or triggerName
			}
		}
	*/
	/*
		TODO // Cast On Critical Strike Support (CoC)
		if env.Player.MainSkill.skillData.triggeredByCoC and not env.Player.MainSkill.skillFlags.minion {
			spellCount := {}
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				match1 := env.Player.MainSkill.activeEffect.grantedEffect.fromItem and skill.socketGroup.slot == env.Player.MainSkill.socketGroup.slot
				match2 := (not env.Player.MainSkill.activeEffect.grantedEffect.fromItem) and skill.socketGroup == env.Player.MainSkill.socketGroup
				if skill.skillTypes[SkillType.Attack] and skill != env.Player.MainSkill and (match1 or match2) {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredByCoC and (match1 or match2) {
					cooldownOverride := skill.skillModList:Override(env.Player.MainSkill.skillCfg, "CooldownRecovery")
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByCoC = nil
				env.Player.MainSkill.infoMessage = "No CoC Triggering Skill Found"
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown)

				// Account for chance to hit/crit
				sourceCritChance := GlobalCache.cachedData["CACHE"][uuid].CritChance
				trigRate = trigRate * sourceCritChance / 100
				trigRate = trigRate * (source.skillData.chanceToTriggerOnCrit or 100) / 100
				if breakdown != nil {
					breakdown.Speed = {
						fmt.Sprintf("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						fmt.Sprintf("x %.2f%% ^8(%s crit chance)", sourceCritChance, source.activeEffect.grantedEffect.name),
						fmt.Sprintf("x %.2f%% ^8(chance to trigger on crit)", source.skillData.chanceToTriggerOnCrit or 100),
						fmt.Sprintf("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = trigRate
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "CoC Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.Player.MainSkill.infoTrigger = "CoC"
			}
		}
	*/
	/*
		TODO // Cast On Melee Kill Support (CoMK)
		if env.Player.MainSkill.skillData.triggeredByMeleeKill and not env.Player.MainSkill.skillFlags.minion and env.ModDB.Flag(nil, "Condition:KilledRecently") {
			spellCount := {}
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				match1 := env.Player.MainSkill.activeEffect.grantedEffect.fromItem and skill.socketGroup.slot == env.Player.MainSkill.socketGroup.slot
				match2 := (not env.Player.MainSkill.activeEffect.grantedEffect.fromItem) and skill.socketGroup == env.Player.MainSkill.socketGroup
				if skill.skillTypes[SkillType.Attack] and skill.skillTypes[SkillType.Melee] and skill != env.Player.MainSkill and (match1 or match2) {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredByMeleeKill and (match1 or match2) {
					cooldownOverride := skill.skillModList:Override(env.Player.MainSkill.skillCfg, "CooldownRecovery")
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByMeleeKill = nil
				env.Player.MainSkill.infoMessage = "No CoMK Triggering Skill Found"
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown)

				// Account for chance to trigger on Melee Kill
				trigRate = trigRate * source.skillData.chanceToTriggerOnMeleeKill / 100

				if breakdown != nil {
					breakdown.Speed = {
						fmt.Sprintf("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						fmt.Sprintf("x %.2f%% ^8(chance to trigger on melee kill)", source.skillData.chanceToTriggerOnMeleeKill),
						fmt.Sprintf("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = trigRate
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "CoMK Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.Player.MainSkill.infoTrigger = "CoMK"
			}
		}
	*/
	/*
		TODO // Cast While Channelling
		if env.Player.MainSkill.skillData.triggeredWhileChannelling and not env.Player.MainSkill.skillFlags.minion {
			spellCount := {}
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				match1 := env.Player.MainSkill.activeEffect.grantedEffect.fromItem and skill.socketGroup.slot == env.Player.MainSkill.socketGroup.slot
				match2 := (not env.Player.MainSkill.activeEffect.grantedEffect.fromItem) and skill.socketGroup == env.Player.MainSkill.socketGroup
				if skill.skillTypes[SkillType.Channel] and skill != env.Player.MainSkill and (match1 or match2) {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredWhileChannelling and (match1 or match2) {
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = skill.skillData.cooldown, next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredWhileChannelling = nil
				env.Player.MainSkill.infoMessage = "No CwC Triggering Skill Found"
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, nil, spellCount, output, breakdown)

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = trigRate
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "CwC Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.Player.MainSkill.infoTrigger = "CwC"

				env.Player.MainSkill.skillFlags.dontDisplay = true
			}
		}
	*/
	/*
		TODO // Triggered by parent attack
		if env.Minion and env.Player.MainSkill.minion {
			if env.Minion.mainSkill.skillData.triggeredByParentAttack {
				spellCount := {}
				trigRate := 0
				source := nil
				for _, skill := range (env.Player.activeSkillList) {
					if skill.skillTypes[SkillType.Attack] and skill != env.Player.MainSkill {
						source, trigRate = findTriggerSkill(env, skill, source, trigRate)
					}
				}

				icdr := calcLib.mod(env.Minion.mainSkill.skillModList, env.Minion.mainSkill.skillCfg, "CooldownRecovery")
				t_insert(spellCount, { uuid = cacheSkillUUID(env.Minion.mainSkill), cd = env.Minion.mainSkill.skillData.cooldown / icdr, next_trig = 0, count = 0 })

				if not source {
					env.Minion.mainSkill.skillData.triggeredByParentAttack = nil
					env.Minion.mainSkill.infoMessage = "No triggering Skill Found"
					env.Minion.mainSkill.infoMessage2 = "DPS reported assuming regular cast"
					env.Minion.mainSkill.infoTrigger = ""
				} else {
					env.Minion.mainSkill.skillData.triggered = true
					uuid := cacheSkillUUID(source)

					sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed

					// Get action trigger rate
					trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, env.Minion.output, env.Minion.breakdown, false, true)

					// Account for chance to hit
					sourceHitChance := GlobalCache.cachedData["CACHE"][uuid].HitChance
					trigRate = trigRate * sourceHitChance / 100
					if env.Minion.breakdown {
						env.Minion.breakdown.Speed = {
							fmt.Sprintf("%.2fs ^8(adjusted trigger rate)", env.Minion.output.ServerTriggerRate),
							fmt.Sprintf("x %.2f%% ^8(%s Hit chance)", sourceHitChance, source.activeEffect.grantedEffect.name),
							fmt.Sprintf("= %.2f ^8per second", trigRate),
						}
					}

					// Account for Trigger-related INC/MORE modifiers
					addTriggerIncMoreMods(env.Minion.mainSkill, env.Minion.mainSkill)
					env.Minion.mainSkill.skillData.triggerRate = trigRate
					env.Minion.mainSkill.skillData.triggerSource = source
					env.Minion.mainSkill.infoMessage = "Triggering Skill: " + source.activeEffect.grantedEffect.name
					env.Minion.mainSkill.infoTrigger = "Parent attack"
				}
			}
		}
	*/

	/*
		TODO // Fix the configured impale stacks on the enemy
		// 		If the config is missing (blank), then use the maximum number of stacks
		//		If the config is larger than the maximum number of stacks, replace it with the correct maximum
		maxImpaleStacks := env.ModDB.Sum(mod.TypeBase, nil, "ImpaleStacksMax")
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

		for ailment, val := range (ailments) {
			if (enemyDB:Sum(mod.TypeBase, nil, ailment+"Val") > 0
			or env.ModDB.Sum(mod.TypeBase, nil, ailment+"Base", ailment+"Override")
			or (ailment == "Chill" and actor.Output["BonechillEffect"]))
			and not enemyDB:Flag(nil, "Condition:Already"+val.condition) {
				override := 0
				for _, value := range (env.ModDB.Tabulate("BASE", nil, ailment+"Base", ailment+"Override")) {
					mod := value.mod
					effect := mod.value
					if mod.name == ailment+"Override" {
						enemyDB:NewMod("Condition:"+val.condition, "FLAG", true, mod.source)
					}
					if mod.name == ailment+"Base" {
						effect = effect * calcLib.mod(modDB, nil, "Enemy"+ailment+"Effect")
						env.ModDB.NewMod(ailment+"Override", "BASE", effect, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
					}
					override = max(override, effect or 0)
				}
				actor.Output["Maximum"+ailment] = env.ModDB.Override(nil, ailment+"Max") or ailmentData[ailment].max
				actor.Output["Current"+ailment] = math.Floor(min(max(override, enemyDB:Sum(mod.TypeBase, nil, ailment+"Val"), ailment == "Chill" and actor.Output["BonechillEffect"] or 0), actor.Output["Maximum"+ailment]) * (10 ^ ailmentData[ailment].precision)) / (10 ^ ailmentData[ailment].precision)
				for _, mod := range (val.mods(actor.Output["Current"+ailment])) {
					enemyDB:AddMod(mod)
				}
				enemyDB:NewMod("Condition:Already"+val.condition, "FLAG", true, { type = "Condition", var = val.condition } ) // Prevents ailment from applying doubly for minions
			}
		}
	*/

	/*
		TODO // Check for extra auras
		for _, value := range (env.ModDB.List(nil, "ExtraAura")) {
			modList := { value.mod }
			if not value.onlyAllies {
				inc := env.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				more := env.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				env.ModDB.ScaleAddList(modList, (1 + inc / 100) * more)
				if not value.notBuff {
					env.ModDB.multipliers["BuffOnSelf"] = (env.ModDB.multipliers["BuffOnSelf"] or 0) + 1
				}
			}
			if env.Minion and not env.ModDB.Flag(nil, "SelfAurasCannotAffectAllies") {
				inc := env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				more := env.Minion.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				env.Minion.ModDB.ScaleAddList(modList, (1 + inc / 100) * more)
			}
		}
	*/

	/*
		TODO // Check for modifiers to apply to actors affected by player auras or curses
		for _, value := range (env.ModDB.List(nil, "AffectedByAuraMod")) {
			for actor := range (affectedByAura) {
				actor.modDB.AddMod(value.mod)
			}
		}
		for _, value := range (env.ModDB.List(nil, "AffectedByCurseMod")) {
			for actor := range (affectedByCurse) {
				actor.modDB.AddMod(value.mod)
			}
		}
	*/

	// Merge keystones again to catch any that were added by buffs
	mergeKeystones(env)

	/*
		TODO // Special handling for Dancing Dervish
		if env.ModDB.Flag(nil, "DisableWeapons") {
			env.Player.weaponData1 = copyTable(env.data.unarmedWeaponData[env.classId])
			env.ModDB.conditions["Unarmed"] = true
			if not env.Player.Gloves or env.Player.Gloves == None {
				env.ModDB.conditions["Unencumbered"] = true
			}
		} else if env.weaponModList1 {
			env.ModDB.AddList(env.weaponModList1)
		}
	*/

	// Process misc buffs/modifiers
	DoActorMisc(env, env.Player)
	if env.Minion != nil {
		// TODO doActorMisc(env, env.Minion)
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

	// Apply exposures
	for _, element := range []string{"Fire", "Cold", "Lightning"} {
		if !env.Player.ModDB.Flag(nil, "ElementalEquilibrium") || // if Elemental Equilibrium isn't active we just process Exposure normally
			element == "Fire" && !env.EnemyModDB.Flag(nil, "Condition:HitByFireDamage") ||
			element == "Cold" && !env.EnemyModDB.Flag(nil, "Condition:HitByColdDamage") ||
			element == "Lightning" && !env.EnemyModDB.Flag(nil, "Condition:HitByLightningDamage") {
			Min := math.MaxFloat64
			source := mod.Source("")
			for _, Mod := range env.EnemyModDB.Tabulate("BASE", nil, element+"Exposure") {
				if Mod.Value < Min {
					Min = Mod.Value
					source = Mod.Mod.GetSource()
				}
			}
			if Min != math.MaxFloat64 {
				// Modify the magnitude of all exposures
				for _, Mod := range env.Player.ModDB.Tabulate("BASE", nil, "ExtraExposure", "Extra"+element+"Exposure") {
					Min = Min + Mod.Value
				}
				env.EnemyModDB.AddMod(mod.NewFloat(element+"Resist", "BASE", min(Min, utils.Or(env.Player.ModDB.Override(nil, "ExposureMin"), 0))).Source(source))
				env.Player.ModDB.AddMod(mod.NewFlag("Condition:AppliedExposureRecently", true))
			}
		}
	}

	// Handle consecrated ground effects on enemies
	if env.EnemyModDB.Flag(nil, "Condition:OnConsecratedGround") {
		effect := 1 + env.Player.ModDB.Sum(mod.TypeIncrease, nil, "ConsecratedGroundEffect")/100
		env.EnemyModDB.AddMod(mod.NewFloat("DamageTaken", "INC", env.EnemyModDB.Sum(mod.TypeIncrease, nil, "DamageTakenConsecratedGround")*effect).Source("Consecrated Ground"))
	}

	// Defence/offence calculations
	CalculateDefence(env, env.Player)
	CalculateOffence(env, env.Player, env.Player.MainSkill)

	// Minion Defence/offence calculations
	if env.Minion != nil {
		CalculateDefence(env, env.Minion.Actor)
		CalculateOffence(env, env.Minion.Actor, env.Minion.MainSkill)
	}

	/*
		TODO Cache Data
		uuid := cacheSkillUUID(env.Player.MainSkill)
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
	// Set conditions
	if (actor.ItemList["Weapon 2"] != nil && actor.ItemList["Weapon 2"].Type == "Shield") || (actor == env.Player && env.AegisModList != nil) {
		actor.ModDB.Conditions["UsingShield"] = true
	}
	if actor.ItemList["Weapon 2"] == nil {
		actor.ModDB.Conditions["OffHandIsEmpty"] = true
	}
	if actor.WeaponData1.Type == "None" {
		actor.ModDB.Conditions["Unarmed"] = true
		if actor.ItemList["Weapon 2"] == nil && actor.ItemList["Gloves"] == nil {
			actor.ModDB.Conditions["Unencumbered"] = true
		}
	} else {
		info := data.WeaponTypes[actor.WeaponData1.Type]
		actor.ModDB.Conditions["Using"+info.Flag] = true
		if actor.WeaponData1.CountsAsAll1H {
			actor.ModDB.Conditions["UsingAxe"] = true
			actor.ModDB.Conditions["UsingSword"] = true
			actor.ModDB.Conditions["UsingDagger"] = true
			actor.ModDB.Conditions["UsingMace"] = true
			actor.ModDB.Conditions["UsingClaw"] = true
			// GGG stated that a single Varunastra satisfied requirement for wielding two different weapons
			actor.ModDB.Conditions["WieldingDifferentWeaponTypes"] = true
		}
		if info.Melee {
			actor.ModDB.Conditions["UsingMeleeWeapon"] = true
		}
		if info.OneHand {
			actor.ModDB.Conditions["UsingOneHandedWeapon"] = true
		} else {
			actor.ModDB.Conditions["UsingTwoHandedWeapon"] = true
		}
	}
	if actor.WeaponData2.Type != "" {
		info := data.WeaponTypes[actor.WeaponData2.Type]
		actor.ModDB.Conditions["Using"+info.Flag] = true
		if actor.WeaponData2.CountsAsAll1H {
			actor.ModDB.Conditions["UsingAxe"] = true
			actor.ModDB.Conditions["UsingSword"] = true
			actor.ModDB.Conditions["UsingDagger"] = true
			actor.ModDB.Conditions["UsingMace"] = true
			actor.ModDB.Conditions["UsingClaw"] = true
			// GGG stated that a single Varunastra satisfied requirement for wielding two different weapons
			actor.ModDB.Conditions["WieldingDifferentWeaponTypes"] = true
		}
		if info.Melee {
			actor.ModDB.Conditions["UsingMeleeWeapon"] = true
		}
		if info.OneHand {
			actor.ModDB.Conditions["UsingOneHandedWeapon"] = true
		} else {
			actor.ModDB.Conditions["UsingTwoHandedWeapon"] = true
		}
	}
	if actor.WeaponData1.Type != "" && actor.WeaponData2.Type != "" {
		actor.ModDB.Conditions["DualWielding"] = true
		if (actor.WeaponData1.Type == "Claw" || actor.WeaponData1.CountsAsAll1H) && (actor.WeaponData2.Type == "Claw" || actor.WeaponData2.CountsAsAll1H) {
			actor.ModDB.Conditions["DualWieldingClaws"] = true
		}
		if (actor.WeaponData1.Type == "Dagger" || actor.WeaponData1.CountsAsAll1H) && (actor.WeaponData2.Type == "Dagger" || actor.WeaponData2.CountsAsAll1H) {
			actor.ModDB.Conditions["DualWieldingDaggers"] = true
		}
		if utils.OrS(data.WeaponTypes[actor.WeaponData1.Type].Label, string(actor.WeaponData1.Type)) != utils.OrS(data.WeaponTypes[actor.WeaponData2.Type].Label, string(actor.WeaponData2.Type)) {
			info1 := data.WeaponTypes[actor.WeaponData1.Type]
			info2 := data.WeaponTypes[actor.WeaponData2.Type]
			if info1.OneHand && info2.OneHand {
				actor.ModDB.Conditions["WieldingDifferentWeaponTypes"] = true
			}
		}
	}
	if env.ModeCombat {
		if !actor.ModDB.Flag(nil, "NeverCrit") {
			actor.ModDB.Conditions["CritInPast8Sec"] = true
		}
		if !actor.MainSkill.SkillData.Triggered && !actor.MainSkill.SkillFlags[SkillFlagTrap] && !actor.MainSkill.SkillFlags[SkillFlagMine] && !actor.MainSkill.SkillFlags[SkillFlagTotem] {
			if actor.MainSkill.SkillFlags[SkillFlagAttack] {
				actor.ModDB.Conditions["AttackedRecently"] = true
			} else if actor.MainSkill.SkillFlags[SkillFlagSpell] {
				actor.ModDB.Conditions["CastSpellRecently"] = true
			}
			if actor.MainSkill.SkillTypes[data.SkillTypeMovement] {
				actor.ModDB.Conditions["UsedMovementSkillRecently"] = true
			}
			if actor.MainSkill.SkillFlags[SkillFlagMinion] {
				actor.ModDB.Conditions["UsedMinionSkillRecently"] = true
			}
			if actor.MainSkill.SkillTypes[data.SkillTypeVaal] {
				actor.ModDB.Conditions["UsedVaalSkillRecently"] = true
			}
			if actor.MainSkill.SkillTypes[data.SkillTypeChannel] {
				actor.ModDB.Conditions["Channelling"] = true
			}
		}
		if actor.MainSkill.SkillFlags[SkillFlagHit] && !actor.MainSkill.SkillFlags[SkillFlagTrap] && !actor.MainSkill.SkillFlags[SkillFlagMine] && !actor.MainSkill.SkillFlags[SkillFlagTotem] {
			actor.ModDB.Conditions["HitRecently"] = true
		}
		if actor.MainSkill.SkillFlags[SkillFlagTotem] {
			actor.ModDB.Conditions["HaveTotem"] = true
			actor.ModDB.Conditions["SummonedTotemRecently"] = true
		}
		if actor.MainSkill.SkillFlags[SkillFlagMine] {
			actor.ModDB.Conditions["DetonatedMinesRecently"] = true
		}
		if actor.ModDB.Sum(mod.TypeBase, nil, "EnemyScorchChance") > 0 || actor.ModDB.Flag(nil, "CritAlwaysAltAilments") && !actor.ModDB.Flag(nil, "NeverCrit") {
			actor.ModDB.Conditions["CanInflictScorch"] = true
		}
		if actor.ModDB.Sum(mod.TypeBase, nil, "EnemyBrittleChance") > 0 || actor.ModDB.Flag(nil, "CritAlwaysAltAilments") && !actor.ModDB.Flag(nil, "NeverCrit") {
			actor.ModDB.Conditions["CanInflictBrittle"] = true
		}
		if actor.ModDB.Sum(mod.TypeBase, nil, "EnemySapChance") > 0 || actor.ModDB.Flag(nil, "CritAlwaysAltAilments") && !actor.ModDB.Flag(nil, "NeverCrit") {
			actor.ModDB.Conditions["CanInflictSap"] = true
		}
	}
	if env.ModeEffective {
		if env.Player.MainSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "FireExposureChance") > 0 || actor.ModDB.Sum(mod.TypeBase, nil, "FireExposureChance") > 0 {
			actor.ModDB.Conditions["CanApplyFireExposure"] = true
		}
		if env.Player.MainSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "ColdExposureChance") > 0 || actor.ModDB.Sum(mod.TypeBase, nil, "ColdExposureChance") > 0 {
			actor.ModDB.Conditions["CanApplyColdExposure"] = true
		}
		if env.Player.MainSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "LightningExposureChance") > 0 || actor.ModDB.Sum(mod.TypeBase, nil, "LightningExposureChance") > 0 {
			actor.ModDB.Conditions["CanApplyLightningExposure"] = true
		}
	}

	calculateAttributes := func() {
		for p := 1; p <= 2; p++ {
			for _, stat := range []string{"Str", "Dex", "Int"} {
				actor.Output[stat] = math.Max(math.Round(CalcVal(actor.ModDB, stat, nil)), 0)
				if actor.Breakdown != nil {
					actor.Breakdown.Simple(nil, nil, actor.Output[stat], stat)
				}
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
				if pass != 1 {
					for _, stat := range ({"Str","Dex","Int"}) {
						base := classStats["base_"+stat:lower()]
						actor.Output[stat] = min(round(calclib.Val(modDB, stat)), base)
						if breakdown != nil {
							breakdown[stat] = breakdown.simple(nil, nil, actor.Output[stat], stat)
						}

						modDB.NewMod("Omni", "BASE", (modDB.Sum(mod.TypeBase, nil, stat) - base), stat+" conversion Omniscience")
						modDB.NewMod("Omni", "INC", modDB.Sum(mod.TypeIncrease, nil, stat), "Omniscience")
						modDB.NewMod("Omni", "MORE", modDB.Sum("MORE", nil, stat), "Omniscience")
					}
				}

				if pass != 2 {
					// Subtract out double and triple dips
					conversion := { }
					reduction := { }
					for _, type := range ({"BASE", "INC", "MORE"}) {
						conversion[type] = { }
						for _, stat := range ({"StrDex", "StrInt", "DexInt", "All"}) {
							conversion[type][stat] = modDB.Sum(type, nil, stat) or 0
						}
						reduction[type] = conversion[type].StrDex + conversion[type].StrInt + conversion[type].DexInt + 2*conversion[type].All
					}
					modDB.NewMod("Omni", "BASE", -reduction["BASE"], "Reduction from Double/Triple Dipped attributes to Omniscience")
					modDB.NewMod("Omni", "INC", -reduction["INC"], "Reduction from Double/Triple Dipped attributes to Omniscience")
					modDB.NewMod("Omni", "MORE", -reduction["MORE"], "Reduction from Double/Triple Dipped attributes to Omniscience")
				}

				for _, stat := range ({"Str","Dex","Int"}) {
					base := classStats["base_"+stat:lower()]
					actor.Output[stat] = base
				}

				actor.Output["Omni"] = max(round(calclib.Val(modDB, "Omni")), 0)
				if breakdown != nil {
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

	// Calculate total attributes
	actor.Output["TotalAttr"] = actor.Output["Str"] + actor.Output["Dex"] + actor.Output["Int"]

	// Special case for Devotion
	actor.Output["Devotion"] = env.ModDB.Sum(mod.TypeBase, nil, "Devotion")

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
		for _, value := range (modDB.List(nil, "ShrineBuff")) {
			modDB.ScaleAddList({ value.mod }, calcLib.mod(modDB, nil, "BuffEffectOnSelf", "ShrineBuffEffect"))
		}
	*/
	actor.Output["ChaosInoculation"] = utils.Ternary[float64](env.ModDB.Flag(nil, "ChaosInoculation"), 1, 0)

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
		if actor.Breakdown != nil {
			if inc != 0 || more != 1 || conv != 0 {
				actor.Breakdown.AddLine("Life", fmt.Sprintf("%g ^8(base)", base))
				if inc != 0 {
					actor.Breakdown.AddLine("Life", fmt.Sprintf("x %.2f ^8(increased/reduced)", 1+inc/100))
				}
				if more != 1 {
					actor.Breakdown.AddLine("Life", fmt.Sprintf("x %.2f ^8(more/less)", more))
				}
				if conv != 0 {
					actor.Breakdown.AddLine("Life", fmt.Sprintf("x %.2f ^8(converted to Energy Shield)", 1-conv/100))
				}
				actor.Breakdown.AddLine("Life", fmt.Sprintf("= %g", actor.Output["Life"]))
			}
		}
	}
	manaConv := actor.ModDB.Sum(mod.TypeBase, nil, "ManaConvertToArmour")
	actor.Output["Mana"] = utils.RoundTo(calclib.Val(actor.ModDB, "Mana")*(1-manaConv/100), 0)
	base := actor.ModDB.Sum(mod.TypeBase, nil, "Mana")
	inc := actor.ModDB.Sum(mod.TypeIncrease, nil, "Mana")
	more := actor.ModDB.More(nil, "Mana")
	if actor.Breakdown != nil {
		if inc != 0 || more != 1 || manaConv != 0 {
			actor.Breakdown.AddLine("Mana", fmt.Sprintf("%g ^8(base)", base))
			if inc != 0 {
				actor.Breakdown.AddLine("Mana", fmt.Sprintf("x %.2f ^8(increased/reduced)", 1+inc/100))
			}
			if more != 1 {
				actor.Breakdown.AddLine("Mana", fmt.Sprintf("x %.2f ^8(more/less)", more))
			}
			if manaConv != 0 {
				actor.Breakdown.AddLine("Mana", fmt.Sprintf("x %.2f ^8(converted to Armour)", 1-manaConv/100))
			}
			actor.Breakdown.AddLine("Mana", fmt.Sprintf("= %g", actor.Output["Mana"]))
		}
	}
	actor.Output["LowestOfMaximumLifeAndMaximumMana"] = min(actor.Output["Life"], actor.Output["Mana"])
}

func mergeKeystones(env *Environment) {
	/*
		TODO mergeKeystones
		modDB := env.modDB

		for _, name := range (modDB.List(nil, "Keystone")) {
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

	// Conditionally over-write Charge values
	if modDB.Flag(nil, "UsePowerCharges") {
		actor.Output["PowerCharges"] = utils.Or(modDB.Override(nil, "PowerCharges"), actor.Output["PowerChargesMax"])
	}
	if modDB.Flag(nil, "PowerChargesConvertToAbsorptionCharges") {
		// we max with possible Power Charge Override from Config since Absorption Charges won't have their own config entry
		// and are converted from Power Charges
		actor.Output["AbsorptionCharges"] = max(actor.Output["PowerCharges"], min(actor.Output["AbsorptionChargesMax"], actor.Output["AbsorptionChargesMin"]))
		actor.Output["PowerCharges"] = 0
	} else {
		actor.Output["PowerCharges"] = max(actor.Output["PowerCharges"], min(actor.Output["PowerChargesMax"], actor.Output["PowerChargesMin"]))
	}
	actor.Output["RemovablePowerCharges"] = max(actor.Output["PowerCharges"]-actor.Output["PowerChargesMin"], 0)
	if modDB.Flag(nil, "UseFrenzyCharges") {
		actor.Output["FrenzyCharges"] = utils.Or(modDB.Override(nil, "FrenzyCharges"), actor.Output["FrenzyChargesMax"])
	}
	if modDB.Flag(nil, "FrenzyChargesConvertToAfflictionCharges") {
		// we max with possible Power Charge Override from Config since Absorption Charges won't have their own config entry
		// and are converted from Power Charges
		actor.Output["AfflictionCharges"] = max(actor.Output["FrenzyCharges"], min(actor.Output["AfflictionChargesMax"], actor.Output["AfflictionChargesMin"]))
		actor.Output["FrenzyCharges"] = 0
	} else {
		actor.Output["FrenzyCharges"] = max(actor.Output["FrenzyCharges"], min(actor.Output["FrenzyChargesMax"], actor.Output["FrenzyChargesMin"]))
	}
	actor.Output["RemovableFrenzyCharges"] = max(actor.Output["FrenzyCharges"]-actor.Output["FrenzyChargesMin"], 0)
	if modDB.Flag(nil, "UseEnduranceCharges") {
		actor.Output["EnduranceCharges"] = utils.Or(modDB.Override(nil, "EnduranceCharges"), actor.Output["EnduranceChargesMax"])
	}
	if modDB.Flag(nil, "EnduranceChargesConvertToBrutalCharges") {
		// we max with possible Endurance Charge Override from Config since Brutal Charges won't have their own config entry
		// and are converted from Endurance Charges
		actor.Output["BrutalCharges"] = max(actor.Output["EnduranceCharges"], min(actor.Output["BrutalChargesMax"], actor.Output["BrutalChargesMin"]))
		actor.Output["EnduranceCharges"] = 0
	} else {
		actor.Output["EnduranceCharges"] = max(actor.Output["EnduranceCharges"], min(actor.Output["EnduranceChargesMax"], actor.Output["EnduranceChargesMin"]))
	}
	actor.Output["RemovableEnduranceCharges"] = max(actor.Output["EnduranceCharges"]-actor.Output["EnduranceChargesMin"], 0)
	if modDB.Flag(nil, "UseSiphoningCharges") {
		actor.Output["SiphoningCharges"] = utils.Or(modDB.Override(nil, "SiphoningCharges"), actor.Output["SiphoningChargesMax"])
	}
	if modDB.Flag(nil, "UseChallengerCharges") {
		actor.Output["ChallengerCharges"] = utils.Or(modDB.Override(nil, "ChallengerCharges"), actor.Output["ChallengerChargesMax"])
	}
	if modDB.Flag(nil, "UseBlitzCharges") {
		actor.Output["BlitzCharges"] = utils.Or(modDB.Override(nil, "BlitzCharges"), actor.Output["BlitzChargesMax"])
	}
	if env.Player.MainSkill.Minion != nil {
		actor.Output["InspirationCharges"] = utils.Or(modDB.Override(nil, "InspirationCharges"), actor.Output["InspirationChargesMax"])
	}
	if modDB.Flag(nil, "UseGhostShrouds") {
		actor.Output["GhostShrouds"] = utils.Or(modDB.Override(nil, "GhostShrouds"), 3)
	}
	if modDB.Flag(nil, "CryWolfMinimumPower") && modDB.Sum(mod.TypeBase, nil, "WarcryPower") < 10 {
		modDB.AddMod(mod.NewFloat("WarcryPower", "OVERRIDE", 10).Source("Minimum Warcry Power from CryWolf"))
	}
	if modDB.Flag(nil, "WarcryInfinitePower") {
		modDB.AddMod(mod.NewFloat("WarcryPower", "OVERRIDE", 999999).Source("Warcries have infinite power"))
	}
	actor.Output["BloodCharges"] = min(utils.Or(modDB.Override(nil, "BloodCharges"), actor.Output["BloodChargesMax"]), actor.Output["BloodChargesMax"])

	actor.Output["WarcryPower"] = utils.Or(modDB.Override(nil, "WarcryPower"), modDB.Sum(mod.TypeBase, nil, "WarcryPower"))
	actor.Output["CrabBarriers"] = min(utils.Or(modDB.Override(nil, "CrabBarriers"), actor.Output["CrabBarriersMax"]), actor.Output["CrabBarriersMax"])
	actor.Output["TotalCharges"] = actor.Output["PowerCharges"] + actor.Output["FrenzyCharges"] + actor.Output["EnduranceCharges"]
	modDB.Multipliers["WarcryPower"] = actor.Output["WarcryPower"]
	modDB.Multipliers["PowerCharge"] = actor.Output["PowerCharges"]
	modDB.Multipliers["PowerChargeMax"] = actor.Output["PowerChargesMax"]
	modDB.Multipliers["RemovablePowerCharge"] = actor.Output["RemovablePowerCharges"]
	modDB.Multipliers["FrenzyCharge"] = actor.Output["FrenzyCharges"]
	modDB.Multipliers["RemovableFrenzyCharge"] = actor.Output["RemovableFrenzyCharges"]
	modDB.Multipliers["EnduranceCharge"] = actor.Output["EnduranceCharges"]
	modDB.Multipliers["RemovableEnduranceCharge"] = actor.Output["RemovableEnduranceCharges"]
	modDB.Multipliers["TotalCharges"] = actor.Output["TotalCharges"]
	modDB.Multipliers["SiphoningCharge"] = actor.Output["SiphoningCharges"]
	modDB.Multipliers["ChallengerCharge"] = actor.Output["ChallengerCharges"]
	modDB.Multipliers["BlitzCharge"] = actor.Output["BlitzCharges"]
	modDB.Multipliers["InspirationCharge"] = actor.Output["InspirationCharges"]
	modDB.Multipliers["GhostShroud"] = actor.Output["GhostShrouds"]
	modDB.Multipliers["CrabBarrier"] = actor.Output["CrabBarriers"]
	modDB.Multipliers["BrutalCharge"] = actor.Output["BrutalCharges"]
	modDB.Multipliers["AbsorptionCharge"] = actor.Output["AbsorptionCharges"]
	modDB.Multipliers["AfflictionCharge"] = actor.Output["AfflictionCharges"]
	modDB.Multipliers["BloodCharge"] = actor.Output["BloodCharges"]
	// Process enemy modifiers
	for _, value := range modDB.List(nil, "EnemyModifier") {
		actor.Enemy.ModDB.AddMod(value.(mod.EnemyModifier).Mod)
	}

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
