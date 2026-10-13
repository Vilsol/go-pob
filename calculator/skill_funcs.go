package calculator

import (
	"maps"
	"math"
	"slices"
	"sort"

	"github.com/Vilsol/go-pob/calculator/calclib"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

type skillFunc = func(activeSkill *ActiveSkill, output map[string]float64, breakdown *Breakdown)

//pob:src/Modules/CalcOffence.lua:476-481
func skillFuncsFor(id string) map[string]skillFunc {
	return skillFuncs[id]
}

//pob:block:src/Data/Skills/act_int.lua:716-718
var brandPreDamageFunc skillFunc = func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
	activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency /
		(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "Speed", "BrandActivationFrequency")/100) /
		activeSkill.SkillModList.More(activeSkill.SkillCfg, "BrandActivationFrequency")
}

//pob:block:src/Data/Skills/act_int.lua:21107-21131
var wintertideBrandPreDamageFunc skillFunc = func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
	activeSkill.SkillData.CountsAttachedBrandsInDamage = activeSkill.SkillPart == 1
	activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency /
		(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "Speed", "BrandActivationFrequency")/100) /
		activeSkill.SkillModList.More(activeSkill.SkillCfg, "BrandActivationFrequency")

	if activeSkill.SkillPart != 1 {
		return
	}

	list, cfg, sd := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData

	skillMaxStages := list.Sum(mod.TypeBase, cfg, "Multiplier:WintertideBrandMaxStages")
	debuffDurationMult := float64(1)
	if activeSkill.Actor != nil && activeSkill.Actor.Enemy != nil {
		debuffDurationMult = 1 / max(data.BuffExpirationSlowCap, calclib.Mod(activeSkill.Actor.Enemy.ModDB, cfg, "BuffExpireFaster"))
	}
	duration := float64(calcSkillDuration(list, cfg, sd, &Environment{}, nil) * debuffDurationMult)
	if duration <= 0 || sd.HitTimeOverride <= 0 {
		return
	}
	maxStages := math.Min(math.Floor(duration/sd.HitTimeOverride), skillMaxStages)

	timeToReachMaxStages := float64(maxStages * sd.HitTimeOverride)
	timeAtMaxStages := math.Max(duration-timeToReachMaxStages, 0)
	damagePerStage := list.Sum(mod.TypeBase, cfg, "Multiplier:WintertideBrandDamagePerStage")
	averageStages := (float64(maxStages*(maxStages-1)/2*sd.HitTimeOverride) + float64(maxStages*timeAtMaxStages)) / duration
	averageDamageMultiplier := float64(averageStages * damagePerStage)

	endDurationBase := sd.DurationTertiary + list.Sum(mod.TypeBase, cfg, "Duration", "TertiaryDuration")
	endDurationMod := math.Max(durationModClamped(list, cfg, "Duration", "TertiaryDuration"), 0)
	endDuration := math.Ceil(endDurationBase*endDurationMod*debuffDurationMult*data.ServerTickRate) / data.ServerTickRate

	endUptime := math.Min(endDuration/duration, 1)
	attachedBrandCount := sd.AttachedBrandCount
	endDamageMultiplier := float64(maxStages * damagePerStage)
	dpsMultiplier := float64(attachedBrandCount*(100+averageDamageMultiplier)) + float64(endUptime*(100+endDamageMultiplier)) - 100
	list.AddMod(mod.NewFloat("Damage", mod.TypeMore, dpsMultiplier).Source("Wintertide Brand Average Multiplier"))
}

//pob:block:src/Data/Skills/act_int.lua:13809-13811
var penanceBrandPreDamageFunc skillFunc = func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
	activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency *
		activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:PenanceBrandMaxEnergy") /
		(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "Speed", "BrandActivationFrequency")/100) /
		activeSkill.SkillModList.More(activeSkill.SkillCfg, "BrandActivationFrequency")
}

//pob:block:src/Data/Skills/act_dex.lua:1374-1379
var bladeBlastPreDamageFunc skillFunc = func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
	activeSkill.SkillData.DpsMultiplier = utils.OrDefault(activeSkill.SkillData.DpsMultiplier, 1) *
		activeSkill.SkillData.DpsBaseMultiplier
	if activeSkill.SkillPart == 2 {
		activeSkill.SkillData.HitTimeOverride = 1
	}
}

var cooldownIsHitTime skillFunc = func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
	activeSkill.SkillData.HitTimeOverride = output["Cooldown"]
}

//pob:block:src/Data/Skills/act_int.lua:16721-16727
var sparkPreDamageFunc skillFunc = func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
	if activeSkill.SkillPart != 2 {
		return
	}
	activeSkill.SkillData.DpsMultiplier = utils.OrDefault(activeSkill.SkillData.DpsMultiplier, 1) *
		(1 + math.Floor(output["Duration"]/0.66))
	output["SkillDPSMultiplier"] = activeSkill.SkillData.DpsMultiplier
}

//pob:block:src/Data/Skills/act_str.lua:8241-8243
var perforateDpsMultiplierFloor skillFunc = func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
	activeSkill.SkillData.DpsMultiplier = math.Max(utils.OrDefault(activeSkill.SkillData.DpsMultiplier, 1), 1)
}

//pob:block:src/Data/Skills/act_dex.lua:13776-13784,src/Data/Skills/act_dex.lua:13864-13872
var shrapnelBallistaPreDamage skillFunc = func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
	if !activeSkill.SkillModList.Flag(nil, "SequentialProjectiles") {
		overlap := 1.0
		if activeSkill.SkillTypes[data.SkillTypeRain] {
			overlap = output["ProjectileCount"]
		}
		activeSkill.SkillData.DpsMultiplier = utils.OrDefault(activeSkill.SkillData.DpsMultiplier, 1) *
			min(utils.UnwrapOrF(activeSkill.SkillData.ShrapnelBallistaProjectileOverlap, overlap), output["ProjectileCount"])
	}
	if splitCount := output["SplitCount"]; splitCount > 0 {
		activeSkill.SkillModList.AddMod(mod.NewFloat("DPS", mod.TypeMore, splitCount*100).Source("Split Return").
			Tag(mod.Condition("ReturningProjectile")))
	}
}

//pob:block:src/Data/Skills/act_dex.lua:10231-10238, src/Data/Skills/act_dex.lua:10334-10341
func lancingSteelPreDamageFunc(source mod.Source) skillFunc {
	return func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart != 2 {
			return
		}
		projectileCount := output["ProjectileCount"]
		if projectileCount == 0 {
			return
		}
		percentReducedProjectiles := (projectileCount - 1) / projectileCount
		mult := (activeSkill.SkillModList.More(activeSkill.SkillCfg, "LancingSteelSubsequentDamage") - 1) *
			100 * percentReducedProjectiles
		activeSkill.SkillData.DpsMultiplier = projectileCount
		activeSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, mult).Source(source))
	}
}

//pob:block:src/Data/Skills/act_int.lua:18043-18047
func stormBurstRemainingZapDamage(activeSkill *ActiveSkill, output map[string]float64) {
	skillData := activeSkill.SkillData
	remainingJumps := math.Floor((float64(skillData.Duration*output["DurationMod"]) + 0.001) / skillData.RepeatFrequency)
	activeSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore,
		math.Max(float64(skillData.RemainingDurationDamage*remainingJumps)-100, 0)).Source("Skill:StormBurst"))
}

//pob:block:src/Data/Skills/act_dex.lua:10014-10019
var poisonousConcoctionPreDamageFunc skillFunc = func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
	multiplier := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "ChaosPerLifeFlaskPercent")
	addedFromFlask := output["LifeFlaskRecovery"] * (multiplier / 100)
	activeSkill.SkillModList.AddMod(mod.NewFloat("ChaosMin", mod.TypeBase, addedFromFlask).Source("Life Flask charges consumed"))
	activeSkill.SkillModList.AddMod(mod.NewFloat("ChaosMax", mod.TypeBase, addedFromFlask).Source("Life Flask charges consumed"))
}

