package calculator

import (
	"math"

	"go-pob/calculator/mod"
	"go-pob/data"
	"go-pob/utils"
)

func calcDamage(activeSkill *ActiveSkill, output map[string]float64, cfg *ListCfg, breakdown interface{}, damageType data.DamageType, typeFlags int, convDst *data.DamageType) (float64, float64) {
	typeFlags = typeFlags | data.DamageTypeFlags[damageType]

	// Calculate conversions
	addMin := float64(0)
	addMax := float64(0)

	for _, otherType := range data.DamageType("").Values() {
		if otherType == damageType {
			// Damage can only be converted from damage types that precede this one in the conversion sequence, so stop here
			break
		}

		convMult := activeSkill.ConversionTable[otherType].Targets[damageType]
		if convMult > 0 {
			// Damage is being converted/gained from the other damage type
			min, max := calcDamage(activeSkill, output, cfg, breakdown, otherType, typeFlags, &damageType)
			addMin += min * convMult
			addMax += max * convMult
		}
	}

	if addMin != 0 && addMax != 0 {
		addMin = math.Round(addMin)
		addMax = math.Round(addMax)
	}

	baseMin := output[string(damageType)+"MinBase"]
	baseMax := output[string(damageType)+"MaxBase"]

	if baseMin == 0 && baseMax == 0 {
		// No base damage for this type, don't need to calculate modifiers
		/*
			TODO Breakdown
			if breakdown and (addMin ~= 0 or addMax ~= 0) then
				t_insert(breakdown.damageTypes, {
					source = damageType,
					convSrc = (addMin ~= 0 or addMax ~= 0) and (addMin .. " to " .. addMax),
					total = addMin .. " to " .. addMax,
					convDst = convDst and s_format("%d%% to %s", conversionTable[damageType][convDst] * 100, convDst),
				})
			end
		*/
		return addMin, addMax
	}

	// Combine modifiers
	modNames := data.DamageStatsForType(typeFlags)
	inc := 1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, cfg, modNames...)/100
	more := math.Floor(activeSkill.SkillModList.More(cfg, modNames...)*100+0.50000001) / 100
	moreMinDamage := activeSkill.SkillModList.More(cfg, "Min"+string(damageType)+"Damage")
	moreMaxDamage := activeSkill.SkillModList.More(cfg, "Max"+string(damageType)+"Damage")

	/*
		TODO Breakdown
		if breakdown then
			t_insert(breakdown.damageTypes, {
				source = damageType,
				base = baseMin .. " to " .. baseMax,
				inc = (inc ~= 1 and "x "..inc),
				more = (more ~= 1 and "x "..more),
				convSrc = (addMin ~= 0 or addMax ~= 0) and (addMin .. " to " .. addMax),
				total = (round(baseMin * inc * more) + addMin) .. " to " .. (round(baseMax * inc * more) + addMax),
				convDst = convDst and conversionTable[damageType][convDst] > 0 and s_format("%d%% to %s", conversionTable[damageType][convDst] * 100, convDst),
			})
		end
	*/

	return math.Round(((baseMin * inc * more) + addMin) * moreMinDamage),
		math.Round(((baseMax * inc * more) + addMax) * moreMaxDamage)
}

