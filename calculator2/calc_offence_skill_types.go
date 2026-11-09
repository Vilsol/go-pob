package calculator2

import (
	"math"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/mod"
)

// offenceSkillTypeStats calculates skill type stats. Corpse life needs data.monsterVarietyLifeMult and
// data.mapLevelLifeMult, which are not built, so those outputs are not written.
//
//pob2:block:src/Modules/CalcOffence.lua:1417-1992
func offenceSkillTypeStats(env *Environment, actor *Actor, activeSkill *ActiveSkill, skillFlags map[string]bool, output map[string]float64) {
	skillModList := activeSkill.SkillModList
	skillCfg := activeSkill.SkillCfg
	skillData := activeSkill.SkillData
	enemyDB := actor.Enemy.ModDB
	sum := func(names ...string) float64 { return skillModList.Sum(mod.TypeBase, skillCfg, names...) }
	projectileBase := func() float64 {
		return sum("ProjectileCount") + 2*sum("TwoAdditionalProjectilesChance")/100 + sum("SurpassingProjectileChance")/100
	}
	tick := miscConstant("ServerTickRate")
	breakdown := actor.Breakdown

	if skillFlags["minion"] {
		if minion := activeSkill.Minion; minion != nil && minion.MinionData != nil && minion.MinionData.Limit != "" {
			if o := env.ModDB.Override(nil, minion.MinionData.Limit); o != nil {
				output["ActiveMinionLimit"] = math.Floor(o.Float())
			} else {
				output["ActiveMinionLimit"] = math.Floor(calcLibVal(skillModList, minion.MinionData.Limit, skillCfg) + calcLibVal(skillModList, "ActiveMinionLimit", skillCfg))
			}
		}
		output["SummonedMinionsPerCast"] = math.Floor(calcLibVal(skillModList, "MinionPerCastCount", skillCfg))
		if output["SummonedMinionsPerCast"] == 0 {
			output["SummonedMinionsPerCast"] = 1
		}
	}
	if skillFlags["chaining"] {
		if skillModList.Flag(skillCfg, "CannotChain") || skillModList.Flag(skillCfg, "NoAdditionalChains") {
			actor.setOutputString("ChainMaxString", "Cannot chain")
		} else {
			names := []string{"ChainCountMax"}
			if !skillFlags["projectile"] {
				names = append(names, "BeamChainCountMax")
			}
			output["ChainMax"] = (sum(names...) + sum("ChainChance")/100) * skillModList.More(skillCfg, names...)
			output["TerrainChain"] = math.Min(sum("TerrainChainChance"), 100)
			if skillModList.Flag(skillCfg, "AdditionalProjectilesAddChainsInstead") {
				output["ChainMax"] += float64((projectileBase() - 1) * skillModList.More(skillCfg, "ProjectileCount"))
			}
			output["ChainMaxString"] = output["ChainMax"]
			output["Chain"] = math.Min(output["ChainMax"], sum("ChainCount"))
			output["ChainRemaining"] = math.Max(0, output["ChainMax"]-output["Chain"])
		}
	}
	if skillFlags["projectile"] {
		if skillModList.Flag(nil, "PointBlank") {
			skillModList.AddMod(newMod("Damage", mod.TypeMore, 30, "Point Blank", mod.DistanceRamp([][]float64{{10, 1}, {35, 0}, {120, -1}})).Flag(ModFlagAttack | ModFlagProjectile))
		}
		if skillModList.Flag(nil, "FarShot") {
			skillModList.AddMod(newMod("Damage", mod.TypeMore, 100, "Far Shot", mod.DistanceRamp([][]float64{{10, -0.2}, {25, 0}, {70, 0.6}})).Flag(ModFlagAttack | ModFlagProjectile))
		}
		if skillModList.Flag(skillCfg, "NoAdditionalProjectiles") {
			output["ProjectileCount"] = 1
		} else {
			output["ProjectileCount"] = projectileBase() * skillModList.More(skillCfg, "ProjectileCount")
		}
		if skillModList.Flag(skillCfg, "AdditionalProjectilesAddBouncesInstead") {
			output["BounceCount"] = (projectileBase() + sum("BounceCount") - 1) * skillModList.More(skillCfg, "ProjectileCount")
		}
		if skillModList.Flag(skillCfg, "CannotSplit") || activeSkill.SkillTypes["ProjectileNumber"] {
			if env.buildBreakdown() {
				if splitCount := sum("SplitCount") + enemyDB.Sum(mod.TypeBase, skillCfg, "SelfSplitCount"); splitCount > 0 {
					actor.setOutputString("SplitCountString", "Cannot split")
				}
			}
		} else {
			output["SplitCount"] = sum("SplitCount") + enemyDB.Sum(mod.TypeBase, skillCfg, "SelfSplitCount")
			if skillModList.Flag(skillCfg, "AdditionalProjectilesAddSplitsInstead") {
				output["SplitCount"] += float64((projectileBase() - 1) * skillModList.More(skillCfg, "ProjectileCount"))
			}
			if skillModList.Flag(skillCfg, "AdditionalChainsAddSplitsInstead") {
				output["SplitCount"] += sum("ChainCountMax")
			}
			output["SplitCountString"] = output["SplitCount"]
		}
		if skillModList.Flag(skillCfg, "CannotFork") {
			actor.setOutputString("ForkCountString", "Cannot fork")
		} else if skillModList.Flag(skillCfg, "ForkOnce") {
			skillFlags["forking"] = true
			if skillModList.Flag(skillCfg, "ForkTwice") {
				output["ForkCountMax"] = math.Min(sum("ForkCountMax"), 2)
			} else {
				output["ForkCountMax"] = math.Min(sum("ForkCountMax"), 1)
			}
			output["ForkedCount"] = math.Min(output["ForkCountMax"], sum("ForkedCount"))
			output["ForkCountString"] = output["ForkCountMax"]
			output["ForkRemaining"] = math.Max(0, output["ForkCountMax"]-output["ForkedCount"])
		} else {
			actor.setOutputString("ForkCountString", "0")
		}
		if skillModList.Flag(skillCfg, "CannotPierce") {
			output["PierceCount"] = 0
			actor.setOutputString("PierceCountString", "Cannot pierce")
		} else {
			if skillModList.Flag(skillCfg, "PierceAllTargets") || enemyDB.Flag(nil, "AlwaysPierceSelf") {
				output["PierceCount"] = 100
				actor.setOutputString("PierceCountString", "All targets")
			} else {
				output["PierceCount"] = sum("PierceCount") + sum("PierceChance")/100
				output["PierceCountString"] = output["PierceCount"]
			}
			if output["PierceCount"] > 0 {
				skillFlags["piercing"] = true
			}
			output["PiercedCount"] = math.Min(output["PierceCount"], sum("PiercedCount"))
		}
		output["ProjectileSpeedMod"] = calcLibMod(skillModList, skillCfg, "ProjectileSpeed")
		//pob2:block:src/Modules/CalcOffence.lua:1509-1511
		if breakdown != nil {
			breakdown.setEntry("ProjectileSpeedMod", breakdown.mod(skillModList, skillCfg, "ProjectileSpeed"))
		}
		output["TwoAdditionalProjectiles"] = math.Min(sum("TwoAdditionalProjectilesChance"), 100)
	}
	if skillFlags["melee"] {
		meleeRange := func(i int) float64 {
			cfg := activeSkill.Weapon1Cfg
			if i == 2 {
				cfg = activeSkill.Weapon2Cfg
			}
			if r, ok := weaponData(env, actor, i)["range"]; ok {
				return r + skillModList.Sum(mod.TypeBase, cfg, "MeleeWeaponRange") + float64(10*skillModList.Sum(mod.TypeBase, cfg, "MeleeWeaponRangeMetre"))
			}
			return 6 + sum("UnarmedRange") + float64(10*sum("UnarmedRangeMetre"))
		}
		if skillFlags["weapon1Attack"] {
			actor.WeaponRange1 = meleeRange(1)
		}
		if skillFlags["weapon2Attack"] {
			actor.WeaponRange2 = meleeRange(2)
		}
		if activeSkill.SkillTypes["MeleeSingleTarget"] {
			rng := 100.0
			if skillFlags["weapon1Attack"] {
				rng = math.Min(rng, actor.WeaponRange1)
			}
			if skillFlags["weapon2Attack"] {
				rng = math.Min(rng, actor.WeaponRange2)
			}
			output["WeaponRange"] = rng + 2
			output["WeaponRangeMetre"] = output["WeaponRange"] / 10
			//pob2:block:src/Modules/CalcOffence.lua:1531-1535
			if breakdown != nil {
				breakdown.setEntry("WeaponRange", &BreakdownEntry{Radius: output["WeaponRange"], HasRadius: true})
			}

			baseStrikeCount := 1.0
			output["StrikeTargets"] = baseStrikeCount + sum("AdditionalStrikeTarget")
		}
	}
	if skillFlags["area"] || truthy(skillData["radius"]) || (skillFlags["mine"] && activeSkill.SkillTypes["Aura"]) {
		calcAreaOfEffect(actor, skillModList, skillCfg, skillData, skillFlags, output, actor.Breakdown)
	}
	if activeSkill.SkillTypes["Aura"] {
		names := []string{"AuraEffect"}
		if !truthy(skillData["auraCannotAffectSelf"]) {
			names = append(names, "SkillAuraEffectOnSelf")
		}
		output["AuraEffectMod"] = calcLibMod(skillModList, skillCfg, names...) * calcLibMod(skillModList, skillCfg, "Magnitude")
		//pob2:block:src/Modules/CalcOffence.lua:1546-1548
		if breakdown != nil {
			bdNames := []string{"AuraEffect", "Magnitude"}
			if !truthy(skillData["auraCannotAffectSelf"]) {
				bdNames = append(bdNames, "SkillAuraEffectOnSelf")
			}
			breakdown.setEntry("AuraEffectMod", breakdown.mod(skillModList, skillCfg, bdNames...))
		}
	}
	if activeSkill.SkillTypes["HasReservation"] && !activeSkill.SkillTypes["ReservationBecomesCost"] {
		for _, pool := range []string{"Spirit"} {
			output[pool+"ReservedMod"] = 0
			if calcLibMod(skillModList, skillCfg, "ReservationMultiplier") > 0 && calcLibMod(skillModList, skillCfg, pool+"Reserved", "Reserved") > 0 {
				output[pool+"ReservedMod"] = calcLibMod(skillModList, skillCfg, pool+"Reserved", "Reserved") * floorDec(calcLibMod(skillModList, skillCfg, "ReservationMultiplier"), 4) / math.Max(0, calcLibMod(skillModList, skillCfg, pool+"ReservationEfficiency", "ReservationEfficiency"))
			}
			//pob2:block:src/Modules/CalcOffence.lua:1556-1567
			if breakdown != nil {
				inc := skillModList.Sum(mod.TypeIncrease, skillCfg, pool+"Reserved", "Reserved", "ReservationMultiplier")
				more := skillModList.More(skillCfg, pool+"Reserved", "Reserved", "ReservationMultiplier")
				if inc != 0 && more != 1 {
					breakdown.set(pool+"ReservedMod",
						sFormat("%.2f ^8(increased/reduced)", 1+inc/100),
						sFormat("x %.2f ^8(more/less)", more),
						sFormat("/ %.2f ^8(reservation efficiency)", calcLibMod(skillModList, skillCfg, pool+"ReservationEfficiency", "ReservationEfficiency")),
						sFormat("= %.2f", output[pool+"ReservedMod"]))
				}
			}
		}
	}
	if activeSkill.SkillTypes["AppliesCurse"] {
		output["CurseEffectMod"] = calcLibMod(skillModList, skillCfg, "CurseEffect")
		//pob2:block:src/Modules/CalcOffence.lua:1572-1574
		if breakdown != nil {
			breakdown.setEntry("CurseEffectMod", breakdown.mod(skillModList, skillCfg, "CurseEffect"))
		}

		curseActivationMod := calcLibMod(skillModList, skillCfg, "CurseActivation")
		curseDelayMod := calcLibMod(skillModList, skillCfg, "CurseDelay")
		curseDelay, _ := sdNum(skillData, "curseDelay")
		output["CurseDelayBase"] = curseDelay + sum("CurseDelayBase")
		output["CurseDelay"] = output["CurseDelayBase"] / curseActivationMod * curseDelayMod
		output["CurseDelay"] = math.Ceil(output["CurseDelay"]*tick) / tick
		//pob2:block:src/Modules/CalcOffence.lua:1581-1593
		if breakdown != nil && output["CurseDelay"] != output["CurseDelayBase"] {
			breakdown.set("CurseDelay", sFormat("%.2fs ^8(base)", output["CurseDelayBase"]))
			if curseDelayMod != 1 {
				breakdown.insert("CurseDelay", sFormat("x %.4f ^8(delay modifier)", curseDelayMod))
			}
			if curseActivationMod != 1 {
				breakdown.insert("CurseDelay", sFormat("/ %.4f ^8(activation modifier)", curseActivationMod))
			}
			breakdown.insert("CurseDelay", "rounded up to nearest server tick", sFormat("= %.3fs", output["CurseDelay"]))
		}
	}
	if activeSkill.SkillTypes["Mark"] {
		output["MarkEffectMod"] = calcLibMod(skillModList, skillCfg, "MarkEffect")
		//pob2:block:src/Modules/CalcOffence.lua:1597-1599
		if breakdown != nil {
			breakdown.setEntry("MarkEffectMod", breakdown.mod(skillModList, skillCfg, "MarkEffect"))
		}
	}
	if activeSkill.SkillTypes["PerfectTiming"] {
		perfectTimingMod := calcLibMod(skillModList, skillCfg, "PerfectTiming")
		baseTiming := sum("PerfectTimingBase")
		output["PerfectTiming"] = baseTiming * perfectTimingMod
		//pob2:block:src/Modules/CalcOffence.lua:1605-1612
		if breakdown != nil {
			breakdown.set("PerfectTiming",
				sFormat("%.3fs ^8(Base Timing)", baseTiming),
				sFormat("x %.2f ^8(effect modifiers)", perfectTimingMod),
				"\n",
				sFormat("= %.3fs ^8(Perfect Timing Window)", output["PerfectTiming"]))
		}
	}
	if skillModList.Flag(skillCfg, "RevivingMinion") {
		minionRevivalSpeedMod := calcLibMod(skillModList, skillCfg, "MinionRevivalSpeed")
		baseMinionRevivalTime := miscConstant("MinionRevivalTimeBase")
		if o := skillModList.Override(skillCfg, "BaseReviveTime"); o != nil {
			baseMinionRevivalTime = o.Float()
		}
		output["MinionRevivalSpeed"] = baseMinionRevivalTime * (1 / minionRevivalSpeedMod)
		//pob2:block:src/Modules/CalcOffence.lua:1618-1625
		if breakdown != nil {
			breakdown.set("MinionRevivalSpeed",
				sFormat("%.3fs ^8(Base Revival Time)", baseMinionRevivalTime),
				sFormat("x 1 / %.2f ^8(effect modifiers)", minionRevivalSpeedMod),
				"\n",
				sFormat("= %.3fs ^8(Total Revival Time)", output["MinionRevivalSpeed"]))
		}
	}
	if activeSkill.SkillTypes["Warcry"] {
		fullDuration := calcSkillDuration(skillModList, skillCfg, skillData, env, enemyDB)
		actualCooldown, _, _, _, _ := calcSkillCooldown(skillModList, skillCfg, skillData)
		uptime := 1.0
		if !env.ModDB.Flag(nil, "Condition:WarcryMaxHit") {
			uptime = luaMin(fullDuration/actualCooldown, 1)
		}
		unscaledEffect := calcLibMod(skillModList, skillCfg, "WarcryEffect", "BuffEffect")
		output["WarcryEffectMod"] = unscaledEffect * uptime
		//pob2:block:src/Modules/CalcOffence.lua:1633-1645
		if breakdown != nil {
			breakdown.set("WarcryEffectMod", sFormat("%.2f ^8(effect modifiers)", unscaledEffect))
			if env.ModDB.Flag(nil, "Condition:WarcryMaxHit") || uptime != 1 {
				if env.ModDB.Flag(nil, "Condition:WarcryMaxHit") {
					breakdown.insert("WarcryEffectMod", "* 100% uptime (WarcryMaxHit Override)")
				} else if uptime != 1 {
					breakdown.insert("WarcryEffectMod", sFormat("* %.2f%% ^8(uptime)", uptime*100))
				}
				breakdown.insert("WarcryEffectMod", sFormat("= %.1f%%", output["WarcryEffectMod"]*100))
			}
		}
	}
	if skillModList.Flag(skillCfg, "CanCreateHazards") {
		output["HazardRearmChance"] = math.Min(sum("HazardRearmChance"), 100)
		skillModList.AddMod(newMod("DPS", mod.TypeMore, output["HazardRearmChance"], "Chance To Rearm"))
		// The oracle builds the breakdown, so the uncapped chance is what it shows.
		output["HazardRearmChance"] = sum("HazardRearmChance")
	}
	if activeSkill.SkillTypes["Link"] {
		output["LinkEffectMod"] = calcLibMod(skillModList, skillCfg, "LinkEffect", "BuffEffect")
		//pob2:block:src/Modules/CalcOffence.lua:1656-1658
		if breakdown != nil {
			breakdown.setEntry("LinkEffectMod", breakdown.mod(skillModList, skillCfg, "LinkEffect", "BuffEffect"))
		}
	}
	if activeSkill.SkillTypes["GeneratesRemnants"] {
		remnantEffectMod := calcLibMod(skillModList, skillCfg, "RemnantEffect")
		output["RemnantEffectMod"] = remnantEffectMod
		//pob2:block:src/Modules/CalcOffence.lua:1663-1665
		if breakdown != nil {
			breakdown.setEntry("RemnantEffectMod", breakdown.mod(skillModList, skillCfg, "RemnantEffect"))
		}
		// remnantBreakdown is the three-line breakdown each remnant gain writes, CalcOffence.lua:1669-1720
		remnantBreakdown := func(key, baseFormat string, base float64, totalFormat string) {
			if breakdown != nil {
				breakdown.set(key,
					sFormat(baseFormat, base),
					sFormat("x %.2f ^8(remnant effect modifier)", remnantEffectMod),
					sFormat(totalFormat, output[key]))
			}
		}
		if baseLifeGain := sum("LifeGainPerRemnant"); baseLifeGain > 0 {
			output["LifeGainPerRemnant"] = math.Floor(baseLifeGain * remnantEffectMod)
			remnantBreakdown("LifeGainPerRemnant", "%d ^8(base life per remnant)", baseLifeGain, "= %d ^8(life per remnant)")
		}
		if baseManaGain := sum("ManaGainPerRemnant"); baseManaGain > 0 {
			output["ManaGainPerRemnant"] = math.Floor(baseManaGain * remnantEffectMod)
			remnantBreakdown("ManaGainPerRemnant", "%d ^8(base mana per remnant)", baseManaGain, "= %d ^8(mana per remnant)")
		}
		baseChaosGainPerFlame := sum("BreachFlameChaosGain")
		doubled := 1.0
		if actor.ModDB.Flag(nil, "BreachFlameEffectDoubled") {
			doubled = 2
		}
		if baseChaosGainPerFlame > 0 {
			output["ChaosGainPerFlame"] = math.Floor(baseChaosGainPerFlame * remnantEffectMod * doubled)
			remnantBreakdown("ChaosGainPerFlame", "%d%% ^8(base chaos gain per flame)", baseChaosGainPerFlame, "= %d%% ^8(damage gained as chaos per flame)")
		}
		if baseLifeLeech := sum("BreachFlameLifeLeech"); baseLifeLeech > 0 {
			output["BreachFlameLifeLeech"] = math.Floor(baseLifeLeech * remnantEffectMod)
			remnantBreakdown("BreachFlameLifeLeech", "%d%% ^8(base life leech per flame)", baseLifeLeech, "= %d%% ^8(life leech per flame)")
		}
		if baseManaLeech := sum("BreachFlameManaLeech"); baseManaLeech > 0 {
			output["BreachFlameManaLeech"] = math.Floor(baseManaLeech * remnantEffectMod)
			remnantBreakdown("BreachFlameManaLeech", "%d%% ^8(base mana leech per flame)", baseManaLeech, "= %d%% ^8(mana leech per flame)")
		}
	}
	if activeSkill.SkillTypes["IceCrystal"] {
		iceCrystalLifeMod := calcLibMod(skillModList, skillCfg, "IceCrystalLife")
		baseIceCrystal := sum("IceCrystalLifeBase")
		output["IceCrystalLife"] = baseIceCrystal * iceCrystalLifeMod
		//pob2:block:src/Modules/CalcOffence.lua:1727-1734
		if breakdown != nil {
			breakdown.set("IceCrystalLife",
				sFormat("%.f ^8(Base Crystal Life)", baseIceCrystal),
				sFormat("x %.2f ^8(effect modifiers)", iceCrystalLifeMod),
				"\n",
				sFormat("= %.f ^8(Ice Crystal Life)", output["IceCrystalLife"]))
		}
	}
	_, hasTrapCooldown := sdNum(skillData, "trapCooldown")
	_, hasCooldown := sdNum(skillData, "cooldown")
	if (skillFlags["trap"] || skillFlags["mine"]) && !(hasTrapCooldown || hasCooldown) {
		skillFlags["notAverage"] = true
		skillFlags["showAverage"] = false
		skillData["showAverage"] = false
	}
	if skillFlags["trap"] {
		baseSpeed := 1 / sum("TrapThrowingTime")
		timeMod := calcLibMod(skillModList, skillCfg, "SkillTrapThrowingTime")
		if timeMod > 0 {
			baseSpeed *= 1 / timeMod
		}
		output["TrapThrowingSpeed"] = baseSpeed * calcLibMod(skillModList, skillCfg, "TrapThrowingSpeed") * output["ActionSpeedMod"]
		trapThrowCount := calcLibVal(skillModList, "TrapThrowCount", skillCfg)
		if hasTrapCooldown || hasCooldown {
			trapThrowCount = 1
		}
		output["TrapThrowCount"] = trapThrowCount
		if o := env.ModDB.Override(nil, "TrapThrowCount"); o != nil {
			output["TrapThrowCount"] = o.Float()
		}
		output["TrapThrowingSpeed"] = math.Min(output["TrapThrowingSpeed"], tick)
		output["TrapThrowingTime"] = 1 / output["TrapThrowingSpeed"]
		skillData["timeOverride"] = output["TrapThrowingTime"] / output["TrapThrowCount"]

		baseCooldown, hasBaseCooldown := sdNum(skillData, "trapCooldown")
		if !hasBaseCooldown {
			baseCooldown, hasBaseCooldown = sdNum(skillData, "cooldown")
		}
		if hasBaseCooldown || sum("CooldownRecovery") > 0 {
			if hasBaseCooldown {
				temporalisCooldown := sum("CooldownRecoveryFromTemporalis")
				output["TrapCooldown"] = (baseCooldown + temporalisCooldown) / calcLibMod(skillModList, skillCfg, "CooldownRecovery")
				if temporalisCooldown != 0 {
					output["TrapCooldown"] = math.Max(output["TrapCooldown"], 0.1)
				}
				output["TrapCooldown"] = math.Ceil(output["TrapCooldown"]*tick) / tick
			} else { // Assign Trap Cooldown if the trap/skill does not have cooldown but gain cooldown elsewhere
				cooldown, _, _, _, _ := calcSkillCooldown(skillModList, skillCfg, skillData)
				output["TrapCooldown"] = cooldown
			}
		}
		incArea, moreArea := calcLibMods(skillModList, skillCfg, "TrapTriggerAreaOfEffect")
		areaMod := calcAreaMod(incArea, moreArea)
		output["TrapTriggerRadius"] = calcRadius(miscConstant("TrapTriggerRadiusBase"), areaMod, 1)
		output["TrapTriggerRadiusMetre"] = output["TrapTriggerRadius"] / 10
	} else if hasCooldown || sum("CooldownRecovery") > 0 {
		cooldownMode, ok := configListValue(env, "cooldownMode")
		if !ok {
			cooldownMode = "BASE"
		}
		cooldown, rounded, addedCooldown, _, noCooldownChance := calcSkillCooldown(skillModList, skillCfg, skillData)
		effectiveCooldownMultiplier := 1 - noCooldownChance
		effectiveCooldown := cooldown * effectiveCooldownMultiplier

		output["Cooldown"] = cooldown
		output["EffectiveCooldown"] = cooldown
		// The oracle builds the breakdown, where the average cooldown is taken.
		if cooldownMode == "AVERAGE" && noCooldownChance > 0 {
			output["EffectiveCooldown"] = effectiveCooldown
		}
		//pob2:block:src/Modules/CalcOffence.lua:1816-1836
		if breakdown != nil {
			// skillData.cooldown or 0 + addedCooldown is skillData.cooldown or (0 + addedCooldown)
			baseCooldown, ok := sdNum(skillData, "cooldown")
			if !ok {
				baseCooldown = addedCooldown
			}
			breakdown.set("Cooldown",
				sFormat("%.2fs ^8(base)", baseCooldown),
				sFormat("/ %.2f ^8(increased/reduced cooldown recovery)", 1+skillModList.Sum(mod.TypeIncrease, skillCfg, "CooldownRecovery")/100))
			if cooldownMode == "AVERAGE" && noCooldownChance > 0 {
				lines := append([]string{"Effective Cooldown:"}, breakdown.Entries["Cooldown"].Lines...)
				lines = append(lines,
					sFormat("* %.2f ^8(effect of %d%% chance to not consume cooldown)", effectiveCooldownMultiplier, noCooldownChance*100),
					sFormat("= %.3fs", output["EffectiveCooldown"]))
				breakdown.set("EffectiveCooldown", lines...)
			}
			if rounded {
				breakdown.insert("Cooldown", "rounded up to nearest server tick")
			}
			breakdown.insert("Cooldown", sFormat("= %.3fs", output["Cooldown"]))
		}
	}
	if baseUses, ok := sdNum(skillData, "storedUses"); ok {
		additionalUses := sum("AdditionalCooldownUses")
		output["StoredUses"] = baseUses + additionalUses
		//pob2:block:src/Modules/CalcOffence.lua:1842-1848
		if breakdown != nil {
			plural := func(n float64) string {
				if n == 1 {
					return ""
				}
				return "s"
			}
			breakdown.set("StoredUses", sFormat("%d ^8(skill use%s)", baseUses, plural(baseUses)))
			if additionalUses != 0 {
				breakdown.insert("StoredUses",
					sFormat("+ %d ^8(additional use%s)", additionalUses, plural(additionalUses)),
					sFormat("= %d ^8(total use%s)", output["StoredUses"], plural(output["StoredUses"])))
			}
		}
	}
	if skillFlags["mine"] {
		baseSpeed := 1 / sum("MineLayingTime")
		timeMod := calcLibMod(skillModList, skillCfg, "SkillMineThrowingTime")
		if timeMod > 0 {
			baseSpeed *= 1 / timeMod
		}
		output["MineLayingSpeed"] = baseSpeed * calcLibMod(skillModList, skillCfg, "MineLayingSpeed") * output["ActionSpeedMod"]
		// Calculate additional mine throw
		mineThrowCount := calcLibVal(skillModList, "MineThrowCount", skillCfg)
		if hasTrapCooldown || hasCooldown {
			mineThrowCount = 1
		}
		output["MineThrowCount"] = mineThrowCount
		if o := env.ModDB.Override(nil, "MineThrowCount"); o != nil {
			output["MineThrowCount"] = o.Float()
		}
		if output["MineThrowCount"] >= 1 {
			// Throwing Mines takes 10% more time for each *additional* Mine thrown
			output["MineLayingSpeed"] /= 1 + float64((output["MineThrowCount"]-1)*0.1)
		}

		output["MineLayingSpeed"] = math.Min(output["MineLayingSpeed"], tick)
		output["MineLayingTime"] = 1 / output["MineLayingSpeed"]

		// Trap mine interaction where the Character throws mines, mine throws traps
		if skillFlags["trap"] {
			skillData["timeOverride"] = output["MineLayingTime"] / output["MineThrowCount"] / output["TrapThrowCount"]
		} else {
			skillData["timeOverride"] = output["MineLayingTime"] / output["MineThrowCount"]
		}

		incArea, moreArea := calcLibMods(skillModList, skillCfg, "MineDetonationAreaOfEffect")
		areaMod := calcAreaMod(incArea, moreArea)
		output["MineDetonationRadius"] = calcRadius(miscConstant("MineDetonationRadiusBase"), areaMod, 1)
		output["MineDetonationRadiusMetre"] = output["MineDetonationRadius"] / 10
		if activeSkill.SkillTypes["Aura"] {
			output["MineAuraRadius"] = calcRadius(miscConstant("MineAuraRadiusBase"), output["AreaOfEffectMod"], 1)
			output["MineAuraRadiusMetre"] = output["MineAuraRadius"] / 10
		}
	}
	if skillFlags["totem"] {
		var baseSpeed float64
		if skillFlags["ballista"] {
			baseSpeed = 1 / sum("BallistaPlacementTime")
		} else {
			baseSpeed = 1 / sum("TotemPlacementTime")
		}
		output["TotemPlacementSpeed"] = baseSpeed * calcLibMod(skillModList, skillCfg, "TotemPlacementSpeed") * output["ActionSpeedMod"]
		output["TotemPlacementTime"] = 1 / output["TotemPlacementSpeed"]
		//pob2:block:src/Modules/CalcOffence.lua:1926-1936
		if breakdown != nil {
			lines, _ := breakdown.multiChain(nil, multiChainSpec{
				Label: "Placement speed:",
				Base:  &chainLink{Format: "%.2f ^8(base placement speed)", Value: &baseSpeed},
				Links: []chainLink{
					link("%.2f ^8(increased/reduced placement speed)", 1+skillModList.Sum(mod.TypeIncrease, skillCfg, "TotemPlacementSpeed")/100),
					link("%.2f ^8(more/less placement speed)", skillModList.More(skillCfg, "TotemPlacementSpeed")),
					link("%.2f ^8(action speed modifier)", output["ActionSpeedMod"]),
				},
				Total: sFormat("= %.2f ^8per second", output["TotemPlacementSpeed"]),
			})
			breakdown.set("TotemPlacementTime", lines...)
		}
		if o := skillModList.Override(skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit"); o != nil {
			output["ActiveTotemLimit"] = o.Float()
		} else {
			output["ActiveTotemLimit"] = sum("ActiveTotemLimit", "ActiveBallistaLimit")
		}
		output["TotemsSummoned"] = output["ActiveTotemLimit"]
		if o := env.ModDB.Override(nil, "TotemsSummoned"); o != nil {
			output["TotemsSummoned"] = o.Float()
		}
		//pob2:block:src/Modules/CalcOffence.lua:1939-1943
		if breakdown != nil {
			note := " ^8(can be overridden in the Configuration tab)"
			if _, ok := env.ConfigInput["TotemsSummoned"]; ok {
				note = " ^8(overridden from the Configuration tab)"
			}
			breakdown.set("ActiveTotemLimit", "Totems Summoned: "+luaToString(output["TotemsSummoned"])+note)
		}
		output["TotemLifeMod"] = calcLibMod(skillModList, skillCfg, "TotemLife")
		// Lua indexes nil and raises for a totem level or totem id outside the tables; Go writes nothing.
		if totemLevel, _ := sdNum(skillData, "totemLevel"); totemLevel >= 1 && int(totemLevel) <= len(poe2.MonsterAllyLifeTable) {
			if mult, ok := poe2.TotemLifeMult[activeSkill.SkillTotemID]; ok {
				output["TotemLife"] = round(math.Floor(poe2.MonsterAllyLifeTable[int(totemLevel)-1]*mult) * output["TotemLifeMod"])
			}
		}
		output["TotemEnergyShield"] = sum("TotemEnergyShield")
		output["TotemBlockChance"] = sum("TotemBlockChance")
		output["TotemArmour"] = sum("TotemArmour")
		//pob2:block:src/Modules/CalcOffence.lua:1949-1961
		if breakdown != nil {
			breakdown.setEntry("TotemLifeMod", breakdown.mod(skillModList, skillCfg, "TotemLife"))
			if totemLevel, _ := sdNum(skillData, "totemLevel"); totemLevel >= 1 && int(totemLevel) <= len(poe2.MonsterAllyLifeTable) {
				if mult, ok := poe2.TotemLifeMult[activeSkill.SkillTotemID]; ok {
					breakdown.set("TotemLife",
						"Totem level: "+luaToString(totemLevel),
						luaToString(poe2.MonsterAllyLifeTable[int(totemLevel)-1])+" ^8(base life for a level "+luaToString(totemLevel)+" monster)",
						"x "+luaToString(mult)+" ^8(life multiplier for this totem type)",
						"x "+luaToString(output["TotemLifeMod"])+" ^8(totem life modifier)",
						"= "+luaToString(output["TotemLife"]))
				}
			}
			breakdown.setEntry("TotemEnergyShield", breakdown.mod(skillModList, skillCfg, "TotemEnergyShield"))
			breakdown.setEntry("TotemBlockChance", breakdown.mod(skillModList, skillCfg, "TotemBlockChance"))
			breakdown.setEntry("TotemArmour", breakdown.mod(skillModList, skillCfg, "TotemArmour"))
		}
	}
	if skillFlags["brand"] {
		output["BrandAttachmentRange"] = miscConstant("BrandAttachmentRangeBase") * calcLibMod(skillModList, skillCfg, "BrandAttachmentRange")
		output["BrandAttachmentRangeMetre"] = output["BrandAttachmentRange"] / 10
		output["ActiveBrandLimit"] = sum("ActiveBrandLimit")
	}

	if skillFlags["warcry"] {
		output["WarcryCastTime"] = calcWarcryCastTime(skillModList, skillCfg, skillData, actor)
	}
}