//pob:block:src/Data/Skills/other.lua:6076-6124
var enemyExplodePreDamageFunc skillFunc = func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
	skillData := activeSkill.SkillData
	output["ExplodeChance"] = 0
	if activeSkill.SkillPart != 3 {
		source := activeSkill.ActiveEffect.ExplodeSource
		for _, row := range activeSkill.SkillModList.Tabulate(mod.TypeList, activeSkill.SkillCfg, "ExplodeMod") {
			explodeMod, ok := explodeModValue(row.Mod)
			if !ok || source == nil || string(row.Mod.GetSource()) != source.activeEffectSource() {
				continue
			}
			if explodeMod.Type == "RandomElement" {
				skillData.FireEffectiveExplodePercentage = explodeMod.Amount / 3
				skillData.ColdEffectiveExplodePercentage = explodeMod.Amount / 3
				skillData.LightningEffectiveExplodePercentage = explodeMod.Amount / 3
			} else {
				utils.Set(skillData, explodeMod.Type+"EffectiveExplodePercentage", explodeMod.Amount)
			}
			if activeSkill.SkillPart == 2 {
				output["ExplodeChance"] = 1
			} else {
				output["ExplodeChance"] = explodeMod.Chance
			}
		}
	} else {
		typeAmountChances := map[string]map[float64]float64{}
		for _, value := range activeSkill.SkillModList.List(activeSkill.SkillCfg, "ExplodeMod") {
			explodeMod, ok := value.(ExplodeMod)
			if !ok {
				continue
			}
			amountChance := typeAmountChances[explodeMod.Type]
			if amountChance == nil {
				amountChance = map[float64]float64{}
			}
			amountChance[explodeMod.Amount] += explodeMod.Chance
			typeAmountChances[explodeMod.Type] = amountChance
		}
		types := make([]string, 0, len(typeAmountChances))
		for t := range typeAmountChances {
			types = append(types, t)
		}
		sort.Strings(types)
		for _, typ := range types {
			amountChance := typeAmountChances[typ]
			amounts := make([]float64, 0, len(amountChance))
			for amount := range amountChance {
				amounts = append(amounts, amount)
			}
			sort.Float64s(amounts)
			physExplodeChance := 0.0
			for _, amount := range amounts {
				chance := amountChance[amount]
				amountXChance := float64(amount * chance)
				if typ == "RandomElement" {
					skillData.FireEffectiveExplodePercentage += amountXChance / 3
					skillData.ColdEffectiveExplodePercentage += amountXChance / 3
					skillData.LightningEffectiveExplodePercentage += amountXChance / 3
				} else {
					key := typ + "EffectiveExplodePercentage"
					utils.Set(skillData, key, utils.GetOr(skillData, key, float64(0))+amountXChance)
				}
				if typ == "Physical" {
					physExplodeChance = 1 - (float64((1 - physExplodeChance) * (1 - chance)))
				}
				output["ExplodeChance"] = 1 - (float64((1 - output["ExplodeChance"]) * (1 - chance)))
			}
			if typ == "Physical" && physExplodeChance != 0 {
				activeSkill.SkillModList.NewMod("CalcArmourAsThoughDealing", mod.TypeMore, 100/math.Min(physExplodeChance, 1)-100, "")
			}
		}
	}
	output["ExplodeChance"] = math.Min(output["ExplodeChance"]*100, 100)
}

func explodeModValue(m mod.Mod) (ExplodeMod, bool) {
	if m == nil || m.Value() == nil {
		return ExplodeMod{}, false
	}
	switch v := m.Value().ValueList.(type) {
	case ExplodeMod:
		return v, true
	case *ExplodeMod:
		return *v, true
	}
	return ExplodeMod{}, false
}

//pob:block:src/Data/Skills/minion.lua:757-759,src/Data/Skills/minion.lua:1757-1759
var summonSpeedHitTimePreDamageFunc skillFunc = func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
	summonSpeed := 1.0
	if activeSkill.SummonSkill != nil && activeSkill.SummonSkill.SkillData.SummonSpeed != nil {
		summonSpeed = *activeSkill.SummonSkill.SkillData.SummonSpeed
	}
	activeSkill.SkillData.HitTimeOverride = 1 / summonSpeed
}