func CalculateOffence(env *Environment, actor *Actor, activeSkill *ActiveSkill) {
	/*
		TODO ...
		local modDB = actor.modDB
		local enemyDB = actor.enemy.modDB
		local output = actor.output
		local breakdown = actor.breakdown
		local skillModList = activeSkill.skillModList
		local skillData = activeSkill.skillData
		local skillFlags = activeSkill.skillFlags
		local skillCfg = activeSkill.skillCfg
		if skillData.showAverage then
			skillFlags.showAverage = true
		else
			skillFlags.notAverage = true
		end

		if skillFlags.disable then
			-- Skill is disabled
			output.CombinedDPS = 0
			return
		end
	*/
	/*
		TODO calcAreaOfEffect
		local function calcAreaOfEffect(skillModList, skillCfg, skillData, skillFlags, output, breakdown)
			local incArea, moreArea = calcLib.mods(skillModList, skillCfg, "AreaOfEffect")
			output.AreaOfEffectMod = round(round(incArea * moreArea, 10), 2)
			if skillData.radiusIsWeaponRange then
				local range = 0
				if skillFlags.weapon1Attack then
					range = m_max(range, actor.weaponRange1)
				end
				if skillFlags.weapon2Attack then
					range = m_max(range, actor.weaponRange2)
				end
				skillData.radius = range + 2
			end
			if skillData.radius then
				skillFlags.area = true
				local baseRadius = skillData.radius + (skillData.radiusExtra or 0) + skillModList:Sum("BASE", skillCfg, "AreaOfEffect")
				output.AreaOfEffectRadius = calcRadius(baseRadius, output.AreaOfEffectMod)
				if breakdown then
					local incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint = calcRadiusBreakpoints(baseRadius, incArea, moreArea)
					breakdown.AreaOfEffectRadius = breakdown.area(baseRadius, output.AreaOfEffectMod, output.AreaOfEffectRadius, incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint, skillData.radiusLabel)
				end
				if skillData.radiusSecondary then
					local incAreaSecondary, moreAreaSecondary = calcLib.mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectSecondary")
					output.AreaOfEffectModSecondary = round(round(incAreaSecondary * moreAreaSecondary, 10), 2)
					baseRadius = skillData.radiusSecondary + (skillData.radiusExtra or 0)
					output.AreaOfEffectRadiusSecondary = calcRadius(baseRadius, output.AreaOfEffectModSecondary)
					if breakdown then
						local incAreaBreakpointSecondary, moreAreaBreakpointSecondary, redAreaBreakpointSecondary, lessAreaBreakpointSecondary
						if not skillData.projectileSpeedAppliesToMSAreaOfEffect then
							local incAreaBreakpointSecondary, moreAreaBreakpointSecondary, redAreaBreakpointSecondary, lessAreaBreakpointSecondary = calcRadiusBreakpoints(baseRadius, incAreaSecondary, moreAreaSecondary)
						end
						breakdown.AreaOfEffectRadiusSecondary = breakdown.area(baseRadius, output.AreaOfEffectModSecondary, output.AreaOfEffectRadiusSecondary, incAreaBreakpointSecondary, moreAreaBreakpointSecondary, redAreaBreakpointSecondary, lessAreaBreakpointSecondary, skillData.radiusSecondaryLabel)
					end
				end
				if skillData.radiusTertiary then
					local incAreaTertiary, moreAreaTertiary = calcLib.mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectTertiary")
					output.AreaOfEffectModTertiary = round(round(incAreaTertiary * moreAreaTertiary, 10), 2)
					baseRadius = skillData.radiusTertiary + (skillData.radiusExtra or 0)
					if skillData.projectileSpeedAppliesToMSAreaOfEffect then
						local incSpeedTertiary, moreSpeedTertiary = calcLib.mods(skillModList, skillCfg, "ProjectileSpeed")
						output.SpeedModTertiary = round(round(incSpeedTertiary * moreSpeedTertiary, 10), 2)
						output.AreaOfEffectRadiusTertiary = calcMoltenStrikeTertiaryRadius(baseRadius, skillData.radiusSecondary, output.AreaOfEffectModTertiary, output.SpeedModTertiary)
						if breakdown then
							setMoltenStrikeTertiaryRadiusBreakdown(
								breakdown, skillData.radiusSecondary, baseRadius, skillData.radiusTertiaryLabel,
								incAreaTertiary, moreAreaTertiary, incSpeedTertiary, moreSpeedTertiary
							)
						end
					else
						output.AreaOfEffectRadiusTertiary = calcRadius(baseRadius, output.AreaOfEffectModTertiary)
						if breakdown then
							local incAreaBreakpointTertiary, moreAreaBreakpointTertiary, redAreaBreakpointTertiary, lessAreaBreakpointTertiary = calcRadiusBreakpoints(baseRadius, incAreaTertiary, moreAreaTertiary)
							breakdown.AreaOfEffectRadiusTertiary = breakdown.area(baseRadius, output.AreaOfEffectModTertiary, output.AreaOfEffectRadiusTertiary, incAreaBreakpointTertiary, moreAreaBreakpointTertiary, redAreaBreakpointTertiary, lessAreaBreakpointTertiary, skillData.radiusTertiaryLabel)
						end
					end
				end
			end
			if breakdown then
				breakdown.AreaOfEffectMod = { }
				breakdown.multiChain(breakdown.AreaOfEffectMod, {
					{ "%.2f ^8(increased/reduced)", 1 + skillModList:Sum("INC", skillCfg, "AreaOfEffect") / 100 },
					{ "%.2f ^8(more/less)", skillModList:More(skillCfg, "AreaOfEffect") },
					total = s_format("= %.2f", output.AreaOfEffectMod),
				})
			end
		end
	*/
	/*
		TODO runSkillFunc
		local function runSkillFunc(name)
			local func = activeSkill.activeEffect.grantedEffect[name]
			if func then
				func(activeSkill, output, breakdown)
			end
		end

		runSkillFunc("initialFunc")
	*/
	/*
		TODO isTriggered
		local isTriggered = skillData.triggeredWhileChannelling or skillData.triggeredByCoC or skillData.triggeredByMeleeKill or skillData.triggeredByCospris or skillData.triggeredByMjolner or skillData.triggeredByUnique or skillData.triggeredByFocus or skillData.triggeredByCraft or skillData.triggeredByManaSpent or skillData.triggeredByParentAttack
		skillCfg.skillCond["SkillIsTriggered"] = skillData.triggered or isTriggered
		if skillCfg.skillCond["SkillIsTriggered"] then
			skillFlags.triggered = true
		end
		skillCfg.skillCond["SkillIsFocused"] = skillData.triggeredByFocus
		if skillCfg.skillCond["SkillIsFocused"] then
			skillFlags.focused = true
		end
	*/
	/*
		TODO -- Update skill data
		for _, value in ipairs(skillModList:List(skillCfg, "SkillData")) do
			if value.merge == "MAX" then
				skillData[value.key] = m_max(value.value, skillData[value.key] or 0)
			else
				skillData[value.key] = value.value
			end
		end
	*/
	/*
		TODO -- Add addition stat bonuses
		if skillModList:Flag(nil, "IronGrip") then
			skillModList:NewMod("PhysicalDamage", "INC", actor.strDmgBonus or 0, "Strength", bor(ModFlag.Attack, ModFlag.Projectile))
		end
		if skillModList:Flag(nil, "IronWill") then
			skillModList:NewMod("Damage", "INC", actor.strDmgBonus or 0, "Strength", ModFlag.Spell)
		end

		if skillModList:Flag(nil, "TransfigurationOfBody") then
			skillModList:NewMod("Damage", "INC", m_floor(skillModList:Sum("INC", nil, "Life") * data.misc.Transfiguration), "Transfiguration of Body", ModFlag.Attack)
		end
		if skillModList:Flag(nil, "TransfigurationOfMind") then
			skillModList:NewMod("Damage", "INC", m_floor(skillModList:Sum("INC", nil, "Mana") * data.misc.Transfiguration), "Transfiguration of Mind")
		end
		if skillModList:Flag(nil, "TransfigurationOfSoul") then
			skillModList:NewMod("Damage", "INC", m_floor(skillModList:Sum("INC", nil, "EnergyShield") * data.misc.Transfiguration), "Transfiguration of Soul", ModFlag.Spell)
		end

		if modDB:Flag(nil, "Elusive") and skillModList:Flag(nil, "SupportedByNightblade") then
			local elusiveEffect = output.ElusiveEffectMod / 100
			local nightbladeMulti = skillModList:Sum("BASE", nil, "NightbladeElusiveCritMultiplier")
			skillModList:NewMod("CritMultiplier", "BASE", m_floor(nightbladeMulti * elusiveEffect), "Nightblade")
		end
	*/
	/*
		TODO -- additional charge based modifiers
		if skillModList:Flag(nil, "UseEnduranceCharges") and skillModList:Flag(nil, "EnduranceChargesConvertToBrutalCharges") then
			local tripleDmgChancePerEndurance = modDB:Sum("BASE", nil, "PerBrutalTripleDamageChance")
			modDB:NewMod("TripleDamageChance", "BASE", tripleDmgChancePerEndurance, { type = "Multiplier", var = "BrutalCharge" } )
		end
		if skillModList:Flag(nil, "UseFrenzyCharges") and skillModList:Flag(nil, "FrenzyChargesConvertToAfflictionCharges") then
			local dmgPerAffliction = modDB:Sum("BASE", nil, "PerAfflictionAilmentDamage")
			local effectPerAffliction = modDB:Sum("BASE", nil, "PerAfflictionNonDamageEffect")
			modDB:NewMod("Damage", "MORE", dmgPerAffliction, "Affliction Charges", 0, KeywordFlag.Ailment, { type = "Multiplier", var = "AfflictionCharge" } )
			modDB:NewMod("EnemyChillEffect", "MORE", effectPerAffliction, "Affliction Charges", { type = "Multiplier", var = "AfflictionCharge" } )
			modDB:NewMod("EnemyShockEffect", "MORE", effectPerAffliction, "Affliction Charges", { type = "Multiplier", var = "AfflictionCharge" } )
			modDB:NewMod("EnemyFreezeEffect", "MORE", effectPerAffliction, "Affliction Charges", { type = "Multiplier", var = "AfflictionCharge" } )
			modDB:NewMod("EnemyScorchEffect", "MORE", effectPerAffliction, "Affliction Charges", { type = "Multiplier", var = "AfflictionCharge" } )
			modDB:NewMod("EnemyBrittleEffect", "MORE", effectPerAffliction, "Affliction Charges", { type = "Multiplier", var = "AfflictionCharge" } )
			modDB:NewMod("EnemySapEffect", "MORE", effectPerAffliction, "Affliction Charges", { type = "Multiplier", var = "AfflictionCharge" } )
		end
	*/
	/*
		TODO -- set other limits
		output.ActiveTrapLimit = skillModList:Sum("BASE", skillCfg, "ActiveTrapLimit")
		output.ActiveMineLimit = skillModList:Sum("BASE", skillCfg, "ActiveMineLimit")

	*/
	/*
		TODO -- set flask scaling
		output.LifeFlaskRecovery = env.itemModDB.multipliers["LifeFlaskRecovery"]

		if skillModList:Flag(nil, "Condition:EnergyBladeActive") then
			local dmgMod = calcLib.mod(skillModList, skillCfg, "EnergyBladeDamage")
			local critMod = calcLib.mod(skillModList, skillCfg, "EnergyBladeCritChance")
			local speedMod = calcLib.mod(skillModList, skillCfg, "EnergyBladeAttackSpeed")
			for slotName, weaponData in pairs({ ["Weapon 1"] = "weaponData1", ["Weapon 2"] = "weaponData2" }) do
				if actor.itemList[slotName] and actor.itemList[slotName].weaponData and actor.itemList[slotName].weaponData[1] then
					actor[weaponData].CritChance = actor[weaponData].CritChance * critMod
					actor[weaponData].AttackRate = actor[weaponData].AttackRate * speedMod
					for _, damageType in ipairs(dmgTypeList) do
						actor[weaponData][damageType.."Min"] = (actor[weaponData][damageType.."Min"] or 0) + m_floor(skillModList:Sum("BASE", skillCfg, "EnergyBladeMin"..damageType) * dmgMod)
						actor[weaponData][damageType.."Max"] = (actor[weaponData][damageType.."Max"] or 0) + m_floor(skillModList:Sum("BASE", skillCfg, "EnergyBladeMax"..damageType) * dmgMod)
					end
				end
			end
		end
	*/
	/*
		TODO -- account for Battlemage
		-- Note: we check conditions of Main Hand weapon using actor.itemList as actor.weaponData1 is populated with unarmed values when no weapon slotted.
		if skillModList:Flag(nil, "WeaponDamageAppliesToSpells") and actor.itemList["Weapon 1"] and actor.itemList["Weapon 1"].weaponData and actor.itemList["Weapon 1"].weaponData[1] then
			-- the multiplier below exist for future possible extension of Battlemage modifiers
			local multiplier = (skillModList:Max(skillCfg, "ImprovedWeaponDamageAppliesToSpells") or 100) / 100
			for _, damageType in ipairs(dmgTypeList) do
				skillModList:NewMod(damageType.."Min", "BASE", (actor.weaponData1[damageType.."Min"] or 0) * multiplier, "Battlemage", ModFlag.Spell)
				skillModList:NewMod(damageType.."Max", "BASE", (actor.weaponData1[damageType.."Max"] or 0) * multiplier, "Battlemage", ModFlag.Spell)
			end
		end
		if skillModList:Flag(nil, "MinionDamageAppliesToPlayer") then
			-- Minion Damage conversion from Spiritual Aid and The Scourge
			local multiplier = (skillModList:Max(skillCfg, "ImprovedMinionDamageAppliesToPlayer") or 100) / 100
			for _, value in ipairs(skillModList:List(skillCfg, "MinionModifier")) do
				if value.mod.name == "Damage" and value.mod.type == "INC" then
					local mod = value.mod
					local modifiers = calcLib.getConvertedModTags(mod, multiplier, true)
					skillModList:NewMod("Damage", "INC", mod.value * multiplier, mod.source, mod.flags, mod.keywordFlags, unpack(modifiers))
				end
			end
		end
		if skillModList:Flag(nil, "MinionAttackSpeedAppliesToPlayer") then
			-- Minion Damage conversion from Spiritual Command
			local multiplier = (skillModList:Max(skillCfg, "ImprovedMinionAttackSpeedAppliesToPlayer") or 100) / 100
			-- Minion Attack Speed conversion from Spiritual Command
			for _, value in ipairs(skillModList:List(skillCfg, "MinionModifier")) do
				if value.mod.name == "Speed" and value.mod.type == "INC" and (value.mod.flags == 0 or band(value.mod.flags, ModFlag.Attack) ~= 0) then
					local modifiers = calcLib.getConvertedModTags(value.mod, multiplier, true)
					skillModList:NewMod("Speed", "INC", value.mod.value * multiplier, value.mod.source, ModFlag.Attack, value.mod.keywordFlags, unpack(modifiers))
				end
			end
		end
		if skillModList:Flag(nil, "SpellDamageAppliesToAttacks") then
			-- Spell Damage conversion from Crown of Eyes, Kinetic Bolt, and the Wandslinger notable
			local multiplier = (skillModList:Max(skillCfg, "ImprovedSpellDamageAppliesToAttacks") or 100) / 100
			for i, value in ipairs(skillModList:Tabulate("INC", { flags = ModFlag.Spell }, "Damage")) do
				local mod = value.mod
				if band(mod.flags, ModFlag.Spell) ~= 0 then
					local modifiers = calcLib.getConvertedModTags(mod, multiplier)
					skillModList:NewMod("Damage", "INC", mod.value * multiplier, mod.source, bor(band(mod.flags, bnot(ModFlag.Spell)), ModFlag.Attack), mod.keywordFlags, unpack(modifiers))
					if mod.source == "Strength" then -- Prevent double-dipping from converted strength's damage bonus
						skillModList:ReplaceMod("PhysicalDamage", "INC", 0, "Strength", ModFlag.Melee)
					end
				end
			end
		end
		if skillModList:Flag(nil, "CastSpeedAppliesToAttacks") then
			-- Get all increases for this; assumption is that multiple sources would not stack, so find the max
			local multiplier = (skillModList:Max(skillCfg, "ImprovedCastSpeedAppliesToAttacks") or 100) / 100
			for i, value in ipairs(skillModList:Tabulate("INC", { flags = ModFlag.Cast }, "Speed")) do
				local mod = value.mod
				-- Add a new mod for all mods that are cast only
				-- Replace this with a single mod for the sum?
				if band(mod.flags, ModFlag.Cast) ~= 0 then
					local modifiers = calcLib.getConvertedModTags(mod, multiplier)
					skillModList:NewMod("Speed", "INC", mod.value * multiplier, mod.source, bor(band(mod.flags, bnot(ModFlag.Cast)), ModFlag.Attack), mod.keywordFlags, unpack(modifiers))
				end
			end
		end
		if skillModList:Flag(nil, "ProjectileSpeedAppliesToBowDamage") then
			-- Bow mastery projectile speed to damage with bows conversion
			for i, value in ipairs(skillModList:Tabulate("INC", { }, "ProjectileSpeed")) do
				local mod = value.mod
				skillModList:NewMod("Damage", mod.type, mod.value, mod.source, bor(ModFlag.Bow, ModFlag.Hit), mod.keywordFlags, unpack(mod))
			end
		end
		if skillModList:Flag(nil, "ClawDamageAppliesToUnarmed") then
			-- Claw Damage conversion from Rigwald's Curse
			for i, value in ipairs(skillModList:Tabulate("INC", { flags = ModFlag.Claw, keywordFlags = KeywordFlag.Hit }, "Damage")) do
				local mod = value.mod
				if band(mod.flags, ModFlag.Claw) ~= 0 then
					skillModList:NewMod("Damage", mod.type, mod.value, mod.source, bor(band(mod.flags, bnot(ModFlag.Claw)), ModFlag.Unarmed, ModFlag.Melee), mod.keywordFlags, unpack(mod))
				end
			end
		end
		if skillModList:Flag(nil, "ClawAttackSpeedAppliesToUnarmed") then
			-- Claw Attack Speed conversion from Rigwald's Curse
			for i, value in ipairs(skillModList:Tabulate("INC", { flags = bor(ModFlag.Claw, ModFlag.Attack, ModFlag.Hit) }, "Speed")) do
				local mod = value.mod
				if band(mod.flags, ModFlag.Claw) ~= 0 and band(mod.flags, ModFlag.Attack) ~= 0 then
					skillModList:NewMod("Speed", mod.type, mod.value, mod.source, bor(band(mod.flags, bnot(ModFlag.Claw)), ModFlag.Unarmed), mod.keywordFlags, unpack(mod))
				end
			end
		end
		if skillModList:Flag(nil, "ClawCritChanceAppliesToUnarmed") then
			-- Claw Crit Chance conversion from Rigwald's Curse
			for i, value in ipairs(skillModList:Tabulate("INC", { flags = bor(ModFlag.Claw, ModFlag.Hit) }, "CritChance")) do
				local mod = value.mod
				if band(mod.flags, ModFlag.Claw) ~= 0 then
					skillModList:NewMod("CritChance", mod.type, mod.value, mod.source, bor(band(mod.flags, bnot(ModFlag.Claw)), ModFlag.Unarmed), mod.keywordFlags, unpack(mod))
				end
			end
		end
		if skillModList:Flag(nil, "ClawCritChanceAppliesToMinions") then
			-- Claw Crit Chance conversion from Law of the Wilds
			for i, value in ipairs(skillModList:Tabulate("INC", { flags = bor(ModFlag.Claw, ModFlag.Hit) }, "CritChance")) do
				local mod = value.mod
				if band(mod.flags, ModFlag.Claw) ~= 0 then
					env.minion.modDB:NewMod("CritChance", mod.type, mod.value, mod.source)
				end
			end
		end
		if skillModList:Flag(nil, "ClawCritMultiplierAppliesToMinions") then
			-- Claw Crit Multi conversion from Law of the Wilds
			for i, value in ipairs(skillModList:Tabulate("BASE", { flags = bor(ModFlag.Claw, ModFlag.Hit) }, "CritMultiplier")) do
				local mod = value.mod
				if band(mod.flags, ModFlag.Claw) ~= 0 then
					env.minion.modDB:NewMod("CritMultiplier", mod.type, mod.value, mod.source)
				end
			end
		end
		if skillModList:Flag(nil, "LightRadiusAppliesToAccuracy") then
			-- Light Radius conversion from Corona Solaris
			for i, value in ipairs(skillModList:Tabulate("INC",  { }, "LightRadius")) do
				local mod = value.mod
				skillModList:NewMod("Accuracy", "INC", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
			end
		end
		if skillModList:Flag(nil, "LightRadiusAppliesToAreaOfEffect") then
			-- Light Radius conversion from Wreath of Phrecia
			for i, value in ipairs(skillModList:Tabulate("INC",  { }, "LightRadius")) do
				local mod = value.mod
				skillModList:NewMod("AreaOfEffect", "INC", math.floor(mod.value / 2), mod.source, mod.flags, mod.keywordFlags, unpack(mod))
			end
		end
		if skillModList:Flag(nil, "LightRadiusAppliesToDamage") then
			-- Light Radius conversion from Wreath of Phrecia
			for i, value in ipairs(skillModList:Tabulate("INC",  { }, "LightRadius")) do
				local mod = value.mod
				skillModList:NewMod("Damage", "INC", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
			end
		end
		if skillModList:Flag(nil, "CastSpeedAppliesToTrapThrowingSpeed") then
			-- Cast Speed conversion from Slavedriver's Hand
			for i, value in ipairs(skillModList:Tabulate("INC", { flags = ModFlag.Cast }, "Speed")) do
				local mod = value.mod
				if (mod.flags == 0 or band(mod.flags, ModFlag.Cast) ~= 0) then
					skillModList:NewMod("TrapThrowingSpeed", "INC", mod.value, mod.source, band(mod.flags, bnot(ModFlag.Cast), bnot(ModFlag.Attack)), mod.keywordFlags, unpack(mod))
				end
			end
		end
		if skillData.arrowSpeedAppliesToAreaOfEffect then
			-- Arrow Speed conversion for Galvanic Arrow
			for i, value in ipairs(skillModList:Tabulate("INC", { flags = ModFlag.Bow }, "ProjectileSpeed")) do
				local mod = value.mod
				skillModList:NewMod("AreaOfEffect", "INC", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
			end
		end
		if skillModList:Flag(nil, "SequentialProjectiles") and not skillModList:Flag(nil, "OneShotProj") and not skillModList:Flag(nil,"NoAdditionalProjectiles") and not skillModList:Flag(nil, "TriggeredBySnipe") then
			-- Applies DPS multiplier based on projectile count
			skillData.dpsMultiplier = skillModList:Sum("BASE", skillCfg, "ProjectileCount")
		end
		if skillData.gainPercentBaseWandDamage then
			local mult = skillData.gainPercentBaseWandDamage / 100
			if actor.weaponData1.type == "Wand" and actor.weaponData2.type == "Wand" then
				for _, damageType in ipairs(dmgTypeList) do
					skillModList:NewMod(damageType.."Min", "BASE", ((actor.weaponData1[damageType.."Min"] or 0) + (actor.weaponData2[damageType.."Min"] or 0)) / 2 * mult, "Spellslinger")
					skillModList:NewMod(damageType.."Max", "BASE", ((actor.weaponData1[damageType.."Max"] or 0) + (actor.weaponData2[damageType.."Max"] or 0)) / 2 * mult, "Spellslinger")
				end
			elseif actor.weaponData1.type == "Wand" then
				for _, damageType in ipairs(dmgTypeList) do
					skillModList:NewMod(damageType.."Min", "BASE", (actor.weaponData1[damageType.."Min"] or 0) * mult, "Spellslinger")
					skillModList:NewMod(damageType.."Max", "BASE", (actor.weaponData1[damageType.."Max"] or 0) * mult, "Spellslinger")
				end
			elseif actor.weaponData2.type == "Wand" then
				for _, damageType in ipairs(dmgTypeList) do
					skillModList:NewMod(damageType.."Min", "BASE", (actor.weaponData2[damageType.."Min"] or 0) * mult, "Spellslinger")
					skillModList:NewMod(damageType.."Max", "BASE", (actor.weaponData2[damageType.."Max"] or 0) * mult, "Spellslinger")
				end
			end
		end
		if skillModList:Flag(nil, "TriggeredBySnipe") and activeSkill.skillTypes[SkillType.Triggerable] then
			skillModList:NewMod("Damage", "MORE", 165, "Config", ModFlag.Hit, { type = "Multiplier", var = "SnipeStage" } )
			skillModList:NewMod("Damage", "MORE", 120, "Config", ModFlag.Ailment, { type = "Multiplier", var = "SnipeStage" } )
		end
		if skillModList:Sum("BASE", nil, "CritMultiplierAppliesToDegen") > 0 then
			for i, value in ipairs(skillModList:Tabulate("BASE", skillCfg, "CritMultiplier")) do
				local mod = value.mod
				if mod.source ~= "Base" then -- The global base Crit Multi doesn't apply to ailments with Perfect Agony
					skillModList:NewMod("DotMultiplier", "BASE", m_floor(mod.value / 2), mod.source, ModFlag.Ailment, { type = "Condition", var = "CriticalStrike" }, unpack(mod))
				end
			end
		end
		if skillModList:Flag(nil, "HasSeals") and activeSkill.skillTypes[SkillType.CanRapidFire] then
			-- Applies DPS multiplier based on seals count
			output.SealCooldown = skillModList:Sum("BASE", skillCfg, "SealGainFrequency") / calcLib.mod(skillModList, skillCfg, "SealGainFrequency")
			output.SealMax = skillModList:Sum("BASE", skillCfg, "SealCount")
			output.TimeMaxSeals = output.SealCooldown * output.SealMax

			if not skillData.hitTimeOverride then
				if skillModList:Flag(nil, "UseMaxUnleash") then
					for i, value in ipairs(skillModList:Tabulate("INC",  { }, "MaxSealCrit")) do
						local mod = value.mod
						skillModList:NewMod("CritChance", "INC", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
					end
					env.player.mainSkill.skillData.dpsMultiplier = (1 + output.SealMax * calcLib.mod(skillModList, skillCfg, "SealRepeatPenalty"))
					env.player.mainSkill.skillData.hitTimeOverride = m_max(output.TimeMaxSeals, (1 / activeSkill.activeEffect.grantedEffect.castTime * 1.1 * calcLib.mod(skillModList, skillCfg, "Speed") * output.ActionSpeedMod))
				else
					env.player.mainSkill.skillData.dpsMultiplier = 1 + 1 / output.SealCooldown / (1 / activeSkill.activeEffect.grantedEffect.castTime * 1.1 * calcLib.mod(skillModList, skillCfg, "Speed") * output.ActionSpeedMod) * calcLib.mod(skillModList, skillCfg, "SealRepeatPenalty")
				end
			end

			if breakdown then
				breakdown.SealGainTime = { }
				breakdown.multiChain(breakdown.SealGainTime, {
					label = "Gain frequency:",
					base = s_format("%.2fs ^8(base gain frequency)", skillModList:Sum("BASE", skillCfg, "SealGainFrequency")),
					{ "%.2f ^8(increased/reduced gain frequency)", 1 + skillModList:Sum("INC", skillCfg, "SealGainFrequency") / 100 },
					{ "%.2f ^8(action speed modifier)",  output.ActionSpeedMod },
					total = s_format("= %.2fs ^8per Seal", output.SealCooldown),
				})
			end
		end
		if skillModList:Sum("BASE", skillCfg, "PhysicalDamageGainAsRandom", "PhysicalDamageConvertToRandom", "PhysicalDamageGainAsColdOrLightning") > 0 then
			skillFlags.randomPhys = true
			local physMode = env.configInput.physMode or "AVERAGE"
			for i, value in ipairs(skillModList:Tabulate("BASE", skillCfg, "PhysicalDamageGainAsRandom")) do
				local mod = value.mod
				local effVal = mod.value / 3
				if physMode == "AVERAGE" then
					skillModList:NewMod("PhysicalDamageGainAsFire", "BASE", effVal, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
					skillModList:NewMod("PhysicalDamageGainAsCold", "BASE", effVal, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
					skillModList:NewMod("PhysicalDamageGainAsLightning", "BASE", effVal, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
				elseif physMode == "FIRE" then
					skillModList:NewMod("PhysicalDamageGainAsFire", "BASE", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
				elseif physMode == "COLD" then
					skillModList:NewMod("PhysicalDamageGainAsCold", "BASE", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
				elseif physMode == "LIGHTNING" then
					skillModList:NewMod("PhysicalDamageGainAsLightning", "BASE", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
				end
			end
			for i, value in ipairs(skillModList:Tabulate("BASE", skillCfg, "PhysicalDamageConvertToRandom")) do
				local mod = value.mod
				local effVal = mod.value / 3
				if physMode == "AVERAGE" then
					skillModList:NewMod("PhysicalDamageConvertToFire", "BASE", effVal, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
					skillModList:NewMod("PhysicalDamageConvertToCold", "BASE", effVal, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
					skillModList:NewMod("PhysicalDamageConvertToLightning", "BASE", effVal, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
				elseif physMode == "FIRE" then
					skillModList:NewMod("PhysicalDamageConvertToFire", "BASE", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
				elseif physMode == "COLD" then
					skillModList:NewMod("PhysicalDamageConvertToCold", "BASE", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
				elseif physMode == "LIGHTNING" then
					skillModList:NewMod("PhysicalDamageConvertToLightning", "BASE", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
				end
			end
			for i, value in ipairs(skillModList:Tabulate("BASE", skillCfg, "PhysicalDamageGainAsColdOrLightning")) do
				local mod = value.mod
				local effVal = mod.value / 2
				if physMode == "AVERAGE" or physMode == "FIRE" then
					skillModList:NewMod("PhysicalDamageGainAsCold", "BASE", effVal, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
					skillModList:NewMod("PhysicalDamageGainAsLightning", "BASE", effVal, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
				elseif physMode == "COLD" then
					skillModList:NewMod("PhysicalDamageGainAsCold", "BASE", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
				elseif physMode == "LIGHTNING" then
					skillModList:NewMod("PhysicalDamageGainAsLightning", "BASE", mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
				end
			end
		end

		local isAttack = skillFlags.attack

		runSkillFunc("preSkillTypeFunc")
	*/
	/*
		TODO -- Calculate skill type stats
		if skillFlags.minion then
			if activeSkill.minion and activeSkill.minion.minionData.limit then
				output.ActiveMinionLimit = m_floor(calcLib.val(skillModList, activeSkill.minion.minionData.limit, skillCfg))
			end
		end
		if skillFlags.chaining then
			if skillModList:Flag(skillCfg, "CannotChain") then
				output.ChainMaxString = "Cannot chain"
			else
				output.ChainMax = skillModList:Sum("BASE", skillCfg, "ChainCountMax", not skillFlags.projectile and "BeamChainCountMax" or nil)
				output.ChainMaxString = output.ChainMax
				output.Chain = m_min(output.ChainMax, skillModList:Sum("BASE", skillCfg, "ChainCount"))
				output.ChainRemaining = m_max(0, output.ChainMax - output.Chain)
			end
		end
		if skillFlags.projectile then
			if skillModList:Flag(nil, "PointBlank") then
				skillModList:NewMod("Damage", "MORE", 30, "Point Blank", bor(ModFlag.Attack, ModFlag.Projectile), { type = "DistanceRamp", ramp = {{10,1},{35,0},{150,-1}} })
			end
			if skillModList:Flag(nil, "FarShot") then
				skillModList:NewMod("Damage", "MORE", 100, "Far Shot", bor(ModFlag.Attack, ModFlag.Projectile), { type = "DistanceRamp", ramp = {{10, -0.2}, {35, 0}, {70, 0.6}} })
			end
			if skillModList:Flag(skillCfg, "NoAdditionalProjectiles") then
				output.ProjectileCount = 1
			else
				local projBase = skillModList:Sum("BASE", skillCfg, "ProjectileCount")
				local projMore = skillModList:More(skillCfg, "ProjectileCount")
				output.ProjectileCount = m_floor(projBase * projMore)
			end
			if skillModList:Flag(skillCfg, "AdditionalProjectilesAddBouncesInstead") then
				local projBase = skillModList:Sum("BASE", skillCfg, "ProjectileCount") + skillModList:Sum("BASE", skillCfg, "BounceCount") - 1
				local projMore = skillModList:More(skillCfg, "ProjectileCount")
				output.BounceCount = m_floor(projBase * projMore)
			end
			if skillModList:Flag(skillCfg, "CannotFork") then
				output.ForkCountString = "Cannot fork"
			elseif skillModList:Flag(skillCfg, "ForkOnce") then
				skillFlags.forking = true
				if skillModList:Flag(skillCfg, "ForkTwice") then
					output.ForkCountMax = m_min(skillModList:Sum("BASE", skillCfg, "ForkCountMax"), 2)
				else
					output.ForkCountMax = m_min(skillModList:Sum("BASE", skillCfg, "ForkCountMax"), 1)
				end
				output.ForkedCount = m_min(output.ForkCountMax, skillModList:Sum("BASE", skillCfg, "ForkedCount"))
				output.ForkCountString = output.ForkCountMax
				output.ForkRemaining = m_max(0, output.ForkCountMax - output.ForkedCount)
			else
				output.ForkCountString = "0"
			end
			if skillModList:Flag(skillCfg, "CannotPierce") then
				output.PierceCount = 0
				output.PierceCountString = "Cannot pierce"
			else
				if skillModList:Flag(skillCfg, "PierceAllTargets") or enemyDB:Flag(nil, "AlwaysPierceSelf") then
					output.PierceCount = 100
					output.PierceCountString = "All targets"
				else
					output.PierceCount = skillModList:Sum("BASE", skillCfg, "PierceCount")
					output.PierceCountString = output.PierceCount
				end
				if output.PierceCount > 0 then
					skillFlags.piercing = true
				end
				output.PiercedCount = m_min(output.PierceCount, skillModList:Sum("BASE", skillCfg, "PiercedCount"))
			end
			output.ProjectileSpeedMod = calcLib.mod(skillModList, skillCfg, "ProjectileSpeed")
			if breakdown then
				breakdown.ProjectileSpeedMod = breakdown.mod(skillModList, skillCfg, "ProjectileSpeed")
			end
		end
		if skillFlags.melee then
			if skillFlags.weapon1Attack then
				actor.weaponRange1 = (actor.weaponData1.range and actor.weaponData1.range + skillModList:Sum("BASE", activeSkill.weapon1Cfg, "MeleeWeaponRange")) or (6 + skillModList:Sum("BASE", skillCfg, "UnarmedRange"))
			end
			if skillFlags.weapon2Attack then
				actor.weaponRange2 = (actor.weaponData2.range and actor.weaponData2.range + skillModList:Sum("BASE", activeSkill.weapon2Cfg, "MeleeWeaponRange")) or (6 + skillModList:Sum("BASE", skillCfg, "UnarmedRange"))
			end
			if activeSkill.skillTypes[SkillType.MeleeSingleTarget] then
				local range = 100
				if skillFlags.weapon1Attack then
					range = m_min(range, actor.weaponRange1)
				end
				if skillFlags.weapon2Attack then
					range = m_min(range, actor.weaponRange2)
				end
				output.WeaponRange = range + 2
				if breakdown then
					breakdown.WeaponRange = {
						radius = output.WeaponRange
					}
				end
			end
		end
		if skillFlags.area or skillData.radius or (skillFlags.mine and activeSkill.skillTypes[SkillType.Aura]) then
			calcAreaOfEffect(skillModList, skillCfg, skillData, skillFlags, output, breakdown)
		end
		if activeSkill.skillTypes[SkillType.Aura] then
			output.AuraEffectMod = calcLib.mod(skillModList, skillCfg, "AuraEffect")
			if breakdown then
				breakdown.AuraEffectMod = breakdown.mod(skillModList, skillCfg, "AuraEffect")
			end
		end
		if activeSkill.skillTypes[SkillType.HasReservation] and not activeSkill.skillTypes[SkillType.ReservationBecomesCost] then
			for _, pool in ipairs({"Life", "Mana"}) do
				output[pool .. "ReservedMod"] = 0
				if calcLib.mod(skillModList, skillCfg, "SupportManaMultiplier") > 0 and calcLib.mod(skillModList, skillCfg, pool .. "Reserved", "Reserved") > 0 then
					output[pool .. "ReservedMod"] = calcLib.mod(skillModList, skillCfg, pool .. "Reserved", "Reserved") * calcLib.mod(skillModList, skillCfg, "SupportManaMultiplier") / m_max(0, calcLib.mod(skillModList, skillCfg, pool .. "ReservationEfficiency", "ReservationEfficiency"))
				end
				if breakdown then
					local inc = skillModList:Sum("INC", skillCfg, pool .. "Reserved", "Reserved", "SupportManaMultiplier")
					local more = skillModList:More(skillCfg, pool .. "Reserved", "Reserved", "SupportManaMultiplier")
					if inc ~= 0 and more ~= 1 then
						breakdown[pool .. "ReservedMod"] = {
							s_format("%.2f ^8(increased/reduced)", 1 + inc/100),
							s_format("x %.2f ^8(more/less)", more),
							s_format("/ %.2f ^8(reservation efficiency)", calcLib.mod(skillModList, skillCfg, pool .. "ReservationEfficiency", "ReservationEfficiency")),
							s_format("= %.2f", output[pool .. "ReservedMod"]),
						}
					end
				end
			end
		end
		if activeSkill.skillTypes[SkillType.Hex] or activeSkill.skillTypes[SkillType.Mark] then
			output.CurseEffectMod = calcLib.mod(skillModList, skillCfg, "CurseEffect")
			if breakdown then
				breakdown.CurseEffectMod = breakdown.mod(skillModList, skillCfg, "CurseEffect")
			end
		end
		if (skillFlags.trap or skillFlags.mine) and not (skillData.trapCooldown or skillData.cooldown) then
			skillFlags.notAverage = true
			skillFlags.showAverage = false
			skillData.showAverage = false
		end
		if skillFlags.trap then
			local baseSpeed = 1 / skillModList:Sum("BASE", skillCfg, "TrapThrowingTime")
			local timeMod = calcLib.mod(skillModList, skillCfg, "SkillTrapThrowingTime")
			if timeMod > 0 then
				baseSpeed = baseSpeed * (1 / timeMod)
			end
			output.TrapThrowingSpeed = baseSpeed * calcLib.mod(skillModList, skillCfg, "TrapThrowingSpeed") * output.ActionSpeedMod
			output.TrapThrowingSpeed = m_min(output.TrapThrowingSpeed, data.misc.ServerTickRate)
			output.TrapThrowingTime = 1 / output.TrapThrowingSpeed
			skillData.timeOverride = output.TrapThrowingTime
			if breakdown then
				breakdown.TrapThrowingSpeed = { }
				breakdown.multiChain(breakdown.TrapThrowingSpeed, {
					label = "Throwing rate:",
					base = s_format("%.2f ^8(base throwing rate)", baseSpeed),
					{ "%.2f ^8(increased/reduced throwing speed)", 1 + skillModList:Sum("INC", skillCfg, "TrapThrowingSpeed") / 100 },
					{ "%.2f ^8(more/less throwing speed)", skillModList:More(skillCfg, "TrapThrowingSpeed") },
					{ "%.2f ^8(action speed modifier)",  output.ActionSpeedMod },
					total = s_format("= %.2f ^8per second", output.TrapThrowingSpeed),
				})
			end
			if breakdown and timeMod > 0 then
				breakdown.TrapThrowingTime = { }
				breakdown.multiChain(breakdown.TrapThrowingTime, {
					label = "Throwing time:",
					base = s_format("%.2f ^8(base throwing time)", 1 / (output.TrapThrowingSpeed * timeMod)),
					{ "%.2f ^8(total modifier)", timeMod },
					total = s_format("= %.2f ^8seconds per throw", output.TrapThrowingTime),
				})
			end

			local baseCooldown = skillData.trapCooldown or skillData.cooldown
			if baseCooldown then
				output.TrapCooldown = baseCooldown / calcLib.mod(skillModList, skillCfg, "CooldownRecovery")
				output.TrapCooldown = m_ceil(output.TrapCooldown * data.misc.ServerTickRate) / data.misc.ServerTickRate
				if breakdown then
					breakdown.TrapCooldown = {
						s_format("%.2fs ^8(base)", skillData.trapCooldown or skillData.cooldown or 4),
						s_format("/ %.2f ^8(increased/reduced cooldown recovery)", 1 + skillModList:Sum("INC", skillCfg, "CooldownRecovery") / 100),
						s_format("rounded up to nearest server tick"),
						s_format("= %.3fs", output.TrapCooldown)
					}
				end
			end
			local incArea, moreArea = calcLib.mods(skillModList, skillCfg, "TrapTriggerAreaOfEffect")
			local areaMod = round(round(incArea * moreArea, 10), 2)
			output.TrapTriggerRadius = calcRadius(data.misc.TrapTriggerRadiusBase, areaMod)
			if breakdown then
				local incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint = calcRadiusBreakpoints(data.misc.TrapTriggerRadiusBase, incArea, moreArea)
				breakdown.TrapTriggerRadius = breakdown.area(data.misc.TrapTriggerRadiusBase, areaMod, output.TrapTriggerRadius, incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint)
			end
		elseif skillData.cooldown then
			output.Cooldown = calcSkillCooldown(skillModList, skillCfg, skillData)
			if breakdown then
				breakdown.Cooldown = {
					s_format("%.2fs ^8(base)", skillData.cooldown + skillModList:Sum("BASE", skillCfg, "CooldownRecovery")),
					s_format("/ %.2f ^8(increased/reduced cooldown recovery)", 1 + skillModList:Sum("INC", skillCfg, "CooldownRecovery") / 100),
					s_format("rounded up to nearest server tick"),
					s_format("= %.3fs", output.Cooldown)
				}
			end
		end
		if skillFlags.mine then
			local baseSpeed = 1 / skillModList:Sum("BASE", skillCfg, "MineLayingTime")
			local timeMod = calcLib.mod(skillModList, skillCfg, "SkillMineThrowingTime")
			if timeMod > 0 then
				baseSpeed = baseSpeed * (1 / timeMod)
			end
			output.MineLayingSpeed = baseSpeed * calcLib.mod(skillModList, skillCfg, "MineLayingSpeed") * output.ActionSpeedMod
			output.MineLayingSpeed = m_min(output.MineLayingSpeed, data.misc.ServerTickRate)
			output.MineLayingTime = 1 / output.MineLayingSpeed
			skillData.timeOverride = output.MineLayingTime
			if breakdown then
				breakdown.MineLayingTime = { }
				breakdown.multiChain(breakdown.MineLayingTime, {
					label = "Throwing rate:",
					base = s_format("%.2f ^8(base throwing rate)", baseSpeed),
					{ "%.2f ^8(increased/reduced throwing speed)", 1 + skillModList:Sum("INC", skillCfg, "MineLayingSpeed") / 100 },
					{ "%.2f ^8(more/less throwing speed)", skillModList:More(skillCfg, "MineLayingSpeed") },
					{ "%.2f ^8(action speed modifier)",  output.ActionSpeedMod },
					total = s_format("= %.2f ^8per second", output.MineLayingSpeed),
				})
			end
			if breakdown and timeMod > 0 then
				breakdown.MineThrowingTime = { }
				breakdown.multiChain(breakdown.MineThrowingTime, {
				label = "Throwing time:",
					base = s_format("%.2f ^8(base throwing time)", 1 / (output.MineLayingSpeed * timeMod)),
					{ "%.2f ^8(total modifier)", timeMod },
					total = s_format("= %.2f ^8seconds per throw", output.MineLayingTime),
				})
			end

			local incArea, moreArea = calcLib.mods(skillModList, skillCfg, "MineDetonationAreaOfEffect")
			local areaMod = round(round(incArea * moreArea, 10), 2)
			output.MineDetonationRadius = calcRadius(data.misc.MineDetonationRadiusBase, areaMod)
			if breakdown then
				local incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint = calcRadiusBreakpoints(data.misc.MineDetonationRadiusBase, incArea, moreArea)
				breakdown.MineDetonationRadius = breakdown.area(data.misc.MineDetonationRadiusBase, areaMod, output.MineDetonationRadius, incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint)
			end
			if activeSkill.skillTypes[SkillType.Aura] then
				output.MineAuraRadius = calcRadius(data.misc.MineAuraRadiusBase, output.AreaOfEffectMod)
				if breakdown then
					local incArea, moreArea = calcLib.mods(skillModList, skillCfg, "AreaOfEffect")
					local incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint = calcRadiusBreakpoints(data.misc.MineAuraRadiusBase, incArea, moreArea)
					breakdown.MineAuraRadius = breakdown.area(data.misc.MineAuraRadiusBase, output.AreaOfEffectMod, output.MineAuraRadius, incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint)
				end
			end
		end
		if skillFlags.totem then
			if skillFlags.ballista then
				baseSpeed = 1 / skillModList:Sum("BASE", skillCfg, "BallistaPlacementTime")
			else
				baseSpeed = 1 / skillModList:Sum("BASE", skillCfg, "TotemPlacementTime")
			end
			output.TotemPlacementSpeed = baseSpeed * calcLib.mod(skillModList, skillCfg, "TotemPlacementSpeed") * output.ActionSpeedMod
			output.TotemPlacementTime = 1 / output.TotemPlacementSpeed
			if breakdown then
				breakdown.TotemPlacementTime = { }
				breakdown.multiChain(breakdown.TotemPlacementTime, {
					label = "Placement speed:",
					base = s_format("%.2f ^8(base placement speed)", baseSpeed),
					{ "%.2f ^8(increased/reduced placement speed)", 1 + skillModList:Sum("INC", skillCfg, "TotemPlacementSpeed") / 100 },
					{ "%.2f ^8(more/less placement speed)", skillModList:More(skillCfg, "TotemPlacementSpeed") },
					{ "%.2f ^8(action speed modifier)",  output.ActionSpeedMod },
					total = s_format("= %.2f ^8per second", output.TotemPlacementSpeed),
				})
			end
			output.ActiveTotemLimit = skillModList:Sum("BASE", skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
			output.TotemsSummoned = env.modDB:Override(nil, "TotemsSummoned") or output.ActiveTotemLimit
			if breakdown then
				breakdown.ActiveTotemLimit = {
					"Totems Summoned: "..output.TotemsSummoned..(env.configInput.TotemsSummoned and " ^8(overridden from the Configuration tab)" or " ^8(can be overridden in the Configuration tab)"),
				}
			end
			output.TotemLifeMod = calcLib.mod(skillModList, skillCfg, "TotemLife")
			output.TotemLife = round(m_floor(env.data.monsterAllyLifeTable[skillData.totemLevel] * env.data.totemLifeMult[activeSkill.skillTotemId]) * output.TotemLifeMod)
			if breakdown then
				breakdown.TotemLifeMod = breakdown.mod(skillModList, skillCfg, "TotemLife")
				breakdown.TotemLife = {
					"Totem level: "..skillData.totemLevel,
					env.data.monsterAllyLifeTable[skillData.totemLevel].." ^8(base life for a level "..skillData.totemLevel.." monster)",
					"x "..env.data.totemLifeMult[activeSkill.skillTotemId].." ^8(life multiplier for this totem type)",
					"x "..output.TotemLifeMod.." ^8(totem life modifier)",
					"= "..output.TotemLife,
				}
			end
		end
		if skillFlags.brand then
			output.BrandAttachmentRange = data.misc.BrandAttachmentRangeBase * calcLib.mod(skillModList, skillCfg, "BrandAttachmentRange")
			output.ActiveBrandLimit = skillModList:Sum("BASE", skillCfg, "ActiveBrandLimit")
			if breakdown then
				breakdown.BrandAttachmentRange = { radius = output.BrandAttachmentRange }
			end
		end

		if skillFlags.warcry then
			output.WarcryCastTime = calcWarcryCastTime(skillModList, skillCfg, actor)
		end

		if skillFlags.corpse then
			output.CorpseLevel = skillModList:Sum("BASE", skillCfg, "CorpseLevel")
			output.BaseCorpseLife = env.data.monsterLifeTable[output.CorpseLevel or 1] * (env.data.monsterVarietyLifeMult[skillData.corpseMonsterVariety] or 1) * (env.data.mapLevelLifeMult[env.enemyLevel] or 1)
			output.CorpseLifeInc = 1 + (skillModList:Sum("INC", skillCfg, "CorpseLife") or 0) / 100
			output.CorpseLife = output.BaseCorpseLife * output.CorpseLifeInc
			if breakdown then
				breakdown.CorpseLife = {
					s_format("%d ^8(base life of a level %d monster)", env.data.monsterLifeTable[output.CorpseLevel or 1], output.CorpseLevel or "n/a"),
					s_format("x %.2f ^8(%s variety multiplier)", env.data.monsterVarietyLifeMult[skillData.corpseMonsterVariety] or 1, skillData.corpseMonsterVariety),
					s_format("x %.2f ^8(map level %d monster life multiplier from config)", env.data.mapLevelLifeMult[env.enemyLevel] or 1, env.enemyLevel),
					s_format(" = %d ^8(base corpse life)", output.BaseCorpseLife),
					s_format(""),
					s_format("x %.2f ^8(corpse maximum life increases)", output.CorpseLifeInc),
					s_format(" = %d", output.CorpseLife),
				}
			end
		end
	*/
	/*
		TODO -- General's Cry
		if skillData.triggeredByGeneralsCry then
			local mirageActiveSkill = nil

			-- Find the active General's Cry gem to get active properties
			for _, skill in ipairs(actor.activeSkillList) do
				if skill.activeEffect.grantedEffect.name == "General's Cry" and actor.mainSkill.socketGroup.slot == activeSkill.socketGroup.slot then
					mirageActiveSkill = skill
					break
				end
			end

			if mirageActiveSkill then
				local cooldown = calcSkillCooldown(mirageActiveSkill.skillModList, mirageActiveSkill.skillCfg, mirageActiveSkill.skillData)

				-- Non-channelled skills only attack once, disregard attack rate
				if not activeSkill.skillTypes[SkillType.Channel] then
					skillData.timeOverride = 1
				end

				-- Supported Attacks Count as Exerted
				for _, value in ipairs(env.modDB:Tabulate("INC", skillCfg, "ExertIncrease")) do
					local mod = value.mod
					skillModList:NewMod("Damage", mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags)
				end
				for _, value in ipairs(env.modDB:Tabulate("MORE", skillCfg, "ExertIncrease")) do
					local mod = value.mod
					skillModList:NewMod("Damage", mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags)
				end
				for _, value in ipairs(env.modDB:Tabulate("MORE", skillCfg, "ExertAttackIncrease")) do
					local mod = value.mod
					skillModList:NewMod("Damage", mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags)
				end
				for _, value in ipairs(env.modDB:Tabulate("BASE", skillCfg, "ExertDoubleDamageChance")) do
					local mod = value.mod
					skillModList:NewMod("DoubleDamageChance", mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags)
				end
				local maxMirageWarriors = 0
				for _, value in ipairs(mirageActiveSkill.skillModList:Tabulate("BASE", skillCfg, "GeneralsCryDoubleMaxCount")) do
					local mod = value.mod
					skillModList:NewMod("QuantityMultiplier", mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags)
					maxMirageWarriors = maxMirageWarriors + mod.value
				end
				env.player.mainSkill.infoMessage = tostring(maxMirageWarriors) .. " GC Mirage Warriors using " .. activeSkill.activeEffect.grantedEffect.name

				-- Scale dps with GC's cooldown
				if skillData.dpsMultiplier then
					skillData.dpsMultiplier = skillData.dpsMultiplier * (1 / cooldown)
				else
					skillData.dpsMultiplier = 1 / cooldown
				end
			end
		end
	*/
	/*
		TODO -- Skill duration
		local debuffDurationMult = 1
		if env.mode_effective then
			debuffDurationMult = 1 / m_max(data.misc.BuffExpirationSlowCap, calcLib.mod(enemyDB, skillCfg, "BuffExpireFaster"))
		end
		do
			output.DurationMod = calcLib.mod(skillModList, skillCfg, "Duration", "PrimaryDuration", "SkillAndDamagingAilmentDuration", skillData.mineDurationAppliesToSkill and "MineDuration" or nil)
			if breakdown then
				breakdown.DurationMod = breakdown.mod(skillModList, skillCfg, "Duration", "PrimaryDuration", "SkillAndDamagingAilmentDuration", skillData.mineDurationAppliesToSkill and "MineDuration" or nil)
				if breakdown.DurationMod and skillData.durationSecondary then
					t_insert(breakdown.DurationMod, 1, "Primary duration:")
				end
			end
			local durationBase = (skillData.duration or 0) + skillModList:Sum("BASE", skillCfg, "Duration", "PrimaryDuration")
			if durationBase > 0 then
				output.Duration = durationBase * output.DurationMod
				if skillData.debuff then
					output.Duration = output.Duration * debuffDurationMult
				end
				output.Duration = m_ceil(output.Duration * data.misc.ServerTickRate) / data.misc.ServerTickRate
				if breakdown and output.Duration ~= durationBase then
					breakdown.Duration = {
						s_format("%.2fs ^8(base)", durationBase),
					}
					if output.DurationMod ~= 1 then
						t_insert(breakdown.Duration, s_format("x %.4f ^8(duration modifier)", output.DurationMod))
					end
					if skillData.debuff and debuffDurationMult ~= 1 then
						t_insert(breakdown.Duration, s_format("/ %.3f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
					end
					t_insert(breakdown.Duration, s_format("rounded up to nearest server tick"))
					t_insert(breakdown.Duration, s_format("= %.3fs", output.Duration))
				end
			end
			durationBase = (skillData.durationSecondary or 0) + skillModList:Sum("BASE", skillCfg, "Duration", "SecondaryDuration")
			if durationBase > 0 then
				local durationMod = calcLib.mod(skillModList, skillCfg, "Duration", "SecondaryDuration", "SkillAndDamagingAilmentDuration", skillData.mineDurationAppliesToSkill and "MineDuration" or nil)
				output.DurationSecondary = durationBase * durationMod
				if skillData.debuffSecondary then
					output.DurationSecondary = output.DurationSecondary * debuffDurationMult
				end
				output.DurationSecondary = m_ceil(output.DurationSecondary * data.misc.ServerTickRate) / data.misc.ServerTickRate
				if breakdown and output.DurationSecondary ~= durationBase then
					breakdown.SecondaryDurationMod = breakdown.mod(skillModList, skillCfg, "Duration", "SecondaryDuration", "SkillAndDamagingAilmentDuration", skillData.mineDurationAppliesToSkill and "MineDuration" or nil)
					if breakdown.SecondaryDurationMod then
						t_insert(breakdown.SecondaryDurationMod, 1, "Secondary duration:")
					end
					breakdown.DurationSecondary = {
						s_format("%.2fs ^8(base)", durationBase),
					}
					if output.DurationMod ~= 1 then
						t_insert(breakdown.DurationSecondary, s_format("x %.4f ^8(duration modifier)", durationMod))
					end
					if skillData.debuffSecondary and debuffDurationMult ~= 1 then
						t_insert(breakdown.DurationSecondary, s_format("/ %.3f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
					end
					t_insert(breakdown.DurationSecondary, s_format("rounded up to nearest server tick"))
					t_insert(breakdown.DurationSecondary, s_format("= %.3fs", output.DurationSecondary))
				end
			end
			durationBase = (skillData.auraDuration or 0)
			if durationBase > 0 then
				local durationMod = calcLib.mod(skillModList, skillCfg, "Duration", "SkillAndDamagingAilmentDuration")
				output.AuraDuration = durationBase * durationMod
				output.AuraDuration = m_ceil(output.AuraDuration * data.misc.ServerTickRate) / data.misc.ServerTickRate
				if breakdown and output.AuraDuration ~= durationBase then
					breakdown.AuraDuration = {
						s_format("%.2fs ^8(base)", durationBase),
						s_format("x %.4f ^8(duration modifier)", durationMod),
						s_format("rounded up to nearest server tick"),
						s_format("= %.3fs", output.AuraDuration),
					}
				end
			end
			durationBase = (skillData.reserveDuration or 0)
			if durationBase > 0 then
				local durationMod = calcLib.mod(skillModList, skillCfg, "Duration", "SkillAndDamagingAilmentDuration")
				output.ReserveDuration = durationBase * durationMod
				output.ReserveDuration = m_ceil(output.ReserveDuration * data.misc.ServerTickRate) / data.misc.ServerTickRate
				if breakdown and output.ReserveDuration ~= durationBase then
					breakdown.ReserveDuration = {
						s_format("%.2fs ^8(base)", durationBase),
						s_format("x %.4f ^8(duration modifier)", durationMod),
						s_format("rounded up to nearest server tick"),
						s_format("= %.3fs", output.ReserveDuration),
					}
				end
			end
		end
	*/
	/*
		TODO -- Calculate costs (may be slightly off due to rounding differences)
		local costs = {
			["Mana"] = { type = "Mana", upfront = true, percent = false, text = "mana", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
			["Life"] = { type = "Life", upfront = true, percent = false, text = "life", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
			["ES"] = { type = "ES", upfront = true, percent = false, text = "ES", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
			["Rage"] = { type = "Rage", upfront = true, percent = false, text = "rage", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
			["ManaPercent"] = { type = "Mana", upfront = true, percent = true, text = "mana", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
			["LifePercent"] = { type = "Life", upfront = true, percent = true, text = "life", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
			["ManaPerMinute"] = { type = "Mana", upfront = false, percent = false, text = "mana/s", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
			["LifePerMinute"] = { type = "Life", upfront = false, percent = false, text = "life/s", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
			["ManaPercentPerMinute"] = { type = "Mana", upfront = false, percent = true, text = "mana/s", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
			["LifePercentPerMinute"] = { type = "Life", upfront = false, percent = true, text = "life/s", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
			["ESPerMinute"] = { type = "ES", upfront = false, percent = false, text = "ES/s", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
			["ESPercentPerMinute"] = { type = "ES", upfront = false, percent = true, text = "ES/s", baseCost = 0, totalCost = 0, baseCostNoMult = 0 },
		}
	*/
	/*
		TODO -- First pass to calculate base costs.  Used for cost conversion (e.g. Petrified Blood)
		for resource, val in pairs(costs) do
			local skillCost = activeSkill.activeEffect.grantedEffectLevel.cost and activeSkill.activeEffect.grantedEffectLevel.cost[resource] or nil
			local baseCost = round(skillCost and skillCost / data.costs[resource].Divisor or 0, 2)
			local baseCostNoMult = skillModList:Sum("BASE", skillCfg, resource.."CostNoMult") or 0
			local totalCost = 0
			if val.upfront then
				baseCost = baseCost + skillModList:Sum("BASE", skillCfg, resource.."CostBase")
				if resource == "Mana" and skillData.baseManaCostIsAtLeastPercentUnreservedMana then
					baseCost = m_max(baseCost, m_floor((output.ManaUnreserved or 0) * skillData.baseManaCostIsAtLeastPercentUnreservedMana / 100))
				end
				totalCost = skillModList:Sum("BASE", skillCfg, resource.."Cost")
				if activeSkill.skillTypes[SkillType.ReservationBecomesCost] then
					local reservedFlat = activeSkill.skillData[val.text.."ReservationFlat"] or activeSkill.activeEffect.grantedEffectLevel[val.text.."ReservationFlat"] or 0
					baseCost = baseCost + reservedFlat
					local reservedPercent = activeSkill.skillData[val.text.."ReservationPercent"] or activeSkill.activeEffect.grantedEffectLevel[val.text.."ReservationPercent"] or 0
					baseCost = baseCost + (m_floor((output[resource] or 0) * reservedPercent / 100))
				end
			end
			if val.type == "Mana" and skillModList:Flag(skillCfg, "CostLifeInsteadOfMana") then
				local target = resource:gsub("Mana", "Life")
				costs[target].baseCost = costs[target].baseCost + baseCost
				baseCost = 0
				costs[target].totalCost = costs[target].totalCost + totalCost
				totalCost = 0
				costs[target].baseCostNoMult = costs[target].baseCostNoMult + baseCostNoMult
				baseCostNoMult = 0
			end
			-- Extra cost (e.g. Petrified Blood) calculations happen after cost conversion (e.g. Blood Magic)
			if val.type == "Mana" and skillModList:Sum("BASE", skillCfg, "ManaCostAsLifeCost") then
				local target = resource:gsub("Mana", "Life")
				costs[target].baseCost = costs[target].baseCost + (baseCost + baseCostNoMult) * skillModList:Sum("BASE", skillCfg, "ManaCostAsLifeCost") / 100
			end
			val.baseCost = val.baseCost + baseCost
			val.totalCost = val.totalCost + totalCost
			val.baseCostNoMult = val.baseCostNoMult + baseCostNoMult
		end
		for resource, val in pairs(costs) do
			local dec = val.upfront and 0 or 2
			local costName = (val.upfront and resource or resource:gsub("Minute", "Second")).."Cost"
			local mult = floor(skillModList:More(skillCfg, "SupportManaMultiplier"), 2)
			local more = floor(skillModList:More(skillCfg, val.type.."Cost", "Cost"), 2)
			local inc = skillModList:Sum("INC", skillCfg, val.type.."Cost", "Cost")
			output[costName] = floor(val.baseCost * mult + val.baseCostNoMult, dec)
			output[costName] = floor(m_abs(inc / 100) * output[costName], dec) * (inc >= 0 and 1 or -1) + output[costName]
			output[costName] = floor(m_abs(more - 1) * output[costName], dec) * (more >= 1 and 1 or -1) + output[costName]
			output[costName] = m_max(0, floor(output[costName] + val.totalCost, dec))
			if breakdown and output[costName] ~= val.baseCost then
				breakdown[costName] = {
					s_format("%.2f"..(val.percent and "%%" or "").." ^8(base "..val.text.." cost)", val.baseCost)
				}
				if mult ~= 1 then
					t_insert(breakdown[costName], s_format("x %.2f ^8(cost multiplier)", mult))
				end
				if val.baseCostNoMult ~= 0 then
					t_insert(breakdown[costName], s_format("+ %d ^8(additional "..val.text.." cost)", val.baseCostNoMult))
				end
				if inc ~= 0 then
					t_insert(breakdown[costName], s_format("x %.2f ^8(increased/reduced "..val.text.." cost)", 1 + inc/100))
				end
				if more ~= 1 then
					t_insert(breakdown[costName], s_format("x %.2f ^8(more/less "..val.text.." cost)", more))
				end
				if val.totalCost ~= 0 then
					t_insert(breakdown[costName], s_format("%+d ^8(total "..val.text.." cost)", val.totalCost))
				end
				t_insert(breakdown[costName], s_format("= %"..(val.upfront and "d" or ".2f")..(val.percent and "%%" or ""), output[costName]))
			end
		end
	*/
	/*
		TODO -- account for Sacrificial Zeal
		-- Note: Sacrificial Zeal grants Added Spell Physical Damage equal to 25% of the Skill's Mana Cost, and causes you to take Physical Damage over Time, for 4 seconds
		if skillModList:Flag(nil, "Condition:SacrificialZeal") then
			local multiplier = 0.25
			skillModList:NewMod("PhysicalMin", "BASE", m_floor(output.ManaCost * multiplier), "Sacrificial Zeal", ModFlag.Spell)
			skillModList:NewMod("PhysicalMax", "BASE", m_floor(output.ManaCost * multiplier), "Sacrificial Zeal", ModFlag.Spell)
		end

		runSkillFunc("preDamageFunc")
	*/
	/*
		TODO -- Handle corpse explosions
		if skillData.explodeCorpse and (skillData.corpseLife or env.enemyLevel) then
			local localCorpseLife = skillData.corpseLife or data.monsterLifeTable[env.enemyLevel];
			local damageType = skillData.corpseExplosionDamageType or "Fire"
			skillData[damageType.."BonusMin"] = localCorpseLife * ( skillData.corpseExplosionLifeMultiplier or skillData.selfFireExplosionLifeMultiplier )
			skillData[damageType.."BonusMax"] = localCorpseLife * ( skillData.corpseExplosionLifeMultiplier or skillData.selfFireExplosionLifeMultiplier )
		end
	*/

	// Cache global damage disabling flags
	canDeal := make(map[data.DamageType]bool)
	for _, damageType := range data.DamageType("").Values() {
		canDeal[damageType] = !activeSkill.SkillModList.Flag(activeSkill.SkillCfg, "DealNo"+string(damageType))
	}

	// Calculate damage conversion percentages
	activeSkill.ConversionTable = make(map[data.DamageType]ConversionTable)
	totalDamageTypes := len(data.DamageType("").Values())
	for damageTypeIndex := 0; damageTypeIndex < totalDamageTypes; damageTypeIndex++ {
		damageType := data.DamageType("").Values()[damageTypeIndex]
		globalConv := make(map[data.DamageType]float64)
		skillConv := make(map[data.DamageType]float64)
		add := make(map[data.DamageType]float64)
		globalTotal := float64(0)
		skillTotal := float64(0)

		for otherTypeIndex := damageTypeIndex + 1; otherTypeIndex < totalDamageTypes; otherTypeIndex++ {
			// For all possible destination types, check for global and skill conversions
			otherType := data.DamageType("").Values()[otherTypeIndex]

			globalNames := []string{string(damageType) + "DamageConvertTo" + string(otherType)}
			if damageType.IsElemental() {
				globalNames = append(globalNames, "ElementalDamageConvertTo"+string(otherType))
			}
			if damageType != data.DamageTypeChaos {
				globalNames = append(globalNames, "NonChaosDamageConvertTo"+string(otherType))
			}
			globalConv[otherType] = activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, globalNames...)
			globalTotal += globalConv[otherType]

			skillConv[otherType] = activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Skill"+string(damageType)+"DamageConvertTo"+string(otherType))
			skillTotal += skillConv[otherType]

			addNames := []string{string(damageType) + "DamageGainAs" + string(otherType)}
			if damageType.IsElemental() {
				addNames = append(addNames, "ElementalDamageGainAs"+string(otherType))
			}
			if damageType != data.DamageTypeChaos {
				addNames = append(addNames, "NonChaosDamageGainAs"+string(otherType))
			}
			add[otherType] = activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, addNames...)
		}

		if skillTotal > 100 {
			// Skill conversion exceeds 100%, scale it down and remove non-skill conversions
			factor := 100 / skillTotal
			for convType, val := range skillConv {
				// Overconversion is fixed in 3.0, so I finally get to uncomment this line!
				skillConv[convType] = val * factor
			}
			for convType := range globalConv {
				globalConv[convType] = 0
			}
		} else if globalTotal+skillTotal > 100 {
			// Conversion exceeds 100%, scale down non-skill conversions
			factor := (100 - skillTotal) / globalTotal
			for convType, val := range globalConv {
				globalConv[convType] = val * factor
			}
			globalTotal = globalTotal * factor
		}

		dmgTable := ConversionTable{
			Targets: make(map[data.DamageType]float64),
		}
		for convType, val := range globalConv {
			dmgTable.Targets[convType] = (val + skillConv[convType] + add[convType]) / 100
		}

		dmgTable.Mult = 1 - math.Min((globalTotal+skillTotal)/100, 1)
		activeSkill.ConversionTable[damageType] = dmgTable
	}

	activeSkill.ConversionTable[data.DamageTypeChaos] = ConversionTable{
		Mult: 1,
	}

	// Configure damage passes
	passList := make([]*DamagePass, 0)
	if activeSkill.SkillFlags[SkillFlagAttack] {
		actor.OutputTable[OutTableMainHand] = make(map[string]float64)
		actor.OutputTable[OutTableOffHand] = make(map[string]float64)
		critOverride := activeSkill.SkillModList.Override(activeSkill.SkillCfg, "WeaponBaseCritChance")
		if activeSkill.SkillFlags[SkillFlagWeapon1Attack] {
			if actor.Breakdown != nil {
				// TODO Breakdown
				// breakdown.MainHand = LoadModule(calcs.breakdownModule, skillModList, output.MainHand)
			}
			activeSkill.Weapon1Cfg.SkillStats = actor.OutputTable[OutTableMainHand]
			source := actor.WeaponData1 // TODO Copy
			if critOverride != nil && source != nil && source["Type"] != "None" {
				source["CritChance"] = critOverride.(float64)
			}
			passList = append(passList, &DamagePass{
				Label:     "Main Hand",
				Source:    source,
				Config:    activeSkill.Weapon1Cfg,
				Output:    actor.OutputTable[OutTableMainHand],
				Breakdown: actor.Breakdown,
			})
		}

		if activeSkill.SkillFlags[SkillFlagWeapon2Attack] {
			if actor.Breakdown != nil {
				// TODO Breakdown
				// breakdown.OffHand = LoadModule(calcs.breakdownModule, skillModList, output.OffHand)
			}
			activeSkill.Weapon2Cfg.SkillStats = actor.OutputTable[OutTableOffHand]
			source := utils.CopyMap(actor.WeaponData2) // TODO Copy
			if critOverride != nil && source != nil && source["Type"] != "None" {
				source["CritChance"] = critOverride.(float64)
			}
			if utils.Has(activeSkill.SkillData, "CritChance") {
				source["CritChance"] = activeSkill.SkillData["CritChance"]
			}
			if utils.Has(activeSkill.SkillData, "SetOffHandPhysicalMin") && utils.Has(activeSkill.SkillData, "SetOffHandPhysicalMax") {
				source["PhysicalMin"] = activeSkill.SkillData["SetOffHandPhysicalMin"]
				source["PhysicalMax"] = activeSkill.SkillData["SetOffHandPhysicalMax"]
			}
			if utils.Has(activeSkill.SkillData, "AttackTime") {
				source["AttackRate"] = utils.Ptr[float64](1000 / activeSkill.SkillData["AttackTime"].(float64))
			}
			passList = append(passList, &DamagePass{
				Label:     "Off Hand",
				Source:    source,
				Config:    activeSkill.Weapon2Cfg,
				Output:    actor.OutputTable[OutTableOffHand],
				Breakdown: actor.Breakdown,
			})
		}
	} else {
		passList = append(passList, &DamagePass{
			Label:     "Skill",
			Source:    activeSkill.SkillData,
			Config:    activeSkill.SkillCfg,
			Output:    actor.Output,
			Breakdown: actor.Breakdown,
		})
	}

	type CombineMode string
	const (
		ModeOr            = CombineMode("OR")
		ModeAdd           = CombineMode("ADD")
		ModeAverage       = CombineMode("AVERAGE")
		ModeChance        = CombineMode("CHANCE")
		ModeChanceAilment = CombineMode("CHANCE_AILMENT")
		ModeDPS           = CombineMode("DPS")
	)

	combineStat := func(stat string, mode CombineMode) {
		// Combine stats from Main Hand and Off Hand according to the mode
		if mode == ModeOr || utils.MissingOrFalse(activeSkill.SkillFlags, SkillFlagBothWeaponAttack) {
			if utils.Has(actor.OutputTable[OutTableMainHand], stat) {
				actor.Output[stat] = actor.OutputTable[OutTableMainHand][stat]
			} else {
				actor.Output[stat] = actor.OutputTable[OutTableOffHand][stat]
			}
		} else if mode == ModeAdd {
			actor.Output[stat] = actor.OutputTable[OutTableMainHand][stat] + actor.OutputTable[OutTableOffHand][stat]
		} else if mode == ModeAverage {
			sum := actor.OutputTable[OutTableMainHand][stat] + actor.OutputTable[OutTableOffHand][stat]
			actor.Output[stat] = sum / 2
		} else if mode == ModeChance {
			if utils.Has(actor.OutputTable[OutTableMainHand], stat) && utils.Has(actor.OutputTable[OutTableOffHand], stat) {
				/*
					TODO Chance
					local mainChance = output.MainHand[...] * output.MainHand.HitChance
					local offChance = output.OffHand[...] * output.OffHand.HitChance
					local mainPortion = mainChance / (mainChance + offChance)
					local offPortion = offChance / (mainChance + offChance)
					output[stat] = output.MainHand[stat] * mainPortion + output.OffHand[stat] * offPortion
				*/
				/*
					TODO Breakdown
					if breakdown then
						if not breakdown[stat] then
							breakdown[stat] = { }
						end
						t_insert(breakdown[stat], "Contribution from Main Hand:")
						t_insert(breakdown[stat], s_format("%.1f", output.MainHand[stat]))
						t_insert(breakdown[stat], s_format("x %.3f ^8(portion of instances created by main hand)", mainPortion))
						t_insert(breakdown[stat], s_format("= %.1f", output.MainHand[stat] * mainPortion))
						t_insert(breakdown[stat], "Contribution from Off Hand:")
						t_insert(breakdown[stat], s_format("%.1f", output.OffHand[stat]))
						t_insert(breakdown[stat], s_format("x %.3f ^8(portion of instances created by off hand)", offPortion))
						t_insert(breakdown[stat], s_format("= %.1f", output.OffHand[stat] * offPortion))
						t_insert(breakdown[stat], "Total:")
						t_insert(breakdown[stat], s_format("%.1f + %.1f", output.MainHand[stat] * mainPortion, output.OffHand[stat] * offPortion))
						t_insert(breakdown[stat], s_format("= %.1f", output[stat]))
					end
				*/
			} else {
				if utils.Has(actor.OutputTable[OutTableMainHand], stat) {
					actor.Output[stat] = actor.OutputTable[OutTableMainHand][stat]
				} else {
					actor.Output[stat] = actor.OutputTable[OutTableOffHand][stat]
				}
			}
		} else if mode == ModeChanceAilment {
			if utils.Has(actor.OutputTable[OutTableMainHand], stat) && utils.Has(actor.OutputTable[OutTableOffHand], stat) {
				/*
					TODO Chance Ailment
					local mainChance = output.MainHand[...] * output.MainHand.HitChance
					local offChance = output.OffHand[...] * output.OffHand.HitChance
					local mainPortion = mainChance / (mainChance + offChance)
					local offPortion = offChance / (mainChance + offChance)
					local maxInstance = m_max(output.MainHand[stat], output.OffHand[stat])
					local minInstance = m_min(output.MainHand[stat], output.OffHand[stat])
					local stackName = stat:gsub("DPS","") .. "Stacks"
					local maxInstanceStacks = m_min(1, (globalOutput[stackName] or 1) / (globalOutput[stackName.."Max"] or 1))
					output[stat] = maxInstance * maxInstanceStacks + minInstance * (1 - maxInstanceStacks)
				*/
				/*
					TODO Breakdown
					if breakdown then
						if not breakdown[stat] then breakdown[stat] = { } end
						t_insert(breakdown[stat], s_format(""))
						t_insert(breakdown[stat], s_format("%.2f%% of ailment stacks use maximum damage", maxInstanceStacks * 100))
						t_insert(breakdown[stat], s_format("Max Damage comes from %s", output.MainHand[stat] >= output.OffHand[stat] and "Main Hand" or "Off Hand"))
						t_insert(breakdown[stat], s_format("= %.1f", maxInstance * maxInstanceStacks))
						if maxInstanceStacks < 1 then
							t_insert(breakdown[stat], s_format("%.2f%% of ailment stacks use non-maximum damage", (1-maxInstanceStacks) * 100))
							t_insert(breakdown[stat], s_format("= %.1f", minInstance * (1 - maxInstanceStacks)))
						end
						t_insert(breakdown[stat], "")
						t_insert(breakdown[stat], "Total:")
						if maxInstanceStacks < 1 then
							t_insert(breakdown[stat], s_format("%.1f + %.1f", maxInstance * maxInstanceStacks, minInstance * (1 - maxInstanceStacks)))
						end
						t_insert(breakdown[stat], s_format("= %.1f", output[stat]))
					end
				*/
			} else {
				if utils.Has(actor.OutputTable[OutTableMainHand], stat) {
					actor.Output[stat] = actor.OutputTable[OutTableMainHand][stat]
				} else {
					actor.Output[stat] = actor.OutputTable[OutTableOffHand][stat]
				}
				/*
					TODO Breakdown
					if breakdown then
						if not breakdown[stat] then breakdown[stat] = { } end
						t_insert(breakdown[stat], s_format("All ailment stacks comes from %s", output.MainHand[stat] and "Main Hand" or "Off Hand"))
					end
				*/
			}
		} else if mode == ModeDPS {
			actor.Output[stat] = actor.OutputTable[OutTableMainHand][stat] + actor.OutputTable[OutTableOffHand][stat]
			if utils.MissingOrFalse(activeSkill.SkillData, "DoubleHitsWhenDualWielding") {
				actor.Output[stat] = actor.Output[stat] / 2
			}
		}
	}

	// TODO storedMainHandAccuracy
	var storedMainHandAccuracy *float64 = nil
	for _, pass := range passList {
		// Calculate hit chance
		pass.Output["Accuracy"] = math.Max(0, CalcVal(activeSkill.SkillModList, "Accuracy", pass.Config))
		/*
			TODO Breakdown
			if breakdown then
				breakdown.Accuracy = breakdown.simple(nil, cfg, output.Accuracy, "Accuracy")
			end
		*/

		if activeSkill.SkillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.Label == "Main Hand" {
			storedMainHandAccuracy = utils.Ptr(pass.Output["Accuracy"])
		} else if activeSkill.SkillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.Label == "Off Hand" && storedMainHandAccuracy != nil {
			pass.Output["Accuracy"] = *storedMainHandAccuracy
			/*
				TODO Breakdown
				if breakdown then
					breakdown.Accuracy = {
						"Using Main Hand Accuracy due to Mastery: "..output.Accuracy,
					}
				end
			*/
		}

		if utils.MissingOrFalse(activeSkill.SkillFlags, SkillFlagAttack) ||
			activeSkill.SkillModList.Flag(pass.Config, "CannotBeEvaded") ||
			utils.HasTrue(activeSkill.SkillData, "CannotBeEvaded") ||
			(env.ModeEffective && actor.Enemy.ModDB.Flag(nil, "CannotEvade")) {
			pass.Output["HitChance"] = 100
		} else {
			enemyEvasion := math.Max(math.Round(CalcVal(actor.Enemy.ModDB, "Evasion", nil)), 0)
			pass.Output["HitChance"] = CalcHitChance(enemyEvasion, pass.Output["Accuracy"]) * CalcMod(activeSkill.SkillModList, pass.Config, "HitChance")
			/*
				TODO Breakdown
				if breakdown then
					breakdown.HitChance = {
						"Enemy level: "..env.enemyLevel..(env.configInput.enemyLevel and " ^8(overridden from the Configuration tab" or " ^8(can be overridden in the Configuration tab)"),
						"Average enemy evasion: "..enemyEvasion,
						"Approximate hit chance: "..output.HitChance.."%",
					}
				end
			*/
		}
		/*
			TODO -- Check Precise Technique Keystone condition per pass as MH/OH might have different values
			local condName = pass.label:gsub(" ", "") .. "AccRatingHigherThanMaxLife"
			skillModList.conditions[condName] = output.Accuracy > env.player.output.Life
		*/

		if (activeSkill.ActiveEffect.GrantedEffect.CastTime == nil || *activeSkill.ActiveEffect.GrantedEffect.CastTime == 0) && utils.Has(activeSkill.SkillData, "CastTimeOverride") {
			pass.Output["Time"] = 0
			pass.Output["Speed"] = 0
		} else if utils.Has(activeSkill.SkillData, "TimeOverride") {
			pass.Output["Time"] = activeSkill.SkillData["TimeOverride"].(float64)
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else if utils.HasTrue(activeSkill.SkillData, "FixedCastTime") {
			pass.Output["Time"] = *activeSkill.ActiveEffect.GrantedEffect.CastTime
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else if utils.Has(activeSkill.SkillData, "TriggerTime") && utils.HasTrue(activeSkill.SkillData, "Triggered") {
			activeSkillsLinked := activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "ActiveSkillsLinkedToTrigger")
			if activeSkillsLinked > 0 {
				pass.Output["Time"] = activeSkill.SkillData["TriggerTime"].(float64) / (1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, pass.Config, "CooldownRecovery")/100) * activeSkillsLinked
			} else {
				pass.Output["Time"] = activeSkill.SkillData["TriggerTime"].(float64) / (1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, pass.Config, "CooldownRecovery")/100)
			}
			pass.Output["TriggerTime"] = pass.Output["Time"]
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else if utils.Has(activeSkill.SkillData, "TriggerRate") && utils.HasTrue(activeSkill.SkillData, "Triggered") {
			/*
				TODO -- Account for trigger unleash
				if skillData.triggerUnleash then
					-- process the source trigger skill to get it's full data
					local calcMode = env.mode == "CALCS" and "CALCS" or "MAIN"
					for _, triggerSkill in ipairs(actor.activeSkillList) do
						if cacheSkillUUID(triggerSkill) == skillData.triggerSourceUUID then
							calcs.buildActiveSkill(env, calcMode, triggerSkill)
							break
						end
					end
					local cachedSourceSkill = GlobalCache.cachedData[calcMode][skillData.triggerSourceUUID]
					-- if properly processed, get it's dpsMultiplier to increase triggerRate
					if cachedSourceSkill then
						skillData.unleashTriggerRate = skillData.triggerRate * (cachedSourceSkill.ActiveSkill.skillData.dpsMultiplier or 1)
						if breakdown then
							breakdown.Speed = {
								s_format("%.2f ^8(trigger rate)", skillData.triggerRate),
								s_format("* %.2f ^8(multiplier from Unleash)", cachedSourceSkill.ActiveSkill.skillData.dpsMultiplier or 1),
								s_format("= %.2f", skillData.unleashTriggerRate),
							}
						end
						-- over-write the triggerRate modifier after breakdown as other calcs use it
						skillData.triggerRate = skillData.unleashTriggerRate
					end
					-- give this activeSkill "HasSeals" flag so Configuration Option for UseMaxUnleash is available
					activeSkill.skillFlags.HasSeals = true
				end
			*/
			pass.Output["Time"] = 1 / activeSkill.SkillData["TriggerRate"].(float64)
			pass.Output["TriggerTime"] = pass.Output["Time"]
			pass.Output["Speed"] = activeSkill.SkillData["TriggerRate"].(float64)
			activeSkill.SkillData["ShowAverage"] = false
		} else if utils.HasTrue(activeSkill.SkillData, "TriggeredByBrand") && utils.HasTrue(activeSkill.SkillData, "Triggered") {
			ArcanistSpellsLinked := activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "ArcanistSpellsLinked")
			if ArcanistSpellsLinked == 0 {
				ArcanistSpellsLinked = 1
			}
			pass.Output["Time"] = 1 / (1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, pass.Config, "Speed", "BrandActivationFrequency")/100) / activeSkill.SkillModList.More(pass.Config, "BrandActivationFrequency") * ArcanistSpellsLinked
			pass.Output["TriggerTime"] = pass.Output["Time"]
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else {
			baseTime := float64(0)
			if activeSkill.SkillFlags[SkillFlagAttack] {
				if utils.Has(activeSkill.SkillData, "CastTimeOverride") {
					// Skill is overriding weapon attack speed
					baseTime = *activeSkill.ActiveEffect.GrantedEffect.CastTime / (1 + (utils.GetOr(pass.Source, "AttackSpeedInc", utils.Interface(float64(0))).(float64))/100)
				} else if CalcMod(activeSkill.SkillModList, activeSkill.SkillCfg, "SkillAttackTime") > 0 {
					baseTime = (1/utils.GetOr(pass.Source, "AttackRate", utils.Interface(float64(1))).(float64) + activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "Speed")) * CalcMod(activeSkill.SkillModList, activeSkill.SkillCfg, "SkillAttackTime")
				} else {
					baseTime = 1/utils.GetOr(pass.Source, "AttackRate", utils.Interface(float64(1))).(float64) + activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "Speed")
				}
			} else {
				baseTime = 1
				if utils.Has(activeSkill.SkillData, "CastTimeOverride") {
					baseTime = activeSkill.SkillData["CastTimeOverride"].(float64)
				} else if activeSkill.ActiveEffect.GrantedEffect.CastTime != nil {
					baseTime = *activeSkill.ActiveEffect.GrantedEffect.CastTime
				}
			}

			inc := activeSkill.SkillModList.Sum(mod.TypeIncrease, pass.Config, "Speed")
			more := activeSkill.SkillModList.More(pass.Config, "Speed")

			pass.Output["Speed"] = 1 / baseTime * utils.RoundTo((1+inc/100)*more, 2)
			pass.Output["CastRate"] = pass.Output["Speed"]
			pass.Output["Repeats"] = 1 + activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "RepeatCount")

			if activeSkill.SkillFlags[SkillFlagSelfCast] {
				// Self-cast skill; apply action speed
				pass.Output["Speed"] = pass.Output["Speed"] * actor.Output["ActionSpeedMod"]
				pass.Output["CastRate"] = pass.Output["Speed"]
			}

			if utils.Has(pass.Output, "Cooldown") {
				pass.Output["Speed"] = math.Min(pass.Output["Speed"], 1/pass.Output["Cooldown"]*pass.Output["Repeats"])
			}

			if utils.Has(pass.Output, "Cooldown") && activeSkill.SkillFlags[SkillFlagSelfCast] {
				activeSkill.SkillFlags[SkillFlagNotAverage] = true
				activeSkill.SkillFlags[SkillFlagShowAverage] = false
				activeSkill.SkillData["ShowAverage"] = false
			}

			if utils.MissingOrFalse(activeSkill.SkillTypes, data.SkillTypeChannel) {
				pass.Output["Speed"] = math.Min(pass.Output["Speed"], data.ServerTickRate*pass.Output["Repeats"])
			}

			if pass.Output["Speed"] == 0 {
				pass.Output["Time"] = 0
			} else {
				pass.Output["Time"] = 1 / pass.Output["Speed"]
			}

			/*
				TODO Breakdown
				if breakdown then
					breakdown.Speed = { }
					breakdown.multiChain(breakdown.Speed, {
						base = s_format("%.2f ^8(base)", 1 / baseTime),
						{ "%.2f ^8(increased/reduced)", 1 + inc/100 },
						{ "%.2f ^8(more/less)", more },
						{ "%.2f ^8(action speed modifier)", skillFlags.selfCast and globalOutput.ActionSpeedMod or 1 },
						total = s_format("= %.2f ^8casts per second", output.CastRate)
					})
					if output.Cooldown and (1 / output.Cooldown) < output.CastRate then
						t_insert(breakdown.Speed, s_format("\n"))
						t_insert(breakdown.Speed, s_format("1 / %.2f ^8(skill cooldown)", output.Cooldown))
						if output.Repeats > 1 then
							t_insert(breakdown.Speed, s_format("x %d ^8(repeat count)", output.Repeats))
						end
						t_insert(breakdown.Speed, s_format("= %.2f ^8(casts per second)", output.Repeats / output.Cooldown))
						t_insert(breakdown.Speed, s_format("\n"))
						t_insert(breakdown.Speed, s_format("= %.2f ^8(lower of cast rates)", output.Speed))
					end
				end
				if breakdown and calcLib.mod(skillModList, skillCfg, "SkillAttackTime") > 0 then
					breakdown.Time = { }
					breakdown.multiChain(breakdown.Time, {
						base = s_format("%.2f ^8(base)", 1 / (output.Speed * calcLib.mod(skillModList, skillCfg, "SkillAttackTime") )),
						{ "%.2f ^8(total modifier)", calcLib.mod(skillModList, skillCfg, "SkillAttackTime")  },
						total = s_format("= %.2f ^8seconds per attack", output.Time)
					})
				end
			*/
		}
		/*
			TODO Time override
			if skillData.hitTimeOverride and not skillData.triggeredOnDeath then
				output.HitTime = skillData.hitTimeOverride
				output.HitSpeed = 1 / output.HitTime
				--Brands always have hitTimeOverride
				if skillFlags.brand then
					output.BrandTicks = m_floor(output.Duration * output.HitSpeed)
				end
			elseif skillData.hitTimeMultiplier and output.Time and not skillData.triggeredOnDeath then
				output.HitTime = output.Time * skillData.hitTimeMultiplier
				output.HitSpeed = 1 / output.HitTime
			end
		*/
	}

	if utils.HasTrue(activeSkill.SkillFlags, SkillFlagAttack) {
		// Combine hit chance and attack speed
		combineStat("HitChance", ModeAverage)
		combineStat("Speed", ModeAverage)
		combineStat("HitSpeed", ModeOr)

		if actor.Output["Speed"] == 0 {
			actor.Output["Time"] = 0
		} else {
			actor.Output["Time"] = 1 / actor.Output["Speed"]
		}

		if actor.Output["Time"] > 1 {
			actor.ModDB.AddMod(mod.NewFlag("Condition:OneSecondAttackTime", true))
		}

		if utils.HasTrue(activeSkill.SkillFlags, SkillFlagBothWeaponAttack) {
			/*
				TODO Breakdown
				if breakdown then
					breakdown.Speed = {
						"Both weapons:",
						s_format("(%.2f + %.2f) / 2", output.MainHand.Speed, output.OffHand.Speed),
						s_format("= %.2f", output.Speed),
					}
				end
			*/
		}
	}

	quantityMultiplier := math.Max(activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "QuantityMultiplier"), 1)
	if quantityMultiplier > 1 {
		actor.Output["QuantityMultiplier"] = quantityMultiplier
	}

	for _, pass := range passList {
		/*
			TODO Passes
			globalOutput, globalBreakdown = output, breakdown
			local source, output, cfg, breakdown = pass.source, pass.output, pass.cfg, pass.breakdown
		*/

		// Exerted Attack members
		actor.Output["OffensiveWarcryEffect"] = 1
		actor.Output["MaxOffensiveWarcryEffect"] = 1
		actor.Output["TheoreticalOffensiveWarcryEffect"] = 1
		actor.Output["TheoreticalMaxOffensiveWarcryEffect"] = 1
		actor.Output["RallyingHitEffect"] = 1
		actor.Output["AilmentWarcryEffect"] = 1

		/*
			local exertedDoubleDamage = env.modDB:Sum("BASE", cfg, "ExertDoubleDamageChance")
			if env.mode_buffs then
				-- Iterative over all the active skills to account for exerted attacks provided by warcries
				if (activeSkill.activeEffect.grantedEffect.name == "Vaal Ground Slam" or not activeSkill.skillTypes[SkillType.Vaal]) and not activeSkill.skillTypes[SkillType.Channel] and not activeSkill.skillModList:Flag(cfg, "SupportedByMultistrike") then
					for index, value in ipairs(actor.activeSkillList) do
						if value.activeEffect.grantedEffect.name == "Ancestral Cry" and activeSkill.skillTypes[SkillType.MeleeSingleTarget] and not globalOutput.AncestralCryCalculated then
							globalOutput.AncestralCryDuration = calcSkillDuration(value.skillModList, value.skillCfg, value.skillData, env, enemyDB)
							globalOutput.AncestralCryCooldown = calcSkillCooldown(value.skillModList, value.skillCfg, value.skillData)
							output.GlobalWarcryCooldown = env.modDB:Sum("BASE", nil, "GlobalWarcryCooldown")
							output.GlobalWarcryCount = env.modDB:Sum("BASE", nil, "GlobalWarcryCount")
							if modDB:Flag(nil, "WarcryShareCooldown") then
								globalOutput.AncestralCryCooldown = globalOutput.AncestralCryCooldown + (output.GlobalWarcryCooldown - globalOutput.AncestralCryCooldown) / output.GlobalWarcryCount
							end
							globalOutput.AncestralCryCastTime = calcWarcryCastTime(value.skillModList, value.skillCfg, actor)
							globalOutput.AncestralExertsCount = env.modDB:Sum("BASE", nil, "NumAncestralExerts") or 0
							local baseUptimeRatio = m_min((globalOutput.AncestralExertsCount / output.Speed) / (globalOutput.AncestralCryCooldown + globalOutput.AncestralCryCastTime), 1) * 100
							local additionalCooldownUses = value.skillModList:Sum("BASE", value.skillCfg, "AdditionalCooldownUses")
							globalOutput.AncestralUpTimeRatio = m_min(100, baseUptimeRatio * (additionalCooldownUses + 1))
							if globalBreakdown then
								globalBreakdown.AncestralUpTimeRatio = { }
								t_insert(globalBreakdown.AncestralUpTimeRatio, s_format("(%d ^8(number of exerts)", globalOutput.AncestralExertsCount))
								t_insert(globalBreakdown.AncestralUpTimeRatio, s_format("/ %.2f) ^8(attacks per second)", output.Speed))
								if globalOutput.AncestralCryCastTime > 0 then
									t_insert(globalBreakdown.AncestralUpTimeRatio, s_format("/ (%.2f ^8(warcry cooldown)", globalOutput.AncestralCryCooldown))
									t_insert(globalBreakdown.AncestralUpTimeRatio, s_format("+ %.2f) ^8(warcry casttime)", globalOutput.AncestralCryCastTime))
								else
									t_insert(globalBreakdown.AncestralUpTimeRatio, s_format("/ %.2f ^8(average warcry cooldown)", globalOutput.AncestralCryCooldown))
								end
								t_insert(globalBreakdown.AncestralUpTimeRatio, s_format("= %d%%", globalOutput.AncestralUpTimeRatio))
							end
							globalOutput.AncestralCryCalculated = true
						elseif value.activeEffect.grantedEffect.name == "Infernal Cry" and not globalOutput.InfernalCryCalculated then
							globalOutput.InfernalCryDuration = calcSkillDuration(value.skillModList, value.skillCfg, value.skillData, env, enemyDB)
							globalOutput.InfernalCryCooldown = calcSkillCooldown(value.skillModList, value.skillCfg, value.skillData)
							output.GlobalWarcryCooldown = env.modDB:Sum("BASE", nil, "GlobalWarcryCooldown")
							output.GlobalWarcryCount = env.modDB:Sum("BASE", nil, "GlobalWarcryCount")
							if modDB:Flag(nil, "WarcryShareCooldown") then
								globalOutput.InfernalCryCooldown = globalOutput.InfernalCryCooldown + (output.GlobalWarcryCooldown - globalOutput.InfernalCryCooldown) / output.GlobalWarcryCount
							end
							globalOutput.InfernalCryCastTime = calcWarcryCastTime(value.skillModList, value.skillCfg, actor)
							if activeSkill.skillTypes[SkillType.Melee] then
								globalOutput.InfernalExertsCount = env.modDB:Sum("BASE", nil, "NumInfernalExerts") or 0
								local baseUptimeRatio = m_min((globalOutput.InfernalExertsCount / output.Speed) / (globalOutput.InfernalCryCooldown + globalOutput.InfernalCryCastTime), 1) * 100
								local additionalCooldownUses = value.skillModList:Sum("BASE", value.skillCfg, "AdditionalCooldownUses")
								globalOutput.InfernalUpTimeRatio = m_min(100, baseUptimeRatio * (additionalCooldownUses + 1))
								if globalBreakdown then
									globalBreakdown.InfernalUpTimeRatio = { }
									t_insert(globalBreakdown.InfernalUpTimeRatio, s_format("(%d ^8(number of exerts)", globalOutput.InfernalExertsCount))
									t_insert(globalBreakdown.InfernalUpTimeRatio, s_format("/ %.2f) ^8(attacks per second)", output.Speed))
									if globalOutput.InfernalCryCastTime > 0 then
										t_insert(globalBreakdown.InfernalUpTimeRatio, s_format("/ (%.2f ^8(warcry cooldown)", globalOutput.InfernalCryCooldown))
										t_insert(globalBreakdown.InfernalUpTimeRatio, s_format("+ %.2f) ^8(warcry casttime)", globalOutput.InfernalCryCastTime))
									else
										t_insert(globalBreakdown.InfernalUpTimeRatio, s_format("/ %.2f ^8(average warcry cooldown)", globalOutput.InfernalCryCooldown))
									end
									t_insert(globalBreakdown.InfernalUpTimeRatio, s_format("= %d%%", globalOutput.InfernalUpTimeRatio))
								end
							end
							globalOutput.InfernalCryCalculated = true
						elseif value.activeEffect.grantedEffect.name == "Intimidating Cry" and activeSkill.skillTypes[SkillType.Melee] and not globalOutput.IntimidatingCryCalculated then
							globalOutput.CreateWarcryOffensiveCalcSection = true
							globalOutput.IntimidatingCryDuration = calcSkillDuration(value.skillModList, value.skillCfg, value.skillData, env, enemyDB)
							globalOutput.IntimidatingCryCooldown = calcSkillCooldown(value.skillModList, value.skillCfg, value.skillData)
							output.GlobalWarcryCooldown = env.modDB:Sum("BASE", nil, "GlobalWarcryCooldown")
							output.GlobalWarcryCount = env.modDB:Sum("BASE", nil, "GlobalWarcryCount")
							if modDB:Flag(nil, "WarcryShareCooldown") then
								globalOutput.IntimidatingCryCooldown = globalOutput.IntimidatingCryCooldown + (output.GlobalWarcryCooldown - globalOutput.IntimidatingCryCooldown) / output.GlobalWarcryCount
							end
							globalOutput.IntimidatingCryCastTime = calcWarcryCastTime(value.skillModList, value.skillCfg, actor)
							globalOutput.IntimidatingExertsCount = env.modDB:Sum("BASE", nil, "NumIntimidatingExerts") or 0
							local baseUptime = m_min((globalOutput.IntimidatingExertsCount / output.Speed) / (globalOutput.IntimidatingCryCooldown + globalOutput.IntimidatingCryCastTime), 1) * 100
							local additionalCooldownUses = value.skillModList:Sum("BASE", value.skillCfg, "AdditionalCooldownUses")
							globalOutput.IntimidatingUpTimeRatio = m_min(100, baseUptime * (additionalCooldownUses + 1))
							if globalBreakdown then
								globalBreakdown.IntimidatingUpTimeRatio = { }
								t_insert(globalBreakdown.IntimidatingUpTimeRatio, s_format("(%d ^8(number of exerts)", globalOutput.IntimidatingExertsCount))
								t_insert(globalBreakdown.IntimidatingUpTimeRatio, s_format("/ %.2f) ^8(attacks per second)", output.Speed))
								if 	globalOutput.IntimidatingCryCastTime > 0 then
									t_insert(globalBreakdown.IntimidatingUpTimeRatio, s_format("/ (%.2f ^8(warcry cooldown)", globalOutput.IntimidatingCryCooldown))
									t_insert(globalBreakdown.IntimidatingUpTimeRatio, s_format("+ %.2f) ^8(warcry casttime)", globalOutput.IntimidatingCryCastTime))
								else
									t_insert(globalBreakdown.IntimidatingUpTimeRatio, s_format("/ %.2f ^8(average warcry cooldown)", globalOutput.IntimidatingCryCooldown))
								end
								t_insert(globalBreakdown.IntimidatingUpTimeRatio, s_format("= %d%%", globalOutput.IntimidatingUpTimeRatio))
							end
							local ddChance = m_min(skillModList:Sum("BASE", cfg, "DoubleDamageChance") + (env.mode_effective and enemyDB:Sum("BASE", cfg, "SelfDoubleDamageChance") or 0) + exertedDoubleDamage, 100)
							globalOutput.IntimidatingAvgDmg = 2 * (1 - ddChance / 100) -- 1
							if globalBreakdown then
								globalBreakdown.IntimidatingAvgDmg = {
									s_format("Average Intimidating Cry Damage:"),
									s_format("%.2f%% ^8(base double damage increase to hit 100%%)", (1 - ddChance / 100) * 100 ),
									s_format("x %d ^8(double damage multiplier)", 2),
									s_format("= %.2f", globalOutput.IntimidatingAvgDmg),
								}
							end
							globalOutput.IntimidatingHitEffect = 1 + globalOutput.IntimidatingAvgDmg * globalOutput.IntimidatingUpTimeRatio / 100
							globalOutput.IntimidatingMaxHitEffect = 1 + globalOutput.IntimidatingAvgDmg
							if globalBreakdown then
								globalBreakdown.IntimidatingHitEffect = {
									s_format("1 + (%.2f ^8(average exerted damage)", globalOutput.IntimidatingAvgDmg),
									s_format("x %.2f) ^8(uptime %%)", globalOutput.IntimidatingUpTimeRatio / 100),
									s_format("= %.2f", globalOutput.IntimidatingHitEffect),
								}
							end

							globalOutput.TheoreticalOffensiveWarcryEffect = globalOutput.TheoreticalOffensiveWarcryEffect * globalOutput.IntimidatingHitEffect
							globalOutput.TheoreticalMaxOffensiveWarcryEffect = globalOutput.TheoreticalMaxOffensiveWarcryEffect * globalOutput.IntimidatingMaxHitEffect
							globalOutput.IntimidatingCryCalculated = true
						elseif value.activeEffect.grantedEffect.name == "Rallying Cry" and activeSkill.skillTypes[SkillType.Melee] and not globalOutput.RallyingCryCalculated then
							globalOutput.CreateWarcryOffensiveCalcSection = true
							globalOutput.RallyingCryDuration = calcSkillDuration(value.skillModList, value.skillCfg, value.skillData, env, enemyDB)
							globalOutput.RallyingCryCooldown = calcSkillCooldown(value.skillModList, value.skillCfg, value.skillData)
							output.GlobalWarcryCooldown = env.modDB:Sum("BASE", nil, "GlobalWarcryCooldown")
							output.GlobalWarcryCount = env.modDB:Sum("BASE", nil, "GlobalWarcryCount")
							if modDB:Flag(nil, "WarcryShareCooldown") then
								globalOutput.RallyingCryCooldown = globalOutput.RallyingCryCooldown + (output.GlobalWarcryCooldown - globalOutput.RallyingCryCooldown) / output.GlobalWarcryCount
							end
							globalOutput.RallyingCryCastTime = calcWarcryCastTime(value.skillModList, value.skillCfg, actor)
							globalOutput.RallyingExertsCount = env.modDB:Sum("BASE", nil, "NumRallyingExerts") or 0
							local baseUptimeRatio = m_min((globalOutput.RallyingExertsCount / output.Speed) / (globalOutput.RallyingCryCooldown + globalOutput.RallyingCryCastTime), 1) * 100
							local additionalCooldownUses = value.skillModList:Sum("BASE", value.skillCfg, "AdditionalCooldownUses")
							globalOutput.RallyingUpTimeRatio = m_min(100, baseUptimeRatio * (additionalCooldownUses + 1))
							if globalBreakdown then
								globalBreakdown.RallyingUpTimeRatio = { }
								t_insert(globalBreakdown.RallyingUpTimeRatio, s_format("(%d ^8(number of exerts)", globalOutput.RallyingExertsCount))
								t_insert(globalBreakdown.RallyingUpTimeRatio, s_format("/ %.2f) ^8(attacks per second)", output.Speed))
								if 	globalOutput.RallyingCryCastTime > 0 then
									t_insert(globalBreakdown.RallyingUpTimeRatio, s_format("/ (%.2f ^8(warcry cooldown)", globalOutput.RallyingCryCooldown))
									t_insert(globalBreakdown.RallyingUpTimeRatio, s_format("+ %.2f) ^8(warcry casttime)", globalOutput.RallyingCryCastTime))
								else
									t_insert(globalBreakdown.RallyingUpTimeRatio, s_format("/ %.2f ^8(average warcry cooldown)", globalOutput.RallyingCryCooldown))
								end
								t_insert(globalBreakdown.RallyingUpTimeRatio, s_format("= %d%%", globalOutput.RallyingUpTimeRatio))
							end
							globalOutput.RallyingAvgDmg = m_min(env.modDB:Sum("BASE", cfg, "Multiplier:NearbyAlly"), 5) * (env.modDB:Sum("BASE", nil, "RallyingExertMoreDamagePerAlly") / 100)
							if globalBreakdown then
								globalBreakdown.RallyingAvgDmg = {
									s_format("Average Rallying Cry Damage:"),
									s_format("%.2f ^8(average damage multiplier per ally)", env.modDB:Sum("BASE", nil, "RallyingExertMoreDamagePerAlly") / 100),
									s_format("x %d ^8(number of nearby allies (max=5))", m_min(env.modDB:Sum("BASE", cfg, "Multiplier:NearbyAlly"), 5)),
									s_format("= %.2f", globalOutput.RallyingAvgDmg),
								}
							end
							globalOutput.RallyingHitEffect = 1 + globalOutput.RallyingAvgDmg * globalOutput.RallyingUpTimeRatio / 100
							globalOutput.RallyingMaxHitEffect = 1 + globalOutput.RallyingAvgDmg
							if globalBreakdown then
								globalBreakdown.RallyingHitEffect = {
									s_format("1 + (%.2f ^8(average exerted damage)", globalOutput.RallyingAvgDmg),
									s_format("x %.2f) ^8(uptime %%)", globalOutput.RallyingUpTimeRatio / 100),
									s_format("= %.2f", globalOutput.RallyingHitEffect),
								}
							end
							globalOutput.OffensiveWarcryEffect = globalOutput.OffensiveWarcryEffect * globalOutput.RallyingHitEffect
							globalOutput.MaxOffensiveWarcryEffect = globalOutput.MaxOffensiveWarcryEffect * globalOutput.RallyingMaxHitEffect
							globalOutput.TheoreticalOffensiveWarcryEffect = globalOutput.TheoreticalOffensiveWarcryEffect * globalOutput.RallyingHitEffect
							globalOutput.TheoreticalMaxOffensiveWarcryEffect = globalOutput.TheoreticalMaxOffensiveWarcryEffect * globalOutput.RallyingMaxHitEffect
							globalOutput.RallyingCryCalculated = true

						elseif value.activeEffect.grantedEffect.name == "Seismic Cry" and activeSkill.skillTypes[SkillType.Slam] and not globalOutput.SeismicCryCalculated then
							globalOutput.CreateWarcryOffensiveCalcSection = true
							globalOutput.SeismicCryDuration = calcSkillDuration(value.skillModList, value.skillCfg, value.skillData, env, enemyDB)
							globalOutput.SeismicCryCooldown = calcSkillCooldown(value.skillModList, value.skillCfg, value.skillData)
							output.GlobalWarcryCooldown = env.modDB:Sum("BASE", nil, "GlobalWarcryCooldown")
							output.GlobalWarcryCount = env.modDB:Sum("BASE", nil, "GlobalWarcryCount")
							if modDB:Flag(nil, "WarcryShareCooldown") then
								globalOutput.SeismicCryCooldown = globalOutput.SeismicCryCooldown + (output.GlobalWarcryCooldown - globalOutput.SeismicCryCooldown) / output.GlobalWarcryCount
							end
							globalOutput.SeismicCryCastTime = calcWarcryCastTime(value.skillModList, value.skillCfg, actor)
							globalOutput.SeismicExertsCount = env.modDB:Sum("BASE", nil, "NumSeismicExerts") or 0
							local baseUptimeRatio = m_min((globalOutput.SeismicExertsCount / output.Speed) / (globalOutput.SeismicCryCooldown + globalOutput.SeismicCryCastTime), 1) * 100
							local additionalCooldownUses = value.skillModList:Sum("BASE", value.skillCfg, "AdditionalCooldownUses")
							globalOutput.SeismicUpTimeRatio = m_min(100, baseUptimeRatio * (additionalCooldownUses + 1))
							if globalBreakdown then
								globalBreakdown.SeismicUpTimeRatio = { }
								t_insert(globalBreakdown.SeismicUpTimeRatio, s_format("(%d ^8(number of exerts)", globalOutput.SeismicExertsCount))
								t_insert(globalBreakdown.SeismicUpTimeRatio, s_format("/ %.2f) ^8(attacks per second)", output.Speed))
								if 	globalOutput.SeismicCryCastTime > 0 then
									t_insert(globalBreakdown.SeismicUpTimeRatio, s_format("/ (%.2f ^8(warcry cooldown)", globalOutput.SeismicCryCooldown))
									t_insert(globalBreakdown.SeismicUpTimeRatio, s_format("+ %.2f) ^8(warcry casttime)", globalOutput.SeismicCryCastTime))
								else
									t_insert(globalBreakdown.SeismicUpTimeRatio, s_format("/ %.2f ^8(average warcry cooldown)", globalOutput.SeismicCryCooldown))
								end
								t_insert(globalBreakdown.SeismicUpTimeRatio, s_format("= %d%%", globalOutput.SeismicUpTimeRatio))
							end
							-- calculate the stacking AoE modifier of Seismic slams
							local SeismicAoEPerExert = env.modDB:Sum("BASE", cfg, "SeismicIncAoEPerExert") / 100
							local AoEImpact = 0
							local MaxSingleAoEImpact = 0
							for i = 1, globalOutput.SeismicExertsCount do
								AoEImpact = AoEImpact + (i * SeismicAoEPerExert)
								MaxSingleAoEImpact = MaxSingleAoEImpact + SeismicAoEPerExert
							end
							local AvgAoEImpact = AoEImpact / globalOutput.SeismicExertsCount

							-- account for AoE increase
							if activeSkill.skillModList:Flag(nil, "Condition:WarcryMaxHit") then
								skillModList:NewMod("AreaOfEffect", "INC", MaxSingleAoEImpact * 100, "Max Seismic Exert AoE")
							else
								skillModList:NewMod("AreaOfEffect", "INC", m_floor(AvgAoEImpact * globalOutput.SeismicUpTimeRatio), "Avg Seismic Exert AoE")
							end
							calcAreaOfEffect(skillModList, skillCfg, skillData, skillFlags, globalOutput, globalBreakdown)
							globalOutput.SeismicCryCalculated = true
						end
					end

					if activeSkill.skillModList:Flag(nil, "Condition:WarcryMaxHit") then
						globalOutput.AilmentWarcryEffect = globalOutput.MaxOffensiveWarcryEffect
						skillData.showAverage = true
						skillFlags.showAverage = true
						skillFlags.notAverage = false
					else
						globalOutput.AilmentWarcryEffect = globalOutput.OffensiveWarcryEffect
					end

					-- Calculate Exerted Attack Uptime
					-- There are various strategies a player could use to maximize either warcry effect stacking or staggering
					-- 1) they don't pay attention and therefore we calculated exerted attack uptime as just the maximum uptime of any enabled warcries that exert attacks
					globalOutput.ExertedAttackUptimeRatio = m_max(m_max(m_max(globalOutput.AncestralUpTimeRatio or 0, globalOutput.InfernalUpTimeRatio or 0), m_max(globalOutput.IntimidatingUpTimeRatio or 0, globalOutput.RallyingUpTimeRatio or 0)), globalOutput.SeismicUpTimeRatio or 0)
					if globalBreakdown then
						globalBreakdown.ExertedAttackUptimeRatio = { }
						t_insert(globalBreakdown.ExertedAttackUptimeRatio, s_format("Maximum of:"))
						if globalOutput.AncestralUpTimeRatio then
							t_insert(globalBreakdown.ExertedAttackUptimeRatio, s_format("%d%% ^8(Ancestral Cry Uptime)", globalOutput.AncestralUpTimeRatio or 0))
						end
						if globalOutput.InfernalUpTimeRatio then
							t_insert(globalBreakdown.ExertedAttackUptimeRatio, s_format("%d%% ^8(Infernal Cry Uptime)", globalOutput.InfernalUpTimeRatio or 0))
						end
						if globalOutput.IntimidatingUpTimeRatio then
							t_insert(globalBreakdown.ExertedAttackUptimeRatio, s_format("%d%% ^8(Intimidating Cry Uptime)", globalOutput.IntimidatingUpTimeRatio or 0))
						end
						if globalOutput.RallyingUpTimeRatio then
							t_insert(globalBreakdown.ExertedAttackUptimeRatio, s_format("%d%% ^8(Rallying Cry Uptime)", globalOutput.RallyingUpTimeRatio or 0))
						end
						if globalOutput.SeismicUpTimeRatio then
							t_insert(globalBreakdown.ExertedAttackUptimeRatio, s_format("%d%% ^8(Seismic Cry Uptime)", globalOutput.SeismicUpTimeRatio or 0))
						end
						t_insert(globalBreakdown.ExertedAttackUptimeRatio, s_format("= %d%%", globalOutput.ExertedAttackUptimeRatio))
					end
					if globalOutput.ExertedAttackUptimeRatio > 0 then
						local incExertedAttacks = skillModList:Sum("INC", cfg, "ExertIncrease")
						local moreExertedAttacks = skillModList:Sum("MORE", cfg, "ExertIncrease")
						local moreExertedAttackDamage = skillModList:Sum("MORE", cfg, "ExertAttackIncrease")
						if activeSkill.skillModList:Flag(nil, "Condition:WarcryMaxHit") then
							skillModList:NewMod("Damage", "INC", incExertedAttacks, "Exerted Attacks")
							skillModList:NewMod("Damage", "MORE", moreExertedAttacks, "Exerted Attacks")
							skillModList:NewMod("Damage", "MORE", moreExertedAttackDamage, "Exerted Attack Damage", ModFlag.Attack)
						else
							skillModList:NewMod("Damage", "INC", incExertedAttacks * globalOutput.ExertedAttackUptimeRatio / 100, "Uptime Scaled Exerted Attacks")
							skillModList:NewMod("Damage", "MORE", moreExertedAttacks * globalOutput.ExertedAttackUptimeRatio / 100, "Uptime Scaled Exerted Attacks")
							skillModList:NewMod("Damage", "MORE", moreExertedAttackDamage * globalOutput.ExertedAttackUptimeRatio / 100, "Uptime Scaled Exerted Attack Damage", ModFlag.Attack)
						end
						globalOutput.ExertedAttackAvgDmg = calcLib.mod(skillModList, skillCfg, "ExertIncrease")
						globalOutput.ExertedAttackAvgDmg = globalOutput.ExertedAttackAvgDmg * calcLib.mod(skillModList, skillCfg, "ExertAttackIncrease")
						globalOutput.ExertedAttackHitEffect = globalOutput.ExertedAttackAvgDmg * globalOutput.ExertedAttackUptimeRatio / 100
						globalOutput.ExertedAttackMaxHitEffect = globalOutput.ExertedAttackAvgDmg
						if globalBreakdown then
							globalBreakdown.ExertedAttackHitEffect = {
								s_format("(%.2f ^8(average exerted damage)", globalOutput.ExertedAttackAvgDmg),
								s_format("x %.2f) ^8(uptime %%)", globalOutput.ExertedAttackUptimeRatio / 100),
								s_format("= %.2f", globalOutput.ExertedAttackHitEffect),
							}
						end
					end
				end
			end
		*/

		pass.Output["RuthlessBlowHitEffect"] = 1
		pass.Output["RuthlessBlowBleedEffect"] = 1
		pass.Output["FistOfWarHitEffect"] = 1
		pass.Output["FistOfWarAilmentEffect"] = 1

		/*
			TODO --
			if env.mode_combat then
				-- Calculate Ruthless Blow chance/multipliers + Fist of War multipliers
				output.RuthlessBlowMaxCount = skillModList:Sum("BASE", cfg, "RuthlessBlowMaxCount")
				if output.RuthlessBlowMaxCount > 0 then
					output.RuthlessBlowChance = round(100 / output.RuthlessBlowMaxCount)
				else
					output.RuthlessBlowChance = 0
				end
				output.RuthlessBlowHitMultiplier = 1 + skillModList:Sum("BASE", cfg, "RuthlessBlowHitMultiplier") / 100
				output.RuthlessBlowBleedMultiplier = 1 + skillModList:Sum("BASE", cfg, "RuthlessBlowBleedMultiplier") / 100
				output.RuthlessBlowHitEffect = 1 - output.RuthlessBlowChance / 100 + output.RuthlessBlowChance / 100 * output.RuthlessBlowHitMultiplier
				output.RuthlessBlowBleedEffect = 1 - output.RuthlessBlowChance / 100 + output.RuthlessBlowChance / 100 * output.RuthlessBlowBleedMultiplier

				globalOutput.FistOfWarCooldown = skillModList:Sum("BASE", cfg, "FistOfWarCooldown") or 0
				-- If Fist of War & Active Skill is a Slam Skill & NOT a Vaal Skill
				if globalOutput.FistOfWarCooldown ~= 0 and activeSkill.skillTypes[SkillType.Slam] and not activeSkill.skillTypes[SkillType.Vaal] then
					globalOutput.FistOfWarHitMultiplier = skillModList:Sum("BASE", cfg, "FistOfWarHitMultiplier") / 100
					globalOutput.FistOfWarAilmentMultiplier = skillModList:Sum("BASE", cfg, "FistOfWarAilmentMultiplier") / 100
					globalOutput.FistOfWarUptimeRatio = m_min( (1 / output.Speed) / globalOutput.FistOfWarCooldown, 1) * 100
					if globalBreakdown then
						globalBreakdown.FistOfWarUptimeRatio = {
							s_format("min( (1 / %.2f) ^8(second per attack)", output.Speed),
							s_format("/ %.2f, 1) ^8(fist of war cooldown)", globalOutput.FistOfWarCooldown),
							s_format("= %d%%", globalOutput.FistOfWarUptimeRatio),
						}
					end
					globalOutput.AvgFistOfWarHit = globalOutput.FistOfWarHitMultiplier
					globalOutput.AvgFistOfWarHitEffect = 1 + globalOutput.FistOfWarHitMultiplier * (globalOutput.FistOfWarUptimeRatio / 100)
					if globalBreakdown then
						globalBreakdown.AvgFistOfWarHitEffect = {
							s_format("1 + (%.2f ^8(fist of war hit multiplier)", globalOutput.FistOfWarHitMultiplier),
							s_format("x %.2f) ^8(fist of war uptime ratio)", globalOutput.FistOfWarUptimeRatio / 100),
							s_format("= %.2f", globalOutput.AvgFistOfWarHitEffect),
						}
					end
					globalOutput.AvgFistOfWarAilmentEffect = 1 + globalOutput.FistOfWarAilmentMultiplier * (globalOutput.FistOfWarUptimeRatio / 100)
					globalOutput.MaxFistOfWarHitEffect = 1 + globalOutput.FistOfWarHitMultiplier
					globalOutput.MaxFistOfWarAilmentEffect = 1 + globalOutput.FistOfWarAilmentMultiplier
					if activeSkill.skillModList:Flag(nil, "Condition:WarcryMaxHit") then
						output.FistOfWarHitEffect = globalOutput.MaxFistOfWarHitEffect
						output.FistOfWarAilmentEffect = globalOutput.MaxFistOfWarAilmentEffect
					else
						output.FistOfWarHitEffect = globalOutput.AvgFistOfWarHitEffect
						output.FistOfWarAilmentEffect = globalOutput.AvgFistOfWarAilmentEffect
					end
					globalOutput.TheoreticalOffensiveWarcryEffect = globalOutput.TheoreticalOffensiveWarcryEffect * globalOutput.AvgFistOfWarHitEffect
					globalOutput.TheoreticalMaxOffensiveWarcryEffect = globalOutput.TheoreticalMaxOffensiveWarcryEffect * globalOutput.MaxFistOfWarHitEffect
				else
					output.FistOfWarHitEffect = 1
					output.FistOfWarAilmentEffect = 1
				end
			end
		*/

		// Calculate crit chance, crit multiplier, and their combined effect
		if activeSkill.SkillModList.Flag(nil, "NeverCrit") {
			pass.Output["PreEffectiveCritChance"] = 0
			pass.Output["CritChance"] = 0
			pass.Output["CritMultiplier"] = 0
			pass.Output["BonusCritDotMultiplier"] = 0
			pass.Output["CritEffect"] = 1
		} else {
			baseCrit := float64(0)

			if pass.Source["CritChance"] != nil {
				baseCrit = pass.Source["CritChance"].(float64)
			}

			critOverride := activeSkill.SkillModList.Override(pass.Config, "CritChance")
			if critOverride != nil {
				baseCrit = critOverride.(float64)
			}

			if baseCrit == 100 {
				pass.Output["PreEffectiveCritChance"] = 100
				pass.Output["CritChance"] = 100
			} else {
				base := float64(0)
				inc := float64(0)
				more := float64(0)
				if critOverride == nil {
					base = activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "CritChance")
					inc = activeSkill.SkillModList.Sum(mod.TypeIncrease, pass.Config, "CritChance")
					more = activeSkill.SkillModList.More(pass.Config, "CritChance")

					if env.ModeEffective {
						base += actor.Enemy.ModDB.Sum(mod.TypeBase, nil, "SelfCritChance")
						inc += actor.Enemy.ModDB.Sum(mod.TypeIncrease, nil, "SelfCritChance")
					}
				}

				pass.Output["CritChance"] = (baseCrit + base) * (1 + inc/100) * more

				// For Breakdown
				// preCapCritChance := pass.Output["CritChance"]
				pass.Output["CritChance"] = math.Min(pass.Output["CritChance"], 100)

				if baseCrit+base > 0 {
					pass.Output["CritChance"] = math.Max(pass.Output["CritChance"], 0)
				}

				pass.Output["PreEffectiveCritChance"] = pass.Output["CritChance"]
				// For Breakdown
				// preLuckyCritChance := pass.Output["CritChance"]

				if env.ModeEffective && activeSkill.SkillModList.Flag(pass.Config, "CritChanceLucky") {
					pass.Output["CritChance"] = (1 - math.Pow(1-pass.Output["CritChance"]/100, 2)) * 100
				}

				// For Breakdown
				// preHitCheckCritChance := pass.Output["CritChance"]
				if env.ModeEffective {
					pass.Output["CritChance"] = pass.Output["CritChance"] * pass.Output["HitChance"] / 100
				}

				/*
					TODO Breakdown
					if breakdown and output.CritChance ~= baseCrit then
						breakdown.CritChance = { }
						if base ~= 0 then
							t_insert(breakdown.CritChance, s_format("(%g + %g) ^8(base)", baseCrit, base))
						else
							t_insert(breakdown.CritChance, s_format("%g ^8(base)", baseCrit + base))
						end
						if inc ~= 0 then
							t_insert(breakdown.CritChance, s_format("x %.2f", 1 + inc/100).." ^8(increased/reduced)")
						end
						if more ~= 1 then
							t_insert(breakdown.CritChance, s_format("x %.2f", more).." ^8(more/less)")
						end
						t_insert(breakdown.CritChance, s_format("= %.2f%% ^8(crit chance)", output.PreEffectiveCritChance))
						if preCapCritChance > 100 then
							local overCap = preCapCritChance - 100
							t_insert(breakdown.CritChance, s_format("Crit is overcapped by %.2f%% (%d%% increased Critical Strike Chance)", overCap, overCap / more / (baseCrit + base) * 100))
						end
						if env.mode_effective and skillModList:Flag(cfg, "CritChanceLucky") then
							t_insert(breakdown.CritChance, "Crit Chance is Lucky:")
							t_insert(breakdown.CritChance, s_format("1 - (1 - %.4f) x (1 - %.4f)", preLuckyCritChance / 100, preLuckyCritChance / 100))
							t_insert(breakdown.CritChance, s_format("= %.2f%%", preHitCheckCritChance))
						end
						if env.mode_effective and output.HitChance < 100 then
							t_insert(breakdown.CritChance, "Crit confirmation roll:")
							t_insert(breakdown.CritChance, s_format("%.2f%%", preHitCheckCritChance))
							t_insert(breakdown.CritChance, s_format("x %.2f ^8(chance to hit)", output.HitChance / 100))
							t_insert(breakdown.CritChance, s_format("= %.2f%%", output.CritChance))
						end
					end
				*/
			}

			if activeSkill.SkillModList.Flag(pass.Config, "NoCritMultiplier") {
				pass.Output["CritMultiplier"] = 1
			} else {
				extraDamage := activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplier") / 100
				multiOverride := activeSkill.SkillModList.Override(activeSkill.SkillCfg, "CritMultiplier")
				if multiOverride != nil {
					extraDamage = (multiOverride.(float64) - 100) / 100
				}

				if env.ModeEffective {
					/*
						TODO Breakdown
						local enemyInc = 1 + enemyDB:Sum("INC", nil, "SelfCritMultiplier") / 100
						extraDamage = extraDamage + enemyDB:Sum("BASE", nil, "SelfCritMultiplier") / 100
						extraDamage = round(extraDamage * enemyInc, 2)
						if breakdown and enemyInc ~= 1 then
							breakdown.CritMultiplier = {
								s_format("%d%% ^8(additional extra damage)", (enemyDB:Sum("BASE", nil, "SelfCritMultiplier") + skillModList:Sum("BASE", cfg, "CritMultiplier")) / 100),
								s_format("x %.2f ^8(increased/reduced extra crit damage taken by enemy)", enemyInc),
								s_format("= %d%% ^8(extra crit damage)", extraDamage * 100),
							}
						end
					*/
				}

				pass.Output["CritMultiplier"] = 1 + math.Max(0, extraDamage)
			}

			critChancePercentage := pass.Output["CritChance"] / 100
			pass.Output["CritEffect"] = 1 - critChancePercentage + critChancePercentage*pass.Output["CritMultiplier"]
			pass.Output["CritEffect"] = (activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplier") - 50) * activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplierAppliesToDegen") / 1000

			/*
				TODO Breakdown
				if breakdown and output.CritEffect ~= 1 then
					breakdown.CritEffect = {
						s_format("(1 - %.4f) ^8(portion of damage from non-crits)", critChancePercentage),
						s_format("+ [ (%.4f x %g) ^8(portion of damage from crits)", critChancePercentage, output.CritMultiplier),
						s_format("= %.3f", output.CritEffect),
					}
				end
			*/
		}

		pass.Output["ScaledDamageEffect"] = 1

		/*
			TODO -- Calculate chance and multiplier for dealing triple damage on Normal and Crit
			output.TripleDamageChanceOnCrit = m_min(skillModList:Sum("BASE", cfg, "TripleDamageChanceOnCrit"), 100)
			output.TripleDamageChance = m_min(skillModList:Sum("BASE", cfg, "TripleDamageChance") or 0 + (env.mode_effective and enemyDB:Sum("BASE", cfg, "SelfTripleDamageChance") or 0) + (output.TripleDamageChanceOnCrit * output.CritChance / 100), 100)
			output.TripleDamageEffect = 1 + (2 * output.TripleDamageChance / 100)
			output.ScaledDamageEffect = output.ScaledDamageEffect * output.TripleDamageEffect
		*/
		/*
			TODO -- Calculate chance and multiplier for dealing double damage on Normal and Crit
			output.DoubleDamageChanceOnCrit = m_min(skillModList:Sum("BASE", cfg, "DoubleDamageChanceOnCrit"), 100)
			output.DoubleDamageChance = m_min(skillModList:Sum("BASE", cfg, "DoubleDamageChance") + (env.mode_effective and enemyDB:Sum("BASE", cfg, "SelfDoubleDamageChance") or 0) + (output.DoubleDamageChanceOnCrit * output.CritChance / 100), 100)
			if globalOutput.IntimidatingUpTimeRatio and activeSkill.skillModList:Flag(nil, "Condition:WarcryMaxHit") then
				output.DoubleDamageChance = 100
			elseif globalOutput.IntimidatingUpTimeRatio then
				output.DoubleDamageChance = m_min(output.DoubleDamageChance + globalOutput.IntimidatingUpTimeRatio, 100)
			end
		*/
		/*
			TODO -- Triple Damage overrides Double Damage. If you have both, it's the same as just having Triple
			-- We need to subtract the probability of both happening in favor of Triple Damage
			if output.TripleDamageChance > 0 then
				output.DoubleDamageChance = m_max(output.DoubleDamageChance - output.TripleDamageChance * output.DoubleDamageChance / 100, 0)
			end
			output.DoubleDamageEffect = 1 + output.DoubleDamageChance / 100
			output.ScaledDamageEffect = output.ScaledDamageEffect * output.DoubleDamageEffect
		*/
		/*
			TODO -- Calculate culling DPS
			local criticalCull = skillModList:Max(cfg, "CriticalCullPercent") or 0
			if criticalCull > 0 then
				criticalCull = criticalCull * (output.CritChance / 100)
			end
			local regularCull = skillModList:Max(cfg, "CullPercent") or 0
			local maxCullPercent = m_max(criticalCull, regularCull)
			globalOutput.CullPercent = maxCullPercent
			globalOutput.CullMultiplier = 100 / (100 - globalOutput.CullPercent)
		*/

		// Calculate base hit damage
		for _, damageType := range data.DamageType("").Values() {
			damageTypeMin := string(damageType) + "Min"
			damageTypeMax := string(damageType) + "Max"

			baseMultiplier := float64(1)
			if activeSkill.ActiveEffect.GrantedEffect.BaseMultiplier != nil {
				baseMultiplier = *activeSkill.ActiveEffect.GrantedEffect.BaseMultiplier
			} else if activeSkill.SkillData["BaseMultiplier"] != nil {
				baseMultiplier = activeSkill.SkillData["BaseMultiplier"].(float64)
			}

			damageEffectiveness := float64(1)
			if activeSkill.ActiveEffect.GrantedEffect.DamageEffectiveness != nil {
				damageEffectiveness = *activeSkill.ActiveEffect.GrantedEffect.DamageEffectiveness
			} else if activeSkill.SkillData["DamageEffectiveness"] != nil {
				damageEffectiveness = activeSkill.SkillData["DamageEffectiveness"].(float64)
			}

			addedMin := activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, damageTypeMin) + actor.Enemy.ModDB.Sum(mod.TypeBase, pass.Config, "Self"+damageTypeMin)
			addedMax := activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, damageTypeMax) + actor.Enemy.ModDB.Sum(mod.TypeBase, pass.Config, "Self"+damageTypeMax)
			addedMult := CalcMod(activeSkill.SkillModList, pass.Config, "Added"+string(damageType)+"Damage", "AddedDamage")

			baseMin := (utils.GetOr(pass.Source, damageTypeMin, utils.Interface(float64(0))).(float64)+
				utils.GetOr(pass.Source, string(damageType)+"BonusMin", utils.Interface(float64(0))).(float64))*
				baseMultiplier + addedMin*damageEffectiveness*addedMult

			baseMax := (utils.GetOr(pass.Source, damageTypeMax, utils.Interface(float64(0))).(float64)+
				utils.GetOr(pass.Source, string(damageType)+"BonusMax", utils.Interface(float64(0))).(float64))*
				baseMultiplier + addedMax*damageEffectiveness*addedMult

			pass.Output[damageTypeMin+"Base"] = baseMin
			pass.Output[damageTypeMax+"Base"] = baseMax

			/*
				TODO Breakdown
				if breakdown then
					breakdown[damageType] = { damageTypes = { } }
					if baseMin ~= 0 and baseMax ~= 0 then
						t_insert(breakdown[damageType], "Base damage:")
						local plus = ""
						if (source[damageTypeMin] or 0) ~= 0 or (source[damageTypeMax] or 0) ~= 0 then
							t_insert(breakdown[damageType], s_format("%d to %d ^8(base damage from %s)", source[damageTypeMin], source[damageTypeMax], source.type and "weapon" or "skill"))
							if baseMultiplier ~= 1 then
								t_insert(breakdown[damageType], s_format("x %.2f ^8(base damage multiplier)", baseMultiplier))
							end
							plus = "+ "
						end
						if addedMin ~= 0 or addedMax ~= 0 then
							t_insert(breakdown[damageType], s_format("%s%d to %d ^8(added damage)", plus, addedMin, addedMax))
							if damageEffectiveness ~= 1 then
								t_insert(breakdown[damageType], s_format("x %.2f ^8(damage effectiveness)", damageEffectiveness))
							end
							if addedMult ~= 1 then
								t_insert(breakdown[damageType], s_format("x %.2f ^8(added damage multiplier)", addedMult))
							end
						end
						t_insert(breakdown[damageType], s_format("= %.1f to %.1f", baseMin, baseMax))
					end
				end
			*/
		}

		totalHitMin := float64(0)
		totalHitMax := float64(0)
		totalHitAvg := float64(0)

		totalCritMin := float64(0)
		totalCritMax := float64(0)
		totalCritAvg := float64(0)

		ghostReaver := activeSkill.SkillModList.Flag(nil, "GhostReaver")

		pass.Output["LifeLeech"] = 0
		pass.Output["LifeLeechInstant"] = 0
		pass.Output["EnergyShieldLeech"] = 0
		pass.Output["EnergyShieldLeechInstant"] = 0
		pass.Output["ManaLeech"] = 0
		pass.Output["ManaLeechInstant"] = 0
		pass.Output["ImpaleStoredHitAvg"] = 0

		// Calculate hit damage for each damage type
		// TODO(post) Going from 1-2 for legacy Lua reasons. Should probably be changed
		for p := 1; p <= 2; p++ {
			// Pass 1 is critical strike damage, pass 2 is non-critical strike
			pass.Config.SkillCond["CriticalStrike"] = p == 1
			lifeLeechTotal := float64(0)
			energyShieldLeechTotal := float64(0)
			manaLeechTotal := float64(0)

			noLifeLeech := activeSkill.SkillModList.Flag(pass.Config, "CannotLeechLife") || actor.Enemy.ModDB.Flag(nil, "CannotLeechLifeFromSelf")
			noEnergyShieldLeech := activeSkill.SkillModList.Flag(pass.Config, "CannotLeechEnergyShield") || actor.Enemy.ModDB.Flag(nil, "CannotLeechEnergyShieldFromSelf")
			noManaLeech := activeSkill.SkillModList.Flag(pass.Config, "CannotLeechMana") || actor.Enemy.ModDB.Flag(nil, "CannotLeechManaFromSelf")

			for _, damageType := range data.DamageType("").Values() {
				damageTypeHitMin := float64(0)
				damageTypeHitMax := float64(0)
				damageTypeHitAvg := float64(0)
				damageTypeLuckyChance := float64(0)
				damageTypeHitAvgLucky := float64(0)
				damageTypeHitAvgNotLucky := float64(0)

				if utils.HasTrue(activeSkill.SkillFlags, SkillFlagHit) && utils.HasTrue(canDeal, damageType) {
					damageTypeHitMin, damageTypeHitMax = calcDamage(activeSkill, pass.Output, pass.Config, nil, damageType, 0, nil)
					convMult := activeSkill.ConversionTable[damageType].Mult

					/*
						TODO Breakdown
						if pass == 2 and breakdown then
							t_insert(breakdown[damageType], "Hit damage:")
							t_insert(breakdown[damageType], s_format("%d to %d ^8(total damage)", damageTypeHitMin, damageTypeHitMax))
							if convMult ~= 1 then
								t_insert(breakdown[damageType], s_format("x %g ^8(%g%% converted to other damage types)", convMult, (1-convMult)*100))
							end
							if output.TripleDamageEffect ~= 1 then
								t_insert(breakdown[damageType], s_format("x %.2f ^8(multiplier from %.2f%% chance to deal triple damage)", output.TripleDamageEffect, output.TripleDamageChance))
							end
							if output.DoubleDamageEffect ~= 1 then
								t_insert(breakdown[damageType], s_format("x %.2f ^8(multiplier from %.2f%% chance to deal double damage)", output.DoubleDamageEffect, output.DoubleDamageChance))
							end
							if output.RuthlessBlowHitEffect ~= 1 then
								t_insert(breakdown[damageType], s_format("x %.2f ^8(ruthless blow effect modifier)", output.RuthlessBlowHitEffect))
							end
							if output.FistOfWarHitEffect ~= 1 then
								t_insert(breakdown[damageType], s_format("x %.2f ^8(fist of war effect modifier)", output.FistOfWarHitEffect))
							end
							if globalOutput.OffensiveWarcryEffect ~= 1  and not activeSkill.skillModList:Flag(nil, "Condition:WarcryMaxHit") then
								t_insert(breakdown[damageType], s_format("x %.2f ^8(aggregated warcry exerted effect modifier)", globalOutput.OffensiveWarcryEffect))
							end
							if globalOutput.MaxOffensiveWarcryEffect ~= 1 and activeSkill.skillModList:Flag(nil, "Condition:WarcryMaxHit") then
								t_insert(breakdown[damageType], s_format("x %.2f ^8(aggregated max warcry exerted effect modifier)", globalOutput.MaxOffensiveWarcryEffect))
							end
						end
					*/

					if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
						pass.Output["AllMult"] = convMult * pass.Output["ScaledDamageEffect"] * pass.Output["RuthlessBlowHitEffect"] * pass.Output["FistOfWarHitEffect"] * actor.Output["MaxOffensiveWarcryEffect"]
					} else {
						pass.Output["AllMult"] = convMult * pass.Output["ScaledDamageEffect"] * pass.Output["RuthlessBlowHitEffect"] * pass.Output["FistOfWarHitEffect"] * actor.Output["OffensiveWarcryEffect"]
					}

					allMult := pass.Output["AllMult"]
					if p == 1 {
						// Apply crit multiplier
						allMult *= pass.Output["CritMultiplier"]
					}

					damageTypeHitMin *= allMult
					damageTypeHitMax *= allMult

					if activeSkill.SkillModList.Flag(activeSkill.SkillCfg, "LuckyHits") ||
						(p == 2 && damageType == data.DamageTypeLightning && activeSkill.SkillModList.Flag(activeSkill.SkillCfg, "LightningNoCritLucky")) ||
						(p == 1 && activeSkill.SkillModList.Flag(activeSkill.SkillCfg, "CritLucky")) ||
						((damageType == data.DamageTypeLightning || damageType == data.DamageTypeCold || damageType == data.DamageTypeFire) && activeSkill.SkillModList.Flag(activeSkill.SkillCfg, "ElementalLuckHits")) {
						damageTypeLuckyChance = 1
					} else {
						damageTypeLuckyChance = math.Min(activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "LuckyHitsChance"), 100) / 100
					}

					damageTypeHitAvgNotLucky = damageTypeHitMin/2 + damageTypeHitMax/2
					damageTypeHitAvgLucky = damageTypeHitMin/3 + 2*damageTypeHitMax/3
					damageTypeHitAvg = damageTypeHitAvgNotLucky*(1-damageTypeLuckyChance) + damageTypeHitAvgLucky*damageTypeLuckyChance

					if (damageTypeHitMin != 0 || damageTypeHitMax != 0) && env.ModeEffective {
						// Apply enemy resistances and damage taken modifiers
						resist := float64(0)
						pen := float64(0)
						// TODO Breakdown
						// sourceRes := data.DamageType("")
						takenInc := actor.Enemy.ModDB.Sum(mod.TypeIncrease, pass.Config, "DamageTaken", string(damageType)+"DamageTaken")
						takenMore := actor.Enemy.ModDB.More(pass.Config, "DamageTaken", string(damageType)+"DamageTaken")

						// Check if player is supposed to ignore a damage type, or if it's ignored on enemy side
						useThisResist := func(damageType data.DamageType) bool {
							names := []string{"Ignore" + string(damageType) + "Resistance"}
							if damageType.IsElemental() {
								names = append(names, "IgnoreElementalResistances")
							}
							return !activeSkill.SkillModList.Flag(pass.Config, names...) && !actor.Enemy.ModDB.Flag(nil, "SelfIgnore"+string(damageType)+"Resistance")
						}

						if damageType == data.DamageTypePhysical {
							// store pre-armour physical damage from attacks for impale calculations
							if p == 1 {
								pass.Output["ImpaleStoredHitAvg"] = pass.Output["ImpaleStoredHitAvg"] + damageTypeHitAvg*(pass.Output["CritChance"]/100)
							} else {
								pass.Output["ImpaleStoredHitAvg"] = pass.Output["ImpaleStoredHitAvg"] + damageTypeHitAvg*(1-pass.Output["CritChance"]/100)
							}
							enemyArmour := math.Max(CalcVal(actor.Enemy.ModDB, "Armour", nil), 0)
							armourReduction := CalcArmourReductionF(enemyArmour, damageTypeHitAvg)
							if activeSkill.SkillModList.Flag(pass.Config, "IgnoreEnemyPhysicalDamageReduction") {
								resist = 0
							} else {
								resist = math.Min(math.Max(0, actor.Enemy.ModDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction")+activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "EnemyPhysicalDamageReduction")+armourReduction), data.DamageReductionCap)
							}
						} else {
							if (activeSkill.SkillModList.Flag(pass.Config, "ChaosDamageUsesLowestResistance") && damageType == data.DamageTypeChaos) ||
								(activeSkill.SkillModList.Flag(pass.Config, "ElementalDamageUsesLowestResistance") && damageType.IsElemental()) {
								// Default to using the current damage type
								elementUsed := damageType
								if damageType.IsElemental() {
									resist = math.Min(actor.Enemy.ModDB.Sum(mod.TypeBase, nil, string(damageType)+"Resist", "ElementalResist")*CalcMod(actor.Enemy.ModDB, nil, string(damageType)+"Resist", "ElementalResist"), data.EnemyMaxResist)
									takenInc += actor.Enemy.ModDB.Sum(mod.TypeIncrease, pass.Config, "ElementalDamageTaken")
								} else if damageType == data.DamageTypeChaos {
									resist = math.Min(actor.Enemy.ModDB.Sum(mod.TypeBase, nil, "ChaosResist")*CalcMod(actor.Enemy.ModDB, nil, "ChaosResist"), data.EnemyMaxResist)
								}

								// Find the lowest resist of all the elements and use that if it's lower
								for _, eleDamageType := range data.DamageType("").Values() {
									if eleDamageType.IsElemental() && useThisResist(eleDamageType) && damageType != eleDamageType {
										currentElementResist := math.Min(actor.Enemy.ModDB.Sum(mod.TypeBase, nil, string(eleDamageType)+"Resist", "ElementalResist")*CalcMod(actor.Enemy.ModDB, nil, string(eleDamageType)+"Resist", "ElementalResist"), data.EnemyMaxResist)
										// If it's explicitly lower, then use the resist and update which element we're using to account for penetration
										if resist > currentElementResist {
											resist = currentElementResist
											elementUsed = eleDamageType
										}
									}
								}

								// Update the penetration based on the element used
								if elementUsed.IsElemental() {
									pen = activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, string(elementUsed)+"Penetration", "ElementalPenetration")
								} else if elementUsed == data.DamageTypeChaos {
									pen = activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "ChaosPenetration")
								}
								// TODO Breakdown
								// sourceRes = elementUsed
							} else if damageType.IsElemental() {
								resist = actor.Enemy.ModDB.Sum(mod.TypeBase, nil, string(damageType)+"Resist")
								if env.ModDB.Flag(nil, "Enemy"+string(damageType)+"ResistEqualToYours") {
									resist = env.Player.Output[string(damageType)+"Resist"]
								} else {
									base := resist + actor.Enemy.ModDB.Sum(mod.TypeBase, nil, "ElementalResist")
									resist = base * CalcMod(actor.Enemy.ModDB, nil, string(damageType)+"Resist")
								}
								pen = activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, string(damageType)+"Penetration", "ElementalPenetration")
								takenInc += actor.Enemy.ModDB.Sum(mod.TypeIncrease, pass.Config, "ElementalDamageTaken")
							} else if damageType == data.DamageTypeChaos {
								resist = actor.Enemy.ModDB.Sum(mod.TypeBase, nil, string(damageType)+"Resist")
								pen = activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "ChaosPenetration")
							}

							resist = math.Max(math.Min(resist, data.EnemyMaxResist), data.ResistFloor)
						}

						if utils.HasTrue(activeSkill.SkillFlags, SkillFlagProjectile) {
							takenInc += actor.Enemy.ModDB.Sum(mod.TypeIncrease, nil, "ProjectileDamageTaken")
						}

						if utils.HasTrue(activeSkill.SkillFlags, SkillFlagProjectile) && utils.HasTrue(activeSkill.SkillFlags, SkillFlagAttack) {
							takenInc += actor.Enemy.ModDB.Sum(mod.TypeIncrease, nil, "ProjectileAttackDamageTaken")
						}

						if utils.HasTrue(activeSkill.SkillFlags, SkillFlagTrap) || utils.HasTrue(activeSkill.SkillFlags, SkillFlagMine) {
							takenInc += actor.Enemy.ModDB.Sum(mod.TypeIncrease, nil, "TrapMineDamageTaken")
						}

						effMult := (1 + takenInc/100) * takenMore
						// TODO Breakdown
						// useRes := useThisResist(damageType)
						if damageType.IsElemental() && activeSkill.SkillModList.Flag(pass.Config, "CannotElePenIgnore") {
							effMult *= 1 - resist/100
						} else {
							effMult *= 1 - (resist-pen)/100
						}

						damageTypeHitMin = damageTypeHitMin * effMult
						damageTypeHitMax = damageTypeHitMax * effMult
						damageTypeHitAvg = damageTypeHitAvg * effMult

						if env.Mode == OutputModeCalcs {
							pass.Output[string(damageType)+"EffMult"] = effMult
						}
						/*
							TODO Breakdown
							if pass == 2 and breakdown and (effMult ~= 1 or sourceRes ~= 0) and skillModList:Flag(cfg, isElemental[damageType] and "CannotElePenIgnore" or nil) then
								t_insert(breakdown[damageType], s_format("x %.3f ^8(effective DPS modifier)", effMult))
								breakdown[damageType.."EffMult"] = breakdown.effMult(damageType, resist, 0, takenInc, effMult, takenMore, sourceRes, useRes)
							elseif pass == 2 and breakdown and (effMult ~= 1 or sourceRes ~= 0) then
								t_insert(breakdown[damageType], s_format("x %.3f ^8(effective DPS modifier)", effMult))
								breakdown[damageType.."EffMult"] = breakdown.effMult(damageType, resist, pen, takenInc, effMult, takenMore, sourceRes, useRes)
							end
						*/
					}
					/*
						TODO Breakdown
						if pass == 2 and breakdown then
							t_insert(breakdown[damageType], s_format("= %d to %d", damageTypeHitMin, damageTypeHitMax))
						end
					*/

					// Beginning of Leech Calculation for this DamageType
					if utils.HasTrue(activeSkill.SkillFlags, SkillFlagMine) || utils.HasTrue(activeSkill.SkillFlags, SkillFlagTrap) || utils.HasTrue(activeSkill.SkillFlags, SkillFlagTotem) {
						if !noLifeLeech {
							lifeLeech := activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "DamageLifeLeechToPlayer")
							if lifeLeech > 0 {
								lifeLeechTotal += damageTypeHitAvg * lifeLeech / 100
							}
						}
					} else {
						if !noLifeLeech {
							lifeLeech := float64(0)
							if activeSkill.SkillModList.Flag(nil, "LifeLeechBasedOnChaosDamage") {
								if damageType == data.DamageTypeChaos {
									lifeLeech = activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, "DamageLeech", "DamageLifeLeech", "PhysicalDamageLifeLeech", "LightningDamageLifeLeech", "ColdDamageLifeLeech", "FireDamageLifeLeech", "ChaosDamageLifeLeech", "ElementalDamageLifeLeech") + actor.Enemy.ModDB.Sum(mod.TypeBase, pass.Config, "SelfDamageLifeLeech")/100
								} else {
									lifeLeech = 0
								}
							} else {
								names := []string{"DamageLeech", "DamageLifeLeech", string(damageType) + "DamageLifeLeech"}
								if damageType.IsElemental() {
									names = append(names, "ElementalDamageLifeLeech")
								}
								lifeLeech = activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, names...) + actor.Enemy.ModDB.Sum(mod.TypeBase, pass.Config, "SelfDamageLifeLeech")/100
							}
							if lifeLeech > 0 {
								lifeLeechTotal = lifeLeechTotal + damageTypeHitAvg*lifeLeech/100
							}
						}

						if !noEnergyShieldLeech {
							names := []string{"DamageEnergyShieldLeech", string(damageType) + "DamageEnergyShieldLeech"}
							if damageType.IsElemental() {
								names = append(names, "ElementalDamageEnergyShieldLeech")
							}
							energyShieldLeech := activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, names...) + actor.Enemy.ModDB.Sum(mod.TypeBase, pass.Config, "SelfDamageEnergyShieldLeech")/100
							if energyShieldLeech > 0 {
								energyShieldLeechTotal = energyShieldLeechTotal + damageTypeHitAvg*energyShieldLeech/100
							}
						}

						if !noManaLeech {
							names := []string{"DamageManaLeech", string(damageType) + "DamageManaLeech"}
							if damageType.IsElemental() {
								names = append(names, "ElementalDamageManaLeech")
							}
							manaLeech := activeSkill.SkillModList.Sum(mod.TypeBase, pass.Config, names...) + actor.Enemy.ModDB.Sum(mod.TypeBase, pass.Config, "SelfDamageManaLeech")/100
							if manaLeech > 0 {
								manaLeechTotal = manaLeechTotal + damageTypeHitAvg*manaLeech/100
							}
						}
					}
				} else {
					/*
						TODO Breakdown
						if breakdown then
							breakdown[damageType] = {
								"You can't deal "..damageType.." damage"
							}
						end
					*/
				}

				if p == 1 {
					pass.Output[string(damageType)+"CritAverage"] = damageTypeHitAvg
					totalCritAvg = totalCritAvg + damageTypeHitAvg
					totalCritMin = totalCritMin + damageTypeHitMin
					totalCritMax = totalCritMax + damageTypeHitMax
				} else {
					if env.Mode == "CALCS" {
						pass.Output[string(damageType)+"Min"] = damageTypeHitMin
						pass.Output[string(damageType)+"Max"] = damageTypeHitMax
					}
					pass.Output[string(damageType)+"HitAverage"] = damageTypeHitAvg
					totalHitAvg = totalHitAvg + damageTypeHitAvg
					totalHitMin = totalHitMin + damageTypeHitMin
					totalHitMax = totalHitMax + damageTypeHitMax
				}
			}

			if utils.Has(activeSkill.SkillData, "LifeLeechPerUse") {
				lifeLeechTotal += activeSkill.SkillData["LifeLeechPerUse"].(float64)
			}

			if utils.Has(activeSkill.SkillData, "ManaLeechPerUse") {
				manaLeechTotal += activeSkill.SkillData["ManaLeechPerUse"].(float64)
			}

			portion := 1 - pass.Output["CritChance"]/100
			if p == 1 {
				portion = pass.Output["CritChance"] / 100
			}

			if activeSkill.SkillModList.Flag(pass.Config, "InstantLifeLeech") && !ghostReaver {
				pass.Output["LifeLeechInstant"] += lifeLeechTotal * portion
			} else {
				pass.Output["LifeLeech"] += lifeLeechTotal * portion
			}

			if activeSkill.SkillModList.Flag(pass.Config, "InstantEnergyShieldLeech") {
				pass.Output["EnergyShieldLeechInstant"] += energyShieldLeechTotal * portion
			} else {
				pass.Output["EnergyShieldLeech"] += energyShieldLeechTotal * portion
			}

			if activeSkill.SkillModList.Flag(pass.Config, "InstantManaLeech") {
				pass.Output["ManaLeechInstant"] += manaLeechTotal * portion
			} else {
				pass.Output["ManaLeech"] += manaLeechTotal * portion
			}
		}

		pass.Output["TotalMin"] = totalHitMin
		pass.Output["TotalMax"] = totalHitMax

		/*
			TODO ElementalEquilibrium
			if skillModList:Flag(skillCfg, "ElementalEquilibrium") and not env.configInput.EEIgnoreHitDamage and (output.FireHitAverage + output.ColdHitAverage + output.LightningHitAverage > 0) then
				-- Update enemy hit-by-damage-type conditions
				enemyDB.conditions.HitByFireDamage = output.FireHitAverage > 0
				enemyDB.conditions.HitByColdDamage = output.ColdHitAverage > 0
				enemyDB.conditions.HitByLightningDamage = output.LightningHitAverage > 0
			end
		*/
		/*
			local highestType = "Physical"
			TODO -- For each damage type, calculate percentage of total damage. Also tracks the highest damage type and outputs a Condition:TypeIsHighestDamageType flag for whichever the highest type is
			for _, damageType in ipairs(dmgTypeList) do
				if output[damageType.."HitAverage"] > 0 then
					local portion = output[damageType.."HitAverage"] / totalHitAvg * 100
					local highestPortion = output[highestType.."HitAverage"] / totalHitAvg * 100
					if portion > highestPortion then
						highestType = damageType
						highestPortion = portion
					end
					if breakdown then
						t_insert(breakdown[damageType], s_format("Portion of total damage: %d%%", portion))
					end
				end
			end
			skillModList:NewMod("Condition:"..highestType.."IsHighestDamageType", "FLAG", true, "Config")

			local hitRate = output.HitChance / 100 * (globalOutput.HitSpeed or globalOutput.Speed) * (skillData.dpsMultiplier or 1)
		*/
		/*
			TODO -- Calculate leech
			local function getLeechInstances(amount, total)
				if total == 0 then
					return 0, 0
				end
				local duration = amount / total / data.misc.LeechRateBase
				return duration, duration * hitRate
			end
			if ghostReaver then
				output.EnergyShieldLeech = output.EnergyShieldLeech + output.LifeLeech
				output.EnergyShieldLeechInstant = output.EnergyShieldLeechInstant + output.LifeLeechInstant
				output.LifeLeech = 0
				output.LifeLeechInstant = 0
			end
			output.LifeLeech = m_min(output.LifeLeech, globalOutput.MaxLifeLeechInstance)
			output.LifeLeechDuration, output.LifeLeechInstances = getLeechInstances(output.LifeLeech, globalOutput.Life)
			output.LifeLeechInstantRate = output.LifeLeechInstant * hitRate
			output.EnergyShieldLeech = m_min(output.EnergyShieldLeech, globalOutput.MaxEnergyShieldLeechInstance)
			output.EnergyShieldLeechDuration, output.EnergyShieldLeechInstances = getLeechInstances(output.EnergyShieldLeech, globalOutput.EnergyShield)
			output.EnergyShieldLeechInstantRate = output.EnergyShieldLeechInstant * hitRate
			output.ManaLeech = m_min(output.ManaLeech, globalOutput.MaxManaLeechInstance)
			output.ManaLeechDuration, output.ManaLeechInstances = getLeechInstances(output.ManaLeech, globalOutput.Mana)
			output.ManaLeechInstantRate = output.ManaLeechInstant * hitRate
		*/
		/*
			TODO -- Calculate gain on hit
			if skillFlags.mine or skillFlags.trap or skillFlags.totem then
				output.LifeOnHit = 0
				output.EnergyShieldOnHit = 0
				output.ManaOnHit = 0
			else
				output.LifeOnHit = skillModList:Sum("BASE", cfg, "LifeOnHit") + enemyDB:Sum("BASE", cfg, "SelfLifeOnHit")
				output.EnergyShieldOnHit = skillModList:Sum("BASE", cfg, "EnergyShieldOnHit") + enemyDB:Sum("BASE", cfg, "SelfEnergyShieldOnHit")
				output.ManaOnHit = skillModList:Sum("BASE", cfg, "ManaOnHit") + enemyDB:Sum("BASE", cfg, "SelfManaOnHit")
			end
			output.LifeOnHitRate = output.LifeOnHit * hitRate
			output.EnergyShieldOnHitRate = output.EnergyShieldOnHit * hitRate
			output.ManaOnHitRate = output.ManaOnHit * hitRate
		*/

		// Calculate average damage and final DPS
		pass.Output["AverageHit"] = totalHitAvg*(1-pass.Output["CritChance"]/100) + totalCritAvg*pass.Output["CritChance"]/100
		pass.Output["AverageDamage"] = pass.Output["AverageHit"] * pass.Output["HitChance"] / 100

		selectedSpeed, gotSpeed := actor.Output["HitSpeed"]
		if !gotSpeed || selectedSpeed == 0 {
			selectedSpeed = actor.Output["Speed"]
		}

		AAA := pass.Output["AverageDamage"]
		BBB := utils.GetOr(activeSkill.SkillData, "DpsMultiplier", utils.Interface(float64(1))).(float64)
		pass.Output["TotalDPS"] = AAA * selectedSpeed * BBB * quantityMultiplier
		/*
			TODO Breakdown
			if breakdown then
				if output.CritEffect ~= 1 then
					breakdown.AverageHit = { }
					if skillModList:Flag(skillCfg, "LuckyHits") then
						t_insert(breakdown.AverageHit, s_format("(1/3) x %d + (2/3) x %d = %.1f ^8(average from non-crits)", totalHitMin, totalHitMax, totalHitAvg))
					end
					if skillModList:Flag(skillCfg, "CritLucky") or skillModList:Flag(skillCfg, "LuckyHits") then
						t_insert(breakdown.AverageHit, s_format("(1/3) x %d + (2/3) x %d = %.1f ^8(average from crits)", totalCritMin, totalCritMax, totalCritAvg))
						t_insert(breakdown.AverageHit, "")
					end
					t_insert(breakdown.AverageHit, s_format("%.1f x (1 - %.4f) ^8(damage from non-crits)", totalHitAvg, output.CritChance / 100))
					t_insert(breakdown.AverageHit, s_format("+ %.1f x %.4f ^8(damage from crits)", totalCritAvg, output.CritChance / 100))
					t_insert(breakdown.AverageHit, s_format("= %.1f", output.AverageHit))
				end
				if isAttack then
					breakdown.AverageDamage = { }
					t_insert(breakdown.AverageDamage, s_format("%s:", pass.label))
					t_insert(breakdown.AverageDamage, s_format("%.1f ^8(average hit)", output.AverageHit))
					t_insert(breakdown.AverageDamage, s_format("x %.2f ^8(chance to hit)", output.HitChance / 100))
					t_insert(breakdown.AverageDamage, s_format("= %.1f", output.AverageDamage))
				end
			end
		*/
	}
	/*
		TODO isAttack
		if isAttack then
			-- Combine crit stats, average damage and DPS
			combineStat("PreEffectiveCritChance", "AVERAGE")
			combineStat("CritChance", "AVERAGE")
			combineStat("CritMultiplier", "AVERAGE")
			combineStat("AverageDamage", "DPS")
			combineStat("TotalDPS", "DPS")
			combineStat("LifeLeechDuration", "DPS")
			combineStat("LifeLeechInstances", "DPS")
			combineStat("LifeLeechInstant", "DPS")
			combineStat("LifeLeechInstantRate", "DPS")
			combineStat("EnergyShieldLeechDuration", "DPS")
			combineStat("EnergyShieldLeechInstances", "DPS")
			combineStat("EnergyShieldLeechInstant", "DPS")
			combineStat("EnergyShieldLeechInstantRate", "DPS")
			combineStat("ManaLeechDuration", "DPS")
			combineStat("ManaLeechInstances", "DPS")
			combineStat("ManaLeechInstant", "DPS")
			combineStat("ManaLeechInstantRate", "DPS")
			combineStat("LifeOnHit", "DPS")
			combineStat("LifeOnHitRate", "DPS")
			combineStat("EnergyShieldOnHit", "DPS")
			combineStat("EnergyShieldOnHitRate", "DPS")
			combineStat("ManaOnHit", "DPS")
			combineStat("ManaOnHitRate", "DPS")
			if skillFlags.bothWeaponAttack then
				if breakdown then
					breakdown.AverageDamage = { }
					t_insert(breakdown.AverageDamage, "Both weapons:")
					if skillData.doubleHitsWhenDualWielding then
						t_insert(breakdown.AverageDamage, s_format("%.1f + %.1f ^8(skill hits with both weapons at once)", output.MainHand.AverageDamage, output.OffHand.AverageDamage))
					else
						t_insert(breakdown.AverageDamage, s_format("(%.1f + %.1f) / 2 ^8(skill alternates weapons)", output.MainHand.AverageDamage, output.OffHand.AverageDamage))
					end
					t_insert(breakdown.AverageDamage, s_format("= %.1f", output.AverageDamage))
				end
			end
		end
		if env.mode == "CALCS" then
			if skillData.showAverage then
				output.DisplayDamage = formatNumSep(s_format("%.1f", output.AverageDamage)) .. " average damage"
			else
				output.DisplayDamage = formatNumSep(s_format("%.1f", output.TotalDPS)) .. " DPS"
			end
		end
		if breakdown then
			if isAttack then
				breakdown.TotalDPS = {
					s_format("%.1f ^8(average damage)", output.AverageDamage),
					output.HitSpeed and s_format("x %.2f ^8(hit rate)", output.HitSpeed) or s_format("x %.2f ^8(attack rate)", output.Speed),
				}
			elseif isTriggered then
				breakdown.TotalDPS = {
					s_format("%.1f ^8(average damage)", output.AverageDamage),
					output.HitSpeed and s_format("x %.2f ^8(hit rate)", output.HitSpeed) or s_format("x %.2f ^8(trigger rate)", output.Speed),
				}
			else
				breakdown.TotalDPS = {
					s_format("%.1f ^8(average hit)", output.AverageDamage),
					output.HitSpeed and s_format("x %.2f ^8(hit rate)", output.HitSpeed) or s_format("x %.2f ^8(cast rate)", output.Speed),
				}
			end
			if skillData.dpsMultiplier then
				t_insert(breakdown.TotalDPS, s_format("x %g ^8(DPS multiplier for this skill)", skillData.dpsMultiplier))
			end
			if quantityMultiplier > 1 then
				t_insert(breakdown.TotalDPS, s_format("x %g ^8(quantity multiplier for this skill)", quantityMultiplier))
			end
			t_insert(breakdown.TotalDPS, s_format("= %.1f", output.TotalDPS))
		end
	*/
	/*
		TODO -- Calculate leech rates
		output.LifeLeechInstanceRate = output.Life * data.misc.LeechRateBase * calcLib.mod(skillModList, skillCfg, "LifeLeechRate")
		output.LifeLeechRate = output.LifeLeechInstantRate + m_min(output.LifeLeechInstances * output.LifeLeechInstanceRate, output.MaxLifeLeechRate) * output.LifeRecoveryRateMod
		output.LifeLeechPerHit = output.LifeLeechInstant + m_min(output.LifeLeechInstanceRate, output.MaxLifeLeechRate) * output.LifeLeechDuration * output.LifeRecoveryRateMod
		output.EnergyShieldLeechInstanceRate = output.EnergyShield * data.misc.LeechRateBase * calcLib.mod(skillModList, skillCfg, "EnergyShieldLeechRate")
		output.EnergyShieldLeechRate = output.EnergyShieldLeechInstantRate + m_min(output.EnergyShieldLeechInstances * output.EnergyShieldLeechInstanceRate, output.MaxEnergyShieldLeechRate) * output.EnergyShieldRecoveryRateMod
		output.EnergyShieldLeechPerHit = output.EnergyShieldLeechInstant + m_min(output.EnergyShieldLeechInstanceRate, output.MaxEnergyShieldLeechRate) * output.EnergyShieldLeechDuration * output.EnergyShieldRecoveryRateMod
		output.ManaLeechInstanceRate = output.Mana * data.misc.LeechRateBase * calcLib.mod(skillModList, skillCfg, "ManaLeechRate")
		output.ManaLeechRate = output.ManaLeechInstantRate + m_min(output.ManaLeechInstances * output.ManaLeechInstanceRate, output.MaxManaLeechRate) * output.ManaRecoveryRateMod
		output.ManaLeechPerHit = output.ManaLeechInstant + m_min(output.ManaLeechInstanceRate, output.MaxManaLeechRate) * output.ManaLeechDuration * output.ManaRecoveryRateMod
		-- On full life, Immortal Ambition treats life leech as energy shield leech
		if skillModList:Flag(nil, "ImmortalAmbition") then
			output.EnergyShieldLeechRate = output.EnergyShieldLeechRate + output.LifeLeechRate
			output.EnergyShieldLeechPerHit = output.EnergyShieldLeechPerHit  + output.LifeLeechPerHit
			-- Clears output.LifeLeechRate to disable leechLife flag
			output.LifeLeechRate = 0
		end
		skillFlags.leechLife = output.LifeLeechRate > 0
		skillFlags.leechES = output.EnergyShieldLeechRate > 0
		skillFlags.leechMana = output.ManaLeechRate > 0
		if skillData.showAverage then
			output.LifeLeechGainPerHit = output.LifeLeechPerHit + output.LifeOnHit
			output.EnergyShieldLeechGainPerHit = output.EnergyShieldLeechPerHit + output.EnergyShieldOnHit
			output.ManaLeechGainPerHit = output.ManaLeechPerHit + output.ManaOnHit
		else
			output.LifeLeechGainRate = output.LifeLeechRate + output.LifeOnHitRate
			output.EnergyShieldLeechGainRate = output.EnergyShieldLeechRate + output.EnergyShieldOnHitRate
			output.ManaLeechGainRate = output.ManaLeechRate + output.ManaOnHitRate
		end
		if breakdown then
			if skillFlags.leechLife then
				breakdown.LifeLeech = breakdown.leech(output.LifeLeechInstant, output.LifeLeechInstantRate, output.LifeLeechInstances, output.Life, "LifeLeechRate", output.MaxLifeLeechRate, output.LifeLeechDuration)
			end
			if skillFlags.leechES then
				breakdown.EnergyShieldLeech = breakdown.leech(output.EnergyShieldLeechInstant, output.EnergyShieldLeechInstantRate, output.EnergyShieldLeechInstances, output.EnergyShield, "EnergyShieldLeechRate", output.MaxEnergyShieldLeechRate, output.EnergyShieldLeechDuration)
			end
			if skillFlags.leechMana then
				breakdown.ManaLeech = breakdown.leech(output.ManaLeechInstant, output.ManaLeechInstantRate, output.ManaLeechInstances, output.Mana, "ManaLeechRate", output.MaxManaLeechRate, output.ManaLeechDuration)
			end
		end
	*/
	/*
		TODO Calculate Ailments
		local ailmentData = data.nonDamagingAilment
		for _, ailment in ipairs(ailmentTypeList) do
			skillFlags[string.lower(ailment)] = false
		end
		skillFlags.igniteCanStack = skillModList:Flag(skillCfg, "IgniteCanStack")
		skillFlags.igniteToChaos = skillModList:Flag(skillCfg, "IgniteToChaos")
		skillFlags.impale = false
		for _, pass in ipairs(passList) do
			globalOutput, globalBreakdown = output, breakdown
			local source, output, cfg, breakdown = pass.source, pass.output, pass.cfg, pass.breakdown

			-- Calculate chance to inflict secondary dots/status effects
			cfg.skillCond["CriticalStrike"] = true
			if not skillFlags.attack or skillModList:Flag(cfg, "CannotBleed") then
				output.BleedChanceOnCrit = 0
			else
				output.BleedChanceOnCrit = m_min(100, skillModList:Sum("BASE", cfg, "BleedChance") + enemyDB:Sum("BASE", nil, "SelfBleedChance"))
			end
			if not skillFlags.hit or skillModList:Flag(cfg, "CannotPoison") then
				output.PoisonChanceOnCrit = 0
			else
				output.PoisonChanceOnCrit = m_min(100, skillModList:Sum("BASE", cfg, "PoisonChance") + enemyDB:Sum("BASE", nil, "SelfPoisonChance"))
			end
			if not skillFlags.hit or skillModList:Flag(cfg, "CannotKnockback") then
				output.KnockbackChanceOnCrit = 0
			else
				output.KnockbackChanceOnCrit = skillModList:Sum("BASE", cfg, "EnemyKnockbackChance")
			end
			cfg.skillCond["CriticalStrike"] = false
			if not skillFlags.attack or skillModList:Flag(cfg, "CannotBleed") then
				output.BleedChanceOnHit = 0
			else
				output.BleedChanceOnHit = m_min(100, skillModList:Sum("BASE", cfg, "BleedChance") + enemyDB:Sum("BASE", nil, "SelfBleedChance"))
			end
			if not skillFlags.hit or skillModList:Flag(cfg, "CannotPoison") then
				output.PoisonChanceOnHit = 0
				output.ChaosPoisonChance = 0
			else
				output.PoisonChanceOnHit = m_min(100, skillModList:Sum("BASE", cfg, "PoisonChance") + enemyDB:Sum("BASE", nil, "SelfPoisonChance"))
				output.ChaosPoisonChance = m_min(100, skillModList:Sum("BASE", cfg, "ChaosPoisonChance"))
			end
			for _, ailment in ipairs(elementalAilmentTypeList) do
				local chance = skillModList:Sum("BASE", cfg, "Enemy"..ailment.."Chance") + enemyDB:Sum("BASE", nil, "Self"..ailment.."Chance")
				if ailment == "Chill" then
					chance = 100
				end
				if skillFlags.hit and not skillModList:Flag(cfg, "Cannot"..ailment) then
					output[ailment.."ChanceOnHit"] = m_min(100, chance)
					if skillModList:Flag(cfg, "CritsDontAlways"..ailment) -- e.g. Painseeker
					or (ailmentData[ailment] and ailmentData[ailment].alt and not skillModList:Flag(cfg, "CritAlwaysAltAilments")) then -- e.g. Secrets of Suffering
						output[ailment.."ChanceOnCrit"] = output[ailment.."ChanceOnHit"]
					else
						output[ailment.."ChanceOnCrit"] = 100
					end
				else
					output[ailment.."ChanceOnHit"] = 0
					output[ailment.."ChanceOnCrit"] = 0
				end
				if (output[ailment.."ChanceOnHit"] + (skillModList:Flag(cfg, "NeverCrit") and 0 or output[ailment.."ChanceOnCrit"])) > 0 then
					skillFlags["inflict"..ailment] = true
				end
			end
			if not skillFlags.hit or skillModList:Flag(cfg, "CannotKnockback") then
				output.KnockbackChanceOnHit = 0
			else
				output.KnockbackChanceOnHit = skillModList:Sum("BASE", cfg, "EnemyKnockbackChance")
			end
			output.ImpaleChance = m_min(100, skillModList:Sum("BASE", cfg, "ImpaleChance"))
			if skillModList:Sum("BASE", cfg, "FireExposureChance") > 0 then
				skillFlags.applyFireExposure = true
			end
			if skillModList:Sum("BASE", cfg, "ColdExposureChance") > 0 then
				skillFlags.applyColdExposure = true
			end
			if skillModList:Sum("BASE", cfg, "LightningExposureChance") > 0 then
				skillFlags.applyLightningExposure = true
			end
			if env.mode_effective then
				for _, ailment in ipairs(ailmentTypeList) do
					local mult = 1 - enemyDB:Sum("BASE", nil, "Avoid"..ailment) / 100
					output[ailment.."ChanceOnHit"] = output[ailment.."ChanceOnHit"] * mult
					output[ailment.."ChanceOnCrit"] = output[ailment.."ChanceOnCrit"] * mult
					if ailment == "Poison" then
						output.ChaosPoisonChance = output.ChaosPoisonChance * mult
					end
				end
			end

			local igniteMode = env.configInput.igniteMode or "AVERAGE"
			if igniteMode == "CRIT" then
				for _, ailment in ipairs(ailmentTypeList) do
					output[ailment.."ChanceOnHit"] = 0
				end
			end

			---Calculates normal and crit damage to be used in non-damaging ailment calculations
			---@param ailment string
			---@return number, number @average hit damage, average crit damage
			local function calcAverageSourceDamage(ailment)
				local sourceHitDmg, sourceCritDmg = 0, 0
				for _, type in ipairs(dmgTypeList) do
					if canDeal[type] and (function()
						if type == ailmentData[ailment].associatedType then
							return not skillModList:Flag(cfg, type.."Cannot"..ailment)
						else
							return skillModList:Flag(cfg, type.."Can"..ailment)
						end
					end)() then
						sourceHitDmg = sourceHitDmg + output[type.."HitAverage"]
						sourceCritDmg = sourceCritDmg + output[type.."CritAverage"]
					end
				end
				return sourceHitDmg, sourceCritDmg
			end

			local function calcAilmentDamage(type, sourceHitDmg, sourceCritDmg)
				-- Calculate the inflict chance and base damage of a secondary effect (bleed/poison/ignite/shock/freeze)
				local chanceOnHit, chanceOnCrit = output[type.."ChanceOnHit"], output[type.."ChanceOnCrit"]
				local chanceFromHit = chanceOnHit * (1 - output.CritChance / 100)
				local chanceFromCrit = chanceOnCrit * output.CritChance / 100
				local chance = chanceFromHit + chanceFromCrit
				output[type.."Chance"] = chance
				local baseFromHit = sourceHitDmg * chanceFromHit / (chanceFromHit + chanceFromCrit)
				local baseFromCrit = sourceCritDmg * chanceFromCrit / (chanceFromHit + chanceFromCrit)
				local baseVal = baseFromHit + baseFromCrit
				local sourceMult = skillModList:More(nil, type.."AsThoughDealing")
				if breakdown and chance ~= 0 then
					local breakdownChance = breakdown[type.."Chance"] or { }
					breakdown[type.."Chance"] = breakdownChance
					if breakdownChance[1] then
						t_insert(breakdownChance, "")
					end
					if isAttack then
						t_insert(breakdownChance, pass.label..":")
					end
					t_insert(breakdownChance, s_format("Chance on Non-crit: %d%%", chanceOnHit))
					t_insert(breakdownChance, s_format("Chance on Crit: %d%%", chanceOnCrit))
					if chanceOnHit ~= chanceOnCrit then
						t_insert(breakdownChance, "Combined chance:")
						t_insert(breakdownChance, s_format("%d x (1 - %.4f) ^8(chance from non-crits)", chanceOnHit, output.CritChance/100))
						t_insert(breakdownChance, s_format("+ %d x %.4f ^8(chance from crits)", chanceOnCrit, output.CritChance/100))
						t_insert(breakdownChance, s_format("= %.2f", chance))
					end
				end
				if breakdown and baseVal > 0 then
					local breakdownDPS = breakdown[type.."DPS"] or { }
					breakdown[type.."DPS"] = breakdownDPS
					if breakdownDPS[1] then
						t_insert(breakdownDPS, "")
					end
					if isAttack then
						t_insert(breakdownDPS, pass.label..":")
					end
					if sourceHitDmg == sourceCritDmg then
						t_insert(breakdownDPS, "Total damage:")
						t_insert(breakdownDPS, s_format("%.1f ^8(source damage)",sourceHitDmg))
						if sourceMult > 1 then
							t_insert(breakdownDPS, s_format("x %.2f ^8(inflicting as though dealing more damage)", sourceMult))
							t_insert(breakdownDPS, s_format("= %.1f", baseVal * sourceMult))
						end
					else
						if baseFromHit > 0 then
							t_insert(breakdownDPS, "Damage from Non-crits:")
							t_insert(breakdownDPS, s_format("%.1f ^8(source damage from non-crits)", sourceHitDmg))
							t_insert(breakdownDPS, s_format("x %.3f ^8(portion of instances created by non-crits)", chanceFromHit / (chanceFromHit + chanceFromCrit)))
							if sourceMult == 1 or baseFromCrit ~= 0 then
								t_insert(breakdownDPS, s_format("= %.1f", baseFromHit))
							end
						end
						if baseFromCrit > 0 then
							t_insert(breakdownDPS, "Damage from Crits:")
							t_insert(breakdownDPS, s_format("%.1f ^8(source damage from crits)", sourceCritDmg))
							t_insert(breakdownDPS, s_format("x %.3f ^8(portion of instances created by crits)", chanceFromCrit / (chanceFromHit + chanceFromCrit)))
							if sourceMult == 1 or baseFromHit ~= 0 then
								t_insert(breakdownDPS, s_format("= %.1f", baseFromCrit))
							end
						end
						if baseFromHit > 0 and baseFromCrit > 0 then
							t_insert(breakdownDPS, "Total damage:")
							t_insert(breakdownDPS, s_format("%.1f + %.1f", baseFromHit, baseFromCrit))
							if sourceMult == 1 then
								t_insert(breakdownDPS, s_format("= %.1f", baseVal))
							end
						end
						if sourceMult > 1 then
							t_insert(breakdownDPS, s_format("x %.2f ^8(inflicting as though dealing more damage)", sourceMult))
							t_insert(breakdownDPS, s_format("= %.1f", baseVal * sourceMult))
						end
					end
				end
				return baseVal
			end

			-- Calculate bleeding chance and damage
			if canDeal.Physical and (output.BleedChanceOnHit + output.BleedChanceOnCrit) > 0 then
				activeSkill[pass.label ~= "Off Hand" and "bleedCfg" or "OHbleedCfg"] = {
					skillName = skillCfg.skillName,
					skillPart = skillCfg.skillPart,
					skillTypes = skillCfg.skillTypes,
					slotName = skillCfg.slotName,
					flags = bor(ModFlag.Dot, ModFlag.Ailment, band(cfg.flags, ModFlag.WeaponMask), band(cfg.flags, ModFlag.Melee) ~= 0 and ModFlag.MeleeHit or 0),
					keywordFlags = bor(band(cfg.keywordFlags, bnot(KeywordFlag.Hit)), KeywordFlag.Bleed, KeywordFlag.Ailment, KeywordFlag.PhysicalDot),
					skillCond = setmetatable({["CriticalStrike"] = true }, { __index = function(table, key) return skillCfg.skillCond[key] or cfg.skillCond[key] end } ),
					skillDist = skillCfg.skillDist,
				}
				local dotCfg = pass.label ~= "Off Hand" and activeSkill.bleedCfg or activeSkill.OHbleedCfg
				local sourceHitDmg, sourceCritDmg
				if breakdown then
					breakdown.BleedPhysical = { damageTypes = { } }
				end

				-- For bleeds we will be using a weighted average calculation
				local configStacks = enemyDB:Sum("BASE", nil, "Multiplier:BleedStacks")
				local maxStacks = skillModList:Override(cfg, "BleedStacksMax") or skillModList:Sum("BASE", cfg, "BleedStacksMax")
				globalOutput.BleedStacksMax = maxStacks
				local durationBase = skillData.bleedDurationIsSkillDuration and skillData.duration or data.misc.BleedDurationBase
				local durationMod = calcLib.mod(skillModList, dotCfg, "EnemyBleedDuration", "SkillAndDamagingAilmentDuration", skillData.bleedIsSkillEffect and "Duration" or nil) * calcLib.mod(enemyDB, nil, "SelfBleedDuration") / calcLib.mod(enemyDB, dotCfg, "BleedExpireRate")
				local rateMod = calcLib.mod(skillModList, cfg, "BleedFaster") + enemyDB:Sum("INC", nil, "SelfBleedFaster")  / 100
				globalOutput.BleedDuration = durationBase * durationMod / rateMod * debuffDurationMult
				local bleedStacks = (output.HitChance / 100) * (globalOutput.BleedDuration / output.Time) / maxStacks
				bleedStacks = configStacks > 0 and m_min(bleedStacks, configStacks / maxStacks) or bleedStacks
				globalOutput.BleedStackPotential = bleedStacks
				if globalBreakdown then
					globalBreakdown.BleedStackPotential = {
						s_format(colorCodes.CUSTOM.."NOTE: Calculation uses new Weighted Avg Ailment formula"),
						s_format(""),
						s_format("%.2f ^8(chance to hit)", output.HitChance / 100),
						s_format("* (%.2f / %.2f) ^8(BleedDuration / Attack Time)", globalOutput.BleedDuration, output.Time),
						s_format("/ %d ^8(max number of stacks)", maxStacks),
						s_format("= %.2f", globalOutput.BleedStackPotential),
					}
				end

				for sub_pass = 1, 2 do
					if skillModList:Flag(dotCfg, "AilmentsAreNeverFromCrit") or sub_pass == 1 then
						dotCfg.skillCond["CriticalStrike"] = false
					else
						dotCfg.skillCond["CriticalStrike"] = true
					end
					local min, max = calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.BleedPhysical, "Physical", 0)
					output.BleedPhysicalMin = min
					output.BleedPhysicalMax = max
					if sub_pass == 2 then
						output.CritBleedDotMulti = 1 + skillModList:Sum("BASE", dotCfg, "DotMultiplier", "PhysicalDotMultiplier") / 100
						sourceCritDmg = (min + (max - min) / m_pow(2, 1 / (bleedStacks + 1))) * output.CritBleedDotMulti
					else
						output.BleedDotMulti = 1 + skillModList:Sum("BASE", dotCfg, "DotMultiplier", "PhysicalDotMultiplier") / 100
						sourceHitDmg = (min + (max - min) / m_pow(2, 1 / (bleedStacks + 1))) * output.BleedDotMulti
					end
				end
				if globalBreakdown then
					if sourceHitDmg == sourceCritDmg then
						globalBreakdown.BleedDPS = {
							s_format(colorCodes.CUSTOM.."NOTE: Calculation uses new Weighted Avg Ailment formula"),
							s_format(""),
							s_format("Dmg Derivation:"),
							s_format("(%.2f + (%.2f - %.2f) ^8(min source physical + (max source physical - min source physical)", output.BleedPhysicalMin, output.BleedPhysicalMax, output.BleedPhysicalMin),
							s_format("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", bleedStacks),
							s_format("* %.2f ^8(Bleed DoT Multi)", output.BleedDotMulti),
							s_format("= %.2f", sourceHitDmg),
						}
					else
						globalBreakdown.BleedDPS = {
							s_format(colorCodes.CUSTOM.."NOTE: Calculation uses new Weighted Avg Ailment formula"),
							s_format(""),
							s_format("Non-Crit Dmg Derivation:"),
							s_format("(%.2f + (%.2f - %.2f) ^8(min source physical + (max source physical - min source physical)", output.BleedPhysicalMin, output.BleedPhysicalMax, output.BleedPhysicalMin),
							s_format("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", bleedStacks),
							s_format("* %.2f ^8(Bleed DoT Multi for Non-Crit)", output.BleedDotMulti),
							s_format("= %.2f", sourceHitDmg),
							s_format(""),
							s_format("Crit Dmg Derivation:"),
							s_format("(%.2f + (%.2f - %.2f) ^8(min source physical + (max source physical - min source physical)", output.BleedPhysicalMin, output.BleedPhysicalMax, output.BleedPhysicalMin),
							s_format("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", bleedStacks),
							s_format("* %.2f ^8(Bleed DoT Multi for Crit)", output.CritBleedDotMulti),
							s_format("= %.2f", sourceCritDmg),
						}
					end
				end
				local basePercent = skillData.bleedBasePercent or data.misc.BleedPercentBase
				local baseVal = calcAilmentDamage("Bleed", sourceHitDmg, sourceCritDmg) * basePercent / 100 * output.RuthlessBlowBleedEffect * output.FistOfWarAilmentEffect * globalOutput.AilmentWarcryEffect
				if baseVal > 0 then
					skillFlags.bleed = true
					skillFlags.duration = true
					local effMult = 1
					if env.mode_effective then
						local resist = m_min(m_max(0, enemyDB:Sum("BASE", nil, "PhysicalDamageReduction")), data.misc.DamageReductionCap)
						local takenInc = enemyDB:Sum("INC", dotCfg, "DamageTaken", "DamageTakenOverTime", "PhysicalDamageTaken", "PhysicalDamageTakenOverTime")
						local takenMore = enemyDB:More(dotCfg, "DamageTaken", "DamageTakenOverTime", "PhysicalDamageTaken", "PhysicalDamageTakenOverTime")
						effMult = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
						globalOutput["BleedEffMult"] = effMult
						if breakdown and effMult ~= 1 then
							globalBreakdown.BleedEffMult = breakdown.effMult("Physical", resist, 0, takenInc, effMult, takenMore)
						end
					end
					local effectMod = calcLib.mod(skillModList, dotCfg, "AilmentEffect")
					output.BaseBleedDPS = baseVal * effectMod * rateMod * effMult
					bleedStacks = m_min(maxStacks, (output.HitChance / 100) * globalOutput.BleedDuration / output.Time)
					local chanceToHitInOneSecInterval = 1 - m_pow(1 - (output.HitChance / 100), output.Speed)
					output.BleedDPS = (baseVal * effectMod * rateMod * effMult) * bleedStacks * chanceToHitInOneSecInterval
					-- reset bleed stacks to actual number doing damage after weighted avg DPS calculation is done
					globalOutput.BleedStacks = bleedStacks
					globalOutput.BleedDamage = output.BaseBleedDPS * globalOutput.BleedDuration
					if breakdown then
						if output.CritBleedDotMulti and (output.CritBleedDotMulti ~= output.BleedDotMulti) then
							local chanceFromHit = output.BleedChanceOnHit / 100 * (1 - globalOutput.CritChance / 100)
							local chanceFromCrit = output.BleedChanceOnCrit / 100 * output.CritChance / 100
							local totalFromHit = chanceFromHit / (chanceFromHit + chanceFromCrit)
							local totalFromCrit = chanceFromCrit / (chanceFromHit + chanceFromCrit)
							breakdown.BleedDotMulti = breakdown.critDot(output.BleedDotMulti, output.CritBleedDotMulti, totalFromHit, totalFromCrit)
							output.BleedDotMulti = (output.BleedDotMulti * totalFromHit) + (output.CritBleedDotMulti * totalFromCrit)
						end
						t_insert(breakdown.BleedDPS, s_format("x %.2f ^8(bleed deals %d%% per second)", basePercent/100, basePercent))
						if effectMod ~= 1 then
							t_insert(breakdown.BleedDPS, s_format("x %.2f ^8(ailment effect modifier)", effectMod))
						end
						if output.RuthlessBlowBleedEffect ~= 1 then
							t_insert(breakdown.BleedDPS, s_format("x %.2f ^8(ruthless blow effect modifier)", output.RuthlessBlowBleedEffect))
						end
						if output.FistOfWarAilmentEffect ~= 1 then
							t_insert(breakdown.BleedDPS, s_format("x %.2f ^8(fist of war effect modifier)", output.FistOfWarAilmentEffect))
						end
						if globalOutput.AilmentWarcryEffect > 1 then
							t_insert(breakdown.BleedDPS, s_format("x %.2f ^8(combined ailment warcry effect modifier)", globalOutput.AilmentWarcryEffect))
						end
						t_insert(breakdown.BleedDPS, s_format("= %.1f", baseVal))
						breakdown.multiChain(breakdown.BleedDPS, {
							label = "Bleed DPS:",
							base = s_format("%.1f ^8(total damage per second)", baseVal),
							{ "%.2f ^8(ailment effect modifier)", effectMod },
							{ "%.2f ^8(damage rate modifier)", rateMod },
							{ "%.3f ^8(effective DPS modifier)", effMult },
							{ "%d ^8(bleed stacks)", globalOutput.BleedStacks },
							{ "%.3f ^8(bleed chance based on chance to hit each second)", chanceToHitInOneSecInterval },
							total = s_format("= %.1f ^8per second", output.BleedDPS),
						})
						if globalOutput.BleedDuration ~= durationBase then
							globalBreakdown.BleedDuration = {
								s_format("%.2fs ^8(base duration)", durationBase)
							}
							if durationMod ~= 1 then
								t_insert(globalBreakdown.BleedDuration, s_format("x %.2f ^8(duration modifier)", durationMod))
							end
							if rateMod ~= 1 then
								t_insert(globalBreakdown.BleedDuration, s_format("/ %.2f ^8(damage rate modifier)", rateMod))
							end
							if debuffDurationMult ~= 1 then
								t_insert(globalBreakdown.BleedDuration, s_format("/ %.2f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
							end
							t_insert(globalBreakdown.BleedDuration, s_format("= %.2fs", globalOutput.BleedDuration))
						end
					end
				end
			end

			-- Calculate poison chance and damage
			if canDeal.Chaos and (output.PoisonChanceOnHit + output.PoisonChanceOnCrit + output.ChaosPoisonChance) > 0 then
				activeSkill[pass.label ~= "Off Hand" and "poisonCfg" or "OHpoisonCfg"] = {
					skillName = skillCfg.skillName,
					skillPart = skillCfg.skillPart,
					skillTypes = skillCfg.skillTypes,
					slotName = skillCfg.slotName,
					flags = bor(ModFlag.Dot, ModFlag.Ailment, band(cfg.flags, ModFlag.WeaponMask), band(cfg.flags, ModFlag.Melee) ~= 0 and ModFlag.MeleeHit or 0),
					keywordFlags = bor(band(cfg.keywordFlags, bnot(KeywordFlag.Hit)), KeywordFlag.Poison, KeywordFlag.Ailment, KeywordFlag.ChaosDot),
					skillCond = setmetatable({["CriticalStrike"] = true }, { __index = function(table, key) return skillCfg.skillCond[key] or cfg.skillCond[key] end } ),
					skillDist = skillCfg.skillDist,
				}
				local dotCfg = pass.label ~= "Off Hand" and activeSkill.poisonCfg or activeSkill.OHpoisonCfg
				local sourceHitDmg, sourceCritDmg
				if breakdown then
					breakdown.PoisonPhysical = { damageTypes = { } }
					breakdown.PoisonLightning = { damageTypes = { } }
					breakdown.PoisonCold = { damageTypes = { } }
					breakdown.PoisonFire = { damageTypes = { } }
					breakdown.PoisonChaos = { damageTypes = { } }
				end
				for sub_pass = 1, 2 do
					if skillModList:Flag(dotCfg, "AilmentsAreNeverFromCrit") or sub_pass == 1 then
						dotCfg.skillCond["CriticalStrike"] = false
					else
						dotCfg.skillCond["CriticalStrike"] = true
					end
					local totalMin, totalMax = 0, 0
					do
						local min, max = calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.PoisonChaos, "Chaos", 0)
						output.PoisonChaosMin = min
						output.PoisonChaosMax = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					end
					local nonChaosMult = 1
					if output.ChaosPoisonChance > 0 and output.PoisonChaosMax > 0 then
						-- Additional chance for chaos
						local chance = (sub_pass == 2) and "PoisonChanceOnCrit" or "PoisonChanceOnHit"
						local chaosChance = m_min(100, output[chance] + output.ChaosPoisonChance)
						nonChaosMult = output[chance] / chaosChance
						output[chance] = chaosChance
					end
					if canDeal.Lightning and skillModList:Flag(cfg, "LightningCanPoison") then
						local min, max = calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.PoisonLightning, "Lightning", dmgTypeFlags.Chaos)
						output.PoisonLightningMin = min
						output.PoisonLightningMax = max
						totalMin = totalMin + min * nonChaosMult
						totalMax = totalMax + max * nonChaosMult
					end
					if canDeal.Cold and skillModList:Flag(cfg, "ColdCanPoison") then
						local min, max = calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.PoisonCold, "Cold", dmgTypeFlags.Chaos)
						output.PoisonColdMin = min
						output.PoisonColdMax = max
						totalMin = totalMin + min * nonChaosMult
						totalMax = totalMax + max * nonChaosMult
					end
					if canDeal.Fire and skillModList:Flag(cfg, "FireCanPoison") then
						local min, max = calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.PoisonFire, "Fire", dmgTypeFlags.Chaos)
						output.PoisonFireMin = min
						output.PoisonFireMax = max
						totalMin = totalMin + min * nonChaosMult
						totalMax = totalMax + max * nonChaosMult
					end
					if canDeal.Physical then
						local min, max = calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.PoisonPhysical, "Physical", dmgTypeFlags.Chaos)
						output.PoisonPhysicalMin = min
						output.PoisonPhysicalMax = max
						totalMin = totalMin + min * nonChaosMult
						totalMax = totalMax + max * nonChaosMult
					end
					if sub_pass == 2 then
						output.CritPoisonDotMulti = 1 + skillModList:Sum("BASE", dotCfg, "DotMultiplier", "ChaosDotMultiplier") / 100
						sourceCritDmg = (totalMin + totalMax) / 2 * output.CritPoisonDotMulti
					else
						output.PoisonDotMulti = 1 + skillModList:Sum("BASE", dotCfg, "DotMultiplier", "ChaosDotMultiplier") / 100
						sourceHitDmg = (totalMin + totalMax) / 2 * output.PoisonDotMulti
					end
				end
				if globalBreakdown then
					globalBreakdown.PoisonDPS = {
						s_format("Ailment mode: %s ^8(can be changed in the Configuration tab)", igniteMode == "CRIT" and "Crits Only" or "Average Damage")
					}
				end
				local baseVal = calcAilmentDamage("Poison", sourceHitDmg, sourceCritDmg) * data.misc.PoisonPercentBase * output.FistOfWarAilmentEffect * globalOutput.AilmentWarcryEffect
				if baseVal > 0 then
					skillFlags.poison = true
					skillFlags.duration = true
					local effMult = 1
					if env.mode_effective then
						local resist = m_min(enemyDB:Sum("BASE", nil, "ChaosResist") * calcLib.mod(enemyDB, nil, "ChaosResist"), data.misc.EnemyMaxResist)
						local takenInc = enemyDB:Sum("INC", dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
						local takenMore = enemyDB:More(dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
						effMult = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
						globalOutput["PoisonEffMult"] = effMult
						if breakdown and effMult ~= 1 then
							globalBreakdown.PoisonEffMult = breakdown.effMult("Chaos", resist, 0, takenInc, effMult, takenMore)
						end
					end
					local effectMod = calcLib.mod(skillModList, dotCfg, "AilmentEffect")
					local rateMod = calcLib.mod(skillModList, cfg, "PoisonFaster") + enemyDB:Sum("INC", nil, "SelfPoisonFaster")  / 100
					output.PoisonDPS = baseVal * effectMod * rateMod * effMult
					local durationBase
					if skillData.poisonDurationIsSkillDuration then
						durationBase = skillData.duration
					else
						durationBase = data.misc.PoisonDurationBase
					end
					local durationMod = calcLib.mod(skillModList, dotCfg, "EnemyPoisonDuration", "SkillAndDamagingAilmentDuration", skillData.poisonIsSkillEffect and "Duration" or nil) * calcLib.mod(enemyDB, nil, "SelfPoisonDuration")
					globalOutput.PoisonDuration = durationBase * durationMod / rateMod * debuffDurationMult
					output.PoisonDamage = output.PoisonDPS * globalOutput.PoisonDuration
					if skillData.showAverage then
						output.TotalPoisonAverageDamage = output.HitChance / 100 * output.PoisonChance / 100 * output.PoisonDamage
						output.TotalPoisonDPS = output.PoisonDPS
					else
						output.TotalPoisonStacks = output.HitChance / 100 * output.PoisonChance / 100 * globalOutput.PoisonDuration * (globalOutput.HitSpeed or globalOutput.Speed) * (skillData.dpsMultiplier or 1) * (skillData.stackMultiplier or 1) * quantityMultiplier
						output.TotalPoisonDPS = output.PoisonDPS * output.TotalPoisonStacks
					end
					if breakdown then
						if output.CritPoisonDotMulti and (output.CritPoisonDotMulti ~= output.PoisonDotMulti) then
							local chanceFromHit = output.PoisonChanceOnHit / 100 * (1 - globalOutput.CritChance / 100)
							local chanceFromCrit = output.PoisonChanceOnCrit / 100 * globalOutput.CritChance / 100
							local totalFromHit = chanceFromHit / (chanceFromHit + chanceFromCrit)
							local totalFromCrit = chanceFromCrit / (chanceFromHit + chanceFromCrit)
							breakdown.PoisonDotMulti = breakdown.critDot(output.PoisonDotMulti, output.CritPoisonDotMulti, totalFromHit, totalFromCrit)
							output.PoisonDotMulti = (output.PoisonDotMulti * totalFromHit) + (output.CritPoisonDotMulti * totalFromCrit)
						end
						t_insert(breakdown.PoisonDPS, "x 0.30 ^8(poison deals 30% per second)")
						t_insert(breakdown.PoisonDPS, s_format("= %.1f", baseVal, 1))
						breakdown.multiChain(breakdown.PoisonDPS, {
							label = "Poison DPS:",
							base = s_format("%.1f ^8(total damage per second)", baseVal),
							{ "%.2f ^8(ailment effect modifier)", effectMod },
							{ "%.2f ^8(damage rate modifier)", rateMod },
							{ "%.3f ^8(effective DPS modifier)", effMult },
							total = s_format("= %.1f ^8per second", output.PoisonDPS),
						})
						if globalOutput.PoisonDuration ~= 2 then
							globalBreakdown.PoisonDuration = {
								s_format("%.2fs ^8(base duration)", durationBase)
							}
							if durationMod ~= 1 then
								t_insert(globalBreakdown.PoisonDuration, s_format("x %.2f ^8(duration modifier)", durationMod))
							end
							if rateMod ~= 1 then
								t_insert(globalBreakdown.PoisonDuration, s_format("/ %.2f ^8(damage rate modifier)", rateMod))
							end
							if debuffDurationMult ~= 1 then
								t_insert(globalBreakdown.PoisonDuration, s_format("/ %.2f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
							end
							t_insert(globalBreakdown.PoisonDuration, s_format("= %.2fs", globalOutput.PoisonDuration))
						end
						breakdown.PoisonDamage = { }
						if isAttack then
							t_insert(breakdown.PoisonDamage, pass.label..":")
						end
						t_insert(breakdown.PoisonDamage, s_format("%.1f ^8(damage per second)", output.PoisonDPS))
						t_insert(breakdown.PoisonDamage, s_format("x %.2fs ^8(poison duration)", globalOutput.PoisonDuration))
						t_insert(breakdown.PoisonDamage, s_format("= %.1f ^8damage per poison stack", output.PoisonDamage))
						if not skillData.showAverage then
							breakdown.TotalPoisonStacks = { }
							if isAttack then
								t_insert(breakdown.TotalPoisonStacks, pass.label..":")
							end
							breakdown.multiChain(breakdown.TotalPoisonStacks, {
								base = s_format("%.2fs ^8(poison duration)", globalOutput.PoisonDuration),
								{ "%.2f ^8(poison chance)", output.PoisonChance / 100 },
								{ "%.2f ^8(hit chance)", output.HitChance / 100 },
								{ "%.2f ^8(hits per second)", globalOutput.HitSpeed or globalOutput.Speed },
								{ "%g ^8(dps multiplier for this skill)", skillData.dpsMultiplier or 1 },
								{ "%g ^8(stack multiplier for this skill)", skillData.stackMultiplier or 1 },
								{ "%g ^8(quantity multiplier for this skill)", quantityMultiplier },
								total = s_format("= %.1f", output.TotalPoisonStacks),
							})
						end
					end
				end
			end

			-- Calculate ignite chance and damage
			if canDeal.Fire and (output.IgniteChanceOnHit + output.IgniteChanceOnCrit) > 0 then
				activeSkill[pass.label ~= "Off Hand" and "igniteCfg" or "OHigniteCfg"] = {
					skillName = skillCfg.skillName,
					skillPart = skillCfg.skillPart,
					skillTypes = skillCfg.skillTypes,
					slotName = skillCfg.slotName,
					flags = bor(ModFlag.Dot, ModFlag.Ailment, band(cfg.flags, ModFlag.WeaponMask), band(cfg.flags, ModFlag.Melee) ~= 0 and ModFlag.MeleeHit or 0),
					keywordFlags = bor(band(cfg.keywordFlags, bnot(KeywordFlag.Hit)), KeywordFlag.Ignite, KeywordFlag.Ailment, KeywordFlag.FireDot),
					skillCond = setmetatable({["CriticalStrike"] = true }, { __index = function(table, key) return skillCfg.skillCond[key] or cfg.skillCond[key] end } ),
					skillDist = skillCfg.skillDist,
				}
				local dotCfg = pass.label ~= "Off Hand" and activeSkill.igniteCfg or activeSkill.OHigniteCfg
				local sourceHitDmg, sourceCritDmg
				if breakdown then
					breakdown.IgnitePhysical = { damageTypes = { } }
					breakdown.IgniteLightning = { damageTypes = { } }
					breakdown.IgniteCold = { damageTypes = { } }
					breakdown.IgniteFire = { damageTypes = { } }
					breakdown.IgniteChaos = { damageTypes = { } }
				end

				-- For ignites we will be using a weighted average calculation
				local maxStacks = 1
				if skillFlags.igniteCanStack then
					maxStacks = maxStacks + skillModList:Sum("BASE", cfg, "IgniteStacks")
				end
				globalOutput.IgniteStacksMax = maxStacks

				local rateMod = (calcLib.mod(skillModList, cfg, "IgniteBurnFaster") + enemyDB:Sum("INC", nil, "SelfIgniteBurnFaster") / 100)  / calcLib.mod(skillModList, cfg, "IgniteBurnSlower")
				local durationBase = data.misc.IgniteDurationBase
				local durationMod = m_max(calcLib.mod(skillModList, dotCfg, "EnemyIgniteDuration", "SkillAndDamagingAilmentDuration") * calcLib.mod(enemyDB, nil, "SelfIgniteDuration"), 0)
				globalOutput.IgniteDuration = durationBase * durationMod / rateMod * debuffDurationMult
				globalOutput.IgniteDuration = globalOutput.IgniteDuration > data.misc.IgniteMinDuration and globalOutput.IgniteDuration or 0
				local igniteStacks = 1
				if not skillData.triggeredOnDeath then
					igniteStacks = (globalOutput.IgniteDuration / output.Time) / maxStacks
				end
				globalOutput.IgniteStackPotential = igniteStacks
				if globalBreakdown then
					globalBreakdown.IgniteStackPotential = {
						s_format(colorCodes.CUSTOM.."NOTE: Calculation uses new Weighted Avg Ailment formula"),
						s_format(""),
						s_format("(%.2f / %.2f) ^8(IgniteDuration / Cast Time)", globalOutput.IgniteDuration, output.Time),
						s_format("/ %d ^8(max number of stacks)", maxStacks),
						s_format("= %.2f", globalOutput.IgniteStackPotential),
					}
				end

				for sub_pass = 1, 2 do
					if skillModList:Flag(dotCfg, "AilmentsAreNeverFromCrit") or sub_pass == 1 then
						dotCfg.skillCond["CriticalStrike"] = false
					else
						dotCfg.skillCond["CriticalStrike"] = true
					end
					local totalMin, totalMax = 0, 0
					if canDeal.Physical and skillModList:Flag(cfg, "PhysicalCanIgnite") then
						local min, max = calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.IgnitePhysical, "Physical", dmgTypeFlags.Fire)
						output.IgnitePhysicalMin = min
						output.IgnitePhysicalMax = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					end
					if canDeal.Lightning and skillModList:Flag(cfg, "LightningCanIgnite") then
						local min, max = calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.IgniteLightning, "Lightning", dmgTypeFlags.Fire)
						output.IgniteLightningMin = min
						output.IgniteLightningMax = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					end
					if canDeal.Cold and skillModList:Flag(cfg, "ColdCanIgnite") then
						local min, max = calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.IgniteCold, "Cold", dmgTypeFlags.Fire)
						output.IgniteColdMin = min
						output.IgniteColdMax = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					end
					if canDeal.Fire and not skillModList:Flag(cfg, "FireCannotIgnite") then
						local min, max = calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.IgniteFire, "Fire", 0)
						output.IgniteFireMin = min
						output.IgniteFireMax = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					end
					if canDeal.Chaos and skillModList:Flag(cfg, "ChaosCanIgnite") then
						local min, max = calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.IgniteChaos, "Chaos", dmgTypeFlags.Fire)
						output.IgniteChaosMin = min
						output.IgniteChaosMax = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					end
					if sub_pass == 2 then
						output.CritIgniteDotMulti = 1 + skillModList:Sum("BASE", dotCfg, "DotMultiplier", "FireDotMultiplier") / 100
						sourceCritDmg = (totalMin + (totalMax - totalMin) / m_pow(2, 1 / (igniteStacks + 1))) * output.CritIgniteDotMulti
					else
						output.IgniteDotMulti = 1 + skillModList:Sum("BASE", dotCfg, "DotMultiplier", "FireDotMultiplier") / 100
						sourceHitDmg = (totalMin + (totalMax - totalMin) / m_pow(2, 1 / (igniteStacks + 1))) * output.IgniteDotMulti
					end
					output.IgniteTotalMin = totalMin
					output.IgniteTotalMax = totalMax
				end
				if globalBreakdown then
					if sourceHitDmg == sourceCritDmg then
						globalBreakdown.IgniteDPS = {
							s_format(colorCodes.CUSTOM.."NOTE: Calculation uses new Weighted Avg Ailment formula"),
							s_format(""),
							s_format("Dmg Derivation:"),
							s_format("(%.2f + (%.2f - %.2f) ^8(min combined sources + (max combined sources - min combined sources)", output.IgniteTotalMin, output.IgniteTotalMax, output.IgniteTotalMin),
							s_format("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", igniteStacks),
							s_format("* %.2f ^8(Ignite DoT Multi)", output.IgniteDotMulti),
							s_format("= %.2f", sourceHitDmg),
						}
					else
						globalBreakdown.IgniteDPS = {
							s_format(colorCodes.CUSTOM.."NOTE: Calculation uses new Weighted Avg Ailment formula"),
							s_format(""),
							s_format("Non-Crit Dmg Derivation:"),
							s_format("(%.2f + (%.2f - %.2f) ^8(min combined sources + (max combined sources - min combined sources)", output.IgniteTotalMin, output.IgniteTotalMax, output.IgniteTotalMin),
							s_format("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", igniteStacks),
							s_format("* %.2f ^8(Ignite DoT Multi for Non-Crit)", output.IgniteDotMulti),
							s_format("= %.2f", sourceHitDmg),
							s_format(""),
							s_format("Crit Dmg Derivation:"),
							s_format("(%.2f + (%.2f - %.2f) ^8(min combined sources + (max combined sources - min combined sources)", output.IgniteTotalMin, output.IgniteTotalMax, output.IgniteTotalMin),
							s_format("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", igniteStacks),
							s_format("* %.2f ^8(Ignite DoT Multi for Crit)", output.CritIgniteDotMulti),
							s_format("= %.2f", sourceCritDmg),
						}
					end
				end
				local baseVal = calcAilmentDamage("Ignite", sourceHitDmg, sourceCritDmg) * data.misc.IgnitePercentBase * output.FistOfWarAilmentEffect * globalOutput.AilmentWarcryEffect
				if baseVal > 0 then
					skillFlags.ignite = true
					local effMult = 1
					if env.mode_effective then
						if skillModList:Flag(cfg, "IgniteToChaos") then
							local resist = m_min(enemyDB:Sum("BASE", nil, "ChaosResist") * calcLib.mod(enemyDB, nil, "ChaosResist"), data.misc.EnemyMaxResist)
							local takenInc = enemyDB:Sum("INC", dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
							local takenMore = enemyDB:More(dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
							effMult = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
							globalOutput["IgniteEffMult"] = effMult
							if breakdown and effMult ~= 1 then
								globalBreakdown.IgniteEffMult = breakdown.effMult("Chaos", resist, 0, takenInc, effMult, takenMore)
							end
						else
							local resist = m_min(enemyDB:Sum("BASE", nil, "FireResist", "ElementalResist") * calcLib.mod(enemyDB, nil, "FireResist", "ElementalResist"), data.misc.EnemyMaxResist)
							local takenInc = enemyDB:Sum("INC", dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
							local takenMore = enemyDB:More(dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
							effMult = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
							globalOutput["IgniteEffMult"] = effMult
							if breakdown and effMult ~= 1 then
								breakdown.IgniteEffMult = breakdown.effMult("Fire", resist, 0, takenInc, effMult, takenMore)
							end
						end
					end
					local effectMod = calcLib.mod(skillModList, dotCfg, "AilmentEffect")
					igniteStacks = 1
					if not skillData.triggeredOnDeath then
						igniteStacks = m_min(maxStacks, (output.HitChance / 100) * globalOutput.IgniteDuration / output.Time)
					end
					output.IgniteDPS = baseVal * effectMod * rateMod * effMult * igniteStacks
					globalOutput.IgniteDamage = output.IgniteDPS * globalOutput.IgniteDuration
					if skillFlags.igniteCanStack then
						output.IgniteDamage = output.IgniteDPS * globalOutput.IgniteDuration
						output.IgniteStacksMax = maxStacks
						output.TotalIgniteDPS = output.IgniteDPS
					end

					if breakdown then
						t_insert(breakdown.IgniteDPS, "x 0.9 ^8(ignite deals 90% per second)")
						t_insert(breakdown.IgniteDPS, s_format("= %.1f", baseVal, 1))
						breakdown.multiChain(breakdown.IgniteDPS, {
							label = "Ignite DPS:",
							base = s_format("%.1f ^8(total damage per second)", baseVal),
							{ "%.2f ^8(ailment effect modifier)", effectMod },
							{ "%.2f ^8(burn rate modifier)", rateMod },
							{ "%.3f ^8(effective DPS modifier)", effMult },
							{ "%d ^8(ignite stacks)", output.IgniteStacksMax },
							total = s_format("= %.1f ^8per second", output.IgniteDPS),
						})
						if output.CritIgniteDotMulti and (output.CritIgniteDotMulti ~= output.IgniteDotMulti) then
							local chanceFromHit = output.IgniteChanceOnHit / 100 * (1 - globalOutput.CritChance / 100)
							local chanceFromCrit = output.IgniteChanceOnCrit / 100 * output.CritChance / 100
							local totalFromHit = chanceFromHit / (chanceFromHit + chanceFromCrit)
							local totalFromCrit = chanceFromCrit / (chanceFromHit + chanceFromCrit)
							breakdown.IgniteDotMulti = breakdown.critDot(output.IgniteDotMulti, output.CritIgniteDotMulti, totalFromHit, totalFromCrit)
							output.IgniteDotMulti = (output.IgniteDotMulti * totalFromHit) + (output.CritIgniteDotMulti * totalFromCrit)
						end
						if skillFlags.igniteCanStack then
							breakdown.IgniteDamage = { }
							if isAttack then
								t_insert(breakdown.IgniteDamage, pass.label..":")
							end
							t_insert(breakdown.IgniteDamage, s_format("%.1f ^8(damage per second)", output.IgniteDPS))
							t_insert(breakdown.IgniteDamage, s_format("x %.2fs ^8(ignite duration)", globalOutput.IgniteDuration))
							t_insert(breakdown.IgniteDamage, s_format("= %.1f ^8damage per ignite stack", output.IgniteDamage))
						end
						if globalOutput.IgniteDuration ~= data.misc.IgniteDurationBase then
							globalBreakdown.IgniteDuration = {
								s_format("%.2fs ^8(base duration)", durationBase)
							}
							if durationMod ~= 1 then
								t_insert(globalBreakdown.IgniteDuration, s_format("x %.2f ^8(duration modifier)", durationMod))
							end
							if rateMod ~= 1 then
								t_insert(globalBreakdown.IgniteDuration, s_format("/ %.2f ^8(burn rate modifier)", rateMod))
							end
							if debuffDurationMult ~= 1 then
								t_insert(globalBreakdown.IgniteDuration, s_format("/ %.2f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
							end
							t_insert(globalBreakdown.IgniteDuration, s_format("= %.2fs", globalOutput.IgniteDuration))
						end
					end
				end
			end

			-- Calculate non-damaging ailments effect and duration modifiers
			local isBoss = env.configInput["enemyIsBoss"] ~= "None"
			local enemyBaseLife = data.monsterLifeTable[env.enemyLevel] * enemyDB:More(nil, "Life")
			local enemyMapLifeMult = 1
			local enemyMapAilmentMult = 1
			if env.enemyLevel >= 66 then
				enemyMapLifeMult = isBoss and data.mapLevelBossLifeMult[env.enemyLevel] or data.mapLevelLifeMult[env.enemyLevel]
				enemyMapAilmentMult = isBoss and data.mapLevelBossAilmentMult[env.enemyLevel] or enemyMapAilmentMult
			end
			local enemyTypeMult = isBoss and 7.68 or 1
			local enemyThreshold = enemyBaseLife * enemyTypeMult * enemyMapLifeMult * enemyMapAilmentMult * enemyDB:More(nil, "AilmentThreshold")

			local bonechill = output.BonechillEffect or enemyDB:Sum("BASE", nil, "DesiredBonechillEffect")
			local ailments = {
				["Chill"] = {
					effList = { 10, 20 },
					effect = function(damage, effectMod) return 50 * ((damage / enemyThreshold) ^ 0.4) * effectMod end,
					thresh = function(damage, value, effectMod) return damage * ((50 * effectMod / value) ^ 2.5) end,
					ramping = bonechill > 0,
				},
				["Shock"] = {
					effList = { 10, 20, 40 },
					effect = function(damage, effectMod) return 50 * ((damage / enemyThreshold) ^ 0.4) * effectMod end,
					thresh = function(damage, value, effectMod) return damage * ((50 * effectMod / value) ^ 2.5) end,
					ramping = true,
				},
				["Scorch"] = {
					effList = { 5, 10, 20 },
					effect = function(damage, effectMod) return 50 * ((damage / enemyThreshold) ^ 0.4) * effectMod end,
					thresh = function(damage, value, effectMod) return damage * ((50 * effectMod / value) ^ 2.5) end,
					ramping = true,
				},
				["Brittle"] = {
					effList = { 5, 10 },
					effect = function(damage, effectMod) return 25 * ((damage / enemyThreshold) ^ 0.4) * effectMod end,
					thresh = function(damage, value, effectMod) return damage * ((25 * effectMod / value) ^ 2.5) end,
					ramping = true,
				},
				["Sap"] = {
					effList = { 5, 10 },
					effect = function(damage, effectMod) return (100 / 3) * ((damage / enemyThreshold) ^ 0.4) * effectMod end,
					thresh = function(damage, value, effectMod) return damage * ((100 / 3 * effectMod / value) ^ 2.5) end,
					ramping = false,
				},
			}
			if activeSkill.skillTypes[SkillType.ChillingArea] or activeSkill.skillTypes[SkillType.NonHitChill] then
				skillFlags.chill = true
				output.ChillEffectMod = skillModList:Sum("INC", cfg, "EnemyChillEffect")
				output.ChillDurationMod = 1 + skillModList:Sum("INC", cfg, "EnemyChillDuration") / 100
				output.ChillSourceEffect = m_min(skillModList:Override(nil, "ChillMax") or ailmentData.Chill.max, m_floor(ailmentData.Chill.default * (1 + output.ChillEffectMod / 100)))
				if breakdown then
					breakdown.DotChill = { }
					breakdown.multiChain(breakdown.DotChill, {
						label = s_format("Effect of Chill: ^8(capped at %d%%)", skillModList:Override(nil, "ChillMax") or ailmentData.Chill.max),
						base = s_format("%d%% ^8(base)", ailmentData.Chill.default),
						{ "%.2f ^8(increased effect of chill)", 1 + output.ChillEffectMod / 100},
						total = s_format("= %.0f%%", output.ChillSourceEffect)
					})
				end
			end
			if (output.FreezeChanceOnHit + output.FreezeChanceOnCrit) > 0 then
				if globalBreakdown then
					globalBreakdown.FreezeDurationMod = {
						s_format("Ailment mode: %s ^8(can be changed in the Configuration tab)", igniteMode == "CRIT" and "Crits Only" or "Average Damage")
					}
				end
				local baseVal = calcAilmentDamage("Freeze", calcAverageSourceDamage("Freeze")) * skillModList:More(cfg, "FreezeAsThoughDealing")
				if baseVal > 0 then
					skillFlags.freeze = true
					skillFlags.chill = true
					output.FreezeDurationMod = 1 + skillModList:Sum("INC", cfg, "EnemyFreezeDuration") / 100 + enemyDB:Sum("INC", nil, "SelfFreezeDuration") / 100
					if breakdown then
						t_insert(breakdown.FreezeDPS, s_format("For freeze to apply for the minimum of 0.3 seconds, target must have no more than %.0f Ailment Threshold.", baseVal * 20 * output.FreezeDurationMod))
						t_insert(breakdown.FreezeDPS, s_format("^8(Ailment Threshold is about equal to Life except on bosses where it is about half of their life)"))
					end
				end
			end
			for ailment, val in pairs(ailments) do
				if (output[ailment.."ChanceOnHit"] + output[ailment.."ChanceOnCrit"]) > 0 then
					if globalBreakdown then
						globalBreakdown[ailment.."EffectMod"] = {
							s_format("Ailment mode: %s ^8(can be changed in the Configuration tab)", igniteMode == "CRIT" and "Crits Only" or "Average Damage")
						}
					end
					local damage = calcAilmentDamage(ailment, calcAverageSourceDamage(ailment)) * skillModList:More(cfg, ailment.."AsThoughDealing")
					if damage > 0 then
						skillFlags[string.lower(ailment)] = true
						local incDur = skillModList:Sum("INC", cfg, "Enemy"..ailment.."Duration") + enemyDB:Sum("INC", nil, "Self"..ailment.."Duration")
						local moreDur = skillModList:More(cfg, "Enemy"..ailment.."Duration") * enemyDB:More(nil, "Self"..ailment.."Duration")
						output[ailment.."Duration"] = ailmentData[ailment].duration * (1 + incDur / 100) * moreDur * debuffDurationMult
						output[ailment.."EffectMod"] = calcLib.mod(skillModList, cfg, "Enemy"..ailment.."Effect")
						if breakdown then
							local maximum = skillModList:Override(nil, ailment.."Max") or ailmentData[ailment].max
							local current = m_max(m_min(ailment == "Chill" and bonechill or globalOutput["Current"..ailment] or 0, maximum), 0)
							local desired = m_max(m_min(enemyDB:Sum("BASE", nil, "Desired"..ailment.."Val"), maximum), 0)
							if ailmentData[ailment].min ~= 0 then
								t_insert(val.effList, ailmentData[ailment].min)
							end
							if enemyThreshold > 0 then
								t_insert(val.effList, val.effect(damage, output[ailment.."EffectMod"]))
							end
							if not isValueInArray(val.effList, maximum) then
								t_insert(val.effList, maximum)
							end
							if current > 0 and not isValueInArray(val.effList, current) then
								t_insert(val.effList, current)
							end
							if desired > 0 and not isValueInArray(val.effList, desired) and current == 0 then
								t_insert(val.effList, desired)
							end
							breakdown[ailment.."DPS"].label = "Resulting ailment effect"..((current > 0 and val.ramping) and s_format(" ^8(with a ^7%s%% ^8%s on the enemy)^7", current, ailment) or "")
							breakdown[ailment.."DPS"].footer = s_format("^8(ailment threshold is about equal to life, except on bosses that have specific ailement thresholds)\n(the above table shows that when the enemy has X ailment threshold, you ^8%s for Y)", ailment:lower())
							breakdown[ailment.."DPS"].rowList = { }
							breakdown[ailment.."DPS"].colList = {
								{ label = "Ailment Threshold", key = "thresh" },
								{ label = ailment.." Effect", key = "effect" },
							}
							table.sort(val.effList)
							for _, value in ipairs(val.effList) do
								local thresh = val.thresh(damage, value, output[ailment.."EffectMod"])
								local decCheck = value / m_floor(value)
								local precision = ailmentData[ailment].precision
								value = m_floor(value * (10 ^ precision)) / (10 ^ precision)
								local valueFormat = "%."..tostring(precision).."f%%"
								local threshString = s_format("%d", thresh)..(m_floor(thresh + 0.5) == m_floor(enemyThreshold + 0.5) and s_format(" ^8(%s)", env.configInput.enemyIsBoss) or "")
								local labels = { }
								if decCheck == 1 and value ~= 0 then
									if ailment == "Chill" and value == bonechill then
										t_insert(labels, "bonechill")
									elseif value == current then
										t_insert(labels, "current")
									end
									if value == desired then
										t_insert(labels, "desired")
									end
									if value == maximum then
										t_insert(labels, "maximum")
									end
									if value == ailmentData[ailment].min then
										t_insert(labels, "minimum")
									end
								end
								t_insert(breakdown[ailment.."DPS"].rowList, {
									effect = s_format(valueFormat, value)..(next(labels) ~= nil and " ^8("..table.concat(labels, ", ")..")" or ""),
									thresh = threshString,
								})
							end
						end
						if breakdown and output[ailment.."Duration"] ~= ailmentData[ailment].duration then
							breakdown[ailment.."Duration"] = { }
							if isAttack then
								t_insert(breakdown[ailment.."Duration"], pass.label..":")
							end
							t_insert(breakdown[ailment.."Duration"], s_format("%.2fs ^8(base duration)", ailmentData[ailment].duration))
							if incDur ~= 0 then
								t_insert(breakdown[ailment.."Duration"], s_format("x %.2f ^8(increased/reduced duration)", 1 + incDur / 100))
							end
							if moreDur ~= 1 then
								t_insert(breakdown[ailment.."Duration"], s_format("x %.2f ^8(more/less duration)", moreDur))
							end
							if debuffDurationMult ~= 1 then
								t_insert(breakdown[ailment.."Duration"], s_format("/ %.2f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
							end
							t_insert(breakdown[ailment.."Duration"], s_format("= %.2fs", output[ailment.."Duration"]))
						end
					end
				end
			end

			-- Calculate knockback chance/distance
			output.KnockbackChance = m_min(100, output.KnockbackChanceOnHit * (1 - output.CritChance / 100) + output.KnockbackChanceOnCrit * output.CritChance / 100 + enemyDB:Sum("BASE", nil, "SelfKnockbackChance"))
			if output.KnockbackChance > 0 then
				output.KnockbackDistance = round(4 * calcLib.mod(skillModList, cfg, "EnemyKnockbackDistance"))
				if breakdown then
					breakdown.KnockbackDistance = {
						radius = output.KnockbackDistance,
					}
				end
			end

			-- Calculate enemy stun modifiers
			local enemyStunThresholdRed = -skillModList:Sum("INC", cfg, "EnemyStunThreshold")
			if enemyStunThresholdRed > 75 then
				output.EnemyStunThresholdMod = 1 - (75 + (enemyStunThresholdRed - 75) * 25 / (enemyStunThresholdRed - 50)) / 100
			else
				output.EnemyStunThresholdMod = 1 - enemyStunThresholdRed / 100
			end
			local base = skillData.baseStunDuration or 0.35
			local incDur = skillModList:Sum("INC", cfg, "EnemyStunDuration")
			local incRecov = enemyDB:Sum("INC", nil, "StunRecovery")
			output.EnemyStunDuration = base * (1 + incDur / 100) / (1 + incRecov / 100)
			if breakdown then
				if output.EnemyStunDuration ~= base then
					breakdown.EnemyStunDuration = {
						s_format("%.2fs ^8(base duration)", base),
					}
					if incDur ~= 0 then
						t_insert(breakdown.EnemyStunDuration, s_format("x %.2f ^8(increased/reduced stun duration)", 1 + incDur/100))
					end
					if incRecov ~= 0 then
						t_insert(breakdown.EnemyStunDuration, s_format("/ %.2f ^8(increased/reduced enemy stun recovery)", 1 + incRecov/100))
					end
					t_insert(breakdown.EnemyStunDuration, s_format("= %.2fs", output.EnemyStunDuration))
				end
			end

			-- Calculate impale chance and modifiers
			if canDeal.Physical and output.ImpaleChance > 0 then
				skillFlags.impale = true
				local impaleChance = m_min(output.ImpaleChance/100, 1)
				local maxStacks = skillModList:Sum("BASE", cfg, "ImpaleStacksMax") -- magic number: base stacks duration
				local configStacks = enemyDB:Sum("BASE", cfg, "Multiplier:ImpaleStacks")
				local impaleStacks = m_min(maxStacks, configStacks)

				local baseStoredDamage = data.misc.ImpaleStoredDamageBase
				local storedExpectedDamageIncOnBleed = skillModList:Sum("INC", cfg, "ImpaleEffectOnBleed")*skillModList:Sum("BASE", cfg, "BleedChance")/100
				local storedExpectedDamageInc = (skillModList:Sum("INC", cfg, "ImpaleEffect") + storedExpectedDamageIncOnBleed)/100
				local storedExpectedDamageMore = round(skillModList:More(cfg, "ImpaleEffect"), 2)
				local storedExpectedDamageModifier = (1 + storedExpectedDamageInc) * storedExpectedDamageMore
				local impaleStoredDamage = baseStoredDamage * storedExpectedDamageModifier
				local impaleHitDamageMod = impaleStoredDamage * impaleStacks  -- Source: https://www.reddit.com/r/pathofexile/comments/chgqqt/impale_and_armor_interaction/

				local enemyArmour = m_max(calcLib.val(enemyDB, "Armour"), 0)
				local impaleArmourReduction = calcs.armourReductionF(enemyArmour, impaleHitDamageMod * output.impaleStoredHitAvg)
				local impaleResist = m_min(m_max(0, enemyDB:Sum("BASE", nil, "PhysicalDamageReduction") + skillModList:Sum("BASE", cfg, "EnemyImpalePhysicalDamageReduction") + impaleArmourReduction), data.misc.DamageReductionCap)

				local impaleDMGModifier = impaleHitDamageMod * (1 - impaleResist / 100) * impaleChance

				globalOutput.ImpaleStacksMax = maxStacks
				globalOutput.ImpaleStacks = impaleStacks
				--ImpaleStoredDamage should be named ImpaleEffect or similar
				--Using the variable name ImpaleEffect breaks the calculations sidebar (?!)
				output.ImpaleStoredDamage = impaleStoredDamage * 100
				output.ImpaleModifier = 1 + impaleDMGModifier

				if breakdown then
					breakdown.ImpaleStoredDamage = {}
					t_insert(breakdown.ImpaleStoredDamage, "10% ^8(base value)")
					t_insert(breakdown.ImpaleStoredDamage, s_format("x %.2f ^8(increased effectiveness)", storedExpectedDamageModifier))
					t_insert(breakdown.ImpaleStoredDamage, s_format("= %.1f%%", output.ImpaleStoredDamage))

					breakdown.ImpaleModifier = {}
					t_insert(breakdown.ImpaleModifier, s_format("%d ^8(number of stacks, can be overridden in the Configuration tab)", impaleStacks))
					t_insert(breakdown.ImpaleModifier, s_format("x %.3f ^8(stored damage)", impaleStoredDamage))
					t_insert(breakdown.ImpaleModifier, s_format("x %.2f ^8(impale chance)", impaleChance))
					t_insert(breakdown.ImpaleModifier, s_format("x %.2f ^8(impale enemy physical damage reduction)", (1 - impaleResist / 100)))
					t_insert(breakdown.ImpaleModifier, s_format("= %.3f ^8(impale damage multiplier)", impaleDMGModifier))
				end
			end
		end
	*/
	/*
		TODO -- Combine secondary effect stats
		if isAttack then
			combineStat("BleedChance", "AVERAGE")
			combineStat("BleedDPS", "CHANCE_AILMENT", "BleedChance")
			combineStat("PoisonChance", "AVERAGE")
			combineStat("PoisonDPS", "CHANCE", "PoisonChance")
			combineStat("TotalPoisonDPS", "DPS")
			combineStat("PoisonDamage", "CHANCE", "PoisonChance")
			if skillData.showAverage then
				combineStat("TotalPoisonAverageDamage", "DPS")
			else
				combineStat("TotalPoisonStacks", "DPS")
			end
			combineStat("IgniteChance", "AVERAGE")
			combineStat("IgniteDPS", "CHANCE_AILMENT", "IgniteChance")
			if skillFlags.igniteCanStack then
				combineStat("IgniteDamage", "CHANCE", "IgniteChance")
				if skillData.showAverage then
					combineStat("TotalIgniteAverageDamage", "DPS")
					combineStat("IgniteStacksMax", "DPS")
					combineStat("TotalIgniteDPS", "DPS")
				else
					combineStat("IgniteStacksMax", "DPS")
					combineStat("TotalIgniteDPS", "DPS")
				end
			end
			combineStat("ChillEffectMod", "AVERAGE")
			combineStat("ChillDuration", "AVERAGE")
			combineStat("ShockChance", "AVERAGE")
			combineStat("ShockDuration", "AVERAGE")
			combineStat("ShockEffectMod", "AVERAGE")
			combineStat("FreezeChance", "AVERAGE")
			combineStat("FreezeDurationMod", "AVERAGE")
			combineStat("ScorchChance", "AVERAGE")
			combineStat("ScorchEffectMod", "AVERAGE")
			combineStat("ScorchDuration", "AVERAGE")
			combineStat("BrittleChance", "AVERAGE")
			combineStat("BrittleEffectMod", "AVERAGE")
			combineStat("BrittleDuration", "AVERAGE")
			combineStat("SapChance", "AVERAGE")
			combineStat("SapEffectMod", "AVERAGE")
			combineStat("SapDuration", "AVERAGE")
			combineStat("ImpaleChance", "AVERAGE")
			combineStat("ImpaleStoredDamage", "AVERAGE")
			combineStat("ImpaleModifier", "CHANCE", "ImpaleChance")
		end

		if skillFlags.hit and skillData.decay and canDeal.Chaos then
			-- Calculate DPS for Essence of Delirium's Decay effect
			skillFlags.decay = true
			activeSkill.decayCfg = {
				skillName = skillCfg.skillName,
				skillPart = skillCfg.skillPart,
				skillTypes = skillCfg.skillTypes,
				slotName = skillCfg.slotName,
				flags = ModFlag.Dot,
				keywordFlags = bor(band(skillCfg.keywordFlags, bnot(KeywordFlag.Hit)), KeywordFlag.ChaosDot),
			}
			local dotCfg = activeSkill.decayCfg
			local effMult = 1
			if env.mode_effective then
				local resist = m_min(enemyDB:Sum("BASE", nil, "ChaosResist") * calcLib.mod(enemyDB, nil, "ChaosResist"), data.misc.EnemyMaxResist)
				local takenInc = enemyDB:Sum("INC", nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
				local takenMore = enemyDB:More(nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
				effMult = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
				output["DecayEffMult"] = effMult
				if breakdown and effMult ~= 1 then
					breakdown.DecayEffMult = breakdown.effMult("Chaos", resist, 0, takenInc, effMult, takenMore)
				end
			end
			local inc = skillModList:Sum("INC", dotCfg, "Damage", "ChaosDamage")
			local more = round(skillModList:More(dotCfg, "Damage", "ChaosDamage"), 2)
			local mult = skillModList:Sum("BASE", dotTypeCfg, "DotMultiplier", "ChaosDotMultiplier")
			output.DecayDPS = skillData.decay * (1 + inc/100) * more * (1 + mult/100) * effMult
			output.DecayDuration = 8 * debuffDurationMult
			if breakdown then
				breakdown.DecayDPS = { }
				breakdown.dot(breakdown.DecayDPS, skillData.decay, inc, more, mult, nil, nil, effMult, output.DecayDPS)
				if output.DecayDuration ~= 8 then
					breakdown.DecayDuration = {
						s_format("%.2fs ^8(base duration)", 8)
					}
					if debuffDurationMult ~= 1 then
						t_insert(breakdown.DecayDuration, s_format("/ %.2f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
					end
					t_insert(breakdown.DecayDuration, s_format("= %.2fs", output.DecayDuration))
				end
			end
		end
	*/
	/*
		TODO -- Calculate skill DOT components
		local dotCfg = {
			skillName = skillCfg.skillName,
			skillPart = skillCfg.skillPart,
			skillTypes = skillCfg.skillTypes,
			slotName = skillCfg.slotName,
			flags = bor(ModFlag.Dot, skillCfg.flags),
			keywordFlags = band(skillCfg.keywordFlags, bnot(KeywordFlag.Hit)),
		}
		if bor(dotCfg.flags, ModFlag.Area) == dotCfg.flags and not skillData.dotIsArea then
			dotCfg.flags = band(dotCfg.flags, bnot(ModFlag.Area))
		end
		if bor(dotCfg.flags, ModFlag.Projectile) == dotCfg.flags and not skillData.dotIsProjectile then
			dotCfg.flags = band(dotCfg.flags, bnot(ModFlag.Projectile))
		end
		if bor(dotCfg.flags, ModFlag.Spell) == dotCfg.flags and not skillData.dotIsSpell then
			dotCfg.flags = band(dotCfg.flags, bnot(ModFlag.Spell))
		end
		if bor(dotCfg.flags, ModFlag.Attack) == dotCfg.flags and not skillData.dotIsAttack then
			dotCfg.flags = band(dotCfg.flags, bnot(ModFlag.Attack))
		end
		if bor(dotCfg.flags, ModFlag.Hit) == dotCfg.flags and not skillData.dotIsHit then
			dotCfg.flags = band(dotCfg.flags, bnot(ModFlag.Hit))
		end
	*/
	/*
		TODO -- spell_damage_modifiers_apply_to_skill_dot does not apply to enemy damage taken
		local dotTakenCfg = copyTable(dotCfg, true)
		if (skillData.dotIsSpell) then
			dotTakenCfg.flags = band(dotTakenCfg.flags, bnot(ModFlag.Spell))
		end

		activeSkill.dotCfg = dotCfg
		output.TotalDotInstance = 0

		runSkillFunc("preDotFunc")

		for _, damageType in ipairs(dmgTypeList) do
			local dotTypeCfg = copyTable(dotCfg, true)
			dotTypeCfg.keywordFlags = bor(dotTypeCfg.keywordFlags, KeywordFlag[damageType.."Dot"])
			activeSkill["dot"..damageType.."Cfg"] = dotTypeCfg
			local baseVal
			if canDeal[damageType] then
				baseVal = skillData[damageType.."Dot"] or 0
			else
				baseVal = 0
			end
			if baseVal > 0 or (output[damageType.."Dot"] or 0) > 0 then
				skillFlags.dot = true
				local effMult = 1
				if env.mode_effective then
					local resist = 0
					local takenInc = enemyDB:Sum("INC", dotTakenCfg, "DamageTaken", "DamageTakenOverTime", damageType.."DamageTaken", damageType.."DamageTakenOverTime")
					local takenMore = enemyDB:More(dotTakenCfg, "DamageTaken", "DamageTakenOverTime", damageType.."DamageTaken", damageType.."DamageTakenOverTime")
					if damageType == "Physical" then
						resist = m_max(0, m_min(enemyDB:Sum("BASE", nil, "PhysicalDamageReduction"), data.misc.DamageReductionCap))
					else
						if env.modDB:Flag(nil, "Enemy"..damageType.."ResistEqualToYours") then
							resist = env.player.output[damageType.."Resist"]
						else
							resist = enemyDB:Sum("BASE", nil, damageType.."Resist")
							if isElemental[damageType] then
								local base = resist + enemyDB:Sum("BASE", dotTypeCfg, "ElementalResist")
								resist = base * calcLib.mod(enemyDB, nil, damageType.."Resist")
								takenInc = takenInc + enemyDB:Sum("INC", dotTypeCfg, "ElementalDamageTaken")
							end
						end
						resist = m_min(resist, data.misc.EnemyMaxResist)
					end
					effMult = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
					output[damageType.."DotEffMult"] = effMult
					if breakdown and effMult ~= 1 then
						breakdown[damageType.."DotEffMult"] = breakdown.effMult(damageType, resist, 0, takenInc, effMult, takenMore)
					end
				end
				local inc = skillModList:Sum("INC", dotTypeCfg, "Damage", damageType.."Damage", isElemental[damageType] and "ElementalDamage" or nil)
				local more = round(skillModList:More(dotTypeCfg, "Damage", damageType.."Damage", isElemental[damageType] and "ElementalDamage" or nil), 2)
				local mult = skillModList:Sum("BASE", dotTypeCfg, "DotMultiplier", damageType.."DotMultiplier")
				local aura = activeSkill.skillTypes[SkillType.Aura] and not activeSkill.skillTypes[SkillType.RemoteMined] and calcLib.mod(skillModList, dotTypeCfg, "AuraEffect")
				local total = baseVal * (1 + inc/100) * more * (1 + mult/100) * (aura or 1) * effMult
				if output[damageType.."Dot"] == 0 or output[damageType.."Dot"] == nil then
					output[damageType.."Dot"] = total
					output.TotalDotInstance = output.TotalDotInstance + total
				else
					output.TotalDotInstance = output.TotalDotInstance + total + (output[damageType.."Dot"] or 0)
				end
				if breakdown then
					breakdown[damageType.."Dot"] = { }
					breakdown.dot(breakdown[damageType.."Dot"], baseVal, inc, more, mult, nil, aura, effMult, total)
				end
			end
		end
		if skillModList:Flag(nil, "DotCanStack") then
			skillFlags.DotCanStack = true
			local speed = output.Speed
			-- Check if skill is being triggered via Mine (e.g., Blastchain Mine Support) or Trap
			-- if "yes", you cannot use output.Speed but rather should use output.MineLayingSpeed or output.TrapThrowingSpeed
			if band(dotCfg.keywordFlags, KeywordFlag.Mine) ~= 0 then
				speed = output.MineLayingSpeed
			elseif band(dotCfg.keywordFlags, KeywordFlag.Trap) ~= 0 then
				speed = output.TrapThrowingSpeed
			end
			output.TotalDot = output.TotalDotInstance * speed * output.Duration * (skillData.dpsMultiplier or 1) * quantityMultiplier
			if breakdown then
				breakdown.TotalDot = {
					s_format("%.1f ^8(Damage per Instance)", output.TotalDotInstance),
					s_format("x %.2f ^8(hits per second)", speed),
					s_format("x %.2f ^8(skill duration)", output.Duration),
				}
				if skillData.dpsMultiplier then
					t_insert(breakdown.TotalDot, s_format("x %g ^8(DPS multiplier for this skill)", skillData.dpsMultiplier))
				end
				if quantityMultiplier > 1 then
					t_insert(breakdown.TotalDot, s_format("x %g ^8(quantity multiplier for this skill)", quantityMultiplier))
				end
				t_insert(breakdown.TotalDot, s_format("= %.1f", output.TotalDot))
			end
		else
			output.TotalDot = output.TotalDotInstance
		end
	*/
	/*
		TODO -- The Saviour
		if activeSkill.activeEffect.grantedEffect.name == "Reflection" then
			local usedSkill = nil
			local usedSkillBestDps = 0
			local calcMode = env.mode == "CALCS" and "CALCS" or "MAIN"
			for _, triggerSkill in ipairs(actor.activeSkillList) do
				if triggerSkill ~= activeSkill and triggerSkill.skillTypes[SkillType.Attack] and band(triggerSkill.skillCfg.flags, bor(ModFlag.Sword, ModFlag.Weapon1H)) == bor(ModFlag.Sword, ModFlag.Weapon1H) then
					-- Grab a fully-processed by calcs.perform() version of the skill that Mirage Warrior(s) will use
					local uuid = cacheSkillUUID(triggerSkill)
					if not GlobalCache.cachedData[calcMode][uuid] then
						calcs.buildActiveSkill(env, calcMode, triggerSkill)
						env.dontCache = true
					end
					-- We found a skill and it can crit
					if GlobalCache.cachedData[calcMode][uuid] and GlobalCache.cachedData[calcMode][uuid].CritChance and GlobalCache.cachedData[calcMode][uuid].CritChance > 0 then
						if not usedSkill then
							usedSkill = GlobalCache.cachedData[calcMode][uuid].ActiveSkill
							usedSkillBestDps = GlobalCache.cachedData[calcMode][uuid].TotalDPS
						else
							if GlobalCache.cachedData[calcMode][uuid].TotalDPS > usedSkillBestDps then
								usedSkill = GlobalCache.cachedData[calcMode][uuid].ActiveSkill
								usedSkillBestDps = GlobalCache.cachedData[calcMode][uuid].TotalDPS
							end
						end
					end
				end
			end

			if usedSkill then
				local moreDamage = activeSkill.skillModList:Sum("BASE", activeSkill.skillCfg, "SaviourMirageWarriorLessDamage")
				local maxMirageWarriors = activeSkill.skillModList:Sum("BASE", activeSkill.skillCfg, "SaviourMirageWarriorMaxCount")
				local newSkill, newEnv = calcs.copyActiveSkill(env, calcMode, usedSkill)

				-- Add new modifiers to new skill (which already has all the old skill's modifiers)
				newSkill.skillModList:NewMod("Damage", "MORE", moreDamage, "The Saviour", activeSkill.ModFlags, activeSkill.KeywordFlags)
				if env.player.itemList["Weapon 1"] and env.player.itemList["Weapon 2"] and env.player.itemList["Weapon 1"].name == env.player.itemList["Weapon 2"].name then
					maxMirageWarriors = maxMirageWarriors / 2
				end
				newSkill.skillModList:NewMod("QuantityMultiplier", "BASE", maxMirageWarriors, "The Saviour Mirage Warriors", activeSkill.ModFlags, activeSkill.KeywordFlags)

				if usedSkill.skillPartName then
					env.player.mainSkill.skillPart = usedSkill.skillPart
					env.player.mainSkill.skillPartName = usedSkill.skillPartName
					env.player.mainSkill.infoMessage2 = usedSkill.activeEffect.grantedEffect.name
				else
					env.player.mainSkill.skillPartName = usedSkill.activeEffect.grantedEffect.name
				end

				-- Recalculate the offensive/defensive aspects of this new skill
				newEnv.player.mainSkill = newSkill
				calcs.perform(newEnv)
				env.player.mainSkill = newSkill

				env.player.mainSkill.infoMessage = tostring(maxMirageWarriors) .. " Mirage Warriors using " .. usedSkill.activeEffect.grantedEffect.name

				-- Re-link over the output
				env.player.output = newEnv.player.output
				if newSkill.minion then
					env.minion = newEnv.player.mainSkill.minion
					env.minion.output = newEnv.minion.output
				end

				-- Make any necessary corrections to output
				env.player.output.ManaCost = 0

				-- Re-link over the breakdown (if present)
				if newEnv.player.breakdown then
					env.player.breakdown = newEnv.player.breakdown

					-- Make any necessary corrections to breakdown
					env.player.breakdown.ManaCost = nil

					if newSkill.minion then
						env.minion.breakdown = newEnv.minion.breakdown
					end
				end
			else
				activeSkill.infoMessage2 = "No Saviour active skill found"
			end
		end
	*/
	/*
		TODO -- Calculate combined DPS estimate, including DoTs
		local baseDPS = output[(skillData.showAverage and "AverageDamage") or "TotalDPS"]
		output.CombinedDPS = baseDPS
		output.CombinedAvg = baseDPS
		if skillFlags.dot then
			output.CombinedDPS = output.CombinedDPS + (output.TotalDot or 0)
			output.WithDotDPS = baseDPS + (output.TotalDot or 0)
		end
		if quantityMultiplier > 1 and output.TotalPoisonDPS then
			output.TotalPoisonDPS = output.TotalPoisonDPS * quantityMultiplier
		end
		if skillData.showAverage then
			output.CombinedDPS = output.CombinedDPS + (output.TotalPoisonDPS or 0)
			output.CombinedAvg = output.CombinedAvg + (output.PoisonDamage or 0)
			output.WithPoisonDPS = baseDPS + (output.TotalPoisonAverageDamage or 0)
		else
			output.CombinedDPS = output.CombinedDPS + (output.TotalPoisonDPS or 0)
			output.WithPoisonDPS = baseDPS + (output.TotalPoisonDPS or 0)
		end
		if skillFlags.ignite then
			if skillFlags.igniteCanStack then
				if skillData.showAverage then
					output.CombinedDPS = output.CombinedDPS + output.TotalIgniteDPS
					output.CombinedAvg = output.CombinedDPS + output.IgniteDamage
				else
					output.CombinedDPS = output.CombinedDPS + output.TotalIgniteDPS
					output.WithIgniteDPS = baseDPS + output.TotalIgniteDPS
				end
			elseif skillData.showAverage then
				output.WithIgniteDPS = baseDPS + output.IgniteDamage
				output.CombinedDPS = output.CombinedDPS + output.IgniteDPS
				output.CombinedAvg = output.CombinedAvg + output.IgniteDamage
			else
				output.WithIgniteDPS = baseDPS + output.IgniteDPS
				output.CombinedDPS = output.CombinedDPS + output.IgniteDPS
			end
		else
			output.WithIgniteDPS = baseDPS
		end
		if skillFlags.bleed then
			if skillData.showAverage then
				output.WithBleedDPS = baseDPS + output.BleedDamage
				output.CombinedDPS = output.CombinedDPS + output.BleedDPS
				output.CombinedAvg = output.CombinedAvg + output.BleedDamage
			else
				output.WithBleedDPS = baseDPS + output.BleedDPS
				output.CombinedDPS = output.CombinedDPS + output.BleedDPS
			end
		else
			output.WithBleedDPS = baseDPS
		end
		if skillFlags.decay then
			output.CombinedDPS = output.CombinedDPS + output.DecayDPS
		end
		output.TotalDotDPS = (output.TotalDot or 0) + (output.TotalPoisonDPS or 0) + (output.TotalIgniteDPS or output.IgniteDPS or 0) + (output.BleedDPS or 0) + (output.DecayDPS or 0)
		if skillFlags.impale then
			if skillFlags.attack then
				output.ImpaleHit = ((output.MainHand.PhysicalHitAverage or output.OffHand.PhysicalHitAverage) + (output.OffHand.PhysicalHitAverage or output.MainHand.PhysicalHitAverage)) / 2 * (1-output.CritChance/100) + ((output.MainHand.PhysicalCritAverage or output.OffHand.PhysicalCritAverage) + (output.OffHand.PhysicalCritAverage or output.MainHand.PhysicalCritAverage)) / 2 * (output.CritChance/100)
				if skillData.doubleHitsWhenDualWielding and skillFlags.bothWeaponAttack then
					output.ImpaleHit = output.ImpaleHit * 2
				end
			else
				output.ImpaleHit = output.PhysicalHitAverage * (1-output.CritChance/100) + output.PhysicalCritAverage * (output.CritChance/100)
			end
			output.ImpaleDPS = output.ImpaleHit * ((output.ImpaleModifier or 1) - 1) * output.HitChance / 100 * (skillData.dpsMultiplier or 1)
			if skillData.showAverage then
				output.WithImpaleDPS = output.AverageDamage + output.ImpaleDPS
				output.CombinedAvg = output.CombinedAvg + output.ImpaleDPS
			else
				skillFlags.notAverage = true
				output.ImpaleDPS = output.ImpaleDPS * (output.HitSpeed or output.Speed)
				output.WithImpaleDPS = output.TotalDPS + output.ImpaleDPS
			end
			if quantityMultiplier > 1 then
				output.ImpaleDPS = output.ImpaleDPS * quantityMultiplier
			end
			output.CombinedDPS = output.CombinedDPS + output.ImpaleDPS
			if breakdown then
				breakdown.ImpaleDPS = {}
				t_insert(breakdown.ImpaleDPS, s_format("%.2f ^8(average physical hit)", output.ImpaleHit))
				t_insert(breakdown.ImpaleDPS, s_format("x %.2f ^8(chance to hit)", output.HitChance / 100))
				if skillFlags.notAverage then
					t_insert(breakdown.ImpaleDPS, output.HitSpeed and s_format("x %.2f ^8(hit rate)", output.HitSpeed) or s_format("x %.2f ^8(%s rate)", output.Speed, skillFlags.attack and "attack" or "cast"))
				end
				t_insert(breakdown.ImpaleDPS, s_format("x %.2f ^8(impale damage multiplier)", ((output.ImpaleModifier or 1) - 1)))
				if skillData.dpsMultiplier then
					t_insert(breakdown.ImpaleDPS, s_format("x %g ^8(dps multiplier for this skill)", skillData.dpsMultiplier))
				end
				if quantityMultiplier > 1 then
					t_insert(breakdown.ImpaleDPS, s_format("x %g ^8(quantity multiplier for this skill)", quantityMultiplier))
				end
				t_insert(breakdown.ImpaleDPS, s_format("= %.1f", output.ImpaleDPS))
			end
		end

		local bestCull = 1
		if activeSkill.mirage and activeSkill.mirage.output and activeSkill.mirage.output.TotalDPS then
			local mirageCount = activeSkill.mirage.count or 1
			output.MirageDPS = activeSkill.mirage.output.TotalDPS * mirageCount
			output.CombinedDPS = output.CombinedDPS + activeSkill.mirage.output.TotalDPS * mirageCount

			if activeSkill.mirage.output.IgniteDPS and activeSkill.mirage.output.IgniteDPS > (output.IgniteDPS or 0) then
				output.MirageDPS = output.MirageDPS + activeSkill.mirage.output.IgniteDPS
				output.IgniteDPS = 0
			end
			if activeSkill.mirage.output.BleedDPS and activeSkill.mirage.output.BleedDPS > (output.BleedDPS or 0) then
				output.MirageDPS = output.MirageDPS + activeSkill.mirage.output.BleedDPS
				output.BleedDPS = 0
			end

			if activeSkill.mirage.output.PoisonDPS then
				output.MirageDPS = output.MirageDPS + activeSkill.mirage.output.PoisonDPS * mirageCount
				output.CombinedDPS = output.CombinedDPS + activeSkill.mirage.output.PoisonDPS * mirageCount
			end
			if activeSkill.mirage.output.ImpaleDPS then
				output.MirageDPS = output.MirageDPS + activeSkill.mirage.output.ImpaleDPS * mirageCount
				output.CombinedDPS = output.CombinedDPS + activeSkill.mirage.output.ImpaleDPS * mirageCount
			end
			if activeSkill.mirage.output.DecayDPS then
				output.MirageDPS = output.MirageDPS + activeSkill.mirage.output.DecayDPS
				output.CombinedDPS = output.CombinedDPS + activeSkill.mirage.output.DecayDPS
			end
			if activeSkill.mirage.output.TotalDot and (skillFlags.DotCanStack or not output.TotalDot or output.TotalDot == 0) then
				output.MirageDPS = output.MirageDPS + activeSkill.mirage.output.TotalDot * (skillFlags.DotCanStack and mirageCount or 1)
				output.CombinedDPS = output.CombinedDPS + activeSkill.mirage.output.TotalDot * (skillFlags.DotCanStack and mirageCount or 1)
			end
			if activeSkill.mirage.output.CullMultiplier > 1 then
				bestCull = activeSkill.mirage.output.CullMultiplier
			end
		end

		bestCull = m_max(bestCull, output.CullMultiplier)
		output.CullingDPS = output.CombinedDPS * (bestCull - 1)
		output.CombinedDPS = output.CombinedDPS * bestCull
	*/
}
