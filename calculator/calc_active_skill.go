package calculator

import (
	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

// CreateActiveSkill Create an active skill using the given active gem and list of support gems
// It will determine the base flag set, and check which of the support gems can support this skill
func CreateActiveSkill(activeEffect *GemEffect, supportList []*GemEffect, actor *Actor, socketGroup *pob.Skill, summonSkill *ActiveSkill) *ActiveSkill {
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

	activeSkill.EffectList = make([]*GemEffect, 1)
	activeSkill.EffectList[0] = activeEffect

	for _, supportEffect := range supportList {
		// Pass 1: Add skill types from compatible supports
		if CalcCanGrantedEffectSupportActiveSkill(supportEffect.GrantedEffect, activeSkill) {
			for _, skillType := range supportEffect.GrantedEffect.Raw.AddTypes {
				activeSkill.SkillTypes[data.SkillType(poe.ActiveSkillTypes[skillType].ID)] = true
			}
		}
	}

	for _, supportEffect := range supportList {
		// Pass 2: Add all compatible supports
		if CalcCanGrantedEffectSupportActiveSkill(supportEffect.GrantedEffect, activeSkill) {
			activeSkill.EffectList = append(activeSkill.EffectList, supportEffect)
			if supportEffect.IsSupporting != nil && activeEffect.SrcInstance != nil {
				supportEffect.IsSupporting[activeEffect.SrcInstance] = true
			}
			/*
				TODO Manually added support flags
				if supportEffect.grantedEffect.addFlags and not summonSkill then
					-- Support skill adds flags to supported skills (eg. Remote Mine adds 'mine')
					for k in pairs(supportEffect.grantedEffect.addFlags) do
						skillFlags[k] = true
					end
				end
			*/
		}
	}

	return activeSkill
}

func CalcMergeSkillInstanceMods(env *Environment, modList *ModList, skillEffect *GemEffect, extraStats []interface{}) {
	CalcValidateGemLevel(skillEffect)

	grantedEffect := skillEffect.GrantedEffect
	stats := CalcBuildSkillInstanceStats(skillEffect, grantedEffect)

	/*
		TODO extraStats
		if extraStats and extraStats[1] then
			for _, stat in pairs(extraStats) do
				stats[stat.key] = (stats[stat.key] or 0) + stat.value
			end
		end
	*/

	for stat, statValue := range stats {
		statMap := raw2.GetCalculatedGrantedEffect(grantedEffect.Raw).GetCalculatedStatMap().Get(stat)
		if statMap != nil {
			for _, m := range statMap.Mods {
				mergeLevelMod(modList, m, utils.UnwrapOrF(statMap.Value, statValue)*utils.UnwrapOrF(statMap.Mult, 1)/utils.UnwrapOrF(statMap.Div, 1)+utils.UnwrapOrF(statMap.Base, 0))
			}
		}
	}

	// TODO modList:AddList(grantedEffect.baseMods)
}

// mergeLevelMod Merges level modifier with given mod list
func mergeLevelMod(modList *ModList, m mod.Mod, value float64) {
	if value == 0 {
		modList.AddMod(m)
		return
	}

	newMod := m.Clone()

	switch cast := newMod.(type) {
	case *mod.FloatMod:
		cast.ModValue = value
	case *mod.ListMod:
		switch inner := cast.Value().(type) {
		case *mod.SkillData:
			inner.Value = value
		}
	}

	/*
		TODO mergeLevelMod
		if type(newMod.value) == "table" then
			newMod.value = copyTable(newMod.value, true)
			if newMod.value.mod then
				newMod.value.mod = copyTable(newMod.value.mod, true)
				newMod.value.mod.value = value
			else
				newMod.value.value = value
			end
		else
			newMod.value = value
		end
	*/

	modList.AddMod(newMod)
}

func CalcBuildActiveSkillModList(env *Environment, activeSkill *ActiveSkill) {
	skillTypes := activeSkill.SkillTypes
	skillFlags := activeSkill.SkillFlags
	activeEffect := activeSkill.ActiveEffect
	activeGrantedEffect := activeEffect.GrantedEffect

	if env.ModeBuffs {
		skillFlags[SkillFlagBuffs] = true
	}
	if env.ModeCombat {
		skillFlags[SkillFlagCombat] = true
	}
	if env.ModeEffective {
		skillFlags[SkillFlagEffective] = true
	}

	// Handle multipart skills
	activeGemParts := activeGrantedEffect.Parts
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
	*/

	if skillFlags[SkillFlagShieldAttack] {
		// Special handling for Spectral Shield Throw
		skillFlags[SkillFlagWeapon2Attack] = true
		activeSkill.Weapon2Flags = 0
	} else {
		// Set weapon flags

		weaponTypes := [][]data.ItemClassName{activeGrantedEffect.WeaponTypes()}
		for _, skillEffect := range activeSkill.EffectList {
			if skillEffect.GrantedEffect.Raw.IsSupport && skillEffect.GrantedEffect.WeaponTypes() != nil {
				weaponTypes = append(weaponTypes, skillEffect.GrantedEffect.WeaponTypes())
			}
		}

		weapon1Flags, weapon1Info := getWeaponFlags(env, activeSkill.Actor.WeaponData1, weaponTypes)
		if weapon1Flags == 0 && activeSkill.SummonSkill != nil {
			// Minion skills seem to ignore weapon types
			weapon1Flags = data.WeaponTypes[data.None].ModFlag
			weapon1Info = data.WeaponTypes[data.None]
		}

		if weapon1Flags != 0 {
			if skillFlags[SkillFlagAttack] {
				activeSkill.Weapon1Flags = weapon1Flags
				skillFlags[SkillFlagWeapon1Attack] = true
				if weapon1Info.Melee && skillFlags[SkillFlagMelee] {
					delete(skillFlags, SkillFlagProjectile)
				} else if !weapon1Info.Melee && skillFlags[SkillFlagProjectile] {
					delete(skillFlags, SkillFlagMelee)
				}
			}
		} else if skillTypes[data.SkillTypeDualWieldOnly] || skillTypes[data.SkillTypeMainHandOnly] || skillFlags[SkillFlagForceMainHand] || weapon1Info != nil {
			// Skill requires a compatible main hand weapon
			skillFlags[SkillFlagDisable] = true
			activeSkill.DisableReason = "Main Hand weapon is not usable with this skill"
		}

		if utils.MissingOrFalse(skillTypes, data.SkillTypeMainHandOnly) && utils.MissingOrFalse(skillFlags, SkillFlagForceMainHand) {
			weapon2Flags, weapon2Info := getWeaponFlags(env, activeSkill.Actor.WeaponData2, weaponTypes)
			if weapon2Flags != 0 {
				if skillFlags[SkillFlagAttack] {
					activeSkill.Weapon2Flags = weapon2Flags
					skillFlags[SkillFlagWeapon2Attack] = true
				}
			} else if skillTypes[data.SkillTypeDualWieldOnly] || weapon2Info != nil {
				// Skill requires a compatible off hand weapon
				skillFlags[SkillFlagDisable] = true
				if activeSkill.DisableReason != "" {
					activeSkill.DisableReason = "Off Hand weapon is not usable with this skill"
				}
			} else if skillFlags[SkillFlagDisable] {
				// Neither weapon is compatible
				activeSkill.DisableReason = "No usable weapon equipped"
			}
		}

		if skillFlags[SkillFlagAttack] {
			skillFlags[SkillFlagBothWeaponAttack] = skillFlags[SkillFlagWeapon1Attack] && skillFlags[SkillFlagWeapon2Attack]
		}
	}

	// Build skill mod flag set
	skillModFlags := mod.MFlag(0)

	if utils.HasTrue(skillFlags, SkillFlagHit) {
		skillModFlags |= mod.MFlagHit
	}

	if utils.HasTrue(skillFlags, SkillFlagAttack) {
		skillModFlags |= mod.MFlagAttack
	} else {
		skillModFlags |= mod.MFlagCast
		if utils.HasTrue(skillFlags, SkillFlagSpell) {
			skillModFlags |= mod.MFlagSpell
		}
	}

	if utils.HasTrue(skillFlags, SkillFlagMelee) {
		skillModFlags |= mod.MFlagMelee
	} else if utils.HasTrue(skillFlags, SkillFlagProjectile) {
		skillModFlags |= mod.MFlagProjectile
		skillFlags[SkillFlagChaining] = true
	}

	if utils.HasTrue(skillFlags, SkillFlagArea) {
		skillModFlags |= mod.MFlagArea
	}

	// Build skill keyword flag set
	skillKeywordFlags := mod.KeywordFlag(0)

	if skillFlags[SkillFlagHit] {
		skillKeywordFlags |= mod.KeywordFlagHit
	}

	if skillTypes[data.SkillTypeAura] {
		skillKeywordFlags |= mod.KeywordFlagAura
	}

	if skillTypes[data.SkillTypeHex] || skillTypes[data.SkillTypeMark] {
		skillKeywordFlags |= mod.KeywordFlagCurse
	}

	if skillTypes[data.SkillTypeWarcry] {
		skillKeywordFlags |= mod.KeywordFlagWarcry
	}

	if skillTypes[data.SkillTypeMovement] {
		skillKeywordFlags |= mod.KeywordFlagMovement
	}

	if skillTypes[data.SkillTypeVaal] {
		skillKeywordFlags |= mod.KeywordFlagVaal
	}

	if skillTypes[data.SkillTypeLightning] {
		skillKeywordFlags |= mod.KeywordFlagLightning
	}

	if skillTypes[data.SkillTypeCold] {
		skillKeywordFlags |= mod.KeywordFlagCold
	}

	if skillTypes[data.SkillTypeFire] {
		skillKeywordFlags |= mod.KeywordFlagFire
	}

	if skillTypes[data.SkillTypeChaos] {
		skillKeywordFlags |= mod.KeywordFlagChaos
	}

	if skillFlags[SkillFlagWeapon1Attack] && activeSkill.Weapon1Flags&mod.MFlagBow != 0 {
		skillKeywordFlags |= mod.KeywordFlagBow
	}

	if skillFlags[SkillFlagBrand] {
		skillKeywordFlags |= mod.KeywordFlagBrand
	}

	if skillFlags[SkillFlagTotem] {
		skillKeywordFlags |= mod.KeywordFlagTotem
	} else if skillFlags[SkillFlagTrap] {
		skillKeywordFlags |= mod.KeywordFlagTrap
	} else if skillFlags[SkillFlagMine] {
		skillKeywordFlags |= mod.KeywordFlagMine
	} else {
		skillFlags[SkillFlagSelfCast] = true
	}

	if skillTypes[data.SkillTypeAttack] {
		skillKeywordFlags |= mod.KeywordFlagAttack
	}

	if skillTypes[data.SkillTypeSpell] && !skillFlags[SkillFlagCast] {
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

	// Build config structure for modifier searches
	if skillFlags[SkillFlagWeapon1Attack] {
		cond := utils.CopyMap(activeSkill.SkillCfg.SkillCond)
		cond["MainHandAttack"] = true
		activeSkill.Weapon1Cfg = &ListCfg{
			Flags:        utils.Ptr(skillModFlags | activeSkill.Weapon1Flags),
			KeywordFlags: activeSkill.SkillCfg.KeywordFlags,
			Source:       activeSkill.SkillCfg.Source,
			SkillStats:   activeSkill.SkillCfg.SkillStats,
			SkillCond:    cond,
		}
	}

	if skillFlags[SkillFlagWeapon2Attack] {
		cond := utils.CopyMap(activeSkill.SkillCfg.SkillCond)
		cond["OffHandAttack"] = true
		activeSkill.Weapon1Cfg = &ListCfg{
			Flags:        utils.Ptr(skillModFlags | activeSkill.Weapon2Flags),
			KeywordFlags: activeSkill.SkillCfg.KeywordFlags,
			Source:       activeSkill.SkillCfg.Source,
			SkillStats:   activeSkill.SkillCfg.SkillStats,
			SkillCond:    cond,
		}
	}

	// Initialise skill modifier list
	skillModList := NewModList()
	skillModList.Parent = activeSkill.Actor.ModDB
	activeSkill.SkillModList = skillModList
	activeSkill.BaseSkillModList = skillModList

	/*
		TODO -- Initialise skill modifier list
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

	// Add support gem modifiers to skill mod list
	for _, skillEffect := range activeSkill.EffectList {
		if skillEffect.GrantedEffect.Raw.IsSupport {
			CalcMergeSkillInstanceMods(env, skillModList, skillEffect, nil)
			level := raw2.GetCalculatedGrantedEffect(skillEffect.GrantedEffect.Raw).GetCalculatedLevels()[skillEffect.Level]
			if level.ManaMultiplier != nil {
				// TODO skillEffect.grantedEffect.modSource
				skillModList.AddMod(mod.NewFloat("SupportManaMultiplier", mod.TypeBase, *level.ManaMultiplier))
			}
			if level.ManaReservationPercent != nil {
				activeSkill.SkillData["ManaReservationPercent"] = *level.ManaReservationPercent
			}
			if level.Cooldown != nil {
				activeSkill.SkillData["Cooldown"] = *level.Cooldown
			}
		}
	}

	/*
		TODO -- Apply gem/quality modifiers from support gems
		for _, value in ipairs(skillModList:List(activeSkill.skillCfg, "SupportedGemProperty")) do
			if value.keyword == "active_skill" and activeSkill.activeEffect.gemData then
				activeEffect[value.key] = activeEffect[value.key] + value.value
			end
		end
	*/

	// Add active gem modifiers

	// TODO Minions
	// activeEffect.actorLevel = activeSkill.actor.minionData and activeSkill.actor.level
	CalcMergeSkillInstanceMods(env, skillModList, activeEffect, skillModList.List(activeSkill.SkillCfg, "ExtraSkillStat"))
	activeEffect.GrantedEffectLevel = raw2.GetCalculatedGrantedEffect(activeGrantedEffect.Raw).GetCalculatedLevels()[activeEffect.Level]

	// Add extra modifiers from granted effect level
	level := activeEffect.GrantedEffectLevel
	if level.CritChance != nil {
		activeSkill.SkillData["CritChance"] = *level.CritChance
	}

	/*
		TODO damageMultiplier
		if level.damageMultiplier then
			skillModList:NewMod("Damage", "MORE", level.damageMultiplier, activeEffect.grantedEffect.modSource, ModFlag.Attack)
		end
	*/

	if level.AttackTime != nil {
		activeSkill.SkillData["AttackTime"] = *level.AttackTime
	}

	if level.AttackSpeedMultiplier != nil {
		// TODO activeEffect.grantedEffect.modSource
		skillModList.AddMod(mod.NewFloat("Speed", mod.TypeMore, *level.AttackSpeedMultiplier).Flag(mod.MFlagAttack))
	}

	if level.Cooldown != nil {
		activeSkill.SkillData["Cooldown"] = *level.Cooldown
	}

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

	// Extract skill data
	for _, value := range utils.CastSlice[*mod.SkillData](env.ModDB.List(activeSkill.SkillCfg, "SkillData")) {
		activeSkill.SkillData[value.Key] = value.Value
	}
	for _, value := range utils.CastSlice[*mod.SkillData](skillModList.List(activeSkill.SkillCfg, "SkillData")) {
		activeSkill.SkillData[value.Key] = value.Value
	}

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

func getWeaponFlags(env *Environment, weaponData map[string]interface{}, weaponTypes [][]data.ItemClassName) (mod.MFlag, *data.WeaponTypeInfo) {
	if _, ok := weaponData["type"]; !ok {
		return 0, nil
	}

	info := data.WeaponTypes[data.ItemClassName(weaponData["type"].(string))]

	if info == nil {
		return 0, nil
	}

	if weaponTypes != nil {
		/*
			for _, types in ipairs(weaponTypes) do
				if not types[weaponData.type] and
				(not weaponData.countsAsAll1H or not (types["Claw"] or types["Dagger"] or types["One Handed Axe"] or types["One Handed Mace"] or types["One Handed Sword"])) then
					return nil, info
				end
			end
		*/
	}

	flags := info.ModFlag
	if utils.HasTrue(weaponData, "CountsAsAll1H") {
		flags = mod.MFlagAxe | mod.MFlagClaw | mod.MFlagDagger | mod.MFlagMace | mod.MFlagSword
	}

	if weaponData["type"] != "None" {
		flags |= mod.MFlagWeapon
		if info.OneHand {
			flags |= mod.MFlagWeapon1H
		} else {
			flags |= mod.MFlagWeapon2H
		}

		if info.Melee {
			flags |= mod.MFlagWeaponMelee
		} else {
			flags |= mod.MFlagWeaponRanged
		}
	}

	return flags, info
}