var skillFuncs = map[string]map[string]skillFunc{
	"GAZombieCorpseGroundImpact": {"preDamageFunc": summonSpeedHitTimePreDamageFunc},
	"MinionInstability":          {"preDamageFunc": summonSpeedHitTimePreDamageFunc},

	"EnemyExplode": {"preDamageFunc": enemyExplodePreDamageFunc},

	"Spark":     {"preDamageFunc": sparkPreDamageFunc},
	"SparkAltX": {"preDamageFunc": sparkPreDamageFunc},
	"SparkAltY": {"preDamageFunc": sparkPreDamageFunc},

	"Perforate":     {"preDamageFunc": perforateDpsMultiplierFloor},
	"PerforateAltX": {"preDamageFunc": perforateDpsMultiplierFloor},
	"PerforateAltY": {"preDamageFunc": perforateDpsMultiplierFloor},

	"LancingSteelAltX": {"preDamageFunc": lancingSteelPreDamageFunc("Skill:LancingSteelAltX")},

	//pob:block:src/Data/Skills/act_int.lua:12642-12662
	"LightningTendrilsAltX": {
		"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
			interval := activeSkill.SkillData.PulseInterval
			if interval == 0 {
				return
			}
			switch activeSkill.SkillPart {
			case 2:
				activeSkill.SkillModList.AddMod(mod.NewFloat("DPS", mod.TypeMore, -(1/interval)*100).
					Source("Normal pulse").Tag(mod.SkillPart(2)))
			case 3:
				activeSkill.SkillModList.AddMod(mod.NewFloat("DPS", mod.TypeMore, -(interval-1)/interval*100).
					Source("Stronger pulse").Tag(mod.SkillPart(3)))
			}
		},
		"postCritFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
			if activeSkill.SkillPart != 1 {
				return
			}
			interval := activeSkill.SkillData.PulseInterval
			if interval == 0 {
				return
			}
			pulseDamage := activeSkill.SkillData.PulseDamage / 100
			critChance := output["PreEffectiveCritChance"] / 100
			effectiveCritChance := output["CritChance"] / 100
			critMulti := output["CritMultiplier"]
			denom := interval * ((1 - effectiveCritChance) + float64(critMulti*effectiveCritChance))
			if denom == 0 {
				return
			}
			averageMore := 100 * ((float64((interval-1)*(1+float64(critChance*(critMulti-1))))+float64((1+pulseDamage)*critMulti))/denom - 1)
			activeSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, averageMore).
				Source("Average Pulse Damage").
				KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).
				Tag(mod.SkillPart(1)))
		},
	},

	"Barrage":     {"preDamageFunc": barrageAllProjectiles},
	"BarrageAltX": {"preDamageFunc": barrageAllProjectiles},

	"VaalLightningArrow": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart == 2 {
			activeSkill.SkillData.DpsMultiplier = output["ProjectileCount"]
		}
	}},

	"KineticFusillade":     {"preDamageFunc": kineticFusilladePreDamageFunc("Skill:KineticFusillade"), "postCritFunc": kineticFusilladePostCritFunc},
	"KineticFusilladeAltX": {"preDamageFunc": kineticFusilladePreDamageFunc("Skill:KineticFusilladeAltX"), "postCritFunc": kineticFusilladePostCritFunc},

	"ToxicRain": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		podOverlap := activeSkill.SkillData.PodOverlapMultiplier
		if podOverlap == 0 {
			podOverlap = 1
		}
		activeSkill.SkillData.DpsMultiplier = math.Min(podOverlap, output["ProjectileCount"])
	}},
	"ToxicRainAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		podOverlap := activeSkill.SkillData.PodOverlapMultiplier
		if podOverlap == 0 {
			podOverlap = 1
		}
		activeSkill.SkillData.DpsMultiplier = math.Min(podOverlap, output["ProjectileCount"])
	}},
	"ToxicRainAltY": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		podOverlap := activeSkill.SkillData.PodOverlapMultiplier
		if podOverlap == 0 {
			podOverlap = 1
		}
		activeSkill.SkillData.DpsMultiplier = math.Min(podOverlap, output["ProjectileCount"])
	}},

	"BlastRain": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart == 2 {
			activeSkill.SkillData.DpsMultiplier = output["ProjectileCount"]
		}
	}},

	"BlazingSalvo": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart == 2 {
			activeSkill.SkillData.DpsMultiplier = output["ProjectileCount"]
		}
	}},

	"IceSpear":     {"preDamageFunc": iceSpearSecondForm},
	"IceSpearAltX": {"preDamageFunc": iceSpearSecondForm},

	"LancingSteel": {"preDamageFunc": lancingSteelPreDamageFunc("Skill:LancingSteel")},

	//pob:src/Data/Skills/act_int.lua:4361-4386, src/Data/Skills/act_int.lua:7417
	"ForbiddenRite": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		baseTakenNames := []string{"DamageTaken", "ChaosDamageTaken", "DamageTakenWhenHit", "ChaosDamageTakenWhenHit"}
		baseTakenFlat := activeSkill.SkillModList.Sum(mod.TypeBase, nil, baseTakenNames...)
		baseTakenInc := activeSkill.SkillModList.Sum(mod.TypeIncrease, nil, baseTakenNames...)
		baseTakenMore := activeSkill.SkillModList.More(nil, baseTakenNames...)
		chaosDamageTaken := math.Max((1+baseTakenInc/100)*baseTakenMore, 0)
		chaosFlat := math.Floor(float64(baseTakenFlat*chaosDamageTaken) + 0.5)

		life, energyShield, chaosResistance := output["Life"], output["EnergyShield"], output["ChaosResist"]
		if activeSkill.SkillFlags[SkillFlagTotem] {
			life, energyShield, chaosResistance = output["TotemLife"], output["TotemEnergyShield"], output["TotemChaosResist"]
		}

		add := float64(life*activeSkill.SkillData.LifeDealtAsChaos) + float64(energyShield*activeSkill.SkillData.EnergyShieldDealtAsChaos)
		selfDamageTakenLife := math.Floor(math.Floor(float64(life*activeSkill.SkillData.SelfDamageTakenLife)+0.5) * (100 - chaosResistance) / 100 * chaosDamageTaken)
		selfDamageTakenES := math.Floor(math.Floor(float64(energyShield*activeSkill.SkillData.SelfDamageTakenES)+0.5) * (100 - chaosResistance) / 100 * chaosDamageTaken)
		activeSkill.SkillData.ChaosMin += add
		activeSkill.SkillData.ChaosMax += add

		if activeSkill.SkillPart == 2 {
			activeSkill.SkillData.DpsMultiplier = utils.OrDefault(activeSkill.SkillData.DpsMultiplier, 1) * (output["ProjectileCount"] + 1)
		}

		output["FRDamageTaken"] = selfDamageTakenLife + selfDamageTakenES + chaosFlat
	}},

	//pob:src/Data/Skills/act_int.lua:7549-7589
	"ForbiddenRiteAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		baseTakenNames := []string{"DamageTaken", "ChaosDamageTaken", "DamageTakenWhenHit", "ChaosDamageTakenWhenHit"}
		baseTakenFlat := activeSkill.SkillModList.Sum(mod.TypeBase, nil, baseTakenNames...)
		baseTakenInc := activeSkill.SkillModList.Sum(mod.TypeIncrease, nil, baseTakenNames...)
		baseTakenMore := activeSkill.SkillModList.More(nil, baseTakenNames...)
		chaosDamageTaken := math.Max((1+baseTakenInc/100)*baseTakenMore, 0)
		chaosFlat := math.Floor(float64(baseTakenFlat*chaosDamageTaken) + 0.5)

		energyShield, chaosResistance := output["EnergyShield"], output["ChaosResist"]
		if activeSkill.SkillFlags[SkillFlagTotem] {
			energyShield, chaosResistance = output["TotemEnergyShield"], output["TotemChaosResist"]
		}

		add := float64(energyShield * activeSkill.SkillData.EnergyShieldDealtAsChaos)
		selfDamageTakenES := math.Floor(math.Floor(float64(energyShield*activeSkill.SkillData.SelfDamageTakenES)+0.5) * (100 - chaosResistance) / 100 * chaosDamageTaken)
		activeSkill.SkillData.ChaosMin += add
		activeSkill.SkillData.ChaosMax += add

		if activeSkill.SkillPart == 2 {
			activeSkill.SkillData.DpsMultiplier = utils.OrDefault(activeSkill.SkillData.DpsMultiplier, 1) * (output["ProjectileCount"] + 1)
		}

		output["FRDamageTaken"] = selfDamageTakenES + chaosFlat
	}},

	//pob:src/Data/Skills/act_int.lua:3889-3903
	//pob:src/Data/Skills/sup_str.lua:2539-2547
	"AvengingFlame": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		totemLife := 0.0
		if cache := activeSkill.SkillData.triggerSource; cache != nil {
			totemLife = cache.TotemLife
		}
		add := totemLife * activeSkill.SkillData.LifeDealtAsFire / 100
		activeSkill.SkillData.FireMax += add
		activeSkill.SkillData.FireMin += add
	}},

	"DarkPact": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		var life float64
		if activeSkill.SkillPart == 1 {
			life = output["Life"]
			if activeSkill.SkillFlags[SkillFlagTotem] {
				life = output["TotemLife"]
			}
		} else {
			life = activeSkill.SkillData.SkeletonLife
		}

		add := life * activeSkill.SkillData.LifeDealtAsChaos / 100
		activeSkill.SkillData.ChaosMin += add
		activeSkill.SkillData.ChaosMax += add
	}},

	//pob:block:src/Data/Skills/act_int.lua:4006-4022
	"DarkPactAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		life := output["Life"]
		if activeSkill.SkillFlags[SkillFlagTotem] {
			life = output["TotemLife"]
		}
		add := life * activeSkill.SkillData.PercentageLifeSacrificed / 100 *
			(activeSkill.SkillData.PercentageSacrificedDealtAsChaos / 100)
		if activeSkill.SkillPart != 2 {
			add /= 7 / (1 + math.Min(100, activeSkill.SkillData.AdditionalRuinChance)/100)
		}
		activeSkill.SkillData.ChaosMin += math.Floor(add)
		activeSkill.SkillData.ChaosMax += math.Floor(add)
	}},

	//pob:block:src/Data/Skills/act_int.lua:2298-2308
	"Bodyswap":     {"preDamageFunc": bodyswapSelfExplosion},
	"BodyswapAltX": {"preDamageFunc": bodyswapSelfExplosion},

	//pob:block:src/Data/Skills/act_int.lua:13527-13532
	"Manabond": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		missingUnreserved := utils.OrDefault(activeSkill.SkillData.ManabondMissingUnreservedManaPercentage, 100)
		gained := math.Floor((activeSkill.SkillData.ManabondMissingManaGainPercent / 100) *
			(missingUnreserved / 100) * output["ManaUnreserved"])
		const src = "Manabond gain % missing unreserved mana as base lightning damage"
		activeSkill.SkillModList.NewMod("LightningMin", mod.TypeBase, gained, src)
		activeSkill.SkillModList.NewMod("LightningMax", mod.TypeBase, gained, src)
	}},

	//pob:block:src/Data/Skills/act_str.lua:3300-3303
	"Earthquake": {"preDamageFunc": durationScaledDamageInc("EarthquakeDurationIncDamage", "Skill:Earthquake")},
	//pob:block:src/Data/Skills/act_int.lua:20445-20448
	"VoltaxicBurst": {"preDamageFunc": durationScaledDamageInc("VoltaxicDurationIncDamage", "Skill:VoltaxicBurst")},

	"BladeFlurry": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart == 2 && activeSkill.SkillData.NumStages > 0 {
			numStages := activeSkill.SkillData.NumStages
			channelMulti := float64(0)
			for i := float64(1); i <= numStages; i++ {
				channelMulti += 0.8 + float64(0.2*i)
			}
			channelMulti /= 0.8 + float64(0.2*numStages)
			activeSkill.SkillData.DpsMultiplier = channelMulti/numStages + 1
		}
	}},
	"BladeFlurryAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart == 2 && activeSkill.SkillData.NumStages > 0 {
			numStages := activeSkill.SkillData.NumStages
			channelMulti := float64(0)
			for i := float64(1); i <= numStages; i++ {
				channelMulti += 0.8 + float64(0.2*i)
			}
			channelMulti /= 0.8 + float64(0.2*numStages)
			activeSkill.SkillData.DpsMultiplier = channelMulti/numStages + 1
		}
	}},

	"InfernalBlow":     {"preDamageFunc": infernalBlowDebuff("InfernalBlow")},
	"InfernalBlowAltX": {"preDamageFunc": infernalBlowDebuff("InfernalBlowAltX")},

	//pob:block:src/Data/Skills/act_int.lua:18036-18048
	"StormBurst": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		skillData := activeSkill.SkillData
		switch activeSkill.SkillPart {
		case 2:
			duration := skillData.Duration * output["DurationMod"]
			jumpPeriod := skillData.RepeatFrequency * 10
			skillData.DpsMultiplier = utils.OrDefault(skillData.DpsMultiplier, 1) *
				(1 + math.Floor(duration*10/jumpPeriod))
		case 3:
			stormBurstRemainingZapDamage(activeSkill, output)
		}
	}},

	//pob:src/Data/Skills/act_int.lua:15428-15433
	"RighteousFire": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		if activeSkill.SkillFlags[SkillFlagTotem] && output["TotemLife"] > 1 {
			activeSkill.SkillData.FireDot = float64(output["TotemLife"]*activeSkill.SkillData.RFLifeMultiplier) +
				float64(output["TotemEnergyShield"]*activeSkill.SkillData.RFESMultiplier)
		} else if output["LifeUnreserved"] > 1 {
			activeSkill.SkillData.FireDot = float64(output["Life"]*activeSkill.SkillData.RFLifeMultiplier) +
				float64(output["EnergyShield"]*activeSkill.SkillData.RFESMultiplier)
		}
	}},

	"RighteousFireAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		if output["LifeUnreserved"] > 1 {
			activeSkill.SkillData.FireDot = output["Mana"] * activeSkill.SkillData.RFManaMultiplier
		}
	}},

	"VaalRighteousFire": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		if activeSkill.SkillFlags[SkillFlagTotem] {
			activeSkill.SkillData.FireDot = output["TotemLife"] *
				activeSkill.SkillData.PercentSacrificed * activeSkill.SkillData.RFMultiplier
		} else {
			activeSkill.SkillData.FireDot = (output["Life"] + output["EnergyShield"]) *
				activeSkill.SkillData.PercentSacrificed * activeSkill.SkillData.RFMultiplier
		}
	}},

	"ArmageddonBrand":     {"preDamageFunc": brandPreDamageFunc},
	"ArmageddonBrandAltY": {"preDamageFunc": brandPreDamageFunc},
	"BrandSupport":        {"preDamageFunc": brandPreDamageFunc},
	"StormBrand":          {"preDamageFunc": brandPreDamageFunc},
	"StormBrandAltX":      {"preDamageFunc": brandPreDamageFunc},
	"PenanceBrand":        {"preDamageFunc": penanceBrandPreDamageFunc},
	"PenanceBrandAltX":    {"preDamageFunc": brandPreDamageFunc},
	"PenanceBrandAltY":    {"preDamageFunc": brandPreDamageFunc},
	"WintertideBrand":     {"preDamageFunc": wintertideBrandPreDamageFunc},

	"Vortex":     {"preDamageFunc": cooldownIsHitTime},
	"VortexAltX": {"preDamageFunc": cooldownIsHitTime},

	//pob:block:src/Data/Skills/act_dex.lua:1374-1379
	"BladeBlast":     {"preDamageFunc": bladeBlastPreDamageFunc},
	"BladeBlastAltX": {"preDamageFunc": bladeBlastPreDamageFunc},

	//pob:block:src/Data/Skills/act_int.lua:8972-8976
	"Hydrosphere": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart < 1 || activeSkill.SkillPart > 3 {
			return
		}
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency /
			(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "HydroSphereFrequency")/100)
	}},

	"HeraldOfThunder": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency /
			(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "HeraldStormFrequency")/100)
	}},

	//pob:src/Data/Skills/other.lua:2130-2132
	"HeraldOfTheBreach": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency /
			(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "PulseFrequencyPerPressure")/100)
	}},

	"VoidSphere": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency /
			(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "VoidSphereFrequency")/100)
	}},
	"VoidSphereAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency /
			(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "VoidSphereFrequency")/100)
	}},

	"WinterOrb": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		rateMod := 1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "HitRate", "Speed")/100
		mult := activeSkill.SkillModList.More(activeSkill.SkillCfg, "HitRate")
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency / rateMod / mult
	}},

	"OrbOfStorms": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.HitFrequency /
			calclib.Mod(activeSkill.SkillModList, activeSkill.SkillCfg, "Speed")
	}},

	"StormRain": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart == 2 {
			activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.HitFrequency /
				(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "StormRainBeamFrequency")/100)
			activeSkill.SkillData.DpsMultiplier = utils.OrDefault(activeSkill.SkillData.BeamOverlapMultiplier, 1)
		}
	}},

	//pob:block:src/Data/Skills/act_dex.lua:11848-11853
	"StormRainAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart == 2 {
			activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.HitFrequency /
				(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "StormRainBeamFrequency")/100)
			activeSkill.SkillData.DpsMultiplier = min(utils.UnwrapOrF(activeSkill.SkillData.ActiveArrowMultiplier, 1),
				activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "StormRainAllowedStormArrows"))
		}
	}},

	//pob:block:src/Data/Skills/act_dex.lua:11955-11961
	"StormRainAltY": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart == 2 {
			activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.HitFrequency /
				(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "StormRainBeamFrequency")/100)
			activeSkill.SkillData.DpsMultiplier =
				activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "StormRainAllowedStormArrows")
		}
	}},

	//pob:block:src/Data/Skills/act_dex.lua:16048-16059
	//pob:src/Data/Skills/act_dex.lua:18031-18033
	"Tornado": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.DamageInterval
	}},
	//pob:src/Data/Skills/act_dex.lua:18124-18126
	"TornadoAltY": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.DamageInterval
		activeSkill.SkillData.DpsMultiplier = utils.OrDefault(activeSkill.SkillData.DpsMultiplier, 1) *
			activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:TornadoofElementalTurbulenceStage")
	}},

	//pob:src/Data/Skills/act_dex.lua:2209-2211
	"BladeVortex": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		blades := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:BladeVortexBlade")
		bladeVortexHitTime(activeSkill, blades)
	}},
	//pob:src/Data/Skills/act_dex.lua:2429-2431
	"VaalBladeVortex": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		bladeVortexHitTime(activeSkill, activeSkill.SkillData.VaalBladeVortexBlade)
	}},
	//pob:src/Data/Skills/act_dex.lua:2787-2789
	"BladefallAltZ": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		divisor := 1 + activeSkill.SkillData.IncVolleyFrequency/100
		if divisor == 0 {
			return
		}
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.HitFrequency / divisor
	}},
	//pob:src/Data/Skills/act_int.lua:6800-6802
	"Flameblast": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		flameblastHitTimeMultiplier(activeSkill, "Multiplier:FlameblastStage")
	}},
	//pob:src/Data/Skills/act_int.lua:6905-6907
	"FlameblastAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		flameblastHitTimeMultiplier(activeSkill, "Multiplier:FlameblastofCelerityStage")
	}},
	//pob:src/Data/Skills/act_int.lua:7008-7010
	"FlameblastAltY": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		flameblastHitTimeMultiplier(activeSkill, "Multiplier:FlameblastofContractionStage")
	}},
	//pob:src/Data/Skills/act_int.lua:4602-4606
	"DivineIre": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart != 2 {
			return
		}
		activeSkill.SkillData.HitTimeMultiplier = activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:DivineIreStage")
	}},
	//pob:src/Data/Skills/act_int.lua:4712-4716
	"DivineIreAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart != 2 {
			return
		}
		activeSkill.SkillData.HitTimeMultiplier = activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:DivineIreofHolyLightningStage")
	}},
	//pob:src/Data/Skills/act_int.lua:4823-4825
	"DivineIreAltY": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeMultiplier = activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:DivineIreofDisintegrationStage")
	}},
	//pob:src/Data/Skills/act_int.lua:10284-10288
	"Incinerate": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		incinerateHitTimeMultiplier(activeSkill, "Multiplier:IncinerateStage")
	}},
	//pob:src/Data/Skills/act_int.lua:10422-10426
	"IncinerateAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		incinerateHitTimeMultiplier(activeSkill, "Multiplier:IncinerateofExpanseStage")
	}},
	//pob:src/Data/Skills/act_int.lua:10561-10565
	"IncinerateAltY": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		incinerateHitTimeMultiplier(activeSkill, "Multiplier:IncinerateofVentingStage")
	}},
	//pob:src/Data/Skills/act_int.lua:8543-8545
	"GalvanicField": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency
	}},
	//pob:src/Data/Skills/act_int.lua:8663-8665
	"GalvanicFieldAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency
	}},
	//pob:src/Data/Skills/other.lua:2468-2470
	"LightningBolt": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeOverride = output["Cooldown"]
	}},
	"TornadoShot": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart != 2 || output["ReturnChance"] != 0 {
			return
		}
		averageSecondaryProjectiles := output["ProjectileCount"] + output["SplitCount"]
		if activeSkill.SkillModList.Flag(nil, "SequentialProjectiles") &&
			!activeSkill.SkillModList.Flag(nil, "OneShotProj") &&
			!activeSkill.SkillModList.Flag(nil, "NoAdditionalProjectiles") &&
			!activeSkill.SkillModList.Flag(nil, "SingleProjectile") &&
			!activeSkill.SkillModList.Flag(nil, "TriggeredBySnipe") {
			averageSecondaryProjectiles = 1 + output["SplitCount"]
		}
		chanceForSecondaryProjectilesToHit := min(utils.UnwrapOrF(activeSkill.SkillData.TornadoShotSecondaryHitChance,
			20*activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "tornadoShotSecondaryProjectiles"))/100, 1)
		activeSkill.SkillData.DpsMultiplier = utils.OrDefault(activeSkill.SkillData.DpsMultiplier, 1) *
			(1 + float64(chanceForSecondaryProjectilesToHit*averageSecondaryProjectiles))
	}},

	"ShrapnelBallista":     {"preDamageFunc": shrapnelBallistaPreDamage},
	"ShrapnelBallistaAltX": {"preDamageFunc": shrapnelBallistaPreDamage},
	//pob:block:src/Data/Skills/act_str.lua:5680-5682
	"HeraldOfAsh": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.FireDot = utils.UnwrapOrF(activeSkill.SkillData.HoaOverkill, 0) *
			(1 + activeSkill.SkillData.HoaMoreBurn/100) * activeSkill.SkillData.HoaOverkillPercent
	}},
	//pob:block:src/Data/Skills/act_str.lua:7536-7540
	"MoltenShell": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		add := utils.UnwrapOrF(activeSkill.SkillData.MoltenShellDamageMitigated, 0) * activeSkill.SkillData.MoltenShellReflect / 100
		activeSkill.SkillData.FireMin = add
		activeSkill.SkillData.FireMax = add
	}},
	//pob:block:src/Data/Skills/act_str.lua:7638-7642
	"VaalMoltenShell": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		add := utils.UnwrapOrF(activeSkill.SkillData.VaalMoltenShellDamageMitigated, 0) * activeSkill.SkillData.MoltenShellReflect / 100
		activeSkill.SkillData.FireMin = add
		activeSkill.SkillData.FireMax = add
	}},

	"Cremation": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart == 1 {
			activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.CremationFireRate /
				(1 + activeSkill.SkillData.CremationFireRateIncrease)
		}
	}},
	"CremationAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart == 1 {
			activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.CremationFireRate /
				(1 + activeSkill.SkillData.CremationFireRateIncrease)
		}
	}},
	//pob:src/Data/Skills/act_dex.lua:4590-4592
	"CremationAltY": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.CremationFireRate /
			(1 + activeSkill.SkillData.CremationFireRateIncrease)
	}},

	//pob:block:src/Data/Skills/act_str.lua:10663-10667
	"StaticStrike": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart == 2 {
			activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.RepeatFrequency /
				(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, "StaticStrikeFrequency")/100)
		}
	}},

	//pob:src/Data/Skills/act_dex.lua:10014-10019
	"PoisonousConcoction": {"preDamageFunc": poisonousConcoctionPreDamageFunc},

	"PoisonousConcoctionAltX": {"preDamageFunc": poisonousConcoctionPreDamageFunc},

	//pob:src/Data/Skills/act_dex.lua:4701-4714
	"Cyclone": {"initialFunc": cycloneInitialFunc},

	//pob:src/Data/Skills/act_dex.lua:4932-4945
	"VaalCyclone": {"initialFunc": cycloneInitialFunc},

	//pob:src/Data/Skills/act_dex.lua:4807-4820
	"CycloneAltX": {"initialFunc": cycloneAltXInitialFunc},

	//pob:src/Data/Skills/act_dex.lua:7524-7526
	//pob:block:src/Data/Skills/act_dex.lua:12987-12989
	"ScourgeArrow": {"preDamageFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		activeSkill.SkillData.HitTimeMultiplier = math.Max(
			activeSkill.SkillModList.Sum(mod.TypeBase, nil, "Multiplier:ScourgeArrowStage")-0.5, 0.5)
	}},

	//pob:src/Data/Skills/other.lua:2207-2209

	//pob:src/Data/Skills/act_int.lua:574-717
	"BallLightning": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		skillCfg := activeSkill.SkillCfg
		skillData := activeSkill.SkillData
		skillModList := activeSkill.SkillModList

		superchance := skillModList.Sum(mod.TypeBase, skillCfg, "ball_lightning_superball_%_chance") / 100
		output["SuperchargePct"] = superchance * 100

		dpsMultiplier := float64(0)
		switch activeSkill.SkillPart {
		case 1:
			if superchance > 0 {
				dpsMultiplier = 1 + float64(0.5*superchance)
			} else {
				dpsMultiplier = 1
			}
		case 2:
			secsPerStrike := skillData.StrikeInterval
			durationSecs := skillData.Duration
			maxStrikes := math.Floor(durationSecs / secsPerStrike)

			incSpeedMult, moreSpeedMult := calclib.Mods(skillModList, skillCfg, "ProjectileSpeed")
			ballDistPerSec := skillData.ProjectileSpeed * incSpeedMult * moreSpeedMult
			ballDistPerStrike := ballDistPerSec * secsPerStrike

			for isSuperballAsInt := 0; isSuperballAsInt <= 1; isSuperballAsInt++ {
				isSuperball := isSuperballAsInt > 0
				if isSuperball && superchance == 0 {
					break
				}
				baseStrikeRadius := output["AreaOfEffectRadius"]
				strikeRadius := baseStrikeRadius
				if isSuperball {
					strikeRadius = math.Floor(float64(strikeRadius*math.Sqrt(1.5)) + 0.5)
				}

				var castDist float64
				switch {
				case skillCfg != nil && skillCfg.SkillDistSet:
					castDist = skillCfg.SkillDist
				case activeSkill.SkillFlags[SkillFlagTriggered]:
					castDist = math.Floor(baseStrikeRadius / 2)
				default:
					castDist = baseStrikeRadius
				}

				firstStrikeIdxThatHits := math.Max(1, math.Ceil((castDist-strikeRadius)/ballDistPerStrike))
				lastStrikeIdxThatHits := math.Floor(
					math.Min(data.ProjectileDistanceCap, castDist+strikeRadius) / ballDistPerStrike)
				numStrikes := math.Max(0, math.Min(maxStrikes, lastStrikeIdxThatHits+1-firstStrikeIdxThatHits))

				if isSuperball {
					effNumStrikes := numStrikes * superchance
					dpsMultiplier += float64(effNumStrikes * 1.5)
				} else {
					effNumStrikes := float64(numStrikes * (1 - superchance))
					dpsMultiplier += effNumStrikes
				}
			}
		}

		if dpsMultiplier != 1 {
			skillData.DpsMultiplier = dpsMultiplier
			output["SkillDPSMultiplier"] = dpsMultiplier
		}
	}},

	//pob:block:src/Data/Skills/act_int.lua:1252-1275
	"BallLightningAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart != 2 && activeSkill.SkillPart != 3 {
			return
		}
		skillData := activeSkill.SkillData
		numStrikes := math.Floor(skillData.Duration / skillData.StrikeInterval)
		if numStrikes != 1 {
			skillData.DpsMultiplier = utils.OrDefault(skillData.DpsMultiplier, 1) * numStrikes
			output["SkillDPSMultiplier"] = utils.OrDefault(output["SkillDPSMultiplier"], 1) * numStrikes
		}
		output["NormalHitsPerCast"] = numStrikes
	}},

	//pob:block:src/Data/Skills/act_int.lua:1375-1393
	"BallLightningAltY": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		skillData := activeSkill.SkillData
		numStrikes := math.Floor(skillData.Duration / skillData.StrikeInterval)
		skillData.DpsMultiplier = utils.OrDefault(skillData.DpsMultiplier, 1) * numStrikes
		output["NormalHitsPerCast"] = numStrikes
		output["SkillDPSMultiplier"] = utils.OrDefault(output["SkillDPSMultiplier"], 1) * numStrikes
	}},

	//pob:block:src/Data/Skills/act_int.lua:18151-18157
	"StormBurstAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart != 2 {
			return
		}
		stormBurstRemainingZapDamage(activeSkill, output)
	}},

	//pob:src/Data/Skills/act_int.lua:804-815
	"Bane": {"preSkillTypeFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		curseCount := float64(0)
		for _, skill := range activeSkill.Actor.ActiveSkillList {
			if skill.SocketGroup != activeSkill.SocketGroup {
				continue
			}
			if applied, _ := skill.SkillModList.GetCondition("AppliedByBane", nil, false); !applied {
				continue
			}
			curseCount++
			if curseCount == output["EnemyCurseLimit"] {
				break
			}
		}
		activeSkill.SkillModList.NewMod("Multiplier:CurseApplied", mod.TypeBase, curseCount, "Base")
	}},
	"BaneAltX": {"preSkillTypeFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		curseCount := float64(0)
		for _, skill := range activeSkill.Actor.ActiveSkillList {
			if skill.SocketGroup != activeSkill.SocketGroup {
				continue
			}
			if applied, _ := skill.SkillModList.GetCondition("AppliedByBane", nil, false); !applied {
				continue
			}
			curseCount++
			if curseCount == output["EnemyCurseLimit"] {
				break
			}
		}
		activeSkill.SkillModList.NewMod("Multiplier:CurseApplied", mod.TypeBase, curseCount, "Base")
	}},

	//pob:src/Data/Skills/act_str.lua:9019-9028
	"RageVortex": {"preSkillTypeFunc": func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart != 2 {
			return
		}
		maxRage := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "MaximumRage")
		sacrificePercentage := activeSkill.SkillData.MaxRageVortexSacrificePercentage / 100
		configOverride := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:RageSacrificedStacks")
		maxSacrificedRage := math.Floor(sacrificePercentage * maxRage)
		stacks := maxSacrificedRage
		if configOverride > 0 {
			stacks = configOverride
		}
		activeSkill.SkillModList.NewMod("Multiplier:RageSacrificed", mod.TypeBase, math.Min(stacks, maxSacrificedRage), "Skill:RageVortex")
	}},

	//pob:src/Data/Skills/act_int.lua:5518-5520
	"EyeOfWinter": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		ramp := projectileSpeedRamp(output, 60)
		activeSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore,
			activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "EyeOfWinterRamp")).
			Source("Skill:EyeOfWinter").Tag(mod.DistanceRamp(ramp)))
	}},

	//pob:src/Data/Skills/act_int.lua:4509-4513
	"FreezingPulse": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		ramp := projectileSpeedRamp(output, 60)
		activeSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, -50).
			Source("Skill:FreezingPulse").Tag(mod.DistanceRamp(ramp)))
		activeSkill.SkillModList.AddMod(mod.NewFloat("EnemyFreezeChance", mod.TypeBase, 25).
			Source("Skill:FreezingPulse").Tag(mod.DistanceRamp([][]float64{{0, 1}, {15 * output["ProjectileSpeedMod"], 0}})))
	}},

	//pob:block:src/Data/Skills/act_dex.lua:13619-13668
	"SeismicTrapAltX": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		overlapChance := areaOverlapChance(activeSkill, output)
		output["OverlapChance"] = overlapChance * 100
		maxWaves := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "MaximumWaves")
		dpsMultiplier := float64(1)
		switch activeSkill.SkillPart {
		case 2:
			dpsMultiplier = maxWaves * overlapChance
		case 3:
			dpsMultiplier = maxWaves
		}
		applyDpsMultiplier(activeSkill, output, dpsMultiplier)
	}},

	"SeismicTrap": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData

		baseInterval := skillData.RepeatInterval
		incFrequency := 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "TrapThrowingSpeed", "SeismicPulseFrequency")/100
		moreFrequency := skillModList.More(skillCfg, "TrapThrowingSpeed", "SeismicPulseFrequency")
		wavePulseRate := incFrequency * moreFrequency / baseInterval
		skillData.HitTimeOverride = 1 / wavePulseRate
		output["WavePulseRate"] = wavePulseRate

		incDuration := 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "Duration")/100
		moreDuration := skillModList.More(skillCfg, "Duration")
		duration := skillData.Duration * incDuration * moreDuration
		pulses := math.Floor(duration * wavePulseRate)
		output["PulsesPerTrap"] = pulses
		effectiveDuration := pulses / wavePulseRate
		averageActiveTraps := effectiveDuration / output["TrapCooldown"]
		output["AverageActiveTraps"] = averageActiveTraps

		overlapChance := areaOverlapChance(activeSkill, output)
		output["OverlapChance"] = overlapChance * 100

		maxWaves := skillModList.Sum(mod.TypeBase, skillCfg, "MaximumWaves")
		dpsMultiplier := float64(1)
		switch activeSkill.SkillPart {
		case 2:
			dpsMultiplier = maxWaves * overlapChance
		case 3:
			dpsMultiplier = maxWaves
		case 4:
			dpsMultiplier = averageActiveTraps
		case 5:
			dpsMultiplier = averageActiveTraps * maxWaves * overlapChance
		case 6:
			dpsMultiplier = averageActiveTraps * maxWaves
		}
		applyDpsMultiplier(activeSkill, output, dpsMultiplier)
	}},

	//pob:block:src/Data/Skills/act_int.lua:11667-11788
	"LightningSpireTrap": {"preDamageFunc": lightningSpireTrap("")},
	//pob:block:src/Data/Skills/act_int.lua:11912-12028
	"LightningSpireTrapAltX": {"preDamageFunc": lightningSpireTrap("AltX")},
	//pob:block:src/Data/Skills/act_int.lua:12150-12260
	"LightningSpireTrapAltY": {"preDamageFunc": lightningSpireTrap("AltY")},

	//pob:block:src/Data/Skills/act_int.lua:7933-7936
	"FrostBombAltY": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		duration := math.Floor(activeSkill.SkillData.Duration * output["DurationMod"] * 10)
		activeSkill.SkillModList.NewMod("Multiplier:100msFrostBombDuration", mod.TypeBase, duration, "Skill:FrostBombAltY")
	}},

	//pob:block:src/Data/Skills/act_dex.lua:7161-7173
	"ExplosiveTrap": {"preDamageFunc": func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		skillModList, skillCfg := activeSkill.SkillModList, activeSkill.SkillCfg
		hitChance := func(enemyRadius, areaDamageRadius, areaSpreadRadius float64) float64 {
			damagingAreaRadius := areaDamageRadius + enemyRadius - 1
			return math.Min(damagingAreaRadius*damagingAreaRadius/(areaSpreadRadius*areaSpreadRadius), 1)
		}
		enemyRadius := skillModList.Sum(mod.TypeBase, skillCfg, "EnemyRadius")
		if override := skillModList.Override(skillCfg, "EnemyRadius"); override != nil {
			enemyRadius = override.Float()
		}
		fullRadius := output["AreaOfEffectRadiusSecondary"]
		overlapChance := float64(0)
		marginWidth := activeSkill.SkillData.RadiusTertiaryBaseMargin*2 + 1
		for _, smallRadius := range slices.Sorted(maps.Keys(activeSkill.areaOfEffectRadiusTertiaryOccurrences)) {
			occurrenceCount := activeSkill.areaOfEffectRadiusTertiaryOccurrences[smallRadius]
			overlapChance += hitChance(enemyRadius, smallRadius, fullRadius) * occurrenceCount / marginWidth
		}
		output["OverlapChance"] = overlapChance * 100
		smallExplosionsPerTrap := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "SmallExplosions")
		output["SmallExplosionsPerTrap"] = smallExplosionsPerTrap

		dpsMultiplier := float64(1)
		switch activeSkill.SkillPart {
		case 2:
			dpsMultiplier = 1 + float64(smallExplosionsPerTrap*overlapChance)
		case 3:
			dpsMultiplier = 1 + smallExplosionsPerTrap
		}
		applyDpsMultiplier(activeSkill, output, dpsMultiplier)
	}},

	"MoltenStrike":     {"preDamageFunc": moltenStrikeOverlap},
	"MoltenStrikeAltX": {"preDamageFunc": moltenStrikeZenithOverlap},
}

