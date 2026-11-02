package calculator2

import (
	"math"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

//pob2:block:src/Modules/CalcOffence.lua:3341-3347
func getLeechInstances(amount, total, hitRate float64) (float64, float64) {
	if total == 0 {
		return 0, 0
	}
	duration := amount / total / miscConstant("LeechRateBase")
	return duration, duration * hitRate
}

// calcAncestralBoost is the dynamic way of calculating the Ancestral Boost from a single source
// without duplicating the code. Its output is calcs.offence's own, not the pass's: the closure is
// defined before the pass loop declares its local.
//
//pob2:block:src/Modules/CalcOffence.lua:3351-3383
func (h *offenceHit) calcAncestralBoost(skillName string, moreDmg float64, uptimeOverride float64, hasUptimeOverride bool) {
	h.actor.setOutputFlag("CreateWarcryOffensiveCalcSection", true) // labels for the CalcSection
	globalOutput := h.actor.Output
	skillNameVar := strings.ReplaceAll(skillName, " ", "") // Fist Of War -> FistOfWar

	globalOutput[skillNameVar+"DamageMultiplier"] = moreDmg
	if hasUptimeOverride {
		globalOutput[skillNameVar+"UptimeRatio"] = uptimeOverride
	} else {
		globalOutput[skillNameVar+"UptimeRatio"] = math.Min((1/globalOutput["Speed"])/globalOutput[skillNameVar+"Cooldown"], 1) * 100
	}
	globalOutput["Avg"+skillNameVar+"Damage"] = globalOutput[skillNameVar+"DamageMultiplier"]
	globalOutput["Avg"+skillNameVar+"DamageEffect"] = 1 + globalOutput["Avg"+skillNameVar+"Damage"]*(globalOutput[skillNameVar+"UptimeRatio"]/100)
	globalOutput["Max"+skillNameVar+"DamageEffect"] = 1 + globalOutput[skillNameVar+"DamageMultiplier"]
	if h.activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
		globalOutput[skillNameVar+"DamageEffect"] = globalOutput["Max"+skillNameVar+"DamageEffect"]
	} else {
		globalOutput[skillNameVar+"DamageEffect"] = globalOutput["Avg"+skillNameVar+"DamageEffect"]
	}
	h.calcAreaOfEffect()
	globalOutput["TheoreticalOffensiveWarcryEffect"] *= globalOutput["Avg"+skillNameVar+"DamageEffect"]
	globalOutput["TheoreticalMaxOffensiveWarcryEffect"] *= globalOutput["Max"+skillNameVar+"DamageEffect"]
}

// combine Ancestral Empowerment with other sources of Slam Ancestral Boost, namely Fist of War, when both active
//
//pob2:block:src/Modules/CalcOffence.lua:3386-3429
func (h *offenceHit) calcCombinedAncestralBoost(skillName string, moreDmg float64, uptimeOverride float64, additionalSkillName string) {
	h.actor.setOutputFlag("CreateWarcryOffensiveCalcSection", true) // labels for the CalcSection
	globalOutput := h.actor.Output
	skillNameVar := strings.ReplaceAll(skillName, " ", "") // Fist Of War -> FistOfWar

	globalOutput[skillNameVar+"DamageMultiplier"] = moreDmg
	// for CalcSections, set the AncestralEmpowerment damage for mod breakdown
	globalOutput[skillNameVar+"CombinedDamageMultiplier"] = globalOutput[skillNameVar+"DamageMultiplier"]
	skillNameVar += "Combined"
	additionalSkillNameVar := strings.ReplaceAll(additionalSkillName, " ", "")

	globalOutput[skillNameVar+"UptimeRatio"] = uptimeOverride
	globalOutput[skillNameVar+"UptimeRatio"] = math.Min(globalOutput[skillNameVar+"UptimeRatio"]+globalOutput[additionalSkillNameVar+"UptimeRatio"], 100)
	globalOutput["Avg"+skillNameVar+"Damage"] = globalOutput[skillNameVar+"DamageMultiplier"]
	globalOutput["Avg"+skillNameVar+"DamageEffect"] = 1 + globalOutput["Avg"+skillNameVar+"Damage"]*(globalOutput[skillNameVar+"UptimeRatio"]/100)
	globalOutput["Max"+skillNameVar+"DamageEffect"] = 1 + globalOutput[skillNameVar+"DamageMultiplier"]
	if h.activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
		globalOutput[skillNameVar+"DamageEffect"] = globalOutput["Max"+skillNameVar+"DamageEffect"]
	} else {
		globalOutput[skillNameVar+"DamageEffect"] = globalOutput["Avg"+skillNameVar+"DamageEffect"]
	}
	h.calcAreaOfEffect()
	globalOutput["TheoreticalOffensiveWarcryEffect"] *= globalOutput["Avg"+skillNameVar+"DamageEffect"]
	globalOutput["TheoreticalMaxOffensiveWarcryEffect"] *= globalOutput["Max"+skillNameVar+"DamageEffect"]
}

// offenceHit is calcs.offence's state across the hit half.
type offenceHit struct {
	env         *Environment
	actor       *Actor
	activeSkill *ActiveSkill
	skillFlags  map[string]bool
	state       *offenceHitState
	monsterLife float64
	// calculated holds the booleans calcs.offence keeps on its output (the *Calculated marks).
	calculated map[string]bool
}

func (h *offenceHit) calcAreaOfEffect() {
	as := h.activeSkill
	calcAreaOfEffect(h.actor, as.SkillModList, as.SkillCfg, as.SkillData, h.skillFlags, h.actor.Output)
}

// Check if player is supposed to ignore a damage type, or if it's ignored on enemy side
//
//pob2:block:src/Modules/CalcOffence.lua:3431-3433
func (h *offenceHit) useThisResist(cfg *moddb.ListCfg, damageType string) bool {
	names := []string{"Ignore" + damageType + "Resistance"}
	if isElemental[damageType] {
		names = append(names, "IgnoreElementalResistances")
	}
	return !h.activeSkill.SkillModList.Flag(cfg, names...) && !h.actor.Enemy.ModDB.Flag(nil, "SelfIgnore"+damageType+"Resistance")
}

