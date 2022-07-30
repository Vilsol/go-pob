package calculator

import (
	"go-pob/calculator/mod"
	"go-pob/data"
	"go-pob/utils"
)

// CreateActiveSkill Create an active skill using the given active gem and list of support gems
// It will determine the base flag set, and check which of the support gems can support this skill
func CreateActiveSkill(activeEffect *ActiveEffect, supportList []interface{}, actor *Actor, socketGroup interface{}, summonSkill interface{}) *ActiveSkill {
	activeSkill := &ActiveSkill{
		ActiveEffect: activeEffect,
		SupportList:  supportList,
		SkillData:    make(map[string]interface{}),
		Actor:        actor,
		SocketGroup:  socketGroup,
		SummonSkill:  summonSkill,
	}

	activeSkill.SkillTypes = utils.CopyMap(activeEffect.GrantedEffect.SkillTypes)

	/*
		TODO -- Initialise skill types
		if activeGrantedEffect.minionSkillTypes then
			activeSkill.minionSkillTypes = copyTable(activeGrantedEffect.minionSkillTypes)
		end
	*/

	activeSkill.SkillFlags = utils.CopyMap(activeEffect.GrantedEffect.BaseFlags)
	activeSkill.SkillFlags[SkillFlagHit] = activeSkill.SkillFlags[SkillFlagHit] || activeSkill.SkillTypes[data.SkillTypeAttack] || activeSkill.SkillTypes[data.SkillTypeDamage] || activeSkill.SkillTypes[data.SkillTypeProjectile]

	/*
		TODO -- Process support skills
		activeSkill.effectList = { activeEffect }
		for _, supportEffect in ipairs(supportList) do
			-- Pass 1: Add skill types from compatible supports
			if calcLib.canGrantedEffectSupportActiveSkill(supportEffect.grantedEffect, activeSkill) then
				for _, skillType in pairs(supportEffect.grantedEffect.addSkillTypes) do
					activeSkill.skillTypes[skillType] = true
				end
			end
		end
		for _, supportEffect in ipairs(supportList) do
			-- Pass 2: Add all compatible supports
			if calcLib.canGrantedEffectSupportActiveSkill(supportEffect.grantedEffect, activeSkill) then
				t_insert(activeSkill.effectList, supportEffect)
				if supportEffect.isSupporting and activeEffect.srcInstance then
					supportEffect.isSupporting[activeEffect.srcInstance] = true
				end
				if supportEffect.grantedEffect.addFlags and not summonSkill then
					-- Support skill adds flags to supported skills (eg. Remote Mine adds 'mine')
					for k in pairs(supportEffect.grantedEffect.addFlags) do
						skillFlags[k] = true
					end
				end
			end
		end
	*/

	return activeSkill
}