func lightningSpireTrap(variant string) skillFunc {
	return func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData

		output["AreaOfEffectRadius"] = skillData.Radius

		baseInterval := skillData.RepeatInterval
		wavePulseRate := 1 / baseInterval
		if variant == "" {
			incFrequency := 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "TrapThrowingSpeed")/100
			moreFrequency := skillModList.More(skillCfg, "TrapThrowingSpeed")
			wavePulseRate = incFrequency * moreFrequency / baseInterval
		}
		skillData.HitTimeOverride = 1 / wavePulseRate
		if variant == "" {
			output["WavePulseRate"] = wavePulseRate
		}
		incDuration := 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "Duration")/100
		moreDuration := skillModList.More(skillCfg, "Duration")
		duration := skillData.Duration * incDuration * moreDuration
		pulses := math.Floor(duration * wavePulseRate)
		output["PulsesPerTrap"] = pulses
		effectiveDuration := pulses / wavePulseRate
		var averageActiveTraps float64
		if variant == "AltX" {
			actionSpeedMod := 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "ActionSpeed")/100
			baseSpeed := 1 / skillModList.Sum(mod.TypeBase, skillCfg, "TrapThrowingTime")
			throwSpeed := baseSpeed * calclib.Mod(skillModList, skillCfg, "TrapThrowingSpeed") * actionSpeedMod
			throwSpeed = math.Min(throwSpeed, data.ServerTickRate)
			throwTime := 1 / throwSpeed
			averageActiveTraps = effectiveDuration / throwTime
		} else {
			cooldown := output["TrapCooldown"]
			averageActiveTraps = effectiveDuration / cooldown
		}
		output["AverageActiveTraps"] = averageActiveTraps

		overlapChance := areaOverlapChance(activeSkill, output)
		output["OverlapChance"] = overlapChance * 100

		maxWaves := skillModList.Sum(mod.TypeBase, skillCfg, "MaximumWaves")
		dpsMultiplier := float64(1)
		switch activeSkill.SkillPart {
		case 2:
			dpsMultiplier = maxWaves * overlapChance
		case 3:
			dpsMultiplier = maxWaves
		case 4:
			dpsMultiplier = averageActiveTraps
		case 5:
			dpsMultiplier = averageActiveTraps * maxWaves * overlapChance
		case 6:
			dpsMultiplier = averageActiveTraps * maxWaves
		}
		applyDpsMultiplier(activeSkill, output, dpsMultiplier)
	}
}