//pob2:block:src/Modules/CalcOffence.lua:1994-2191
func offenceDurations(env *Environment, activeSkill *ActiveSkill, skillFlags map[string]bool, output map[string]float64, breakdown *Breakdown) {
	skillModList := activeSkill.SkillModList
	skillCfg := activeSkill.SkillCfg
	skillData := activeSkill.SkillData
	enemyDB := activeSkill.Actor.Enemy.ModDB
	tick := miscConstant("ServerTickRate")
	sd := func(key string) float64 { v, _ := sdNum(skillData, key); return v }

	// Skill duration
	debuffDurationMult := 1.0
	if env.ModeEffective {
		debuffDurationMult = 1 / math.Max(miscConstant("BuffExpirationSlowCap"), calcLibMod(enemyDB, skillCfg, "BuffExpireFaster"))
	}
	output["DurationMod"] = calcLibMod(skillModList, skillCfg, durationModNames(skillData, "Duration", "PrimaryDuration", "DamagingAilmentDuration")...)
	output["DurationMod"] = math.Max(output["DurationMod"], 0)
	//pob2:block:src/Modules/CalcOffence.lua:2002-2007
	if breakdown != nil {
		e := breakdown.mod(skillModList, skillCfg, durationModNames(skillData, "Duration", "PrimaryDuration", "DamagingAilmentDuration")...)
		if _, ok := sdNum(skillData, "durationSecondary"); e != nil && ok {
			e.Lines = append([]string{"Primary duration:"}, e.Lines...)
		}
		breakdown.setEntry("DurationMod", e)
	}
	durationBase := sd("duration") + skillModList.Sum(mod.TypeBase, skillCfg, "Duration", "PrimaryDuration")
	// activeSkill.minion is not built, so its PermanentDuration flag is not checked.
	if durationBase > 0 {
		output["Duration"] = durationBase * output["DurationMod"]
		if truthy(skillData["debuff"]) {
			output["Duration"] *= debuffDurationMult
		}
		output["Duration"] = math.Ceil(output["Duration"]*tick) / tick
		//pob2:block:src/Modules/CalcOffence.lua:2015-2027
		if breakdown != nil && output["Duration"] != durationBase {
			breakdown.set("Duration", sFormat("%.2fs ^8(base)", durationBase))
			if output["DurationMod"] != 1 {
				breakdown.insert("Duration", sFormat("x %.4f ^8(duration modifier)", output["DurationMod"]))
			}
			if truthy(skillData["debuff"]) && debuffDurationMult != 1 {
				breakdown.insert("Duration", sFormat("/ %.3f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
			}
			breakdown.insert("Duration", "rounded up to nearest server tick", sFormat("= %.3fs", output["Duration"]))
		}
	}
	for _, d := range []struct{ key, data, name, debuff, modKey, label string }{
		{"DurationSecondary", "durationSecondary", "SecondaryDuration", "debuffSecondary", "SecondaryDurationMod", "Secondary duration:"},
		{"DurationTertiary", "durationTertiary", "TertiaryDuration", "debuffTertiary", "TertiaryDurationMod", "Tertiary duration:"},
	} {
		durationBase = sd(d.data) + skillModList.Sum(mod.TypeBase, skillCfg, "Duration", d.name)
		if durationBase > 0 {
			durationMod := calcLibMod(skillModList, skillCfg, durationModNames(skillData, "Duration", d.name, "DamagingAilmentDuration")...)
			durationMod = math.Max(durationMod, 0)
			output[d.key] = durationBase * durationMod
			if truthy(skillData[d.debuff]) {
				output[d.key] *= debuffDurationMult
			}
			output[d.key] = math.Ceil(output[d.key]*tick) / tick
			//pob2:block:src/Modules/CalcOffence.lua:2038-2054
			if breakdown != nil && output[d.key] != durationBase {
				e := breakdown.mod(skillModList, skillCfg, durationModNames(skillData, "Duration", d.name, "DamagingAilmentDuration")...)
				if e != nil {
					e.Lines = append([]string{d.label}, e.Lines...)
				}
				breakdown.setEntry(d.modKey, e)
				breakdown.set(d.key, sFormat("%.2fs ^8(base)", durationBase))
				// the primary DurationMod gates the line, as the Lua does
				if output["DurationMod"] != 1 {
					breakdown.insert(d.key, sFormat("x %.4f ^8(duration modifier)", durationMod))
				}
				if truthy(skillData[d.debuff]) && debuffDurationMult != 1 {
					breakdown.insert(d.key, sFormat("/ %.3f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
				}
				breakdown.insert(d.key, "rounded up to nearest server tick", sFormat("= %.3fs", output[d.key]))
			}
		}
	}
	for _, d := range []struct{ key, data string }{{"AuraDuration", "auraDuration"}, {"ReserveDuration", "reserveDuration"}} {
		durationBase = sd(d.data)
		if durationBase > 0 {
			durationMod := calcLibMod(skillModList, skillCfg, "Duration", "DamagingAilmentDuration")
			durationMod = math.Max(durationMod, 0)
			output[d.key] = durationBase * durationMod
			output[d.key] = math.Ceil(output[d.key]*tick) / tick
			//pob2:block:src/Modules/CalcOffence.lua:2089-2096
			if breakdown != nil && output[d.key] != durationBase {
				breakdown.set(d.key,
					sFormat("%.2fs ^8(base)", durationBase),
					sFormat("x %.4f ^8(duration modifier)", durationMod),
					"rounded up to nearest server tick",
					sFormat("= %.3fs", output[d.key]))
			}
		}
	}
	durationBase = sd("soulPreventionDuration")
	if durationBase > 0 {
		second := "DamagingAilmentDuration"
		if truthy(skillData["skillEffectAppliesToSoulGainPrevention"]) {
			second = "Duration"
		}
		durationMod := calcLibMod(skillModList, skillCfg, durationModNames(skillData, "SoulGainPreventionDuration", second)...)
		durationMod = math.Max(durationMod, 0)
		output["SoulGainPreventionDuration"] = durationBase * durationMod
		output["SoulGainPreventionDuration"] = math.Max(math.Ceil(output["SoulGainPreventionDuration"]*tick), 1) / tick
		//pob2:block:src/Modules/CalcOffence.lua:2119-2126
		if breakdown != nil && output["SoulGainPreventionDuration"] != durationBase {
			breakdown.set("SoulGainPreventionDuration",
				sFormat("%.2fs ^8(base)", durationBase),
				sFormat("x %.4f ^8(duration modifier)", durationMod),
				"rounded up to nearest server tick",
				sFormat("= %.3fs", output["SoulGainPreventionDuration"]))
		}
	}
	output["TotemDurationMod"] = calcLibMod(skillModList, skillCfg, "TotemDuration")
	output["TotemDurationMod"] = math.Max(output["TotemDurationMod"], 0)
	totemDurationBase := skillModList.Sum(mod.TypeBase, skillCfg, "TotemDuration")
	output["TotemDuration"] = math.Ceil(totemDurationBase*output["TotemDurationMod"]*tick) / tick
	//pob2:block:src/Modules/CalcOffence.lua:2132-2142
	if breakdown != nil {
		breakdown.setEntry("TotemDurationMod", breakdown.mod(skillModList, skillCfg, "TotemDuration"))
		breakdown.set("TotemDuration", sFormat("%.2fs ^8(base)", totemDurationBase))
		if output["TotemDurationMod"] != 1 {
			breakdown.insert("TotemDuration", sFormat("x %.4f ^8(duration modifier)", output["TotemDurationMod"]))
		}
		breakdown.insert("TotemDuration", sFormat("rounded up to nearest server tick"), sFormat("= %.3fs", output["TotemDuration"]))
	}

	// Skill uptime
	if !activeSkill.SkillTypes["Vaal"] { // exclude vaal skills as we currently don't support soul generation or gain prevention.
		cooldown := output["Cooldown"]
		for _, durationType := range []string{"Duration", "DurationSecondary", "DurationTertiary", "AuraDuration", "reserveDuration"} {
			duration := output[durationType]
			if duration != 0 && cooldown != 0 {
				var uptime float64
				if skillModList.Flag(skillCfg, "NoCooldownRecoveryInDuration") {
					uptime = duration / (cooldown + duration)
				} else {
					uptime = duration / cooldown
				}
				uptime = math.Min(uptime, 1)
				output[durationType+"Uptime"] = uptime * 100
				//pob2:block:src/Modules/CalcOffence.lua:2160-2172
				if breakdown != nil {
					if skillModList.Flag(skillCfg, "NoCooldownRecoveryInDuration") {
						breakdown.set(durationType+"Uptime", sFormat("%.2fs / (%.2fs + %.2fs)", duration, cooldown, duration), sFormat("= %d%%", output[durationType+"Uptime"]))
					} else {
						breakdown.set(durationType+"Uptime", sFormat("%.2fs / %.2fs", duration, cooldown), sFormat("= %d%%", output[durationType+"Uptime"]))
					}
				}
			}
		}
	}

	if activeSkill.SkillTypes["DetonatesAfterTime"] {
		base := activeSkill.SkillModList.Sum(mod.TypeBase, skillCfg, "DetonationTime")
		inc, more := calcLibMods(skillModList, skillCfg, "DetonationTime")
		output["DetonationTime"] = base * inc * more
		//pob2:block:src/Modules/CalcOffence.lua:2182-2190
		if breakdown != nil {
			lines, _ := breakdown.multiChain(nil, multiChainSpec{
				Base:  &chainLink{Format: "%.2fs ^8(base)", Value: &base},
				Links: []chainLink{link("%.2f ^8(increased/reduced detonation time)", inc), link("%.2f ^8(more/less detonation time)", more)},
				Total: sFormat("= %.2fs", output["DetonationTime"]),
			})
			breakdown.set("DetonationTime", lines...)
		}
	}
}

// costOrder is costs.order; costUpfront marks its upfront resources.
var costOrder = []string{"Mana", "Life", "ES", "Ward", "Soul", "Rage", "ManaPercent", "LifePercent", "WardPercent", "ManaPerMinute", "LifePerMinute", "WardPerMinute", "ManaPercentPerMinute", "LifePercentPerMinute", "ESPerMinute", "ESPercentPerMinute"}

var costUpfront = map[string]bool{"Mana": true, "Life": true, "ES": true, "Ward": true, "Soul": true, "Rage": true, "ManaPercent": true, "LifePercent": true, "WardPercent": true}

type skillCost struct {
	typ, text                                              string
	baseCost, baseCostRaw, totalCost, baseCostNoMult       float64
	finalBaseCost                                          float64
	upfront, percent, unaffectedByGenericCostMults, hasRaw bool
}

// offenceCosts calculates costs (may be slightly off due to rounding differences). It returns
// output[resource.."HasCost"], which is also in Actor.OutputFlags.
//
//pob2:block:src/Modules/CalcOffence.lua:2193-2395
func offenceCosts(env *Environment, actor *Actor, activeSkill *ActiveSkill, output map[string]float64) map[string]bool {
	skillModList := activeSkill.SkillModList
	skillCfg := activeSkill.SkillCfg
	order := costOrder
	costs := map[string]*skillCost{
		"Mana":                 {typ: "Mana", upfront: true, text: "mana", hasRaw: true},
		"Life":                 {typ: "Life", upfront: true, text: "life"},
		"ES":                   {typ: "ES", upfront: true, text: "ES"},
		"Ward":                 {typ: "Ward", upfront: true, text: "runic ward"},
		"Soul":                 {typ: "Soul", upfront: true, unaffectedByGenericCostMults: true, text: "soul"},
		"Rage":                 {typ: "Rage", upfront: true, text: "rage"},
		"ManaPercent":          {typ: "Mana", upfront: true, percent: true, text: "mana"},
		"LifePercent":          {typ: "Life", upfront: true, percent: true, text: "life"},
		"WardPercent":          {typ: "Ward", upfront: true, percent: true, text: "runic ward"},
		"ManaPerMinute":        {typ: "Mana", text: "mana/s"},
		"LifePerMinute":        {typ: "Life", text: "life/s"},
		"WardPerMinute":        {typ: "Ward", text: "runic ward/s"},
		"ManaPercentPerMinute": {typ: "Mana", percent: true, text: "mana/s"},
		"LifePercentPerMinute": {typ: "Life", percent: true, text: "life/s"},
		"ESPerMinute":          {typ: "ES", text: "ES/s"},
		"ESPercentPerMinute":   {typ: "ES", percent: true, text: "ES/s"},
	}
	hasCosts := map[string]bool{}
	if skillModList.Flag(skillCfg, "HasNoCost") {
		return hasCosts
	}
	sum := func(names ...string) float64 { return skillModList.Sum(mod.TypeBase, skillCfg, names...) }
	//Support cost multipliers are calculated first and rounded down after 4 digits
	mult := floorDec(skillModList.More(skillCfg, "SupportManaMultiplier"), 4)
	// First pass to calculate base costs. Used for cost conversion (e.g. Petrified Blood)
	additionalLifeCost := sum("BaseManaCostAsLifeCost") / 100                // Extra cost (e.g. Petrified Blood) calculations
	additionalESCost := sum("ManaCostAsEnergyShieldCost") / 100              // Extra cost (e.g. Replica Covenant) calculations
	hybridLifeCost := math.Min(sum("HybridManaAndLifeCost_Life"), 100) / 100 // Blood Magic, Lifetap and tree mods capped at 100
	wardCostPctOfLife := sum("WardCostAsPercentOfLifeCost") / 100            // Extra cost (e.g. Scouring Flame) calculations
	wardCostPctOfMana := sum("WardCostAsPercentOfManaCost") / 100
	levelCost, _ := grantedEffectLevelCost(activeSkill.ActiveEffect)
	var levelExtra map[string]float64
	if activeSkill.ActiveEffect.GrantedEffectLevel != nil {
		levelExtra = activeSkill.ActiveEffect.GrantedEffectLevel.Extra
	}
	for _, resource := range order {
		val := costs[resource]
		skillCost, hasSkillCost := levelCost[resource]
		if o := skillModList.Override(skillCfg, "Base"+resource+"CostOverride"); o != nil {
			skillCost, hasSkillCost = o.Float(), true
		}
		baseCost := 0.0
		if hasSkillCost {
			baseCost = round(skillCost/costDivisor(resource), 2)
		}
		baseCostNoMult := sum(resource + "CostNoMult") // Flat cost from gem e.g. Divine Blessing
		divineBlessingCorrection := 0.0
		if val.upfront {
			baseCost += sum(resource + "CostBase") // Rage Cost or Cost an additional x% max Life/ES
			val.totalCost = sum(resource+"Cost", "Cost")
			if resource == "Mana" && activeSkill.SkillTypes["ReservationBecomesCost"] && !val.percent { //Divine Blessing / Totem auras
				reservedFlat, ok := sdNum(activeSkill.SkillData, val.text+"ReservationFlat")
				if !ok {
					reservedFlat = levelExtra[val.text+"ReservationFlat"]
				}
				baseCost += reservedFlat
				reservedPercent, ok := sdNum(activeSkill.SkillData, val.text+"ReservationPercent")
				if !ok {
					reservedPercent = levelExtra[val.text+"ReservationPercent"]
				}
				baseCost += math.Floor(output[resource] * reservedPercent / 100)
				//Divine Blessing / Totem aura skills that have a percent reservation, round instead of floor the value. This corrects the final result if it would round up
				divineBlessingCorrection = round(output[resource]*reservedPercent/100*mult) - math.Floor(output[resource]*reservedPercent/100*mult)
			}
		}
		val.baseCost += baseCost
		val.baseCostNoMult += baseCostNoMult
		finalBaseCostRaw := float64(val.baseCost*mult) + val.baseCostNoMult + divineBlessingCorrection
		val.finalBaseCost = math.Floor(finalBaseCostRaw)
		if val.hasRaw {
			val.baseCostRaw = float64(val.baseCost*mult) + val.baseCostNoMult + divineBlessingCorrection
		}
		switch {
		case val.typ == "Life":
			manaType := strings.Replace(resource, "Life", "Mana", 1)
			manaCost := costs[manaType]
			if skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") { // Blood Magic / Lifetap in PoE 1, not used in PoE 2 yet
				val.baseCost += manaCost.baseCost
				val.baseCostNoMult += manaCost.baseCostNoMult
				val.finalBaseCost += manaCost.finalBaseCost
				if val.upfront {
					manaTotalCost := sum(manaType + "Cost")
					val.totalCost += math.Max(0, manaTotalCost)
					manaCost.totalCost -= manaTotalCost
				}
				manaCost.baseCost = 0
				if manaCost.hasRaw {
					manaCost.baseCostRaw = 0
				}
				manaCost.finalBaseCost = 0
				manaCost.baseCostNoMult = 0
			} else if additionalLifeCost > 0 || hybridLifeCost > 0 {
				val.baseCost = manaCost.baseCost
				val.finalBaseCost = round(finalBaseCostRaw + round(manaCost.finalBaseCost*hybridLifeCost) + float64(math.Floor(val.baseCost*mult)*additionalLifeCost))
				if val.upfront && hybridLifeCost > 0 {
					// Positive flat Mana cost is converted and rounded separately from the base cost.
					val.totalCost += round(math.Max(0, sum(manaType+"Cost")) * hybridLifeCost)
				}
			}
		case val.typ == "ES":
			manaCost := costs[strings.Replace(resource, "ES", "Mana", 1)]
			if additionalESCost > 0 {
				val.baseCost = manaCost.baseCost
				val.finalBaseCost = round(finalBaseCostRaw + round(manaCost.finalBaseCost*additionalESCost))
			}
		case val.typ == "Ward" && resource == "Ward":
			wardCostFromOtherCosts := 0.0
			if wardCostPctOfMana != 0 {
				wardCostFromOtherCosts += round(costs["Mana"].finalBaseCost * wardCostPctOfMana)
			}
			if wardCostPctOfLife != 0 {
				wardCostFromOtherCosts += round(costs["Life"].finalBaseCost * wardCostPctOfLife)
			}
			if wardCostFromOtherCosts > 0 {
				val.baseCost += wardCostFromOtherCosts
				val.finalBaseCost += wardCostFromOtherCosts
			}
		case val.typ == "Rage":
			if skillModList.Flag(skillCfg, "CostRageInsteadOfSouls") { // Hateforge
				val.baseCost = costs["Soul"].baseCost
				val.baseCostNoMult += costs["Soul"].baseCostNoMult
				val.finalBaseCost = costs["Soul"].baseCost
				mult = 1
				costs["Soul"].baseCost = 0
				costs["Soul"].baseCostNoMult = 0
				costs["Soul"].finalBaseCost = 0
			}
		}
	}
	for _, costResource := range order {
		val := costs[costResource]
		resource := costResource
		if !val.upfront {
			resource = strings.Replace(resource, "Minute", "Second", 1)
		}
		hasCost := val.baseCost > 0 || val.totalCost > 0 || val.baseCostNoMult > 0 || val.finalBaseCost > 0
		hasCosts[resource] = hasCost
		actor.setOutputFlag(resource+"HasCost", hasCost)
		costName := resource + "Cost"
		costNameRaw := costName + "Raw"
		var moreType, inc float64
		moreCost := 1.0
		costEfficiency := calcLibMod(skillModList, skillCfg, val.typ+"CostEfficiency", "CostEfficiency")
		if !val.unaffectedByGenericCostMults {
			output[costName] = val.finalBaseCost
			moreType = skillModList.More(skillCfg, val.typ+"Cost")
			moreCost = skillModList.More(skillCfg, "Cost")
			inc = skillModList.Sum(mod.TypeIncrease, skillCfg, val.typ+"Cost", "Cost")
			if val.hasRaw {
				output[costNameRaw] = math.Max(0, math.Max(0, (1+inc/100)*val.baseCostRaw*moreType*moreCost/costEfficiency)+val.totalCost)
			}
			if inc < 0 {
				output[costName] = math.Max(0, math.Ceil((1+inc/100)*output[costName]))
			} else {
				output[costName] = math.Max(0, math.Floor((1+inc/100)*output[costName]))
			}
			if moreType < 1 {
				output[costName] = math.Max(0, math.Ceil(moreType*output[costName]))
			} else {
				output[costName] = math.Max(0, math.Floor(moreType*output[costName]))
			}
			if moreCost < 1 {
				output[costName] = math.Max(0, math.Ceil(moreCost*output[costName]))
			} else {
				output[costName] = math.Max(0, math.Floor(moreCost*output[costName]))
			}
			// Apply cost efficiency (similar to reservation efficiency)
			output[costName] = math.Max(0, output[costName]/costEfficiency)
			if val.upfront && !val.percent {
				// The game stores upfront resource costs as whole numbers after efficiency.
				output[costName] = round(output[costName])
			}
			output[costName] = math.Max(0, output[costName]+val.totalCost)
			if val.typ == "Mana" && hybridLifeCost > 0 { // Life/Mana Mastery
				output[costName] = math.Max(0, math.Floor((1-hybridLifeCost)*output[costName]))
				if _, ok := output[costNameRaw]; ok {
					output[costNameRaw] = math.Max(0, (1-hybridLifeCost)*output[costNameRaw])
				}
			}
		} else {
			moreType = skillModList.More(skillCfg, val.typ+"Cost")
			inc = skillModList.Sum(mod.TypeIncrease, skillCfg, val.typ+"Cost")
			output[costName] = math.Floor(val.baseCost + val.baseCostNoMult)
			output[costName] = math.Max(0, (1+inc/100)*output[costName])
			output[costName] = math.Max(0, moreType*output[costName])
			// Apply cost efficiency for unaffected costs too
			output[costName] = math.Max(0, output[costName]/costEfficiency)
			if val.upfront && !val.percent {
				// The game stores upfront resource costs as whole numbers after efficiency.
				output[costName] = round(output[costName])
			}
			output[costName] = math.Max(0, output[costName]+val.totalCost)
			if val.hasRaw {
				output[costNameRaw] = math.Max(0, math.Max(0, (1+inc/100)*(val.baseCostRaw+val.baseCostNoMult)*moreType/costEfficiency)+val.totalCost)
			}
		}
		//pob2:block:src/Modules/CalcOffence.lua:2358-2393
		if breakdown := actor.Breakdown; breakdown != nil && hasCost {
			pct := ""
			if val.percent {
				pct = "%%"
			}
			lines := []string{sFormat("%.2f"+pct+" ^8(base "+val.text+" cost)", val.baseCost)}
			if mult != 1 {
				lines = append(lines, sFormat("x %.4f ^8(cost multiplier)", mult))
			}
			if val.baseCostNoMult != 0 {
				lines = append(lines, sFormat("+ %d ^8(additional "+val.text+" cost)", val.baseCostNoMult))
			}
			if val.typ == "Life" && (hybridLifeCost+additionalLifeCost) != 0 && !skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") {
				lines = append(lines, sFormat("x %.2f ^8(mana cost conversion)", hybridLifeCost+additionalLifeCost))
			}
			if val.typ == "ES" && additionalESCost != 0 && !skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") {
				lines = append(lines, sFormat("x %.2f ^8(mana cost conversion)", additionalESCost))
			}
			if inc != 0 {
				lines = append(lines, sFormat("x %.2f ^8(increased/reduced "+val.text+" cost)", 1+inc/100))
			}
			if moreCost != 1 {
				lines = append(lines, sFormat("x %.2f ^8(more/less cost)", moreCost))
			}
			if moreType != 1 {
				lines = append(lines, sFormat("x %.2f ^8(more/less "+val.text+" cost)", moreType))
			}
			if costEfficiency != 1 {
				lines = append(lines, sFormat("/ %.2f ^8("+val.text+" cost efficiency)", costEfficiency))
			}
			if val.totalCost != 0 {
				lines = append(lines, sFormat("%+d ^8(total "+val.text+" cost)", val.totalCost))
			}
			if val.typ == "Mana" && hybridLifeCost > 0 {
				lines = append(lines, sFormat("x %.2f ^8(%d%% paid for with life)", 1-hybridLifeCost, hybridLifeCost*100))
			}
			verb := ".2f"
			if val.upfront {
				verb = "d"
			}
			lines = append(lines, sFormat("= %"+verb+pct, output[costName]))
			breakdown.set(costName, lines...)
		}
	}
	return hasCosts
}