// offenceWarcryExerts iterates over all the active skills to account for exerted attacks provided
// by warcries. The breakdowns are not ported.
//
//pob2:block:src/Modules/CalcOffence.lua:3445-3667
func (h *offenceHit) offenceWarcryExerts(cfg *moddb.ListCfg) {
	env, actor, activeSkill := h.env, h.actor, h.activeSkill
	globalOutput := actor.Output
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	skillModList, skillData := activeSkill.SkillModList, activeSkill.SkillData

	// Exerted Attack members
	exertedDoubleDamage := env.ModDB.Sum(mod.TypeBase, cfg, "ExertDoubleDamageChance")
	globalOutput["OffensiveWarcryEffect"] = 1
	globalOutput["MaxOffensiveWarcryEffect"] = 1
	globalOutput["TheoreticalOffensiveWarcryEffect"] = 1
	globalOutput["TheoreticalMaxOffensiveWarcryEffect"] = 1
	globalOutput["RallyingHitEffect"] = 1
	globalOutput["AilmentWarcryEffect"] = 1
	globalOutput["GlobalWarcryUptimeRatio"] = 0

	if !env.ModeBuffs {
		return
	}
	st := activeSkill.SkillTypes
	if st["NeverExertable"] || st["Triggered"] || st["OtherThingUsesSkill"] || st["Retaliation"] {
		return
	}
	calculated := func(name string) bool {
		return h.calculated[name]
	}
	// warcryTimes sets <prefix>Duration, Cooldown and CastTime and returns the exert uptime ratio.
	warcryTimes := func(value *ActiveSkill, prefix, countKey, countMod string) float64 {
		globalOutput[prefix+"Duration"] = calcSkillDuration(value.SkillModList, value.SkillCfg, value.SkillData, env, enemyDB)
		globalOutput[prefix+"Cooldown"], _, _, _, _ = calcSkillCooldown(value.SkillModList, value.SkillCfg, value.SkillData)
		globalOutput[prefix+"CastTime"] = calcWarcryCastTime(value.SkillModList, value.SkillCfg, value.SkillData, actor)
		if countKey == "" {
			return 0
		}
		globalOutput[countKey] = env.ModDB.Sum(mod.TypeBase, nil, countMod)
		baseUptimeRatio := math.Min((globalOutput[countKey]/globalOutput["Speed"])/(globalOutput[prefix+"Cooldown"]+globalOutput[prefix+"CastTime"]), 1) * 100
		storedUses, ok := sdNum(value.SkillData, "storedUses")
		if !ok {
			storedUses = 0 + value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "AdditionalCooldownUses")
		}
		return math.Min(100, baseUptimeRatio*storedUses)
	}
	for _, value := range actor.ActiveSkillList {
		if value == nil || value.ActiveEffect == nil || value.ActiveEffect.GrantedEffect == nil || value.SkillModList == nil {
			continue
		}
		switch name := value.ActiveEffect.GrantedEffect.Name; {
		case name == "Ancestral Cry" && st["MeleeSingleTarget"] && !calculated("AncestralCryCalculated"):
			globalOutput["AncestralUpTimeRatio"] = warcryTimes(value, "AncestralCry", "AncestralExertsCount", "NumAncestralExerts")
			globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["AncestralUpTimeRatio"]
			h.calculated["AncestralCryCalculated"] = true
			h.actor.setOutputFlag("AncestralCryCalculated", true)
		case name == "Infernal Cry" && st["Melee"] && !calculated("InfernalCryCalculated"):
			h.actor.setOutputFlag("CreateWarcryOffensiveCalcSection", true)
			globalOutput["InfernalCryUptimeRatio"] = warcryTimes(value, "InfernalCry", "InfernalEmpoweredCount", "NumInfernalEmpowers")
			globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["InfernalCryUptimeRatio"]
			infernalGainAsFire := modDB.Sum(mod.TypeBase, nil, "InfernalExtraFireDamage")
			if infernalGainAsFire > 0 {
				infernalUptime := globalOutput["InfernalCryUptimeRatio"]
				if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
					infernalUptime = 100
				}
				skillModList.AddMod(newMod("DamageGainAsFire", mod.TypeBase, infernalGainAsFire*infernalUptime/100, "Uptime Scaled Infernal Cry").Flag(ModFlagMelee))
			}
			h.calculated["InfernalCryCalculated"] = true
			h.actor.setOutputFlag("InfernalCryCalculated", true)
		case name == "Intimidating Cry" && st["Melee"] && !calculated("IntimidatingCryCalculated"):
			h.actor.setOutputFlag("CreateWarcryOffensiveCalcSection", true)
			globalOutput["IntimidatingUpTimeRatio"] = warcryTimes(value, "IntimidatingCry", "IntimidatingExertsCount", "NumIntimidatingExerts")
			globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["IntimidatingUpTimeRatio"]
			selfDouble := 0.0
			if env.ModeEffective {
				selfDouble = enemyDB.Sum(mod.TypeBase, cfg, "SelfDoubleDamageChance")
			}
			ddChance := math.Min(skillModList.Sum(mod.TypeBase, cfg, "DoubleDamageChance")+selfDouble+exertedDoubleDamage, 100)
			globalOutput["IntimidatingAvgDmg"] = 2 * (1 - ddChance/100)
			globalOutput["IntimidatingHitEffect"] = 1 + globalOutput["IntimidatingAvgDmg"]*globalOutput["IntimidatingUpTimeRatio"]/100
			globalOutput["IntimidatingMaxHitEffect"] = 1 + globalOutput["IntimidatingAvgDmg"]
			globalOutput["TheoreticalOffensiveWarcryEffect"] *= globalOutput["IntimidatingHitEffect"]
			globalOutput["TheoreticalMaxOffensiveWarcryEffect"] *= globalOutput["IntimidatingMaxHitEffect"]
			h.calculated["IntimidatingCryCalculated"] = true
			h.actor.setOutputFlag("IntimidatingCryCalculated", true)
		case name == "Rallying Cry" && st["Melee"] && !calculated("RallyingCryCalculated"):
			h.actor.setOutputFlag("CreateWarcryOffensiveCalcSection", true)
			globalOutput["RallyingUpTimeRatio"] = warcryTimes(value, "RallyingCry", "RallyingExertsCount", "NumRallyingExerts")
			globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["RallyingUpTimeRatio"]
			globalOutput["RallyingAvgDmg"] = math.Min(env.ModDB.Sum(mod.TypeBase, cfg, "Multiplier:NearbyAlly"), 5) * (env.ModDB.Sum(mod.TypeBase, nil, "RallyingExertMoreDamagePerAlly") / 100)
			globalOutput["RallyingHitEffect"] = 1 + globalOutput["RallyingAvgDmg"]*globalOutput["RallyingUpTimeRatio"]/100
			globalOutput["RallyingMaxHitEffect"] = 1 + globalOutput["RallyingAvgDmg"]
			globalOutput["OffensiveWarcryEffect"] *= globalOutput["RallyingHitEffect"]
			globalOutput["MaxOffensiveWarcryEffect"] *= globalOutput["RallyingMaxHitEffect"]
			globalOutput["TheoreticalOffensiveWarcryEffect"] *= globalOutput["RallyingHitEffect"]
			globalOutput["TheoreticalMaxOffensiveWarcryEffect"] *= globalOutput["RallyingMaxHitEffect"]
			h.calculated["RallyingCryCalculated"] = true
			h.actor.setOutputFlag("RallyingCryCalculated", true)
		case name == "Seismic Cry" && st["Slam"] && !calculated("SeismicCryCalculated"):
			h.actor.setOutputFlag("CreateWarcryOffensiveCalcSection", true)
			globalOutput["SeismicUpTimeRatio"] = warcryTimes(value, "SeismicCry", "SeismicExertsCount", "NumSeismicExerts")
			globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["SeismicUpTimeRatio"]
			// account for AoE increase
			if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
				skillModList.NewMod("AreaOfEffect", mod.TypeMore, env.ModDB.Sum(mod.TypeBase, nil, "SeismicMoreAoE"), "Max Seismic Exert AoE")
			} else {
				skillModList.NewMod("AreaOfEffect", mod.TypeMore, math.Floor(env.ModDB.Sum(mod.TypeBase, nil, "SeismicMoreAoE")/100*globalOutput["SeismicUpTimeRatio"]), "Avg Seismic Exert AoE")
			}
			h.calcAreaOfEffect()
			h.calculated["SeismicCryCalculated"] = true
			h.actor.setOutputFlag("SeismicCryCalculated", true)
		case name == "Battlemage's Cry" && !calculated("BattleMageCryCalculated"):
			if st["Melee"] {
				globalOutput["BattlemageUpTimeRatio"] = warcryTimes(value, "BattleMageCry", "BattleCryExertsCount", "NumBattlemageExerts")
				globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["BattlemageUpTimeRatio"]
			} else {
				warcryTimes(value, "BattleMageCry", "", "")
			}
			h.calculated["BattleMageCryCalculated"] = true
			h.actor.setOutputFlag("BattleMageCryCalculated", true)
		}
	}

	if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
		globalOutput["AilmentWarcryEffect"] = globalOutput["MaxOffensiveWarcryEffect"]
		skillData["showAverage"] = true
		h.skillFlags["showAverage"] = true
		h.skillFlags["notAverage"] = false
	} else {
		globalOutput["AilmentWarcryEffect"] = globalOutput["OffensiveWarcryEffect"]
	}
}

// offenceAncestralBoosts handles Ruthless, Ancestral Boost and Empowered attack uptimes.
//
//pob2:block:src/Modules/CalcOffence.lua:3669-3808
func (h *offenceHit) offenceAncestralBoosts(pass *offencePass) {
	env, activeSkill := h.env, h.activeSkill
	globalOutput := h.actor.Output
	output := pass.output
	cfg := pass.cfg
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	st := activeSkill.SkillTypes

	output["FistOfWarDamageEffect"] = 1
	output["AncestralCallDamageEffect"] = 1
	output["AncestralEmpowermentDamageEffect"] = 1
	output["AncestralEmpowermentCombinedDamageEffect"] = 1
	if env.ModeCombat {
		ruthlessEffect, ok := configListValue(env, "ruthlessSupportMode")
		if !ok {
			ruthlessEffect = "AVERAGE"
		}
		ruthlessBlowMaxCount := skillModList.Sum(mod.TypeBase, cfg, "RuthlessBlowMaxCount")
		ruthlessBlowChance := 0.0
		mirageUses, _ := sdNum(skillData, "mirageUses")
		if ruthlessBlowMaxCount > 0 && (!skillCfg.SkillCond["usedByMirage"] || mirageUses > ruthlessBlowMaxCount) {
			switch ruthlessEffect {
			case "AVERAGE":
				ruthlessBlowChance = round(100 / ruthlessBlowMaxCount)
			case "MAX":
				ruthlessBlowChance = 100
				skillModList.NewMod("DPS", mod.TypeMore, -100+100/ruthlessBlowMaxCount, "Only Ruthless Blows")
			}
		}
		ruthlessBlowStunMultiplier := skillModList.Sum(mod.TypeBase, cfg, "RuthlessBlowStunMultiplier") / 100
		ruthlessBlowStunEffect := (ruthlessBlowChance / 100) * ruthlessBlowStunMultiplier
		skillModList.NewMod("EnemyHeavyStunBuildup", mod.TypeMore, ruthlessBlowStunEffect*100, "Ruthless Blows")

		// Condition:AncestrallyBoosted * AncestralBoostEffect (e.g. Fist of War III)
		ancestrallyBoostedMoreDamageMulti := skillModList.Sum(mod.TypeBase, cfg, "AncestralBoostMoreDamage") / 100

		globalOutput["FistOfWarCooldown"] = skillModList.Sum(mod.TypeBase, cfg, "FistOfWarCooldown")
		if skillModList.Flag(cfg, "AncestralEmpowerment") && st["Slam"] && !st["Vaal"] && !st["OtherThingUsesSkill"] {
			if globalOutput["FistOfWarCooldown"] != 0 { // get the fist of war calcs in output to use in Empowerment
				h.calcAncestralBoost("Fist Of War", ancestrallyBoostedMoreDamageMulti, 0, false)
				globalOutput["TheoreticalOffensiveWarcryEffect"] = 1 // reset effects from FistOfWar calc, we combine later
				globalOutput["TheoreticalMaxOffensiveWarcryEffect"] = 1

				h.calcCombinedAncestralBoost("Ancestral Empowerment", ancestrallyBoostedMoreDamageMulti, 50, "Fist Of War")
				delete(globalOutput, "FistOfWarUptimeRatio") // hide from CalcSections, but we need it for the combined calc first
			} else {
				h.calcAncestralBoost("Ancestral Empowerment", ancestrallyBoostedMoreDamageMulti, 50, true)
			}
		}
		// If Fist of War & Active Skill is a Slam Skill & NOT a Vaal Skill & NOT used by mirage or other
		if !skillModList.Flag(cfg, "AncestralEmpowerment") && globalOutput["FistOfWarCooldown"] != 0 && st["Slam"] && !st["Vaal"] && !st["OtherThingUsesSkill"] {
			h.calcAncestralBoost("Fist Of War", ancestrallyBoostedMoreDamageMulti, 0, false)
		} else {
			output["FistOfWarDamageEffect"] = 1
		}

		globalOutput["AncestralCallCooldown"] = skillModList.Sum(mod.TypeBase, cfg, "AncestralCallCooldown")
		// If Ancestral Call & Active Skill is NOT a Vaal Skill & NOT used by mirage or other & NOT a Channel Skill
		if globalOutput["AncestralCallCooldown"] != 0 && !st["Vaal"] && !st["OtherThingUsesSkill"] && !st["Channel"] {
			h.calcAncestralBoost("Ancestral Call", ancestrallyBoostedMoreDamageMulti, 0, false)
		} else {
			output["AncestralCallDamageEffect"] = 1
		}
	}

	// Calculate Empowered Attack Uptime
	// Ancestrally Boosted attacks are also Empowered, so we need to check for the highest uptime among both lists
	// we can use these ratio flags to max the uptime for attacks/skills that guarantee Ancestrally Boosted or Empowered in specific cases like Crescendo III
	maxEmpoweredUptimeRatio := skillModList.Flag(cfg, "MaxBoostedUptimeRatio") || skillModList.Flag(cfg, "MaxEmpoweredUptimeRatio")
	ancestralBoostList := []string{"AncestralEmpowermentCombinedUptimeRatio", "AncestralEmpowermentUptimeRatio", "FistOfWarUptimeRatio", "AncestralCallUptimeRatio"}
	combinedList := append([]string{"InfernalCryUptimeRatio"}, ancestralBoostList...)

	for _, cryTimeRatio := range combinedList {
		if maxEmpoweredUptimeRatio {
			globalOutput["ExertedAttackUptimeRatio"] = 100
		} else {
			globalOutput["ExertedAttackUptimeRatio"] = math.Max(globalOutput["ExertedAttackUptimeRatio"], globalOutput[cryTimeRatio])
		}
	}
	if globalOutput["ExertedAttackUptimeRatio"] > 0 && !h.calculated["ExertedAttackUptimeRatioCalculated"] {
		incExertedAttacks := skillModList.Sum(mod.TypeIncrease, cfg, "EmpoweredIncrease")
		if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
			skillModList.AddMod(newMod("Damage", mod.TypeIncrease, incExertedAttacks, "Empowered Attacks").Flag(ModFlagAttack))
		} else {
			skillModList.AddMod(newMod("Damage", mod.TypeIncrease, incExertedAttacks*globalOutput["ExertedAttackUptimeRatio"]/100, "Uptime Scaled Empowered Attacks").Flag(ModFlagAttack))
		}
		globalOutput["ExertedAttackAvgDmg"] = calcLibMod(skillModList, skillCfg, "EmpoweredIncrease")
		globalOutput["ExertedAttackHitEffect"] = globalOutput["ExertedAttackAvgDmg"] * globalOutput["ExertedAttackUptimeRatio"] / 100
		globalOutput["ExertedAttackMaxHitEffect"] = globalOutput["ExertedAttackAvgDmg"]
		h.calculated["ExertedAttackUptimeRatioCalculated"] = true
		h.actor.setOutputFlag("ExertedAttackUptimeRatioCalculated", true)
	}

	maxBoostedUptimeRatio := skillModList.Flag(cfg, "MaxBoostedUptimeRatio")
	// Calculate Ancestrally Boosted Attack Uptime
	for _, boostUptimeRatio := range ancestralBoostList {
		if maxBoostedUptimeRatio {
			globalOutput["BoostedAttackUptimeRatio"] = 100
		} else {
			globalOutput["BoostedAttackUptimeRatio"] = math.Max(globalOutput["BoostedAttackUptimeRatio"], globalOutput[boostUptimeRatio])
		}
	}
	if globalOutput["BoostedAttackUptimeRatio"] > 0 && !h.calculated["BoostedAttackUptimeRatioCalculated"] {
		incBoostedAttacks := skillModList.Sum(mod.TypeIncrease, cfg, "AncestralBoostDamage")
		incAoEBoostedAttacks := skillModList.Sum(mod.TypeIncrease, cfg, "AncestralBoostAreaOfEffect")
		if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
			skillModList.AddMod(newMod("Damage", mod.TypeIncrease, incBoostedAttacks, "Ancestrally Boosted Attacks").Flag(ModFlagAttack))
			skillModList.NewMod("AreaOfEffect", mod.TypeIncrease, incAoEBoostedAttacks, "Ancestrally Boosted Attacks")
		} else {
			skillModList.AddMod(newMod("Damage", mod.TypeIncrease, incBoostedAttacks*globalOutput["BoostedAttackUptimeRatio"]/100, "Uptime Scaled Ancestrally Boosted Attacks").Flag(ModFlagAttack))
			skillModList.NewMod("AreaOfEffect", mod.TypeIncrease, incAoEBoostedAttacks*globalOutput["BoostedAttackUptimeRatio"]/100, "Uptime Scaled Ancestrally Boosted AoE")
		}
		h.calcAreaOfEffect()

		globalOutput["BoostedAttackAvgDmg"] = calcLibMod(skillModList, skillCfg, "AncestralBoostDamage")
		globalOutput["BoostedAttackHitEffect"] = globalOutput["BoostedAttackAvgDmg"] * globalOutput["BoostedAttackUptimeRatio"] / 100
		globalOutput["BoostedAttackMaxHitEffect"] = globalOutput["BoostedAttackAvgDmg"]
		h.calculated["BoostedAttackUptimeRatioCalculated"] = true
		h.actor.setOutputFlag("BoostedAttackUptimeRatioCalculated", true)
	}
}

