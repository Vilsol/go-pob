package calculator2

import (
	"math"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

//pob2:src/Modules/CalcOffence.lua:229-235
func calcAreaMod(incArea, moreArea float64) float64 {
	baseAreaPercent := roundSymmetric(incArea * 100)
	finalAreaPercent := roundSymmetric(math.Max(0, moreArea) * 100)
	areaPercent := baseAreaPercent * finalAreaPercent / 100
	if areaPercent >= 0 {
		areaPercent = math.Floor(areaPercent)
	} else {
		areaPercent = math.Ceil(areaPercent)
	}
	return math.Max(0, areaPercent) / 100
}

// calcRadius calculates skill radius; radiusMod 1 is Lua's nil.
//
//pob2:src/Modules/CalcOffence.lua:242-247
func calcRadius(baseRadius, areaMod, radiusMod float64) float64 {
	radiusPercent := math.Floor(100 * math.Sqrt(areaMod))
	// Final radius modifiers apply after Area of Effect has been converted to radius.
	radiusPercent = math.Floor(radiusPercent * radiusMod)
	return math.Floor(baseRadius * radiusPercent / 100)
}

// calcMoltenStrikeTertiaryRadius calculates the tertiary radius for Molten Strike, correctly handling the deadzone.
//
//pob2:src/Modules/CalcOffence.lua:254-259
func calcMoltenStrikeTertiaryRadius(baseRadius, deadzoneRadius, areaMod, speedMod float64) float64 {
	// For now, we assume that PoE only rounds at the end.
	maxDistIgnoringSpeed := math.Sqrt(baseRadius*baseRadius*areaMod - deadzoneRadius*deadzoneRadius*(areaMod-1))
	return math.Floor((maxDistIgnoringSpeed-deadzoneRadius)*speedMod + deadzoneRadius)
}

// calcCrossbowReloadTime returns false where Lua returns nil.
//
//pob2:src/Modules/CalcOffence.lua:368-376
func calcCrossbowReloadTime(weaponData map[string]float64, boltSkill *ActiveSkill) (float64, bool) {
	baseReloadTime, ok := weaponData["ReloadTime"]
	if !ok {
		return 0, false
	}
	reloadTimeMulti := calcLibMod(boltSkill.SkillModList, boltSkill.SkillCfg, "ReloadSpeed", "Speed")
	return baseReloadTime / reloadTimeMulti, true
}

type crossbowAmmoStats struct {
	cost         map[string]float64
	boltCount    float64
	reloadTime   float64
	hasReload    bool
	hasCostTable bool
}

// calcCrossbowAmmoStats calculates stats from the parent Ammo skill that are not available on children,
// such as mana cost and reload speed.
//
//pob2:src/Modules/CalcOffence.lua:381-408
func calcCrossbowAmmoStats(env *Environment, actor *Actor, activeSkill *ActiveSkill) crossbowAmmoStats {
	// Iterate over all skills in activeSkillList. If one is an ammo skill from the same base gem as current skill, take those stats
	for _, skill := range actor.ActiveSkillList {
		if skill.SkillTypes["CrossbowAmmoSkill"] && skill.SkillCfg != nil && skill.SkillCfg.GemData == activeSkill.SkillCfg.GemData {
			// transfer the actual mods modifying base crossbow bolt count and reload speed from ammo skill to active skill
			for _, m := range skill.BaseSkillModList.Mods() {
				if m.Name() == "CrossbowBoltCount" || m.Name() == "ReloadSpeed" {
					activeSkill.SkillModList.ReplaceMod(mod.NewFloat(m.Name(), m.Type(), m.Value().Float()).Source(m.GetSource()))
				}
			}
			stats := crossbowAmmoStats{boltCount: activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "CrossbowBoltCount")}
			stats.cost, stats.hasCostTable = grantedEffectLevelCost(skill.ActiveEffect)
			stats.reloadTime, stats.hasReload = calcCrossbowReloadTime(weaponData(env, actor, 1), activeSkill)
			return stats
		}
	}
	// Ensure minimum 1 base bolt count in any case
	activeSkill.SkillModList.ReplaceMod(newMod("CrossbowBoltCount", mod.TypeBase, 1, mod.Source(activeSkill.ActiveEffect.GrantedEffect.Name)))
	stats := crossbowAmmoStats{boltCount: math.Max(activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "CrossbowBoltCount"), 1)} // ensure minimum one bolt
	stats.cost, stats.hasCostTable = grantedEffectLevelCost(activeSkill.ActiveEffect)
	stats.reloadTime, stats.hasReload = calcCrossbowReloadTime(weaponData(env, actor, 1), activeSkill)
	return stats
}

func grantedEffectLevelCost(effect *GemEffect) (map[string]float64, bool) {
	if effect == nil || effect.GrantedEffectLevel == nil || effect.GrantedEffectLevel.Cost == nil {
		return nil, false
	}
	return effect.GrantedEffectLevel.Cost, true
}