//pob:block:src/Data/Skills/act_dex.lua:7793-7801
func areaOverlapChance(activeSkill *ActiveSkill, output map[string]float64) float64 {
	skillModList, skillCfg := activeSkill.SkillModList, activeSkill.SkillCfg
	enemyRadius := skillModList.Sum(mod.TypeBase, skillCfg, "EnemyRadius")
	if override := skillModList.Override(skillCfg, "EnemyRadius"); override != nil {
		enemyRadius = override.Float()
	}
	damagingAreaRadius := output["AreaOfEffectRadiusSecondary"] + enemyRadius - 1
	spread := output["AreaOfEffectRadius"]
	return math.Min(damagingAreaRadius*damagingAreaRadius/(spread*spread), 1)
}

//pob:block:src/Data/Skills/act_dex.lua:1026-1030
var barrageAllProjectiles skillFunc = func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
	if activeSkill.SkillPart == 2 {
		activeSkill.SkillData.DpsMultiplier = output["ProjectileCount"]
	}
}

//pob:block:src/Data/Skills/act_int.lua:9805-9808
var iceSpearSecondForm skillFunc = func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
	if activeSkill.SkillPart == 3 || activeSkill.SkillPart == 4 {
		activeSkill.SkillData.DpsMultiplier = utils.OrDefault(activeSkill.SkillData.DpsMultiplier, 1) *
			output["ProjectileCount"]
	}
}