// offenceCrit calculates crit chance, crit multiplier, and their combined effect. The breakdowns
// are not ported.
//
//pob2:block:src/Modules/CalcOffence.lua:3816-4040
func (h *offenceHit) offenceCrit(pass *offencePass) {
	env, actor, activeSkill := h.env, h.actor, h.activeSkill
	enemyDB := actor.Enemy.ModDB
	output, cfg, source := pass.output, pass.cfg, pass.source
	skillModList, skillCfg := activeSkill.SkillModList, activeSkill.SkillCfg

	inevitableCrits := false
	repeats, ok := output["Repeats"]
	if !ok {
		repeats = 1
	}
	switch {
	case skillModList.Flag(cfg, "NeverCrit"):
		output["PreEffectiveCritChance"] = 0
		output["CritChance"] = 0
		output["CritMultiplier"] = 0
		output["BonusCritDotMultiplier"] = 0
		output["CritEffect"] = 1
	case skillModList.Flag(cfg, "SpellSkillsCannotDealCriticalStrikesExceptOnFinalRepeat"):
		if repeats == 1 {
			output["PreEffectiveCritChance"] = 0
			output["CritChance"] = 0
			output["CritMultiplier"] = 0
			output["BonusCritDotMultiplier"] = 0
			output["CritEffect"] = 1
		} else if skillModList.Flag(cfg, "SpellSkillsAlwaysDealCriticalStrikesOnFinalRepeat") {
			repeatMode, _ := configListValue(env, "repeatMode")
			switch repeatMode {
			case "None":
				output["PreEffectiveCritChance"] = 0
				output["CritChance"] = 0
			case "AVERAGE":
				output["PreEffectiveCritChance"] = 100 / output["Repeats"]
				output["CritChance"] = 100 / output["Repeats"]
			default:
				output["PreEffectiveCritChance"] = 100
				output["CritChance"] = 100
			}
		}
	default:
		var critOverride float64
		hasCritOverride := false
		if o := skillModList.Override(cfg, "CritChance"); o != nil {
			critOverride, hasCritOverride = o.Float(), true
		}
		// destructive link
		// actor.parent.output.MainHand and actor.partyMembers are not carried; the parent's weapon
		// data stands in for both.
		if skillModList.Flag(cfg, "MainHandCritIsEqualToParent") && actor.Parent != nil {
			critOverride, hasCritOverride = weaponData(env, actor.Parent, 1)["CritChance"], true
		} else if skillModList.Flag(cfg, "MainHandCritIsEqualToPartyMember") {
			critOverride, hasCritOverride = 0, true
		}
		baseCrit := source.num("CritChance")
		if hasCritOverride {
			baseCrit = critOverride
		}

		baseCritFromMainHand := skillModList.Flag(cfg, "BaseCritFromMainHand")
		baseCritFromParentMainHand := skillModList.Flag(cfg, "AttackCritIsEqualToParentMainHand")
		if o := skillModList.Override(cfg, "CritChanceBase"); o != nil {
			baseCrit = o.Float()
		} else if baseCritFromMainHand {
			baseCrit = weaponData(env, actor, 1)["CritChance"]
		} else if baseCritFromParentMainHand && actor.Parent != nil {
			if v, ok := weaponData(env, actor.Parent, 1)["CritChance"]; ok {
				baseCrit = v
			}
		}

		if hasCritOverride && critOverride == 100 {
			output["PreEffectiveCritChance"] = 100
			output["PreBifurcateCritChance"] = 100
			output["CritChance"] = 100
		} else {
			base := 0.0
			inc := 0.0
			more := 1.0
			if !hasCritOverride {
				selfBase, selfInc := 0.0, 0.0
				if env.ModeEffective {
					selfBase = enemyDB.Sum(mod.TypeBase, nil, "SelfCritChance")
					selfInc = enemyDB.Sum(mod.TypeIncrease, nil, "SelfCritChance")
				}
				base = skillModList.Sum(mod.TypeBase, cfg, "CritChance") + selfBase
				inc = skillModList.Sum(mod.TypeIncrease, cfg, "CritChance") + selfInc
				more = skillModList.More(cfg, "CritChance")
			}
			// The game uses an integer permyriad chance for each critical strike roll.
			output["CritChance"] = round((baseCrit+base)*(1+inc/100)*more*100) / 100
			critCap := skillModList.Sum(mod.TypeBase, cfg, "CritChanceCap")
			if o := skillModList.Override(nil, "CritChanceCap"); o != nil {
				critCap = o.Float()
			}
			output["CritChance"] = math.Min(output["CritChance"], critCap)
			if (baseCrit + base) > 0 {
				output["CritChance"] = math.Max(output["CritChance"], 0)
			}
			output["PreEffectiveCritChance"] = output["CritChance"]
			if env.ModeEffective {
				// Hits that make an accuracy check make a second accuracy check if they roll a crit. If that fails, the crit is nullified and it just stays a normal hit
				// https://www.pathofexile.com/forum/view-thread/11707/filter-account-type/staff/page/10#p748465
				output["CritChance"] = output["CritChance"] * output["AccuracyHitChance"] / 100
			}
			if env.ModeEffective && skillModList.Flag(cfg, "CritChanceLucky") {
				output["CritChance"] = (1 - math.Pow(1-output["CritChance"]/100, 2)) * 100
			}
			output["PreBifurcateCritChance"] = output["CritChance"]
			if env.ModeEffective && skillModList.Flag(cfg, "BifurcateCrit") {
				output["CritChance"] = (1 - math.Pow(1-output["CritChance"]/100, 2)) * 100
			}
			if env.ModeEffective && skillModList.Flag(cfg, "InevitableCriticalHits") && output["CritChance"] > 0 {
				inevitableCrits = true
				// Lucky crits use their effective crit chance without an extra roll-down penalty.
				// Bifurcated crits roll twice per roll-down; only both rolls failing advances the penalty.
				critChance := output["CritChance"] / 100
				nonCritChance := 1 - critChance

				critBonusMultiplier := 1*critChance + // 100% crit damage, crit% of the time
					0.7*nonCritChance*critChance + // 70% if we roll non-crit then a crit
					0.4*math.Pow(nonCritChance, 2)*critChance + // 40% if we roll two non-crit then a crit
					0.1*math.Pow(nonCritChance, 3)*critChance // 10% if we roll three non-crits then a crit
				if skillModList.Flag(cfg, "BifurcateCrit") {
					// The terminating stage can crit twice, so scale its one-or-more-crit bonus
					// by the expected number of crits on that stage.
					critBonusMultiplier = critBonusMultiplier * 2 * output["PreBifurcateCritChance"] / output["CritChance"]
					output["CritBifurcates"] = math.Pow(output["PreBifurcateCritChance"], 2) / output["CritChance"]
				}

				// This gets rounded when used in damage logic, so round it ahead of time to make the breakdown accurate (and less ugly)
				lessCritBonus := round((1-critBonusMultiplier)*-100.0, 0)
				skillModList.NewMod("CritMultiplier", mod.TypeMore, lessCritBonus, "Tree:55135")

				// For the sake of any logic that depends on it, every hit is considered a crit
				output["CritChance"] = 100
			}
		}
	}
	if _, ok := output["CritEffect"]; !ok {
		if skillModList.Flag(cfg, "NoCritMultiplier") {
			output["CritMultiplier"] = 1
		} else {
			extraDamage := skillModList.Sum(mod.TypeBase, cfg, "CritMultiplier") / 100
			extraDamageInc := 1 + skillModList.Sum(mod.TypeIncrease, cfg, "CritMultiplier")/100
			extraDamageMore := skillModList.More(cfg, "CritMultiplier")
			extraDamage = extraDamage * extraDamageInc * extraDamageMore
			if multiOverride := skillModList.Override(skillCfg, "CritMultiplier"); multiOverride != nil {
				extraDamage = multiOverride.Float() / 100
			}

			output["PreEffectiveCritMultiplier"] = 1 + extraDamage
			// if crit bifurcates are enabled, roll for crit twice and add multiplier for each
			if env.ModeEffective && skillModList.Flag(cfg, "BifurcateCrit") && !inevitableCrits {
				// get crit chance and calculate odds of critting twice
				critChancePercentage := output["PreBifurcateCritChance"]
				bifurcateMultiChance := math.Pow(critChancePercentage, 2) / 100
				effectiveCritChance := output["CritChance"]
				conditionalBifurcateChance := 0.0
				if effectiveCritChance > 0 {
					conditionalBifurcateChance = bifurcateMultiChance / effectiveCritChance
				}
				output["CritBifurcates"] = 1 + conditionalBifurcateChance
				damageBonus := extraDamage
				bifurcatedBonus := conditionalBifurcateChance * extraDamage
				extraDamage = damageBonus + bifurcatedBonus
				skillModList.NewMod("CritMultiplier", mod.TypeMore, floorDec(conditionalBifurcateChance*100, 2), "Bifurcated Crit Damage Bonus")
			}

			if env.ModeEffective {
				enemyInc := 1 + enemyDB.Sum(mod.TypeIncrease, nil, "SelfCritMultiplier")/100
				extraDamage += enemyDB.Sum(mod.TypeBase, nil, "SelfCritMultiplier") / 100
				extraDamage = round(extraDamage*enemyInc, 2)
			}
			output["CritMultiplier"] = 1 + math.Max(0, extraDamage)
		}
		critChancePercentage := output["CritChance"] / 100
		output["CritEffect"] = 1 - critChancePercentage + critChancePercentage*output["CritMultiplier"]
		output["BonusCritDotMultiplier"] = (skillModList.Sum(mod.TypeBase, cfg, "CritMultiplier") - 50) * skillModList.Sum(mod.TypeBase, cfg, "CritMultiplierAppliesToDegen") / 10000
	}
}