// calcSkillCooldown returns the cooldown, whether it was rounded to server ticks, the added cooldown
// (hasAdded false where Lua returns nil) and the chance not to consume a cooldown.
//
//pob2:src/Modules/CalcOffence.lua:410-433
func calcSkillCooldown(skillModList *moddb.ModList, skillCfg *moddb.ListCfg, skillData map[string]any) (float64, bool, float64, bool, float64) {
	var cooldown float64
	cooldownOverride := skillModList.Override(skillCfg, "CooldownRecovery")
	addedCooldown := skillModList.Sum(mod.TypeBase, skillCfg, "CooldownRecovery")
	temporalisCooldown := skillModList.Sum(mod.TypeBase, skillCfg, "CooldownRecoveryFromTemporalis")
	noCooldownChance := skillModList.Sum(mod.TypeBase, skillCfg, "CooldownChanceNotConsume")
	skillCooldown, hasCooldown := sdNum(skillData, "cooldown")
	cooldownBase := skillCooldown + addedCooldown
	if hasCooldown {
		cooldownBase += temporalisCooldown
	}
	if cooldownOverride != nil {
		cooldown = cooldownOverride.Float()
	} else {
		cooldown = cooldownBase / math.Max(0, calcLibMod(skillModList, skillCfg, "CooldownRecovery"))
	}
	if hasCooldown && temporalisCooldown != 0 {
		cooldown = math.Max(cooldown, 0.1)
	}
	// If a skill can store extra uses and has a cooldown, it doesn't round the cooldown value to server ticks
	storedUses, _ := sdNum(skillData, "storedUses")
	vaalStoredUses, _ := sdNum(skillData, "VaalStoredUses")
	if storedUses > 1 || vaalStoredUses > 1 || skillModList.Sum(mod.TypeBase, skillCfg, "AdditionalCooldownUses") > 0 {
		// if the skill does not have an innate cooldown, we need to pass the added otherwise things go boom
		if hasCooldown {
			return cooldown, false, 0, false, noCooldownChance
		}
		return cooldown, false, addedCooldown, true, noCooldownChance
	}
	cooldown = math.Ceil(cooldown*miscConstant("ServerTickRate")) / miscConstant("ServerTickRate")
	return cooldown, true, addedCooldown, true, noCooldownChance
}

//pob2:src/Modules/CalcOffence.lua:435-444
func calcWarcryCastTime(skillModList *moddb.ModList, skillCfg *moddb.ListCfg, skillData map[string]any, actor *Actor) float64 {
	baseSpeed := 1 / skillModList.Sum(mod.TypeBase, skillCfg, "WarcryCastTime")
	warcryCastTime := baseSpeed * calcLibMod(skillModList, skillCfg, "WarcrySpeed") * actionSpeedMod(actor)
	warcryCastTime = math.Min(warcryCastTime, miscConstant("ServerTickRate"))
	warcryCastTime = 1 / warcryCastTime
	if skillModList.Flag(skillCfg, "InstantWarcry") || truthy(skillData["SupportedByAutoexertion"]) {
		warcryCastTime = 0
	}
	return warcryCastTime
}

// calcBuffExpirationMult calculates effect of buff/debuff expiration rate on actors
//
//pob2:src/Modules/CalcOffence.lua:447-449
func calcBuffExpirationMult(actorDB moddb.ModStoreFuncs, cfg *moddb.ListCfg) float64 {
	return 1 / math.Max(miscConstant("BuffExpirationSlowCap"), calcLibMod(actorDB, cfg, "BuffExpireFaster"))
}

//pob2:src/Modules/CalcOffence.lua:451-464
func calcSkillDuration(skillModList *moddb.ModList, skillCfg *moddb.ListCfg, skillData map[string]any, env *Environment, enemyDB *moddb.ModDB) float64 {
	durationMod := calcLibMod(skillModList, skillCfg, durationModNames(skillData, "Duration", "PrimaryDuration", "DamagingAilmentDuration")...)
	durationMod = math.Max(durationMod, 0)
	skillDuration, _ := sdNum(skillData, "duration")
	durationBase := skillDuration + skillModList.Sum(mod.TypeBase, skillCfg, "Duration", "PrimaryDuration")
	duration := durationBase * durationMod
	debuffDurationMult := 1.0
	if env.ModeEffective {
		debuffDurationMult = calcBuffExpirationMult(enemyDB, skillCfg)
	}
	if truthy(skillData["debuff"]) {
		duration *= debuffDurationMult
	}
	return duration
}

// durationModNames appends MineDuration where skillData.mineDurationAppliesToSkill does; Lua passes nil otherwise.
func durationModNames(skillData map[string]any, names ...string) []string {
	if truthy(skillData["mineDurationAppliesToSkill"]) {
		return append(names, "MineDuration")
	}
	return names
}

// sdNum is skillData[key] as a number; false where it is nil or not a number.
func sdNum(skillData map[string]any, key string) (float64, bool) {
	switch v := skillData[key].(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	}
	return 0, false
}