//pob:block:src/Data/Skills/act_str.lua:7052-7060
func infernalBlowDebuff(skillID mod.Source) skillFunc {
	return func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		effect := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "DebuffEffect")
		if activeSkill.SkillPart == 2 || activeSkill.SkillPart == 3 {
			activeSkill.SkillModList.NewMod("Damage", mod.TypeMore, effect, mod.Source("Skill:")+skillID,
				mod.Multiplier("DebuffStack").Base(-100+effect))
		}
		if activeSkill.SkillPart == 3 {
			activeSkill.SkillData.DpsMultiplier = 1.0 / 6
		}
	}
}

//pob:block:src/Data/Skills/act_dex.lua:3916-3919
func applyDpsMultiplier(activeSkill *ActiveSkill, output map[string]float64, dpsMultiplier float64) {
	if dpsMultiplier == 1 {
		return
	}
	activeSkill.SkillData.DpsMultiplier = utils.OrF(activeSkill.SkillData.DpsMultiplier, 1) * dpsMultiplier
	output["SkillDPSMultiplier"] = utils.OrF(output["SkillDPSMultiplier"], 1) * dpsMultiplier
}

//pob:block:src/Data/Skills/act_int.lua:5519,src/Data/Skills/act_int.lua:4510-4511
func projectileSpeedRamp(output map[string]float64, upper float64) [][]float64 {
	return [][]float64{{0, 0}, {upper * output["ProjectileSpeedMod"], 1}}
}