// offenceDamagePass calculates damage (exerts, crits, ruthless, DPS, etc) for one pass. The
// breakdowns are not ported.
//
//pob2:block:src/Modules/CalcOffence.lua:3435-4719
func (h *offenceHit) offenceDamagePass(pass *offencePass, quantityMultiplier float64) {
	env, actor, activeSkill, state, skillFlags := h.env, h.actor, h.activeSkill, h.state, h.skillFlags
	globalOutput := actor.Output
	enemyDB := actor.Enemy.ModDB
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	source, output, cfg := pass.source, pass.output, pass.cfg

	h.offenceWarcryExerts(cfg)
	h.offenceAncestralBoosts(pass)

	// Calculate maximum sustainable fuses and explosion rate for Explosive Arrow
	// Explosive Arrow's explosiveArrowFunc (CalcOffence.lua:3812-3814) is not ported.

	h.offenceCrit(pass)

	//pob2:block:src/Modules/CalcOffence.lua:4042-4065
	output["ScaledDamageEffect"] = 1

	// Calculate chance and multiplier for dealing triple damage on Normal and Crit
	output["TripleDamageChanceOnCrit"] = math.Min(skillModList.Sum(mod.TypeBase, cfg, "TripleDamageChanceOnCrit"), 100)
	// `Sum(...) or 0 + ...` is `Sum(...) or (0 + ...)`, and Sum is never nil.
	output["TripleDamageChance"] = math.Min(skillModList.Sum(mod.TypeBase, cfg, "TripleDamageChance"), 100)
	output["TripleDamageEffect"] = 2 * output["TripleDamageChance"] / 100

	// Calculate chance and multiplier for dealing double damage on Normal and Crit
	output["DoubleDamageChanceOnCrit"] = math.Min(skillModList.Sum(mod.TypeBase, cfg, "DoubleDamageChanceOnCrit"), 100)
	selfDouble := 0.0
	if env.ModeEffective {
		selfDouble = enemyDB.Sum(mod.TypeBase, cfg, "SelfDoubleDamageChance")
	}
	output["DoubleDamageChance"] = math.Min(skillModList.Sum(mod.TypeBase, cfg, "DoubleDamageChance")+selfDouble+(output["DoubleDamageChanceOnCrit"]*output["CritChance"]/100), 100)
	if intimidating, ok := globalOutput["IntimidatingUpTimeRatio"]; ok && activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
		output["DoubleDamageChance"] = 100
	} else if ok {
		output["DoubleDamageChance"] = math.Min(output["DoubleDamageChance"]+intimidating, 100)
	}
	// Triple Damage overrides Double Damage. If you have both, it's the same as just having Triple
	// We need to subtract the probability of both happening in favor of Triple Damage
	if output["TripleDamageChance"] > 0 {
		output["DoubleDamageChance"] = math.Max(output["DoubleDamageChance"]-output["TripleDamageChance"]*output["DoubleDamageChance"]/100, 0)
	}
	output["DoubleDamageEffect"] = output["DoubleDamageChance"] / 100
	output["ScaledDamageEffect"] *= 1 + output["DoubleDamageEffect"] + output["TripleDamageEffect"]

	dpsMultiplier, ok := sdNum(skillData, "dpsMultiplier")
	if !ok {
		dpsMultiplier = 1
	}
	output["DpsMultiplier"] = dpsMultiplier * calcLibMod(skillModList, skillCfg, "DPS")

	hitSpeed, ok := globalOutput["HitSpeed"]
	if !ok {
		hitSpeed = globalOutput["Speed"]
	}
	hitRate := output["HitChance"] / 100 * hitSpeed * output["DpsMultiplier"]

	//pob2:block:src/Modules/CalcOffence.lua:4069-4100
	enemyRarity := "Normal"
	if enemyDB.Flag(nil, "Condition:Unique") {
		enemyRarity = "Unique"
	} else if enemyDB.Flag(nil, "Condition:RareOrUnique") {
		enemyRarity = "Rare"
	}
	// Calculate culling DPS
	criticalCull := 0.0
	if o := skillModList.Override(cfg, "CriticalCullPercent"); o != nil {
		criticalCull = o.Float()
	} else if skillModList.Flag(cfg, "CritCanCull") {
		criticalCull = gameConstant("CullingStrike" + enemyRarity + "Threshold")
	}
	if criticalCull > 0 {
		criticalCull = math.Min(criticalCull, criticalCull*(1-math.Pow(1-output["CritChance"]/100, hitRate)))
	}
	regularCull := 0.0
	if o := skillModList.Override(cfg, "CullPercent"); o != nil {
		regularCull = o.Float()
	} else if skillModList.Flag(cfg, "CanCull") {
		regularCull = gameConstant("CullingStrike" + enemyRarity + "Threshold")
	}
	incCullPercent := 1 + skillModList.Sum(mod.TypeIncrease, cfg, "CullPercent")/100
	maxCullPercent := math.Max(criticalCull, regularCull) * incCullPercent
	globalOutput["CullPercent"] = maxCullPercent
	globalOutput["CullMultiplier"] = 100 / (100 - globalOutput["CullPercent"])

	//Calculate reservation DPS
	globalOutput["ReservationDpsMultiplier"] = 100 / (100 - enemyDB.Sum(mod.TypeBase, nil, "LifeReservationPercent"))

	//pob2:block:src/Modules/CalcOffence.lua:4102-4125
	for _, damageType := range dmgTypeList {
		state.gainTable[damageType] = map[string]float64{}
		for _, toType := range dmgTypeList {
			globalNames := []string{"DamageAs" + toType, "DamageGainAs" + toType, damageType + "DamageAs" + toType, damageType + "DamageGainAs" + toType}
			skillNames := []string{"SkillDamageGainAs" + toType, "Skill" + damageType + "DamageGainAs" + toType}
			if isElemental[damageType] {
				globalNames = append(globalNames, "ElementalDamageAs"+toType, "ElementalDamageGainAs"+toType)
				skillNames = append(skillNames, "SkillElementalDamageGainAs"+toType)
			}
			if damageType != "Chaos" {
				globalNames = append(globalNames, "NonChaosDamageAs"+toType, "NonChaosDamageGainAs"+toType)
				skillNames = append(skillNames, "SkillNonChaosDamageGainAs"+toType)
			}
			globalGain := math.Max(skillModList.Sum(mod.TypeBase, skillCfg, globalNames...), 0)
			skillGain := math.Max(skillModList.Sum(mod.TypeBase, skillCfg, skillNames...), 0)
			if skillModList.Flag(skillCfg, "DamageGainIsOnlyCold") && toType != "Cold" {
				state.gainTable[damageType]["Cold"] += (globalGain + skillGain) / 100
			} else {
				state.gainTable[damageType][toType] += (globalGain + skillGain) / 100
			}
		}
	}

	// Calculate base hit damage
	//pob2:block:src/Modules/CalcOffence.lua:4127-4158
	baseMultiplier, ok := activeSkill.ActiveEffect.GrantedEffectLevel.Extra["baseMultiplier"]
	if !ok {
		if baseMultiplier, ok = sdNum(skillData, "baseMultiplier"); !ok {
			baseMultiplier = 1
		}
	}
	for _, damageType := range dmgTypeList {
		damageTypeMin := damageType + "Min"
		damageTypeMax := damageType + "Max"
		addedMin := skillModList.Sum(mod.TypeBase, cfg, damageTypeMin) + enemyDB.Sum(mod.TypeBase, cfg, "Self"+damageTypeMin)
		addedMax := skillModList.Sum(mod.TypeBase, cfg, damageTypeMax) + enemyDB.Sum(mod.TypeBase, cfg, "Self"+damageTypeMax)
		addedMult := calcLibMod(skillModList, cfg, "Added"+damageType+"Damage", "AddedDamage")
		baseMin := (source.num(damageTypeMin) + source.num(damageType+"BonusMin") + (addedMin * addedMult)) * baseMultiplier
		baseMax := (source.num(damageTypeMax) + source.num(damageType+"BonusMax") + (addedMax * addedMult)) * baseMultiplier
		output[damageTypeMin+"Base"] = baseMin
		output[damageTypeMax+"Base"] = baseMax
	}

	//pob2:block:src/Modules/CalcOffence.lua:4160-4190
	for _, damageType := range dmgTypeList {
		convMult := state.conversionTable[damageType].mult
		convertedMin, convertedMax := calcConvertedDamage(state, output, damageType)
		gainedMin, gainedMax := calcGainedDamage(state, output, damageType)
		baseMin := output[damageType+"MinBase"]
		baseMax := output[damageType+"MaxBase"]
		output[damageType+"SummedMinBase"] = baseMin*convMult + convertedMin + gainedMin
		output[damageType+"SummedMaxBase"] = baseMax*convMult + convertedMax + gainedMax
		output[damageType+"StoredCombinedAvg"] = 0
	}

	// Calculate hit damage for each damage type
	//pob2:block:src/Modules/CalcOffence.lua:4192-4476
	totalHitMin, totalHitMax, totalHitAvg := 0.0, 0.0, 0.0
	totalCritMin, totalCritMax, totalCritAvg := 0.0, 0.0, 0.0
	ghostReaver := skillModList.Flag(nil, "GhostReaver")
	output["LifeLeech"] = 0
	output["LifeLeechInstant"] = 0
	output["EnergyShieldLeech"] = 0
	output["EnergyShieldLeechInstant"] = 0
	output["ManaLeech"] = 0
	output["ManaLeechInstant"] = 0

	for critPass := 1; critPass <= 2; critPass++ {
		// Pass 1 is critical strike damage, pass 2 is non-critical strike
		cfg.SkillCond["CriticalStrike"] = critPass == 1
		lifeLeechTotal := 0.0
		energyShieldLeechTotal := 0.0
		manaLeechTotal := 0.0
		noLifeLeech := skillModList.Flag(cfg, "CannotLeechLife") || enemyDB.Flag(nil, "CannotLeechLifeFromSelf") || skillModList.Flag(cfg, "CannotGainLife")
		noEnergyShieldLeech := skillModList.Flag(cfg, "CannotLeechEnergyShield") || enemyDB.Flag(nil, "CannotLeechEnergyShieldFromSelf") || skillModList.Flag(cfg, "CannotGainEnergyShield")
		noManaLeech := skillModList.Flag(cfg, "CannotLeechMana") || enemyDB.Flag(nil, "CannotLeechManaFromSelf") || skillModList.Flag(cfg, "CannotGainMana")
		for _, damageType := range dmgTypeList {
			damageTypeHitMin, damageTypeHitMax, damageTypeHitAvg := 0.0, 0.0, 0.0
			if skillFlags["hit"] && state.canDeal[damageType] {
				damageTypeHitMin, damageTypeHitMax = calcDamage(activeSkill, output, cfg, damageType, 0)
				if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
					output["allMult"] = output["ScaledDamageEffect"] * output["FistOfWarDamageEffect"] * output["AncestralCallDamageEffect"] * output["AncestralEmpowermentDamageEffect"] * output["AncestralEmpowermentCombinedDamageEffect"] * globalOutput["MaxOffensiveWarcryEffect"]
				} else {
					output["allMult"] = output["ScaledDamageEffect"] * output["FistOfWarDamageEffect"] * output["AncestralCallDamageEffect"] * output["AncestralEmpowermentDamageEffect"] * output["AncestralEmpowermentCombinedDamageEffect"] * globalOutput["OffensiveWarcryEffect"]
				}
				allMult := output["allMult"]
				if critPass == 1 {
					// Apply crit multiplier
					allMult *= output["CritMultiplier"]
				}
				damageTypeHitMin *= allMult
				damageTypeHitMax *= allMult
				var damageTypeLuckyChance float64
				if skillModList.Flag(skillCfg, "LuckyHits") ||
					(critPass == 2 && damageType == "Lightning" && skillModList.Flag(skillCfg, "LightningNoCritLucky")) ||
					(critPass == 1 && skillModList.Flag(skillCfg, "CritLucky")) ||
					((damageType == "Lightning" || damageType == "Cold" || damageType == "Fire") && skillModList.Flag(skillCfg, "ElementalLuckHits")) {
					damageTypeLuckyChance = 1
				} else {
					damageTypeLuckyChance = math.Min(skillModList.Sum(mod.TypeBase, skillCfg, damageType+"LuckyHitsChance", "LuckyHitsChance"), 100) / 100
				}
				damageTypeHitAvgNotLucky := damageTypeHitMin/2 + damageTypeHitMax/2
				damageTypeHitAvgLucky := damageTypeHitMin/3 + 2*damageTypeHitMax/3
				damageTypeHitAvg = damageTypeHitAvgNotLucky*(1-damageTypeLuckyChance) + damageTypeHitAvgLucky*damageTypeLuckyChance

				// Store pre-resist/armour/penetration hit damage for ailment calculations
				if critPass == 1 {
					output[damageType+"StoredCombinedAvg"] += damageTypeHitAvg * (output["CritChance"] / 100)
					output[damageType+"StoredCritAvg"] = damageTypeHitAvg
					output[damageType+"StoredCritMin"] = damageTypeHitMin
					output[damageType+"StoredCritMax"] = damageTypeHitMax
				} else {
					output[damageType+"StoredCombinedAvg"] += damageTypeHitAvg * (1 - output["CritChance"]/100)
					output[damageType+"StoredHitAvg"] = damageTypeHitAvg
					output[damageType+"StoredHitMin"] = damageTypeHitMin
					output[damageType+"StoredHitMax"] = damageTypeHitMax
				}

				if (damageTypeHitMin != 0 || damageTypeHitMax != 0) && env.ModeEffective {
					effMult := h.hitEffMult(cfg, damageType, damageTypeHitAvg)
					damageTypeHitMin *= effMult
					damageTypeHitMax *= effMult
					damageTypeHitAvg *= effMult
				}

				// Beginning of Leech Calculation for this DamageType
				lifeLeech := 0.0
				energyShieldLeech := 0.0
				manaLeech := 0.0

				if skillFlags["mine"] || skillFlags["trap"] || skillFlags["totem"] {
					lifeLeech = skillModList.Sum(mod.TypeBase, cfg, "DamageLifeLeechToPlayer")
				} else {
					lifeLeech = getBaseLeech("Life", damageType, skillModList, cfg, enemyDB)
					energyShieldLeech = getBaseLeech("EnergyShield", damageType, skillModList, cfg, enemyDB)
					manaLeech = getBaseLeech("Mana", damageType, skillModList, cfg, enemyDB)
				}

				if ghostReaver && !noLifeLeech {
					energyShieldLeech += lifeLeech
					lifeLeech = 0
				}

				if lifeLeech > 0 && !noLifeLeech {
					lifeLeechTotal += damageTypeHitAvg * lifeLeech / 100
				}
				if manaLeech > 0 && !noManaLeech {
					manaLeechTotal += damageTypeHitAvg * manaLeech / 100
				}
				if energyShieldLeech > 0 && !noEnergyShieldLeech {
					energyShieldLeechTotal += damageTypeHitAvg * energyShieldLeech / 100
				}
			}
			if critPass == 1 {
				output[damageType+"CritAverage"] = damageTypeHitAvg
				totalCritAvg += damageTypeHitAvg
				totalCritMin += damageTypeHitMin
				totalCritMax += damageTypeHitMax
			} else {
				output[damageType+"HitAverage"] = damageTypeHitAvg
				totalHitAvg += damageTypeHitAvg
				totalHitMin += damageTypeHitMin
				totalHitMax += damageTypeHitMax
			}
		}
		if v, ok := sdNum(skillData, "lifeLeechPerUse"); ok {
			lifeLeechTotal += v
		}
		if v, ok := sdNum(skillData, "manaLeechPerUse"); ok {
			manaLeechTotal += v
		}

		// leech caps per instance
		lifeLeechTotal = math.Min(lifeLeechTotal, globalOutput["MaxLifeLeechInstance"])
		energyShieldLeechTotal = math.Min(energyShieldLeechTotal, globalOutput["MaxEnergyShieldLeechInstance"])
		manaLeechTotal = math.Min(manaLeechTotal, globalOutput["MaxManaLeechInstance"])

		portion := 1 - output["CritChance"]/100
		if critPass == 1 {
			portion = output["CritChance"] / 100
		}
		output["LifeLeech"] += lifeLeechTotal * portion
		output["EnergyShieldLeech"] += energyShieldLeechTotal * portion
		output["ManaLeech"] += manaLeechTotal * portion
	}
	output["TotalMin"] = totalHitMin
	output["TotalMax"] = totalHitMax

	//pob2:block:src/Modules/CalcOffence.lua:4478-4502
	eeIgnore := env.ConfigInput["EEIgnoreHitDamage"]
	if skillModList.Flag(skillCfg, "ElementalEquilibrium") && !(eeIgnore.IsBoolean && eeIgnore.Boolean) && (output["FireHitAverage"]+output["ColdHitAverage"]+output["LightningHitAverage"] > 0) {
		// Update enemy hit-by-damage-type conditions
		moddb.SetCondition(enemyDB.ModStore, "HitByFireDamage", output["FireHitAverage"] > 0)
		moddb.SetCondition(enemyDB.ModStore, "HitByColdDamage", output["ColdHitAverage"] > 0)
		moddb.SetCondition(enemyDB.ModStore, "HitByLightningDamage", output["LightningHitAverage"] > 0)
	}

	highestType := "Physical"

	// For each damage type, calculate percentage of total damage. Also tracks the highest damage type and outputs a Condition:TypeIsHighestDamageType flag for whichever the highest type is
	for _, damageType := range dmgTypeList {
		if output[damageType+"HitAverage"] > 0 {
			skillModList.AddMod(newFlagMod("Condition:"+damageType+"HasDamage", "Config"))
			if output[damageType+"HitAverage"] > output[highestType+"HitAverage"] {
				highestType = damageType
			}
		}
	}
	if !skillModList.Flag(nil, "IsHighestDamageTypeOVERRIDE") {
		skillModList.AddMod(newFlagMod("Condition:"+highestType+"IsHighestDamageType", "Config"))
	}

	//Instant Leech
	//pob2:block:src/Modules/CalcOffence.lua:4506-4561
	output["LifeLeechInstantProportion"] = math.Max(math.Min(skillModList.Sum(mod.TypeBase, cfg, "InstantLifeLeech"), 100), 0) / 100
	if output["LifeLeechInstantProportion"] > 0 {
		output["LifeLeechInstant"] = output["LifeLeech"] * output["LifeLeechInstantProportion"]
		output["LifeLeech"] *= 1 - output["LifeLeechInstantProportion"]
	}
	output["ManaLeechInstantProportion"] = math.Max(math.Min(skillModList.Sum(mod.TypeBase, cfg, "InstantManaLeech"), 100), 0) / 100
	if output["ManaLeechInstantProportion"] > 0 {
		output["ManaLeechInstant"] = output["ManaLeech"] * output["ManaLeechInstantProportion"]
		output["ManaLeech"] *= 1 - output["ManaLeechInstantProportion"]
	}
	output["EnergyShieldLeechInstantProportion"] = math.Max(math.Min(skillModList.Sum(mod.TypeBase, cfg, "InstantEnergyShieldLeech"), 100), 0) / 100
	if skillModList.Flag(cfg, "ManaLeechRecoversEnergyShield") {
		output["EnergyShieldLeechInstantProportion"] += output["ManaLeechInstantProportion"]
	}
	if output["EnergyShieldLeechInstantProportion"] > 0 {
		output["EnergyShieldLeechInstant"] = output["EnergyShieldLeech"] * output["EnergyShieldLeechInstantProportion"]
		output["EnergyShieldLeech"] *= 1 - output["EnergyShieldLeechInstantProportion"]
	}

	output["LifeLeechDuration"], output["LifeLeechInstances"] = getLeechInstances(output["LifeLeech"], globalOutput["Life"], hitRate)
	output["LifeLeechInstantRate"] = output["LifeLeechInstant"] * hitRate
	output["EnergyShieldLeechDuration"], output["EnergyShieldLeechInstances"] = getLeechInstances(output["EnergyShieldLeech"], globalOutput["EnergyShield"], hitRate)
	output["EnergyShieldLeechInstantRate"] = output["EnergyShieldLeechInstant"] * hitRate
	output["ManaLeechDuration"], output["ManaLeechInstances"] = getLeechInstances(output["ManaLeech"], globalOutput["Mana"], hitRate)
	output["ManaLeechInstantRate"] = output["ManaLeechInstant"] * hitRate

	// Calculate gain on hit
	if skillFlags["mine"] || skillFlags["trap"] || skillFlags["totem"] {
		output["LifeOnHit"] = 0
		output["EnergyShieldOnHit"] = 0
		output["ManaOnHit"] = 0
	} else {
		output["LifeOnHit"] = 0
		if !skillModList.Flag(cfg, "CannotGainLife") && !skillModList.Flag(cfg, "CannotRecoverLifeOutsideLeech") {
			output["LifeOnHit"] = skillModList.Sum(mod.TypeBase, cfg, "LifeOnHit") + enemyDB.Sum(mod.TypeBase, cfg, "SelfLifeOnHit")
		}
		output["EnergyShieldOnHit"] = 0
		if !skillModList.Flag(cfg, "CannotGainEnergyShield") {
			output["EnergyShieldOnHit"] = skillModList.Sum(mod.TypeBase, cfg, "EnergyShieldOnHit") + enemyDB.Sum(mod.TypeBase, cfg, "SelfEnergyShieldOnHit")
		}
		output["ManaOnHit"] = 0
		if !skillModList.Flag(cfg, "CannotGainMana") {
			output["ManaOnHit"] = skillModList.Sum(mod.TypeBase, cfg, "ManaOnHit") + enemyDB.Sum(mod.TypeBase, cfg, "SelfManaOnHit")
		}
	}
	output["LifeOnHitRate"] = output["LifeOnHit"] * hitRate
	output["EnergyShieldOnHitRate"] = output["EnergyShieldOnHit"] * hitRate
	output["ManaOnHitRate"] = output["ManaOnHit"] * hitRate

	// Calculate gain on kill
	if skillFlags["mine"] || skillFlags["trap"] || skillFlags["totem"] {
		output["LifeOnKill"] = 0
		output["EnergyShieldOnKill"] = 0
		output["ManaOnKill"] = 0
	} else {
		output["LifeOnKill"] = 0
		if !skillModList.Flag(cfg, "CannotGainLife") && !skillModList.Flag(cfg, "CannotRecoverLifeOutsideLeech") {
			output["LifeOnKill"] = math.Floor(skillModList.Sum(mod.TypeBase, cfg, "LifeOnKill"))
		}
		output["EnergyShieldOnKill"] = 0
		if !skillModList.Flag(cfg, "CannotGainEnergyShield") {
			output["EnergyShieldOnKill"] = math.Floor(skillModList.Sum(mod.TypeBase, cfg, "EnergyShieldOnKill"))
		}
		output["ManaOnKill"] = 0
		if !skillModList.Flag(cfg, "CannotGainMana") {
			output["ManaOnKill"] = math.Floor(skillModList.Sum(mod.TypeBase, cfg, "ManaOnKill"))
		}
	}

	// Enemy Regeneration Rate
	output["EnemyLifeRegen"] = enemyDB.Sum(mod.TypeIncrease, cfg, "LifeRegen")
	output["EnemyManaRegen"] = enemyDB.Sum(mod.TypeIncrease, cfg, "ManaRegen")
	output["EnemyEnergyShieldRegen"] = enemyDB.Sum(mod.TypeIncrease, cfg, "EnergyShieldRegen")

	// Calculate average damage and final DPS
	//pob2:block:src/Modules/CalcOffence.lua:4564-4576
	output["AverageHit"] = totalHitAvg*(1-output["CritChance"]/100) + totalCritAvg*output["CritChance"]/100
	if skillFlags["monsterExplode"] {
		output["AverageHitToMonsterLifePercentage"] = output["AverageHit"] / h.monsterLife * 100
		if truthy(skillData["hitChanceIsExplodeChance"]) {
			output["HitChance"] = output["ExplodeChance"]
		}
	}
	output["AverageDamage"] = output["AverageHit"] * output["HitChance"] / 100
	if v, ok := output["AverageBurstHits"]; ok {
		globalOutput["AverageBurstHits"] = v
	} else {
		globalOutput["AverageBurstHits"] = 1
	}
	repeatPenalty := 1.0
	if skillModList.Flag(nil, "HasSeals") && skillModList.Flag(nil, "DamageSeal") && !skillModList.Flag(nil, "NoRepeatBonuses") {
		repeatPenalty = calcLibMod(skillModList, skillCfg, "SealRepeatPenalty")
	}
	globalOutput["AverageBurstDamage"] = output["AverageDamage"] + output["AverageDamage"]*(globalOutput["AverageBurstHits"]-1)*repeatPenalty
	h.actor.setOutputFlag("ShowBurst", globalOutput["AverageBurstHits"] > 1)
	output["TotalDPS"] = output["AverageDamage"] * hitSpeed * output["DpsMultiplier"] * quantityMultiplier
	//pob2:block:src/Modules/CalcOffence.lua:4577-4598
	if breakdown := pass.breakdown; breakdown != nil {
		if output["CritEffect"] != 1 {
			var lines []string
			if skillModList.Flag(skillCfg, "LuckyHits") {
				lines = append(lines, sFormat("(1/3) x %d + (2/3) x %d = %.1f ^8(average from non-crits)", totalHitMin, totalHitMax, totalHitAvg))
			}
			if skillModList.Flag(skillCfg, "CritLucky") || skillModList.Flag(skillCfg, "LuckyHits") {
				lines = append(lines, sFormat("(1/3) x %d + (2/3) x %d = %.1f ^8(average from crits)", totalCritMin, totalCritMax, totalCritAvg), "")
			}
			breakdown.set("AverageHit", append(lines,
				sFormat("%.1f x (1 - %.4f) ^8(damage from non-crits)", totalHitAvg, output["CritChance"]/100),
				sFormat("+ %.1f x %.4f ^8(damage from crits)", totalCritAvg, output["CritChance"]/100),
				sFormat("= %.1f", output["AverageHit"]),
			)...)
		}
		if output["HitChance"] < 100 {
			breakdown.set("AverageDamage", sFormat("%s:", pass.label), sFormat("%.1f ^8(average hit)", output["AverageHit"]), sFormat("x %.2f ^8(chance to hit)", output["HitChance"]/100), sFormat("= %.1f", output["AverageDamage"]))
		}
	}
	//pob2:block:src/Modules/CalcOffence.lua:4599-4618
	if globalBreakdown := h.actor.Breakdown; globalBreakdown != nil && globalOutput["AverageBurstDamage"] > 0 {
		lines := []string{sFormat("%.1f ^8(average hit)", output["AverageHit"])}
		if output["HitChance"] < 100 {
			lines = append(lines, sFormat("x %.2f ^8(chance to hit)", output["HitChance"]/100))
		}
		if repeatPenalty < 1 {
			lines = append(lines,
				sFormat("x %.2f ^8(number of repeats)", globalOutput["AverageBurstHits"]-1),
				sFormat("x %.2f ^8(repeat penalty)", repeatPenalty),
				sFormat("= %.1f ^8(repeat damage total)", globalOutput["AverageBurstDamage"]-output["AverageDamage"]),
				"",
				sFormat("+ %.1f ^8(first hit)", output["AverageHit"]),
			)
			if output["HitChance"] < 100 {
				lines = append(lines, sFormat("x %.2f ^8(chance to hit)", output["HitChance"]/100))
			}
		} else {
			lines = append(lines, sFormat("x %.2f ^8(number of hits)", globalOutput["AverageBurstHits"]))
		}
		globalBreakdown.set("AverageBurstDamage", append(lines, sFormat("= %.1f ^8(total burst damage)", globalOutput["AverageBurstDamage"]))...)
	}

	h.offencePvp(pass, totalHitMin, totalHitMax, hitSpeed)
}