// floorDec rounds down a number to the nearest dec decimal places.
//
//pob2:src/Modules/Common.lua:734-741
func floorDec(val float64, dec int) float64 {
	mult := math.Pow(10, float64(dec))
	return math.Floor(val*mult+0.0001) / mult
}

// maxOr is modList:Max(cfg, ...) or fallback: Max is nil when no MAX mod applies.
func maxOr(modList *moddb.ModList, cfg *moddb.ListCfg, fallback float64, names ...string) float64 {
	if len(modList.Tabulate(mod.TypeMAX, cfg, names...)) == 0 {
		return fallback
	}
	return modList.Max(cfg, names...)
}

// configListValue is env.configInput[name] for a list option: the saved value, else the defaultIndex
// value ConfigTab writes into the input; false where it is nil.
func configListValue(env *Environment, name string) (string, bool) {
	if v, ok := env.ConfigInput[name]; ok && v.IsString {
		return v.String, true
	}
	for _, cv := range configVarList {
		if cv.name == name && cv.defaultState != nil && cv.defaultState.IsString {
			return cv.defaultState.String, true
		}
	}
	return "", false
}

func flagsCfg(flags mod.MFlag) *moddb.ListCfg {
	return &moddb.ListCfg{Flags: &flags}
}

func keywordFlagsCfg(keywordFlags mod.KeywordFlag) *moddb.ListCfg {
	return &moddb.ListCfg{KeywordFlags: &keywordFlags}
}

func flagsKeywordCfg(flags mod.MFlag, keywordFlags mod.KeywordFlag) *moddb.ListCfg {
	return &moddb.ListCfg{Flags: &flags, KeywordFlags: &keywordFlags}
}

// copyMod is skillModList:NewMod(name, type, value, source, flags, keywordFlags, unpack(mod)).
func copyMod(name string, modType mod.Type, value float64, source mod.Source, flags mod.MFlag, keywordFlags mod.KeywordFlag, tags []mod.Tag) mod.Mod {
	m := mod.NewFloat(name, modType, value).Source(source).Flag(flags).KeywordFlag(keywordFlags)
	if len(tags) > 0 {
		m.Tag(tags...)
	}
	return m
}

// getConvertedModTagsMinion is calcLib.getConvertedModTags with minionMods set: a parent ActorCondition
// becomes the minion's own Condition.
//
//pob2:src/Modules/CalcTools.lua:238-253
func getConvertedModTagsMinion(m mod.Mod, multiplier float64) []mod.Tag {
	modifiers := getConvertedModTags(m, multiplier)
	for k, value := range m.Tags() {
		if ac, ok := value.(*mod.ActorConditionTag); ok && ac.Actor != nil && *ac.Actor == "parent" {
			modifiers[k] = mod.Condition(ac.VariableList...)
		}
	}
	return modifiers
}

func minionModifierMod(v any) mod.Mod {
	switch x := v.(type) {
	case mod.MinionModifier:
		return x.Mod
	case *mod.MinionModifier:
		return x.Mod
	}
	return nil
}