func cycloneRangeInitialFunc(source string) skillFunc {
	return func(activeSkill *ActiveSkill, _ map[string]float64, _ *Breakdown) {
		rng := float64(0)
		if activeSkill.SkillFlags[SkillFlagWeapon1Attack] && activeSkill.Actor.WeaponData1 != nil &&
			activeSkill.Actor.WeaponData1.Range != 0 {
			weapon1RangeBonus := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.Weapon1Cfg, "MeleeWeaponRange") +
				float64(10*activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.Weapon1Cfg, "MeleeWeaponRangeMetre")) +
				activeSkill.Actor.WeaponData1.RangeBonus
			if activeSkill.SkillFlags[SkillFlagWeapon2Attack] && activeSkill.Actor.WeaponData2 != nil &&
				activeSkill.Actor.WeaponData2.Range != 0 {
				rng = (weapon1RangeBonus +
					activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.Weapon2Cfg, "MeleeWeaponRange") +
					float64(10*activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.Weapon2Cfg, "MeleeWeaponRangeMetre")) +
					activeSkill.Actor.WeaponData2.RangeBonus) / 2
			} else {
				rng = weapon1RangeBonus
			}
		} else {
			rng = activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "UnarmedRange") +
				float64(10*activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "UnarmedRangeMetre"))
		}
		activeSkill.SkillModList.AddMod(mod.NewFloat("Multiplier:AdditionalMeleeRange", mod.TypeBase, rng).Source(mod.Source(source)))
	}
}

var cycloneInitialFunc = cycloneRangeInitialFunc("Skill:Cyclone")
var cycloneAltXInitialFunc = cycloneRangeInitialFunc("Skill:CycloneAltX")

//	  COMPUTES output.Cooldown off the PARENT skill and then works out how many

func kineticFusilladePreDamageFunc(source string) skillFunc {
	return func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		if activeSkill.SkillPart != 1 {
			return
		}
		activeSkill.SkillData.DpsMultiplier = output["ProjectileCount"]

		moreDamagePerProj := activeSkill.SkillData.DamagePerProjectile
		if moreDamagePerProj == 0 || output["ProjectileCount"] <= 1 {
			return
		}
		avgMoreMult := moreDamagePerProj * (output["ProjectileCount"] - 1) / 2
		activeSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, avgMoreMult).Source(mod.Source(source)))
		output["KineticFusilladeAvgMoreMult"] = avgMoreMult
	}
}