// hitEffMult applies enemy resistances and damage taken modifiers to one damage type of a hit. The
// party member resistance source and the breakdowns are not ported.
//
//pob2:block:src/Modules/CalcOffence.lua:4284-4403
func (h *offenceHit) hitEffMult(cfg *moddb.ListCfg, damageType string, damageTypeHitAvg float64) float64 {
	env, actor, activeSkill, skillFlags := h.env, h.actor, h.activeSkill, h.skillFlags
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	skillModList := activeSkill.SkillModList

	resist := 0.0
	pen := 0.0
	minPen := 0.0
	takenInc := enemyDB.Sum(mod.TypeIncrease, cfg, "DamageTaken", damageType+"DamageTaken")
	takenMore := enemyDB.More(cfg, "DamageTaken", damageType+"DamageTaken")

	if damageType == "Physical" {
		enemyArmourMin := 0.0
		configSource := mod.Source("Config")
		if canBreak, _ := modDB.GetCondition("CanArmourBreakBelowZero", cfg, false); canBreak { // check for possibility to break Armour below zero
			enemyArmourMin = -enemyDB.Sum(mod.TypeBase, &moddb.ListCfg{Source: &configSource}, "Armour")                                                                                                                                                  // adjust minimum armour value
			enemyDB.ReplaceMod(newMod("Armour", mod.TypeOverride, -enemyDB.Sum(mod.TypeBase, &moddb.ListCfg{Source: &configSource}, "Armour"), "ArmourBreak", mod.Condition("ArmourBrokenBelowZeroMax"), mod.GlobalEffect("Debuff").Name("ArmourBreak"))) // if Config is set to Max, add mod with max value (use replace to avoid doubling)
		}
		var enemyArmour float64
		if o := enemyDB.Override(nil, "Armour"); o != nil {
			enemyArmour = o.Float()
		} else {
			enemyArmour = math.Max(calcLibVal(enemyDB, "Armour", nil), enemyArmourMin)
		}
		chanceIgnoreEnemyArmour := math.Min(skillModList.Sum(mod.TypeBase, cfg, "ChanceToIgnoreEnemyArmour")/100, 1)
		// check for mods that ignore Armour
		ignoreEnemyArmour := calcLibVal(enemyDB, "IgnoreArmour", nil)
		if skillModList.Flag(cfg, "IgnoreEnemyArmour") {
			ignoreEnemyArmour = enemyArmour
		}
		if enemyArmour > 0 {
			enemyArmour = math.Max(enemyArmour-ignoreEnemyArmour, 0)
		} // subtract ignored value up to zero, if Armour is still positive (to allow future support of negative Armour)
		armourReduction := armourReductionF(enemyArmour, damageTypeHitAvg*skillModList.More(cfg, "CalcArmourAsThoughDealing"))
		chanceToIgnoreEnemyPhysicalDamageReduction := math.Min(skillModList.Sum(mod.TypeBase, cfg, "ChanceToIgnoreEnemyPhysicalDamageReduction"), 100)
		if chanceToIgnoreEnemyPhysicalDamageReduction > 0 && chanceToIgnoreEnemyPhysicalDamageReduction < 100 {
			switch m, _ := configListValue(env, "ChanceToIgnoreEnemyPhysicalDamageReductionMode"); m {
			case "MAX":
				chanceToIgnoreEnemyPhysicalDamageReduction = 100
			case "MIN":
				chanceToIgnoreEnemyPhysicalDamageReduction = 0
			}
		}
		if skillModList.Flag(cfg, "IgnoreEnemyPhysicalDamageReduction") || chanceToIgnoreEnemyPhysicalDamageReduction >= 100 {
			resist = 0
		} else {
			resist = math.Min(math.Max(-miscConstant("NegArmourDmgBonusCap"), enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction")+skillModList.Sum(mod.TypeBase, cfg, "EnemyPhysicalDamageReduction")+armourReduction*(1-chanceIgnoreEnemyArmour)), miscConstant("EnemyPhysicalDamageReductionCap"))
			if resist > 0 {
				resist *= 1 - (skillModList.Sum(mod.TypeBase, nil, "PartialIgnoreEnemyPhysicalDamageReduction")/100 + chanceToIgnoreEnemyPhysicalDamageReduction/100)
			}
		}
	} else {
		resist = calcResistForType(env, enemyDB, damageType, cfg)
		if ((skillModList.Flag(cfg, "ChaosDamageUsesLowestResistance") || skillModList.Flag(cfg, "ChaosDamageUsesHighestResistance")) && damageType == "Chaos") ||
			(skillModList.Flag(cfg, "ElementalDamageUsesLowestResistance") && isElemental[damageType]) {
			// Default to using the current damage type
			elementUsed := damageType
			if isElemental[damageType] {
				takenInc += enemyDB.Sum(mod.TypeIncrease, cfg, "ElementalDamageTaken")
			}
			// Find the lowest resist of all the elements and use that if it's lower
			for _, eleDamageType := range dmgTypeList {
				if isElemental[eleDamageType] && h.useThisResist(cfg, eleDamageType) && damageType != eleDamageType {
					currentElementResist := calcResistForType(env, enemyDB, eleDamageType, cfg)
					// If it's explicitly lower, then use the resist and update which element we're using to account for penetration
					if skillModList.Flag(cfg, "ChaosDamageUsesHighestResistance") {
						if resist < currentElementResist {
							resist = currentElementResist
							elementUsed = eleDamageType
						}
					} else if resist > currentElementResist {
						resist = currentElementResist
						elementUsed = eleDamageType
					}
				}
			}
			// Update the penetration based on the element used
			if isElemental[elementUsed] {
				pen = skillModList.Sum(mod.TypeBase, cfg, elementUsed+"Penetration", "ElementalPenetration")
				minPen = skillModList.Sum(mod.TypeBase, cfg, elementUsed+"PenetrationMinimum", "ElementalPenetrationMinimum")
			} else if elementUsed == "Chaos" {
				pen = skillModList.Sum(mod.TypeBase, cfg, "ChaosPenetration")
			}
		} else if isElemental[damageType] {
			pen = skillModList.Sum(mod.TypeBase, cfg, damageType+"Penetration", "ElementalPenetration")
			minPen = skillModList.Sum(mod.TypeBase, cfg, damageType+"PenetrationMinimum", "ElementalPenetrationMinimum")
			takenInc += enemyDB.Sum(mod.TypeIncrease, cfg, "ElementalDamageTaken")
		} else if damageType == "Chaos" {
			pen = skillModList.Sum(mod.TypeBase, cfg, "ChaosPenetration")
		}
	}

	invertChance := 0.0
	if isElemental[damageType] {
		invertChance = math.Max(math.Min(skillModList.Sum(mod.TypeChance, cfg, "HitsInvertEleResChance"), 1), 0)
	}
	if skillFlags["projectile"] {
		takenInc += enemyDB.Sum(mod.TypeIncrease, nil, "ProjectileDamageTaken")
	}
	if skillFlags["projectile"] && skillFlags["attack"] {
		takenInc += enemyDB.Sum(mod.TypeIncrease, nil, "ProjectileAttackDamageTaken")
	}
	if skillFlags["trap"] || skillFlags["mine"] {
		takenInc += enemyDB.Sum(mod.TypeIncrease, nil, "TrapMineDamageTaken")
	}
	effMult := (1 + takenInc/100) * takenMore
	useRes := h.useThisResist(cfg, damageType)
	cannotElePenIgnore := isElemental[damageType] && skillModList.Flag(cfg, "CannotElePenIgnore")
	ignoreNonNegativeEleRes := isElemental[damageType] && modDB.Flag(cfg, "IgnoreNonNegativeEleRes")
	normalHitResist := calcHitResist(resist, minPen, pen, cannotElePenIgnore, ignoreNonNegativeEleRes)
	invertedHitResist := calcHitResist(-resist, minPen, pen, cannotElePenIgnore, ignoreNonNegativeEleRes)
	if cannotElePenIgnore || useRes {
		var effectiveResist float64
		if isElemental[damageType] && invertChance > 0 {
			effectiveResist = normalHitResist*(1-invertChance) + invertedHitResist*invertChance
		} else {
			effectiveResist = normalHitResist
		}
		effMult *= 1 - effectiveResist/100
	}
	return effMult
}

// offencePvp calculates the PvP values. The breakdowns are not ported.
//
//pob2:block:src/Modules/CalcOffence.lua:4621-4718
func (h *offenceHit) offencePvp(pass *offencePass, totalHitMin, totalHitMax, hitSpeed float64) {
	env, skillFlags := h.env, h.skillFlags
	globalOutput := h.actor.Output
	output, cfg := pass.output, pass.cfg
	skillModList, skillData := h.activeSkill.SkillModList, h.activeSkill.SkillData

	//setup flags
	skillFlags["isPvP"] = false
	skillFlags["notAttackPvP"] = false
	skillFlags["attackPvP"] = false
	skillFlags["weapon1AttackPvP"] = false
	skillFlags["weapon2AttackPvP"] = false
	skillFlags["notAveragePvP"] = false

	pvpScaling := env.ConfigInput["PvpScaling"]
	if !(pvpScaling.IsBoolean && pvpScaling.Boolean) {
		return
	}
	skillFlags["isPvP"] = true
	skillFlags["attackPvP"] = skillFlags["attack"]
	skillFlags["notAttackPvP"] = !skillFlags["attack"]
	skillFlags["weapon1AttackPvP"] = skillFlags["weapon1Attack"]
	skillFlags["weapon2AttackPvP"] = skillFlags["weapon2Attack"]
	skillFlags["notAveragePvP"] = skillFlags["notAverage"]
	pvpTvalue, ok := configInputNumber(env, "multiplierPvpTvalueOverride")
	if ok {
		pvpTvalue /= 1000
	} else {
		if cooldown, ok := sdNum(skillData, "cooldown"); ok {
			pvpTvalue = cooldown
		} else if skillFlags["mine"] {
			t, ok := output["MineLayingTime"]
			if !ok {
				t = 1
			}
			pvpTvalue = t / globalOutput["ActionSpeedMod"]
		} else if skillFlags["trap"] {
			t, ok := output["TrapThrowingTime"]
			if !ok {
				t = 1
			}
			pvpTvalue = t / globalOutput["ActionSpeedMod"]
		} else {
			pvpTvalue = 1 / (hitSpeed / globalOutput["ActionSpeedMod"]) * skillModList.More(cfg, "PvpTvalueMultiplier")
		}
		if pvpTvalue > 2147483647 {
			pvpTvalue = 1
		}
	}
	pvpMultiplier := skillModList.More(cfg, "PvpDamageMultiplier")

	pvpNonElemental1 := miscConstant("PvpNonElemental1")
	pvpNonElemental2 := miscConstant("PvpNonElemental2")
	pvpElemental1 := miscConstant("PvpElemental1")
	pvpElemental2 := miscConstant("PvpElemental2")

	percentageNonElemental := (output["PhysicalHitAverage"] + output["ChaosHitAverage"]) / (totalHitMin + totalHitMax) * 2
	percentageElemental := 1 - percentageNonElemental
	portionNonElemental := math.Pow(output["AverageHit"]/pvpTvalue/pvpNonElemental2, pvpNonElemental1) * pvpTvalue * pvpNonElemental2 * percentageNonElemental
	portionElemental := math.Pow(output["AverageHit"]/pvpTvalue/pvpElemental2, pvpElemental1) * pvpTvalue * pvpElemental2 * percentageElemental
	output["PvpAverageHit"] = (portionNonElemental + portionElemental) * pvpMultiplier
	output["PvpAverageDamage"] = output["PvpAverageHit"] * output["HitChance"] / 100
	output["PvpTotalDPS"] = output["PvpAverageDamage"] * hitSpeed * output["DpsMultiplier"]

	// fix for these being nan
	if math.IsNaN(output["PvpAverageHit"]) {
		output["PvpAverageHit"] = 0
	}
	if math.IsNaN(output["PvpAverageDamage"]) {
		output["PvpAverageDamage"] = 0
	}
	if math.IsNaN(output["PvpTotalDPS"]) {
		output["PvpTotalDPS"] = 0
	}
}

// offenceCombineDamage combines crit stats, average damage and DPS. The breakdowns are not ported.
//
//pob2:block:src/Modules/CalcOffence.lua:4721-4850
func (h *offenceHit) offenceCombineDamage() {
	state, skillFlags := h.state, h.skillFlags
	output := h.actor.Output
	skillData := h.activeSkill.SkillData
	if !state.isAttack {
		return
	}
	combine := func(stat, mode string) {
		state.combineStat(output, skillFlags, skillData, stat, mode)
	}
	combine("PreEffectiveCritChance", "AVERAGE")
	combine("CritChance", "CRIT")
	combine("PreEffectiveCritMultiplier", "AVERAGE")
	combine("CritMultiplier", "AVERAGE")
	combine("CritBifurcates", "AVERAGE")
	for _, stat := range []string{"AverageDamage", "PvpAverageDamage", "DpsMultiplier", "TotalDPS", "PvpTotalDPS",
		"LifeLeechDuration", "LifeLeechInstances", "LifeLeechInstant", "LifeLeechInstantRate", "LifeLeechInstantProportion",
		"EnergyShieldLeechDuration", "EnergyShieldLeechInstances", "EnergyShieldLeechInstant", "EnergyShieldLeechInstantRate", "EnergyShieldLeechInstantProportion",
		"ManaLeechDuration", "ManaLeechInstances", "ManaLeechInstant", "ManaLeechInstantRate", "ManaLeechInstantProportion",
		"LifeOnHit", "LifeOnHitRate", "LifeOnKill", "EnergyShieldOnHit", "EnergyShieldOnHitRate", "EnergyShieldOnKill",
		"ManaOnHit", "ManaOnHitRate", "ManaOnKill"} {
		combine(stat, "DPS")
	}
	for _, damageType := range dmgTypeList {
		combine(damageType+"StoredCombinedAvg", "DPS")
	}
	// Crossbows:
	if h.activeSkill.SkillTypes["CrossbowSkill"] && !h.activeSkill.SkillTypes["Grenade"] {
		// Combine stats related to reload and bolt functionality
		for _, stat := range []string{"FiringRate", "ReloadTime", "EffectiveReloadTime", "ReloadRate", "BoltCount", "EffectiveBoltCount", "TotalFiringTime", "ChanceToNotConsumeAmmo", "ChanceToReloadInstantly"} {
			combine(stat, "AVERAGE")
		}
		// Game data specifies "base skill show average damage instead of dps" for many crossbow skills, where that doesn't make sense for PoB (e.g. Explosive Shot)
		skillData["showAverage"] = false
		skillFlags["showAverage"] = false
		skillFlags["notAverage"] = true
	}
	//pob2:block:src/Modules/CalcOffence.lua:4826-4837
	if breakdown := h.actor.Breakdown; breakdown != nil && skillFlags["bothWeaponAttack"] {
		mh, oh := state.mainHand["AverageDamage"], state.offHand["AverageDamage"]
		line := sFormat("(%.1f + %.1f) / 2 ^8(skill alternates weapons)", mh, oh)
		if truthy(skillData["combinesHitsWhenDualWielding"]) {
			line = sFormat("%.1f + %.1f ^8(skill hits with both weapons at once)", mh, oh)
		} else if truthy(skillData["doubleHitsWhenDualWielding"]) {
			line = sFormat("(%.1f + %.1f) / 2 ^8(skill hits once with each weapon)", mh, oh)
		}
		breakdown.set("AverageDamage", "Both weapons:", line, sFormat("= %.1f", output["AverageDamage"]))
	}
}

//pob2:block:src/Modules/CalcOffence.lua:4858-4881
func (h *offenceHit) totalDPSBreakdown() {
	output := h.actor.Output
	breakdown := h.actor.Breakdown
	rate := func(label string) string {
		if v, ok := output["HitSpeed"]; ok {
			return sFormat("x %.2f ^8(hit rate)", v)
		}
		return sFormat("x %.2f ^8("+label+")", output["Speed"])
	}
	var lines []string
	switch {
	case h.state.isAttack:
		lines = []string{sFormat("%.1f ^8(average damage)", output["AverageDamage"]), rate("attack rate")}
	case truthy(h.activeSkill.SkillData["triggered"]):
		lines = []string{sFormat("%.1f ^8(average damage)", output["AverageDamage"]), rate("trigger rate")}
	default:
		lines = []string{sFormat("%.1f ^8(average hit)", output["AverageDamage"]), rate("cast rate")}
	}
	if output["DpsMultiplier"] != 1 {
		lines = append(lines, sFormat("x %g ^8(DPS multiplier for this skill)", output["DpsMultiplier"]))
	}
	if h.state.quantityMultiplier > 1 {
		lines = append(lines, sFormat("x %g ^8(quantity multiplier for this skill)", h.state.quantityMultiplier))
	}
	breakdown.set("TotalDPS", append(lines, sFormat("= %.1f", output["TotalDPS"]))...)
}

// offenceLeechRates calculates leech rates.
//
//pob2:block:src/Modules/CalcOffence.lua:4903-4948
func (h *offenceHit) offenceLeechRates() {
	output := h.actor.Output
	skillFlags := h.skillFlags
	skillModList, skillCfg, skillData := h.activeSkill.SkillModList, h.activeSkill.SkillCfg, h.activeSkill.SkillData

	if skillFlags["minion"] {
		hitSpeed, ok := output["HitSpeed"]
		if !ok {
			hitSpeed = output["Speed"]
		}
		skillData["summonSpeed"] = output["SummonedMinionsPerCast"] * hitSpeed * output["DpsMultiplier"]
	}

	leechRateBase := miscConstant("LeechRateBase")
	output["LifeLeechInstanceRate"] = output["Life"] * leechRateBase * calcLibMod(skillModList, skillCfg, "LifeLeechRate")
	output["LifeLeechRate"] = output["LifeLeechInstances"] * output["LifeLeechInstanceRate"]
	output["LifeLeechPerHit"] = output["LifeLeechInstanceRate"]
	output["EnergyShieldLeechInstanceRate"] = output["EnergyShield"] * leechRateBase * calcLibMod(skillModList, skillCfg, "EnergyShieldLeechRate")
	output["EnergyShieldLeechRate"] = output["EnergyShieldLeechInstances"] * output["EnergyShieldLeechInstanceRate"]
	output["EnergyShieldLeechPerHit"] = output["EnergyShieldLeechInstanceRate"]
	output["ManaLeechInstanceRate"] = output["Mana"] * leechRateBase * calcLibMod(skillModList, skillCfg, "ManaLeechRate")
	output["ManaLeechRate"] = output["ManaLeechInstances"] * output["ManaLeechInstanceRate"]
	output["ManaLeechPerHit"] = output["ManaLeechInstanceRate"]
	// On full life, Immortal Ambition treats life leech as energy shield leech
	if skillModList.Flag(nil, "ImmortalAmbition") {
		output["EnergyShieldLeechRate"] += output["LifeLeechRate"]
		output["EnergyShieldLeechPerHit"] += output["LifeLeechPerHit"]
		// Clears output.LifeLeechRate to disable leechLife flag
		output["LifeLeechRate"] = 0
		output["LifeLeechPerHit"] = 0
	}
	// Disable non-instant life leech
	if skillModList.Flag(nil, "UnaffectedByNonInstantLifeLeech") {
		output["LifeLeechRate"] = 0
		output["LifeLeechPerHit"] = 0
		output["LifeLeechInstances"] = 0
	}
	output["LifeLeechRate"] = output["LifeLeechInstantRate"] + math.Min(output["LifeLeechRate"], output["MaxLifeLeechRate"])*output["LifeRecoveryRateMod"]
	output["LifeLeechPerHit"] = output["LifeLeechInstant"] + math.Min(output["LifeLeechPerHit"], output["MaxLifeLeechRate"])*output["LifeLeechDuration"]*output["LifeRecoveryRateMod"]
	output["EnergyShieldLeechRate"] = output["EnergyShieldLeechInstantRate"] + math.Min(output["EnergyShieldLeechRate"], output["MaxEnergyShieldLeechRate"])*output["EnergyShieldRecoveryRateMod"]
	output["EnergyShieldLeechPerHit"] = output["EnergyShieldLeechInstant"] + math.Min(output["EnergyShieldLeechPerHit"], output["MaxEnergyShieldLeechRate"])*output["EnergyShieldLeechDuration"]*output["EnergyShieldRecoveryRateMod"]
	output["ManaLeechRate"] = output["ManaLeechInstantRate"] + math.Min(output["ManaLeechRate"], output["MaxManaLeechRate"])*output["ManaRecoveryRateMod"]
	output["ManaLeechPerHit"] = output["ManaLeechInstant"] + math.Min(output["ManaLeechPerHit"], output["MaxManaLeechRate"])*output["ManaLeechDuration"]*output["ManaRecoveryRateMod"]
	skillFlags["leechLife"] = output["LifeLeechRate"] > 0
	skillFlags["leechES"] = output["EnergyShieldLeechRate"] > 0
	skillFlags["leechMana"] = output["ManaLeechRate"] > 0
	if truthy(skillData["showAverage"]) {
		output["LifeLeechGainPerHit"] = output["LifeLeechPerHit"] + output["LifeOnHit"]
		output["EnergyShieldLeechGainPerHit"] = output["EnergyShieldLeechPerHit"] + output["EnergyShieldOnHit"]
		output["ManaLeechGainPerHit"] = output["ManaLeechPerHit"] + output["ManaOnHit"]
	} else {
		output["LifeLeechGainRate"] = output["LifeLeechRate"] + output["LifeOnHitRate"]
		output["EnergyShieldLeechGainRate"] = output["EnergyShieldLeechRate"] + output["EnergyShieldOnHitRate"]
		output["ManaLeechGainRate"] = output["ManaLeechRate"] + output["ManaOnHitRate"]
	}
}

// offenceHitHalf runs CalcOffence's damage passes, then the ailments, damage over time and the
// combined DPS. hasCosts is output[resource.."HasCost"] from the cost calculation.
func offenceHitHalf(env *Environment, actor *Actor, activeSkill *ActiveSkill, skillFlags map[string]bool, monsterLife float64, hasCosts map[string]bool) *offenceHitState {
	state := &offenceHitState{isAttack: skillFlags["attack"], monsterLife: monsterLife}
	h := &offenceHit{env: env, actor: actor, activeSkill: activeSkill, skillFlags: skillFlags, state: state, monsterLife: monsterLife, calculated: map[string]bool{}}
	offenceConversionTables(state, activeSkill.SkillModList, activeSkill.SkillCfg)
	offencePasses(env, actor, activeSkill, skillFlags, state)
	offenceHitSpeed(env, actor, activeSkill, skillFlags, state)
	offenceCombineSpeed(env, actor, activeSkill, skillFlags, state)
	state.quantityMultiplier = offenceMovementPenalty(actor, activeSkill)
	for _, pass := range state.passList {
		h.offenceDamagePass(pass, state.quantityMultiplier)
	}
	h.offenceCombineDamage()
	if h.actor.Breakdown != nil {
		h.totalDPSBreakdown()
	}
	h.offenceLeechRates()
	h.offenceAilments()
	h.offenceDots()
	h.offenceLinkCosts(hasCosts)
	h.offenceCombinedDPS()
	return state
}