// offence performs the offence calculations up to the ailments; the ailment half
// (CalcOffence.lua:4968-6471) is not ported. The CALCS-mode
// statSetCalcs, the grantedEffect initialFunc/preSkillTypeFunc (no PoE2 skill has one) and the
// breakdowns are not ported.
//
//pob2:src/Modules/CalcOffence.lua:527-2424
func offence(env *Environment, actor *Actor, activeSkill *ActiveSkill) {
	if activeSkill == nil || activeSkill.SkillModList == nil || activeSkill.SkillCfg == nil ||
		activeSkill.ActiveEffect == nil || activeSkill.ActiveEffect.StatSet == nil {
		return
	}
	modDB := actor.ModDB
	output := actor.Output

	skillModList := activeSkill.SkillModList
	skillData := activeSkill.SkillData
	skillFlags := activeSkill.ActiveEffect.StatSet.SkillFlags
	if skillFlags == nil {
		skillFlags = map[string]bool{}
		activeSkill.ActiveEffect.StatSet.SkillFlags = skillFlags
	}
	skillCfg := activeSkill.SkillCfg
	if truthy(skillData["showAverage"]) {
		skillFlags["showAverage"] = true
	} else {
		skillFlags["notAverage"] = true
	}

	if skillFlags["disable"] {
		// Skill is disabled
		output["CombinedDPS"] = 0
		return
	}

	// Calculate armour break
	output["ArmourBreakPerHit"] = calcLibVal(skillModList, "ArmourBreakPerHit", skillCfg)

	getSkillNameFromFlag := func(flag string) (string, bool) {
		sourceMod := skillModList.Tabulate(mod.TypeFlag, nil, flag)
		if len(sourceMod) == 0 || sourceMod[0].Mod == nil {
			return "", false
		}
		dataSkill := env.data.effect(strings.Replace(string(sourceMod[0].Mod.GetSource()), "Skill:", "", 1))
		if dataSkill == nil {
			return "", false
		}
		return dataSkill.Name, true
	}

	ranges := &weaponRanges{}

	// Update skill data
	//pob2:block:src/Modules/CalcOffence.lua:728-744
	skillCfg.SkillCond["SkillIsTriggered"] = truthy(skillData["triggered"])
	if skillCfg.SkillCond["SkillIsTriggered"] {
		skillFlags["triggered"] = true
	}
	skillCfg.SkillCond["SkillIsFocused"] = truthy(skillData["chanceToTriggerOnFocus"])
	if skillCfg.SkillCond["SkillIsFocused"] {
		skillFlags["focused"] = true
	}
	for _, v := range skillModList.List(skillCfg, "SkillData") {
		var sd mod.SkillData
		switch x := v.(type) {
		case mod.SkillData:
			sd = x
		case *mod.SkillData:
			sd = *x
		default:
			continue
		}
		if sd.Merge == "MAX" {
			cur, _ := sdNum(skillData, sd.Key)
			skillData[sd.Key] = math.Max(sd.Value, cur)
		} else {
			skillData[sd.Key] = sd.SetValue()
		}
	}

	// Add addition stat bonuses
	//pob2:block:src/Modules/CalcOffence.lua:747-761
	if skillModList.Flag(nil, "TransfigurationOfBody") {
		skillModList.AddMod(newMod("Damage", mod.TypeIncrease, math.Floor(skillModList.Sum(mod.TypeIncrease, nil, "Life")*miscConstant("Transfiguration")), "Transfiguration of Body").Flag(ModFlagAttack))
	}
	if skillModList.Flag(nil, "TransfigurationOfMind") {
		skillModList.AddMod(newMod("Damage", mod.TypeIncrease, math.Floor(skillModList.Sum(mod.TypeIncrease, nil, "Mana")*miscConstant("Transfiguration")), "Transfiguration of Mind"))
	}
	if skillModList.Flag(nil, "TransfigurationOfSoul") {
		skillModList.AddMod(newMod("Damage", mod.TypeIncrease, math.Floor(skillModList.Sum(mod.TypeIncrease, nil, "EnergyShield")*miscConstant("Transfiguration")), "Transfiguration of Soul").Flag(ModFlagSpell))
	}
	if modDB.Flag(nil, "Elusive") && skillModList.Flag(nil, "SupportedByNightblade") {
		elusiveEffect := output["ElusiveEffectMod"] / 100
		nightbladeMulti := skillModList.Sum(mod.TypeBase, nil, "NightbladeElusiveCritMultiplier")
		skillModList.AddMod(newMod("CritMultiplier", mod.TypeBase, math.Floor(nightbladeMulti*elusiveEffect), "Nightblade"))
	}

	// set other limits
	//pob2:block:src/Modules/CalcOffence.lua:764-765
	output["ActiveTrapLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveTrapLimit")
	output["ActiveMineLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveMineLimit")

	// set flask scaling
	//pob2:block:src/Modules/CalcOffence.lua:768-772
	for _, name := range []string{"LifeFlaskRecovery", "LifeFlask1MaxCharges", "LifeFlask2MaxCharges", "ManaFlask1MaxCharges", "ManaFlask2MaxCharges"} {
		if v, ok := env.ItemModDB.Multipliers[name]; ok {
			output[name] = v
		}
	}

	weaponData1 := weaponData(env, actor, 1)
	weaponData2 := weaponData(env, actor, 2)
	weapon1Equipped := actor.ItemList["Weapon 1"] != nil && actor.ItemList["Weapon 1"].WeaponData[1] != nil

	// Energy Blade's weapon base swap (CalcOffence.lua:774-789) needs the weapon's base name on
	// actor.weaponData, which the item data does not carry; it is not ported.

	// account for Battlemage
	// Note: we check conditions of Main Hand weapon using actor.itemList as actor.weaponData1 is populated with unarmed values when no weapon slotted.
	//pob2:block:src/Modules/CalcOffence.lua:793-800
	if skillModList.Flag(nil, "Battlemage") && weapon1Equipped {
		multiplier := maxOr(skillModList, skillCfg, 100, "MainHandWeaponDamageAppliesToSpells") / 100
		for _, damageType := range dmgTypeList {
			skillModList.AddMod(newMod(damageType+"Min", mod.TypeBase, math.Floor(weaponData1[damageType+"Min"]*multiplier), "Battlemage").Flag(ModFlagSpell))
			skillModList.AddMod(newMod(damageType+"Max", mod.TypeBase, math.Floor(weaponData1[damageType+"Max"]*multiplier), "Battlemage").Flag(ModFlagSpell))
		}
	}
	//pob2:block:src/Modules/CalcOffence.lua:801-818
	weapon1info := weaponTypeInfo[actor.WeaponData1Type]
	_, hasWeapon2Info := weaponTypeInfo[actor.WeaponData2Type]
	// -- account for Spellblade
	if len(skillModList.Tabulate(mod.TypeMAX, skillCfg, "OneHandWeaponDamageAppliesToSpells")) > 0 && weapon1Equipped && weapon1info.melee && weapon1info.oneHand {
		spellbladeMulti := skillModList.Max(skillCfg, "OneHandWeaponDamageAppliesToSpells")
		multiplier := spellbladeMulti / 100
		if hasWeapon2Info {
			multiplier *= 0.6
		}
		for _, damageType := range dmgTypeList {
			skillModList.AddMod(newMod(damageType+"Min", mod.TypeBase, math.Floor(weaponData1[damageType+"Min"]*multiplier), "Spellblade Main Hand").Flag(ModFlagSpell))
			skillModList.AddMod(newMod(damageType+"Max", mod.TypeBase, math.Floor(weaponData1[damageType+"Max"]*multiplier), "Spellblade Main Hand").Flag(ModFlagSpell))
		}
		if hasWeapon2Info {
			for _, damageType := range dmgTypeList {
				skillModList.AddMod(newMod(damageType+"Min", mod.TypeBase, math.Floor(weaponData2[damageType+"Min"]*multiplier), "Spellblade Off Hand").Flag(ModFlagSpell))
				skillModList.AddMod(newMod(damageType+"Max", mod.TypeBase, math.Floor(weaponData2[damageType+"Max"]*multiplier), "Spellblade Off Hand").Flag(ModFlagSpell))
			}
		}
	}
	// Add bonus mods for Tactician's "Watch How I Do It" (technically this could be done in ModParser, but it would always add 10 mods instead of just the necessary ones)
	//pob2:block:src/Modules/CalcOffence.lua:820-834
	if actor.Parent != nil && modDB.Flag(nil, "GainMainHandDmgFromParent") && actor.Parent.ItemList["Weapon 1"] != nil {
		var modSource mod.Source
		for _, value := range modDB.Tabulate(mod.TypeFlag, nil, "GainMainHandDmgFromParent") {
			modSource = value.Mod.GetSource()
		}
		modValue := actor.Parent.ModDB.Sum(mod.TypeBase, &moddb.ListCfg{Source: &modSource}, "Multiplier:MainHandDamageToAllies")
		parentWeaponData1 := weaponData(env, actor.Parent, 1)
		for _, damageType := range dmgTypeList {
			for _, bound := range []string{"Min", "Max"} {
				if _, ok := parentWeaponData1[damageType+bound]; ok {
					tag := mod.PercentStat(damageType+bound+"OnWeapon 1", modValue)
					tag.TagActor = "parent"
					skillModList.AddMod(newMod(damageType+bound, mod.TypeBase, 1, modSource, tag, mod.SkillType("Attack")))
				}
			}
		}
	}
	// Apply Gemling's "Integrated Efficiency" Passive
	//pob2:block:src/Modules/CalcOffence.lua:836-873
	if skillModList.Flag(nil, "SkillDamageIncreasedPerRedSupport") ||
		skillModList.Flag(nil, "SkillSpeedIncreasedPerGreenSupport") ||
		skillModList.Flag(nil, "SkillCritChanceIncreasedPerBlueSupport") {
		redSupportCount, greenSupportCount, blueSupportCount := 0.0, 0.0, 0.0
		for _, support := range activeSkill.SupportList {
			if support.GemData == nil || support.GemData.GrantedEffect == nil {
				continue
			}
			switch support.GemData.GrantedEffect.Color {
			case 1:
				redSupportCount++
			case 2:
				greenSupportCount++
			case 3:
				blueSupportCount++
			}
		}
		lookupModData := func(flag string) (mod.Source, float64) {
			result := actor.ModDB.Tabulate(mod.TypeFlag, nil, flag)
			if len(result) == 0 {
				return "", 0
			}
			return result[0].Mod.GetSource(), result[0].Mod.Value().Float()
		}
		// Conditionally check each to avoid future issues if a mod is reused in isolation
		if skillModList.Flag(nil, "SkillDamageIncreasedPerRedSupport") {
			source, value := lookupModData("SkillDamageIncreasedPerRedSupport")
			skillModList.AddMod(newMod("Damage", mod.TypeIncrease, redSupportCount*value, source))
		}
		if skillModList.Flag(nil, "SkillSpeedIncreasedPerGreenSupport") {
			source, value := lookupModData("SkillSpeedIncreasedPerGreenSupport")
			skillModList.AddMod(newMod("Speed", mod.TypeIncrease, greenSupportCount*value, source))
			skillModList.AddMod(newMod("WarcrySpeed", mod.TypeIncrease, greenSupportCount*value, source))
			skillModList.AddMod(newMod("TotemPlacementSpeed", mod.TypeIncrease, greenSupportCount*value, source))
		}
		if skillModList.Flag(nil, "SkillCritChanceIncreasedPerBlueSupport") {
			source, value := lookupModData("SkillCritChanceIncreasedPerBlueSupport")
			skillModList.AddMod(newMod("CritChance", mod.TypeIncrease, blueSupportCount*value, source))
		}
	}
	offenceStatConversions(env, actor, activeSkill, skillFlags, skillCfg)

	//pob2:block:src/Modules/CalcOffence.lua:1107-1113
	if truthy(skillData["arrowSpeedAppliesToAreaOfEffect"]) {
		// Arrow Speed conversion for Galvanic Arrow
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, flagsCfg(ModFlagBow), "ProjectileSpeed") {
			m := value.Mod
			skillModList.AddMod(copyMod("AreaOfEffect", mod.TypeIncrease, m.Value().Float(), m.GetSource(), m.Flags(), m.KeywordFlags(), m.Tags()))
		}
	}
	//pob2:block:src/Modules/CalcOffence.lua:1114-1128
	if activeSkill.SkillTypes["Barrageable"] && skillModList.Flag(nil, "SequentialProjectiles") && !skillModList.Flag(nil, "OneShotProj") && !skillModList.Flag(nil, "NoAdditionalProjectiles") && !skillModList.Flag(nil, "TriggeredBySnipe") {
		// Applies DPS multiplier based on projectile count
		if skillModList.Sum(mod.TypeBase, skillCfg, "BarrageRepeats") > 0 {
			dpsMulti := (1 + skillModList.Sum(mod.TypeBase, skillCfg, "BarrageRepeats")) * calcLibMod(skillModList, skillCfg, "BarrageRepeatDamage")
			skillModList.AddMod(newMod("DPS", mod.TypeMore, dpsMulti, "Barrage Repeats"))
		} else {
			additionalProjectiles := (skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount") + 2*skillModList.Sum(mod.TypeBase, skillCfg, "TwoAdditionalProjectilesChance")/100 + skillModList.Sum(mod.TypeBase, skillCfg, "SurpassingProjectileChance")/100 - 1) * skillModList.More(skillCfg, "ProjectileCount")
			if additionalProjectiles > 0 {
				barrageAttackTimePenalty := skillModList.Sum(mod.TypeBase, skillCfg, "BarrageAttackTimePenalty")
				if barrageAttackTimePenalty == 0 {
					barrageAttackTimePenalty = 100
				} // If not otherwise specified on the skill, each additional projectile adds 100% of attack time
				skillModList.ReplaceMod(newMod("SkillAttackTime", mod.TypeMore, barrageAttackTimePenalty*additionalProjectiles, mod.Source(activeSkill.ActiveEffect.GrantedEffect.Name+": "+luaNum(barrageAttackTimePenalty)+"% attack time per add. projectile")))
				skillModList.AddMod(newMod("DPS", mod.TypeMore, skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount")+2*skillModList.Sum(mod.TypeBase, skillCfg, "TwoAdditionalProjectilesChance")/100+skillModList.Sum(mod.TypeBase, skillCfg, "SurpassingProjectileChance")/100, "Barrage Repeats"))
			}
		}
	}
	offenceRepeats(env, activeSkill, skillFlags, output)
	offenceWeaponSpellDamage(env, actor, activeSkill)

	// Crossbow calculations
	// Calculate ammo stats for bolt skills
	//pob2:block:src/Modules/CalcOffence.lua:1274-1279
	if activeSkill.SkillTypes["CrossbowSkill"] && !(activeSkill.SkillTypes["Grenade"] || activeSkill.SkillTypes["CrossbowAmmoSkill"]) {
		ammoStats := calcCrossbowAmmoStats(env, actor, activeSkill)
		if activeSkill.ActiveEffect.GrantedEffectLevel != nil {
			activeSkill.ActiveEffect.GrantedEffectLevel.Cost = ammoStats.cost // inherit base mana cost
		}
		skillData["boltCount"] = ammoStats.boltCount
		if ammoStats.hasReload {
			skillData["reloadTime"] = ammoStats.reloadTime
		} else {
			delete(skillData, "reloadTime")
		}
	}

	//pob2:block:src/Modules/CalcOffence.lua:1281-1284
	if activeSkill.SkillTypes["Grenade"] {
		detonateTwice := math.Min(skillModList.Sum(mod.TypeBase, skillCfg, "GrenadeActivateTwice"), 100)
		modDB.AddMod(newMod("DPS", mod.TypeMore, detonateTwice, "Grenade Activate Twice"))
	}

	// Dual wield DPS multiplier
	//pob2:block:src/Modules/CalcOffence.lua:1287-1289
	if skillFlags["bothWeaponAttack"] && truthy(skillData["doubleHitsWhenDualWielding"]) {
		skillModList.AddMod(newMod("DPS", mod.TypeMore, 100, "Hits with both weapons"))
	}

	//pob2:block:src/Modules/CalcOffence.lua:1291-1336
	if skillModList.Flag(nil, "HasSeals") && !skillModList.Flag(nil, "NoRepeatBonuses") {
		// Applies seal bonuses based on seal count
		castTime := activeSkill.ActiveEffect.GrantedEffect.CastTime
		totalCastSpeed := 1 / castTime * calcLibMod(skillModList, skillCfg, "Speed")
		output["SealCooldown"] = castTime * skillModList.Sum(mod.TypeBase, skillCfg, "SealGainFrequency") / calcLibMod(skillModList, skillCfg, "SealGainFrequency") / 100
		output["SealMax"] = skillModList.Sum(mod.TypeBase, skillCfg, "SealCount")
		output["TimeMaxSeals"] = output["SealCooldown"] * output["SealMax"]

		if skillModList.Flag(nil, "DamageSeal") {
			output["AverageBurstHits"] = output["SealMax"]
		}
		mainSkill := env.Player.MainSkill
		if _, hasOverride := skillData["hitTimeOverride"]; skillModList.Flag(nil, "DamageSeal") && !hasOverride {
			skillName, ok := getSkillNameFromFlag("DamageSeal")
			if !ok {
				skillName = "Support"
			}
			if skillModList.Flag(nil, "UseMaxUnleash") {
				for _, value := range skillModList.Tabulate(mod.TypeIncrease, &moddb.ListCfg{}, "MaxSealCrit") {
					m := value.Mod
					skillModList.AddMod(copyMod("CritChance", mod.TypeIncrease, m.Value().Float(), m.GetSource(), m.Flags(), m.KeywordFlags(), m.Tags()))
				}
				mainSkill.SkillModList.AddMod(newMod("DPS", mod.TypeMore, (output["SealMax"]*calcLibMod(skillModList, skillCfg, "SealRepeatPenalty"))*100, mod.Source(skillName)))
				mainSkill.SkillData["hitTimeOverride"] = math.Max(output["TimeMaxSeals"], totalCastSpeed*1.1)
			} else {
				mainSkill.SkillModList.AddMod(newMod("DPS", mod.TypeMore, round(1/output["SealCooldown"]/(totalCastSpeed*1.1)*calcLibMod(skillModList, skillCfg, "SealRepeatPenalty")*100, 2), mod.Source(skillName)))
			}
		}
		if skillModList.Flag(nil, "AreaSeal") {
			skillName, ok := getSkillNameFromFlag("AreaSeal")
			if !ok {
				skillName = "Support"
			}
			sealArea := (calcLibMod(skillModList, skillCfg, "SealRepeatPenalty") - 1) * 100
			if skillModList.Flag(nil, "UseMaxUnleash") {
				mainSkill.SkillModList.AddMod(newMod("AreaOfEffect", mod.TypeIncrease, output["SealMax"]*sealArea, mod.Source(skillName)))
				mainSkill.SkillData["hitTimeOverride"] = math.Max(output["TimeMaxSeals"], totalCastSpeed*1.1)
			} else {
				mainSkill.SkillModList.AddMod(newMod("AreaOfEffect", mod.TypeIncrease, round(1/output["SealCooldown"]/totalCastSpeed*sealArea, 2), mod.Source(skillName)))
			}
		}
	}
	offenceRandomElement(env, activeSkill, skillFlags)

	// momentum stacks
	//pob2:block:src/Modules/CalcOffence.lua:1399-1411
	if skillModList.Flag(nil, "SupportedByMomentum") {
		maxMomentumStacks := skillModList.Sum(mod.TypeBase, skillCfg, "MomentumStacksMax")
		extraMomentumStacks := skillModList.Sum(mod.TypeBase, skillCfg, "MomentumStacksExtra")
		if maxMomentumStacks > 0 {
			if !modDB.HasMod(mod.TypeBase, nil, "Multiplier:MomentumStacks") {
				modDB.AddMod(newMod("Multiplier:MomentumStacks", mod.TypeBase, math.Min((maxMomentumStacks+extraMomentumStacks)/2, maxMomentumStacks), "Config", mod.Condition("Combat")))
			} else if modDB.Sum(mod.TypeBase, nil, "Multiplier:MomentumStacks") > maxMomentumStacks {
				modDB.ReplaceMod(newMod("Multiplier:MomentumStacks", mod.TypeBase, maxMomentumStacks, "Config", mod.Condition("Combat")))
			}
		} else if modDB.HasMod(mod.TypeBase, nil, "Multiplier:MomentumStacks") {
			modDB.ReplaceMod(newMod("Multiplier:MomentumStacks", mod.TypeBase, 0, "Config"))
		}
	}

	offenceSkillTypeStats(env, actor, activeSkill, skillFlags, output, ranges)
	offenceDurations(env, activeSkill, skillFlags, output)
	hasCosts := offenceCosts(env, actor, activeSkill, output)

	// Eldritch Battery adds maximum Energy Shield to the Mana available for payable-cost damage.
	//pob2:block:src/Modules/CalcOffence.lua:2397
	output["ManaCostPayablePool"] = output["ManaUnreserved"]
	if modDB.Flag(nil, "EnergyShieldProtectsMana") {
		output["ManaCostPayablePool"] += output["EnergyShield"]
	}

	// account for Sacrificial Zeal
	// Note: Sacrificial Zeal grants Added Spell Physical Damage equal to 25% of the Skill's Mana Cost, and causes you to take Physical Damage over Time, for 4 seconds
	//pob2:block:src/Modules/CalcOffence.lua:2401-2405
	if skillModList.Flag(nil, "Condition:SacrificialZeal") && hasCosts["Mana"] {
		multiplier := 0.25
		skillModList.AddMod(newMod("PhysicalMin", mod.TypeBase, math.Floor(output["ManaCost"]*multiplier), "Sacrificial Zeal").Flag(ModFlagSpell))
		skillModList.AddMod(newMod("PhysicalMax", mod.TypeBase, math.Floor(output["ManaCost"]*multiplier), "Sacrificial Zeal").Flag(ModFlagSpell))
	}

	monsterLife := offencePreDamage(env, activeSkill, skillFlags, output)

	offenceHitHalf(env, actor, activeSkill, skillFlags, ranges, monsterLife)
}

// weaponRanges stands in for actor.weaponRange1/2, which offence sets for melee skills.
type weaponRanges struct {
	weaponRange1, weaponRange2 float64
}

//pob2:src/Modules/CalcOffence.lua:566-684
func calcAreaOfEffect(ranges *weaponRanges, skillModList *moddb.ModList, skillCfg *moddb.ListCfg, skillData map[string]any, skillFlags map[string]bool, output map[string]float64) {
	incArea, moreArea := calcLibMods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectPrimary")
	output["AreaOfEffectMod"] = calcAreaMod(incArea, moreArea)
	radiusMod := calcLibMod(skillModList, skillCfg, "AreaOfEffectRadius")
	if truthy(skillData["radiusIsWeaponRange"]) {
		rng := 0.0
		if skillFlags["weapon1Attack"] {
			rng = math.Max(rng, ranges.weaponRange1)
		}
		if skillFlags["weapon2Attack"] {
			rng = math.Max(rng, ranges.weaponRange2)
		}
		skillData["radius"] = rng + 2
	}
	if radius, ok := sdNum(skillData, "radius"); ok {
		skillFlags["area"] = true
		radiusExtra, _ := sdNum(skillData, "radiusExtra")
		baseRadius := radius + radiusExtra + skillModList.Sum(mod.TypeBase, skillCfg, "AreaOfEffect")
		output["AreaOfEffectRadius"] = calcRadius(baseRadius, output["AreaOfEffectMod"], radiusMod)
		output["AreaOfEffectRadiusMetres"] = output["AreaOfEffectRadius"] / 10
		radiusSecondary, hasSecondary := sdNum(skillData, "radiusSecondary")
		if hasSecondary {
			incAreaSecondary, moreAreaSecondary := calcLibMods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectSecondary")
			output["AreaOfEffectModSecondary"] = calcAreaMod(incAreaSecondary, moreAreaSecondary)
			baseRadius = radiusSecondary + radiusExtra
			output["AreaOfEffectRadiusSecondary"] = calcRadius(baseRadius, output["AreaOfEffectModSecondary"], radiusMod)
			output["AreaOfEffectRadiusSecondaryMetres"] = output["AreaOfEffectRadiusSecondary"] / 10
		}
		if radiusTertiary, ok := sdNum(skillData, "radiusTertiary"); ok {
			incAreaTertiary, moreAreaTertiary := calcLibMods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectTertiary")
			output["AreaOfEffectModTertiary"] = calcAreaMod(incAreaTertiary, moreAreaTertiary)
			baseRadius = radiusTertiary + radiusExtra
			if margin, ok := sdNum(skillData, "radiusTertiaryBaseMargin"); truthy(skillData["projectileSpeedAppliesToMSAreaOfEffect"]) {
				incSpeedTertiary, moreSpeedTertiary := calcLibMods(skillModList, skillCfg, "ProjectileSpeed")
				output["SpeedModTertiary"] = round(round(incSpeedTertiary*moreSpeedTertiary, 10), 2)
				output["AreaOfEffectRadiusTertiary"] = calcMoltenStrikeTertiaryRadius(baseRadius, radiusSecondary, output["AreaOfEffectModTertiary"], output["SpeedModTertiary"])
			} else if ok {
				// Currently only on explosive trap in the form of "Smaller explosions have between 30% reduced and 30% increased base radius at random"
				marginFrac := margin / 100
				marginWidth := margin*2 + 1
				// Calculate all the possible base radii
				baseRadiiOccurrences := map[float64]float64{}
				for deviation := 1 - marginFrac; deviation <= 1+marginFrac+0.01; deviation += 0.01 {
					baseRadiiOccurrences[math.Floor(baseRadius*deviation)]++
				}
				// Calculate the modified radius for each base radius
				sumOfRandomRadii := 0.0
				for adjustedBaseRadius, occurrenceCount := range baseRadiiOccurrences {
					sumOfRandomRadii += calcRadius(adjustedBaseRadius, output["AreaOfEffectModTertiary"], radiusMod) * occurrenceCount
				}
				output["AreaOfEffectRadiusTertiary"] = sumOfRandomRadii / marginWidth
			} else {
				output["AreaOfEffectRadiusTertiary"] = calcRadius(baseRadius, output["AreaOfEffectModTertiary"], radiusMod)
			}
			output["AreaOfEffectRadiusTertiaryMetres"] = output["AreaOfEffectRadiusTertiary"] / 10
		}
	}
}