var kineticFusilladePostCritFunc skillFunc = func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
	projectileCount := float64(1)
	if activeSkill.SkillPart == 1 {
		projectileCount = output["ProjectileCount"]
	}

	baseTimeForAllProjectiles := float64(activeSkill.SkillData.DelayPerProjectile * (projectileCount - 1))
	effectiveDelay := (activeSkill.SkillData.Duration + baseTimeForAllProjectiles) * output["DurationMod"]
	effectiveDelayRounded := math.Ceil(effectiveDelay/data.ServerTickTime) * data.ServerTickTime
	if effectiveDelayRounded == 0 {
		return
	}
	maxEffectiveAPS := 1 / effectiveDelayRounded
	output["KineticFusilladeMaxEffectiveAPS"] = maxEffectiveAPS

	if activeSkill.SkillPart != 1 {
		return
	}
	currentAPS := output["Speed"]
	if currentAPS <= maxEffectiveAPS {
		return
	}
	originalMultiplier := activeSkill.SkillData.DpsMultiplier
	if originalMultiplier == 0 {
		originalMultiplier = output["ProjectileCount"]
	}
	activeSkill.SkillData.DpsMultiplier = originalMultiplier * (maxEffectiveAPS / currentAPS)
}

//pob:block:src/Data/Skills/act_str.lua:7777-7801
func moltenStrikeOverlapChance(activeSkill *ActiveSkill, output map[string]float64) (float64, bool) {
	if activeSkill.SkillPart == 1 {
		return 0, false
	}

	skillModList, skillCfg := activeSkill.SkillModList, activeSkill.SkillCfg
	enemyRadius := skillModList.Sum(mod.TypeBase, skillCfg, "EnemyRadius")
	if override := skillModList.Override(skillCfg, "EnemyRadius"); override != nil {
		enemyRadius = override.Float()
	}
	ballRadius := output["AreaOfEffectRadius"]
	innerRadius := output["AreaOfEffectRadiusSecondary"]
	outerRadius := output["AreaOfEffectRadiusTertiary"]

	overlapChance := math.Min(1, (enemyRadius+ballRadius-innerRadius)/(outerRadius-innerRadius))
	if math.IsNaN(overlapChance) {
		return 0, false
	}
	output["OverlapChance"] = overlapChance * 100
	return overlapChance, true
}

//pob:block:src/Data/Skills/act_str.lua:7767-7819
func moltenStrikeOverlap(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
	overlapChance, ok := moltenStrikeOverlapChance(activeSkill, output)
	if !ok {
		return
	}
	if activeSkill.SkillPart == 3 {
		applyDpsMultiplier(activeSkill, output, overlapChance*output["ProjectileCount"])
	}
}

//pob:block:src/Data/Skills/act_str.lua:7948-8019
func moltenStrikeZenithOverlap(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
	overlapChance, ok := moltenStrikeOverlapChance(activeSkill, output)
	if !ok {
		return
	}
	numProjectiles := output["ProjectileCount"]
	switch activeSkill.SkillPart {
	case 3, 5:
		applyDpsMultiplier(activeSkill, output, overlapChance*numProjectiles)
	case 6:
		fifthAttackMulti := 1 + activeSkill.SkillData.FifthStrikeDamage/100
		fifthAttackOverallMulti := fifthAttackMulti * overlapChance * (numProjectiles + activeSkill.SkillData.FifthStrikeProjectiles)
		applyDpsMultiplier(activeSkill, output, float64(0.8*overlapChance*numProjectiles)+float64(0.2*fifthAttackOverallMulti))
	}
}

func durationScaledDamageInc(modName string, source mod.Source) skillFunc {
	return func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
		duration := math.Floor(activeSkill.SkillData.Duration * output["DurationMod"] * 10)
		activeSkill.SkillModList.NewMod("Damage", mod.TypeIncrease,
			activeSkill.SkillModList.Sum(mod.TypeIncrease, activeSkill.SkillCfg, modName)*duration, source)
	}
}

//pob:block:src/Data/Skills/act_int.lua:2413-2423
var bodyswapSelfExplosion skillFunc = func(activeSkill *ActiveSkill, output map[string]float64, _ *Breakdown) {
	if activeSkill.SkillPart != 1 {
		return
	}
	life := output["Life"]
	if activeSkill.SkillFlags[SkillFlagTotem] {
		life = output["TotemLife"]
	}
	activeSkill.SkillData.FireBonusMin = life * activeSkill.SkillData.SelfFireExplosionLifeMultiplier
	activeSkill.SkillData.FireBonusMax = life * activeSkill.SkillData.SelfFireExplosionLifeMultiplier
}

//pob:block:src/Data/Skills/act_dex.lua:6696-6775
func explosiveArrowFunc(activeSkill *ActiveSkill, output map[string]float64, globalOutput map[string]float64, env *Environment, enemyDB *moddb.ModDB) {
	if activeSkill.SkillPart != 1 && activeSkill.SkillPart != 2 {
		return
	}

	skillModList, skillCfg := activeSkill.SkillModList, activeSkill.SkillCfg
	duration := calcSkillDuration(skillModList, skillCfg, activeSkill.SkillData, env, enemyDB)
	fuseLimit := skillModList.Sum(mod.TypeBase, skillCfg, "ExplosiveArrowMaxFuseCount")

	isTotem := activeSkill.SkillFlags[SkillFlagTotem]
	activeTotems := float64(0)
	if isTotem {
		activeTotems = utils.Or(env.ModDB.Override(nil, "TotemsSummoned"),
			skillModList.Sum(mod.TypeBase, skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit"))
	}

	barrageProjectiles := float64(0)
	if skillModList.Flag(nil, "SequentialProjectiles") && !skillModList.Flag(nil, "OneShotProj") &&
		!skillModList.Flag(nil, "NoAdditionalProjectiles") && !skillModList.Flag(nil, "TriggeredBySnipe") {
		barrageProjectiles = skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount")
		if barrageProjectiles != 0 {
			activeSkill.SkillData.DpsMultiplier /= barrageProjectiles
		}
	}

	fuseApplicationRate := (output["HitChance"] / 100) * globalOutput["Speed"] *
		activeSkill.SkillData.DpsMultiplier * utils.OrF(barrageProjectiles, 1)
	if isTotem {
		fuseApplicationRate *= activeTotems
	}

	if activeSkill.SkillPart == 2 {
		maximum := math.Min(math.Floor(fuseApplicationRate*duration)+1, fuseLimit)
		skillModList.NewMod("Multiplier:ExplosiveArrowStage", mod.TypeBase, maximum, "Base")
		skillModList.NewMod("Multiplier:ExplosiveArrowStageAfterFirst", mod.TypeBase, maximum-1, "Base")
		globalOutput["MaxExplosiveArrowFuseCalculated"] = maximum
	} else {
		delete(globalOutput, "MaxExplosiveArrowFuseCalculated")
	}

	hitTime := duration
	if fuseApplicationRate != 0 {
		timeToMaxFuses := fuseLimit / fuseApplicationRate
		if activeSkill.SkillPart == 2 || activeSkill.ActiveStageCount+1 >= fuseLimit {
			hitTime = math.Min(duration, timeToMaxFuses)
		}
	}
	if hitTime <= 0 {
		return
	}
	globalOutput["HitTime"] = hitTime
	globalOutput["HitSpeed"] = 1 / hitTime
}

//pob:src/Data/Skills/act_dex.lua:2209-2211
func bladeVortexHitTime(activeSkill *ActiveSkill, blades float64) {
	divisor := 1 + float64(blades*activeSkill.SkillData.HitFrequencyPerBlade)
	if divisor == 0 {
		return
	}
	activeSkill.SkillData.HitTimeOverride = activeSkill.SkillData.HitFrequency / divisor
}

//pob:src/Data/Skills/act_int.lua:6800-6802
func flameblastHitTimeMultiplier(activeSkill *ActiveSkill, stageMultiplier string) {
	stages := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, stageMultiplier)
	minimum := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:FlameblastMinimumStage")
	activeSkill.SkillData.HitTimeMultiplier = math.Max(stages-minimum, 1)
}

//pob:src/Data/Skills/act_int.lua:10284-10288
func incinerateHitTimeMultiplier(activeSkill *ActiveSkill, stageMultiplier string) {
	if activeSkill.SkillPart != 2 {
		return
	}
	stages := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, stageMultiplier)
	minimum := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "Multiplier:IncinerateMinimumStage")
	activeSkill.SkillData.HitTimeMultiplier = math.Max(stages-minimum-0.4175, 0.5825)
}