func CalcBuildActiveSkillModList(env *Environment, activeSkill *ActiveSkill) {
	/*
		local skillTypes = activeSkill.skillTypes
		local skillFlags = activeSkill.skillFlags
		local activeEffect = activeSkill.activeEffect
		local activeGrantedEffect = activeEffect.grantedEffect
	*/

	if env.ModeBuffs {
		activeSkill.SkillFlags[SkillFlagBuffs] = true
	}
	if env.ModeCombat {
		activeSkill.SkillFlags[SkillFlagCombat] = true
	}
	if env.ModeEffective {
		activeSkill.SkillFlags[SkillFlagEffective] = true
	}

	// Handle multipart skills
	activeGemParts := activeSkill.ActiveEffect.GrantedEffect.Parts
	if activeGemParts != nil {
		/*
			TODO Handle multipart skills
			if env.mode == "CALCS" and activeSkill == env.player.mainSkill then
				activeEffect.srcInstance.skillPartCalcs = m_min(#activeGemParts, activeEffect.srcInstance.skillPartCalcs or 1)
				activeSkill.skillPart = activeEffect.srcInstance.skillPartCalcs
			else
				activeEffect.srcInstance.skillPart = m_min(#activeGemParts, activeEffect.srcInstance.skillPart or 1)
				activeSkill.skillPart = activeEffect.srcInstance.skillPart
			end
			local part = activeGemParts[activeSkill.skillPart]
			for k, v in pairs(part) do
				if v == true then
					skillFlags[k] = true
				elseif v == false then
					skillFlags[k] = nil
				end
			end
			activeSkill.skillPartName = part.name
			skillFlags.multiPart = #activeGemParts > 1
		*/
	}
	/*
		TODO Shield Attacks
		if (skillTypes[SkillType.RequiresShield] or skillFlags.shieldAttack) and not activeSkill.summonSkill and (not activeSkill.actor.itemList["Weapon 2"] or activeSkill.actor.itemList["Weapon 2"].type ~= "Shield") then
			-- Skill requires a shield to be equipped
			skillFlags.disable = true
			activeSkill.disableReason = "This skill requires a Shield"
		end

		if skillFlags.shieldAttack then
			-- Special handling for Spectral Shield Throw
			skillFlags.weapon2Attack = true
			activeSkill.weapon2Flags = 0
		else
			-- Set weapon flags
			local weaponTypes = { activeGrantedEffect.weaponTypes }
			for _, skillEffect in pairs(activeSkill.effectList) do
				if skillEffect.grantedEffect.support and skillEffect.grantedEffect.weaponTypes then
					t_insert(weaponTypes, skillEffect.grantedEffect.weaponTypes)
				end
			end
			local weapon1Flags, weapon1Info = getWeaponFlags(env, activeSkill.actor.weaponData1, weaponTypes)
			if not weapon1Flags and activeSkill.summonSkill then
				-- Minion skills seem to ignore weapon types
				weapon1Flags, weapon1Info = ModFlag[env.data.weaponTypeInfo["None"].flag], env.data.weaponTypeInfo["None"]
			end
			if weapon1Flags then
				if skillFlags.attack then
					activeSkill.weapon1Flags = weapon1Flags
					skillFlags.weapon1Attack = true
					if weapon1Info.melee and skillFlags.melee then
						skillFlags.projectile = nil
					elseif not weapon1Info.melee and skillFlags.projectile then
						skillFlags.melee = nil
					end
				end
			elseif skillTypes[SkillType.DualWieldOnly] or skillTypes[SkillType.MainHandOnly] or skillFlags.forceMainHand or weapon1Info then
				-- Skill requires a compatible main hand weapon
				skillFlags.disable = true
				activeSkill.disableReason = "Main Hand weapon is not usable with this skill"
			end
			if not skillTypes[SkillType.MainHandOnly] and not skillFlags.forceMainHand then
				local weapon2Flags, weapon2Info = getWeaponFlags(env, activeSkill.actor.weaponData2, weaponTypes)
				if weapon2Flags then
					if skillFlags.attack then
						activeSkill.weapon2Flags = weapon2Flags
						skillFlags.weapon2Attack = true
					end
				elseif skillTypes[SkillType.DualWieldOnly] or weapon2Info then
					-- Skill requires a compatible off hand weapon
					skillFlags.disable = true
					activeSkill.disableReason = activeSkill.disableReason or "Off Hand weapon is not usable with this skill"
				elseif skillFlags.disable then
					-- Neither weapon is compatible
					activeSkill.disableReason = "No usable weapon equipped"
				end
			end
			if skillFlags.attack then
				skillFlags.bothWeaponAttack = skillFlags.weapon1Attack and skillFlags.weapon2Attack
			end
		end
	*/

	// Build skill mod flag set
	skillModFlags := data.ModFlag(0)

	if utils.HasTrue(activeSkill.SkillFlags, SkillFlagHit) {
		skillModFlags |= data.ModFlagHit
	}

	if utils.HasTrue(activeSkill.SkillFlags, SkillFlagAttack) {
		skillModFlags |= data.ModFlagAttack
	} else {
		skillModFlags |= data.ModFlagCast
		if utils.HasTrue(activeSkill.SkillFlags, SkillFlagSpell) {
			skillModFlags |= data.ModFlagSpell
		}
	}

	if utils.HasTrue(activeSkill.SkillFlags, SkillFlagMelee) {
		skillModFlags |= data.ModFlagMelee
	} else if utils.HasTrue(activeSkill.SkillFlags, SkillFlagProjectile) {
		skillModFlags |= data.ModFlagProjectile
		activeSkill.SkillFlags[SkillFlagChaining] = true
	}

	if utils.HasTrue(activeSkill.SkillFlags, SkillFlagArea) {
		skillModFlags |= data.ModFlagArea
	}

	// Build skill keyword flag set
	skillKeywordFlags := mod.KeywordFlag(0)

	if activeSkill.SkillFlags[SkillFlagHit] {
		skillKeywordFlags |= mod.KeywordFlagHit
	}

	if activeSkill.SkillTypes[data.SkillTypeAura] {
		skillKeywordFlags |= mod.KeywordFlagAura
	}

	if activeSkill.SkillTypes[data.SkillTypeHex] || activeSkill.SkillTypes[data.SkillTypeMark] {
		skillKeywordFlags |= mod.KeywordFlagCurse
	}

	if activeSkill.SkillTypes[data.SkillTypeWarcry] {
		skillKeywordFlags |= mod.KeywordFlagWarcry
	}

	if activeSkill.SkillTypes[data.SkillTypeMovement] {
		skillKeywordFlags |= mod.KeywordFlagMovement
	}

	if activeSkill.SkillTypes[data.SkillTypeVaal] {
		skillKeywordFlags |= mod.KeywordFlagVaal
	}

	if activeSkill.SkillTypes[data.SkillTypeLightning] {
		skillKeywordFlags |= mod.KeywordFlagLightning
	}

	if activeSkill.SkillTypes[data.SkillTypeCold] {
		skillKeywordFlags |= mod.KeywordFlagCold
	}

	if activeSkill.SkillTypes[data.SkillTypeFire] {
		skillKeywordFlags |= mod.KeywordFlagFire
	}

	if activeSkill.SkillTypes[data.SkillTypeChaos] {
		skillKeywordFlags |= mod.KeywordFlagChaos
	}

	if activeSkill.SkillFlags[SkillFlagWeapon1Attack] && activeSkill.Weapon1Flags&data.ModFlagBow != 0 {
		skillKeywordFlags |= mod.KeywordFlagBow
	}

	if activeSkill.SkillFlags[SkillFlagBrand] {
		skillKeywordFlags |= mod.KeywordFlagBrand
	}

	if activeSkill.SkillFlags[SkillFlagTotem] {
		skillKeywordFlags |= mod.KeywordFlagTotem
	} else if activeSkill.SkillFlags[SkillFlagTrap] {
		skillKeywordFlags |= mod.KeywordFlagTrap
	} else if activeSkill.SkillFlags[SkillFlagMine] {
		skillKeywordFlags |= mod.KeywordFlagMine
	} else {
		activeSkill.SkillFlags[SkillFlagSelfCast] = true
	}

	if activeSkill.SkillTypes[data.SkillTypeAttack] {
		skillKeywordFlags |= mod.KeywordFlagAttack
	}

	if activeSkill.SkillTypes[data.SkillTypeSpell] && !activeSkill.SkillFlags[SkillFlagCast] {
		skillKeywordFlags |= mod.KeywordFlagSpell
	}
	/*
		TODO -- Get skill totem ID for totem skills
		-- This is used to calculate totem life
		if skillFlags.totem then
			activeSkill.skillTotemId = activeGrantedEffect.skillTotemId
			if not activeSkill.skillTotemId then
				if activeGrantedEffect.color == 2 then
					activeSkill.skillTotemId = 2
				elseif activeGrantedEffect.color == 3 then
					activeSkill.skillTotemId = 3
				else
					activeSkill.skillTotemId = 1
				end
			end
		end
	*/
	/*
		TODO -- Calculate Distance for meleeDistance or projectileDistance (for melee proximity, e.g. Impact)
		effectiveRange := float64(0)
		if skillFlags.melee then
			effectiveRange = env.configInput.meleeDistance
		else
			effectiveRange = env.configInput.projectileDistance
		end
	*/
	activeSkill.SkillCfg = &ListCfg{
		Flags:        utils.Ptr(skillModFlags | activeSkill.Weapon1Flags | activeSkill.Weapon2Flags),
		KeywordFlags: utils.Ptr(skillKeywordFlags),
		SkillCond:    make(map[string]bool),
		/*
			TODO
			skillName = activeGrantedEffect.name:gsub("^Vaal ",""):gsub("Summon Skeletons","Summon Skeleton"), -- This allows modifiers that target specific skills to also apply to their Vaal counterpart
			summonSkillName = activeSkill.summonSkill and activeSkill.summonSkill.activeEffect.grantedEffect.name,
			skillGem = activeEffect.gemData,
			skillGrantedEffect = activeGrantedEffect,
			skillPart = activeSkill.skillPart,
			skillTypes = activeSkill.skillTypes,
			skillDist = env.mode_effective and effectiveRange,
			slotName = activeSkill.slotName,
		*/
	}
	/*
		TODO -- Build config structure for modifier searches
		if skillFlags.weapon1Attack then
			activeSkill.weapon1Cfg = copyTable(activeSkill.skillCfg, true)
			activeSkill.weapon1Cfg.skillCond = setmetatable({ ["MainHandAttack"] = true }, { __index = activeSkill.skillCfg.skillCond })
			activeSkill.weapon1Cfg.flags = bor(skillModFlags, activeSkill.weapon1Flags)
		end
		if skillFlags.weapon2Attack then
			activeSkill.weapon2Cfg = copyTable(activeSkill.skillCfg, true)
			activeSkill.weapon2Cfg.skillCond = setmetatable({ ["OffHandAttack"] = true }, { __index = activeSkill.skillCfg.skillCond })
			activeSkill.weapon2Cfg.flags = bor(skillModFlags, activeSkill.weapon2Flags)
		end
	*/
	/*
		TODO -- Initialise skill modifier list
		local skillModList = new("ModList", activeSkill.actor.modDB)
		activeSkill.skillModList = skillModList
		activeSkill.baseSkillModList = skillModList

		if skillModList:Flag(activeSkill.skillCfg, "DisableSkill") and not skillModList:Flag(activeSkill.skillCfg, "EnableSkill") then
			skillFlags.disable = true
			activeSkill.disableReason = "Skills of this type are disabled"
		end

		if skillFlags.disable then
			wipeTable(skillFlags)
			skillFlags.disable = true
			calcLib.validateGemLevel(activeEffect)
			activeEffect.grantedEffectLevel = activeGrantedEffect.levels[activeEffect.level]
			return
		end
	*/
	/*
		TODO -- Add support gem modifiers to skill mod list
		for _, skillEffect in pairs(activeSkill.effectList) do
			if skillEffect.grantedEffect.support then
				calcs.mergeSkillInstanceMods(env, skillModList, skillEffect)
				local level = skillEffect.grantedEffect.levels[skillEffect.level]
				if level.manaMultiplier then
					skillModList:NewMod("SupportManaMultiplier", "MORE", level.manaMultiplier, skillEffect.grantedEffect.modSource)
				end
				if level.manaReservationPercent then
					activeSkill.skillData.manaReservationPercent = level.manaReservationPercent
				end
				if level.cooldown then
					activeSkill.skillData.cooldown = level.cooldown
				end
			end
		end
	*/
	/*
		TODO -- Apply gem/quality modifiers from support gems
		for _, value in ipairs(skillModList:List(activeSkill.skillCfg, "SupportedGemProperty")) do
			if value.keyword == "active_skill" and activeSkill.activeEffect.gemData then
				activeEffect[value.key] = activeEffect[value.key] + value.value
			end
		end
	*/
	/*
		TODO -- Add active gem modifiers
		activeEffect.actorLevel = activeSkill.actor.minionData and activeSkill.actor.level
		calcs.mergeSkillInstanceMods(env, skillModList, activeEffect, skillModList:List(activeSkill.skillCfg, "ExtraSkillStat"))
		activeEffect.grantedEffectLevel = activeGrantedEffect.levels[activeEffect.level]
	*/
	/*
		TODO -- Add extra modifiers from granted effect level
		local level = activeEffect.grantedEffectLevel
		activeSkill.skillData.CritChance = level.critChance
		if level.damageMultiplier then
			skillModList:NewMod("Damage", "MORE", level.damageMultiplier, activeEffect.grantedEffect.modSource, ModFlag.Attack)
		end
		if level.attackTime then
			activeSkill.skillData.attackTime = level.attackTime
		end
		if level.attackSpeedMultiplier then
			skillModList:NewMod("Speed", "MORE", level.attackSpeedMultiplier, activeEffect.grantedEffect.modSource, ModFlag.Attack)
		end
		if level.cooldown then
			activeSkill.skillData.cooldown = level.cooldown
		end
	*/
	/*
		TODO -- Add extra modifiers from other sources
		activeSkill.extraSkillModList = { }
		for _, value in ipairs(skillModList:List(activeSkill.skillCfg, "ExtraSkillMod")) do
			skillModList:AddMod(value.mod)
			t_insert(activeSkill.extraSkillModList, value.mod)
		end
	*/
	/*
		TODO -- Find totem level
		if skillFlags.totem then
			activeSkill.skillData.totemLevel = activeEffect.grantedEffectLevel.levelRequirement
		end
	*/
	/*
		TODO -- Add active mine multiplier
		if skillFlags.mine then
			activeSkill.activeMineCount = (env.mode == "CALCS" and activeEffect.srcInstance.skillMineCountCalcs) or (env.mode ~= "CALCS" and activeEffect.srcInstance.skillMineCount)
			if activeSkill.activeMineCount and activeSkill.activeMineCount > 0 then
				skillModList:NewMod("Multiplier:ActiveMineCount", "BASE", activeSkill.activeMineCount, "Base")
				env.enemy.modDB.multipliers["ActiveMineCount"] = m_max(activeSkill.activeMineCount or 0, env.enemy.modDB.multipliers["ActiveMineCount"] or 0)
			end
		end

		if skillModList:Sum("BASE", activeSkill.skillCfg, "Multiplier:"..activeGrantedEffect.name:gsub("%s+", "").."MaxStages") > 0 then
			skillFlags.multiStage = true
			activeSkill.activeStageCount = (env.mode == "CALCS" and activeEffect.srcInstance.skillStageCountCalcs) or (env.mode ~= "CALCS" and activeEffect.srcInstance.skillStageCount)
			local limit = skillModList:Sum("BASE", activeSkill.skillCfg, "Multiplier:"..activeGrantedEffect.name:gsub("%s+", "").."MaxStages")
			if limit > 0 then
				if activeSkill.activeStageCount and activeSkill.activeStageCount > 0 then
					skillModList:NewMod("Multiplier:"..activeGrantedEffect.name:gsub("%s+", "").."Stage", "BASE", m_min(limit, activeSkill.activeStageCount), "Base")
					activeSkill.activeStageCount = (activeSkill.activeStageCount or 0) - 1
					skillModList:NewMod("Multiplier:"..activeGrantedEffect.name:gsub("%s+", "").."StageAfterFirst", "BASE", m_min(limit - 1, activeSkill.activeStageCount), "Base")
				end
			end
		end
	*/
	/*
		TODO -- Extract skill data
		for _, value in ipairs(env.modDB:List(activeSkill.skillCfg, "SkillData")) do
			activeSkill.skillData[value.key] = value.value
		end
		for _, value in ipairs(skillModList:List(activeSkill.skillCfg, "SkillData")) do
			activeSkill.skillData[value.key] = value.value
		end
	*/
	/*
		TODO -- Create minion
		local minionList, isSpectre
		if activeGrantedEffect.minionList then
			if activeGrantedEffect.minionList[1] then
				minionList = copyTable(activeGrantedEffect.minionList)
			else
				minionList = copyTable(env.build.spectreList)
				isSpectre = true
			end
		else
			minionList = { }
		end
		for _, skillEffect in ipairs(activeSkill.effectList) do
			if skillEffect.grantedEffect.support and skillEffect.grantedEffect.addMinionList then
				for _, minionType in ipairs(skillEffect.grantedEffect.addMinionList) do
					t_insert(minionList, minionType)
				end
			end
		end
		activeSkill.minionList = minionList
		if minionList[1] and not activeSkill.actor.minionData then
			local minionType
			if env.mode == "CALCS" and activeSkill == env.player.mainSkill then
				local index = isValueInArray(minionList, activeEffect.srcInstance.skillMinionCalcs) or 1
				minionType = minionList[index]
				activeEffect.srcInstance.skillMinionCalcs = minionType
			else
				local index = isValueInArray(minionList, activeEffect.srcInstance.skillMinion) or 1
				minionType = minionList[index]
				activeEffect.srcInstance.skillMinion = minionType
			end
			if minionType then
				local minion = { }
				activeSkill.minion = minion
				skillFlags.haveMinion = true
				minion.parent = env.player
				minion.enemy = env.enemy
				minion.type = minionType
				minion.minionData = env.data.minions[minionType]
				minion.level = activeSkill.skillData.minionLevelIsEnemyLevel and env.enemyLevel or activeSkill.skillData.minionLevel or activeEffect.grantedEffectLevel.levelRequirement
				-- fix minion level between 1 and 100
				minion.level = m_min(m_max(minion.level,1),100)
				minion.itemList = { }
				minion.uses = activeGrantedEffect.minionUses
				minion.lifeTable = isSpectre and env.data.monsterLifeTable or env.data.monsterAllyLifeTable
				local attackTime = minion.minionData.attackTime * (1 - (minion.minionData.damageFixup or 0))
				local damage = env.data.monsterDamageTable[minion.level] * minion.minionData.damage * attackTime
				if activeGrantedEffect.minionHasItemSet then
					if env.mode == "CALCS" and activeSkill == env.player.mainSkill then
						if not env.build.itemsTab.itemSets[activeEffect.srcInstance.skillMinionItemSetCalcs] then
							activeEffect.srcInstance.skillMinionItemSetCalcs = env.build.itemsTab.itemSetOrderList[1]
						end
						minion.itemSet = env.build.itemsTab.itemSets[activeEffect.srcInstance.skillMinionItemSetCalcs]
					else
						if not env.build.itemsTab.itemSets[activeEffect.srcInstance.skillMinionItemSet] then
							activeEffect.srcInstance.skillMinionItemSet = env.build.itemsTab.itemSetOrderList[1]
						end
						minion.itemSet = env.build.itemsTab.itemSets[activeEffect.srcInstance.skillMinionItemSet]
					end
				end
				if activeSkill.skillData.minionUseBowAndQuiver and env.player.weaponData1.type == "Bow" then
					minion.weaponData1 = env.player.weaponData1
				elseif env.theIronMass and minionType == "RaisedSkeleton" then
					minion.weaponData1 = env.player.weaponData1
				else
					minion.weaponData1 = {
						type = minion.minionData.weaponType1 or "None",
						AttackRate = 1 / attackTime,
						CritChance = 5,
						PhysicalMin = round(damage * (1 - minion.minionData.damageSpread)),
						PhysicalMax = round(damage * (1 + minion.minionData.damageSpread)),
						range = minion.minionData.attackRange,
					}
				end
				minion.weaponData2 = { }
				if minion.uses then
					if minion.uses["Weapon 1"] then
						if minion.itemSet then
							local item = env.build.itemsTab.items[minion.itemSet[minion.itemSet.useSecondWeaponSet and "Weapon 1 Swap" or "Weapon 1"].selItemId]
							if item and item.weaponData then
								minion.weaponData1 = item.weaponData[1]
							end
						else
							minion.weaponData1 = env.player.weaponData1
						end
					end
					if minion.uses["Weapon 2"] then
						if minion.itemSet then
							local item = env.build.itemsTab.items[minion.itemSet[minion.itemSet.useSecondWeaponSet and "Weapon 2 Swap" or "Weapon 2"].selItemId]
							if item and item.weaponData then
								minion.weaponData2 = item.weaponData[2]
							end
						else
							minion.weaponData2 = env.player.weaponData2
						end
					end
				end
			end
		end
	*/
	/*
		TODO -- Separate global effect modifiers (mods that can affect defensive stats or other skills)
		local i = 1
		while skillModList[i] do
			local effectType, effectName, effectTag
			for _, tag in ipairs(skillModList[i]) do
				if tag.type == "GlobalEffect" then
					effectType = tag.effectType
					effectName = tag.effectName or activeGrantedEffect.name
					effectTag = tag
					break
				end
			end
			if effectTag and effectTag.modCond and not skillModList:GetCondition(effectTag.modCond, activeSkill.skillCfg) then
				t_remove(skillModList, i)
			elseif effectType then
				local buff
				for _, skillBuff in ipairs(activeSkill.buffList) do
					if skillBuff.type == effectType and skillBuff.name == effectName then
						buff = skillBuff
						break
					end
				end
				if not buff then
					buff = {
						type = effectType,
						name = effectName,
						allowTotemBuff = effectTag.allowTotemBuff,
						cond = effectTag.effectCond,
						enemyCond = effectTag.effectEnemyCond,
						stackVar = effectTag.effectStackVar,
						stackLimit = effectTag.effectStackLimit,
						stackLimitVar = effectTag.effectStackLimitVar,
						applyNotPlayer = effectTag.applyNotPlayer,
						applyMinions = effectTag.applyMinions,
						modList = { },
						unscalableModList = { },
					}
					if skillModList[i].source == activeGrantedEffect.modSource then
						-- Inherit buff configuration from the active skill
						buff.activeSkillBuff = true
						buff.applyNotPlayer = buff.applyNotPlayer or activeSkill.skillData.buffNotPlayer
						buff.applyMinions = buff.applyMinions or activeSkill.skillData.buffMinions
						buff.applyAllies = activeSkill.skillData.buffAllies
						buff.allowTotemBuff = activeSkill.skillData.allowTotemBuff
					end
					t_insert(activeSkill.buffList, buff)
				end
				local match = false
				local modList = effectTag.unscalable and buff.unscalableModList or buff.modList
				for d = 1, #modList do
					local destMod = modList[d]
					if modLib.compareModParams(skillModList[i], destMod) and (destMod.type == "BASE" or destMod.type == "INC") then
						destMod = copyTable(destMod)
						destMod.value = destMod.value + skillModList[i].value
						modList[d] = destMod
						match = true
						break
					end
				end
				if not match then
					t_insert(modList, skillModList[i])
				end
				t_remove(skillModList, i)
			else
				i = i + 1
			end
		end

		if activeSkill.buffList[1] then
			-- Add to auxiliary skill list
			t_insert(env.auxSkillList, activeSkill)
		end
	*/
}
