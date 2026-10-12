package calculator

import (
	"fmt"
	"math"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/calculator/calclib"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

//pob:block:src/Classes/ConfigTab.lua:1237-1238
func resolvedConfigOption(build *pob.PathOfBuilding, name string) string {
	return utils.OrS(build.GetStringOption(name), configIndexDefaults[name])
}

type costEntry struct {
	Type                         string
	Upfront                      bool
	Percent                      bool
	UnaffectedByGenericCostMults bool
	Text                         string
	BaseCost                     float64
	TotalCost                    float64
	BaseCostNoMult               float64
	FinalBaseCost                float64
	BaseCostRaw                  *float64
}

func (c *costEntry) zeroBaseCostRaw() {
	if c.BaseCostRaw == nil {
		return
	}
	zero := float64(0)
	c.BaseCostRaw = &zero
}

func costRound(up bool, v float64) float64 {
	if up {
		return math.Ceil(v)
	}
	return math.Floor(v)
}

//pob:block:src/Data/Costs.lua:5-88
var costDivisors = map[string]float64{
	"Mana":                  1,
	"Life":                  1,
	"ES":                    1,
	"Rage":                  1,
	"ManaPercent":           1,
	"LifePercent":           1,
	"UnreservedManaPercent": 1,
	"ManaPerMinute":         60,
	"LifePerMinute":         60,
	"ManaPercentPerMinute":  60,
	"LifePercentPerMinute":  60,
	"ESPerMinute":           60,
	"ESPercentPerMinute":    60,
	"Soul":                  1,
}

//pob:src/Modules/CalcOffence.lua:68-136
func calcDamage(activeSkill *ActiveSkill, output map[string]float64, cfg *moddb.ListCfg, breakdown *Breakdown, breakdownKey string, damageType data.DamageType, typeFlags int, convDst *data.DamageType) (float64, float64) {
	typeFlags = typeFlags | data.DamageTypeFlags[damageType]

	// Calculate conversions
	addMin := float64(0)
	addMax := float64(0)

	for _, otherType := range data.DamageType("").Values() {
		if otherType == damageType {
			// Damage can only be converted from damage types that precede this one in the conversion sequence, so stop here
			break
		}

		convMult := activeSkill.conversionTableFor(cfg)[otherType].Targets[damageType]
		if convMult > 0 {
			// Damage is being converted/gained from the other damage type
			minDamage, maxDamage := calcDamage(activeSkill, output, cfg, breakdown, breakdownKey, otherType, typeFlags, &damageType)
			addMin += minDamage * convMult
			addMax += maxDamage * convMult
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
		if breakdown != nil && (addMin != 0 || addMax != 0) {
			row := BDamageType{
				Source:  string(damageType),
				ConvSrc: utils.Ternary(addMin != 0 || addMax != 0, utils.Ptr(fmt.Sprint(addMin)+" to "+fmt.Sprint(addMax)), nil),
				Total:   fmt.Sprint(addMin) + " to " + fmt.Sprint(addMax),
			}
			if convDst != nil {
				table := activeSkill.conversionTableFor(cfg)[damageType]
				row.ConvDst = utils.Ptr(fmt.Sprintf("%d%% to %s", int(table.Conversion[*convDst]*100), *convDst))
				row.GainDst = utils.Ptr(fmt.Sprintf("%d%% gained as %s", int(table.Gain[*convDst]*100), *convDst))
			}
			breakdown.DamageType(breakdownKey, row)
		}
		return addMin, addMax
	}

	// Combine modifiers
	modNames := data.DamageStatsForType(typeFlags)
	inc := 1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, cfg, modNames...)/100
	more := activeSkill.SkillModList.More(cfg, modNames...)
	genericMoreMinDamage := activeSkill.SkillModList.More(cfg, "MinDamage")
	genericMoreMaxDamage := activeSkill.SkillModList.More(cfg, "MaxDamage")
	moreMinDamage := activeSkill.SkillModList.More(cfg, "Min"+string(damageType)+"Damage")
	moreMaxDamage := activeSkill.SkillModList.More(cfg, "Max"+string(damageType)+"Damage")
	incMinDamage := 1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, cfg, "Min"+string(damageType)+"Damage")/100
	incMaxDamage := 1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, cfg, "Max"+string(damageType)+"Damage")/100

	if breakdown != nil {
		row := BDamageType{
			Source:  string(damageType),
			Base:    utils.Ptr(fmt.Sprint(baseMin) + " to " + fmt.Sprint(baseMax)),
			Inc:     utils.Ternary(inc != 1, utils.Ptr("x "+fmt.Sprint(inc)), nil),
			More:    utils.Ternary(more != 1, utils.Ptr("x "+fmt.Sprint(more)), nil),
			ConvSrc: utils.Ternary(addMin != 0 || addMax != 0, utils.Ptr(fmt.Sprint(addMin)+" to "+fmt.Sprint(addMax)), nil),
			Total:   fmt.Sprint(utils.RoundTo(baseMin*inc*more, 0)+addMin) + " to " + fmt.Sprint(utils.RoundTo(baseMax*inc*more, 0)+addMax),
		}
		//pob:block:src/Modules/CalcOffence.lua:129-130
		if convDst != nil {
			table := activeSkill.conversionTableFor(cfg)[damageType]
			if table.Conversion[*convDst] > 0 {
				row.ConvDst = utils.Ptr(fmt.Sprintf("%d%% to %s", int(table.Conversion[*convDst]*100), *convDst))
			}
			if table.Gain[*convDst] > 0 {
				row.GainDst = utils.Ptr(fmt.Sprintf("%d%% gained as %s", int(table.Gain[*convDst]*100), *convDst))
			}
		}
		breakdown.DamageType(breakdownKey, row)
	}

	return math.Round(((baseMin*inc*more)*genericMoreMinDamage + addMin) * moreMinDamage * incMinDamage),
		math.Round(((baseMax*inc*more)*genericMoreMaxDamage + addMax) * moreMaxDamage * incMaxDamage)
}

//pob:src/Modules/CalcOffence.lua:127-137
func calcAilmentSourceDamage(activeSkill *ActiveSkill, output map[string]float64, cfg *moddb.ListCfg, breakdown *Breakdown, breakdownKey string, damageType data.DamageType, typeFlags int) (float64, float64) {
	minDamage, maxDamage := calcDamage(activeSkill, output, cfg, breakdown, breakdownKey, damageType, typeFlags, nil)
	convMult := activeSkill.conversionTableFor(cfg)[damageType].Mult
	if breakdown != nil && convMult != 1 {
		breakdown.AddLine(breakdownKey, "Source damage:")
		breakdown.AddLine(breakdownKey, fmt.Sprintf("%.2f to %.2f ^8(total damage)", minDamage, maxDamage))
		breakdown.AddLine(breakdownKey, fmt.Sprintf("x %g ^8(%g%% converted to other damage types)", convMult, (1-convMult)*100))
		breakdown.AddLine(breakdownKey, fmt.Sprintf("= %.2f to %.2f", minDamage*convMult, maxDamage*convMult))
	}
	return minDamage * convMult, maxDamage * convMult
}

func buildConversionTable(skillModList *moddb.ModList, cfg *moddb.ListCfg) map[data.DamageType]ConversionTable {
	table := make(map[data.DamageType]ConversionTable)
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
			globalConv[otherType] = math.Max(skillModList.Sum(mod.TypeBase, cfg, globalNames...), 0)
			globalTotal += globalConv[otherType]

			skillConv[otherType] = math.Max(skillModList.Sum(mod.TypeBase, cfg, "Skill"+string(damageType)+"DamageConvertTo"+string(otherType)), 0)
			skillTotal += skillConv[otherType]

			addNames := []string{string(damageType) + "DamageGainAs" + string(otherType)}
			if damageType.IsElemental() {
				addNames = append(addNames, "ElementalDamageGainAs"+string(otherType))
			}
			if damageType != data.DamageTypeChaos {
				addNames = append(addNames, "NonChaosDamageGainAs"+string(otherType))
			}
			add[otherType] = math.Max(skillModList.Sum(mod.TypeBase, cfg, addNames...), 0)
		}

		if skillTotal > 100 {
			// Skill conversion exceeds 100%, scale it down and remove non-skill conversions
			factor := 100 / skillTotal
			for convType, val := range skillConv {
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
			Targets:    make(map[data.DamageType]float64),
			Conversion: make(map[data.DamageType]float64),
			Gain:       make(map[data.DamageType]float64),
		}
		for convType, val := range globalConv {
			dmgTable.Conversion[convType] = (val + skillConv[convType]) / 100
			dmgTable.Gain[convType] = add[convType] / 100
			dmgTable.Targets[convType] = (val + skillConv[convType] + add[convType]) / 100
		}

		dmgTable.Mult = 1 - math.Min((globalTotal+skillTotal)/100, 1)
		table[damageType] = dmgTable
	}

	table[data.DamageTypeChaos] = ConversionTable{
		Mult: 1,
	}
	return table
}

func (a *ActiveSkill) conversionTableFor(cfg *moddb.ListCfg) map[data.DamageType]ConversionTable {
	//pob:src/Modules/CalcOffence.lua:75
	if cfg != nil {
		if t, ok := a.CfgConversionTable[cfg]; ok {
			return t
		}
	}
	return a.ConversionTable
}

//pob:src/Modules/CalcOffence.lua:155-157
func calcRadius(baseRadius float64, areaMod float64) float64 {
	return math.Floor(baseRadius * math.Floor(100*math.Sqrt(areaMod)) / 100)
}

type radiusBreakpoints struct{ inc, more, red, less int }

//pob:src/Modules/CalcOffence.lua:176-203
func calcRadiusBreakpoints(baseRadius float64, incArea float64, moreArea float64) *radiusBreakpoints {
	radius := calcRadius(baseRadius, utils.RoundTo(utils.RoundTo(incArea*moreArea, 10), 2))
	if radius <= 0 {
		return nil
	}
	bp := &radiusBreakpoints{}
	for {
		bp.inc++
		if calcRadius(baseRadius, utils.RoundTo(utils.RoundTo((incArea+float64(bp.inc)/100)*moreArea, 10), 2)) > radius {
			break
		}
	}
	for {
		bp.red++
		if calcRadius(baseRadius, utils.RoundTo(utils.RoundTo((incArea-float64(bp.red)/100)*moreArea, 10), 2)) < radius {
			break
		}
	}
	for {
		bp.more++
		if calcRadius(baseRadius, utils.RoundTo(utils.RoundTo(incArea*moreArea*(1+float64(bp.more)/100), 10), 2)) > radius {
			break
		}
	}
	for {
		bp.less++
		if calcRadius(baseRadius, utils.RoundTo(utils.RoundTo(incArea*moreArea*(1-float64(bp.less)/100), 10), 2)) < radius {
			break
		}
	}
	return bp
}

//pob:src/Modules/CalcOffence.lua:164-169
func calcMoltenStrikeTertiaryRadius(baseRadius float64, deadzoneRadius float64, areaMod float64, speedMod float64) float64 {
	maxDistIgnoringSpeed := math.Sqrt(baseRadius*baseRadius*areaMod - deadzoneRadius*deadzoneRadius*(areaMod-1))
	return math.Floor((maxDistIgnoringSpeed-deadzoneRadius)*speedMod + deadzoneRadius)
}

//pob:src/Modules/CalcOffence.lua:1049
var pointBlankDistanceRamp = [][]float64{{10, 1}, {35, 0}, {120, -1}}

//pob:src/Modules/CalcOffence.lua:1052
var farShotDistanceRamp = [][]float64{{10, -0.2}, {25, 0}, {70, 0.6}}

var nonDamagingAilmentList = []data.Ailment{
	data.AilmentChill,
	data.AilmentShock,
	data.AilmentScorch,
	data.AilmentBrittle,
	data.AilmentSap,
}

var damageTypeDotKeywordFlags = map[data.DamageType]mod.KeywordFlag{
	data.DamageTypePhysical:  mod.KeywordFlagPhysicalDot,
	data.DamageTypeLightning: mod.KeywordFlagLightningDot,
	data.DamageTypeCold:      mod.KeywordFlagColdDot,
	data.DamageTypeFire:      mod.KeywordFlagFireDot,
	data.DamageTypeChaos:     mod.KeywordFlagChaosDot,
}

//pob:block:src/Modules/CalcOffence.lua:5689-5694
func skillDotBase(skillData *SkillData, damageType data.DamageType) float64 {
	switch damageType {
	case data.DamageTypePhysical:
		return skillData.PhysicalDot
	case data.DamageTypeLightning:
		return skillData.LightningDot
	case data.DamageTypeCold:
		return skillData.ColdDot
	case data.DamageTypeFire:
		return skillData.FireDot
	case data.DamageTypeChaos:
		return skillData.ChaosDot
	}
	return 0
}

//pob:block:src/Modules/CalcOffence.lua:1443-1444
func copyModAs(src mod.Mod, name string) mod.Mod {
	return mod.NewFloat(name, src.Type(), src.Value().Float()).
		Source(src.GetSource()).
		Flag(src.Flags()).
		KeywordFlag(src.KeywordFlags())
}

func activeSkillSlot(activeSkill *ActiveSkill) string {
	if activeSkill == nil {
		return ""
	}
	if socketGroup, ok := activeSkill.SocketGroup.(*pob.Skill); ok && socketGroup != nil {
		return socketGroup.Slot
	}
	return ""
}

//pob:src/Modules/CalcOffence.lua:262-267
func calcSkillCooldown(skillModList *moddb.ModList, skillCfg *moddb.ListCfg, skillData *SkillData) float64 {
	cooldown := utils.Or(
		skillModList.Override(skillCfg, "CooldownRecovery"),
		(skillData.Cooldown+skillModList.Sum(mod.TypeBase, skillCfg, "CooldownRecovery"))/max(0, calclib.Mod(skillModList, skillCfg, "CooldownRecovery")),
	)
	if (skillData.StoredUses != nil && *skillData.StoredUses > 1) ||
		skillModList.Sum(mod.TypeBase, skillCfg, "AdditionalCooldownUses") > 0 {
		return cooldown
	}
	return math.Ceil(cooldown*data.ServerTickRate) / data.ServerTickRate
}

//pob:src/Modules/CalcOffence.lua:269-278
func calcWarcryCastTime(skillModList *moddb.ModList, skillCfg *moddb.ListCfg, skillData *SkillData, actor *Actor) float64 {
	baseSpeed := 1 / skillModList.Sum(mod.TypeBase, skillCfg, "WarcryCastTime")
	warcryCastTime := baseSpeed * calclib.Mod(skillModList, skillCfg, "WarcrySpeed") * CalcActionSpeedMod(actor)
	warcryCastTime = min(warcryCastTime, data.ServerTickRate)
	warcryCastTime = 1 / warcryCastTime
	if skillModList.Flag(skillCfg, "InstantWarcry") || (skillData != nil && skillData.TriggeredByAutoexertion) {
		warcryCastTime = 0
	}
	return warcryCastTime
}

//pob:src/Modules/CalcOffence.lua:280-292
func calcSkillDuration(skillModList *moddb.ModList, skillCfg *moddb.ListCfg, skillData *SkillData, env *Environment, enemyDB *moddb.ModDB) float64 {
	durationMod := durationModClamped(skillModList, skillCfg, "Duration", "PrimaryDuration", utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
	durationBase := skillData.Duration + skillModList.Sum(mod.TypeBase, skillCfg, "Duration", "PrimaryDuration")
	duration := durationBase * durationMod
	debuffDurationMult := float64(1)
	if env.ModeEffective {
		debuffDurationMult = 1 / max(data.BuffExpirationSlowCap, calclib.Mod(enemyDB, skillCfg, "BuffExpireFaster"))
	}
	if skillData.Debuff {
		duration = duration * debuffDurationMult
	}
	return duration
}

//pob:block:src/Export/Scripts/miscdata.lua:95-102
func totemLifeMult(skillTotemID int) (float64, bool) {
	for _, variation := range poe.SkillTotemVariations {
		if variation.SkillTotemsKey != skillTotemID {
			continue
		}
		return float64(raw.MonsterVarieties()[variation.MonsterVarietiesKey].LifeMultiplier) / 100, true
	}
	return 0, false
}

//pob:block:src/Modules/CalcMirages.lua:123
func saviourMirageWarriorCandidate(skill *ActiveSkill, mainSkill *ActiveSkill) bool {
	const swordOneHand = mod.MFlagSword | mod.MFlagWeapon1H
	if skill == mainSkill || skill.SkillCfg == nil {
		return false
	}
	if !skill.SkillTypes[data.SkillTypeAttack] {
		return false
	}
	if skill.SkillTypes[data.SkillTypeSummonsTotem] {
		return false
	}
	if skill.Marked || skill.SkillCfg.SkillCond["usedByMirage"] {
		return false
	}
	return skill.SkillCfg.Flags != nil && *skill.SkillCfg.Flags&swordOneHand == swordOneHand
}

//pob:src/Modules/CalcOffence.lua:294-4092
func CalculateOffence(env *Environment, actor *Actor, activeSkill *ActiveSkill) {
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	outputTable := actor.OutputTable
	breakdown := actor.Breakdown
	skillModList := activeSkill.SkillModList
	skillData := activeSkill.SkillData
	skillFlags := activeSkill.SkillFlags
	skillCfg := activeSkill.SkillCfg

	if skillData.ShowAverage {
		skillFlags[SkillFlagShowAverage] = true
	} else {
		skillFlags[SkillFlagNotAverage] = true
	}

	if skillFlags[SkillFlagDisable] {
		// Skill is disabled
		actor.Output["CombinedDPS"] = 0
		return
	}

	//pob:src/Modules/CalcOffence.lua:461-474
	calcResistForType := func(damageType data.DamageType, cfg *moddb.ListCfg) float64 {
		maxResist := float64(data.EnemyMaxResist)
		if !enemyDB.Flag(nil, "DoNotChangeMaxResFromConfig") {
			configured := float64(data.EnemyMaxResist)
			if v, ok := env.Build.GetNumberOptionOK("enemy" + string(damageType) + "Resist"); ok {
				configured = v
			}
			maxResist = math.Min(math.Max(configured, data.EnemyMaxResist), data.MaxResistCap)
		}

		var resist float64
		switch override := enemyDB.Override(cfg, string(damageType)+"Resist"); {
		case override != nil:
			resist = override.Float()
		case env.ModDB.Flag(nil, "Enemy"+string(damageType)+"ResistEqualToYours"):
			resist = env.Player.Output[string(damageType)+"Resist"]
		default:
			names := []string{string(damageType) + "Resist"}
			if damageType.IsElemental() {
				names = append(names, "ElementalResist")
			}
			resist = enemyDB.Sum(mod.TypeBase, cfg, names...) * math.Max(CalcMod(enemyDB, cfg, names...), 0)
		}
		return math.Max(math.Min(resist, maxResist), data.ResistFloor)
	}

	calcAreaOfEffect := func(skillModList *moddb.ModList, skillCfg *moddb.ListCfg, skillData *SkillData, skillFlags map[SkillFlag]bool, output map[string]float64, breakdown *Breakdown) {
		//pob:src/Modules/CalcOffence.lua:346
		incArea, moreArea := calclib.Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectPrimary")
		output["AreaOfEffectMod"] = utils.RoundTo(utils.RoundTo(incArea*moreArea, 10), 2)
		if skillData.RadiusIsWeaponRange {
			rangeVal := float64(0)
			if skillFlags[SkillFlagWeapon1Attack] {
				rangeVal = max(rangeVal, actor.WeaponRange1)
			}
			if skillFlags[SkillFlagWeapon2Attack] {
				rangeVal = max(rangeVal, actor.WeaponRange2)
			}
			skillData.Radius = rangeVal + 2
		}
		if skillData.Radius != 0 {
			skillFlags[SkillFlagArea] = true
			baseRadius := skillData.Radius + skillData.RadiusExtra + skillModList.Sum(mod.TypeBase, skillCfg, "AreaOfEffect")
			output["AreaOfEffectRadius"] = calcRadius(baseRadius, output["AreaOfEffectMod"])
			output["AreaOfEffectRadiusMetres"] = output["AreaOfEffectRadius"] / 10
			if breakdown != nil {
				breakdown.area("AreaOfEffectRadius", baseRadius, output["AreaOfEffectMod"], output["AreaOfEffectRadius"], calcRadiusBreakpoints(baseRadius, incArea, moreArea), "")
			}
			if skillData.RadiusSecondary != 0 {
				incAreaSecondary, moreAreaSecondary := calclib.Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectSecondary")
				output["AreaOfEffectModSecondary"] = utils.RoundTo(utils.RoundTo(incAreaSecondary*moreAreaSecondary, 10), 2)
				baseRadius = skillData.RadiusSecondary + skillData.RadiusExtra
				output["AreaOfEffectRadiusSecondary"] = calcRadius(baseRadius, output["AreaOfEffectModSecondary"])
				output["AreaOfEffectRadiusSecondaryMetres"] = output["AreaOfEffectRadiusSecondary"] / 10
				if breakdown != nil {
					var bp *radiusBreakpoints
					if !skillData.ProjectileSpeedAppliesToMSAreaOfEffect {
						bp = calcRadiusBreakpoints(baseRadius, incAreaSecondary, moreAreaSecondary)
					}
					breakdown.area("AreaOfEffectRadiusSecondary", baseRadius, output["AreaOfEffectModSecondary"], output["AreaOfEffectRadiusSecondary"], bp, "")
				}
			}
			if skillData.RadiusTertiary != 0 {
				incAreaTertiary, moreAreaTertiary := calclib.Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectTertiary")
				output["AreaOfEffectModTertiary"] = utils.RoundTo(utils.RoundTo(incAreaTertiary*moreAreaTertiary, 10), 2)
				baseRadius = skillData.RadiusTertiary + skillData.RadiusExtra
				if skillData.ProjectileSpeedAppliesToMSAreaOfEffect {
					incSpeedTertiary, moreSpeedTertiary := calclib.Mods(skillModList, skillCfg, "ProjectileSpeed")
					output["SpeedModTertiary"] = utils.RoundTo(utils.RoundTo(incSpeedTertiary*moreSpeedTertiary, 10), 2)
					output["AreaOfEffectRadiusTertiary"] = calcMoltenStrikeTertiaryRadius(baseRadius, skillData.RadiusSecondary, output["AreaOfEffectModTertiary"], output["SpeedModTertiary"])
				} else if skillData.RadiusTertiaryBaseMargin != 0 {
					//pob:block:src/Modules/CalcOffence.lua:395-415
					margin := skillData.RadiusTertiaryBaseMargin / 100
					marginWidth := skillData.RadiusTertiaryBaseMargin*2 + 1
					baseRadiiOccurrences := make(map[float64]float64)
					for deviation := 1 - margin; deviation <= 1+margin+0.01; deviation += 0.01 {
						radiusForDeviation := math.Floor(baseRadius * deviation)
						baseRadiiOccurrences[radiusForDeviation]++
					}
					sumOfRandomRadii := float64(0)
					radiiOccurrences := make(map[float64]float64)
					for adjustedBaseRadius, occurrenceCount := range baseRadiiOccurrences {
						radiusForDeviation := calcRadius(adjustedBaseRadius, output["AreaOfEffectModTertiary"])
						sumOfRandomRadii += radiusForDeviation * occurrenceCount
						radiiOccurrences[radiusForDeviation] += occurrenceCount
					}
					output["AreaOfEffectRadiusTertiary"] = sumOfRandomRadii / marginWidth
					activeSkill.areaOfEffectRadiusTertiaryOccurrences = radiiOccurrences
				} else {
					output["AreaOfEffectRadiusTertiary"] = calcRadius(baseRadius, output["AreaOfEffectModTertiary"])
					if breakdown != nil {
						breakdown.area("AreaOfEffectRadiusTertiary", baseRadius, output["AreaOfEffectModTertiary"], output["AreaOfEffectRadiusTertiary"], calcRadiusBreakpoints(baseRadius, incAreaTertiary, moreAreaTertiary), "")
					}
				}
				output["AreaOfEffectRadiusTertiaryMetres"] = output["AreaOfEffectRadiusTertiary"] / 10
			}
		}
	}

	runSkillFunc := func(name string) {
		f := activeSkill.ActiveEffect.GrantedEffect.Funcs[name]
		if f != nil {
			f(activeSkill, actor.Output, breakdown)
		}
	}

	runSkillFunc("initialFunc")
	if skillCfg.SkillCond == nil {
		skillCfg.SkillCond = make(map[string]bool)
	}
	//pob: Modules/CalcOffence.lua:485-488
	skillCfg.SkillCond["SkillIsTriggered"] = skillData.Triggered
	if skillCfg.SkillCond["SkillIsTriggered"] {
		skillFlags[SkillFlagTriggered] = true
	}
	skillCfg.SkillCond["SkillIsFocused"] = skillData.TriggeredByFocus
	if skillCfg.SkillCond["SkillIsFocused"] {
		skillFlags[SkillFlagFocused] = true
	}
	// Update skill data
	for _, value := range skillModList.List(skillCfg, "SkillData") {
		cVal := value.(mod.SkillData)
		if cVal.Merge == "MAX" {
			utils.Set(skillData, cVal.Key, max(cVal.Value, utils.GetOr[float64](skillData, cVal.Key, 0)))
		} else {
			utils.Set(skillData, cVal.Key, cVal.Value)
		}
	}
	// Add addition stat bonuses
	if skillModList.Flag(nil, "IronGrip") {
		skillModList.AddMod(mod.NewFloat("PhysicalDamage", "INC", actor.StrDmgBonus).Source("Strength").Flag(mod.MFlagAttack | mod.MFlagProjectile))
	}
	if skillModList.Flag(nil, "IronWill") {
		skillModList.AddMod(mod.NewFloat("Damage", "INC", actor.StrDmgBonus).Source("Strength").Flag(mod.MFlagSpell))
	}

	if skillModList.Flag(nil, "TransfigurationOfBody") {
		skillModList.AddMod(mod.NewFloat("Damage", "INC", math.Floor(skillModList.Sum(mod.TypeIncrease, nil, "Life")*data.Transfiguration)).Source("Transfiguration of Body").Flag(mod.MFlagAttack))
	}
	if skillModList.Flag(nil, "TransfigurationOfMind") {
		skillModList.AddMod(mod.NewFloat("Damage", "INC", math.Floor(skillModList.Sum(mod.TypeIncrease, nil, "Mana")*data.Transfiguration)).Source("Transfiguration of Mind"))
	}
	if skillModList.Flag(nil, "TransfigurationOfSoul") {
		skillModList.AddMod(mod.NewFloat("Damage", "INC", math.Floor(skillModList.Sum(mod.TypeIncrease, nil, "EnergyShield")*data.Transfiguration)).Source("Transfiguration of Soul").Flag(mod.MFlagSpell))
	}

	if modDB.Flag(nil, "Elusive") && skillModList.Flag(nil, "SupportedByNightblade") {
		elusiveEffect := actor.Output["ElusiveEffectMod"] / 100
		nightbladeMulti := skillModList.Sum(mod.TypeBase, nil, "NightbladeElusiveCritMultiplier")
		skillModList.AddMod(mod.NewFloat("CritMultiplier", "BASE", math.Floor(nightbladeMulti*elusiveEffect)).Source("Nightblade"))
	}
	if skillModList.Flag(nil, "UseEnduranceCharges") && skillModList.Flag(nil, "EnduranceChargesConvertToBrutalCharges") {
		tripleDmgChancePerEndurance := modDB.Sum(mod.TypeBase, nil, "PerBrutalTripleDamageChance")
		modDB.AddMod(mod.NewFloat("TripleDamageChance", "BASE", tripleDmgChancePerEndurance).Tag(mod.Multiplier("BrutalCharge")))
	}
	if skillModList.Flag(nil, "UseFrenzyCharges") && skillModList.Flag(nil, "FrenzyChargesConvertToAfflictionCharges") {
		dmgPerAffliction := modDB.Sum(mod.TypeBase, nil, "PerAfflictionAilmentDamage")
		effectPerAffliction := modDB.Sum(mod.TypeBase, nil, "PerAfflictionNonDamageEffect")
		modDB.AddMod(mod.NewFloat("Damage", "MORE", dmgPerAffliction).Source("Affliction Charges").KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemyChillEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemyShockEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemyFreezeEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemyScorchEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemyBrittleEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemySapEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
	}
	// set other limits
	actor.Output["ActiveTrapLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveTrapLimit")
	actor.Output["ActiveMineLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveMineLimit")

	// set flask scaling
	//pob:block:src/Modules/CalcOffence.lua:532-533
	actor.Output["LifeFlaskRecovery"] = env.ItemModDB.Multipliers["LifeFlaskRecovery"]
	actor.Output["LifeFlaskCharges"] = env.ItemModDB.Multipliers["LifeFlaskCharges"]

	//pob:block:src/Modules/CalcOffence.lua:535-550
	if modDB.Conditions[conditionAffectedByEnergyBlade] {
		dmgMod := calclib.Mod(skillModList, skillCfg, "EnergyBladeDamage")
		speedMod := calclib.Mod(skillModList, skillCfg, "EnergyBladeAttackSpeed")

		for _, hand := range []struct {
			slotName   string
			weaponData *SkillData
		}{
			{"Weapon 1", actor.WeaponData1},
			{"Weapon 2", actor.WeaponData2},
		} {
			item := actor.ItemList[hand.slotName]
			if item == nil || len(item.WeaponData) == 0 || hand.weaponData == nil {
				continue
			}
			if wd, _ := item.WeaponData[0].(*SkillData); wd == nil {
				continue
			}
			base := ItemBaseByName(item.Name)
			if base == nil || base.Weapon == nil {
				continue
			}

			hand.weaponData.CritChance = base.Weapon.CritChanceBase
			hand.weaponData.AttackRate = base.Weapon.AttackRateBase * speedMod
			hand.weaponData.Range = float64(base.Weapon.Range)

			for _, damageType := range data.DamageType("").Values() {
				baseMin, baseMax := float64(0), float64(0)
				if damageType == data.DamageTypePhysical {
					baseMin, baseMax = float64(base.Weapon.PhysicalMin), float64(base.Weapon.PhysicalMax)
				}
				utils.Set(hand.weaponData, string(damageType+"Min"), baseMin+math.Floor(skillModList.Sum(mod.TypeBase, skillCfg, string("EnergyBladeMin"+damageType))*dmgMod))
				utils.Set(hand.weaponData, string(damageType+"Max"), baseMax+math.Floor(skillModList.Sum(mod.TypeBase, skillCfg, string("EnergyBladeMax"+damageType))*dmgMod))
			}
		}
	}
	// account for Battlemage
	// Note: we check conditions of Main Hand weapon using actor.itemList as actor.weaponData1 is populated with unarmed values when no weapon slotted.
	if skillModList.Flag(nil, "Battlemage") && actor.ItemList["Weapon 1"] != nil && actor.ItemList["Weapon 1"].WeaponData != nil && actor.ItemList["Weapon 1"].WeaponData[1] != nil {
		multiplier := utils.OrF(skillModList.Max(skillCfg, "MainHandWeaponDamageAppliesToSpells"), 100) / 100
		for _, damageType := range data.DamageType("").Values() {
			skillModList.AddMod(mod.NewFloat(string(damageType+"Min"), "BASE", utils.GetOr[float64](actor.WeaponData1, string(damageType+"Min"), 0)*multiplier).Source("Battlemage").Flag(mod.MFlagSpell))
			skillModList.AddMod(mod.NewFloat(string(damageType+"Max"), "BASE", utils.GetOr[float64](actor.WeaponData1, string(damageType+"Max"), 0)*multiplier).Source("Battlemage").Flag(mod.MFlagSpell))
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:562-579
	weapon1Info := data.WeaponTypes[actor.WeaponData1.Type]
	weapon2Info := data.WeaponTypes[actor.WeaponData2.Type]
	spellbladeMulti := skillModList.Max(skillCfg, "OneHandWeaponDamageAppliesToSpells")
	if spellbladeMulti != 0 && actor.ItemList["Weapon 1"] != nil && actor.ItemList["Weapon 1"].WeaponData != nil &&
		actor.ItemList["Weapon 1"].WeaponData[1] != nil && weapon1Info != nil && weapon1Info.Melee && weapon1Info.OneHand {
		multiplier := spellbladeMulti / 100
		if weapon2Info != nil {
			multiplier *= 0.6
		}
		for _, damageType := range data.DamageType("").Values() {
			skillModList.AddMod(mod.NewFloat(string(damageType+"Min"), "BASE",
				math.Floor(utils.GetOr[float64](actor.WeaponData1, string(damageType+"Min"), 0)*multiplier)).Source("Spellblade Main Hand").Flag(mod.MFlagSpell))
			skillModList.AddMod(mod.NewFloat(string(damageType+"Max"), "BASE",
				math.Floor(utils.GetOr[float64](actor.WeaponData1, string(damageType+"Max"), 0)*multiplier)).Source("Spellblade Main Hand").Flag(mod.MFlagSpell))
		}
		if weapon2Info != nil {
			for _, damageType := range data.DamageType("").Values() {
				skillModList.AddMod(mod.NewFloat(string(damageType+"Min"), "BASE",
					math.Floor(utils.GetOr[float64](actor.WeaponData2, string(damageType+"Min"), 0)*multiplier)).Source("Spellblade Off Hand").Flag(mod.MFlagSpell))
				skillModList.AddMod(mod.NewFloat(string(damageType+"Max"), "BASE",
					math.Floor(utils.GetOr[float64](actor.WeaponData2, string(damageType+"Max"), 0)*multiplier)).Source("Spellblade Off Hand").Flag(mod.MFlagSpell))
			}
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:580
	if skillModList.Flag(nil, "MinionDamageAppliesToPlayer") || skillModList.Flag(skillCfg, "MinionDamageAppliesToPlayer") {
		// Minion Damage conversion from Spiritual Aid and The Scourge
		multiplier := utils.OrF(skillModList.Max(skillCfg, "ImprovedMinionDamageAppliesToPlayer"), 100) / 100
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			cValue := value.(mod.MinionModifier)
			if cValue.Mod.Name() == "Damage" && cValue.Mod.Type() == "INC" {
				Mod := cValue.Mod
				modifiers := calclib.GetConvertedModTags(Mod, multiplier, true)
				skillModList.AddMod(mod.NewFloat("Damage", "INC", Mod.Value().Float()*multiplier).Source(Mod.GetSource()).Flag(Mod.Flags()).KeywordFlag(Mod.KeywordFlags()).Tag(modifiers...))
			}
		}
	}
	if skillModList.Flag(nil, "MinionAttackSpeedAppliesToPlayer") {
		multiplier := utils.OrF(skillModList.Max(skillCfg, "ImprovedMinionAttackSpeedAppliesToPlayer"), 100) / 100
		// Minion Attack Speed conversion from Spiritual Command
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			cValue := value.(mod.MinionModifier)
			if cValue.Mod.Name() == "Speed" && cValue.Mod.Type() == "INC" && (cValue.Mod.Flags() == 0 || (cValue.Mod.Flags()&mod.MFlagAttack) != 0) {
				modifiers := calclib.GetConvertedModTags(cValue.Mod, multiplier, true)
				skillModList.AddMod(mod.NewFloat("Speed", "INC", cValue.Mod.Value().Float()*multiplier).Source(cValue.Mod.GetSource()).Flag(mod.MFlagAttack).KeywordFlag(cValue.Mod.KeywordFlags()).Tag(modifiers...))
			}
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:611-617
	if skillModList.Flag(nil, "EvasionAppliesToSpellDamage") {
		for _, value := range skillModList.Tabulate("INC", nil, "Evasion") {
			Mod := value.Mod
			skillModList.AddMod(mod.NewFloat("Damage", Mod.Type(), Mod.Value().Float()).
				Source(Mod.GetSource()).
				Flag(mod.MFlagSpell).
				KeywordFlag(Mod.KeywordFlags()).
				Tag(Mod.Tags()...))
		}
	}
	if skillModList.Flag(nil, "SpellDamageAppliesToAttacks") || skillModList.Flag(skillCfg, "SpellDamageAppliesToAttacks") {
		// Spell Damage conversion from Crown of Eyes, Kinetic Bolt, and the Wandslinger notable
		multiplier := utils.OrF(skillModList.Max(skillCfg, "ImprovedSpellDamageAppliesToAttacks"), 100) / 100
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagSpell)}, "Damage") {
			Mod := value.Mod
			if (Mod.Flags() & mod.MFlagSpell) != 0 {
				modifiers := calclib.GetConvertedModTags(Mod, multiplier, false)
				skillModList.AddMod(mod.NewFloat("Damage", "INC", Mod.Value().Float()*multiplier).Source(Mod.GetSource()).Flag((Mod.Flags() & (^mod.MFlagSpell)) | mod.MFlagAttack).KeywordFlag(Mod.KeywordFlags()).Tag(modifiers...))
				if Mod.GetSource() == "Strength" { // Prevent double-dipping from converted strength's damage bonus
					skillModList.ReplaceMod(mod.NewFloat("PhysicalDamage", mod.TypeIncrease, 0).Source("Strength").Flag(mod.MFlagMelee))
				}
			}
		}
	}
	if skillModList.Flag(nil, "CastSpeedAppliesToAttacks") {
		// Get all increases for this; assumption is that multiple sources would not stack, so find the max
		multiplier := utils.OrF(skillModList.Max(skillCfg, "ImprovedCastSpeedAppliesToAttacks"), 100) / 100
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagCast)}, "Speed") {
			// Add a new mod for all mods that are cast only
			// Replace this with a single mod for the sum?
			if (value.Mod.Flags() & mod.MFlagCast) != 0 {
				modifiers := calclib.GetConvertedModTags(value.Mod, multiplier, false)
				skillModList.AddMod(mod.NewFloat("Speed", "INC", value.Mod.Value().Float()*multiplier).Source(value.Mod.GetSource()).Flag((value.Mod.Flags() & (^mod.MFlagCast)) | mod.MFlagAttack).KeywordFlag(value.Mod.KeywordFlags()).Tag(modifiers...))
			}
		}
	}
	if skillModList.Flag(nil, "ProjectileSpeedAppliesToBowDamage") {
		// Bow mastery projectile speed to damage with bows conversion
		for _, value := range skillModList.Tabulate("INC", nil, "ProjectileSpeed") {
			skillModList.AddMod(mod.NewFloat("Damage", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(mod.MFlagBow | mod.MFlagHit).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
		}
	}
	if skillModList.Flag(nil, "ClawDamageAppliesToUnarmed") {
		// Claw Damage conversion from Rigwald's Curse
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagClaw | mod.MFlagHit), KeywordFlags: utils.Ptr(mod.KeywordFlagHit)}, "Damage") {
			if (value.Mod.Flags() & mod.MFlagClaw) != 0 {
				skillModList.AddMod(mod.NewFloat("Damage", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag((value.Mod.Flags() & (^mod.MFlagClaw)) | mod.MFlagUnarmed | mod.MFlagMelee).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawAttackSpeedAppliesToUnarmed") {
		// Claw Attack Speed conversion from Rigwald's Curse
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagClaw | mod.MFlagAttack | mod.MFlagHit)}, "Speed") {
			if (value.Mod.Flags()&mod.MFlagClaw) != 0 && (value.Mod.Flags()&mod.MFlagAttack) != 0 {
				skillModList.AddMod(mod.NewFloat("Speed", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag((value.Mod.Flags() & (^mod.MFlagClaw)) | mod.MFlagUnarmed).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritChanceAppliesToUnarmed") {
		// Claw Crit Chance conversion from Rigwald's Curse
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagClaw | mod.MFlagHit)}, "CritChance") {
			if (value.Mod.Flags() & mod.MFlagClaw) != 0 {
				skillModList.AddMod(mod.NewFloat("CritChance", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag((value.Mod.Flags() & (^mod.MFlagClaw)) | mod.MFlagUnarmed).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritChanceAppliesToMinions") {
		// Claw Crit Chance conversion from Law of the Wilds
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagClaw | mod.MFlagHit)}, "CritChance") {
			if (value.Mod.Flags() & mod.MFlagClaw) != 0 {
				env.Minion.ModDB.AddMod(mod.NewFloat("CritChance", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritMultiplierAppliesToMinions") {
		// Claw Crit Multi conversion from Law of the Wilds
		for _, value := range skillModList.Tabulate("BASE", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagClaw | mod.MFlagHit)}, "CritMultiplier") {
			if (value.Mod.Flags() & mod.MFlagClaw) != 0 {
				env.Minion.ModDB.AddMod(mod.NewFloat("CritMultiplier", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()))
			}
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:725-731
	if skillModList.Flag(nil, "FirePenIncreasedByUncappedFireRes") {
		if tab := modDB.Tabulate(mod.TypeFlag, nil, "FirePenIncreasedByUncappedFireRes"); len(tab) > 0 {
			skillModList.AddMod(mod.NewFloat("FirePenetration", mod.TypeBase, actor.Output["FireResistOverCap"]).
				Source(tab[0].Mod.GetSource()))
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToAccuracy") {
		// Light Radius conversion from Corona Solaris
		for _, value := range skillModList.Tabulate("INC", nil, "LightRadius") {
			skillModList.AddMod(mod.NewFloat("Accuracy", "INC", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToAreaOfEffect") {
		// Light Radius conversion from Wreath of Phrecia
		for _, value := range skillModList.Tabulate("INC", nil, "LightRadius") {
			skillModList.AddMod(mod.NewFloat("AreaOfEffect", "INC", math.Floor(value.Mod.Value().Float()/2)).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToDamage") {
		// Light Radius conversion from Wreath of Phrecia
		for _, value := range skillModList.Tabulate("INC", nil, "LightRadius") {
			skillModList.AddMod(mod.NewFloat("Damage", "INC", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
		}
	}
	if skillModList.Flag(nil, "CastSpeedAppliesToTrapThrowingSpeed") {
		// Cast Speed conversion from Slavedriver's Hand
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagCast)}, "Speed") {
			if value.Mod.Flags() == 0 || (value.Mod.Flags()&mod.MFlagCast) != 0 {
				skillModList.AddMod(mod.NewFloat("TrapThrowingSpeed", "INC", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags() & (^mod.MFlagCast) & (^mod.MFlagAttack)).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			}
		}
	}
	if skillData.ArrowSpeedAppliesToAreaOfEffect {
		// Arrow Speed conversion for Galvanic Arrow
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagBow)}, "ProjectileSpeed") {
			skillModList.AddMod(mod.NewFloat("AreaOfEffect", "INC", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
		}
	}
	//pob:src/Modules/CalcOffence.lua:769-772
	if skillModList.Flag(nil, "SequentialProjectiles") && !skillModList.Flag(nil, "OneShotProj") && !skillModList.Flag(nil, "NoAdditionalProjectiles") && !skillModList.Flag(nil, "SingleProjectile") && !skillModList.Flag(nil, "TriggeredBySnipe") {
		// Applies DPS multiplier based on projectile count
		skillData.DpsMultiplier = skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount")
	}

	//pob:block:src/Modules/CalcOffence.lua:774-785
	repeatAllowed := !skillModList.Flag(nil, "CannotRepeat") &&
		(activeSkill.SkillTypes[data.SkillTypeAttack] || activeSkill.SkillTypes[data.SkillTypeSpell])
	for _, excluded := range []data.SkillType{
		data.SkillTypeInstant, data.SkillTypeChannel, data.SkillTypeTriggered, data.SkillTypeRetaliation, data.SkillTypeNonRepeatable,
	} {
		if activeSkill.SkillTypes[excluded] {
			repeatAllowed = false
		}
	}
	actor.Output["Repeats"] = 1
	if repeatAllowed {
		actor.Output["Repeats"] = 1 + skillModList.Sum(mod.TypeBase, skillCfg, "RepeatCount")
	}
	if actor.Output["Repeats"] > 1 {
		actor.Output["RepeatCount"] = actor.Output["Repeats"]
	}

	// Handle all the multipliers from Repeats.
	//pob:block:src/Modules/CalcOffence.lua:786-869
	repeatMode := resolvedConfigOption(env.Build, "repeatMode")
	if actor.Output["Repeats"] > 1 && repeatMode != "NONE" {
		repeats := actor.Output["Repeats"]
		reEmit := func(src mod.Mod, name string, modType mod.Type, value float64) mod.Mod {
			m := mod.NewFloat(name, modType, value).
				Source(src.GetSource()).
				Flag(src.Flags()).
				KeywordFlag(src.KeywordFlags())
			if tags := src.Tags(); len(tags) > 0 {
				m = m.Tag(tags...)
			}
			return m
		}
		for _, v := range skillModList.Tabulate(mod.TypeIncrease, skillCfg, "RepeatFinalAreaOfEffect") {
			modValue := v.Mod.Value().Float()
			if repeatMode == "AVERAGE" {
				modValue /= repeats
			}
			skillModList.AddMod(reEmit(v.Mod, "AreaOfEffect", mod.TypeIncrease, modValue))
		}
		for _, v := range skillModList.Tabulate(mod.TypeIncrease, skillCfg, "RepeatPerRepeatAreaOfEffect") {
			modValue := v.Mod.Value().Float() * (repeats - 1)
			if repeatMode == "AVERAGE" {
				modValue /= 2
			}
			skillModList.AddMod(reEmit(v.Mod, "AreaOfEffect", mod.TypeIncrease, modValue))
		}
		for _, v := range skillModList.Tabulate(mod.TypeBase, skillCfg, "RepeatFinalDoubleDamageChance") {
			modValue := v.Mod.Value().Float()
			if repeatMode == "AVERAGE" {
				modValue /= repeats
			}
			skillModList.AddMod(reEmit(v.Mod, "DoubleDamageChance", mod.TypeBase, modValue))
		}
		damageFinalMoreValueTotal := float64(1)
		damageMoreValueTotal := float64(0)
		for _, v := range skillModList.Tabulate(mod.TypeMore, skillCfg, "RepeatFinalDamage") {
			modValue := v.Mod.Value().Float()
			damageFinalMoreValueTotal *= 1 + modValue/100
			damageMoreValueTotal += modValue
			if repeatMode == "AVERAGE" && !skillModList.Flag(nil, "OnlyFinalRepeat") {
				modValue /= repeats
			}
			skillModList.AddMod(reEmit(v.Mod, "Damage", mod.TypeMore, modValue))
		}
		for _, v := range skillModList.Tabulate(mod.TypeMore, skillCfg, "RepeatPerRepeatDamage") {
			raw := v.Mod.Value().Float()
			modValue := raw * (repeats - 1)
			if repeatMode == "AVERAGE" {
				if damageFinalMoreValueTotal != 1 {
					modValue = ((100+raw*(repeats-2)/2)*(repeats-1)+
						(100+raw*(repeats-1))*damageFinalMoreValueTotal)/
						(repeats+damageMoreValueTotal/100) - 100
				} else {
					modValue /= 2
				}
			}
			skillModList.AddMod(reEmit(v.Mod, "Damage", mod.TypeMore, modValue))
		}

		var lastMod mod.Mod
		damageFinalMoreValueTotal = damageMoreValueTotal
		for _, rc := range []struct {
			count float64
			name  string
		}{{2, "One"}, {3, "Two"}, {4, "Three"}} {
			if rc.count > repeats {
				break
			} else if repeatMode == "AVERAGE" {
				for _, v := range skillModList.Tabulate(mod.TypeMore, skillCfg, "Repeat"+rc.name+"Damage") {
					damageMoreValueTotal += v.Mod.Value().Float()
					lastMod = v.Mod
				}
			} else if rc.count == repeats {
				for _, v := range skillModList.Tabulate(mod.TypeMore, skillCfg, "Repeat"+rc.name+"Damage") {
					skillModList.AddMod(reEmit(v.Mod, "Damage", mod.TypeMore, v.Mod.Value().Float()))
				}
			}
		}
		if repeatMode == "AVERAGE" && lastMod != nil {
			skillModList.AddMod(reEmit(lastMod, "Damage", mod.TypeMore,
				(damageMoreValueTotal/repeats+100)/(1+damageFinalMoreValueTotal/repeats/100)-100))
		}
		if skillModList.Flag(nil, "FinalRepeatSumsDamage") {
			for _, v := range skillModList.Tabulate(mod.TypeFlag, skillCfg, "FinalRepeatSumsDamage") {
				skillModList.AddMod(reEmit(v.Mod, "Damage", mod.TypeMore,
					(100*repeats+damageFinalMoreValueTotal)/(1+damageFinalMoreValueTotal/100)-100))
			}
		}
		if skillFlags[SkillFlagTrap] || skillFlags[SkillFlagMine] {
			skillModList.AddMod(mod.NewFloat("DPS", mod.TypeMore, (repeats-1)*100).Source("Repeat Count"))
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:871-878
	if skillModList.Flag(nil, "WeaponPhysAppliesToSpells") {
		mult := skillModList.Sum(mod.TypeBase, skillCfg, "WeaponPhysAppliesToSpellsPercent") / 100
		if actor.WeaponData1 != nil {
			skillModList.AddMod(mod.NewFloat("PhysicalMin", mod.TypeBase,
				math.Floor(utils.GetOr[float64](actor.WeaponData1, "PhysicalMin", 0)*mult)).
				Source("Runegraft of the Spellbound").Flag(mod.MFlagSpell))
			skillModList.AddMod(mod.NewFloat("PhysicalMax", mod.TypeBase,
				math.Floor(utils.GetOr[float64](actor.WeaponData1, "PhysicalMax", 0)*mult)).
				Source("Runegraft of the Spellbound").Flag(mod.MFlagSpell))
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:879-897
	if skillData.GainPercentBaseWandDamageToSpells != 0 {
		mult := skillData.GainPercentBaseWandDamageToSpells / 100
		addWandDamage := func(damageType data.DamageType, suffix string, value float64) {
			skillModList.AddMod(mod.NewFloat(string(damageType)+suffix, "BASE", math.Floor(value)).
				Source("Spellslinger").Flag(mod.MFlagSpell))
		}
		for _, damageType := range data.DamageType("").Values() {
			for _, suffix := range []string{"Min", "Max"} {
				w1 := utils.GetOr[float64](actor.WeaponData1, string(damageType)+suffix, 0)
				w2 := utils.GetOr[float64](actor.WeaponData2, string(damageType)+suffix, 0)
				switch {
				case actor.WeaponData1.Type == "Wand" && actor.WeaponData2.Type == "Wand":
					addWandDamage(damageType, suffix, (w1+w2)/2*mult)
				case actor.WeaponData1.Type == "Wand":
					addWandDamage(damageType, suffix, w1*mult)
				case actor.WeaponData2.Type == "Wand":
					addWandDamage(damageType, suffix, w2*mult)
				}
			}
		}
	}
	if skillModList.Flag(nil, "TriggeredBySnipe") && activeSkill.SkillTypes[data.SkillTypeTriggerable] {
		skillModList.AddMod(mod.NewFloat("Damage", "MORE", 165).Source("Config").Flag(mod.MFlagHit).Tag(mod.Multiplier("SnipeStage")))
		skillModList.AddMod(mod.NewFloat("Damage", "MORE", 120).Source("Config").Flag(mod.MFlagAilment).Tag(mod.Multiplier("SnipeStage")))
	}
	if skillModList.Sum(mod.TypeBase, nil, "CritMultiplierAppliesToDegen") > 0 {
		for _, value := range skillModList.Tabulate("BASE", skillCfg, "CritMultiplier") {
			if value.Mod.GetSource() != "Base" { // The global base Crit Multi doesn't apply to ailments with Perfect Agony
				skillModList.AddMod(mod.NewFloat("DotMultiplier", "BASE", math.Floor(value.Mod.Value().Float()/2)).Source(value.Mod.GetSource()).Flag(mod.MFlagAilment).Tag(mod.Condition("CriticalStrike")).Tag(value.Mod.Tags()...))
			}
		}
	}
	if skillModList.Flag(nil, "HasSeals") && activeSkill.SkillTypes[data.SkillTypeCanRapidFire] && !skillModList.Flag(nil, "NoRepeatBonuses") {
		// Applies DPS multiplier based on seals count
		actor.Output["SealCooldown"] = skillModList.Sum(mod.TypeBase, skillCfg, "SealGainFrequency") / calclib.Mod(skillModList, skillCfg, "SealGainFrequency")
		actor.Output["SealMax"] = skillModList.Sum(mod.TypeBase, skillCfg, "SealCount")
		//pob:block:src/Modules/CalcOffence.lua:914
		actor.Output["AverageBurstHits"] = actor.Output["SealMax"]
		actor.Output["TimeMaxSeals"] = actor.Output["SealCooldown"] * actor.Output["SealMax"]

		if skillData.HitTimeOverride == 0 {
			if skillModList.Flag(nil, "UseMaxUnleash") {
				for _, value := range skillModList.Tabulate("INC", nil, "MaxSealCrit") {
					skillModList.AddMod(mod.NewFloat("CritChance", "INC", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
				}
				//pob:block:src/Modules/CalcOffence.lua:923-926
				for _, value := range skillModList.Tabulate("MORE", nil, "MaxSealDamage") {
					skillModList.AddMod(mod.NewFloat("Damage", "MORE", value.Mod.Value().Float()*actor.Output["SealMax"]).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
				}
				env.Player.MainSkill.SkillData.DpsMultiplier = 1 + actor.Output["SealMax"]*calclib.Mod(skillModList, skillCfg, "SealRepeatPenalty")
				env.Player.MainSkill.SkillData.HitTimeOverride = max(actor.Output["TimeMaxSeals"], 1/activeSkill.ActiveEffect.GrantedEffect.CastTime()*1.1*calclib.Mod(skillModList, skillCfg, "Speed")*actor.Output["ActionSpeedMod"])
			} else {
				env.Player.MainSkill.SkillData.DpsMultiplier = 1 + 1/actor.Output["SealCooldown"]/(1/activeSkill.ActiveEffect.GrantedEffect.CastTime()*1.1*calclib.Mod(skillModList, skillCfg, "Speed")*actor.Output["ActionSpeedMod"])*calclib.Mod(skillModList, skillCfg, "SealRepeatPenalty")
			}
		}

		if breakdown != nil {
			breakdown.MultiChain("SealGainTime", BMultiChain{
				Label: "Gain frequency:",
				Base:  &BMultiChainItem{"%.2fs ^8(base gain frequency)", skillModList.Sum(mod.TypeBase, skillCfg, "SealGainFrequency")},
				Total: fmt.Sprintf("= %.2fs ^8per Seal", actor.Output["SealCooldown"]),
				Items: []BMultiChainItem{
					{"%.2f ^8(increased/reduced gain frequency)", 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "SealGainFrequency")/100},
					{"%.2f ^8(action speed modifier)", actor.Output["ActionSpeedMod"]},
				},
			})
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:945-1000
	physMode := utils.OrS(env.Build.GetStringOption("physMode"), "AVERAGE")
	processedRandomMods := make(map[mod.Mod]bool)
	for _, cfg := range []*moddb.ListCfg{skillCfg, activeSkill.Weapon1Cfg, activeSkill.Weapon2Cfg} {
		if cfg == nil {
			continue
		}
		if skillModList.Sum(mod.TypeBase, cfg, "PhysicalDamageGainAsRandom", "PhysicalDamageConvertToRandom", "PhysicalDamageGainAsColdOrLightning") <= 0 {
			continue
		}
		skillFlags[SkillFlagRandomPhys] = true
		derive := func(name string, m mod.Mod, value float64) {
			skillModList.AddMod(mod.NewFloat(name, "BASE", value).Source(m.GetSource()).Flag(m.Flags()).KeywordFlag(m.KeywordFlags()).Tag(m.Tags()...))
		}
		for _, value := range skillModList.Tabulate("BASE", cfg, "PhysicalDamageGainAsRandom") {
			if processedRandomMods[value.Mod] {
				continue
			}
			processedRandomMods[value.Mod] = true
			full := value.Mod.Value().Float()
			effVal := full / 3
			switch physMode {
			case "AVERAGE":
				derive("PhysicalDamageGainAsFire", value.Mod, effVal)
				derive("PhysicalDamageGainAsCold", value.Mod, effVal)
				derive("PhysicalDamageGainAsLightning", value.Mod, effVal)
			case "FIRE":
				derive("PhysicalDamageGainAsFire", value.Mod, full)
			case "COLD":
				derive("PhysicalDamageGainAsCold", value.Mod, full)
			case "LIGHTNING":
				derive("PhysicalDamageGainAsLightning", value.Mod, full)
			}
		}
		for _, value := range skillModList.Tabulate("BASE", cfg, "PhysicalDamageConvertToRandom") {
			if processedRandomMods[value.Mod] {
				continue
			}
			processedRandomMods[value.Mod] = true
			full := value.Mod.Value().Float()
			effVal := full / 3
			switch physMode {
			case "AVERAGE":
				derive("PhysicalDamageConvertToFire", value.Mod, effVal)
				derive("PhysicalDamageConvertToCold", value.Mod, effVal)
				derive("PhysicalDamageConvertToLightning", value.Mod, effVal)
			case "FIRE":
				derive("PhysicalDamageConvertToFire", value.Mod, full)
			case "COLD":
				derive("PhysicalDamageConvertToCold", value.Mod, full)
			case "LIGHTNING":
				derive("PhysicalDamageConvertToLightning", value.Mod, full)
			}
		}
		for _, value := range skillModList.Tabulate("BASE", cfg, "PhysicalDamageGainAsColdOrLightning") {
			if processedRandomMods[value.Mod] {
				continue
			}
			processedRandomMods[value.Mod] = true
			full := value.Mod.Value().Float()
			effVal := full / 2
			switch physMode {
			case "AVERAGE", "FIRE":
				derive("PhysicalDamageGainAsCold", value.Mod, effVal)
				derive("PhysicalDamageGainAsLightning", value.Mod, effVal)
			case "COLD":
				derive("PhysicalDamageGainAsCold", value.Mod, full)
			case "LIGHTNING":
				derive("PhysicalDamageGainAsLightning", value.Mod, full)
			}
		}
	}

	//pob:block:src/Modules/CalcOffence.lua:1003-1016
	if skillModList.Flag(nil, "SupportedByMomentum") {
		maxMomentumStacks := skillModList.Sum(mod.TypeBase, skillCfg, "MomentumStacksMax")
		extraMomentumStacks := skillModList.Sum(mod.TypeBase, skillCfg, "MomentumStacksExtra")
		switch {
		case maxMomentumStacks > 0:
			if !modDB.HasMod(mod.TypeBase, nil, "Multiplier:MomentumStacks") {
				modDB.AddMod(mod.NewFloat("Multiplier:MomentumStacks", mod.TypeBase,
					min((maxMomentumStacks+extraMomentumStacks)/2, maxMomentumStacks)).
					Source("Config").Tag(mod.Condition("Combat")))
			} else if modDB.Sum(mod.TypeBase, nil, "Multiplier:MomentumStacks") > maxMomentumStacks {
				modDB.ReplaceMod(mod.NewFloat("Multiplier:MomentumStacks", mod.TypeBase, maxMomentumStacks).
					Source("Config").Tag(mod.Condition("Combat")))
			}
		case modDB.HasMod(mod.TypeBase, nil, "Multiplier:MomentumStacks"):
			modDB.ReplaceMod(mod.NewFloat("Multiplier:MomentumStacks", mod.TypeBase, 0).Source("Config"))
		}
	}

	runSkillFunc("preSkillTypeFunc")

	isAttack := skillFlags[SkillFlagAttack]

	// Calculate skill type stats
	if activeSkill.Minion != nil {
		if activeSkill.Minion.MinionData.Limit != "" {
			limitName := activeSkill.Minion.MinionData.Limit
			if override := env.ModDB.Override(nil, limitName); override != nil {
				actor.Output["ActiveMinionLimit"] = math.Floor(override.Float())
			} else {
				actor.Output["ActiveMinionLimit"] = math.Floor(calclib.Val(skillModList, limitName, skillCfg) *
					skillModList.More(skillCfg, "ActiveMinionLimit"))
			}
		}
		//pob:block:src/Modules/CalcOffence.lua:1027-1030
		actor.Output["SummonedMinionsPerCast"] = math.Floor(calclib.Val(skillModList, "MinionPerCastCount", skillCfg))
		if actor.Output["SummonedMinionsPerCast"] == 0 {
			actor.Output["SummonedMinionsPerCast"] = 1
		}
	}
	if skillFlags[SkillFlagChaining] {
		if skillModList.Flag(skillCfg, "CannotChain") || skillModList.Flag(skillCfg, "NoAdditionalChains") {
			actor.OutputStrings["ChainMaxString"] = "Cannot chain"
		} else {
			actor.Output["ChainMax"] = skillModList.Sum(mod.TypeBase, skillCfg, "ChainCountMax", utils.Ternary(!skillFlags[SkillFlagProjectile], "BeamChainCountMax", ""))
			if skillModList.Flag(skillCfg, "AdditionalProjectilesAddChainsInstead") {
				projCount := float64(0)
				if !skillModList.Flag(skillCfg, "SingleProjectile") {
					projCount = math.Floor((skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount") - 1) *
						skillModList.More(skillCfg, "ProjectileCount"))
				}
				actor.Output["ChainMax"] += projCount
			}
			actor.Output["ChainMax"] *= skillModList.More(skillCfg, "ChainCountMax")
			actor.Output["ChainMaxString"] = actor.Output["ChainMax"]
			actor.OutputStrings["ChainMaxString"] = fmt.Sprint(actor.Output["ChainMax"])
			actor.Output["Chain"] = min(actor.Output["ChainMax"], skillModList.Sum(mod.TypeBase, skillCfg, "ChainCount"))
			actor.Output["ChainRemaining"] = max(0, actor.Output["ChainMax"]-actor.Output["Chain"])
		}
	}
	if skillFlags[SkillFlagProjectile] {
		if skillModList.Flag(nil, "PointBlank") {
			skillModList.AddMod(mod.NewFloat("Damage", "MORE", 30).Source("Point Blank").Flag(mod.MFlagAttack | mod.MFlagProjectile).Tag(mod.DistanceRamp(pointBlankDistanceRamp)))
		}
		if skillModList.Flag(nil, "FarShot") {
			skillModList.AddMod(mod.NewFloat("Damage", "MORE", 100).Source("Far Shot").Flag(mod.MFlagAttack | mod.MFlagProjectile).Tag(mod.DistanceRamp(farShotDistanceRamp)))
		}
		if skillModList.Flag(skillCfg, "NoAdditionalProjectiles") || skillModList.Flag(skillCfg, "SingleProjectile") {
			actor.Output["ProjectileCount"] = 1
		} else {
			projMin := skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCountMinimum")
			projMax := math.Inf(1)
			if override := skillModList.Override(skillCfg, "ProjectileCountMaximum"); override != nil {
				projMax = override.Float()
			}
			projBase := skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount")
			projMore := skillModList.More(skillCfg, "ProjectileCount")
			proj := math.Floor(projBase * projMore)
			actor.Output["ProjectileCount"] = max(min(proj, projMax), projMin)
		}
		if skillModList.Flag(skillCfg, "AdditionalProjectilesAddBouncesInstead") {
			projBase := float64(0)
			if !skillModList.Flag(skillCfg, "SingleProjectile") {
				projBase = skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount") + skillModList.Sum(mod.TypeBase, skillCfg, "BounceCount") - 1
			}
			projMore := skillModList.More(skillCfg, "ProjectileCount")
			actor.Output["BounceCount"] = math.Floor(projBase * projMore)
		}
		if skillModList.Flag(skillCfg, "CannotSplit") || activeSkill.SkillTypes[data.SkillTypeProjectileNumber] {
			if breakdown != nil {
				splitCount := skillModList.Sum(mod.TypeBase, skillCfg, "SplitCount") +
					enemyDB.Sum(mod.TypeBase, skillCfg, "SelfSplitCount")
				if splitCount > 0 {
					actor.OutputStrings["SplitCountString"] = "Cannot split"
				}
			}
		} else {
			splitCount := skillModList.Sum(mod.TypeBase, skillCfg, "SplitCount") +
				enemyDB.Sum(mod.TypeBase, skillCfg, "SelfSplitCount")
			if skillModList.Flag(skillCfg, "AdditionalProjectilesAddSplitsInstead") {
				addedSplits := float64(0)
				if !skillModList.Flag(skillCfg, "SingleProjectile") {
					addedSplits = math.Floor((skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount") - 1) *
						skillModList.More(skillCfg, "ProjectileCount"))
				}
				splitCount += addedSplits
			}
			if skillModList.Flag(skillCfg, "AdditionalChainsAddSplitsInstead") {
				splitCount += skillModList.Sum(mod.TypeBase, skillCfg, "ChainCountMax")
			}
			actor.Output["SplitCount"] = splitCount
			actor.Output["SplitCountString"] = splitCount
			actor.OutputStrings["SplitCountString"] = fmt.Sprint(splitCount)
		}
		if skillModList.Flag(skillCfg, "CannotFork") {
			actor.OutputStrings["ForkCountString"] = "Cannot fork"
		} else if skillModList.Flag(skillCfg, "ForkOnce") {
			skillFlags[SkillFlagForking] = true
			if skillModList.Flag(skillCfg, "ForkTwice") {
				actor.Output["ForkCountMax"] = min(skillModList.Sum(mod.TypeBase, skillCfg, "ForkCountMax"), 2)
			} else {
				actor.Output["ForkCountMax"] = min(skillModList.Sum(mod.TypeBase, skillCfg, "ForkCountMax"), 1)
			}
			actor.Output["ForkedCount"] = min(actor.Output["ForkCountMax"], skillModList.Sum(mod.TypeBase, skillCfg, "ForkedCount"))
			actor.Output["ForkCountString"] = actor.Output["ForkCountMax"]
			actor.OutputStrings["ForkCountString"] = fmt.Sprint(actor.Output["ForkCountMax"])
			actor.Output["ForkRemaining"] = max(0, actor.Output["ForkCountMax"]-actor.Output["ForkedCount"])
		} else {
			actor.OutputStrings["ForkCountString"] = "0"
		}
		if skillModList.Flag(skillCfg, "CannotPierce") {
			actor.Output["PierceCount"] = 0
			actor.OutputStrings["PierceCountString"] = "Cannot pierce"
		} else {
			if skillModList.Flag(skillCfg, "PierceAllTargets") || enemyDB.Flag(nil, "AlwaysPierceSelf") {
				actor.Output["PierceCount"] = 100
				actor.OutputStrings["PierceCountString"] = "All targets"
			} else {
				actor.Output["PierceCount"] = skillModList.Sum(mod.TypeBase, skillCfg, "PierceCount")
				actor.Output["PierceCountString"] = actor.Output["PierceCount"]
				actor.OutputStrings["PierceCountString"] = fmt.Sprint(actor.Output["PierceCount"])
			}
			if actor.Output["PierceCount"] > 0 {
				skillFlags[SkillFlagPiercing] = true
			}
			actor.Output["PiercedCount"] = min(actor.Output["PierceCount"], skillModList.Sum(mod.TypeBase, skillCfg, "PiercedCount"))
		}
		actor.Output["ProjectileSpeedMod"] = calclib.Mod(skillModList, skillCfg, "ProjectileSpeed")
		if breakdown != nil {
			breakdown.Mod("ProjectileSpeedMod", skillModList, skillCfg, "ProjectileSpeed")
		}
	}
	if skillFlags[SkillFlagMelee] {
		if skillFlags[SkillFlagWeapon1Attack] {
			if actor.WeaponData1.Range != 0 {
				actor.WeaponRange1 = actor.WeaponData1.Range +
					skillModList.Sum(mod.TypeBase, activeSkill.Weapon1Cfg, "MeleeWeaponRange") +
					10*skillModList.Sum(mod.TypeBase, activeSkill.Weapon1Cfg, "MeleeWeaponRangeMetre")
			} else {
				actor.WeaponRange1 = 6 + skillModList.Sum(mod.TypeBase, skillCfg, "UnarmedRange") +
					10*skillModList.Sum(mod.TypeBase, skillCfg, "UnarmedRangeMetre")
			}
		}
		if skillFlags[SkillFlagWeapon2Attack] {
			if actor.WeaponData2.Range != 0 {
				actor.WeaponRange2 = actor.WeaponData2.Range +
					skillModList.Sum(mod.TypeBase, activeSkill.Weapon2Cfg, "MeleeWeaponRange") +
					10*skillModList.Sum(mod.TypeBase, activeSkill.Weapon2Cfg, "MeleeWeaponRangeMetre")
			} else {
				actor.WeaponRange2 = 6 + skillModList.Sum(mod.TypeBase, skillCfg, "UnarmedRange") +
					10*skillModList.Sum(mod.TypeBase, skillCfg, "UnarmedRangeMetre")
			}
		}
		if activeSkill.SkillTypes[data.SkillTypeMeleeSingleTarget] {
			rangeVal := float64(100)
			if skillFlags[SkillFlagWeapon1Attack] {
				rangeVal = min(rangeVal, actor.WeaponRange1)
			}
			if skillFlags[SkillFlagWeapon2Attack] {
				rangeVal = min(rangeVal, actor.WeaponRange2)
			}
			actor.Output["WeaponRange"] = rangeVal + 2
			//pob:block:src/Modules/CalcOffence.lua:1138-1147
			actor.Output["WeaponRangeMetre"] = actor.Output["WeaponRange"] / 10
			const baseStrikeCount = 1
			actor.Output["StrikeTargets"] = baseStrikeCount + skillModList.Sum(mod.TypeBase, skillCfg, "AdditionalStrikeTarget")
		}
	}
	if utils.HasTrue(skillFlags, SkillFlagArea) || skillData.Radius != 0 || (utils.HasTrue(skillFlags, SkillFlagMine) && activeSkill.SkillTypes[data.SkillTypeAura]) {
		calcAreaOfEffect(skillModList, skillCfg, skillData, skillFlags, actor.Output, actor.Breakdown)
	}
	if activeSkill.SkillTypes[data.SkillTypeAura] {
		auraEffectOnSelf := utils.Ternary(
			!(skillData.AuraCannotAffectSelf || activeSkill.SkillTypes[data.SkillTypeAuraAffectsEnemies]),
			"SkillAuraEffectOnSelf", "")
		actor.Output["AuraEffectMod"] = calclib.Mod(skillModList, skillCfg, "AuraEffect", auraEffectOnSelf)
		if breakdown != nil {
			breakdown.Mod("AuraEffectMod", skillModList, skillCfg, "AuraEffect", auraEffectOnSelf)
		}
	}
	//pob:src/Modules/CalcOffence.lua:859-878
	if activeSkill.SkillTypes[data.SkillTypeHasReservation] && !activeSkill.SkillTypes[data.SkillTypeReservationBecomesCost] {
		for _, pool := range []string{"Life", "Mana"} {
			actor.Output[pool+"ReservedMod"] = 0
			if calclib.Mod(skillModList, skillCfg, "SupportManaMultiplier") > 0 && calclib.Mod(skillModList, skillCfg, pool+"Reserved", "Reserved") > 0 {
				actor.Output[pool+"ReservedMod"] = calclib.Mod(skillModList, skillCfg, pool+"Reserved", "Reserved") * calclib.Mod(skillModList, skillCfg, "SupportManaMultiplier") / max(0, calclib.Mod(skillModList, skillCfg, pool+"ReservationEfficiency", "ReservationEfficiency"))
			}
		}
	}
	if activeSkill.SkillTypes[data.SkillTypeHex] || activeSkill.SkillTypes[data.SkillTypeMark] {
		actor.Output["CurseEffectMod"] = calclib.Mod(skillModList, skillCfg, "CurseEffect")
		if breakdown != nil {
			breakdown.Mod("CurseEffectMod", skillModList, skillCfg, "CurseEffect")
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:1185-1191
	if activeSkill.SkillTypes[data.SkillTypeWarcry] {
		fullDuration := calcSkillDuration(skillModList, skillCfg, activeSkill.SkillData, env, enemyDB)
		actualCooldown := utils.Or(
			skillModList.Override(skillCfg, "CooldownRecovery"),
			(activeSkill.SkillData.Cooldown+skillModList.Sum(mod.TypeBase, skillCfg, "CooldownRecovery"))/
				calclib.Mod(skillModList, skillCfg, "CooldownRecovery"),
		)
		uptime := 1.0
		if !env.ModDB.Flag(nil, "Condition:WarcryMaxHit") {
			uptime = min(fullDuration/actualCooldown, 1)
		}
		unscaledEffect := calclib.Mod(skillModList, skillCfg, "WarcryEffect", "BuffEffect")
		actor.Output["WarcryEffectMod"] = unscaledEffect * uptime
		if breakdown != nil {
			breakdown.AddLine("WarcryEffectMod", fmt.Sprintf("%.2f ^8(effect modifiers)", unscaledEffect))
		}
	}
	if activeSkill.SkillTypes[data.SkillTypeLink] {
		actor.Output["LinkEffectMod"] = calclib.Mod(skillModList, skillCfg, "LinkEffect", "BuffEffect")
		if breakdown != nil {
			breakdown.Mod("LinkEffectMod", skillModList, skillCfg, "LinkEffect", "BuffEffect")
		}
	}
	if activeSkill.SkillTypes[data.SkillTypeBuff] && activeSkill.SkillTypes[data.SkillTypeHerald] {
		actor.Output["HeraldBuffEffectMod"] = calclib.Mod(skillModList, skillCfg, "BuffEffect", "BuffEffectOnSelf")
		if breakdown != nil {
			breakdown.Mod("HeraldBuffEffectMod", skillModList, skillCfg, "BuffEffect", "BuffEffectOnSelf")
		}
	}
	if (skillFlags[SkillFlagTrap] || skillFlags[SkillFlagMine]) && !(skillData.TrapCooldown != 0 || skillData.Cooldown != 0) {
		skillFlags[SkillFlagNotAverage] = true
		skillFlags[SkillFlagShowAverage] = false
		skillData.ShowAverage = false
	}
	if skillFlags[SkillFlagTrap] {
		baseSpeed := 1 / skillModList.Sum(mod.TypeBase, skillCfg, "TrapThrowingTime")
		timeMod := calclib.Mod(skillModList, skillCfg, "SkillTrapThrowingTime")
		if timeMod > 0 {
			baseSpeed = baseSpeed * (1 / timeMod)
		}
		actor.Output["TrapThrowingSpeed"] = baseSpeed * calclib.Mod(skillModList, skillCfg, "TrapThrowingSpeed") * actor.Output["ActionSpeedMod"]
		//pob:block:src/Modules/CalcOffence.lua:1230-1237
		trapThrowCount := calclib.Val(skillModList, "TrapThrowCount", skillCfg)
		if skillData.TrapCooldown != 0 || skillData.Cooldown != 0 {
			trapThrowCount = 1
		}
		if override := env.ModDB.Override(nil, "TrapThrowCount"); override != nil {
			trapThrowCount = override.Float()
		}
		actor.Output["TrapThrowCount"] = trapThrowCount
		actor.Output["TrapThrowingSpeed"] = min(actor.Output["TrapThrowingSpeed"], data.ServerTickRate)
		actor.Output["TrapThrowingTime"] = 1 / actor.Output["TrapThrowingSpeed"]
		skillData.TimeOverride = actor.Output["TrapThrowingTime"] / actor.Output["TrapThrowCount"]

		if baseCooldown := utils.OrF(skillData.TrapCooldown, skillData.Cooldown); baseCooldown != 0 {
			actor.Output["TrapCooldown"] = baseCooldown / calclib.Mod(skillModList, skillCfg, "CooldownRecovery")
			actor.Output["TrapCooldown"] = math.Ceil(actor.Output["TrapCooldown"]*data.ServerTickRate) / data.ServerTickRate
		}

		incArea, moreArea := calclib.Mods(skillModList, skillCfg, "TrapTriggerAreaOfEffect")
		areaMod := utils.RoundTo(utils.RoundTo(incArea*moreArea, 10), 2)
		actor.Output["TrapTriggerRadius"] = calcRadius(data.TrapTriggerRadiusBase, areaMod)
		actor.Output["TrapTriggerRadiusMetre"] = actor.Output["TrapTriggerRadius"] / 10
		if actor.Breakdown != nil {
			actor.Breakdown.area("TrapTriggerRadius", data.TrapTriggerRadiusBase, areaMod, actor.Output["TrapTriggerRadius"], calcRadiusBreakpoints(data.TrapTriggerRadiusBase, incArea, moreArea), "")
		}
	} else if skillData.Cooldown != 0 || skillModList.Sum(mod.TypeBase, skillCfg, "CooldownRecovery") != 0 {
		actor.Output["Cooldown"] = calcSkillCooldown(skillModList, skillCfg, skillData)
	}
	//pob:block:src/Modules/CalcOffence.lua:952-963
	if skillData.StoredUses != nil {
		baseUses := *skillData.StoredUses
		additionalUses := skillModList.Sum(mod.TypeBase, skillCfg, "AdditionalCooldownUses", "AdditionalUses")
		actor.Output["StoredUses"] = baseUses + additionalUses
	}
	if skillFlags[SkillFlagMine] {
		baseSpeed := 1 / skillModList.Sum(mod.TypeBase, skillCfg, "MineLayingTime")
		timeMod := calclib.Mod(skillModList, skillCfg, "SkillMineThrowingTime")
		if timeMod > 0 {
			baseSpeed = baseSpeed * (1 / timeMod)
		}
		actor.Output["MineLayingSpeed"] = baseSpeed * calclib.Mod(skillModList, skillCfg, "MineLayingSpeed") * actor.Output["ActionSpeedMod"]
		//pob:block:src/Modules/CalcOffence.lua:1319-1336
		mineThrowCount := calclib.Val(skillModList, "MineThrowCount", skillCfg)
		if skillData.TrapCooldown != 0 || skillData.Cooldown != 0 {
			mineThrowCount = 1
		}
		if override := env.ModDB.Override(nil, "MineThrowCount"); override != nil {
			mineThrowCount = override.Float()
		}
		actor.Output["MineThrowCount"] = mineThrowCount
		if actor.Output["MineThrowCount"] >= 1 {
			// Throwing mines takes 10% more time for each *additional* mine thrown.
			actor.Output["MineLayingSpeed"] = actor.Output["MineLayingSpeed"] / (1 + (actor.Output["MineThrowCount"]-1)*0.1)
		}
		actor.Output["MineLayingSpeed"] = min(actor.Output["MineLayingSpeed"], data.ServerTickRate)
		actor.Output["MineLayingTime"] = 1 / actor.Output["MineLayingSpeed"]
		if skillFlags[SkillFlagTrap] {
			skillData.TimeOverride = actor.Output["MineLayingTime"] / actor.Output["MineThrowCount"] / actor.Output["TrapThrowCount"]
		} else {
			skillData.TimeOverride = actor.Output["MineLayingTime"] / actor.Output["MineThrowCount"]
		}

		incArea, moreArea := calclib.Mods(skillModList, skillCfg, "MineDetonationAreaOfEffect")
		areaMod := utils.RoundTo(utils.RoundTo(incArea*moreArea, 10), 2)
		actor.Output["MineDetonationRadius"] = calcRadius(data.MineDetonationRadiusBase, areaMod)
		actor.Output["MineDetonationRadiusMetre"] = actor.Output["MineDetonationRadius"] / 10
		if actor.Breakdown != nil {
			actor.Breakdown.area("MineDetonationRadius", data.MineDetonationRadiusBase, areaMod, actor.Output["MineDetonationRadius"], calcRadiusBreakpoints(data.MineDetonationRadiusBase, incArea, moreArea), "")
		}

		if activeSkill.SkillTypes[data.SkillTypeAura] {
			actor.Output["MineAuraRadius"] = calcRadius(data.MineAuraRadiusBase, actor.Output["AreaOfEffectMod"])
			actor.Output["MineAuraRadiusMetre"] = actor.Output["MineAuraRadius"] / 10
			if actor.Breakdown != nil {
				incArea, moreArea := calclib.Mods(skillModList, skillCfg, "AreaOfEffect")
				actor.Breakdown.area("MineAuraRadius", data.MineAuraRadiusBase, actor.Output["AreaOfEffectMod"], actor.Output["MineAuraRadius"], calcRadiusBreakpoints(data.MineAuraRadiusBase, incArea, moreArea), "")
			}
		}
	}
	if skillFlags[SkillFlagTotem] {
		baseSpeed := 1 / skillModList.Sum(mod.TypeBase, skillCfg, "TotemPlacementTime")
		if skillFlags[SkillFlagBallista] {
			baseSpeed = 1 / skillModList.Sum(mod.TypeBase, skillCfg, "BallistaPlacementTime")
		}
		actor.Output["TotemPlacementSpeed"] = baseSpeed * calclib.Mod(skillModList, skillCfg, "TotemPlacementSpeed") * actor.Output["ActionSpeedMod"]
		actor.Output["TotemPlacementTime"] = 1 / actor.Output["TotemPlacementSpeed"]
		if breakdown != nil {
			breakdown.MultiChain("TotemPlacementTime", BMultiChain{
				Label: "Placement speed:",
				Base:  &BMultiChainItem{"%.2f ^8(base placement speed)", baseSpeed},
				Total: fmt.Sprintf("= %.2f ^8per second", actor.Output["TotemPlacementSpeed"]),
				Items: []BMultiChainItem{
					{"%.2f ^8(increased/reduced placement speed)", 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "TotemPlacementSpeed")/100},
					{"%.2f ^8(more/less placement speed)", skillModList.More(skillCfg, "TotemPlacementSpeed")},
					{"%.2f ^8(action speed modifier)", actor.Output["ActionSpeedMod"]},
				},
			})
		}
		actor.Output["ActiveTotemLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
		actor.Output["TotemsSummoned"] = utils.Or(env.ModDB.Override(nil, "TotemsSummoned"), actor.Output["ActiveTotemLimit"])
		if breakdown != nil {
			breakdown.AddLine("ActiveTotemLimit",
				"Totems Summoned: "+fmt.Sprint(actor.Output["TotemsSummoned"])+(utils.Ternary(env.Build.GetNumberOption("TotemsSummoned") != 0, " ^8(overridden from the Configuration tab)", " ^8(can be overridden in the Configuration tab)")),
			)
		}
		actor.Output["TotemLifeMod"] = calclib.Mod(skillModList, skillCfg, "TotemLife")
		totemMult, ok := totemLifeMult(activeSkill.SkillTotemId)
		if !ok {
			totemMult = 1
		}
		actor.Output["TotemLife"] = utils.RoundTo(math.Floor(data.MonsterAllyLifeTable[skillData.TotemLevel]*totemMult)*actor.Output["TotemLifeMod"], 0)
		//pob:block:src/Modules/CalcOffence.lua:1407-1409
		actor.Output["TotemEnergyShield"] = skillModList.Sum(mod.TypeBase, skillCfg, "TotemEnergyShield")
		actor.Output["TotemBlockChance"] = skillModList.Sum(mod.TypeBase, skillCfg, "TotemBlockChance")
		actor.Output["TotemArmour"] = skillModList.Sum(mod.TypeBase, skillCfg, "TotemArmour")
		if breakdown != nil {
			breakdown.Mod("TotemLifeMod", skillModList, skillCfg, "TotemLife")
			/*
				TODO Breakdown
				breakdown.AddLine("TotemLife",
					"Totem level: "+skillData.TotemLevel,
					data.MonsterAllyLifeTable[skillData.TotemLevel]+" ^8(base life for a level "+skillData.TotemLevel+" monster)",
					"x "+env.data.totemLifeMult[activeSkill.skillTotemId]+" ^8(life multiplier for this totem type)",
					"x "+Output["TotemLifeMod"]+" ^8(totem life modifier)",
					"= "+Output["TotemLife"],
				)
			*/
		}
	}
	if activeSkill.SkillTypes[data.SkillTypeBrand] {
		actor.Output["BrandAttachmentRange"] = data.BrandAttachmentRangeBase * calclib.Mod(skillModList, skillCfg, "BrandAttachmentRange")
		actor.Output["BrandAttachmentRangeMetre"] = actor.Output["BrandAttachmentRange"] / 10
		actor.Output["ActiveBrandLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveBrandLimit")
		actor.Output["AttachedBrandCount"] = skillData.AttachedBrandCount
	}
	//pob:src/Modules/CalcOffence.lua:1064-1066
	if skillFlags[SkillFlagWarcry] {
		actor.Output["WarcryCastTime"] = calcWarcryCastTime(skillModList, skillCfg, skillData, actor)
	}
	/*
		TODO skillFlags.corpse
		if skillFlags.corpse {
			actor.Output["CorpseLevel"] = skillModList.Sum(mod.TypeBase, skillCfg, "CorpseLevel")
			actor.Output["BaseCorpseLife"] = env.data.monsterLifeTable[actor.Output["CorpseLevel"] or 1] * (env.data.monsterVarietyLifeMult[skillData.corpseMonsterVariety] or 1) * (env.data.mapLevelLifeMult[env.enemyLevel] or 1)
			actor.Output["CorpseLifeInc"] = 1 + (skillModList.Sum(mod.TypeIncrease, skillCfg, "CorpseLife") or 0) / 100
			actor.Output["CorpseLife"] = actor.Output["BaseCorpseLife"] * actor.Output["CorpseLifeInc"]
			if breakdown != nil {
				breakdown.CorpseLife = {
					fmt.Sprintf("%d ^8(base life of a level %d monster)", env.data.monsterLifeTable[actor.Output["CorpseLevel"] or 1], actor.Output["CorpseLevel"] or "n/a"),
					fmt.Sprintf("x %.2f ^8(%s variety multiplier)", env.data.monsterVarietyLifeMult[skillData.corpseMonsterVariety] or 1, skillData.corpseMonsterVariety),
					fmt.Sprintf("x %.2f ^8(map level %d monster life multiplier from config)", env.data.mapLevelLifeMult[env.enemyLevel] or 1, env.enemyLevel),
					fmt.Sprintf(" = %d ^8(base corpse life)", actor.Output["BaseCorpseLife"]),
					fmt.Sprintf(""),
					fmt.Sprintf("x %.2f ^8(corpse maximum life increases)", actor.Output["CorpseLifeInc"]),
					fmt.Sprintf(" = %d", actor.Output["CorpseLife"]),
				}
			}
		}
	*/
	//pob:block:src/Modules/CalcOffence.lua:1051-1103
	generalsCryCooldown := float64(0)
	if skillData.TriggeredByGeneralsCry {
		var mirageActiveSkill *ActiveSkill

		// Find the active General's Cry gem to get active properties
		for _, skill := range actor.ActiveSkillList {
			name := ""
			if as := skill.ActiveEffect.GrantedEffect.Raw.GetActiveSkill(); as != nil {
				name = as.DisplayedName
			}
			if name == "General's Cry" && activeSkillSlot(actor.MainSkill) == activeSkillSlot(activeSkill) {
				mirageActiveSkill = skill
				break
			}
		}

		if mirageActiveSkill != nil {
			cooldown := calcSkillCooldown(mirageActiveSkill.SkillModList, mirageActiveSkill.SkillCfg, mirageActiveSkill.SkillData)

			//pob:src/Modules/CalcMirages.lua:382-386
			if !activeSkill.SkillTypes[data.SkillTypeChannel] {
				skillData.HitTimeOverride = 1
			}

			// Supported Attacks Count as Exerted.
			//pob:src/Modules/CalcMirages.lua:399-419
			for _, value := range skillModList.Tabulate(mod.TypeIncrease, skillCfg, "ExertIncrease") {
				skillModList.AddMod(copyModAs(value.Mod, "Damage"))
			}
			for _, value := range skillModList.Tabulate(mod.TypeMore, skillCfg, "ExertIncrease") {
				skillModList.AddMod(copyModAs(value.Mod, "Damage"))
			}
			for _, value := range skillModList.Tabulate(mod.TypeMore, skillCfg, "ExertAttackIncrease") {
				skillModList.AddMod(copyModAs(value.Mod, "Damage"))
			}
			for _, value := range skillModList.Tabulate(mod.TypeMore, skillCfg, "OverexertionExertAverageIncrease") {
				skillModList.AddMod(copyModAs(value.Mod, "Damage"))
			}
			for _, value := range skillModList.Tabulate(mod.TypeBase, skillCfg, "ExertDoubleDamageChance") {
				skillModList.AddMod(copyModAs(value.Mod, "DoubleDamageChance"))
			}
			maxMirageWarriors := 0.0
			for _, value := range mirageActiveSkill.SkillModList.Tabulate(mod.TypeBase, skillCfg, "GeneralsCryDoubleMaxCount") {
				skillModList.AddMod(copyModAs(value.Mod, "QuantityMultiplier"))
				maxMirageWarriors += value.Mod.Value().Float()
			}
			//pob:block:src/Modules/CalcMirages.lua:380-389,src/Modules/CalcMirages.lua:421-422
			mirageSpawnTime := 0.3 + 0.2*maxMirageWarriors
			if activeSkill.SkillTypes[data.SkillTypeChannel] {
				mirageSpawnTime++
			}
			mirageSpawnTime = luaRound(mirageSpawnTime, 2)
			activeSkill.infoMessage = luaNumber(maxMirageWarriors) + " GC Mirages using " + activeSkill.ActiveEffect.GrantedEffect.Name()
			activeSkill.infoMessage2 = luaNumber(mirageSpawnTime) + "s for " + luaNumber(maxMirageWarriors) + " Mirages to finish Attacking"

			generalsCryCooldown = cooldown
		}
	}
	debuffDurationMult := float64(1)
	// Skill duration
	if env.ModeEffective {
		debuffDurationMult = 1 / max(data.BuffExpirationSlowCap, calclib.Mod(enemyDB, skillCfg, "BuffExpireFaster"))
	}
	{
		actor.Output["DurationMod"] = calclib.Mod(skillModList, skillCfg, "Duration", "PrimaryDuration", utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
		//pob:src/Modules/CalcOffence.lua:1463
		actor.Output["DurationMod"] = max(actor.Output["DurationMod"], 0)
		if breakdown != nil {
			breakdown.Mod("DurationMod", skillModList, skillCfg, "Duration", "PrimaryDuration", utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
			if breakdown.Has("DurationMod") && skillData.DurationSecondary != 0 {
				breakdown.AddLine("DurationMod, 1", "Primary duration:")
			}
		}
		durationBase := (skillData.Duration) + skillModList.Sum(mod.TypeBase, skillCfg, "Duration", "PrimaryDuration")
		if durationBase > 0 {
			actor.Output["Duration"] = durationBase * actor.Output["DurationMod"]
			if skillData.Debuff {
				actor.Output["Duration"] = actor.Output["Duration"] * debuffDurationMult
			}
			actor.Output["Duration"] = math.Ceil(actor.Output["Duration"]*data.ServerTickRate) / data.ServerTickRate
			if breakdown != nil && actor.Output["Duration"] != durationBase {
				breakdown.AddLine("Duration",
					fmt.Sprintf("%.2fs ^8(base)", durationBase),
				)
				if actor.Output["DurationMod"] != 1 {
					breakdown.AddLine("Duration", fmt.Sprintf("x %.4f ^8(duration modifier)", actor.Output["DurationMod"]))
				}
				if skillData.Debuff && debuffDurationMult != 1 {
					breakdown.AddLine("Duration", fmt.Sprintf("/ %.3f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
				}
				breakdown.AddLine("Duration", "rounded up to nearest server tick")
				breakdown.AddLine("Duration", fmt.Sprintf("= %.3fs", actor.Output["Duration"]))
			}
		}
		durationBase = (skillData.DurationSecondary) + skillModList.Sum(mod.TypeBase, skillCfg, "Duration", "SecondaryDuration")
		if durationBase > 0 {
			durationMod := durationModClamped(skillModList, skillCfg, "Duration", "SecondaryDuration", utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
			actor.Output["DurationSecondary"] = durationBase * durationMod
			if skillData.DebuffSecondary {
				actor.Output["DurationSecondary"] = actor.Output["DurationSecondary"] * debuffDurationMult
			}
			actor.Output["DurationSecondary"] = math.Ceil(actor.Output["DurationSecondary"]*data.ServerTickRate) / data.ServerTickRate
			if breakdown != nil && actor.Output["DurationSecondary"] != durationBase {
				breakdown.Mod("SecondaryDurationMod", skillModList, skillCfg, "Duration", "SecondaryDuration", utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
				if breakdown.Has("SecondaryDurationMod") {
					breakdown.AddLine("SecondaryDurationMod, 1", "Secondary duration:")
				}
				breakdown.AddLine("DurationSecondary",
					fmt.Sprintf("%.2fs ^8(base)", durationBase),
				)
				if actor.Output["DurationMod"] != 1 {
					breakdown.AddLine("DurationSecondary", fmt.Sprintf("x %.4f ^8(duration modifier)", durationMod))
				}
				if skillData.DebuffSecondary && debuffDurationMult != 1 {
					breakdown.AddLine("DurationSecondary", fmt.Sprintf("/ %.3f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
				}
				breakdown.AddLine("DurationSecondary", "rounded up to nearest server tick")
				breakdown.AddLine("DurationSecondary", fmt.Sprintf("= %.3fs", actor.Output["DurationSecondary"]))
			}
		}
		//pob:block:src/Modules/CalcOffence.lua:1518-1526
		durationBase = skillData.DurationTertiary + skillModList.Sum(mod.TypeBase, skillCfg, "Duration", "TertiaryDuration")
		if durationBase > 0 {
			durationMod := durationModClamped(skillModList, skillCfg, "Duration", "TertiaryDuration", utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
			actor.Output["DurationTertiary"] = durationBase * durationMod
			if skillData.DebuffTertiary {
				actor.Output["DurationTertiary"] = actor.Output["DurationTertiary"] * debuffDurationMult
			}
			actor.Output["DurationTertiary"] = math.Ceil(actor.Output["DurationTertiary"]*data.ServerTickRate) / data.ServerTickRate
		}
		durationBase = skillData.AuraDuration
		if durationBase > 0 {
			durationMod := durationModClamped(skillModList, skillCfg, "Duration")
			actor.Output["AuraDuration"] = durationBase * durationMod
			actor.Output["AuraDuration"] = math.Ceil(actor.Output["AuraDuration"]*data.ServerTickRate) / data.ServerTickRate
			if breakdown != nil && actor.Output["AuraDuration"] != durationBase {
				breakdown.AddLine("AuraDuration",
					fmt.Sprintf("%.2fs ^8(base)", durationBase),
					fmt.Sprintf("x %.4f ^8(duration modifier)", durationMod),
					"rounded up to nearest server tick",
					fmt.Sprintf("= %.3fs", actor.Output["AuraDuration"]),
				)
			}
		}
		durationBase = skillData.ReserveDuration
		if durationBase > 0 {
			durationMod := durationModClamped(skillModList, skillCfg, "Duration")
			actor.Output["ReserveDuration"] = durationBase * durationMod
			actor.Output["ReserveDuration"] = math.Ceil(actor.Output["ReserveDuration"]*data.ServerTickRate) / data.ServerTickRate
			if breakdown != nil && actor.Output["ReserveDuration"] != durationBase {
				breakdown.AddLine("ReserveDuration",
					fmt.Sprintf("%.2fs ^8(base)", durationBase),
					fmt.Sprintf("x %.4f ^8(duration modifier)", durationMod),
					"rounded up to nearest server tick",
					fmt.Sprintf("= %.3fs", actor.Output["ReserveDuration"]),
				)
			}
		}

		//pob:block:src/Modules/CalcOffence.lua:1575-1589
		durationBase = skillData.SoulPreventionDuration
		if durationBase > 0 {
			durationMod := calclib.Mod(skillModList, skillCfg, "SoulGainPreventionDuration",
				utils.Ternary(skillData.SkillEffectAppliesToSoulGainPrevention, "Duration", ""),
				utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
			durationMod = math.Max(durationMod, 0)
			actor.Output["SoulGainPreventionDuration"] = durationBase * durationMod
			actor.Output["SoulGainPreventionDuration"] = math.Max(math.Ceil(actor.Output["SoulGainPreventionDuration"]*data.ServerTickRate), 1) / data.ServerTickRate
			if breakdown != nil && actor.Output["SoulGainPreventionDuration"] != durationBase {
				breakdown.AddLine("SoulGainPreventionDuration",
					fmt.Sprintf("%.2fs ^8(base)", durationBase),
					fmt.Sprintf("x %.4f ^8(duration modifier)", durationMod),
					"rounded up to nearest server tick",
					fmt.Sprintf("= %.3fs", actor.Output["SoulGainPreventionDuration"]),
				)
			}
		}
		//pob:block:src/Modules/CalcOffence.lua:1590-1593
		actor.Output["TotemDurationMod"] = math.Max(calclib.Mod(skillModList, skillCfg, "TotemDuration"), 0)
		totemDurationBase := skillModList.Sum(mod.TypeBase, skillCfg, "TotemDuration")
		actor.Output["TotemDuration"] = math.Ceil(totemDurationBase*actor.Output["TotemDurationMod"]*data.ServerTickRate) / data.ServerTickRate
		if breakdown != nil {
			breakdown.Mod("TotemDurationMod", skillModList, skillCfg, "TotemDuration")
			breakdown.AddLine("TotemDuration", fmt.Sprintf("%.2fs ^8(base)", totemDurationBase))
			if actor.Output["TotemDurationMod"] != 1 {
				breakdown.AddLine("TotemDuration", fmt.Sprintf("x %.4f ^8(duration modifier)", actor.Output["TotemDurationMod"]))
			}
			breakdown.AddLine("TotemDuration", "rounded up to nearest server tick")
			breakdown.AddLine("TotemDuration", fmt.Sprintf("= %.3fs", actor.Output["TotemDuration"]))
		}
		//pob:block:src/Modules/CalcOffence.lua:1605-1608
		actor.Output["ImpaleDurationMod"] = math.Max(calclib.Mod(skillModList, skillCfg, "ImpaleDuration"), 0)
		impaleDurationBase := float64(data.ImpaledDebuffBaseDurationMs) / 1000
		actor.Output["ImpaleDuration"] = math.Ceil(impaleDurationBase*actor.Output["ImpaleDurationMod"]*data.ServerTickRate*debuffDurationMult) / data.ServerTickRate
		if breakdown != nil {
			breakdown.Mod("ImpaleDurationMod", skillModList, skillCfg, "ImpaleDuration")
		}
	}

	// Skill uptime.
	//pob:block:src/Modules/CalcOffence.lua:1625-1655
	if utils.MissingOrFalse(activeSkill.SkillTypes, data.SkillTypeVaal) {
		// exclude vaal skills as we currently don't support soul generation or gain prevention.
		cooldown := actor.Output["Cooldown"]
		for _, durationType := range []string{"Duration", "DurationSecondary", "DurationTertiary", "AuraDuration", "reserveDuration"} {
			duration := actor.Output[durationType]
			if duration == 0 || cooldown == 0 {
				continue
			}
			var uptime float64
			if skillModList.Flag(skillCfg, "NoCooldownRecoveryInDuration") {
				uptime = duration / (cooldown + duration)
			} else {
				uptime = duration / cooldown
			}
			uptime = math.Min(uptime, 1)
			actor.Output[durationType+"Uptime"] = uptime * 100
			if breakdown != nil {
				if skillModList.Flag(skillCfg, "NoCooldownRecoveryInDuration") {
					breakdown.AddLine(durationType+"Uptime",
						fmt.Sprintf("%.2fs / (%.2fs + %.2fs)", duration, cooldown, duration),
						fmt.Sprintf("= %d%%", int(actor.Output[durationType+"Uptime"])),
					)
				} else {
					breakdown.AddLine(durationType+"Uptime",
						fmt.Sprintf("%.2fs / %.2fs", duration, cooldown),
						fmt.Sprintf("= %d%%", int(actor.Output[durationType+"Uptime"])),
					)
				}
			}
		}
	}

	// Calculate costs (may be slightly off due to rounding differences)
	costs := map[string]*costEntry{
		"Mana":                 {Type: "Mana", Upfront: true, Percent: false, Text: "mana", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0, BaseCostRaw: new(float64)},
		"Life":                 {Type: "Life", Upfront: true, Percent: false, Text: "life", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ES":                   {Type: "ES", Upfront: true, Percent: false, Text: "ES", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"Soul":                 {Type: "Soul", Upfront: true, Percent: false, UnaffectedByGenericCostMults: true, Text: "soul", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"Rage":                 {Type: "Rage", Upfront: true, Percent: false, Text: "rage", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ManaPercent":          {Type: "Mana", Upfront: true, Percent: true, Text: "mana", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"LifePercent":          {Type: "Life", Upfront: true, Percent: true, Text: "life", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ManaPerMinute":        {Type: "Mana", Upfront: false, Percent: false, Text: "mana/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"LifePerMinute":        {Type: "Life", Upfront: false, Percent: false, Text: "life/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ManaPercentPerMinute": {Type: "Mana", Upfront: false, Percent: true, Text: "mana/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"LifePercentPerMinute": {Type: "Life", Upfront: false, Percent: true, Text: "life/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ESPerMinute":          {Type: "ES", Upfront: false, Percent: false, Text: "ES/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ESPercentPerMinute":   {Type: "ES", Upfront: false, Percent: true, Text: "ES/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
	}

	costOrder := []string{
		"Mana", "Life", "ES", "Soul", "Rage", "ManaPercent", "LifePercent",
		"ManaPerMinute", "LifePerMinute", "ManaPercentPerMinute", "LifePercentPerMinute",
		"ESPerMinute", "ESPercentPerMinute",
	}

	hasCost := map[string]bool{}
	//pob:block:src/Modules/CalcOffence.lua:1677-1683
	if !skillModList.Flag(skillCfg, "HasNoCost") {
		mult := math.Floor(skillModList.More(skillCfg, "SupportManaMultiplier")*10000+0.0001) / 10000
		additionalLifeCost := skillModList.Sum(mod.TypeBase, skillCfg, "ManaCostAsLifeCost") / 100
		additionalESCost := skillModList.Sum(mod.TypeBase, skillCfg, "ManaCostAsEnergyShieldCost") / 100
		hybridLifeCost := skillModList.Sum(mod.TypeBase, skillCfg, "HybridManaAndLifeCost_Life") / 100

		// First pass to calculate base costs. Used for cost conversion (e.g. Petrified Blood)
		//pob:block:src/Modules/CalcOffence.lua:1684-1750
		for _, resource := range costOrder {
			val := costs[resource]
			skillCost := float64(activeSkill.ActiveEffect.GrantedEffectLevel.Cost[resource])
			if override := skillModList.Override(skillCfg, "Base"+resource+"CostOverride"); override != nil {
				skillCost = override.ValueFloat
			}
			baseCost := utils.RoundTo(utils.Ternary[float64](skillCost > 0, skillCost/costDivisors[resource], 0), 2)
			baseCostNoMult := skillModList.Sum(mod.TypeBase, skillCfg, resource+"CostNoMult")
			divineBlessingCorrection := float64(0)
			if val.Upfront {
				baseCost = baseCost + skillModList.Sum(mod.TypeBase, skillCfg, resource+"CostBase")
				if resource == "Mana" && skillData.BaseManaCostIsAtLeastPercentUnreservedMana > 0 {
					baseCost = max(baseCost, math.Floor((actor.Output["ManaUnreserved"])*skillData.BaseManaCostIsAtLeastPercentUnreservedMana/100))
				}
				val.TotalCost = skillModList.Sum(mod.TypeBase, skillCfg, resource+"Cost", "Cost")
				if resource == "Mana" && activeSkill.SkillTypes[data.SkillTypeReservationBecomesCost] &&
					!val.Percent && !skillModList.Flag(skillCfg, "CostESInsteadOfManaOrLife") &&
					!skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") {
					reservedFlat := utils.GetOr(activeSkill.SkillData, val.Text+"ReservationFlat", utils.GetOr(activeSkill.ActiveEffect.GrantedEffectLevel, val.Text+"ReservationFlat", float64(0)))
					baseCost = baseCost + reservedFlat
					reservedPercent := utils.GetOr(activeSkill.SkillData, val.Text+"ReservationPercent", utils.GetOr(activeSkill.ActiveEffect.GrantedEffectLevel, val.Text+"ReservationPercent", float64(0)))
					baseCost = baseCost + math.Floor(actor.Output[resource]*reservedPercent/100)
					divineBlessingCorrection = utils.RoundTo(actor.Output[resource]*reservedPercent/100*mult, 0) -
						math.Floor(actor.Output[resource]*reservedPercent/100*mult)
				}
			}
			val.BaseCost = val.BaseCost + baseCost
			val.BaseCostNoMult = val.BaseCostNoMult + baseCostNoMult
			val.FinalBaseCost = math.Floor(val.BaseCost*mult) + val.BaseCostNoMult + divineBlessingCorrection
			if val.BaseCostRaw != nil {
				raw := val.BaseCost*mult + val.BaseCostNoMult + divineBlessingCorrection
				val.BaseCostRaw = &raw
			}

			switch val.Type {
			case "Life":
				manaType := strings.ReplaceAll(resource, "Life", "Mana")
				if skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") { // Blood Magic / Lifetap
					val.BaseCost = val.BaseCost + costs[manaType].BaseCost
					val.BaseCostNoMult = val.BaseCostNoMult + costs[manaType].BaseCostNoMult
					val.FinalBaseCost = val.FinalBaseCost + costs[manaType].FinalBaseCost
					costs[manaType].BaseCost = 0
					costs[manaType].zeroBaseCostRaw()
					costs[manaType].FinalBaseCost = 0
					costs[manaType].BaseCostNoMult = 0
				} else if (additionalLifeCost > 0 || hybridLifeCost > 0) && !skillModList.Flag(skillCfg, "CostESInsteadOfManaOrLife") {
					val.BaseCost = costs[manaType].BaseCost
					val.FinalBaseCost = val.FinalBaseCost + utils.RoundTo(costs[manaType].FinalBaseCost*(hybridLifeCost+additionalLifeCost), 0)
				}
			case "ES":
				manaType := strings.ReplaceAll(resource, "ES", "Mana")
				lifeType := strings.ReplaceAll(resource, "ES", "Life")
				if skillModList.Flag(skillCfg, "CostESInsteadOfManaOrLife") { // Whispers of Infinity
					val.BaseCost = val.BaseCost + costs[manaType].BaseCost + costs[lifeType].BaseCost
					val.BaseCostNoMult = val.BaseCostNoMult + costs[manaType].BaseCostNoMult + costs[lifeType].BaseCostNoMult
					val.FinalBaseCost = val.FinalBaseCost + costs[manaType].FinalBaseCost + costs[lifeType].FinalBaseCost
					costs[manaType].BaseCost = 0
					costs[manaType].zeroBaseCostRaw()
					costs[manaType].FinalBaseCost = 0
					costs[manaType].BaseCostNoMult = 0
					costs[lifeType].BaseCost = 0
					costs[lifeType].zeroBaseCostRaw()
					costs[lifeType].FinalBaseCost = 0
					costs[lifeType].BaseCostNoMult = 0
				} else if additionalESCost > 0 {
					val.BaseCost = costs[manaType].BaseCost
					val.FinalBaseCost = val.FinalBaseCost + utils.RoundTo(costs[manaType].FinalBaseCost*additionalESCost, 0)
				}
			case "Rage":
				if skillModList.Flag(skillCfg, "CostRageInsteadOfSouls") {
					val.BaseCost = costs["Soul"].BaseCost
					val.BaseCostNoMult = val.BaseCostNoMult + costs["Soul"].BaseCostNoMult
					val.FinalBaseCost = costs["Soul"].BaseCost
					mult = 1
					costs["Soul"].BaseCost = 0
					costs["Soul"].BaseCostNoMult = 0
					costs["Soul"].FinalBaseCost = 0
				}
			}
		}

		//pob:block:src/Modules/CalcOffence.lua:1751-1800
		for _, resource := range costOrder {
			val := costs[resource]
			costKey := utils.Ternary(val.Upfront, resource, strings.ReplaceAll(resource, "Minute", "Second"))
			costName := costKey + "Cost"
			costNameRaw := costName + "Raw"
			hasCost[costKey] = val.BaseCost > 0 || val.TotalCost > 0 || val.BaseCostNoMult > 0 || val.FinalBaseCost > 0
			actor.Output[costKey+"HasCost"] = utils.Ternary(hasCost[costKey], 1.0, 0.0)

			costEfficiency := calclib.Mod(skillModList, skillCfg, val.Type+"CostEfficiency", "CostEfficiency")

			moreCost := float64(1)
			var moreType, inc float64
			if !val.UnaffectedByGenericCostMults {
				actor.Output[costName] = val.FinalBaseCost
				moreType = skillModList.More(skillCfg, val.Type+"Cost")
				moreCost = skillModList.More(skillCfg, "Cost")
				inc = skillModList.Sum(mod.TypeIncrease, skillCfg, val.Type+"Cost", "Cost")
				if val.BaseCostRaw != nil {
					actor.Output[costNameRaw] = max(0, max(0, (1+inc/100)**val.BaseCostRaw*moreType*moreCost/costEfficiency)+val.TotalCost)
				}
				actor.Output[costName] = max(0, costRound(inc < 0, (1+inc/100)*actor.Output[costName]))
				actor.Output[costName] = max(0, costRound(moreType < 1, moreType*actor.Output[costName]))
				actor.Output[costName] = max(0, costRound(moreCost < 1, moreCost*actor.Output[costName]))
				actor.Output[costName] = max(0, actor.Output[costName]/costEfficiency)
				actor.Output[costName] = max(0, actor.Output[costName]+val.TotalCost)
				if val.Type == "Mana" && hybridLifeCost > 0 {
					actor.Output[costName] = max(0, math.Floor((1-hybridLifeCost)*actor.Output[costName]))
					if val.BaseCostRaw != nil {
						actor.Output[costNameRaw] = max(0, (1-hybridLifeCost)*actor.Output[costNameRaw])
					}
				}
			} else {
				moreType = skillModList.More(skillCfg, val.Type+"Cost")
				inc = skillModList.Sum(mod.TypeIncrease, skillCfg, val.Type+"Cost")
				actor.Output[costName] = math.Floor(val.BaseCost + val.BaseCostNoMult)
				actor.Output[costName] = max(0, (1+inc/100)*actor.Output[costName])
				actor.Output[costName] = max(0, moreType*actor.Output[costName])
				actor.Output[costName] = max(0, actor.Output[costName]/costEfficiency)
				actor.Output[costName] = max(0, actor.Output[costName]+val.TotalCost)
				if val.BaseCostRaw != nil {
					actor.Output[costNameRaw] = max(0, max(0, (1+inc/100)*(*val.BaseCostRaw+val.BaseCostNoMult)*moreType/costEfficiency)+val.TotalCost)
				}
			}
			if breakdown != nil && hasCost[costKey] {
				breakdown.AddLine(costName,
					fmt.Sprintf("%.2f"+utils.Ternary(val.Percent, "%%", "")+" ^8(base "+val.Text+" cost)", val.BaseCost),
				)
				if mult != 1 {
					breakdown.AddLine(costName, fmt.Sprintf("x %.4f ^8(cost multiplier)", mult))
				}
				if val.BaseCostNoMult != 0 {
					breakdown.AddLine(costName, fmt.Sprintf("+ %d ^8(additional "+val.Text+" cost)", int(val.BaseCostNoMult)))
				}
				if val.Type == "Life" && (hybridLifeCost+additionalLifeCost) != 0 && !skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") {
					breakdown.AddLine(costName, fmt.Sprintf("x %.2f ^8(mana cost conversion)", hybridLifeCost+additionalLifeCost))
				}
				if val.Type == "ES" && additionalESCost != 0 && !skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") && !skillModList.Flag(skillCfg, "CostESInsteadOfManaOrLife") {
					breakdown.AddLine(costName, fmt.Sprintf("x %.2f ^8(mana cost conversion)", additionalESCost))
				}
				if inc != 0 {
					breakdown.AddLine(costName, fmt.Sprintf("x %.2f ^8(increased/reduced "+val.Text+" cost)", 1+inc/100))
				}
				if moreCost != 1 {
					breakdown.AddLine(costName, fmt.Sprintf("x %.2f ^8(more/less cost)", moreCost))
				}
				if moreType != 1 {
					breakdown.AddLine(costName, fmt.Sprintf("x %.2f ^8(more/less "+val.Text+" cost)", moreType))
				}
				if costEfficiency != 1 {
					breakdown.AddLine(costName, fmt.Sprintf("/ %.2f ^8("+val.Text+" cost efficiency)", costEfficiency))
				}
				if val.TotalCost != 0 {
					breakdown.AddLine(costName, fmt.Sprintf("%+d ^8(total "+val.Text+" cost)", int(val.TotalCost)))
				}
				if val.Type == "Mana" && hybridLifeCost > 0 {
					breakdown.AddLine(costName, fmt.Sprintf("x %.2f ^8(%d%% paid for with life)", 1-hybridLifeCost, int(hybridLifeCost*100)))
				}
				breakdown.AddLine(costName, fmt.Sprintf("= %"+utils.Ternary(val.Upfront, "d", ".2f")+utils.Ternary(val.Percent, "%%", ""), actor.Output[costName]))
			}
		}
	}
	/*
		TODO // account for Sacrificial Zeal
		// Note: Sacrificial Zeal grants Added Spell Physical Damage equal to 25% of the Skill's Mana Cost, and causes you to take Physical Damage over Time, for 4 seconds
		if skillModList.Flag(nil, "Condition:SacrificialZeal") {
			multiplier := 0.25
			skillModList.AddMod("PhysicalMin", "BASE", math.Floor(actor.Output["ManaCost"] * multiplier), "Sacrificial Zeal", ModFlag.Spell)
			skillModList.AddMod("PhysicalMax", "BASE", math.Floor(actor.Output["ManaCost"] * multiplier), "Sacrificial Zeal", ModFlag.Spell)
		}
	*/
	runSkillFunc("preDamageFunc")

	//pob:block:src/Modules/CalcOffence.lua:1850-1853
	if activeSkill.SkillTypes[data.SkillTypeBrand] {
		damageLabel := utils.Ternary(skillData.CountsAttachedBrandsInDamage, "Average Damage", "DPS")
		count := int(actor.Output["AttachedBrandCount"])
		activeSkill.infoMessage = fmt.Sprintf("%s for %d attached Brand%s", damageLabel, count, utils.Ternary(count == 1, "", "s"))
	}

	if skillData.ExplodeCorpse && (skillData.CorpseLife != 0 || env.EnemyLevel != 0) {
		localCorpseLife := skillData.CorpseLife
		if localCorpseLife == 0 && env.EnemyLevel < len(data.MonsterLifeTable) {
			localCorpseLife = data.MonsterLifeTable[env.EnemyLevel]
		}
		damageType := skillData.CorpseExplosionDamageType
		if damageType == "" {
			damageType = string(data.DamageTypeFire)
		}
		lifeMultiplier := skillData.CorpseExplosionLifeMultiplier
		if lifeMultiplier == 0 {
			lifeMultiplier = skillData.SelfFireExplosionLifeMultiplier
		}
		utils.Set(skillData, damageType+"BonusMin", localCorpseLife*lifeMultiplier)
		utils.Set(skillData, damageType+"BonusMax", localCorpseLife*lifeMultiplier)
	}

	//pob:block:src/Modules/CalcOffence.lua:1856
	monsterLife := skillData.CorpseLife
	if monsterLife == 0 {
		monsterLife = 100
		if env.EnemyLevel >= 1 && env.EnemyLevel < len(data.MonsterLifeTable) {
			monsterLife = data.MonsterLifeTable[env.EnemyLevel]
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:1862-1869
	if skillFlags[SkillFlagMonsterExplode] {
		for _, damageType := range data.DamageType("").Values() {
			percentage := utils.GetOr(skillData, string(damageType)+"EffectiveExplodePercentage", float64(0))
			base := percentage * monsterLife / 100
			utils.Set(skillData, string(damageType)+"Min", base)
			utils.Set(skillData, string(damageType)+"Max", base)
		}
	}

	// Cache global damage disabling flags
	//pob:src/Modules/CalcOffence.lua:1871-1875
	canDeal := make(map[data.DamageType]bool)
	for _, damageType := range data.DamageType("").Values() {
		canDeal[damageType] = !skillModList.Flag(skillCfg, "DealNo"+string(damageType), "DealNoDamage")
	}

	// Calculate damage conversion percentages
	activeSkill.ConversionTable = buildConversionTable(skillModList, skillCfg)
	activeSkill.CfgConversionTable = map[*moddb.ListCfg]map[data.DamageType]ConversionTable{}
	if activeSkill.Weapon1Cfg != nil {
		activeSkill.CfgConversionTable[activeSkill.Weapon1Cfg] = buildConversionTable(skillModList, activeSkill.Weapon1Cfg)
	}
	if activeSkill.Weapon2Cfg != nil {
		activeSkill.CfgConversionTable[activeSkill.Weapon2Cfg] = buildConversionTable(skillModList, activeSkill.Weapon2Cfg)
	}
	// Configure damage passes
	passList := make([]*DamagePass, 0)
	if skillFlags[SkillFlagAttack] {
		outputTable[OutTableMainHand] = make(map[string]float64)
		outputTable[OutTableOffHand] = make(map[string]float64)
		//pob:block:src/Modules/CalcOffence.lua:1938
		_, preciseTechnique := env.KeystonesAdded["Precise Technique"]
		actor.Output["PreciseTechnique"] = utils.Ternary(preciseTechnique, 1.0, 0.0)
		critOverride := skillModList.Override(skillCfg, "WeaponBaseCritChance")
		if skillFlags[SkillFlagWeapon1Attack] {
			if breakdown != nil {
			}
			activeSkill.Weapon1Cfg.SkillStats = outputTable[OutTableMainHand]
			//pob:src/Modules/CalcOffence.lua:1945
			source := actor.WeaponData1
			if source != nil {
				sourceCopy := *source
				source = &sourceCopy
			}
			if critOverride != nil && source != nil && source.Type != "None" {
				source.CritChance = critOverride.Float()
			}
			passList = append(passList, &DamagePass{
				Label:     "Main Hand",
				Source:    source,
				Config:    activeSkill.Weapon1Cfg,
				Output:    outputTable[OutTableMainHand],
				Breakdown: breakdown,
			})
		}

		if skillFlags[SkillFlagWeapon2Attack] {
			if breakdown != nil {
			}
			activeSkill.Weapon2Cfg.SkillStats = outputTable[OutTableOffHand]
			//pob:src/Modules/CalcOffence.lua:1962
			source := actor.WeaponData2
			if source != nil {
				sourceCopy := *source
				source = &sourceCopy
			}
			if critOverride != nil && source != nil && source.Type != "None" {
				source.CritChance = critOverride.Float()
			}
			if skillData.CritChance > 0 {
				source.CritChance = skillData.CritChance
			}
			if skillData.SetOffHandPhysicalMin != 0 && skillData.SetOffHandPhysicalMax != 0 {
				source.PhysicalMin = skillData.SetOffHandPhysicalMin
				source.PhysicalMax = skillData.SetOffHandPhysicalMax
			}
			//pob:block:src/Modules/CalcOffence.lua:1973-1980
			if skillData.SetOffHandFireMin != 0 && skillData.SetOffHandFireMax != 0 {
				source.FireMin = skillData.SetOffHandFireMin
				source.FireMax = skillData.SetOffHandFireMax
			}
			if skillData.SetOffHandColdMin != 0 && skillData.SetOffHandColdMax != 0 {
				source.ColdMin = skillData.SetOffHandColdMin
				source.ColdMax = skillData.SetOffHandColdMax
			}
			if skillData.AttackTime != 0 {
				source.AttackRate = 1000 / skillData.AttackTime
			}
			passList = append(passList, &DamagePass{
				Label:     "Off Hand",
				Source:    source,
				Config:    activeSkill.Weapon2Cfg,
				Output:    outputTable[OutTableOffHand],
				Breakdown: breakdown,
			})
		}
	} else {
		passList = append(passList, &DamagePass{
			Label:     "Skill",
			Source:    skillData,
			Config:    skillCfg,
			Output:    actor.Output,
			Breakdown: breakdown,
		})
	}

	type CombineMode string
	const (
		ModeOr            = CombineMode("OR")
		ModeAdd           = CombineMode("ADD")
		ModeAverage       = CombineMode("AVERAGE")
		ModeHarmonicMean  = CombineMode("HARMONICMEAN")
		ModeChance        = CombineMode("CHANCE")
		ModeChanceAilment = CombineMode("CHANCE_AILMENT")
		ModeDPS           = CombineMode("DPS")
	)

	mainOrOff := func(stat string) {
		switch {
		case utils.Has(outputTable[OutTableMainHand], stat):
			actor.Output[stat] = outputTable[OutTableMainHand][stat]
		case utils.Has(outputTable[OutTableOffHand], stat):
			actor.Output[stat] = outputTable[OutTableOffHand][stat]
		default:
			delete(actor.Output, stat)
		}
	}

	combineStat := func(stat string, mode CombineMode, idk ...string) {
		// Combine stats from Main Hand and Off Hand according to the mode
		if mode == ModeOr || utils.MissingOrFalse(skillFlags, SkillFlagBothWeaponAttack) {
			mainOrOff(stat)
		} else if mode == ModeAdd {
			actor.Output[stat] = outputTable[OutTableMainHand][stat] + outputTable[OutTableOffHand][stat]
		} else if mode == ModeAverage {
			sum := outputTable[OutTableMainHand][stat] + outputTable[OutTableOffHand][stat]
			actor.Output[stat] = sum / 2
		} else if mode == ModeHarmonicMean {
			mainHand := outputTable[OutTableMainHand][stat]
			offHand := outputTable[OutTableOffHand][stat]
			if mainHand == 0 || offHand == 0 {
				actor.Output[stat] = 0
			} else {
				actor.Output[stat] = 2 / ((1 / mainHand) + (1 / offHand))
			}
		} else if mode == ModeChance {
			if utils.Has(outputTable[OutTableMainHand], stat) && utils.Has(outputTable[OutTableOffHand], stat) {
				mainChance := outputTable[OutTableMainHand][idk[0]] * outputTable[OutTableMainHand]["HitChance"]
				offChance := outputTable[OutTableOffHand][idk[0]] * outputTable[OutTableOffHand]["HitChance"]

				if skillData.DoubleHitsWhenDualWielding {
					mainChance = mainChance / 10000
					offChance = offChance / 10000
					actor.Output[stat] = outputTable[OutTableMainHand][stat]*mainChance + outputTable[OutTableOffHand][stat]*offChance
					if breakdown != nil {
						breakdown.AddLine(stat, "Contribution from Main Hand:")
						breakdown.AddLine(stat, fmt.Sprintf("%.1f", actor.OutputTable[OutTableMainHand][stat]))
						breakdown.AddLine(stat, fmt.Sprintf("x %.3f ^8(chance of main hand)", mainChance))
						breakdown.AddLine(stat, fmt.Sprintf("= %.1f", actor.OutputTable[OutTableMainHand][stat]*mainChance))
						breakdown.AddLine(stat, "Contribution from Off Hand:")
						breakdown.AddLine(stat, fmt.Sprintf("%.1f", actor.OutputTable[OutTableOffHand][stat]))
						breakdown.AddLine(stat, fmt.Sprintf("x %.3f ^8(chance of  off hand)", offChance))
						breakdown.AddLine(stat, fmt.Sprintf("= %.1f", actor.OutputTable[OutTableOffHand][stat]*offChance))
						breakdown.AddLine(stat, "Total:")
						breakdown.AddLine(stat, fmt.Sprintf("%.1f + %.1f", actor.OutputTable[OutTableMainHand][stat]*mainChance, actor.OutputTable[OutTableOffHand][stat]*offChance))
						breakdown.AddLine(stat, fmt.Sprintf("= %.1f", actor.Output[stat]))
					}
					return
				}

				mainPortion := mainChance / (mainChance + offChance)
				offPortion := offChance / (mainChance + offChance)
				actor.Output[stat] = outputTable[OutTableMainHand][stat]*mainPortion + outputTable[OutTableOffHand][stat]*offPortion
				if breakdown != nil {
					breakdown.AddLine(stat, "Contribution from Main Hand:")
					breakdown.AddLine(stat, fmt.Sprintf("%.1f", actor.OutputTable[OutTableMainHand][stat]))
					breakdown.AddLine(stat, fmt.Sprintf("x %.3f ^8(portion of instances created by main hand)", mainPortion))
					breakdown.AddLine(stat, fmt.Sprintf("= %.1f", actor.OutputTable[OutTableMainHand][stat]*mainPortion))
					breakdown.AddLine(stat, "Contribution from Off Hand:")
					breakdown.AddLine(stat, fmt.Sprintf("%.1f", actor.OutputTable[OutTableOffHand][stat]))
					breakdown.AddLine(stat, fmt.Sprintf("x %.3f ^8(portion of instances created by off hand)", offPortion))
					breakdown.AddLine(stat, fmt.Sprintf("= %.1f", actor.OutputTable[OutTableOffHand][stat]*offPortion))
					breakdown.AddLine(stat, "Total:")
					breakdown.AddLine(stat, fmt.Sprintf("%.1f + %.1f", actor.OutputTable[OutTableMainHand][stat]*mainPortion, actor.OutputTable[OutTableOffHand][stat]*offPortion))
					breakdown.AddLine(stat, fmt.Sprintf("= %.1f", actor.Output[stat]))
				}
			} else {
				//pob:src/Modules/CalcOffence.lua:2062
				mainOrOff(stat)
			}
		} else if mode == ModeChanceAilment {
			if utils.Has(outputTable[OutTableMainHand], stat) && utils.Has(outputTable[OutTableOffHand], stat) {
				maxInstance := max(outputTable[OutTableMainHand][stat], outputTable[OutTableOffHand][stat])
				minInstance := min(outputTable[OutTableMainHand][stat], outputTable[OutTableOffHand][stat])

				stackName := strings.ReplaceAll(stat, "DPS", "") + "Stacks"
				stacks := float64(1)
				if utils.Has(actor.Output, stackName) {
					stacks = actor.Output[stackName]
				}
				stacksMax := float64(1)
				if utils.Has(actor.Output, stackName+"Max") {
					stacksMax = actor.Output[stackName+"Max"]
				}
				maxInstanceStacks := min(1, stacks/stacksMax)

				actor.Output[stat] = maxInstance*maxInstanceStacks + minInstance*(1-maxInstanceStacks)
				if breakdown != nil {
					breakdown.AddLine(stat, "")
					breakdown.AddLine(stat, fmt.Sprintf("%.2f%% of ailment stacks use maximum damage", maxInstanceStacks*100))
					breakdown.AddLine(stat, fmt.Sprintf("Max Damage comes from %s", utils.Ternary(actor.OutputTable[OutTableMainHand][stat] >= actor.OutputTable[OutTableOffHand][stat], "Main Hand", "Off Hand")))
					breakdown.AddLine(stat, fmt.Sprintf("= %.1f", maxInstance*maxInstanceStacks))
					if maxInstanceStacks < 1 {
						breakdown.AddLine(stat, fmt.Sprintf("%.2f%% of ailment stacks use non-maximum damage", (1-maxInstanceStacks)*100))
						breakdown.AddLine(stat, fmt.Sprintf("= %.1f", minInstance*(1-maxInstanceStacks)))
					}
					breakdown.AddLine(stat, "")
					breakdown.AddLine(stat, "Total:")
					if maxInstanceStacks < 1 {
						breakdown.AddLine(stat, fmt.Sprintf("%.1f + %.1f", maxInstance*maxInstanceStacks, minInstance*(1-maxInstanceStacks)))
					}
					breakdown.AddLine(stat, fmt.Sprintf("= %.1f", actor.Output[stat]))
				}
			} else {
				//pob:src/Modules/CalcOffence.lua:2093
				mainOrOff(stat)
				if breakdown != nil {
					breakdown.AddLine(stat, fmt.Sprintf("All ailment stacks comes from %s", utils.Ternary(actor.OutputTable[OutTableMainHand][stat] != 0, "Main Hand", "Off Hand")))
				}
			}
		} else if mode == ModeDPS {
			actor.Output[stat] = outputTable[OutTableMainHand][stat] + outputTable[OutTableOffHand][stat]
			if !skillData.DoubleHitsWhenDualWielding {
				actor.Output[stat] = actor.Output[stat] / 2
			}
		}
	}

	var storedMainHandAccuracy *float64
	var storedMainHandAccuracyVsEnemy *float64
	for _, pass := range passList {
		//pob:block:src/Modules/CalcOffence.lua:2115-2119
		if pass.Output["Repeats"] > 1 {
			pass.Output["AverageBurstHits"] = pass.Output["Repeats"]
		}

		// Calculate hit chance
		//pob:block:src/Modules/CalcOffence.lua:2121-2130
		accuracyMore := skillModList.More(pass.Config, "Accuracy")
		base := skillModList.Sum(mod.TypeBase, pass.Config, "Accuracy")
		inc := skillModList.Sum(mod.TypeIncrease, pass.Config, "Accuracy")
		baseVsEnemy := skillModList.Sum(mod.TypeBase, pass.Config, "Accuracy", "AccuracyVsEnemy")
		incVsEnemy := skillModList.Sum(mod.TypeIncrease, pass.Config, "Accuracy", "AccuracyVsEnemy")
		moreVsEnemy := skillModList.More(pass.Config, "Accuracy", "AccuracyVsEnemy")

		pass.Output["Accuracy"] = math.Max(0, math.Floor(base*(1+inc/100)*accuracyMore))
		accuracyVsEnemy := math.Max(0, math.Floor(baseVsEnemy*(1+incVsEnemy/100)*moreVsEnemy))
		if breakdown != nil {
			breakdown.Simple(nil, pass.Config, actor.Output["Accuracy"], "Accuracy")
		}

		if skillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.Label == "Main Hand" {
			storedMainHandAccuracy = utils.Ptr(pass.Output["Accuracy"])
			storedMainHandAccuracyVsEnemy = utils.Ptr(accuracyVsEnemy)
		} else if skillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.Label == "Off Hand" && storedMainHandAccuracy != nil {
			pass.Output["Accuracy"] = *storedMainHandAccuracy
			if storedMainHandAccuracyVsEnemy != nil {
				accuracyVsEnemy = *storedMainHandAccuracyVsEnemy
			}
			if breakdown != nil {
				breakdown.AddLine("Accuracy",
					"Using Main Hand Accuracy due to Mastery: "+fmt.Sprint(pass.Output["Accuracy"]),
				)
			}
		}

		if utils.MissingOrFalse(skillFlags, SkillFlagAttack) ||
			skillModList.Flag(pass.Config, "CannotBeEvaded") ||
			skillData.CannotBeEvaded ||
			(env.ModeEffective && enemyDB.Flag(nil, "CannotEvade")) {
			pass.Output["AccuracyHitChance"] = 100
		} else {
			enemyEvasion := math.Max(math.Round(CalcVal(enemyDB, "Evasion", nil)), 0)
			pass.Output["AccuracyHitChance"] = CalcHitChance(enemyEvasion, accuracyVsEnemy) * CalcMod(skillModList, pass.Config, "HitChance")
			if breakdown != nil {
				breakdown.AddLine("AccuracyHitChance",
					"Enemy level: "+fmt.Sprint(env.EnemyLevel)+utils.Ternary(env.Build.GetNumberOption("enemyLevel") != 0, " ^8(overridden from the Configuration tab", " ^8(can be overridden in the Configuration tab)"),
					"Average enemy evasion: "+fmt.Sprint(enemyEvasion),
					"Approximate hit chance: "+fmt.Sprint(pass.Output["AccuracyHitChance"])+"%",
				)
			}
		}

		// enemy block chance
		//pob:block:src/Modules/CalcOffence.lua:1632-1634
		pass.Output["enemyBlockChance"] = min(max(enemyDB.Sum(mod.TypeBase, pass.Config, "BlockChance"), 0), 100)
		pass.Output["HitChance"] = pass.Output["AccuracyHitChance"] * (1 - pass.Output["enemyBlockChance"]/100)
		//pob:block:src/Modules/CalcOffence.lua:2182-2184
		if pass.Output["enemyBlockChance"] > 0 && !isAttack {
			actor.Output["enemyHasSpellBlock"] = 1
		}
		if breakdown != nil && pass.Output["enemyBlockChance"] > 0 {
			if pass.Output["AccuracyHitChance"] < 100 {
				breakdown.AddLine("HitChance",
					"Accuracy Hit Chance: "+fmt.Sprint(pass.Output["AccuracyHitChance"])+"%",
					"Enemy Block Chance: "+fmt.Sprint(pass.Output["enemyBlockChance"])+"%",
					"Approximate hit chance: "+fmt.Sprint(pass.Output["HitChance"])+"%",
				)
			} else {
				breakdown.AddLine("HitChance",
					"Enemy Block Chance: "+fmt.Sprint(pass.Output["enemyBlockChance"])+"%",
					"Approximate hit chance: "+fmt.Sprint(pass.Output["HitChance"])+"%",
				)
			}
		}
		// Check Precise Technique Keystone condition per pass as MH/OH might have different values
		//pob:src/Modules/CalcOffence.lua:1653-1655
		condName := strings.ReplaceAll(pass.Label, " ", "") + "AccRatingHigherThanMaxLife"
		skillModList.Conditions[condName] = pass.Output["Accuracy"] > env.Player.Output["Life"]

		// Calculate attack/cast speed
		//pob:src/Modules/CalcOffence.lua:1657-1660
		if (activeSkill.ActiveEffect.GrantedEffect.CastTime() == 0) && !PoBSkillDeclaresNoCastTime(activeSkill.ActiveEffect.GrantedEffect.Raw) && skillData.CastTimeOverride == 0 && !skillData.Triggered {
			pass.Output["Time"] = 0
			pass.Output["Speed"] = 0
		} else if skillData.TimeOverride > 0 {
			pass.Output["Time"] = skillData.TimeOverride
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else if skillData.FixedCastTime {
			pass.Output["Time"] = activeSkill.ActiveEffect.GrantedEffect.CastTime()
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else if skillData.TriggerTime > 0 && skillData.Triggered {
			activeSkillsLinked := skillModList.Sum(mod.TypeBase, pass.Config, "ActiveSkillsLinkedToTrigger")
			if activeSkillsLinked > 0 {
				pass.Output["Time"] = skillData.TriggerTime / (1 + skillModList.Sum(mod.TypeIncrease, pass.Config, "CooldownRecovery")/100) * activeSkillsLinked
			} else {
				pass.Output["Time"] = skillData.TriggerTime / (1 + skillModList.Sum(mod.TypeIncrease, pass.Config, "CooldownRecovery")/100)
			}
			pass.Output["TriggerTime"] = pass.Output["Time"]
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else if skillData.TriggerRate != nil && skillData.Triggered {
			/*
				TODO // Account for trigger unleash
				if skillData.triggerUnleash {
					// process the source trigger skill to get it's full data
					calcMode := env.mode == "CALCS" and "CALCS" or "MAIN"
					for _, triggerSkill := range (actor.activeSkillList) {
						if cacheSkillUUID(triggerSkill) == skillData.triggerSourceUUID {
							calcs.buildActiveSkill(env, calcMode, triggerSkill)
							break
						}
					}
					cachedSourceSkill := GlobalCache.cachedData[calcMode][skillData.triggerSourceUUID]
					// if properly processed, get it's dpsMultiplier to increase triggerRate
					if cachedSourceSkill {
						skillData.unleashTriggerRate = skillData.triggerRate * (cachedSourceSkill.ActiveSkill.skillData.dpsMultiplier or 1)
						if breakdown != nil {
							breakdown.Speed = {
								fmt.Sprintf("%.2f ^8(trigger rate)", skillData.triggerRate),
								fmt.Sprintf("* %.2f ^8(multiplier from Unleash)", cachedSourceSkill.ActiveSkill.skillData.dpsMultiplier or 1),
								fmt.Sprintf("= %.2f", skillData.unleashTriggerRate),
							}
						}
						// over-write the triggerRate modifier after breakdown as other calcs use it
						skillData.triggerRate = skillData.unleashTriggerRate
					}
					// give this activeSkill "HasSeals" flag so Configuration Option for UseMaxUnleash is available
					activeSkill.skillFlags.HasSeals = true
				}
			*/
			pass.Output["Time"] = 1 / *skillData.TriggerRate
			pass.Output["TriggerTime"] = pass.Output["Time"]
			pass.Output["Speed"] = *skillData.TriggerRate
			skillData.ShowAverage = false
		} else if skillData.TriggeredByBrand && skillData.Triggered {
			ArcanistSpellsLinked := skillModList.Sum(mod.TypeBase, pass.Config, "ArcanistSpellsLinked")
			if ArcanistSpellsLinked == 0 {
				ArcanistSpellsLinked = 1
			}
			pass.Output["Time"] = 1 / (1 + skillModList.Sum(mod.TypeIncrease, pass.Config, "Speed", "BrandActivationFrequency")/100) / skillModList.More(pass.Config, "BrandActivationFrequency") * ArcanistSpellsLinked
			pass.Output["TriggerTime"] = pass.Output["Time"]
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else {
			baseTime := float64(0)
			if skillFlags[SkillFlagAttack] {
				//pob:src/Modules/CalcOffence.lua:2231-2232
				if skillData.AttackSpeedMultiplier != 0 && pass.Source != nil && pass.Source.AttackRate != 0 {
					pass.Source.AttackRate *= 1 + skillData.AttackSpeedMultiplier/100
				}
				//pob:src/Modules/CalcOffence.lua:2234-2236
				if skillData.CastTimeOverridesAttackTime {
					// Skill is overriding weapon attack speed
					baseTime = activeSkill.ActiveEffect.GrantedEffect.CastTime() / (1 + (utils.OrDefault(pass.Source.AttackSpeedInc, 0))/100)
				} else if CalcMod(skillModList, skillCfg, "SkillAttackTime") > 0 {
					baseTime = (1/utils.OrDefault(pass.Source.AttackRate, 1) + skillModList.Sum(mod.TypeBase, pass.Config, "Speed")) * CalcMod(skillModList, skillCfg, "SkillAttackTime")
				} else {
					baseTime = 1/utils.OrDefault(pass.Source.AttackRate, 1) + skillModList.Sum(mod.TypeBase, pass.Config, "Speed")
				}
			} else {
				baseTime = 1
				if skillData.CastTimeOverride != 0 {
					baseTime = skillData.CastTimeOverride
				} else if activeSkill.ActiveEffect.GrantedEffect.CastTime() != 0 {
					baseTime = activeSkill.ActiveEffect.GrantedEffect.CastTime()
				}
			}

			more := skillModList.More(pass.Config, "Speed")

			pass.Output["Repeats"] = actor.Output["Repeats"]

			//pob:block:src/Modules/CalcOffence.lua:2249-2270
			if skillModList.Flag(nil, "HasTrauma") {
				effectiveAttackRateCap := data.ServerTickRate * pass.Output["Repeats"]
				duration := skillModList.Sum(mod.TypeBase, pass.Config, "TraumaDuration") *
					CalcMod(skillModList, skillCfg, "Duration")
				traumaPerAttack := 1 + min(skillModList.Sum(mod.TypeBase, pass.Config, "ExtraTrauma"), 100)/100
				incAttackSpeedPerTrauma := skillModList.Sum(mod.TypeIncrease, skillCfg, "SpeedPerTrauma")
				configTrauma := skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:TraumaStacks")
				traumaInc := skillModList.Sum(mod.TypeIncrease, pass.Config, "Speed") - incAttackSpeedPerTrauma*configTrauma
				attackSpeedBeforeInc := 1 / baseTime * actor.Output["ActionSpeedMod"] * more
				incAttackSpeedPerTraumaCap := (effectiveAttackRateCap - attackSpeedBeforeInc*(1+traumaInc/100)) / attackSpeedBeforeInc * 100
				traumaRateBeforeInc := traumaPerAttack * (pass.Output["HitChance"] / 100) * attackSpeedBeforeInc / pass.Output["Repeats"]
				trauma := traumaRateBeforeInc * (1 + traumaInc/100) / (1/duration - traumaRateBeforeInc*incAttackSpeedPerTrauma/100)
				if trauma < 0 || incAttackSpeedPerTrauma*trauma > incAttackSpeedPerTraumaCap {
					trauma = traumaPerAttack * (pass.Output["HitChance"] / 100) * effectiveAttackRateCap / pass.Output["Repeats"] * duration
				}
				if utils.HasTrue(skillFlags, SkillFlagBothWeaponAttack) {
					trauma /= 2
				}
				skillModList.AddMod(mod.NewFloat("Multiplier:SustainableTraumaStacks", mod.TypeBase, trauma).
					Source("Maximum Sustainable Trauma Stacks"))
			}

			//pob:block:src/Modules/CalcOffence.lua:2329-2331
			if skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:TraumaStacks") == 0 {
				skillModList.AddMod(mod.NewFloat("Multiplier:TraumaStacks", mod.TypeBase,
					skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:SustainableTraumaStacks")).
					Source("Maximum Sustainable Trauma Stacks"))
			}

			inc := skillModList.Sum(mod.TypeIncrease, pass.Config, "Speed")

			if skillFlags[SkillFlagWarcry] {
				pass.Output["Speed"] = 1 / pass.Output["WarcryCastTime"]
			} else {
				pass.Output["Speed"] = 1 / baseTime * utils.RoundTo((1+inc/100)*more, 2)
			}
			pass.Output["CastRate"] = pass.Output["Speed"]

			if skillFlags[SkillFlagSelfCast] {
				// Self-cast skill; apply action speed
				pass.Output["Speed"] = pass.Output["Speed"] * actor.Output["ActionSpeedMod"]
				pass.Output["CastRate"] = pass.Output["Speed"]
			}

			//pob:block:src/Modules/CalcOffence.lua:1820-1826
			if skillFlags[SkillFlagTotem] {
				// Totem skill. Apply action speed
				totemActionSpeed := 1 + (modDB.Sum(mod.TypeIncrease, nil, "TotemActionSpeed") / 100)
				pass.Output["TotemActionSpeed"] = totemActionSpeed
				pass.Output["Speed"] = pass.Output["Speed"] * totemActionSpeed
				pass.Output["CastRate"] = pass.Output["Speed"]
				//pob:src/Modules/CalcOffence.lua:2352-2355
				if skillData.TotemFireOnce {
					pass.Output["HitTime"] = 1/pass.Output["Speed"] + actor.Output["TotemPlacementTime"]
					pass.Output["HitSpeed"] = 1 / pass.Output["HitTime"]
				}
			}

			//pob:src/Modules/CalcOffence.lua:2357-2360
			if utils.Has(actor.Output, "Cooldown") {
				pass.Output["Cooldown"] = actor.Output["Cooldown"]
				pass.Output["Speed"] = math.Min(pass.Output["Speed"], 1/pass.Output["Cooldown"]*pass.Output["Repeats"])
			}

			if utils.Has(pass.Output, "Cooldown") && skillFlags[SkillFlagSelfCast] {
				skillFlags[SkillFlagNotAverage] = true
				skillFlags[SkillFlagShowAverage] = false
				skillData.ShowAverage = false
			}

			if utils.MissingOrFalse(activeSkill.SkillTypes, data.SkillTypeChannel) {
				pass.Output["Speed"] = math.Min(pass.Output["Speed"], data.ServerTickRate*pass.Output["Repeats"])
			}

			if pass.Output["Speed"] == 0 {
				pass.Output["Time"] = 0
			} else {
				pass.Output["Time"] = 1 / pass.Output["Speed"]
			}

			if breakdown != nil {
				breakdown.MultiChain("Speed", BMultiChain{
					Base:  &BMultiChainItem{"%.2f ^8(base)", 1 / baseTime},
					Total: fmt.Sprintf("= %.2f ^8casts per second", actor.Output["CastRate"]),
					Items: []BMultiChainItem{
						{"%.2f ^8(increased/reduced)", 1 + inc/100},
						{"%.2f ^8(more/less)", more},
						{"%.2f ^8(action speed modifier)", utils.Ternary(skillFlags[SkillFlagSelfCast], actor.Output["ActionSpeedMod"], float64(1))},
					},
				})
				if actor.Output["Cooldown"] != 0 && (1/actor.Output["Cooldown"]) < actor.Output["CastRate"] {
					breakdown.AddLine("Speed", "\n")
					breakdown.AddLine("Speed", fmt.Sprintf("1 / %.2f ^8(skill cooldown)", actor.Output["Cooldown"]))
					if actor.Output["Repeats"] > 1 {
						breakdown.AddLine("Speed", fmt.Sprintf("x %.2f ^8(repeat count)", actor.Output["Repeats"]))
					}
					breakdown.AddLine("Speed", fmt.Sprintf("= %.2f ^8(casts per second)", actor.Output["Repeats"]/actor.Output["Cooldown"]))
					breakdown.AddLine("Speed", "\n")
					breakdown.AddLine("Speed", fmt.Sprintf("= %.2f ^8(lower of cast rates)", actor.Output["Speed"]))
				}
			}
			if breakdown != nil && calclib.Mod(skillModList, skillCfg, "SkillAttackTime") > 0 {
				breakdown.MultiChain("Time", BMultiChain{
					Base:  &BMultiChainItem{"%.2f ^8(base)", 1 / (actor.Output["Speed"] * calclib.Mod(skillModList, skillCfg, "SkillAttackTime"))},
					Total: fmt.Sprintf("= %.2f ^8seconds per attack", actor.Output["Time"]),
					Items: []BMultiChainItem{
						{"%.2f ^8(total modifier)", calclib.Mod(skillModList, skillCfg, "SkillAttackTime")},
					},
				})
			}
		}
		if skillData.HitTimeOverride != 0 && !skillData.TriggeredOnDeath {
			pass.Output["HitTime"] = skillData.HitTimeOverride
			pass.Output["HitSpeed"] = 1 / pass.Output["HitTime"]
			// Brands always have hitTimeOverride
			if activeSkill.SkillTypes[data.SkillTypeBrand] && !skillModList.Flag(nil, "UnlimitedBrandDuration") {
				pass.Output["BrandTicks"] = math.Floor(pass.Output["Duration"] * pass.Output["HitSpeed"])
			}
		} else if skillData.HitTimeMultiplier != 0 && utils.Has(pass.Output, "Time") && !skillData.TriggeredOnDeath {
			pass.Output["HitTime"] = pass.Output["Time"] * skillData.HitTimeMultiplier
			pass.Output["HitSpeed"] = 1 / pass.Output["HitTime"]
		}
	}

	//pob:block:src/Modules/CalcOffence.lua:2436
	if skillModList.Flag(nil, "HasTrauma") {
		actor.Output["SustainableTrauma"] = skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:SustainableTraumaStacks")
	}

	//pob:block:src/Modules/CalcOffence.lua:2437-2439
	modDB.Multipliers["BuffOnSelf"] += skillModList.Sum(mod.TypeBase, nil, "Multiplier:TraumaStacks")
	modDB.Multipliers["BuffOnSelf"] += skillModList.Sum(mod.TypeBase, nil, "Multiplier:VoltaxicWaitingStages")

	if utils.HasTrue(skillFlags, SkillFlagAttack) {
		// Combine hit chance and attack speed
		//pob:block:src/Modules/CalcOffence.lua:1893-1897
		combineStat("AccuracyHitChance", ModeAverage)
		combineStat("HitChance", ModeAverage)
		combineStat("Speed", ModeHarmonicMean)
		combineStat("HitSpeed", ModeOr)
		combineStat("HitTime", ModeOr)

		if actor.Output["Speed"] == 0 {
			actor.Output["Time"] = 0
		} else {
			actor.Output["Time"] = 1 / actor.Output["Speed"]
		}

		if actor.Output["Time"] > 1 {
			modDB.AddMod(mod.NewFlag("Condition:OneSecondAttackTime", true))
		}

		if skillModList.Flag(nil, "UseOffhandAttackSpeed") && !utils.HasTrue(skillFlags, SkillFlagForceMainHand) {
			actor.Output["Speed"] = actor.OutputTable[OutTableOffHand]["Speed"]
			actor.Output["Time"] = actor.OutputTable[OutTableOffHand]["Time"]
			if breakdown != nil {
				breakdown.AddLine("Speed",
					"Use Offhand Weapon Attack Speed:",
					fmt.Sprintf("= %.2f", actor.Output["Speed"]),
				)
			}
		} else if utils.HasTrue(skillFlags, SkillFlagBothWeaponAttack) {
			if breakdown != nil {
				breakdown.AddLine("Speed",
					"Both weapons:",
					fmt.Sprintf("(%.2f + %.2f) / 2", actor.OutputTable[OutTableMainHand]["Speed"], actor.OutputTable[OutTableOffHand]["Speed"]),
					fmt.Sprintf("= %.2f", actor.Output["Speed"]),
				)
			}
		}
		//pob:block:src/Modules/CalcOffence.lua:2474-2486
		if skillData.HitTimeOverride != 0 && !skillData.TriggeredOnDeath {
			actor.Output["HitTime"] = skillData.HitTimeOverride
			actor.Output["HitSpeed"] = 1 / actor.Output["HitTime"]
		} else if skillData.HitTimeMultiplier != 0 && utils.Has(actor.Output, "Time") && !skillData.TriggeredOnDeath {
			actor.Output["HitTime"] = actor.Output["Time"] * skillData.HitTimeMultiplier
			switch {
			case actor.Output["Cooldown"] != 0 && skillData.Triggered:
				actor.Output["HitSpeed"] = 1 / max(actor.Output["HitTime"], actor.Output["Cooldown"])
			case actor.Output["Cooldown"] != 0:
				actor.Output["HitSpeed"] = 1 / (actor.Output["HitTime"] + actor.Output["Cooldown"])
			default:
				actor.Output["HitSpeed"] = min(1/actor.Output["HitTime"], data.ServerTickRate)
			}
		}
	}

	// Other Misc DPS multipliers (like custom source)
	//pob:block:src/Modules/CalcMirages.lua:391-394
	if generalsCryCooldown != 0 {
		skillTime := actor.Output["Time"]
		if hitTime, ok := actor.Output["HitTime"]; ok && activeSkill.SkillTypes[data.SkillTypeChannel] {
			skillTime = hitTime
		}
		cooldown := max(generalsCryCooldown, skillTime)
		skillModList.AddMod(mod.NewFloat("DPS", mod.TypeMore, (1/cooldown-1)*100).Source("General's Cry Cooldown"))
	}

	//pob:block:src/Modules/CalcOffence.lua:2421-2422
	dpsMultiplier := skillData.DpsMultiplier
	if dpsMultiplier == 0 {
		dpsMultiplier = 1
	}
	skillData.DpsMultiplier = dpsMultiplier *
		(1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "DPS")/100) *
		skillModList.More(skillCfg, "DPS")

	//pob:block:src/Modules/CalcOffence.lua:2423-2426
	if activeSkill.SkillTypes[data.SkillTypeBrand] && !skillData.CountsAttachedBrandsInDamage {
		skillData.DpsMultiplier *= actor.Output["AttachedBrandCount"]
		skillDPSMultiplier := actor.Output["SkillDPSMultiplier"]
		if skillDPSMultiplier == 0 {
			skillDPSMultiplier = 1
		}
		actor.Output["SkillDPSMultiplier"] = skillDPSMultiplier * actor.Output["AttachedBrandCount"]
	}

	//pob:block:src/Modules/CalcOffence.lua:2427-2429
	if repeatMode == "FINAL" || skillModList.Flag(nil, "OnlyFinalRepeat") {
		skillData.DpsMultiplier /= utils.OrDefault(actor.Output["Repeats"], 1)
	}

	quantityMultiplier := math.Max(skillModList.Sum(mod.TypeBase, skillCfg, "QuantityMultiplier"), 1)
	if quantityMultiplier > 1 {
		actor.Output["QuantityMultiplier"] = quantityMultiplier
	}

	ancestralCryCalculated := false
	infernalCryCalculated := false
	intimidatingCryCalculated := false
	rallyingCryCalculated := false
	seismicCryCalculated := false
	battleMageCryCalculated := false
	exertedAttackUptimeRatioCalculated := false

	//pob:src/Modules/CalcOffence.lua:1691-1966
	for _, pass := range passList {
		globalOutput := actor.Output
		output, cfg := pass.Output, pass.Config

		// Exerted Attack members
		exertedDoubleDamage := env.ModDB.Sum(mod.TypeBase, cfg, "ExertDoubleDamageChance")
		globalOutput["OffensiveWarcryEffect"] = 1
		globalOutput["MaxOffensiveWarcryEffect"] = 1
		globalOutput["TheoreticalOffensiveWarcryEffect"] = 1
		globalOutput["TheoreticalMaxOffensiveWarcryEffect"] = 1
		globalOutput["RallyingHitEffect"] = 1
		globalOutput["AilmentWarcryEffect"] = 1
		globalOutput["GlobalWarcryUptimeRatio"] = 0

		if env.ModeBuffs {
			// Iterative over all the active skills to account for exerted attacks provided by warcries
			//pob:block:src/Modules/CalcOffence.lua:2541-2542
			if !activeSkill.SkillTypes[data.SkillTypeNeverExertable] &&
				!activeSkill.SkillTypes[data.SkillTypeChannel] &&
				!activeSkill.SkillTypes[data.SkillTypeTriggered] &&
				!activeSkill.SkillTypes[data.SkillTypeOtherThingUsesSkill] &&
				!activeSkill.SkillTypes[data.SkillTypeRetaliation] &&
				!activeSkill.SkillTypes[data.SkillTypeSummonsTotem] {
				for _, value := range actor.ActiveSkillList {
					valueName := ""
					if as := value.ActiveEffect.GrantedEffect.Raw.GetActiveSkill(); as != nil {
						valueName = as.DisplayedName
					}
					if valueName == "Ancestral Cry" && activeSkill.SkillTypes[data.SkillTypeMeleeSingleTarget] && !ancestralCryCalculated {
						globalOutput["AncestralCryDuration"] = calcSkillDuration(value.SkillModList, value.SkillCfg, value.SkillData, env, enemyDB)
						globalOutput["AncestralCryCooldown"] = calcSkillCooldown(value.SkillModList, value.SkillCfg, value.SkillData)
						output["GlobalWarcryCooldown"] = env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
						output["GlobalWarcryCount"] = env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
						if modDB.Flag(nil, "WarcryShareCooldown") {
							globalOutput["AncestralCryCooldown"] = globalOutput["AncestralCryCooldown"] + (output["GlobalWarcryCooldown"]-globalOutput["AncestralCryCooldown"])/output["GlobalWarcryCount"]
						}
						globalOutput["AncestralCryCastTime"] = calcWarcryCastTime(value.SkillModList, value.SkillCfg, value.SkillData, actor)
						globalOutput["AncestralExertsCount"] = env.ModDB.Sum(mod.TypeBase, nil, "NumAncestralExerts")
						baseUptimeRatio := min((globalOutput["AncestralExertsCount"]/globalOutput["Speed"])/(globalOutput["AncestralCryCooldown"]+globalOutput["AncestralCryCastTime"]), 1) * 100
						storedUses := value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "AdditionalCooldownUses")
						if value.SkillData != nil && value.SkillData.StoredUses != nil {
							storedUses += *value.SkillData.StoredUses
						}
						globalOutput["AncestralUpTimeRatio"] = min(100, baseUptimeRatio*storedUses)
						globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["AncestralUpTimeRatio"]
						ancestralCryCalculated = true
					} else if valueName == "Infernal Cry" && !infernalCryCalculated {
						globalOutput["InfernalCryDuration"] = calcSkillDuration(value.SkillModList, value.SkillCfg, value.SkillData, env, enemyDB)
						globalOutput["InfernalCryCooldown"] = calcSkillCooldown(value.SkillModList, value.SkillCfg, value.SkillData)
						output["GlobalWarcryCooldown"] = env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
						output["GlobalWarcryCount"] = env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
						if modDB.Flag(nil, "WarcryShareCooldown") {
							globalOutput["InfernalCryCooldown"] = globalOutput["InfernalCryCooldown"] + (output["GlobalWarcryCooldown"]-globalOutput["InfernalCryCooldown"])/output["GlobalWarcryCount"]
						}
						globalOutput["InfernalCryCastTime"] = calcWarcryCastTime(value.SkillModList, value.SkillCfg, value.SkillData, actor)
						if activeSkill.SkillTypes[data.SkillTypeMelee] {
							globalOutput["InfernalExertsCount"] = env.ModDB.Sum(mod.TypeBase, nil, "NumInfernalExerts")
							baseUptimeRatio := min((globalOutput["InfernalExertsCount"]/globalOutput["Speed"])/(globalOutput["InfernalCryCooldown"]+globalOutput["InfernalCryCastTime"]), 1) * 100
							storedUses := value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "AdditionalCooldownUses")
							if value.SkillData != nil && value.SkillData.StoredUses != nil {
								storedUses += *value.SkillData.StoredUses
							}
							globalOutput["InfernalUpTimeRatio"] = min(100, baseUptimeRatio*storedUses)
							globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["InfernalUpTimeRatio"]
						}
						infernalCryCalculated = true
					} else if valueName == "Intimidating Cry" && activeSkill.SkillTypes[data.SkillTypeMelee] && !intimidatingCryCalculated {
						globalOutput["IntimidatingCryDuration"] = calcSkillDuration(value.SkillModList, value.SkillCfg, value.SkillData, env, enemyDB)
						globalOutput["IntimidatingCryCooldown"] = calcSkillCooldown(value.SkillModList, value.SkillCfg, value.SkillData)
						output["GlobalWarcryCooldown"] = env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
						output["GlobalWarcryCount"] = env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
						if modDB.Flag(nil, "WarcryShareCooldown") {
							globalOutput["IntimidatingCryCooldown"] = globalOutput["IntimidatingCryCooldown"] + (output["GlobalWarcryCooldown"]-globalOutput["IntimidatingCryCooldown"])/output["GlobalWarcryCount"]
						}
						globalOutput["IntimidatingCryCastTime"] = calcWarcryCastTime(value.SkillModList, value.SkillCfg, value.SkillData, actor)
						globalOutput["IntimidatingExertsCount"] = env.ModDB.Sum(mod.TypeBase, nil, "NumIntimidatingExerts")
						baseUptime := min((globalOutput["IntimidatingExertsCount"]/globalOutput["Speed"])/(globalOutput["IntimidatingCryCooldown"]+globalOutput["IntimidatingCryCastTime"]), 1) * 100
						storedUses := value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "AdditionalCooldownUses")
						if value.SkillData != nil && value.SkillData.StoredUses != nil {
							storedUses += *value.SkillData.StoredUses
						}
						globalOutput["IntimidatingUpTimeRatio"] = min(100, baseUptime*storedUses)
						globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["IntimidatingUpTimeRatio"]
						ddChance := min(skillModList.Sum(mod.TypeBase, cfg, "DoubleDamageChance")+utils.Ternary(env.ModeEffective, enemyDB.Sum(mod.TypeBase, cfg, "SelfDoubleDamageChance"), 0)+exertedDoubleDamage, 100)
						globalOutput["IntimidatingAvgDmg"] = 2 * (1 - ddChance/100)
						globalOutput["IntimidatingHitEffect"] = 1 + globalOutput["IntimidatingAvgDmg"]*globalOutput["IntimidatingUpTimeRatio"]/100
						globalOutput["IntimidatingMaxHitEffect"] = 1 + globalOutput["IntimidatingAvgDmg"]
						globalOutput["TheoreticalOffensiveWarcryEffect"] = globalOutput["TheoreticalOffensiveWarcryEffect"] * globalOutput["IntimidatingHitEffect"]
						globalOutput["TheoreticalMaxOffensiveWarcryEffect"] = globalOutput["TheoreticalMaxOffensiveWarcryEffect"] * globalOutput["IntimidatingMaxHitEffect"]
						intimidatingCryCalculated = true
					} else if valueName == "Rallying Cry" && activeSkill.SkillTypes[data.SkillTypeMelee] && !rallyingCryCalculated {
						globalOutput["RallyingCryDuration"] = calcSkillDuration(value.SkillModList, value.SkillCfg, value.SkillData, env, enemyDB)
						globalOutput["RallyingCryCooldown"] = calcSkillCooldown(value.SkillModList, value.SkillCfg, value.SkillData)
						output["GlobalWarcryCooldown"] = env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
						output["GlobalWarcryCount"] = env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
						if modDB.Flag(nil, "WarcryShareCooldown") {
							globalOutput["RallyingCryCooldown"] = globalOutput["RallyingCryCooldown"] + (output["GlobalWarcryCooldown"]-globalOutput["RallyingCryCooldown"])/output["GlobalWarcryCount"]
						}
						globalOutput["RallyingCryCastTime"] = calcWarcryCastTime(value.SkillModList, value.SkillCfg, value.SkillData, actor)
						globalOutput["RallyingExertsCount"] = env.ModDB.Sum(mod.TypeBase, nil, "NumRallyingExerts")
						baseUptimeRatio := min((globalOutput["RallyingExertsCount"]/globalOutput["Speed"])/(globalOutput["RallyingCryCooldown"]+globalOutput["RallyingCryCastTime"]), 1) * 100
						storedUses := value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "AdditionalCooldownUses")
						if value.SkillData != nil && value.SkillData.StoredUses != nil {
							storedUses += *value.SkillData.StoredUses
						}
						globalOutput["RallyingUpTimeRatio"] = min(100, baseUptimeRatio*storedUses)
						globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["RallyingUpTimeRatio"]
						globalOutput["RallyingAvgDmg"] = min(env.ModDB.Sum(mod.TypeBase, cfg, "Multiplier:NearbyAlly"), 5) * (env.ModDB.Sum(mod.TypeBase, nil, "RallyingExertMoreDamagePerAlly") / 100)
						globalOutput["RallyingHitEffect"] = 1 + globalOutput["RallyingAvgDmg"]*globalOutput["RallyingUpTimeRatio"]/100
						globalOutput["RallyingMaxHitEffect"] = 1 + globalOutput["RallyingAvgDmg"]
						globalOutput["OffensiveWarcryEffect"] = globalOutput["OffensiveWarcryEffect"] * globalOutput["RallyingHitEffect"]
						globalOutput["MaxOffensiveWarcryEffect"] = globalOutput["MaxOffensiveWarcryEffect"] * globalOutput["RallyingMaxHitEffect"]
						globalOutput["TheoreticalOffensiveWarcryEffect"] = globalOutput["TheoreticalOffensiveWarcryEffect"] * globalOutput["RallyingHitEffect"]
						globalOutput["TheoreticalMaxOffensiveWarcryEffect"] = globalOutput["TheoreticalMaxOffensiveWarcryEffect"] * globalOutput["RallyingMaxHitEffect"]
						rallyingCryCalculated = true
					} else if valueName == "Seismic Cry" && activeSkill.SkillTypes[data.SkillTypeSlam] && !seismicCryCalculated {
						globalOutput["SeismicCryDuration"] = calcSkillDuration(value.SkillModList, value.SkillCfg, value.SkillData, env, enemyDB)
						globalOutput["SeismicCryCooldown"] = calcSkillCooldown(value.SkillModList, value.SkillCfg, value.SkillData)
						output["GlobalWarcryCooldown"] = env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
						output["GlobalWarcryCount"] = env.ModDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
						if modDB.Flag(nil, "WarcryShareCooldown") {
							globalOutput["SeismicCryCooldown"] = globalOutput["SeismicCryCooldown"] + (output["GlobalWarcryCooldown"]-globalOutput["SeismicCryCooldown"])/output["GlobalWarcryCount"]
						}
						globalOutput["SeismicCryCastTime"] = calcWarcryCastTime(value.SkillModList, value.SkillCfg, value.SkillData, actor)
						globalOutput["SeismicExertsCount"] = env.ModDB.Sum(mod.TypeBase, nil, "NumSeismicExerts")
						baseUptimeRatio := min((globalOutput["SeismicExertsCount"]/globalOutput["Speed"])/(globalOutput["SeismicCryCooldown"]+globalOutput["SeismicCryCastTime"]), 1) * 100
						storedUses := float64(0)
						if value.SkillData.StoredUses != nil {
							storedUses = *value.SkillData.StoredUses
						}
						storedUses += value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "AdditionalCooldownUses")
						globalOutput["SeismicUpTimeRatio"] = min(100, baseUptimeRatio*storedUses)
						globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["SeismicUpTimeRatio"]

						if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
							skillModList.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeMore, env.ModDB.Sum(mod.TypeBase, nil, "SeismicMoreAoE")).Source("Max Seismic Exert AoE"))
						} else {
							skillModList.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeMore, math.Floor(env.ModDB.Sum(mod.TypeBase, nil, "SeismicMoreAoE")/100*globalOutput["SeismicUpTimeRatio"])).Source("Avg Seismic Exert AoE"))
						}
						calcAreaOfEffect(skillModList, skillCfg, skillData, skillFlags, globalOutput, actor.Breakdown)
						seismicCryCalculated = true
					} else if valueName == "Battlemage's Cry" && !battleMageCryCalculated && !value.SkillFlags[SkillFlagDisable] {
						globalOutput["BattleMageCryDuration"] = calcSkillDuration(value.SkillModList, value.SkillCfg, value.SkillData, env, enemyDB)
						globalOutput["BattleMageCryCooldown"] = calcSkillCooldown(value.SkillModList, value.SkillCfg, value.SkillData)
						globalOutput["BattleMageCryCastTime"] = calcWarcryCastTime(value.SkillModList, value.SkillCfg, value.SkillData, actor)
						if activeSkill.SkillTypes[data.SkillTypeMelee] {
							globalOutput["BattleCryExertsCount"] = env.ModDB.Sum(mod.TypeBase, nil, "NumBattlemageExerts")
							baseUptimeRatio := min((globalOutput["BattleCryExertsCount"]/globalOutput["Speed"])/(globalOutput["BattleMageCryCooldown"]+globalOutput["BattleMageCryCastTime"]), 1) * 100
							storedUses := float64(0)
							if value.SkillData.StoredUses != nil {
								storedUses = *value.SkillData.StoredUses
							}
							storedUses += value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "AdditionalCooldownUses")
							globalOutput["BattlemageUpTimeRatio"] = min(100, baseUptimeRatio*storedUses)
							globalOutput["GlobalWarcryUptimeRatio"] += globalOutput["BattlemageUpTimeRatio"]
						}
						battleMageCryCalculated = true
					}
				}

				if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
					globalOutput["AilmentWarcryEffect"] = globalOutput["MaxOffensiveWarcryEffect"]
					skillData.ShowAverage = true
					skillFlags[SkillFlagShowAverage] = true
					skillFlags[SkillFlagNotAverage] = false
				} else {
					globalOutput["AilmentWarcryEffect"] = globalOutput["OffensiveWarcryEffect"]
				}

				// Calculate Exerted Attack Uptime
				// There are various strategies a player could use to maximize either warcry effect stacking or staggering
				// 1) they don't pay attention and therefore we calculated exerted attack uptime as just the maximum uptime of any enabled warcries that exert attacks
				globalOutput["ExertedAttackUptimeRatio"] = max(max(max(globalOutput["AncestralUpTimeRatio"], globalOutput["InfernalUpTimeRatio"]), max(globalOutput["IntimidatingUpTimeRatio"], globalOutput["RallyingUpTimeRatio"])), max(globalOutput["SeismicUpTimeRatio"], globalOutput["BattlemageUpTimeRatio"]))
				if globalOutput["ExertedAttackUptimeRatio"] > 0 && !exertedAttackUptimeRatioCalculated {
					incExertedAttacks := skillModList.Sum(mod.TypeIncrease, cfg, "ExertIncrease")
					moreExertedAttacks := skillModList.Sum(mod.TypeMore, cfg, "ExertIncrease")
					moreExertedAttackDamage := skillModList.Sum(mod.TypeMore, cfg, "ExertAttackIncrease")
					overexertionExertedDamage := skillModList.Sum(mod.TypeMore, cfg, "OverexertionExertAverageIncrease")
					echoesOfCreationExertedDamage := skillModList.Sum(mod.TypeMore, cfg, "EchoesExertAverageIncrease")
					if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
						skillModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, incExertedAttacks).Source("Exerted Attacks"))
						skillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, moreExertedAttacks).Source("Exerted Attacks"))
						skillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, moreExertedAttackDamage).Source("Exerted Attack Damage").Flag(mod.MFlagAttack))
						exertingWarcryCount := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:ExertingWarcryCount")
						skillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, overexertionExertedDamage*exertingWarcryCount).Source("Max Autoexertion Support"))
						skillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, echoesOfCreationExertedDamage*exertingWarcryCount).Source("Max Echoes of Creation"))
					} else {
						skillModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, incExertedAttacks*globalOutput["ExertedAttackUptimeRatio"]/100).Source("Uptime Scaled Exerted Attacks"))
						skillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, moreExertedAttacks*globalOutput["ExertedAttackUptimeRatio"]/100).Source("Uptime Scaled Exerted Attacks"))
						skillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, moreExertedAttackDamage*globalOutput["ExertedAttackUptimeRatio"]/100).Source("Uptime Scaled Exerted Attack Damage").Flag(mod.MFlagAttack))
						skillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, overexertionExertedDamage*globalOutput["GlobalWarcryUptimeRatio"]/100).Source("Uptime Scaled Autoexertion Support"))
						skillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, echoesOfCreationExertedDamage*globalOutput["GlobalWarcryUptimeRatio"]/100).Source("Uptime Scaled Echoes of Creation"))
					}
					globalOutput["ExertedAttackAvgDmg"] = calclib.Mod(skillModList, skillCfg, "ExertIncrease")
					globalOutput["ExertedAttackAvgDmg"] = globalOutput["ExertedAttackAvgDmg"] * calclib.Mod(skillModList, skillCfg, "ExertAttackIncrease", "OverexertionExertAverageIncrease", "EchoesExertAverageIncrease")
					globalOutput["ExertedAttackHitEffect"] = globalOutput["ExertedAttackAvgDmg"] * globalOutput["ExertedAttackUptimeRatio"] / 100
					globalOutput["ExertedAttackMaxHitEffect"] = globalOutput["ExertedAttackAvgDmg"]
					exertedAttackUptimeRatioCalculated = true
				}
			}
		}

		if activeSkill.SkillTypes[data.SkillTypeSpell] && !activeSkill.SkillTypes[data.SkillTypeBrand] {
			for _, value := range actor.ActiveSkillList {
				pactKey, isPact := strings.CutPrefix(value.ActiveEffect.GrantedEffect.Name(), "Pact of ")
				if !isPact {
					continue
				}
				if pactKey == "K'Tash" {
					pactKey = "Ktash"
				}
				isVaal := activeSkill.SkillTypes[data.SkillTypeVaal]
				pactApplies := false
				switch pactKey {
				case "Beidat":
					pactApplies = !isVaal && !activeSkill.SkillTypes[data.SkillTypeChannel] &&
						(skillFlags[SkillFlagProjectile] || activeSkill.SkillTypes[data.SkillTypeCascadable] ||
							(skillFlags[SkillFlagChaining] && !skillFlags[SkillFlagProjectile]))
				case "Ghorr":
					pactApplies = !isVaal && activeSkill.SkillTypes[data.SkillTypeDamageOverTime]
				case "Ktash":
					pactApplies = isVaal && (activeSkill.SkillTypes[data.SkillTypeDamage] ||
						activeSkill.SkillTypes[data.SkillTypeDamageOverTime])
				case "Lycia":
					pactApplies = !isVaal && activeSkill.SkillTypes[data.SkillTypeChannel] && skillFlags[SkillFlagSelfCast]
				}
				calculated := "PactOf" + pactKey + "Calculated"
				if !pactApplies || globalOutput[calculated] != 0 || value.SkillFlags[SkillFlagDisable] {
					continue
				}

				cooldown := calcSkillCooldown(value.SkillModList, value.SkillCfg, value.SkillData)
				castRate := 1 / value.ActiveEffect.GrantedEffect.CastTime() *
					calclib.Mod(value.SkillModList, value.SkillCfg, "Speed") * CalcActionSpeedMod(actor)
				castTime := 1 / min(castRate, data.ServerTickRate)
				count := value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, pactKey+"EmpoweredSpells")
				storedUses := value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "AdditionalCooldownUses")
				if value.SkillData.StoredUses != nil {
					storedUses += *value.SkillData.StoredUses
				}
				uptime := float64(100)
				if globalOutput["Speed"] != 0 {
					uptime = min((count/globalOutput["Speed"])/(cooldown+castTime), 1) * 100
				}
				uptime = min(100, uptime*storedUses)
				effect := uptime
				if skillModList.Flag(nil, "Condition:PactMaxHit") {
					effect = 100
				}
				effectMult := effect / 100

				globalOutput["CreatePactOffensiveCalcSection"] = 1
				globalOutput["PactOf"+pactKey+"Cooldown"] = cooldown
				globalOutput["PactOf"+pactKey+"CastTime"] = castTime
				globalOutput[pactKey+"EmpoweredCount"] = count
				globalOutput[pactKey+"UpTimeRatio"] = uptime
				if actor.Breakdown != nil {
					actor.Breakdown.AddLine(pactKey+"UpTimeRatio",
						fmt.Sprintf("(%.2f ^8(number of Empowered)", count),
						fmt.Sprintf("/ %.2f) ^8(casts per second)", globalOutput["Speed"]),
						fmt.Sprintf("/ (%.2f ^8(pact cooldown)", cooldown),
						fmt.Sprintf("+ %.2f) ^8(pact cast time)", castTime),
						fmt.Sprintf("* %.2f ^8(stored uses)", storedUses),
						fmt.Sprintf("= %.2f%%", uptime),
					)
				}

				if payloads := value.SkillModList.List(value.SkillCfg, pactKey+"PactDamage"); len(payloads) > 0 {
					if damage, ok := payloads[0].(mod.PactDamage); ok && damage.Mod != nil {
						scaled := damage.Mod.Clone()
						if v := scaled.Value(); v != nil && v.Type() == mod.ModValueMultiTypeFloat {
							v.SetFloat(v.Float() * effectMult)
						}
						skillModList.AddMod(scaled.Source(mod.Source("Uptime Scaled " + value.ActiveEffect.GrantedEffect.Name())))
					}
				}

				switch pactKey {
				case "Beidat":
					if skillFlags[SkillFlagProjectile] && !skillModList.Flag(skillCfg, "NoAdditionalProjectiles") &&
						!skillModList.Flag(skillCfg, "SingleProjectile") {
						globalOutput["BeidatAdditionalProjectiles"] = value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "BeidatAdditionalProjectiles") * effectMult
						globalOutput["ProjectileCount"] = globalOutput["ProjectileCount"] + globalOutput["BeidatAdditionalProjectiles"]
					}
					if skillFlags[SkillFlagChaining] && !skillFlags[SkillFlagProjectile] &&
						!skillModList.Flag(skillCfg, "CannotChain") && !skillModList.Flag(skillCfg, "NoAdditionalChains") {
						globalOutput["BeidatAdditionalBeamChains"] = value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "BeidatAdditionalBeamChains") * effectMult
						globalOutput["ChainMax"] = globalOutput["ChainMax"] + globalOutput["BeidatAdditionalBeamChains"]
						globalOutput["ChainMaxString"] = globalOutput["ChainMax"]
						actor.OutputStrings["ChainMaxString"] = fmt.Sprint(globalOutput["ChainMax"])
						globalOutput["ChainRemaining"] = max(0, globalOutput["ChainMax"]-globalOutput["Chain"])
					}
					if activeSkill.SkillTypes[data.SkillTypeCascadable] {
						globalOutput["BeidatAdditionalCascades"] = value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "BeidatAdditionalCascades") * effectMult
					}
				case "Ktash":
					globalOutput["KtashSoulRefundChance"] = value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "KtashPactSoulRefundChance") * effectMult
					if globalOutput["SoulGainPreventionDuration"] != 0 {
						duration := globalOutput["SoulGainPreventionDuration"]
						durationMod := 1 + value.SkillModList.Sum(mod.TypeBase, value.SkillCfg, "KtashPactSoulGainPrevention")*effectMult/100
						globalOutput["SoulGainPreventionDuration"] = max(math.Ceil(duration*durationMod*data.ServerTickRate), 1) / data.ServerTickRate
					}
				}
				globalOutput[calculated] = 1
			}
		}

		pass.Output["RuthlessBlowHitEffect"] = 1
		pass.Output["RuthlessBlowAilmentEffect"] = 1
		pass.Output["FistOfWarDamageEffect"] = 1

		if env.ModeCombat {
			// Calculate Ruthless Blow chance/multipliers + Fist of War multipliers
			pass.Output["RuthlessBlowMaxCount"] = skillModList.Sum(mod.TypeBase, pass.Config, "RuthlessBlowMaxCount")
			if pass.Output["RuthlessBlowMaxCount"] > 0 {
				pass.Output["RuthlessBlowChance"] = utils.RoundTo(100/pass.Output["RuthlessBlowMaxCount"], 0)
			} else {
				pass.Output["RuthlessBlowChance"] = 0
			}
			pass.Output["RuthlessBlowHitMultiplier"] = 1 + skillModList.Sum(mod.TypeBase, pass.Config, "RuthlessBlowHitMultiplier")/100
			pass.Output["RuthlessBlowAilmentMultiplier"] = 1 + skillModList.Sum(mod.TypeBase, pass.Config, "RuthlessBlowAilmentMultiplier")/100
			pass.Output["RuthlessBlowHitEffect"] = 1 - pass.Output["RuthlessBlowChance"]/100 + pass.Output["RuthlessBlowChance"]/100*pass.Output["RuthlessBlowHitMultiplier"]
			pass.Output["RuthlessBlowAilmentEffect"] = 1 - pass.Output["RuthlessBlowChance"]/100 + pass.Output["RuthlessBlowChance"]/100*pass.Output["RuthlessBlowAilmentMultiplier"]

			actor.Output["FistOfWarCooldown"] = skillModList.Sum(mod.TypeBase, pass.Config, "FistOfWarCooldown")
			// If Fist of War & Active Skill is a Slam Skill & NOT a Vaal Skill & NOT used by mirage or other
			if actor.Output["FistOfWarCooldown"] != 0 && activeSkill.SkillTypes[data.SkillTypeSlam] && !activeSkill.SkillTypes[data.SkillTypeVaal] && !activeSkill.SkillTypes[data.SkillTypeOtherThingUsesSkill] {
				actor.Output["FistOfWarDamageMultiplier"] = skillModList.Sum(mod.TypeBase, nil, "FistOfWarDamageMultiplier") / 100
				actor.Output["FistOfWarUptimeRatio"] = min((1/pass.Output["Speed"])/actor.Output["FistOfWarCooldown"], 1) * 100
				if actor.Breakdown != nil {
					actor.Breakdown.AddLine("FistOfWarUptimeRatio",
						fmt.Sprintf("min( (1 / %.2f) ^8(second per attack)", pass.Output["Speed"]),
						fmt.Sprintf("/ %.2f, 1) ^8(fist of war cooldown)", actor.Output["FistOfWarCooldown"]),
						fmt.Sprintf("= %.2f%%", actor.Output["FistOfWarUptimeRatio"]),
					)
				}
				actor.Output["AvgFistOfWarDamage"] = actor.Output["FistOfWarDamageMultiplier"]
				actor.Output["AvgFistOfWarDamageEffect"] = 1 + actor.Output["FistOfWarDamageMultiplier"]*(actor.Output["FistOfWarUptimeRatio"]/100)
				if actor.Breakdown != nil {
					actor.Breakdown.AddLine("AvgFistOfWarDamageEffect",
						fmt.Sprintf("1 + (%.2f ^8(fist of war damage multiplier)", actor.Output["FistOfWarDamageMultiplier"]),
						fmt.Sprintf("x %.2f) ^8(fist of war uptime ratio)", actor.Output["FistOfWarUptimeRatio"]/100),
						fmt.Sprintf("= %.2f", actor.Output["AvgFistOfWarDamageEffect"]),
					)
				}
				actor.Output["MaxFistOfWarDamageEffect"] = 1 + actor.Output["FistOfWarDamageMultiplier"]
				if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
					pass.Output["FistOfWarDamageEffect"] = actor.Output["MaxFistOfWarDamageEffect"]
					skillModList.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeIncrease,
						skillModList.Sum(mod.TypeBase, nil, "FistOfWarIncAoE")).Source("Max Fist of War Boosted AoE"))
				} else {
					pass.Output["FistOfWarDamageEffect"] = actor.Output["AvgFistOfWarDamageEffect"]
					skillModList.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeIncrease,
						math.Floor(skillModList.Sum(mod.TypeBase, nil, "FistOfWarIncAoE")/100*actor.Output["FistOfWarUptimeRatio"])).Source("Avg Fist Of War Boosted AoE"))
				}
				calcAreaOfEffect(skillModList, activeSkill.SkillCfg, activeSkill.SkillData, activeSkill.SkillFlags, actor.Output, actor.Breakdown)
				actor.Output["TheoreticalOffensiveWarcryEffect"] = actor.Output["TheoreticalOffensiveWarcryEffect"] * actor.Output["AvgFistOfWarDamageEffect"]
				actor.Output["TheoreticalMaxOffensiveWarcryEffect"] = actor.Output["TheoreticalMaxOffensiveWarcryEffect"] * actor.Output["MaxFistOfWarDamageEffect"]
			} else {
				pass.Output["FistOfWarDamageEffect"] = 1
			}
		}

		// Calculate maximum sustainable fuses and explosion rate for Explosive Arrow.
		// Does not take into account mines or traps.
		//pob:block:src/Modules/CalcOffence.lua:2948-2952
		if activeSkill.ActiveEffect.GrantedEffect.Name() == "Explosive Arrow" {
			explosiveArrowFunc(activeSkill, pass.Output, actor.Output, env, enemyDB)
		}

		// Calculate crit chance, crit multiplier, and their combined effect
		if skillModList.Flag(nil, "NeverCrit") {
			pass.Output["PreEffectiveCritChance"] = 0
			pass.Output["CritChance"] = 0
			pass.Output["CritMultiplier"] = 0
			pass.Output["BonusCritDotMultiplier"] = 0
			pass.Output["CritEffect"] = 1
		} else {
			baseCrit := float64(0)

			if pass.Source.CritChance != 0 {
				baseCrit = pass.Source.CritChance
			}

			critOverride := skillModList.Override(pass.Config, "CritChance")
			hasCritOverride := critOverride != nil
			if hasCritOverride {
				baseCrit = critOverride.Float()
			}

			//pob:block:src/Modules/CalcOffence.lua:2991-2992
			if skillModList.Flag(pass.Config, "MainHandCritIsEqualToParent") && actor.Parent != nil {
				if mainHand, ok := actor.Parent.OutputTable[OutTableMainHand]; ok && utils.Has(mainHand, "CritChance") {
					baseCrit = mainHand["CritChance"]
					hasCritOverride = true
				} else if actor.Parent.WeaponData1 != nil {
					baseCrit = actor.Parent.WeaponData1.CritChance
					hasCritOverride = true
				}
			}

			critOverrideIs100 := hasCritOverride && baseCrit == 100

			//pob:block:src/Modules/CalcOffence.lua:2998-3004
			if skillModList.Flag(pass.Config, "BaseCritFromMainHand") && actor.WeaponData1 != nil {
				baseCrit = actor.WeaponData1.CritChance
			} else if skillModList.Flag(pass.Config, "AttackCritIsEqualToParentMainHand") &&
				actor.Parent != nil && actor.Parent.WeaponData1 != nil {
				baseCrit = actor.Parent.WeaponData1.CritChance
			}

			//pob:block:src/Modules/CalcOffence.lua:3006-3009
			if critOverrideIs100 {
				pass.Output["PreEffectiveCritChance"] = 100
				pass.Output["PreBifurcateCritChance"] = 100
				pass.Output["CritChance"] = 100
			} else {
				base := float64(0)
				inc := float64(0)
				more := float64(1)
				if !hasCritOverride {
					base = skillModList.Sum(mod.TypeBase, pass.Config, "CritChance")
					inc = skillModList.Sum(mod.TypeIncrease, pass.Config, "CritChance")
					more = skillModList.More(pass.Config, "CritChance")

					if env.ModeEffective {
						base += enemyDB.Sum(mod.TypeBase, nil, "SelfCritChance")
						inc += enemyDB.Sum(mod.TypeIncrease, nil, "SelfCritChance")
					}
				}

				pass.Output["CritChance"] = (baseCrit + base) * (1 + inc/100) * more

				preCapCritChance := pass.Output["CritChance"]
				pass.Output["CritChance"] = math.Min(pass.Output["CritChance"], 100)

				if baseCrit+base > 0 {
					pass.Output["CritChance"] = math.Max(pass.Output["CritChance"], 0)
				}

				pass.Output["PreEffectiveCritChance"] = pass.Output["CritChance"]
				preLuckyCritChance := pass.Output["CritChance"]

				//pob:block:src/Modules/CalcOffence.lua:3027-3042
				critRolls := float64(0)
				if env.ModeEffective && skillModList.Flag(pass.Config, "CritChanceLucky") {
					critRolls++
				}
				if skillModList.Flag(skillCfg, "ExtremeLuck") {
					critRolls *= 2
				}
				if critRolls != 0 {
					p := pass.Output["CritChance"] / 100
					if modDB.Flag(nil, "Unexciting") {
						pass.Output["CritChance"] = (3*p*p - 2*p*p*p) * 100
					} else {
						pass.Output["CritChance"] = (1 - math.Pow(1-p, critRolls+1)) * 100
					}
				}

				//pob:block:src/Modules/CalcOffence.lua:3043-3047
				pass.Output["PreBifurcateCritChance"] = pass.Output["CritChance"]
				if env.ModeEffective && skillModList.Flag(pass.Config, "BifurcateCrit") {
					pass.Output["CritChance"] = (1 - math.Pow(1-pass.Output["CritChance"]/100, 2)) * 100
				}

				preHitCheckCritChance := pass.Output["CritChance"]
				preSkillUseCritChance := pass.Output["CritChance"]
				if env.ModeEffective {
					//pob:block:src/Modules/CalcOffence.lua:3051-3056
					if skillModList.Flag(skillCfg, "Every3UseCrit") {
						pass.Output["CritChance"] = (2*pass.Output["CritChance"] + 100) / 3
					}
					if skillModList.Flag(skillCfg, "Every5UseCrit") {
						pass.Output["CritChance"] = (4*pass.Output["CritChance"] + 100) / 5
					}
					preHitCheckCritChance = pass.Output["CritChance"]
					//pob:block:src/Modules/CalcOffence.lua:2318-2320
					pass.Output["CritChance"] = pass.Output["CritChance"] * pass.Output["AccuracyHitChance"] / 100
				}

				if breakdown != nil && actor.Output["CritChance"] != baseCrit {
					if base != 0 {
						breakdown.AddLine("CritChance", fmt.Sprintf("(%g + %g) ^8(base)", baseCrit, base))
					} else {
						breakdown.AddLine("CritChance", fmt.Sprintf("%g ^8(base)", baseCrit+base))
					}
					if inc != 0 {
						breakdown.AddLine("CritChance", fmt.Sprintf("x %.2f", 1+inc/100)+" ^8(increased/reduced)")
					}
					if more != 1 {
						breakdown.AddLine("CritChance", fmt.Sprintf("x %.2f", more)+" ^8(more/less)")
					}
					breakdown.AddLine("CritChance", fmt.Sprintf("= %.2f%% ^8(crit chance)", actor.Output["PreEffectiveCritChance"]))
					if preCapCritChance > 100 {
						overCap := preCapCritChance - 100
						breakdown.AddLine("CritChance", fmt.Sprintf("Crit is overcapped by %.2f%% (%.2f%% increased Critical Strike Chance)", overCap, overCap/more/(baseCrit+base)*100))
					}
					if env.ModeEffective && critRolls != 0 {
						if skillModList.Flag(skillCfg, "Unexciting") {
							breakdown.AddLine("CritChance", "Crit Chance is Unexciting:")
							breakdown.AddLine("CritChance", fmt.Sprintf("(3 x %.4f^ 2) - (2 x %.4f^ 3)", preLuckyCritChance/100, preLuckyCritChance/100))
						} else {
							breakdown.AddLine("CritChance", "Crit Chance is Lucky:")
							breakdown.AddLine("CritChance", fmt.Sprintf("1 - (1 - %.4f)^ %d", preLuckyCritChance/100, int(critRolls)+1))
						}
						breakdown.AddLine("CritChance", fmt.Sprintf("= %.2f%%", pass.Output["PreBifurcateCritChance"]))
					}
					//pob:block:src/Modules/CalcOffence.lua:3095-3103
					if env.ModeEffective {
						every3 := skillModList.Flag(skillCfg, "Every3UseCrit")
						every5 := skillModList.Flag(skillCfg, "Every5UseCrit")
						if every3 {
							breakdown.AddLine("CritChance", fmt.Sprintf("+ %.2f%% ^8(crit every 3rd use)", (2*preSkillUseCritChance+100)/3-preSkillUseCritChance))
						}
						if every5 {
							breakdown.AddLine("CritChance", fmt.Sprintf("+ %.2f%% ^8(crit every 5th use)", (4*preSkillUseCritChance+100)/5-preSkillUseCritChance))
						}
						if every3 || every5 {
							breakdown.AddLine("CritChance", fmt.Sprintf("= %.2f%%", preHitCheckCritChance))
						}
					}
					if env.ModeEffective && actor.Output["AccuracyHitChance"] < 100 {
						breakdown.AddLine("CritChance", "Crit confirmation roll:")
						breakdown.AddLine("CritChance", fmt.Sprintf("%.2f%%", preHitCheckCritChance))
						breakdown.AddLine("CritChance", fmt.Sprintf("x %.2f ^8(chance to hit)", actor.Output["AccuracyHitChance"]/100))
						breakdown.AddLine("CritChance", fmt.Sprintf("= %.2f%%", actor.Output["CritChance"]))
					}
				}
			}

			if skillModList.Flag(pass.Config, "NoCritMultiplier") {
				pass.Output["CritMultiplier"] = 1
			} else {
				extraDamage := skillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplier") / 100
				multiOverride := skillModList.Override(skillCfg, "CritMultiplier")
				if multiOverride != nil {
					extraDamage = (multiOverride.Float() - 100) / 100
				}

				if env.ModeEffective {
					enemyInc := 1 + enemyDB.Sum(mod.TypeIncrease, nil, "SelfCritMultiplier")/100
					extraDamage = extraDamage + enemyDB.Sum(mod.TypeBase, nil, "SelfCritMultiplier")/100
					extraDamage = utils.RoundTo(extraDamage*enemyInc, 2)
					if breakdown != nil && enemyInc != 1 {
						breakdown.AddLine("CritMultiplier",
							fmt.Sprintf("%.2f%% ^8(additional extra damage)", (enemyDB.Sum(mod.TypeBase, nil, "SelfCritMultiplier")+skillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplier"))/100),
							fmt.Sprintf("x %.2f ^8(increased/reduced extra crit damage taken by enemy)", enemyInc),
							fmt.Sprintf("= %.2f%% ^8(extra crit damage)", extraDamage*100),
						)
					}
				}

				//pob:block:src/Modules/CalcOffence.lua:3136-3170
				critOverrideForBifurcate := skillModList.Override(pass.Config, "CritChance")
				overrideIs100 := critOverrideForBifurcate != nil && critOverrideForBifurcate.Float() == 100
				critChancePercentage, havePreBifurcate := pass.Output["PreBifurcateCritChance"]
				if env.ModeEffective && skillModList.Flag(pass.Config, "BifurcateCrit") &&
					havePreBifurcate && !overrideIs100 {
					bifurcateMultiChance := (critChancePercentage * critChancePercentage) / 100
					effectiveCritChance := pass.Output["CritChance"]

					bifurcateUseChance := float64(1)
					if skillModList.Flag(skillCfg, "Every3UseCrit") {
						bifurcateUseChance = bifurcateUseChance * 2 / 3
					}
					if skillModList.Flag(skillCfg, "Every5UseCrit") {
						bifurcateUseChance = bifurcateUseChance * 4 / 5
					}
					bifurcateMultiChance *= bifurcateUseChance

					conditionalBifurcateChance := float64(0)
					if effectiveCritChance > 0 {
						conditionalBifurcateChance = bifurcateMultiChance / effectiveCritChance
					}
					pass.Output["CritBifurcates"] = 1 + conditionalBifurcateChance
					extraDamage += conditionalBifurcateChance * extraDamage
					// mod doesn't affect output and is purely descriptive
					//pob:block:src/Modules/CalcOffence.lua:3168-3169
					skillModList.AddMod(mod.NewFloat("CritMultiplier", mod.TypeMore, math.Floor(conditionalBifurcateChance*100*100+0.0001)/100).
						Source("Bifurcated Crit Damage Bonus").Flag(mod.MFlagHit))
				}

				pass.Output["CritMultiplier"] = 1 + math.Max(0, extraDamage)
			}

			critChancePercentage := pass.Output["CritChance"] / 100
			pass.Output["CritEffect"] = 1 - critChancePercentage + critChancePercentage*pass.Output["CritMultiplier"]
			pass.Output["BonusCritDotMultiplier"] = (skillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplier") - 50) * skillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplierAppliesToDegen") / 10000

			if breakdown != nil && actor.Output["CritEffect"] != 1 {
				breakdown.AddLine("CritEffect",
					fmt.Sprintf("(1 - %.4f) ^8(portion of damage from non-crits)", critChancePercentage),
					fmt.Sprintf("+ [ (%.4f x %g) ^8(portion of damage from crits)", critChancePercentage, actor.Output["CritMultiplier"]),
					fmt.Sprintf("= %.3f", actor.Output["CritEffect"]),
				)
			}
		}

		//pob:src/Modules/CalcOffence.lua:3184-3186
		if pass.Output["CritChance"] != 0 {
			skillModList.Conditions["CritInPast8Sec"] = true
		}

		//pob:block:src/Modules/CalcOffence.lua:3188-3209
		pass.Output["ScaledDamageEffect"] = 1

		// Calculate chance and multiplier for dealing triple damage on Normal and Crit
		pass.Output["TripleDamageChanceOnCrit"] = min(skillModList.Sum(mod.TypeBase, pass.Config, "TripleDamageChanceOnCrit"), 100)
		pass.Output["TripleDamageChance"] = min(skillModList.Sum(mod.TypeBase, pass.Config, "TripleDamageChance")+(utils.Ternary(env.ModeEffective, enemyDB.Sum(mod.TypeBase, pass.Config, "SelfTripleDamageChance"), 0))+(pass.Output["TripleDamageChanceOnCrit"]*pass.Output["CritChance"]/100), 100)
		pass.Output["TripleDamageEffect"] = 2 * pass.Output["TripleDamageChance"] / 100

		// Calculate chance and multiplier for dealing double damage on Normal and Crit
		pass.Output["DoubleDamageChanceOnCrit"] = min(skillModList.Sum(mod.TypeBase, pass.Config, "DoubleDamageChanceOnCrit"), 100)
		pass.Output["DoubleDamageChance"] = min(skillModList.Sum(mod.TypeBase, pass.Config, "DoubleDamageChance")+(utils.Ternary(env.ModeEffective, enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDoubleDamageChance"), 0))+(pass.Output["DoubleDamageChanceOnCrit"]*pass.Output["CritChance"]/100), 100)
		if actor.Output["IntimidatingUpTimeRatio"] != 0 && activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
			pass.Output["DoubleDamageChance"] = 100
		} else if actor.Output["IntimidatingUpTimeRatio"] != 0 {
			pass.Output["DoubleDamageChance"] = min(pass.Output["DoubleDamageChance"]+actor.Output["IntimidatingUpTimeRatio"], 100)
		}

		// Triple Damage overrides Double Damage. If you have both, it's the same as just having Triple
		// We need to subtract the probability of both happening in favor of Triple Damage
		if pass.Output["TripleDamageChance"] > 0 {
			pass.Output["DoubleDamageChance"] = max(pass.Output["DoubleDamageChance"]-pass.Output["TripleDamageChance"]*pass.Output["DoubleDamageChance"]/100, 0)
		}
		pass.Output["DoubleDamageEffect"] = pass.Output["DoubleDamageChance"] / 100
		pass.Output["ScaledDamageEffect"] = pass.Output["ScaledDamageEffect"] * (1 + pass.Output["DoubleDamageEffect"] + pass.Output["TripleDamageEffect"])

		// Calculate culling DPS
		//pob:src/Modules/CalcOffence.lua:3211
		selectedSpeed, gotSpeed := actor.Output["HitSpeed"]
		if !gotSpeed || selectedSpeed == 0 {
			selectedSpeed = actor.Output["Speed"]
		}
		DpsMultiplier := utils.OrDefault(skillData.DpsMultiplier, 1)
		hitRate := pass.Output["HitChance"] / 100 * selectedSpeed * DpsMultiplier

		criticalCull := skillModList.Max(pass.Config, "CriticalCullPercent")
		if criticalCull > 0 {
			//pob:src/Modules/CalcOffence.lua:3216
			criticalCull = math.Min(criticalCull,
				criticalCull*(1-math.Pow(1-pass.Output["CritChance"]/100, hitRate)))
		}
		regularCull := skillModList.Max(pass.Config, "CullPercent")
		maxCullPercent := max(criticalCull, regularCull)
		actor.Output["CullPercent"] = maxCullPercent
		actor.Output["CullMultiplier"] = 100 / (100 - actor.Output["CullPercent"])

		// Calculate reservation DPS
		//pob:block:src/Modules/CalcOffence.lua:3224
		actor.Output["ReservationDpsMultiplier"] = 100 / (100 - enemyDB.Sum(mod.TypeBase, nil, "LifeReservationPercent"))
		runSkillFunc("postCritFunc")

		//pob:block:src/Modules/CalcOffence.lua:3228-3255
		var addedDamageRedirectType data.DamageType
		if skillModList.Flag(pass.Config, "AllAddedDamageAsLightning") {
			addedDamageRedirectType = data.DamageTypeLightning
		} else if skillModList.Flag(pass.Config, "AllAddedDamageAsCold") {
			addedDamageRedirectType = data.DamageTypeCold
		}
		if addedDamageRedirectType != "" {
			for _, damageType := range data.DamageType("").Values() {
				if damageType == addedDamageRedirectType {
					continue
				}
				for _, suffix := range []string{"Min", "Max"} {
					for _, value := range skillModList.Tabulate(mod.TypeBase, pass.Config, string(damageType)+suffix) {
						if strings.Contains(string(value.Mod.GetSource()), "ElementalHit") {
							continue
						}
						newMod := value.Mod.Clone().SetName(string(addedDamageRedirectType) + suffix)
						newMod.SetReplaced(false)
						skillModList.ConvertMod(string(damageType)+suffix, newMod)
					}
				}
			}
		}

		// Calculate base hit damage
		for _, damageType := range data.DamageType("").Values() {
			damageTypeMin := string(damageType) + "Min"
			damageTypeMax := string(damageType) + "Max"

			baseMultiplier := float64(1)
			if activeSkill.ActiveEffect.GrantedEffectLevel.BaseMultiplier != nil {
				baseMultiplier = *activeSkill.ActiveEffect.GrantedEffectLevel.BaseMultiplier
			} else if skillData.BaseMultiplier != 0 {
				baseMultiplier = skillData.BaseMultiplier
			}

			damageEffectiveness := float64(1)
			if activeSkill.ActiveEffect.GrantedEffectLevel.DamageEffectiveness != nil {
				damageEffectiveness = *activeSkill.ActiveEffect.GrantedEffectLevel.DamageEffectiveness
			} else if skillData.DamageEffectiveness != 0 {
				damageEffectiveness = skillData.DamageEffectiveness
			}

			addedMin := skillModList.Sum(mod.TypeBase, pass.Config, damageTypeMin) + enemyDB.Sum(mod.TypeBase, pass.Config, "Self"+damageTypeMin)
			addedMax := skillModList.Sum(mod.TypeBase, pass.Config, damageTypeMax) + enemyDB.Sum(mod.TypeBase, pass.Config, "Self"+damageTypeMax)
			addedMult := CalcMod(skillModList, pass.Config, "Added"+string(damageType)+"Damage", "AddedDamage")

			baseMin := (utils.GetOr(pass.Source, damageTypeMin, utils.Interface(float64(0))).(float64)+
				utils.GetOr(pass.Source, string(damageType)+"BonusMin", utils.Interface(float64(0))).(float64))*
				baseMultiplier + addedMin*damageEffectiveness*addedMult

			baseMax := (utils.GetOr(pass.Source, damageTypeMax, utils.Interface(float64(0))).(float64)+
				utils.GetOr(pass.Source, string(damageType)+"BonusMax", utils.Interface(float64(0))).(float64))*
				baseMultiplier + addedMax*damageEffectiveness*addedMult

			pass.Output[damageTypeMin+"Base"] = baseMin
			pass.Output[damageTypeMax+"Base"] = baseMax

			if breakdown != nil {
				if baseMin != 0 && baseMax != 0 {
					breakdown.AddLine(string(damageType), "Base damage:")
					plus := ""
					sourceMin := utils.GetOr(pass.Source, damageTypeMin, float64(0))
					sourceMax := utils.GetOr(pass.Source, damageTypeMax, float64(0))
					if sourceMin != 0 || sourceMax != 0 {
						breakdown.AddLine(string(damageType), fmt.Sprintf("%.0f to %.0f ^8(base damage from %s)", sourceMin, sourceMax, utils.Ternary(pass.Source.Type != "", "weapon", "skill")))
						if baseMultiplier != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(base damage multiplier)", baseMultiplier))
						}
						plus = "+ "
					}
					if addedMin != 0 || addedMax != 0 {
						breakdown.AddLine(string(damageType), fmt.Sprintf("%s%.0f to %.0f ^8(added damage)", plus, addedMin, addedMax))
						if damageEffectiveness != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(damage effectiveness)", damageEffectiveness))
						}
						if addedMult != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(added damage multiplier)", addedMult))
						}
					}
					breakdown.AddLine(string(damageType), fmt.Sprintf("= %.1f to %.1f", baseMin, baseMax))
				}
			}
		}

		totalHitMin := float64(0)
		totalHitMax := float64(0)
		totalHitAvg := float64(0)

		totalCritMin := float64(0)
		totalCritMax := float64(0)
		totalCritAvg := float64(0)

		ghostReaver := skillModList.Flag(nil, "GhostReaver")
		ghostReaverLifeLeech := float64(0)

		pass.Output["LifeLeech"] = 0
		pass.Output["LifeLeechInstant"] = 0
		pass.Output["EnergyShieldLeech"] = 0
		pass.Output["EnergyShieldLeechInstant"] = 0
		pass.Output["ManaLeech"] = 0
		pass.Output["ManaLeechInstant"] = 0
		pass.Output["impaleStoredHitAvg"] = 0

		// Calculate hit damage for each damage type
		for p := 1; p <= 2; p++ {
			// Pass 1 is critical strike damage, pass 2 is non-critical strike
			pass.Config.SkillCond["CriticalStrike"] = p == 1
			lifeLeechTotal := float64(0)
			energyShieldLeechTotal := float64(0)
			manaLeechTotal := float64(0)

			noLifeLeech := skillModList.Flag(pass.Config, "CannotLeechLife") || enemyDB.Flag(nil, "CannotLeechLifeFromSelf") || skillModList.Flag(pass.Config, "CannotGainLife")
			noEnergyShieldLeech := skillModList.Flag(pass.Config, "CannotLeechEnergyShield") || enemyDB.Flag(nil, "CannotLeechEnergyShieldFromSelf") || skillModList.Flag(pass.Config, "CannotGainEnergyShield")
			noManaLeech := skillModList.Flag(pass.Config, "CannotLeechMana") || enemyDB.Flag(nil, "CannotLeechManaFromSelf") || skillModList.Flag(pass.Config, "CannotGainMana")

			for _, damageType := range data.DamageType("").Values() {
				damageTypeHitMin := float64(0)
				damageTypeHitMax := float64(0)
				damageTypeHitAvg := float64(0)
				damageTypeLuckyChance := float64(0)
				damageTypeHitAvgLucky := float64(0)
				damageTypeHitAvgNotLucky := float64(0)
				damageTypeHitAvgUnlucky := float64(0)

				if utils.HasTrue(skillFlags, SkillFlagHit) && utils.HasTrue(canDeal, damageType) {
					damageTypeHitMin, damageTypeHitMax = calcDamage(activeSkill, pass.Output, pass.Config, breakdown, string(damageType), damageType, 0, nil)
					convMult := activeSkill.conversionTableFor(cfg)[damageType].Mult

					if p == 2 && breakdown != nil {
						breakdown.AddLine(string(damageType), "Hit damage:")
						breakdown.AddLine(string(damageType), fmt.Sprintf("%.0f to %.0f ^8(total damage)", damageTypeHitMin, damageTypeHitMax))
						if convMult != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %g ^8(%g%% converted to other damage types)", convMult, (1-convMult)*100))
						}
						if pass.Output["DoubleDamageEffect"] != 0 {
							if pass.Output["TripleDamageEffect"] != 0 {
								breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(1 + %.2f + %.2f multiplier from %.1f%% chance to deal double damage and %.0f%% chance to deal triple damage)", 1+pass.Output["DoubleDamageEffect"]+pass.Output["TripleDamageEffect"], pass.Output["DoubleDamageEffect"], pass.Output["TripleDamageEffect"], pass.Output["DoubleDamageChance"], pass.Output["TripleDamageChance"]))
							} else {
								breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(multiplier from %.0f%% chance to deal double damage)", 1+pass.Output["DoubleDamageEffect"], pass.Output["DoubleDamageChance"]))
							}
						} else if pass.Output["TripleDamageEffect"] != 0 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(multiplier from %.0f%% chance to deal triple damage)", 1+pass.Output["TripleDamageEffect"], pass.Output["TripleDamageChance"]))
						}
						if pass.Output["RuthlessBlowHitEffect"] != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(ruthless blow effect modifier)", pass.Output["RuthlessBlowHitEffect"]))
						}
						if pass.Output["FistOfWarDamageEffect"] != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(fist of war effect modifier)", pass.Output["FistOfWarDamageEffect"]))
						}
						if actor.Output["OffensiveWarcryEffect"] != 1 && !activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(aggregated warcry exerted effect modifier)", actor.Output["OffensiveWarcryEffect"]))
						}
						if actor.Output["MaxOffensiveWarcryEffect"] != 1 && activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(aggregated max warcry exerted effect modifier)", actor.Output["MaxOffensiveWarcryEffect"]))
						}
					}

					if skillModList.Flag(nil, "Condition:WarcryMaxHit") {
						pass.Output["allMult"] = convMult * pass.Output["ScaledDamageEffect"] * pass.Output["RuthlessBlowHitEffect"] * pass.Output["FistOfWarDamageEffect"] * actor.Output["MaxOffensiveWarcryEffect"]
					} else {
						pass.Output["allMult"] = convMult * pass.Output["ScaledDamageEffect"] * pass.Output["RuthlessBlowHitEffect"] * pass.Output["FistOfWarDamageEffect"] * actor.Output["OffensiveWarcryEffect"]
					}

					allMult := pass.Output["allMult"]
					if p == 1 {
						// Apply crit multiplier
						allMult *= pass.Output["CritMultiplier"]
					}

					damageTypeHitMin *= allMult
					damageTypeHitMax *= allMult

					if skillModList.Flag(skillCfg, "LuckyHits") ||
						(p == 2 && damageType == data.DamageTypeLightning && skillModList.Flag(skillCfg, "LightningNoCritLucky")) ||
						(p == 1 && skillModList.Flag(skillCfg, "CritLucky")) ||
						(damageType == data.DamageTypeLightning && modDB.Flag(nil, "LightningLuckHits")) ||
						(damageType == data.DamageTypeChaos && modDB.Flag(nil, "ChaosLuckyHits")) ||
						(damageType == data.DamageTypeFire && modDB.Flag(nil, "FireLuckyHits")) ||
						(damageType == data.DamageTypeCold && modDB.Flag(nil, "ColdLuckyHits")) ||
						((damageType == data.DamageTypeLightning || damageType == data.DamageTypeCold || damageType == data.DamageTypeFire) && skillModList.Flag(skillCfg, "ElementalLuckHits")) {
						damageTypeLuckyChance = 1
					} else {
						damageTypeLuckyChance = math.Min(skillModList.Sum(mod.TypeBase, skillCfg, "LuckyHitsChance"), 100) / 100
					}
					if skillModList.Flag(skillCfg, "UnluckyHits") {
						damageTypeLuckyChance--
					}

					rolls := damageTypeLuckyChance
					if skillModList.Flag(skillCfg, "ExtremeLuck") {
						rolls *= 2
					}
					if skillModList.Flag(skillCfg, "Unexciting") {
						rolls = 0
					}
					pass.Output["DamageRolls"] = rolls
					if p == 1 {
						pass.Output["CritDamageRolls"] = rolls
					}
					rolls = math.Abs(rolls) + 2

					damageTypeHitAvgNotLucky = damageTypeHitMin/rolls + damageTypeHitMax/rolls
					damageTypeHitAvgLucky = damageTypeHitMin/rolls + (rolls-1)*damageTypeHitMax/rolls
					damageTypeHitAvgUnlucky = (rolls-1)*damageTypeHitMin/rolls + damageTypeHitMax/rolls
					if damageTypeLuckyChance >= 0 {
						damageTypeHitAvg = damageTypeHitAvgNotLucky*(1-damageTypeLuckyChance) + damageTypeHitAvgLucky*damageTypeLuckyChance
					} else {
						damageTypeHitAvg = damageTypeHitAvgNotLucky*(1-math.Abs(damageTypeLuckyChance)) + damageTypeHitAvgUnlucky*math.Abs(damageTypeLuckyChance)
					}

					if (damageTypeHitMin != 0 || damageTypeHitMax != 0) && env.ModeEffective {
						// Apply enemy resistances and damage taken modifiers
						resist := float64(0)
						pen := float64(0)
						sourceRes := data.DamageType("")
						takenInc := enemyDB.Sum(mod.TypeIncrease, pass.Config, "DamageTaken", string(damageType)+"DamageTaken")
						takenMore := enemyDB.More(pass.Config, "DamageTaken", string(damageType)+"DamageTaken")

						// Check if player is supposed to ignore a damage type, or if it's ignored on enemy side.
						useThisResist := func(damageType data.DamageType) float64 {
							names := []string{"Ignore" + string(damageType) + "Resistance"}
							if damageType.IsElemental() {
								names = append(names, "IgnoreElementalResistances")
							}
							if skillModList.Flag(pass.Config, names...) || enemyDB.Flag(nil, "SelfIgnore"+string(damageType)+"Resistance") {
								return 0
							}
							chanceNames := []string{"ChanceToIgnore" + string(damageType) + "Resistance"}
							if damageType.IsElemental() {
								chanceNames = append(chanceNames, "ChanceToIgnoreElementalResistances")
							}
							chanceToIgnore := math.Min(math.Max(skillModList.Sum(mod.TypeBase, pass.Config, chanceNames...), 0), 100)
							return 1 - chanceToIgnore/100
						}

						if damageType == data.DamageTypePhysical {
							// store pre-armour physical damage from attacks for impale calculations
							if p == 1 {
								pass.Output["impaleStoredHitAvg"] = pass.Output["impaleStoredHitAvg"] + damageTypeHitAvg*(pass.Output["CritChance"]/100)
							} else {
								pass.Output["impaleStoredHitAvg"] = pass.Output["impaleStoredHitAvg"] + damageTypeHitAvg*(1-pass.Output["CritChance"]/100)
							}
							enemyArmour := math.Max(CalcVal(enemyDB, "Armour", nil), 0)
							armourReduction := CalcArmourReductionF(enemyArmour, damageTypeHitAvg)

							chanceToIgnorePDR := math.Min(skillModList.Sum(mod.TypeBase, pass.Config, "ChanceToIgnoreEnemyPhysicalDamageReduction"), 100)
							if chanceToIgnorePDR > 0 && chanceToIgnorePDR < 100 {
								switch env.Build.GetStringOption("ChanceToIgnoreEnemyPhysicalDamageReductionMode") {
								case "MAX":
									chanceToIgnorePDR = 100
								case "MIN":
									chanceToIgnorePDR = 0
								}
							}
							if skillModList.Flag(pass.Config, "IgnoreEnemyPhysicalDamageReduction") || chanceToIgnorePDR >= 100 {
								resist = 0
							} else {
								resist = math.Min(math.Max(0, enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction")+skillModList.Sum(mod.TypeBase, pass.Config, "EnemyPhysicalDamageReduction")+armourReduction), data.EnemyPhysicalDamageReductionCap)
								if resist > 0 {
									resist *= 1 - (skillModList.Sum(mod.TypeBase, nil, "PartialIgnoreEnemyPhysicalDamageReduction")/100 + chanceToIgnorePDR/100)
								}
							}
						} else {
							resist = calcResistForType(damageType, nil)
							if ((skillModList.Flag(pass.Config, "ChaosDamageUsesLowestResistance") ||
								skillModList.Flag(pass.Config, "ChaosDamageUsesHighestResistance")) && damageType == data.DamageTypeChaos) ||
								(skillModList.Flag(pass.Config, "ElementalDamageUsesLowestResistance") && damageType.IsElemental()) {
								// Default to using the current damage type
								elementUsed := damageType
								if damageType.IsElemental() {
									takenInc += enemyDB.Sum(mod.TypeIncrease, pass.Config, "ElementalDamageTaken")
								}

								for _, eleDamageType := range data.DamageType("").Values() {
									if eleDamageType.IsElemental() && useThisResist(eleDamageType) > 0 && damageType != eleDamageType {
										currentElementResist := calcResistForType(eleDamageType, nil)
										if skillModList.Flag(pass.Config, "ChaosDamageUsesHighestResistance") {
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
								if elementUsed.IsElemental() {
									pen = skillModList.Sum(mod.TypeBase, pass.Config, string(elementUsed)+"Penetration", "ElementalPenetration")
								} else if elementUsed == data.DamageTypeChaos {
									pen = skillModList.Sum(mod.TypeBase, pass.Config, "ChaosPenetration")
								}
								sourceRes = elementUsed
							} else if damageType.IsElemental() {
								pen = skillModList.Sum(mod.TypeBase, pass.Config, string(damageType)+"Penetration", "ElementalPenetration")
								takenInc += enemyDB.Sum(mod.TypeIncrease, pass.Config, "ElementalDamageTaken")
							} else if damageType == data.DamageTypeChaos {
								pen = skillModList.Sum(mod.TypeBase, pass.Config, "ChaosPenetration")
							}
						}

						if utils.HasTrue(skillFlags, SkillFlagProjectile) {
							takenInc += enemyDB.Sum(mod.TypeIncrease, nil, "ProjectileDamageTaken")
						}

						if utils.HasTrue(skillFlags, SkillFlagProjectile) && utils.HasTrue(skillFlags, SkillFlagAttack) {
							takenInc += enemyDB.Sum(mod.TypeIncrease, nil, "ProjectileAttackDamageTaken")
						}

						if utils.HasTrue(skillFlags, SkillFlagTrap) || utils.HasTrue(skillFlags, SkillFlagMine) {
							takenInc += enemyDB.Sum(mod.TypeIncrease, nil, "TrapMineDamageTaken")
						}

						invertChanceEle := math.Max(math.Min(skillModList.Sum(mod.TypeChance, pass.Config, "HitsInvertEleResChance"), 1), 0)
						invertChanceChaos := math.Max(math.Min(skillModList.Sum(mod.TypeChance, pass.Config, "HitsInvertChaosResChance"), 1), 0)
						invertChance := float64(0)
						switch {
						case damageType.IsElemental():
							invertChance = invertChanceEle
						case damageType == data.DamageTypeChaos:
							invertChance = invertChanceChaos
						}

						effMult := (1 + takenInc/100) * takenMore
						useResChance := useThisResist(damageType)
						useRes := useResChance > 0
						effectiveResist := resist
						if damageType.IsElemental() && skillModList.Flag(pass.Config, "CannotElePenIgnore") {
							if invertChance > 0 {
								effectiveResist = resist - 2*invertChance*resist
							}
							effMult *= 1 - effectiveResist/100
						} else if useRes {
							if invertChance > 0 {
								effectiveResist = ((resist-pen)*(1-invertChance) + (-resist-pen)*invertChance) * useResChance
							} else {
								effectiveResist = (resist - pen) * useResChance
							}
							effMult *= 1 - effectiveResist/100
						}

						damageTypeHitMin = damageTypeHitMin * effMult
						damageTypeHitMax = damageTypeHitMax * effMult
						damageTypeHitAvg = damageTypeHitAvg * effMult

						if env.Mode == OutputModeCalcs {
							pass.Output[string(damageType)+"EffMult"] = effMult
						}
						if p == 2 && breakdown != nil && (effMult != 1 || sourceRes != "" || invertChance > 0) && skillModList.Flag(pass.Config, utils.Ternary(isElemental[string(damageType)], "CannotElePenIgnore", "")) {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.3f ^8(effective DPS modifier)", effMult))
							breakdown.EffMult(string(damageType+"EffMult"), damageType, resist, 0, takenInc, effMult, takenMore, sourceRes, useRes, invertChance, &effectiveResist)
						} else if p == 2 && breakdown != nil && (effMult != 1 || sourceRes != "" || invertChance > 0) {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.3f ^8(effective DPS modifier)", effMult))
							breakdown.EffMult(string(damageType+"EffMult"), damageType, resist, pen, takenInc, effMult, takenMore, sourceRes, useRes, invertChance, &effectiveResist)
						}
					}
					if p == 2 && breakdown != nil {
						breakdown.AddLine(string(damageType), fmt.Sprintf("= %.2f to %.2f", damageTypeHitMin, damageTypeHitMax))
					}

					// Beginning of Leech Calculation for this DamageType
					if utils.HasTrue(skillFlags, SkillFlagMine) || utils.HasTrue(skillFlags, SkillFlagTrap) || utils.HasTrue(skillFlags, SkillFlagTotem) {
						if !noLifeLeech {
							lifeLeech := skillModList.Sum(mod.TypeBase, pass.Config, "DamageLifeLeechToPlayer")
							if lifeLeech > 0 {
								lifeLeechTotal += damageTypeHitAvg * lifeLeech / 100
							}
						}
					} else {
						if !noLifeLeech {
							lifeLeech := float64(0)
							if skillModList.Flag(nil, "LifeLeechBasedOnChaosDamage") {
								if damageType == data.DamageTypeChaos {
									lifeLeech = skillModList.Sum(mod.TypeBase, pass.Config, "DamageLeech", "DamageLifeLeech", "PhysicalDamageLifeLeech", "LightningDamageLifeLeech", "ColdDamageLifeLeech", "FireDamageLifeLeech", "ChaosDamageLifeLeech", "ElementalDamageLifeLeech") + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDamageLifeLeech")/100
								} else {
									lifeLeech = 0
								}
							} else {
								names := []string{"DamageLeech", "DamageLifeLeech", string(damageType) + "DamageLifeLeech"}
								if damageType.IsElemental() {
									names = append(names, "ElementalDamageLifeLeech")
								}
								lifeLeech = skillModList.Sum(mod.TypeBase, pass.Config, names...) + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDamageLifeLeech")/100
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
							energyShieldLeech := skillModList.Sum(mod.TypeBase, pass.Config, names...) + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDamageEnergyShieldLeech")/100
							if energyShieldLeech > 0 {
								energyShieldLeechTotal = energyShieldLeechTotal + damageTypeHitAvg*energyShieldLeech/100
							}
						}

						if !noManaLeech {
							names := []string{"DamageLeech", "DamageManaLeech", string(damageType) + "DamageManaLeech"}
							if damageType.IsElemental() {
								names = append(names, "ElementalDamageManaLeech")
							}
							manaLeech := skillModList.Sum(mod.TypeBase, pass.Config, names...) + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDamageManaLeech")/100
							if manaLeech > 0 {
								manaLeechTotal = manaLeechTotal + damageTypeHitAvg*manaLeech/100
							}
						}
					}
				} else {
					if breakdown != nil {
						breakdown.AddLine(string(damageType), string("You can't deal "+damageType+" damage"))
					}
				}

				if p == 1 {
					pass.Output[string(damageType)+"CritAverage"] = damageTypeHitAvg
					totalCritAvg = totalCritAvg + damageTypeHitAvg
					totalCritMin = totalCritMin + damageTypeHitMin
					totalCritMax = totalCritMax + damageTypeHitMax
				} else {
					//pob:src/Modules/CalcOffence.lua:3564-3567
					if env.Mode == OutputModeCalcs {
						pass.Output[string(damageType)+"Min"] = damageTypeHitMin
						pass.Output[string(damageType)+"Max"] = damageTypeHitMax
					}
					pass.Output[string(damageType)+"HitAverage"] = damageTypeHitAvg
					totalHitAvg = totalHitAvg + damageTypeHitAvg
					totalHitMin = totalHitMin + damageTypeHitMin
					totalHitMax = totalHitMax + damageTypeHitMax
				}
			}

			if skillData.LifeLeechPerUse != 0 {
				lifeLeechTotal += skillData.LifeLeechPerUse
			}

			if skillData.ManaLeechPerUse != 0 {
				manaLeechTotal += skillData.ManaLeechPerUse
			}

			//pob:block:src/Modules/CalcOffence.lua:3581-3591
			if ghostReaver {
				lifeLeechTotal = min(lifeLeechTotal, actor.Output["MaxEnergyShieldLeechInstance"])
			} else {
				lifeLeechTotal = min(lifeLeechTotal, actor.Output["MaxLifeLeechInstance"])
			}
			energyShieldLeechTotal = min(energyShieldLeechTotal, actor.Output["MaxEnergyShieldLeechInstance"])
			manaLeechTotal = min(manaLeechTotal, actor.Output["MaxManaLeechInstance"])
			if ghostReaver && noEnergyShieldLeech {
				lifeLeechTotal = 0
			}

			portion := 1 - pass.Output["CritChance"]/100
			if p == 1 {
				portion = pass.Output["CritChance"] / 100
			}

			if ghostReaver {
				ghostReaverLifeLeech += lifeLeechTotal * portion
			} else {
				pass.Output["LifeLeech"] += lifeLeechTotal * portion
			}

			pass.Output["EnergyShieldLeech"] += energyShieldLeechTotal * portion
			pass.Output["ManaLeech"] += manaLeechTotal * portion
		}

		pass.Output["TotalMin"] = totalHitMin
		pass.Output["TotalMax"] = totalHitMax

		/*
			TODO ElementalEquilibrium
			if skillModList.Flag(skillCfg, "ElementalEquilibrium") and not env.configInput.EEIgnoreHitDamage and (actor.Output["FireHitAverage"] + actor.Output["ColdHitAverage"] + actor.Output["LightningHitAverage"] > 0) {
				// Update enemy hit-by-damage-type conditions
				enemyDB.conditions.HitByFireDamage = actor.Output["FireHitAverage"] > 0
				enemyDB.conditions.HitByColdDamage = actor.Output["ColdHitAverage"] > 0
				enemyDB.conditions.HitByLightningDamage = actor.Output["LightningHitAverage"] > 0
			}
		*/
		highestType := "Physical"
		// For each damage type, calculate percentage of total damage. Also tracks the highest damage type and outputs a Condition:TypeIsHighestDamageType flag for whichever the highest type is
		for _, damageType := range dmgTypes {
			if pass.Output[damageType+"HitAverage"] > 0 {
				portion := pass.Output[damageType+"HitAverage"] / totalHitAvg * 100
				skillModList.AddMod(mod.NewFlag("Condition:"+damageType+"HasDamage", true).Source("Config"))
				highestPortion := pass.Output[highestType+"HitAverage"] / totalHitAvg * 100
				if portion > highestPortion {
					highestType = damageType
				}
				if breakdown != nil {
					breakdown.AddLine(damageType, fmt.Sprintf("Portion of total damage: %.2f%%", portion))
				}
			}
		}
		skillModList.AddMod(mod.NewFlag("Condition:"+highestType+"IsHighestDamageType", true).Source("Config"))

		// Calculate leech
		getLeechInstances := func(amount float64, total float64) (float64, float64) {
			if total == 0 {
				return 0, 0
			}
			duration := amount / total / data.LeechRateBase
			return duration, duration * hitRate
		}
		pass.Output["LifeLeechInstantProportion"] = max(min(skillModList.Sum(mod.TypeBase, pass.Config, "InstantLifeLeech"), 100), 0) / 100
		if pass.Output["LifeLeechInstantProportion"] > 0 {
			pass.Output["LifeLeechInstant"] = pass.Output["LifeLeech"] * pass.Output["LifeLeechInstantProportion"]
			pass.Output["LifeLeech"] = pass.Output["LifeLeech"] * (1 - pass.Output["LifeLeechInstantProportion"])
		}
		if ghostReaver && ghostReaverLifeLeech > 0 {
			pass.Output["EnergyShieldLeech"] = pass.Output["EnergyShieldLeech"] + ghostReaverLifeLeech
			pass.Output["LifeLeech"] = 0
			pass.Output["LifeLeechInstant"] = 0
		}
		pass.Output["EnergyShieldLeechInstantProportion"] = max(min(skillModList.Sum(mod.TypeBase, pass.Config, "InstantEnergyShieldLeech"), 100), 0) / 100
		if pass.Output["EnergyShieldLeechInstantProportion"] > 0 {
			pass.Output["EnergyShieldLeechInstant"] = pass.Output["EnergyShieldLeech"] * pass.Output["EnergyShieldLeechInstantProportion"]
			pass.Output["EnergyShieldLeech"] = pass.Output["EnergyShieldLeech"] * (1 - pass.Output["EnergyShieldLeechInstantProportion"])
		}
		pass.Output["ManaLeechInstantProportion"] = max(min(skillModList.Sum(mod.TypeBase, pass.Config, "InstantManaLeech"), 100), 0) / 100
		if pass.Output["ManaLeechInstantProportion"] > 0 {
			pass.Output["ManaLeechInstant"] = pass.Output["ManaLeech"] * pass.Output["ManaLeechInstantProportion"]
			pass.Output["ManaLeech"] = pass.Output["ManaLeech"] * (1 - pass.Output["ManaLeechInstantProportion"])
		}
		pass.Output["LifeLeechDuration"], pass.Output["LifeLeechInstances"] = getLeechInstances(pass.Output["LifeLeech"], actor.Output["Life"])
		pass.Output["LifeLeechInstantRate"] = pass.Output["LifeLeechInstant"] * hitRate
		pass.Output["EnergyShieldLeechDuration"], pass.Output["EnergyShieldLeechInstances"] = getLeechInstances(pass.Output["EnergyShieldLeech"], actor.Output["EnergyShield"])
		pass.Output["EnergyShieldLeechInstantRate"] = pass.Output["EnergyShieldLeechInstant"] * hitRate
		pass.Output["ManaLeechDuration"], pass.Output["ManaLeechInstances"] = getLeechInstances(pass.Output["ManaLeech"], actor.Output["Mana"])
		pass.Output["ManaLeechInstantRate"] = pass.Output["ManaLeechInstant"] * hitRate

		// Calculate gain on hit
		//pob:block:src/Modules/CalcOffence.lua:3669-3681
		if utils.HasTrue(skillFlags, SkillFlagMine) || utils.HasTrue(skillFlags, SkillFlagTrap) || utils.HasTrue(skillFlags, SkillFlagTotem) {
			pass.Output["LifeOnHit"] = 0
			pass.Output["EnergyShieldOnHit"] = 0
			pass.Output["ManaOnHit"] = 0
		} else {
			pass.Output["LifeOnHit"] = 0
			if !skillModList.Flag(pass.Config, "CannotGainLife") && !skillModList.Flag(pass.Config, "CannotRecoverLifeOutsideLeech") {
				pass.Output["LifeOnHit"] = skillModList.Sum(mod.TypeBase, pass.Config, "LifeOnHit") + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfLifeOnHit")
			}
			pass.Output["EnergyShieldOnHit"] = 0
			if !skillModList.Flag(pass.Config, "CannotGainEnergyShield") {
				pass.Output["EnergyShieldOnHit"] = skillModList.Sum(mod.TypeBase, pass.Config, "EnergyShieldOnHit") + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfEnergyShieldOnHit")
			}
			pass.Output["ManaOnHit"] = 0
			if !skillModList.Flag(pass.Config, "CannotGainMana") {
				pass.Output["ManaOnHit"] = skillModList.Sum(mod.TypeBase, pass.Config, "ManaOnHit") + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfManaOnHit")
			}
		}
		pass.Output["LifeOnHitRate"] = pass.Output["LifeOnHit"] * hitRate
		pass.Output["EnergyShieldOnHitRate"] = pass.Output["EnergyShieldOnHit"] * hitRate
		pass.Output["ManaOnHitRate"] = pass.Output["ManaOnHit"] * hitRate

		//pob:block:src/Modules/CalcOffence.lua:3683-3692
		if utils.HasTrue(skillFlags, SkillFlagMine) || utils.HasTrue(skillFlags, SkillFlagTrap) || utils.HasTrue(skillFlags, SkillFlagTotem) {
			pass.Output["LifeOnKill"] = 0
			pass.Output["EnergyShieldOnKill"] = 0
			pass.Output["ManaOnKill"] = 0
		} else {
			if skillModList.Flag(pass.Config, "CannotGainLife") || skillModList.Flag(pass.Config, "CannotRecoverLifeOutsideLeech") {
				pass.Output["LifeOnKill"] = 0
			} else {
				pass.Output["LifeOnKill"] = math.Floor(skillModList.Sum(mod.TypeBase, pass.Config, "LifeOnKill"))
			}
			if skillModList.Flag(pass.Config, "CannotGainEnergyShield") {
				pass.Output["EnergyShieldOnKill"] = 0
			} else {
				pass.Output["EnergyShieldOnKill"] = math.Floor(skillModList.Sum(mod.TypeBase, pass.Config, "EnergyShieldOnKill"))
			}
			if skillModList.Flag(pass.Config, "CannotGainMana") {
				pass.Output["ManaOnKill"] = 0
			} else {
				pass.Output["ManaOnKill"] = math.Floor(skillModList.Sum(mod.TypeBase, pass.Config, "ManaOnKill"))
			}
		}

		//pob:block:src/Modules/CalcOffence.lua:3694-3697
		pass.Output["EnemyLifeRegen"] = enemyDB.Sum(mod.TypeIncrease, pass.Config, "LifeRegen")
		pass.Output["EnemyManaRegen"] = enemyDB.Sum(mod.TypeIncrease, pass.Config, "ManaRegen")
		pass.Output["EnemyEnergyShieldRegen"] = enemyDB.Sum(mod.TypeIncrease, pass.Config, "EnergyShieldRegen")

		// Calculate average damage and final DPS
		pass.Output["AverageHit"] = totalHitAvg*(1-pass.Output["CritChance"]/100) + totalCritAvg*pass.Output["CritChance"]/100
		//pob:block:src/Modules/CalcOffence.lua:3701-3706
		if skillFlags[SkillFlagMonsterExplode] {
			pass.Output["AverageHitToMonsterLifePercentage"] = pass.Output["AverageHit"] / monsterLife * 100
			if skillData.HitChanceIsExplodeChance {
				pass.Output["HitChance"] = pass.Output["ExplodeChance"]
			}
		}
		pass.Output["AverageDamage"] = pass.Output["AverageHit"] * pass.Output["HitChance"] / 100

		//pob:block:src/Modules/CalcOffence.lua:2799-2800
		actor.Output["AverageBurstHits"] = utils.OrDefault(pass.Output["AverageBurstHits"], 1)
		//pob:src/Modules/CalcOffence.lua:2800-2801
		repeatPenalty := float64(1)
		if skillModList.Flag(nil, "HasSeals") && activeSkill.SkillTypes[data.SkillTypeCanRapidFire] &&
			!skillModList.Flag(nil, "NoRepeatBonuses") {
			repeatPenalty = calclib.Mod(skillModList, skillCfg, "SealRepeatPenalty")
		}
		actor.Output["AverageBurstDamage"] = pass.Output["AverageDamage"] +
			pass.Output["AverageDamage"]*(actor.Output["AverageBurstHits"]-1)*repeatPenalty

		//pob:src/Modules/CalcOffence.lua:3712
		pass.Output["TotalDPS"] = pass.Output["AverageDamage"] * selectedSpeed * utils.OrDefault(skillData.DpsMultiplier, 1) * quantityMultiplier
		if breakdown != nil {
			if actor.Output["CritEffect"] != 1 {
				if skillModList.Flag(skillCfg, "LuckyHits") {
					breakdown.AddLine("AverageHit", fmt.Sprintf("(1/3) x %.2f + (2/3) x %.2f = %.1f ^8(average from non-crits)", totalHitMin, totalHitMax, totalHitAvg))
				}
				if skillModList.Flag(skillCfg, "CritLucky") || skillModList.Flag(skillCfg, "LuckyHits") {
					breakdown.AddLine("AverageHit", fmt.Sprintf("(1/3) x %.2f + (2/3) x %.2f = %.1f ^8(average from crits)", totalCritMin, totalCritMax, totalCritAvg))
					breakdown.AddLine("AverageHit", "")
				}
				breakdown.AddLine("AverageHit", fmt.Sprintf("%.1f x (1 - %.4f) ^8(damage from non-crits)", totalHitAvg, actor.Output["CritChance"]/100))
				breakdown.AddLine("AverageHit", fmt.Sprintf("+ %.1f x %.4f ^8(damage from crits)", totalCritAvg, actor.Output["CritChance"]/100))
				breakdown.AddLine("AverageHit", fmt.Sprintf("= %.1f", actor.Output["AverageHit"]))
			}
			if isAttack {
				breakdown.AddLine("AverageDamage", fmt.Sprintf("%s:", pass.Label))
				breakdown.AddLine("AverageDamage", fmt.Sprintf("%.1f ^8(average hit)", actor.Output["AverageHit"]))
				breakdown.AddLine("AverageDamage", fmt.Sprintf("x %.2f ^8(chance to hit)", actor.Output["HitChance"]/100))
				breakdown.AddLine("AverageDamage", fmt.Sprintf("= %.1f", actor.Output["AverageDamage"]))
			}
		}

		// Calculate PvP values
		//pob:block:src/Modules/CalcOffence.lua:3778-3865
		if env.Build.GetBooleanOption("PvpScaling") {
			var PvpTvalue float64
			if override, ok := env.Build.GetNumberOptionOK("multiplierPvpTvalueOverride"); ok {
				PvpTvalue = override / 1000
			} else {
				if skillData.Cooldown != 0 {
					PvpTvalue = skillData.Cooldown
				} else if skillFlags[SkillFlagMine] {
					PvpTvalue = utils.OrDefault(output["MineLayingTime"], 1) / globalOutput["ActionSpeedMod"]
				} else if skillFlags[SkillFlagTrap] {
					PvpTvalue = utils.OrDefault(output["TrapThrowingTime"], 1) / globalOutput["ActionSpeedMod"]
				} else {
					PvpTvalue = 1 / (selectedSpeed / globalOutput["ActionSpeedMod"]) * skillModList.More(cfg, "PvpTvalueMultiplier")
				}
				if PvpTvalue > 2147483647 {
					PvpTvalue = 1
				}
			}
			PvpMultiplier := skillModList.More(cfg, "PvpDamageMultiplier")

			PvpNonElemental1 := float64(data.PvpNonElemental1)
			PvpNonElemental2 := float64(data.PvpNonElemental2)
			PvpElemental1 := float64(data.PvpElemental1)
			PvpElemental2 := float64(data.PvpElemental2)

			percentageNonElemental := (output["PhysicalHitAverage"] + output["ChaosHitAverage"]) / (totalHitMin + totalHitMax) * 2
			percentageElemental := 1 - percentageNonElemental
			portionNonElemental := math.Pow(output["AverageHit"]/PvpTvalue/PvpNonElemental2, PvpNonElemental1) * PvpTvalue * PvpNonElemental2 * percentageNonElemental
			portionElemental := math.Pow(output["AverageHit"]/PvpTvalue/PvpElemental2, PvpElemental1) * PvpTvalue * PvpElemental2 * percentageElemental
			output["PvpAverageHit"] = (portionNonElemental + portionElemental) * PvpMultiplier
			output["PvpAverageDamage"] = output["PvpAverageHit"] * output["HitChance"] / 100
			output["PvpTotalDPS"] = output["PvpAverageDamage"] * selectedSpeed * utils.OrDefault(skillData.DpsMultiplier, 1)

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

			if breakdown != nil {
				percentBoth := percentageNonElemental > 0 && percentageElemental > 0
				breakdown.AddLine("PvpAverageHit", fmt.Sprintf("Pvp Formula is (D/(T*M))^E*T*%s, where D is the damage, T is the time taken,", utils.Ternary(percentBoth, "M*P", "M")))
				breakdown.AddLine("PvpAverageHit", fmt.Sprintf(" M is the multiplier%s", utils.Ternary(percentBoth, ", E is the exponent and P is the percentage of that type (ele or non ele)", " and E is the exponent")))
				if percentBoth {
					breakdown.AddLine("PvpAverageHit", fmt.Sprintf("(M= %.1f for ele and %.1f for non-ele)(E= %.2f for ele and %.2f for non-ele)", PvpElemental2, PvpNonElemental2, PvpElemental1, PvpNonElemental1))
					breakdown.AddLine("PvpAverageHit", fmt.Sprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f * %.2f = %.1f", output["AverageHit"], PvpTvalue, PvpNonElemental2, PvpNonElemental1, PvpTvalue, PvpNonElemental2, percentageNonElemental, portionNonElemental))
					breakdown.AddLine("PvpAverageHit", fmt.Sprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f * %.2f = %.1f", output["AverageHit"], PvpTvalue, PvpElemental2, PvpElemental1, PvpTvalue, PvpElemental2, percentageElemental, portionElemental))
					breakdown.AddLine("PvpAverageHit", fmt.Sprintf("(portionNonElemental + portionElemental)%s", utils.Ternary(PvpMultiplier != 1, " * PvP multiplier", " ")))
					if PvpMultiplier != 1 {
						breakdown.AddLine("PvpAverageHit", fmt.Sprintf("(%.1f + %.1f) * %g", portionNonElemental, portionElemental, PvpMultiplier))
					} else {
						breakdown.AddLine("PvpAverageHit", fmt.Sprintf("%.1f + %.1f", portionNonElemental, portionElemental))
					}
				} else if percentageElemental <= 0 {
					breakdown.AddLine("PvpAverageHit", fmt.Sprintf("(M= %.1f for non-ele)(E= %.2f for non-ele)", PvpNonElemental2, PvpNonElemental1))
					breakdown.AddLine("PvpAverageHit", fmt.Sprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f = %.1f", output["AverageHit"], PvpTvalue, PvpNonElemental2, PvpNonElemental1, PvpTvalue, PvpNonElemental2, portionNonElemental))
					if PvpMultiplier != 1 {
						breakdown.AddLine("PvpAverageHit", fmt.Sprintf("%.1f * %g ^8(portionNonElemental * PvP multiplier)", portionNonElemental, PvpMultiplier))
					}
				} else if percentageNonElemental <= 0 {
					breakdown.AddLine("PvpAverageHit", fmt.Sprintf("(M= %.1f for ele)(E= %.2f for ele)", PvpElemental2, PvpElemental1))
					breakdown.AddLine("PvpAverageHit", fmt.Sprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f = %.1f", output["AverageHit"], PvpTvalue, PvpElemental2, PvpElemental1, PvpTvalue, PvpElemental2, portionElemental))
					if PvpMultiplier != 1 {
						breakdown.AddLine("PvpAverageHit", fmt.Sprintf("%.1f * %g ^8(portionElemental * PvP multiplier)", portionElemental, PvpMultiplier))
					}
				}
				breakdown.AddLine("PvpAverageHit", fmt.Sprintf("= %.1f", output["PvpAverageHit"]))
				if isAttack {
					breakdown.AddLine("PvpAverageDamage", fmt.Sprintf("%s:", pass.Label))
					breakdown.AddLine("PvpAverageDamage", fmt.Sprintf("%.1f ^8(average pvp hit)", output["PvpAverageHit"]))
					breakdown.AddLine("PvpAverageDamage", fmt.Sprintf("x %.2f ^8(chance to hit)", output["HitChance"]/100))
					breakdown.AddLine("PvpAverageDamage", fmt.Sprintf("= %.1f", output["PvpAverageDamage"]))
				}
			}
		}
	}

	if isAttack {
		// Combine crit stats, average damage and DPS
		combineStat("PreEffectiveCritChance", "AVERAGE")
		combineStat("CritChance", "AVERAGE")
		combineStat("CritMultiplier", "AVERAGE")
		combineStat("AverageDamage", "DPS")
		//pob:src/Modules/CalcOffence.lua:3874
		combineStat("PvpAverageDamage", "DPS")
		combineStat("TotalDPS", "DPS")
		//pob:src/Modules/CalcOffence.lua:3876
		combineStat("PvpTotalDPS", "DPS")
		combineStat("LifeLeechDuration", "DPS")
		combineStat("LifeLeechInstances", "DPS")
		combineStat("LifeLeechInstant", "DPS")
		combineStat("LifeLeechInstantRate", "DPS")
		combineStat("LifeLeechInstantProportion", "DPS")
		combineStat("EnergyShieldLeechDuration", "DPS")
		combineStat("EnergyShieldLeechInstances", "DPS")
		combineStat("EnergyShieldLeechInstant", "DPS")
		combineStat("EnergyShieldLeechInstantRate", "DPS")
		combineStat("EnergyShieldLeechInstantProportion", "DPS")
		combineStat("ManaLeechDuration", "DPS")
		combineStat("ManaLeechInstances", "DPS")
		combineStat("ManaLeechInstant", "DPS")
		combineStat("ManaLeechInstantRate", "DPS")
		combineStat("ManaLeechInstantProportion", "DPS")
		combineStat("LifeOnHit", "DPS")
		combineStat("LifeOnHitRate", "DPS")
		combineStat("LifeOnKill", "DPS")
		combineStat("EnergyShieldOnHit", "DPS")
		combineStat("EnergyShieldOnHitRate", "DPS")
		combineStat("EnergyShieldOnKill", "DPS")
		combineStat("ManaOnHit", "DPS")
		combineStat("ManaOnHitRate", "DPS")
		combineStat("ManaOnKill", "DPS")
		combineStat("impaleStoredHitAvg", "DPS")

		/*
			TODO bothWeaponAttack
			if skillFlags.bothWeaponAttack {
				if breakdown != nil {
					breakdown.AverageDamage = { }
					breakdown.AddLine("AverageDamage", "Both weapons:")
					if skillData.doubleHitsWhenDualWielding {
						breakdown.AddLine("AverageDamage", fmt.Sprintf("%.1f + %.1f ^8(skill hits with both weapons at once)", actor.Output["MainHand"].AverageDamage, actor.Output["OffHand"].AverageDamage))
					} else {
						breakdown.AddLine("AverageDamage", fmt.Sprintf("(%.1f + %.1f) / 2 ^8(skill alternates weapons)", actor.Output["MainHand"].AverageDamage, actor.Output["OffHand"].AverageDamage))
					}
					breakdown.AddLine("AverageDamage", fmt.Sprintf("= %.1f", actor.Output["AverageDamage"]))
				}
			}
		*/
	}

	/*
		TODO CALCS
		if env.mode == "CALCS" {
			if skillData.ShowAverage {
				actor.Output["DisplayDamage"] = formatNumSep(fmt.Sprintf("%.1f", actor.Output["AverageDamage"])) + " average damage"
			} else {
				actor.Output["DisplayDamage"] = formatNumSep(fmt.Sprintf("%.1f", actor.Output["TotalDPS"])) + " DPS"
			}
		}
	*/
	if breakdown != nil {
		if isAttack {
			breakdown.AddLine("TotalDPS",
				fmt.Sprintf("%.1f ^8(average damage)", actor.Output["AverageDamage"]),
				utils.Ternary(actor.Output["HitSpeed"] != 0, fmt.Sprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"]), fmt.Sprintf("x %.2f ^8(attack rate)", actor.Output["Speed"])),
			)
		} else if skillData.Triggered { //pob: Modules/CalcOffence.lua:3938
			breakdown.AddLine("TotalDPS",
				fmt.Sprintf("%.1f ^8(average damage)", actor.Output["AverageDamage"]),
				utils.Ternary(actor.Output["HitSpeed"] != 0, fmt.Sprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"]), fmt.Sprintf("x %.2f ^8(trigger rate)", actor.Output["Speed"])),
			)
		} else {
			breakdown.AddLine("TotalDPS",
				fmt.Sprintf("%.1f ^8(average hit)", actor.Output["AverageDamage"]),
				utils.Ternary(actor.Output["HitSpeed"] != 0, fmt.Sprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"]), fmt.Sprintf("x %.2f ^8(cast rate)", actor.Output["Speed"])),
			)
		}
		if skillData.DpsMultiplier != 0 {
			breakdown.AddLine("TotalDPS", fmt.Sprintf("x %g ^8(DPS multiplier for this skill)", skillData.DpsMultiplier))
		}
		if quantityMultiplier > 1 {
			breakdown.AddLine("TotalDPS", fmt.Sprintf("x %g ^8(quantity multiplier for this skill)", quantityMultiplier))
		}
		breakdown.AddLine("TotalDPS", fmt.Sprintf("= %.1f", actor.Output["TotalDPS"]))
		//pob:block:src/Modules/CalcOffence.lua:3956-3974
		if env.Build.GetBooleanOption("PvpScaling") {
			rateType := "cast"
			if isAttack {
				rateType = "attack"
			} else if skillData.Triggered {
				rateType = "trigger"
			}
			breakdown.AddLine("PvpTotalDPS",
				fmt.Sprintf("%.1f ^8(average pvp hit)", actor.Output["PvpAverageDamage"]),
				utils.Ternary(actor.Output["HitSpeed"] != 0, fmt.Sprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"]), fmt.Sprintf("x %.2f ^8(%s rate)", actor.Output["Speed"], rateType)),
			)
			if skillData.DpsMultiplier != 1 {
				breakdown.AddLine("PvpTotalDPS", fmt.Sprintf("x %g ^8(DPS multiplier for this skill)", skillData.DpsMultiplier))
			}
			if quantityMultiplier > 1 {
				breakdown.AddLine("PvpTotalDPS", fmt.Sprintf("x %g ^8(quantity multiplier for this skill)", quantityMultiplier))
			}
			breakdown.AddLine("PvpTotalDPS", fmt.Sprintf("= %.1f", actor.Output["PvpTotalDPS"]))
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:3977-3979
	if activeSkill.Minion != nil {
		speed, ok := actor.Output["HitSpeed"]
		if !ok {
			speed = actor.Output["Speed"]
		}
		summonSpeed := actor.Output["SummonedMinionsPerCast"] * speed * skillData.DpsMultiplier
		skillData.SummonSpeed = &summonSpeed
	}

	actor.Output["LifeLeechInstanceRate"] = actor.Output["Life"] * data.LeechRateBase * calclib.Mod(skillModList, skillCfg, "LifeLeechRate")
	actor.Output["EnergyShieldLeechInstanceRate"] = actor.Output["EnergyShield"] * data.LeechRateBase * calclib.Mod(skillModList, skillCfg, "EnergyShieldLeechRate")
	actor.Output["ManaLeechInstanceRate"] = actor.Output["Mana"] * data.LeechRateBase * calclib.Mod(skillModList, skillCfg, "ManaLeechRate")

	actor.Output["LifeLeechRate"] = actor.Output["LifeLeechInstances"] * actor.Output["LifeLeechInstanceRate"]
	actor.Output["LifeLeechPerHit"] = actor.Output["LifeLeechInstanceRate"]
	actor.Output["EnergyShieldLeechRate"] = actor.Output["EnergyShieldLeechInstances"] * actor.Output["EnergyShieldLeechInstanceRate"]
	actor.Output["EnergyShieldLeechPerHit"] = actor.Output["EnergyShieldLeechInstanceRate"]
	actor.Output["ManaLeechRate"] = actor.Output["ManaLeechInstances"] * actor.Output["ManaLeechInstanceRate"]
	actor.Output["ManaLeechPerHit"] = actor.Output["ManaLeechInstanceRate"]

	if skillModList.Flag(nil, "ImmortalAmbition") {
		actor.Output["EnergyShieldLeechRate"] = actor.Output["EnergyShieldLeechRate"] + actor.Output["LifeLeechRate"]
		actor.Output["EnergyShieldLeechPerHit"] = actor.Output["EnergyShieldLeechPerHit"] + actor.Output["LifeLeechPerHit"]
		actor.Output["LifeLeechRate"] = 0
		actor.Output["LifeLeechPerHit"] = 0
	}
	if skillModList.Flag(nil, "UnaffectedByNonInstantLifeLeech") {
		actor.Output["LifeLeechRate"] = 0
		actor.Output["LifeLeechPerHit"] = 0
		actor.Output["LifeLeechInstances"] = 0
	}

	actor.Output["LifeLeechRate"] = actor.Output["LifeLeechInstantRate"] + min(actor.Output["LifeLeechRate"], actor.Output["MaxLifeLeechRate"])*actor.Output["LifeRecoveryRateMod"]
	actor.Output["LifeLeechPerHit"] = actor.Output["LifeLeechInstant"] + min(actor.Output["LifeLeechPerHit"], actor.Output["MaxLifeLeechRate"])*actor.Output["LifeLeechDuration"]*actor.Output["LifeRecoveryRateMod"]
	actor.Output["EnergyShieldLeechRate"] = actor.Output["EnergyShieldLeechInstantRate"] + min(actor.Output["EnergyShieldLeechRate"], actor.Output["MaxEnergyShieldLeechRate"])*actor.Output["EnergyShieldRecoveryRateMod"]
	actor.Output["EnergyShieldLeechPerHit"] = actor.Output["EnergyShieldLeechInstant"] + min(actor.Output["EnergyShieldLeechPerHit"], actor.Output["MaxEnergyShieldLeechRate"])*actor.Output["EnergyShieldLeechDuration"]*actor.Output["EnergyShieldRecoveryRateMod"]
	actor.Output["ManaLeechRate"] = actor.Output["ManaLeechInstantRate"] + min(actor.Output["ManaLeechRate"], actor.Output["MaxManaLeechRate"])*actor.Output["ManaRecoveryRateMod"]
	actor.Output["ManaLeechPerHit"] = actor.Output["ManaLeechInstant"] + min(actor.Output["ManaLeechPerHit"], actor.Output["MaxManaLeechRate"])*actor.Output["ManaLeechDuration"]*actor.Output["ManaRecoveryRateMod"]

	if skillData.ShowAverage {
		actor.Output["LifeLeechGainPerHit"] = actor.Output["LifeLeechPerHit"] + actor.Output["LifeOnHit"]
		actor.Output["EnergyShieldLeechGainPerHit"] = actor.Output["EnergyShieldLeechPerHit"] + actor.Output["EnergyShieldOnHit"]
		actor.Output["ManaLeechGainPerHit"] = actor.Output["ManaLeechPerHit"] + actor.Output["ManaOnHit"]
	} else {
		actor.Output["LifeLeechGainRate"] = actor.Output["LifeLeechRate"] + actor.Output["LifeOnHitRate"]
		actor.Output["EnergyShieldLeechGainRate"] = actor.Output["EnergyShieldLeechRate"] + actor.Output["EnergyShieldOnHitRate"]
		actor.Output["ManaLeechGainRate"] = actor.Output["ManaLeechRate"] + actor.Output["ManaOnHitRate"]
	}
	/*
		TODO breakdown.leech
		if breakdown != nil {
			if skillFlags.leechLife {
				breakdown.LifeLeech = breakdown.leech(actor.Output["LifeLeechInstant"], actor.Output["LifeLeechInstantRate"], actor.Output["LifeLeechInstances"], actor.Output["Life"], "LifeLeechRate", actor.Output["MaxLifeLeechRate"], actor.Output["LifeLeechDuration"])
			}
			if skillFlags.leechES {
				breakdown.EnergyShieldLeech = breakdown.leech(actor.Output["EnergyShieldLeechInstant"], actor.Output["EnergyShieldLeechInstantRate"], actor.Output["EnergyShieldLeechInstances"], actor.Output["EnergyShield"], "EnergyShieldLeechRate", actor.Output["MaxEnergyShieldLeechRate"], actor.Output["EnergyShieldLeechDuration"])
			}
			if skillFlags.leechMana {
				breakdown.ManaLeech = breakdown.leech(actor.Output["ManaLeechInstant"], actor.Output["ManaLeechInstantRate"], actor.Output["ManaLeechInstances"], actor.Output["Mana"], "ManaLeechRate", actor.Output["MaxManaLeechRate"], actor.Output["ManaLeechDuration"])
			}
		}
	*/
	skillFlags[SkillFlagIgniteCanStack] = skillModList.Flag(skillCfg, "IgniteCanStack")
	skillFlags[SkillFlagIgniteToChaos] = skillModList.Flag(skillCfg, "IgniteToChaos")

	for _, pass := range passList {
		globalOutput := actor.Output
		output, cfg := pass.Output, pass.Config

		//pob:block:src/Modules/CalcOffence.lua:4061-4068
		if skillModList.Flag(nil, "DotMultiplierIsCritMultiplier") {
			multi := skillModList.Sum(mod.TypeBase, cfg, "CritMultiplier")
			if override := skillModList.Override(cfg, "CritMultiplier"); override != nil {
				multi = override.Float() - 100
			}
			perfectAgony := mod.NewFloat("DotMultiplier", "OVERRIDE", multi+enemyDB.Sum(mod.TypeBase, cfg, "SelfCritMultiplier")).
				Source("Perfect Agony").Flag(mod.MFlagAilment).Tag(mod.Condition("CriticalStrike"))
			switch {
			case strings.Contains(pass.Label, "Off Hand"):
				perfectAgony = perfectAgony.Tag(mod.Condition("OffHandAttack"))
			case strings.Contains(pass.Label, "Main Hand"):
				perfectAgony = perfectAgony.Tag(mod.Condition("MainHandAttack"))
			}
			skillModList.AddMod(perfectAgony)
		}

		// Calculate chance to inflict secondary dots/status effects
		cfg.SkillCond["CriticalStrike"] = true

		if !skillFlags[SkillFlagAttack] || skillModList.Flag(cfg, "CannotBleed") {
			output["BleedChanceOnCrit"] = 0
		} else {
			output["BleedChanceOnCrit"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "BleedChance")+enemyDB.Sum(mod.TypeBase, nil, "SelfBleedChance"))
		}

		if !skillFlags[SkillFlagHit] || skillModList.Flag(cfg, "CannotPoison") {
			output["PoisonChanceOnCrit"] = 0
		} else {
			output["PoisonChanceOnCrit"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "PoisonChance")+enemyDB.Sum(mod.TypeBase, nil, "SelfPoisonChance"))
		}

		if !skillFlags[SkillFlagHit] {
			output["ImpaleChanceOnCrit"] = 0
		} else {
			output["ImpaleChanceOnCrit"] = utils.Ternary(env.ModeEffective, min(100, skillModList.Sum(mod.TypeBase, cfg, "ImpaleChance")), 0)
		}

		if !skillFlags[SkillFlagHit] || skillModList.Flag(cfg, "CannotKnockback") {
			output["KnockbackChanceOnCrit"] = 0
		} else {
			output["KnockbackChanceOnCrit"] = skillModList.Sum(mod.TypeBase, cfg, "EnemyKnockbackChance")
		}

		cfg.SkillCond["CriticalStrike"] = false

		if !skillFlags[SkillFlagAttack] || skillModList.Flag(cfg, "CannotBleed") {
			output["BleedChanceOnHit"] = 0
		} else {
			output["BleedChanceOnHit"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "BleedChance")+enemyDB.Sum(mod.TypeBase, nil, "SelfBleedChance"))
		}

		if !skillFlags[SkillFlagHit] || skillModList.Flag(cfg, "CannotPoison") {
			output["PoisonChanceOnHit"] = 0
			output["ChaosPoisonChance"] = 0
		} else {
			output["PoisonChanceOnHit"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "PoisonChance")+enemyDB.Sum(mod.TypeBase, nil, "SelfPoisonChance"))
			output["ChaosPoisonChance"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "ChaosPoisonChance"))
		}

		for _, ailment := range data.ElementalAilment("").Values() {
			chance := skillModList.Sum(mod.TypeBase, cfg, "Enemy"+string(ailment)+"Chance") + enemyDB.Sum(mod.TypeBase, nil, "Self"+string(ailment)+"Chance")
			if ailment == data.AilmentChill {
				chance = 100
			}

			//pob:src/Modules/CalcOffence.lua:4114-4118
			if ailment == data.AilmentIgnite && modDB.Flag(nil, "IgniteCanScorch") {
				output["ScorchChance"] = min(100, chance)
				skillModList.AddMod(mod.NewFloat("EnemyScorchChance", mod.TypeBase, chance).Source("Ignite Chance"))
			}

			if skillFlags[SkillFlagHit] && !skillModList.Flag(cfg, "Cannot"+string(ailment)) {
				output[string(ailment)+"ChanceOnHit"] = min(100, chance)

				if skillModList.Flag(cfg, "CritsDontAlways"+string(ailment)) || // e.g. Painseeker
					(ailment.IsNonDamaging() && data.NonDamagingAilments[ailment].Alt && !skillModList.Flag(cfg, "CritAlwaysAltAilments")) { // e.g. Secrets of Suffering
					output[string(ailment)+"ChanceOnCrit"] = output[string(ailment)+"ChanceOnHit"]
				} else {
					output[string(ailment)+"ChanceOnCrit"] = 100
				}
			} else {
				output[string(ailment)+"ChanceOnHit"] = 0
				output[string(ailment)+"ChanceOnCrit"] = 0
			}

			//pob:src/Modules/CalcOffence.lua:4131-4134
			if ailment == data.AilmentScorch && modDB.Flag(nil, "IgniteCanScorch") {
				avoidPercent := enemyDB.Sum(mod.TypeBase, nil, "Avoid"+string(ailment), "AvoidAilments", "AvoidElementalAilments")
				if enemyDB.Flag(nil, string(ailment)+"Immune", "ElementalAilmentImmune") {
					avoidPercent = 100
				}
				if 1-math.Min(avoidPercent, 100)/100 != 0 {
					output["ScorchChanceOnCrit"] = 100
				}
			}

			ailmentChance := output[string(ailment)+"ChanceOnHit"]
			if !skillModList.Flag(cfg, "NeverCrit") {
				ailmentChance += output[string(ailment)+"ChanceOnCrit"]
			}

			if ailmentChance > 0 {
				skillFlags[SkillFlag("inflict"+string(ailment))] = true
			}
		}

		if !skillFlags[SkillFlagHit] || skillModList.Flag(cfg, "CannotKnockback") {
			output["KnockbackChanceOnHit"] = 0
		} else {
			output["KnockbackChanceOnHit"] = skillModList.Sum(mod.TypeBase, cfg, "EnemyKnockbackChance")
		}

		output["ImpaleChance"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "ImpaleChance"))

		if skillModList.Sum(mod.TypeBase, cfg, "FireExposureChance") > 0 {
			skillFlags["applyFireExposure"] = true
		}

		if skillModList.Sum(mod.TypeBase, cfg, "ColdExposureChance") > 0 {
			skillFlags["applyColdExposure"] = true
		}

		if skillModList.Sum(mod.TypeBase, cfg, "LightningExposureChance") > 0 {
			skillFlags["applyLightningExposure"] = true
		}

		if env.ModeEffective {
			//pob:src/Modules/CalcOffence.lua:4107-4113
			elemental := make(map[data.Ailment]bool, 7)
			for _, a := range data.ElementalAilment("").Values() {
				elemental[a] = true
			}
			for _, ailment := range data.Ailment("").Values() {
				avoid := enemyDB.Sum(mod.TypeBase, nil, "Avoid"+string(ailment), "AvoidAilments", "AvoidElementalAilments")
				if elemental[ailment] && enemyDB.Flag(nil, string(ailment)+"Immune", "ElementalAilmentImmune") {
					avoid = 100
				}
				mult := 1 - math.Min(avoid, 100)/100
				output[string(ailment)+"ChanceOnHit"] = output[string(ailment)+"ChanceOnHit"] * mult
				output[string(ailment)+"ChanceOnCrit"] = output[string(ailment)+"ChanceOnCrit"] * mult
				if ailment == data.AilmentPoison {
					output["ChaosPoisonChance"] = output["ChaosPoisonChance"] * mult
				}
			}
		}

		//pob:block:src/Modules/CalcOffence.lua:4165-4170
		ailmentMode := utils.OrS(env.Build.GetStringOption("ailmentMode"), "AVERAGE")
		if ailmentMode == "CRIT" || modDB.Flag(nil, "AilmentsOnlyFromCrit") {
			for _, ailment := range data.Ailment("").Values() {
				output[string(ailment)+"ChanceOnHit"] = 0
			}
		}

		//pob:block:src/Modules/CalcOffence.lua:4175-4179
		if modDB.Flag(nil, "AilmentsAreNeverFromCrit") {
			for _, ailment := range data.Ailment("").Values() {
				if modDB.Flag(nil, "AilmentsOnlyFromCrit") {
					output[string(ailment)+"ChanceOnCrit"] = 0
				} else {
					output[string(ailment)+"ChanceOnCrit"] = output[string(ailment)+"ChanceOnHit"]
				}
			}
		}
		// Calculates normal and crit damage to be used in non-damaging ailment calculations
		calcAverageSourceDamage := func(ailment data.Ailment) (float64, float64) {
			sourceHitDmg, sourceCritDmg := float64(0), float64(0)
			for _, damageType := range data.DamageType("").Values() {
				var canApply bool
				if damageType == data.NonDamagingAilments[ailment].AssociatedType {
					canApply = !skillModList.Flag(cfg, string(damageType)+"Cannot"+string(ailment))
				} else {
					canApply = skillModList.Flag(cfg, string(damageType)+"Can"+string(ailment))
				}
				if canDeal[damageType] && canApply {
					sourceHitDmg += output[string(damageType)+"HitAverage"]
					sourceCritDmg += output[string(damageType)+"CritAverage"]
				}
			}
			return sourceHitDmg, sourceCritDmg
		}

		//pob:block:src/Modules/CalcOffence.lua:3190-3196
		calcAilmentDamage := func(typee string, sourceCritChance float64, sourceHitDmg float64, sourceCritDmg float64) float64 {
			// Calculate the inflict chance and base damage of a secondary effect (bleed/poison/ignite/shock/freeze)
			chanceOnHit, chanceOnCrit := output[typee+"ChanceOnHit"], output[typee+"ChanceOnCrit"]
			chanceFromHit := chanceOnHit * (1 - sourceCritChance/100)
			chanceFromCrit := chanceOnCrit * sourceCritChance / 100
			chance := chanceFromHit + chanceFromCrit
			output[typee+"Chance"] = chance
			baseFromHit := sourceHitDmg * chanceFromHit / (chanceFromHit + chanceFromCrit)
			baseFromCrit := sourceCritDmg * chanceFromCrit / (chanceFromHit + chanceFromCrit)
			baseVal := baseFromHit + baseFromCrit
			/*
				TODO Breakdown
				sourceMult := skillModList.More(nil, typee+"AsThoughDealing")
				if breakdown and chance != 0 {
					breakdownChance := breakdown[type+"Chance"] or { }
					breakdown[type+"Chance"] = breakdownChance
					if breakdownChance[1] {
						t_insert(breakdownChance, "")
					}
					if isAttack {
						t_insert(breakdownChance, pass.label+":")
					}
					t_insert(breakdownChance, fmt.Sprintf("Chance on Non-crit: %d%%", chanceOnHit))
					t_insert(breakdownChance, fmt.Sprintf("Chance on Crit: %d%%", chanceOnCrit))
					if chanceOnHit != chanceOnCrit {
						t_insert(breakdownChance, "Combined chance:")
						t_insert(breakdownChance, fmt.Sprintf("%d x (1 - %.4f) ^8(chance from non-crits)", chanceOnHit, actor.Output["CritChance"]/100))
						t_insert(breakdownChance, fmt.Sprintf("+ %d x %.4f ^8(chance from crits)", chanceOnCrit, actor.Output["CritChance"]/100))
						t_insert(breakdownChance, fmt.Sprintf("= %.2f", chance))
					}
				}
				if breakdown and baseVal > 0 {
					breakdownDPS := breakdown[type+"DPS"] or { }
					breakdown[type+"DPS"] = breakdownDPS
					if breakdownDPS[1] {
						t_insert(breakdownDPS, "")
					}
					if isAttack {
						t_insert(breakdownDPS, pass.label+":")
					}
					if sourceHitDmg == sourceCritDmg {
						t_insert(breakdownDPS, "Total damage:")
						t_insert(breakdownDPS, fmt.Sprintf("%.1f ^8(source damage)",sourceHitDmg))
						if sourceMult > 1 {
							t_insert(breakdownDPS, fmt.Sprintf("x %.2f ^8(inflicting as though dealing more damage)", sourceMult))
							t_insert(breakdownDPS, fmt.Sprintf("= %.1f", baseVal * sourceMult))
						}
					} else {
						if baseFromHit > 0 {
							t_insert(breakdownDPS, "Damage from Non-crits:")
							t_insert(breakdownDPS, fmt.Sprintf("%.1f ^8(source damage from non-crits)", sourceHitDmg))
							t_insert(breakdownDPS, fmt.Sprintf("x %.3f ^8(portion of instances created by non-crits)", chanceFromHit / (chanceFromHit + chanceFromCrit)))
							if sourceMult == 1 or baseFromCrit != 0 {
								t_insert(breakdownDPS, fmt.Sprintf("= %.1f", baseFromHit))
							}
						}
						if baseFromCrit > 0 {
							t_insert(breakdownDPS, "Damage from Crits:")
							t_insert(breakdownDPS, fmt.Sprintf("%.1f ^8(source damage from crits)", sourceCritDmg))
							t_insert(breakdownDPS, fmt.Sprintf("x %.3f ^8(portion of instances created by crits)", chanceFromCrit / (chanceFromHit + chanceFromCrit)))
							if sourceMult == 1 or baseFromHit != 0 {
								t_insert(breakdownDPS, fmt.Sprintf("= %.1f", baseFromCrit))
							}
						}
						if baseFromHit > 0 and baseFromCrit > 0 {
							t_insert(breakdownDPS, "Total damage:")
							t_insert(breakdownDPS, fmt.Sprintf("%.1f + %.1f", baseFromHit, baseFromCrit))
							if sourceMult == 1 {
								t_insert(breakdownDPS, fmt.Sprintf("= %.1f", baseVal))
							}
						}
						if sourceMult > 1 {
							t_insert(breakdownDPS, fmt.Sprintf("x %.2f ^8(inflicting as though dealing more damage)", sourceMult))
							t_insert(breakdownDPS, fmt.Sprintf("= %.1f", baseVal * sourceMult))
						}
					}
				}
			*/
			return baseVal
		}

		// Calculate bleeding chance and damage
		if canDeal[data.DamageTypePhysical] && (output["BleedChanceOnHit"]+output["BleedChanceOnCrit"]) > 0 {
			badIdea := ailmentSkillCond(skillCfg, cfg)

			//pob:src/Modules/CalcOffence.lua:3271-3280
			dotCfg := &moddb.ListCfg{
				SkillName:       skillCfg.SkillName,
				SkillPart:       skillCfg.SkillPart,
				SkillTypes:      skillCfg.SkillTypes,
				SummonSkillName: skillCfg.SummonSkillName,
				SlotName:        skillCfg.SlotName,
				Flags:           utils.Ptr(mod.MFlagDot | mod.MFlagAilment | (cfg.Flags.Get() & mod.MFlagWeaponMask) | utils.Ternary((cfg.Flags.Get()&mod.MFlagMelee) != 0, mod.MFlagMeleeHit, 0)),
				KeywordFlags:    utils.Ptr((cfg.KeywordFlags.Get() & ^mod.KeywordFlagHit) | mod.KeywordFlagBleed | mod.KeywordFlagAilment | mod.KeywordFlagPhysicalDot),
				SkillCond:       badIdea,
				SkillDist:       skillCfg.SkillDist,
				SkillDistSet:    skillCfg.SkillDistSet,
			}

			if strings.Contains(pass.Label, "Off Hand") {
				activeSkill.OHBleedCfg = dotCfg
			} else {
				activeSkill.BleedCfg = dotCfg
			}

			sourceHitDmg := float64(0)
			sourceCritDmg := float64(0)
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.BleedPhysical = { damageTypes = { } }
				}
			*/

			// For bleeds we will be using a weighted average calculation

			configStacks := enemyDB.Sum(mod.TypeBase, nil, "Multiplier:BleedStacks")
			maxStacks := skillModList.Sum(mod.TypeBase, cfg, "BleedStacksMax")
			if skillModList.Override(cfg, "BleedStacksMax") != nil {
				maxStacks = skillModList.Override(cfg, "BleedStacksMax").Float()
			}
			globalOutput["BleedStacksMax"] = maxStacks
			durationBase := utils.Ternary(skillData.BleedDurationIsSkillDuration, skillData.Duration, data.BleedDurationBase)
			if override := skillModList.Override(dotCfg, "BleedDurationBase"); override != nil {
				durationBase = override.Float()
			}

			names := []string{"EnemyBleedDuration", "EnemyAilmentDuration", "DamagingAilmentDuration"}
			if skillData.BleedIsSkillEffect {
				names = append(names, "Duration")
			}
			//pob:src/Modules/CalcOffence.lua:4314-4315
			durationMod := math.Max(CalcMod(skillModList, dotCfg, names...)*CalcMod(enemyDB, nil, "SelfBleedDuration", "SelfAilmentDuration")/CalcMod(enemyDB, dotCfg, "BleedExpireRate"), 0)
			rateMod := CalcMod(skillModList, cfg, "BleedFaster") + enemyDB.Sum(mod.TypeIncrease, nil, "SelfBleedFaster")/100
			globalOutput["BleedDuration"] = durationBase * durationMod / rateMod * debuffDurationMult
			//pob:block:src/Modules/CalcOffence.lua:4319-4340
			bleedChance := output["BleedChanceOnHit"]/100*(1-output["CritChance"]/100) +
				output["BleedChanceOnCrit"]/100*output["CritChance"]/100
			bleedStacks := output["HitChance"] / 100 * bleedChance * utils.OrDefault(skillData.DpsMultiplier, 1)
			hitRate := utils.OrF(globalOutput["HitSpeed"], globalOutput["Speed"])
			if hitRate > 0 {
				// assume skills with no cast, attack, or cooldown time are single cast
				bleedStacks = bleedStacks * globalOutput["BleedDuration"] * hitRate
			}
			if skillFlags[SkillFlagTotem] {
				activeTotems := utils.Or(env.ModDB.Override(nil, "TotemsSummoned"),
					skillModList.Sum(mod.TypeBase, skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit"))
				bleedStacks = bleedStacks * activeTotems
			}
			if configStacks > 0 {
				bleedStacks = configStacks
			}

			bleedStackPotential := bleedStacks / maxStacks
			if o := skillModList.Override(nil, "BleedStackPotentialOverride"); o != nil {
				bleedStackPotential = o.Float() / maxStacks
			}
			globalOutput["BleedStackPotential"] = bleedStackPotential

			//pob:block:src/Modules/CalcOffence.lua:4363-4372
			bleedRollAverage := float64(50)
			if bleedStackPotential > 1 {
				bleedRollAverage = (bleedStacks - (maxStacks-1)/2) / (bleedStacks + 1) * 100
			}
			globalOutput["BleedRollAverage"] = bleedRollAverage
			/*
				TODO Breakdown
				if globalBreakdown {
					globalBreakdown.BleedStackPotential = {
						fmt.Sprintf(colorCodes.CUSTOM+"NOTE: Calculation uses new Weighted Avg Ailment formula"),
						fmt.Sprintf(""),
						fmt.Sprintf("%.2f ^8(chance to hit)", actor.Output["HitChance"] / 100),
						fmt.Sprintf("* (%.2f / %.2f) ^8(BleedDuration / Attack Time)", globalOutput.BleedDuration, actor.Output["Time"]),
						fmt.Sprintf("/ %d ^8(max number of stacks)", maxStacks),
						fmt.Sprintf("= %.2f", globalOutput.BleedStackPotential),
					}
				}
			*/

			for _, sub_pass := range []int{1, 2} {
				dotCfg.SkillCond["CriticalStrike"] = sub_pass != 1

				output["BleedPhysicalMin"], output["BleedPhysicalMax"] = calcAilmentSourceDamage(activeSkill, output, dotCfg, breakdown, "BleedPhysical", "Physical", 0)

				minDmg, maxDmg := output["BleedPhysicalMin"], output["BleedPhysicalMax"]
				//pob:block:src/Modules/CalcOffence.lua:4395-4411
				bleedDotMulti := func() float64 {
					if override := skillModList.Override(dotCfg, "DotMultiplier"); override != nil {
						return 1 + override.Float()/100
					}
					return 1 + skillModList.Sum(mod.TypeBase, dotCfg, "DotMultiplier", "PhysicalDotMultiplier")/100
				}
				if sub_pass == 2 {
					if modDB.Flag(nil, "AilmentsAreNeverFromCrit") {
						dotCfg.SkillCond["CriticalStrike"] = false
						output["CritBleedDotMulti"] = bleedDotMulti()
						dotCfg.SkillCond["CriticalStrike"] = true
					} else {
						output["CritBleedDotMulti"] = bleedDotMulti()
					}
					sourceMinCritDmg := minDmg * output["CritBleedDotMulti"]
					sourceMaxCritDmg := maxDmg * output["CritBleedDotMulti"]
					sourceCritDmg = sourceMinCritDmg + (sourceMaxCritDmg-sourceMinCritDmg)*bleedRollAverage/100
				} else {
					output["BleedDotMulti"] = bleedDotMulti()
					sourceMinHitDmg := minDmg * output["BleedDotMulti"]
					sourceMaxHitDmg := maxDmg * output["BleedDotMulti"]
					sourceHitDmg = sourceMinHitDmg + (sourceMaxHitDmg-sourceMinHitDmg)*bleedRollAverage/100
				}
			}

			/*
				TODO Breakdown
				if globalBreakdown {
					if sourceHitDmg == sourceCritDmg {
						globalBreakdown.BleedDPS = {
							fmt.Sprintf(colorCodes.CUSTOM+"NOTE: Calculation uses new Weighted Avg Ailment formula"),
							fmt.Sprintf(""),
							fmt.Sprintf("Dmg Derivation:"),
							fmt.Sprintf("(%.2f + (%.2f - %.2f) ^8(min source physical + (max source physical - min source physical)", actor.Output["BleedPhysicalMin"], actor.Output["BleedPhysicalMax"], actor.Output["BleedPhysicalMin"]),
							fmt.Sprintf("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", bleedStacks),
							fmt.Sprintf("* %.2f ^8(Bleed DoT Multi)", actor.Output["BleedDotMulti"]),
							fmt.Sprintf("= %.2f", sourceHitDmg),
						}
					} else {
						globalBreakdown.BleedDPS = {
							fmt.Sprintf(colorCodes.CUSTOM+"NOTE: Calculation uses new Weighted Avg Ailment formula"),
							fmt.Sprintf(""),
							fmt.Sprintf("Non-Crit Dmg Derivation:"),
							fmt.Sprintf("(%.2f + (%.2f - %.2f) ^8(min source physical + (max source physical - min source physical)", actor.Output["BleedPhysicalMin"], actor.Output["BleedPhysicalMax"], actor.Output["BleedPhysicalMin"]),
							fmt.Sprintf("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", bleedStacks),
							fmt.Sprintf("* %.2f ^8(Bleed DoT Multi for Non-Crit)", actor.Output["BleedDotMulti"]),
							fmt.Sprintf("= %.2f", sourceHitDmg),
							fmt.Sprintf(""),
							fmt.Sprintf("Crit Dmg Derivation:"),
							fmt.Sprintf("(%.2f + (%.2f - %.2f) ^8(min source physical + (max source physical - min source physical)", actor.Output["BleedPhysicalMin"], actor.Output["BleedPhysicalMax"], actor.Output["BleedPhysicalMin"]),
							fmt.Sprintf("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", bleedStacks),
							fmt.Sprintf("* %.2f ^8(Bleed DoT Multi for Crit)", actor.Output["CritBleedDotMulti"]),
							fmt.Sprintf("= %.2f", sourceCritDmg),
						}
					}
				}
			*/

			basePercent := utils.Ternary(skillData.BleedBasePercent != 0, skillData.BleedBasePercent, data.BleedPercentBase)
			// over-stacking bleed stacks increases the chance a critical bleed is present
			//pob:block:src/Modules/CalcOffence.lua:3363-3365
			//pob:block:src/Modules/CalcOffence.lua:4446
			ailmentCritChance := 100 * (1 - math.Pow(1-output["CritChance"]/100, max(bleedStackPotential, 1)))
			baseVal := calcAilmentDamage("Bleed", ailmentCritChance, sourceHitDmg, sourceCritDmg) * basePercent / 100 * output["RuthlessBlowAilmentEffect"] * output["FistOfWarDamageEffect"] * globalOutput["AilmentWarcryEffect"]
			if baseVal > 0 {
				skillFlags[SkillFlagBleed] = true
				skillFlags[SkillFlagDuration] = true
				effMult := float64(1)
				if env.ModeEffective {
					resist := min(max(0, enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction")), data.EnemyPhysicalDamageReductionCap)
					takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, "DamageTaken", "DamageTakenOverTime", "PhysicalDamageTaken", "PhysicalDamageTakenOverTime")
					takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "PhysicalDamageTaken", "PhysicalDamageTakenOverTime")
					effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
					globalOutput["BleedEffMult"] = effMult
					/*
						TODO Breakdown
						if breakdown and effMult != 1 {
							globalBreakdown.BleedEffMult = breakdown.effMult("Physical", resist, 0, takenInc, effMult, takenMore)
						}
					*/
				}

				effectMod := CalcMod(skillModList, dotCfg, "AilmentEffect")
				//pob:block:src/Modules/CalcOffence.lua:4466-4477
				activeBleeds := min(bleedStacks, maxStacks)
				output["BaseBleedDPS"] = baseVal * effectMod * rateMod * activeBleeds * effMult
				output["BleedDPS"] = min(output["BaseBleedDPS"], data.DotDpsCap)
				globalOutput["BleedStacks"] = bleedStacks
				globalOutput["BleedDamage"] = output["BaseBleedDPS"] * globalOutput["BleedDuration"]
			}
		}
		// Calculate poison chance and damage
		if canDeal[data.DamageTypeChaos] && (output["PoisonChanceOnHit"]+output["PoisonChanceOnCrit"]+output["ChaosPoisonChance"]) > 0 {
			badIdea := ailmentSkillCond(skillCfg, cfg)

			//pob:src/Modules/CalcOffence.lua:3467-3476
			dotCfg := &moddb.ListCfg{
				SkillName:       skillCfg.SkillName,
				SkillPart:       skillCfg.SkillPart,
				SkillTypes:      skillCfg.SkillTypes,
				SummonSkillName: skillCfg.SummonSkillName,
				SlotName:        skillCfg.SlotName,
				Flags:           utils.Ptr(mod.MFlagDot | mod.MFlagAilment | (cfg.Flags.Get() & mod.MFlagWeaponMask) | utils.Ternary((cfg.Flags.Get()&mod.MFlagMelee) != 0, mod.MFlagMeleeHit, 0)),
				KeywordFlags:    utils.Ptr((cfg.KeywordFlags.Get() & ^mod.KeywordFlagHit) | mod.KeywordFlagPoison | mod.KeywordFlagAilment | mod.KeywordFlagChaosDot),
				SkillCond:       badIdea,
				SkillDist:       skillCfg.SkillDist,
				SkillDistSet:    skillCfg.SkillDistSet,
			}

			if strings.Contains(pass.Label, "Off Hand") {
				activeSkill.OHPoisonCfg = dotCfg
			} else {
				activeSkill.PoisonCfg = dotCfg
			}

			rateMod := CalcMod(skillModList, cfg, "PoisonFaster") + enemyDB.Sum(mod.TypeIncrease, nil, "SelfPoisonFaster")/100

			durationBase := float64(data.PoisonDurationBase)
			switch {
			case skillModList.Override(dotCfg, "PoisonDurationBase") != nil:
				durationBase = utils.Or(skillModList.Override(dotCfg, "PoisonDurationBase"), 0)
			case skillData.PoisonDurationIsSkillDuration:
				durationBase = skillData.Duration
			}

			names := []string{"EnemyPoisonDuration", "EnemyAilmentDuration", "DamagingAilmentDuration"}
			if skillData.PoisonIsSkillEffect {
				names = append(names, "Duration")
			}
			durationMod := math.Max(CalcMod(skillModList, dotCfg, names...)*CalcMod(enemyDB, nil, "SelfPoisonDuration", "SelfAilmentDuration"), 0)
			globalOutput["PoisonDuration"] = durationBase * durationMod / rateMod * debuffDurationMult

			//pob:block:src/Modules/CalcOffence.lua:4589-4622
			chaosPoisonChance := utils.Ternary(output["ChaosHitAverage"] > 0, output["ChaosPoisonChance"], 0)
			poisonChanceOnHit := min(100, output["PoisonChanceOnHit"]+chaosPoisonChance)
			poisonChanceOnCrit := min(100, output["PoisonChanceOnCrit"]+chaosPoisonChance)
			poisonChance := poisonChanceOnHit/100*(1-output["CritChance"]/100) + poisonChanceOnCrit/100*output["CritChance"]/100

			additionalPoisonStacks := 1.0
			if !skillModList.Flag(nil, "CannotMultiplePoison") {
				additionalPoisonStacks = 1 +
					min(skillModList.Sum(mod.TypeBase, cfg, "AdditionalPoisonChance")/100, 1) +
					skillModList.Sum(mod.TypeBase, cfg, "AdditionalPoisonStacks")
			}

			poisonStacks := output["HitChance"] / 100 * poisonChance * additionalPoisonStacks *
				utils.OrDefault(skillData.DpsMultiplier, 1) *
				utils.OrDefault(skillData.StackMultiplier, 1) * quantityMultiplier
			if speed := utils.OrDefault(globalOutput["HitSpeed"], globalOutput["Speed"]); speed > 0 {
				poisonStacks = poisonStacks * globalOutput["PoisonDuration"] * speed
			}
			poisonOnEnemy, _ := env.Build.GetNumberOptionOK("multiplierPoisonOnEnemy")
			if poisonStacks < additionalPoisonStacks && poisonOnEnemy == 0 {
				skillModList.AddMod(mod.NewFlag("Condition:NonPoisonedOnly", true).Source("Calculation"))
			}

			sourceHitDmg := float64(0)
			sourceCritDmg := float64(0)
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.PoisonPhysical = { damageTypes = { } }
					breakdown.PoisonLightning = { damageTypes = { } }
					breakdown.PoisonCold = { damageTypes = { } }
					breakdown.PoisonFire = { damageTypes = { } }
					breakdown.PoisonChaos = { damageTypes = { } }
				}
			*/

			for _, sub_pass := range []int{1, 2} {
				dotCfg.SkillCond["CriticalStrike"] = sub_pass != 1

				subBreakdown := utils.Ternary(sub_pass == 1, breakdown, nil)

				totalMin, totalMax := float64(0), float64(0)

				chaosMin, chaosMax := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "PoisonChaos", data.DamageTypeChaos, 0)
				output["PoisonChaosMin"] = chaosMin
				output["PoisonChaosMax"] = chaosMax
				totalMin += chaosMin
				totalMax += chaosMax

				nonChaosMult := float64(1)
				if output["ChaosPoisonChance"] > 0 && output["PoisonChaosMax"] > 0 {
					// Additional chance for chaos
					chance := utils.Ternary(sub_pass == 2, "PoisonChanceOnCrit", "PoisonChanceOnHit")
					chaosChance := min(100, output[chance]+output["ChaosPoisonChance"])
					nonChaosMult = output[chance] / chaosChance
					output[chance] = chaosChance
				}

				if canDeal[data.DamageTypeLightning] && skillModList.Flag(cfg, "LightningCanPoison") {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "PoisonLightning", data.DamageTypeLightning, data.DamageTypeFlags[data.DamageTypeChaos])
					output["PoisonLightningMin"] = minDmg
					output["PoisonLightningMax"] = maxDmg
					totalMin += minDmg * nonChaosMult
					totalMax += maxDmg * nonChaosMult
				}

				if canDeal[data.DamageTypeCold] && skillModList.Flag(cfg, "ColdCanPoison") {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "PoisonCold", data.DamageTypeCold, data.DamageTypeFlags[data.DamageTypeChaos])
					output["PoisonColdMin"] = minDmg
					output["PoisonColdMax"] = maxDmg
					totalMin += minDmg * nonChaosMult
					totalMax += maxDmg * nonChaosMult
				}

				if canDeal[data.DamageTypeFire] && skillModList.Flag(cfg, "FireCanPoison") {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "PoisonFire", data.DamageTypeFire, data.DamageTypeFlags[data.DamageTypeChaos])
					output["PoisonFireMin"] = minDmg
					output["PoisonFireMax"] = maxDmg
					totalMin += minDmg * nonChaosMult
					totalMax += maxDmg * nonChaosMult
				}

				if canDeal[data.DamageTypePhysical] {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "PoisonPhysical", data.DamageTypePhysical, data.DamageTypeFlags[data.DamageTypeChaos])
					output["PoisonPhysicalMin"] = minDmg
					output["PoisonPhysicalMax"] = maxDmg
					totalMin += minDmg * nonChaosMult
					totalMax += maxDmg * nonChaosMult
				}

				poisonDotMulti := func() float64 {
					if override := skillModList.Override(dotCfg, "DotMultiplier"); override != nil {
						return 1 + override.Float()/100
					}
					return 1 + skillModList.Sum(mod.TypeBase, dotCfg, "DotMultiplier", "ChaosDotMultiplier")/100
				}
				if sub_pass == 2 {
					if modDB.Flag(nil, "AilmentsAreNeverFromCrit") {
						dotCfg.SkillCond["CriticalStrike"] = false
						output["CritPoisonDotMulti"] = poisonDotMulti()
						dotCfg.SkillCond["CriticalStrike"] = true
					} else {
						output["CritPoisonDotMulti"] = poisonDotMulti()
					}
					sourceCritDmg = (totalMin + totalMax) / 2 * output["CritPoisonDotMulti"]
				} else {
					output["PoisonDotMulti"] = poisonDotMulti()
					sourceHitDmg = (totalMin + totalMax) / 2 * output["PoisonDotMulti"]
				}
			}

			//pob:src/Modules/CalcOffence.lua:4721
			baseVal := calcAilmentDamage("Poison", output["CritChance"], sourceHitDmg, sourceCritDmg) * data.PoisonPercentBase * output["RuthlessBlowAilmentEffect"] * output["FistOfWarDamageEffect"] * globalOutput["AilmentWarcryEffect"]
			if baseVal > 0 {
				skillFlags[SkillFlagPoison] = true
				skillFlags[SkillFlagDuration] = true

				effMult := float64(1)
				if env.ModeEffective {
					resist := calcResistForType(data.DamageTypeChaos, dotCfg)
					takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
					takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
					effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
					globalOutput["PoisonEffMult"] = effMult
					/*
						TODO Breakdown
						if breakdown and effMult != 1 {
							globalBreakdown.PoisonEffMult = breakdown.effMult("Chaos", resist, 0, takenInc, effMult, takenMore)
						}
					*/
				}

				//pob:block:src/Modules/CalcOffence.lua:4737-4740
				if skillModList.Flag(nil, "Condition:NonPoisonedOnly") {
					poisonStacks = min(additionalPoisonStacks, poisonStacks)
				}
				globalOutput["PoisonStacks"] = poisonStacks

				effectMod := CalcMod(skillModList, dotCfg, "AilmentEffect")
				output["PoisonDPS"] = min(baseVal*effectMod*rateMod*effMult, data.DotDpsCap)
				output["PoisonDamage"] = output["PoisonDPS"] * globalOutput["PoisonDuration"]

				//pob:block:src/Modules/CalcOffence.lua:4750-4755
				groundMult := max(
					skillModList.Max(nil, "PoisonDpsAsCausticGround"),
					enemyDB.Max(nil, "PoisonDpsAsCausticGround"),
				)
				if groundMult > 0 {
					output["CausticGroundDPS"] = min(baseVal*effectMod*rateMod*effMult*groundMult/100, data.DotDpsCap)
				}

				if skillData.ShowAverage {
					//pob:src/Modules/CalcOffence.lua:4768
					output["TotalPoisonAverageDamage"] = output["PoisonDamage"]
					output["TotalPoisonDPS"] = output["PoisonDPS"]
				} else {
					output["TotalPoisonDPS"] = min(output["PoisonDPS"]*poisonStacks, data.DotDpsCap)
				}
				/*
					TODO Breakdown -- see CalcOffence.lua:3139-3196
				*/
			}
		}
		// Calculate ignite chance and damage
		if canDeal[data.DamageTypeFire] && (output["IgniteChanceOnHit"]+output["IgniteChanceOnCrit"]) > 0 {
			badIdea := ailmentSkillCond(skillCfg, cfg)

			//pob:src/Modules/CalcOffence.lua:3692-3701
			dotCfg := &moddb.ListCfg{
				SkillName:       skillCfg.SkillName,
				SkillPart:       skillCfg.SkillPart,
				SkillTypes:      skillCfg.SkillTypes,
				SummonSkillName: skillCfg.SummonSkillName,
				SlotName:        skillCfg.SlotName,
				Flags:           utils.Ptr(mod.MFlagDot | mod.MFlagAilment | (cfg.Flags.Get() & mod.MFlagWeaponMask) | utils.Ternary((cfg.Flags.Get()&mod.MFlagMelee) != 0, mod.MFlagMeleeHit, 0)),
				KeywordFlags:    utils.Ptr((cfg.KeywordFlags.Get() & ^mod.KeywordFlagHit) | mod.KeywordFlagIgnite | mod.KeywordFlagAilment | mod.KeywordFlagFireDot),
				SkillCond:       badIdea,
				SkillDist:       skillCfg.SkillDist,
				SkillDistSet:    skillCfg.SkillDistSet,
			}

			if strings.Contains(pass.Label, "Off Hand") {
				activeSkill.OHIgniteCfg = dotCfg
			} else {
				activeSkill.IgniteCfg = dotCfg
			}

			sourceHitDmg := float64(0)
			sourceCritDmg := float64(0)
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.IgnitePhysical = { damageTypes = { } }
					breakdown.IgniteLightning = { damageTypes = { } }
					breakdown.IgniteCold = { damageTypes = { } }
					breakdown.IgniteFire = { damageTypes = { } }
					breakdown.IgniteChaos = { damageTypes = { } }
				}
			*/

			//pob:block:src/Modules/CalcOffence.lua:4890
			globalOutput["IgniteChancePerHit"] = output["IgniteChanceOnHit"]*(1-output["CritChance"]/100) + output["IgniteChanceOnCrit"]*output["CritChance"]/100

			// For ignites we will be using a weighted average calculation
			maxStacks := float64(1)
			if skillFlags[SkillFlagIgniteCanStack] {
				if override := skillModList.Override(cfg, "IgniteStacks"); override != nil {
					maxStacks = override.Float()
				} else {
					maxStacks += skillModList.Sum(mod.TypeBase, cfg, "IgniteStacks")
				}
			}
			var overrideStackPotential *float64
			if override := skillModList.Override(nil, "IgniteStackPotentialOverride"); override != nil {
				v := override.Float() / maxStacks
				overrideStackPotential = &v
			}
			globalOutput["IgniteStacksMax"] = maxStacks

			rateMod := (CalcMod(skillModList, cfg, "IgniteBurnFaster") + enemyDB.Sum(mod.TypeIncrease, nil, "SelfIgniteBurnFaster")/100) / CalcMod(skillModList, cfg, "IgniteBurnSlower")
			durationBase := float64(data.IgniteDurationBase)
			if override := skillModList.Override(dotCfg, "IgniteDurationBase"); override != nil {
				durationBase = override.Float()
			}
			durationMod := max(CalcMod(skillModList, dotCfg, "EnemyIgniteDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration", "DamagingAilmentDuration")*CalcMod(enemyDB, nil, "SelfIgniteDuration", "SelfAilmentDuration", "SelfElementalAilmentDuration"), 0)
			globalOutput["IgniteDuration"] = durationBase * durationMod / rateMod * debuffDurationMult

			//pob:block:src/Modules/CalcOffence.lua:4905-4918
			igniteChance := output["IgniteChanceOnHit"]/100*(1-output["CritChance"]/100) +
				output["IgniteChanceOnCrit"]/100*output["CritChance"]/100
			igniteStacks := output["HitChance"] / 100 * igniteChance * utils.OrDefault(skillData.DpsMultiplier, 1)
			if !skillData.TriggeredOnDeath {
				_, hasCooldown := output["Cooldown"]
				hitSpeed := utils.OrDefault(globalOutput["HitSpeed"], globalOutput["Speed"])
				switch {
				case hasCooldown:
					igniteStacks = igniteStacks * globalOutput["IgniteDuration"] /
						max(output["Cooldown"], utils.OrDefault(output["HitTime"], output["Time"]))
				case hitSpeed > 0:
					igniteStacks = igniteStacks * globalOutput["IgniteDuration"] * hitSpeed
				}
			}
			if overrideStackPotential != nil {
				globalOutput["IgniteStackPotential"] = *overrideStackPotential
			} else {
				globalOutput["IgniteStackPotential"] = igniteStacks / maxStacks
			}

			//pob:block:src/Modules/CalcOffence.lua:4949-4958
			igniteRollAverage := float64(50)
			if globalOutput["IgniteStackPotential"] > 1 {
				igniteRollAverage = (igniteStacks - (maxStacks-1)/2) / (igniteStacks + 1) * 100
			}
			globalOutput["IgniteRollAverage"] = igniteRollAverage
			/*
				TODO Breakdown
				if globalBreakdown {
					globalBreakdown.IgniteStackPotential = {
						fmt.Sprintf(colorCodes.CUSTOM+"NOTE: Calculation uses new Weighted Avg Ailment formula"),
						fmt.Sprintf(""),
						fmt.Sprintf("(%.2f / %.2f) ^8(IgniteDuration / Cast Time)", globalOutput.IgniteDuration, output.Time),
						fmt.Sprintf("/ %d ^8(max number of stacks)", maxStacks),
						fmt.Sprintf("= %.2f", globalOutput.IgniteStackPotential),
					}
				}
			*/

			for _, sub_pass := range []int{1, 2} {
				dotCfg.SkillCond["CriticalStrike"] = sub_pass != 1

				subBreakdown := utils.Ternary(sub_pass == 1, breakdown, nil)

				totalMin, totalMax := float64(0), float64(0)

				if canDeal[data.DamageTypePhysical] && skillModList.Flag(cfg, "PhysicalCanIgnite") {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "IgnitePhysical", data.DamageTypePhysical, data.DamageTypeFlags[data.DamageTypeFire])
					output["IgnitePhysicalMin"] = minDmg
					output["IgnitePhysicalMax"] = maxDmg
					totalMin += minDmg
					totalMax += maxDmg
				}

				if canDeal[data.DamageTypeLightning] && skillModList.Flag(cfg, "LightningCanIgnite") {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "IgniteLightning", data.DamageTypeLightning, data.DamageTypeFlags[data.DamageTypeFire])
					output["IgniteLightningMin"] = minDmg
					output["IgniteLightningMax"] = maxDmg
					totalMin += minDmg
					totalMax += maxDmg
				}

				if canDeal[data.DamageTypeCold] && skillModList.Flag(cfg, "ColdCanIgnite") {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "IgniteCold", data.DamageTypeCold, data.DamageTypeFlags[data.DamageTypeFire])
					output["IgniteColdMin"] = minDmg
					output["IgniteColdMax"] = maxDmg
					totalMin += minDmg
					totalMax += maxDmg
				}

				if canDeal[data.DamageTypeFire] && !skillModList.Flag(cfg, "FireCannotIgnite") {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "IgniteFire", data.DamageTypeFire, 0)
					output["IgniteFireMin"] = minDmg
					output["IgniteFireMax"] = maxDmg
					totalMin += minDmg
					totalMax += maxDmg
				}

				if canDeal[data.DamageTypeChaos] && skillModList.Flag(cfg, "ChaosCanIgnite") {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "IgniteChaos", data.DamageTypeChaos, data.DamageTypeFlags[data.DamageTypeFire])
					output["IgniteChaosMin"] = minDmg
					output["IgniteChaosMax"] = maxDmg
					totalMin += minDmg
					totalMax += maxDmg
				}

				dotMulti := func() float64 {
					if override := skillModList.Override(dotCfg, "DotMultiplier"); override != nil {
						return 1 + override.Float()/100
					}
					return 1 + skillModList.Sum(mod.TypeBase, dotCfg, "DotMultiplier", "FireDotMultiplier")/100
				}
				if sub_pass == 2 {
					if modDB.Flag(nil, "AilmentsAreNeverFromCrit") {
						dotCfg.SkillCond["CriticalStrike"] = false
						output["CritIgniteDotMulti"] = dotMulti()
						dotCfg.SkillCond["CriticalStrike"] = true
					} else {
						output["CritIgniteDotMulti"] = dotMulti()
					}
					sourceCritDmg = (totalMin + (totalMax-totalMin)*igniteRollAverage/100) * output["CritIgniteDotMulti"]
				} else {
					output["IgniteDotMulti"] = dotMulti()
					sourceHitDmg = (totalMin + (totalMax-totalMin)*igniteRollAverage/100) * output["IgniteDotMulti"]
				}

				output["IgniteTotalMin"] = totalMin
				output["IgniteTotalMax"] = totalMax
			}

			// over-stacking ignite stacks increases the chance a critical ignite is present
			//pob:block:src/Modules/CalcOffence.lua:3822-3824
			ailmentCritChance := 100 * (1 - math.Pow(1-output["CritChance"]/100, max(1, igniteStacks)))
			baseVal := calcAilmentDamage("Ignite", ailmentCritChance, sourceHitDmg, sourceCritDmg) * data.IgnitePercentBase * output["RuthlessBlowAilmentEffect"] * output["FistOfWarDamageEffect"] * globalOutput["AilmentWarcryEffect"]
			if baseVal > 0 {
				skillFlags[SkillFlagIgnite] = true

				effMult := float64(1)
				if env.ModeEffective {
					if skillModList.Flag(cfg, "IgniteToChaos") {
						resist := calcResistForType(data.DamageTypeChaos, dotCfg)
						takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
						takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
						effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
					} else {
						resist := calcResistForType(data.DamageTypeFire, dotCfg)
						takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
						takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
						effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
					}
					globalOutput["IgniteEffMult"] = effMult
					/*
						TODO Breakdown
						if breakdown and effMult != 1 {
							globalBreakdown.IgniteEffMult = breakdown.effMult("Fire", resist, 0, takenInc, effMult, takenMore)
						}
					*/
				}

				effectMod := CalcMod(skillModList, dotCfg, "AilmentEffect")
				//pob:block:src/Modules/CalcOffence.lua:5090-5091
				activeIgnites := min(igniteStacks, maxStacks)

				output["IgniteDPS"] = min(baseVal*effectMod*rateMod*activeIgnites*effMult, data.DotDpsCap)
				globalOutput["IgniteDamage"] = output["IgniteDPS"] * globalOutput["IgniteDuration"]

				//pob:block:src/Modules/CalcOffence.lua:5098-5107
				groundMult := max(
					skillModList.Max(nil, "IgniteDpsAsBurningGround"),
					enemyDB.Max(nil, "IgniteDpsAsBurningGround"),
				)
				if groundMult > 0 {
					resist := calcResistForType(data.DamageTypeFire, dotCfg)
					takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
					takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
					fireEffMult := (1 - resist/100) * (1 + takenInc/100) * takenMore
					globalOutput["BurningGroundDPS"] = min(baseVal*effectMod*rateMod*fireEffMult*groundMult/100, data.DotDpsCap)
				}

				if skillFlags[SkillFlagIgniteCanStack] {
					output["IgniteDamage"] = output["IgniteDPS"] * globalOutput["IgniteDuration"]
					output["IgniteStacksMax"] = maxStacks
					output["TotalIgniteDPS"] = output["IgniteDPS"]
				}
				/*
					TODO Breakdown -- see CalcOffence.lua:3368-3412
				*/
			}
		}
		// Calculate non-damaging ailments effect and duration modifiers
		if activeSkill.SkillTypes[data.SkillTypeChillingArea] || activeSkill.SkillTypes[data.SkillTypeNonHitChill] {
			skillFlags["chill"] = true
			//pob:block:src/Modules/CalcOffence.lua:5256-5261
			incChill := skillModList.Sum(mod.TypeIncrease, cfg, "EnemyChillEffect")
			moreChill := skillModList.More(cfg, "EnemyChillEffect")
			output["ChillEffectMod"] = (1 + incChill/100) * moreChill * calclib.Mod(enemyDB, nil, "SelfChillEffect")

			incChillDuration := skillModList.Sum(mod.TypeIncrease, cfg, "EnemyChillDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration") +
				enemyDB.Sum(mod.TypeIncrease, nil, "SelfChillDuration", "SelfAilmentDuration", "SelfElementalAilmentDuration")
			moreChillDuration := skillModList.More(cfg, "EnemyChillDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration") *
				enemyDB.More(nil, "SelfChillDuration", "SelfAilmentDuration", "SelfElementalAilmentDuration")
			output["ChillDurationMod"] = (1 + incChillDuration/100) * moreChillDuration

			chillMax := data.NonDamagingAilments[data.AilmentChill].Max
			if override := skillModList.Override(nil, "ChillMax"); override != nil {
				chillMax = override.Float()
			}
			output["ChillSourceEffect"] = min(chillMax, math.Floor(*data.NonDamagingAilments[data.AilmentChill].Default*output["ChillEffectMod"]))
		}

		if (output["FreezeChanceOnHit"] + output["FreezeChanceOnCrit"]) > 0 {
			freezeHitDmg, freezeCritDmg := calcAverageSourceDamage(data.AilmentFreeze)
			baseVal := calcAilmentDamage("Freeze", output["CritChance"], freezeHitDmg, freezeCritDmg) * skillModList.More(cfg, "FreezeAsThoughDealing")
			if baseVal > 0 {
				skillFlags["freeze"] = true
				output["FreezeDurationMod"] = 1 + skillModList.Sum(mod.TypeIncrease, cfg, "EnemyFreezeDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration")/100 + enemyDB.Sum(mod.TypeIncrease, nil, "SelfFreezeDuration", "SelfElementalAilmentDuration", "SelfAilmentDuration", "HoarfrostFreezeDuration")/100
			}
		}

		for _, ailment := range nonDamagingAilmentList {
			name := string(ailment)
			if (output[name+"ChanceOnHit"] + output[name+"ChanceOnCrit"]) <= 0 {
				continue
			}

			cfg.SkillCond["CriticalStrike"] = ailmentMode == "CRIT" && !modDB.Flag(nil, "AilmentsAreNeverFromCrit")

			hitDmg, critDmg := calcAverageSourceDamage(ailment)
			damage := calcAilmentDamage(name, output["CritChance"], hitDmg, critDmg) * skillModList.More(cfg, name+"AsThoughDealing")
			if !(damage > 0) { //nolint:staticcheck // NaN-preserving; see above
				continue
			}

			skillFlags[SkillFlag(strings.ToLower(name))] = true

			incDur := skillModList.Sum(mod.TypeIncrease, cfg, "Enemy"+name+"Duration", "EnemyElementalAilmentDuration", "EnemyAilmentDuration") + enemyDB.Sum(mod.TypeIncrease, nil, "Self"+name+"Duration", "SelfElementalAilmentDuration", "SelfAilmentDuration")
			moreDur := skillModList.More(cfg, "Enemy"+name+"Duration", "EnemyElementalAilmentDuration", "EnemyAilmentDuration") * enemyDB.More(nil, "Self"+name+"Duration", "SelfElementalAilmentDuration", "SelfAilmentDuration")
			output[name+"Duration"] = *data.NonDamagingAilments[ailment].Duration * (1 + incDur/100) * moreDur * debuffDurationMult
			output[name+"EffectMod"] = CalcMod(skillModList, cfg, "Enemy"+name+"Effect") * calclib.Mod(enemyDB, nil, "Self"+name+"Effect")
			/*
				TODO Breakdown -- the ailment-threshold table, CalcOffence.lua:3507-3580.
				This is where enemyThreshold, the effect/thresh closures and bonechill are used.
			*/
		}
		// Calculate knockback chance/distance
		//pob:src/Modules/CalcOffence.lua:5391-5400
		output["KnockbackChance"] = min(100, output["KnockbackChanceOnHit"]*(1-output["CritChance"]/100)+output["KnockbackChanceOnCrit"]*output["CritChance"]/100+enemyDB.Sum(mod.TypeBase, nil, "SelfKnockbackChance"))
		if output["KnockbackChance"] > 0 {
			output["KnockbackDistance"] = math.Round(4 * calclib.Mod(skillModList, cfg, "EnemyKnockbackDistance"))
			/*
				TODO Breakdown -- breakdown.KnockbackDistance is a radius table, CalcOffence.lua:3589-3593.
			*/
		}
		// Calculate enemy stun modifiers
		enemyStunThresholdRed := -skillModList.Sum(mod.TypeIncrease, cfg, "EnemyStunThreshold")
		if enemyStunThresholdRed > 75 {
			output["EnemyStunThresholdMod"] = 1 - (75+(enemyStunThresholdRed-75)*25/(enemyStunThresholdRed-50))/100
		} else {
			output["EnemyStunThresholdMod"] = 1 - enemyStunThresholdRed/100
		}
		//pob:block:src/Modules/CalcOffence.lua:5409-5439
		stunBase := skillData.BaseStunDuration
		if stunBase == 0 {
			stunBase = 0.35
		}
		incStunDur := skillModList.Sum(mod.TypeIncrease, cfg, "EnemyStunDuration")
		incStunDurCrit := skillModList.Sum(mod.TypeIncrease, cfg, "EnemyStunDurationOnCrit")
		moreStunDur := skillModList.More(cfg, "EnemyStunDuration")
		chanceToDouble := min(skillModList.Sum(mod.TypeBase, cfg, "DoubleEnemyStunDurationChance")+
			enemyDB.Sum(mod.TypeBase, cfg, "SelfDoubleStunDurationChance"), 100)
		incStunRecov := enemyDB.Sum(mod.TypeIncrease, nil, "StunRecovery")
		output["EnemyStunDuration"] = stunBase * moreStunDur / (1 + incStunRecov/100)
		if incStunDurCrit != 0 && output["CritChance"] != 0 {
			if output["CritChance"] == 100 {
				output["EnemyStunDuration"] *= 1 + (incStunDur+incStunDurCrit)/100
			} else {
				output["EnemyStunDuration"] *= 1 + (incStunDur+incStunDurCrit*output["CritChance"]/100)/100
			}
		} else {
			output["EnemyStunDuration"] *= 1 + incStunDur/100
		}
		if chanceToDouble != 0 {
			output["EnemyStunDuration"] *= 1 + chanceToDouble/100
		}

		/*
			TODO Enemy stun breakdown
			if breakdown != nil {
				if actor.Output["EnemyStunDuration"] != base {
					breakdown.EnemyStunDuration = {
						fmt.Sprintf("%.2fs ^8(base duration)", base),
					}
					if incDur != 0 {
						breakdown.AddLine("EnemyStunDuration", fmt.Sprintf("x %.2f ^8(increased/reduced stun duration)", 1 + incDur/100))
					}
					if incRecov != 0 {
						breakdown.AddLine("EnemyStunDuration", fmt.Sprintf("/ %.2f ^8(increased/reduced enemy stun recovery)", 1 + incRecov/100))
					}
					breakdown.AddLine("EnemyStunDuration", fmt.Sprintf("= %.2fs", actor.Output["EnemyStunDuration"]))
				}
			}
		*/
		if canDeal[data.DamageTypePhysical] && (output["ImpaleChance"]+output["ImpaleChanceOnCrit"]) > 0 {
			skillFlags[SkillFlagImpale] = true
			impaleCritChance := output["CritChance"] / 100
			impaleChance := min(output["ImpaleChance"]/100, 1)*(1-impaleCritChance) + min(output["ImpaleChanceOnCrit"]/100, 1)*impaleCritChance
			maxStacks := skillModList.Sum(mod.TypeBase, cfg, "ImpaleStacksMax") *
				(1 + skillModList.Sum(mod.TypeBase, cfg, "ImpaleAdditionalDurationChance")/100)
			configStacks := enemyDB.Sum(mod.TypeBase, cfg, "Multiplier:ImpaleStacks")
			impaleStacks := min(maxStacks, configStacks)

			baseStoredDamage := float64(data.ImpaleStoredDamageBase)
			//pob:src/Modules/CalcOffence.lua:5482
			storedExpectedDamageIncOnBleed := skillModList.Sum(mod.TypeIncrease, cfg, "ImpaleEffectOnBleed") * min(skillModList.Sum(mod.TypeBase, cfg, "BleedChance")/100, 1)
			storedExpectedDamageInc := (skillModList.Sum(mod.TypeIncrease, cfg, "ImpaleEffect") + storedExpectedDamageIncOnBleed) / 100
			storedExpectedDamageMore := utils.RoundTo(skillModList.More(cfg, "ImpaleEffect"), 2)
			storedExpectedDamageModifier := (1 + storedExpectedDamageInc) * storedExpectedDamageMore
			impaleStoredDamage := baseStoredDamage * storedExpectedDamageModifier
			impaleHitDamageMod := impaleStoredDamage * impaleStacks // Source: https://www.reddit.com/r/pathofexile/comments/chgqqt/impale_and_armor_interaction/

			enemyArmour := max(calclib.Val(enemyDB, "Armour"), 0)
			impaleArmourReduction := CalcArmourReductionF(enemyArmour, impaleHitDamageMod*output["impaleStoredHitAvg"])
			impaleResist := min(max(0, enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction")+skillModList.Sum(mod.TypeBase, cfg, "EnemyImpalePhysicalDamageReduction")+impaleArmourReduction), data.EnemyPhysicalDamageReductionCap)
			//pob:block:src/Modules/CalcOffence.lua:5492-5494
			if skillModList.Flag(cfg, "IgnoreEnemyImpalePhysicalDamageReduction") {
				impaleResist = 0
			}

			impaleTakenCfg := &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagHit)}
			impaleTaken := (1 + enemyDB.Sum(mod.TypeIncrease, impaleTakenCfg, "DamageTaken", "PhysicalDamageTaken", "ReflectedDamageTaken")/100) *
				enemyDB.More(impaleTakenCfg, "DamageTaken", "PhysicalDamageTaken", "ReflectedDamageTaken")

			impaleDMGModifier := impaleHitDamageMod * (1 - impaleResist/100) * impaleChance * impaleTaken

			globalOutput["ImpaleStacksMax"] = maxStacks
			globalOutput["ImpaleStacks"] = impaleStacks
			// ImpaleStoredDamage should be named ImpaleEffect or similar
			// Using the variable name ImpaleEffect breaks the calculations sidebar (?!)
			output["ImpaleStoredDamage"] = impaleStoredDamage * 100
			output["ImpaleModifier"] = 1 + impaleDMGModifier

			if breakdown != nil {
				breakdown.AddLine("ImpaleStoredDamage", "10% ^8(base value)")
				breakdown.AddLine("ImpaleStoredDamage", fmt.Sprintf("x %.2f ^8(increased effectiveness)", storedExpectedDamageModifier))
				breakdown.AddLine("ImpaleStoredDamage", fmt.Sprintf("= %.1f%%", output["ImpaleStoredDamage"]))

				breakdown.AddLine("ImpaleModifier", fmt.Sprintf("%d ^8(number of stacks, can be overridden in the Configuration tab)", int(impaleStacks)))
				breakdown.AddLine("ImpaleModifier", fmt.Sprintf("x %.3f ^8(stored damage)", impaleStoredDamage))
				breakdown.AddLine("ImpaleModifier", fmt.Sprintf("x %.2f ^8(impale chance)", impaleChance))
				breakdown.AddLine("ImpaleModifier", fmt.Sprintf("x %.2f ^8(impale enemy physical damage reduction)", 1-impaleResist/100))
				if impaleTaken != 1 {
					breakdown.AddLine("ImpaleModifier", fmt.Sprintf("x %.2f ^8(impale enemy damage taken)", impaleTaken))
				}
				breakdown.AddLine("ImpaleModifier", fmt.Sprintf("= %.3f ^8(impale damage multiplier)", impaleDMGModifier))
			}
		}
	}

	// Combine secondary effect stats
	if isAttack {
		combineStat("BleedChance", "AVERAGE")
		combineStat("BleedDPS", "CHANCE_AILMENT", "BleedChance")
		combineStat("PoisonChance", "AVERAGE")
		combineStat("PoisonDPS", "CHANCE", "PoisonChance")
		combineStat("CausticGroundDPS", "CHANCE", "PoisonChance")
		combineStat("TotalPoisonDPS", "DPS")
		combineStat("PoisonDamage", "CHANCE", "PoisonChance")

		if skillData.ShowAverage {
			combineStat("TotalPoisonAverageDamage", "DPS")
		} else {
			combineStat("TotalPoisonStacks", "DPS")
		}

		combineStat("IgniteChance", "AVERAGE")
		combineStat("IgniteDPS", "CHANCE_AILMENT", "IgniteChance")

		if skillFlags[SkillFlagIgniteCanStack] {
			combineStat("IgniteDamage", "CHANCE", "IgniteChance")

			if skillData.ShowAverage {
				combineStat("TotalIgniteAverageDamage", "DPS")
				combineStat("IgniteStacksMax", "DPS")
				combineStat("TotalIgniteDPS", "DPS")
			} else {
				combineStat("IgniteStacksMax", "DPS")
				combineStat("TotalIgniteDPS", "DPS")
			}
		}

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
	}

	// Calculate DPS for Essence of Delirium's Decay effect
	//pob:src/Modules/CalcOffence.lua:3714-3741
	if utils.HasTrue(skillFlags, SkillFlagHit) && skillData.Decay != 0 && canDeal[data.DamageTypeChaos] {
		skillFlags[SkillFlagDecay] = true
		activeSkill.DecayCfg = &moddb.ListCfg{
			SkillName:       skillCfg.SkillName,
			SkillPart:       skillCfg.SkillPart,
			SkillTypes:      skillCfg.SkillTypes,
			SummonSkillName: skillCfg.SummonSkillName,
			SlotName:        skillCfg.SlotName,
			Flags:           utils.Ptr(mod.MFlagDot),
			KeywordFlags:    utils.Ptr((skillCfg.KeywordFlags.Get() & ^mod.KeywordFlagHit) | mod.KeywordFlagChaosDot),
		}
		dotCfg := activeSkill.DecayCfg
		effMult := float64(1)
		if env.ModeEffective {
			resist := calcResistForType(data.DamageTypeChaos, dotCfg)
			takenInc := enemyDB.Sum(mod.TypeIncrease, nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			takenMore := enemyDB.More(nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			actor.Output["DecayEffMult"] = effMult
			/*
				TODO Breakdown -- breakdown.DecayEffMult, CalcOffence.lua:3733-3735.
			*/
		}
		inc := skillModList.Sum(mod.TypeIncrease, dotCfg, "Damage", "ChaosDamage")
		more := skillModList.More(dotCfg, "Damage", "ChaosDamage")
		mult := skillModList.Sum(mod.TypeBase, dotCfg, "DotMultiplier") + skillModList.Sum(mod.TypeBase, dotCfg, "ChaosDotMultiplier")
		if override := skillModList.Override(dotCfg, "DotMultiplier"); override != nil {
			mult = override.Float()
		}
		actor.Output["DecayDPS"] = skillData.Decay * (1 + inc/100) * more * (1 + mult/100) * effMult
		actor.Output["DecayDuration"] = 8 * debuffDurationMult
		/*
			TODO Breakdown -- breakdown.DecayDPS / breakdown.DecayDuration,
			CalcOffence.lua:3742-3755.
		*/
	}

	//pob:block:src/Modules/CalcOffence.lua:5619-5645
	if baseDropsBurningGround := modDB.Sum(mod.TypeBase, nil, "DropsBurningGround"); baseDropsBurningGround > 0 {
		if canDeal[data.DamageTypeFire] {
			dotTakenCfg := &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagDot), KeywordFlags: utils.Ptr(mod.KeywordFlag(0))}
			dotTypeCfg := &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagDot), KeywordFlags: utils.Ptr(mod.KeywordFlagFireDot)}
			effMult := float64(1)
			if env.ModeEffective {
				resist := calcResistForType(data.DamageTypeFire, dotTypeCfg)
				takenInc := enemyDB.Sum(mod.TypeIncrease, dotTakenCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
				takenMore := enemyDB.More(dotTakenCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
				effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			}
			inc := modDB.Sum(mod.TypeIncrease, dotTypeCfg, "Damage", "FireDamage", "ElementalDamage")
			more := modDB.More(dotTypeCfg, "Damage", "FireDamage", "ElementalDamage")
			mult := modDB.Sum(mod.TypeBase, dotTypeCfg, "DotMultiplier") + modDB.Sum(mod.TypeBase, dotTypeCfg, "FireDotMultiplier")
			if override := modDB.Override(dotTypeCfg, "DotMultiplier"); override != nil {
				mult = override.Float()
			}
			total := baseDropsBurningGround * (1 + inc/100) * more * (1 + mult/100) * effMult
			if current, ok := actor.Output["BurningGroundDPS"]; !ok || current < total {
				actor.Output["BurningGroundDPS"] = total
			}
		}
	}

	// Calculate skill DOT components
	//pob:block:src/Modules/CalcOffence.lua:4318-4357
	dotFlags := mod.MFlagDot | skillCfg.Flags.Get()
	for _, clear := range []struct {
		flag mod.MFlag
		keep bool
	}{
		{mod.MFlagArea, skillData.DotIsArea},
		{mod.MFlagProjectile, skillData.DotIsProjectile},
		{mod.MFlagSpell, skillData.DotIsSpell},
		{mod.MFlagAttack, skillData.DotIsAttack},
		{mod.MFlagHit, skillData.DotIsHit},
	} {
		if dotFlags&clear.flag == clear.flag && !clear.keep {
			dotFlags &= ^clear.flag
		}
	}

	dotKeywordFlags := skillCfg.KeywordFlags.Get() & ^mod.KeywordFlagHit

	// spell_damage_modifiers_apply_to_skill_dot does not apply to enemy damage taken
	dotTakenFlags := dotFlags
	if skillData.DotIsSpell {
		dotTakenFlags &= ^mod.MFlagSpell
	}
	dotTakenCfg := &moddb.ListCfg{
		SkillName:       skillCfg.SkillName,
		SkillPart:       skillCfg.SkillPart,
		SkillTypes:      skillCfg.SkillTypes,
		SummonSkillName: skillCfg.SummonSkillName,
		SlotName:        skillCfg.SlotName,
		Flags:           utils.Ptr(dotTakenFlags),
		KeywordFlags:    utils.Ptr(dotKeywordFlags),
	}

	activeSkill.DotCfg = &moddb.ListCfg{
		SkillName:       skillCfg.SkillName,
		SkillPart:       skillCfg.SkillPart,
		SkillTypes:      skillCfg.SkillTypes,
		SummonSkillName: skillCfg.SummonSkillName,
		SlotName:        skillCfg.SlotName,
		Flags:           utils.Ptr(dotFlags),
		KeywordFlags:    utils.Ptr(dotKeywordFlags),
	}
	activeSkill.DotTypeCfg = map[data.DamageType]*moddb.ListCfg{}
	actor.Output["TotalDotInstance"] = 0

	for _, damageType := range data.DamageType("").Values() {
		dotTypeCfg := &moddb.ListCfg{
			SkillName:       skillCfg.SkillName,
			SkillPart:       skillCfg.SkillPart,
			SkillTypes:      skillCfg.SkillTypes,
			SummonSkillName: skillCfg.SummonSkillName,
			SlotName:        skillCfg.SlotName,
			Flags:           utils.Ptr(dotFlags),
			KeywordFlags:    utils.Ptr(dotKeywordFlags | damageTypeDotKeywordFlags[damageType]),
		}
		activeSkill.DotTypeCfg[damageType] = dotTypeCfg

		baseVal := float64(0)
		if canDeal[damageType] {
			baseVal = skillDotBase(skillData, damageType)
		}

		if baseVal <= 0 && actor.Output[string(damageType)+"Dot"] <= 0 {
			continue
		}

		skillFlags[SkillFlagDot] = true

		effMult := float64(1)
		if env.ModeEffective {
			resist := float64(0)
			takenInc := enemyDB.Sum(mod.TypeIncrease, dotTakenCfg, "DamageTaken", "DamageTakenOverTime", string(damageType)+"DamageTaken", string(damageType)+"DamageTakenOverTime")
			takenMore := enemyDB.More(dotTakenCfg, "DamageTaken", "DamageTakenOverTime", string(damageType)+"DamageTaken", string(damageType)+"DamageTakenOverTime")

			if damageType.IsElemental() {
				takenInc += enemyDB.Sum(mod.TypeIncrease, dotTakenCfg, "ElementalDamageTaken")
				takenMore *= enemyDB.More(dotTakenCfg, "ElementalDamageTaken")
			}

			if damageType == data.DamageTypePhysical {
				resist = max(0, min(enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction"), data.EnemyPhysicalDamageReductionCap))
			} else {
				resist = calcResistForType(damageType, dotTypeCfg)
			}

			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			actor.Output[string(damageType)+"DotEffMult"] = effMult
		}

		names := []string{"Damage", string(damageType) + "Damage"}
		if damageType.IsElemental() {
			names = append(names, "ElementalDamage")
		}
		inc := skillModList.Sum(mod.TypeIncrease, dotTypeCfg, names...)
		if skillModList.Flag(nil, "dotIsHeraldOfAsh") {
			inc = max(inc-skillModList.Sum(mod.TypeIncrease, skillCfg, names...), 0)
		}
		more := skillModList.More(dotTypeCfg, names...)
		mult := skillModList.Sum(mod.TypeBase, dotTypeCfg, "DotMultiplier") + skillModList.Sum(mod.TypeBase, dotTypeCfg, string(damageType)+"DotMultiplier")
		if override := skillModList.Override(dotTypeCfg, "DotMultiplier"); override != nil {
			mult = override.Float()
		}

		auraMult := float64(1)
		if activeSkill.SkillTypes[data.SkillTypeAura] && !activeSkill.SkillTypes[data.SkillTypeRemoteMined] &&
			!activeSkill.SkillTypes[data.SkillTypeBanner] {
			auraMult = CalcMod(skillModList, dotTypeCfg, "AuraEffect")
		}

		total := baseVal * (1 + inc/100) * more * (1 + mult/100) * auraMult * effMult
		if actor.Output[string(damageType)+"Dot"] == 0 {
			actor.Output[string(damageType)+"Dot"] = total
			actor.Output["TotalDotInstance"] = min(actor.Output["TotalDotInstance"]+total, data.DotDpsCap)
		} else {
			actor.Output["TotalDotInstance"] = min(actor.Output["TotalDotInstance"]+total+actor.Output[string(damageType)+"Dot"], data.DotDpsCap)
		}
	}

	if skillModList.Flag(nil, "DotCanStack") {
		skillFlags[SkillFlagDotCanStack] = true
		speed := actor.Output["Speed"]
		// Check if skill is being triggered via Mine (e.g., Blastchain Mine Support) or Trap
		// if "yes", you cannot use output.Speed but rather should use output.MineLayingSpeed or output.TrapThrowingSpeed
		if dotKeywordFlags&mod.KeywordFlagMine != 0 {
			speed = actor.Output["MineLayingSpeed"]
		} else if dotKeywordFlags&mod.KeywordFlagTrap != 0 {
			speed = actor.Output["TrapThrowingSpeed"]
		}
		actor.Output["TotalDot"] = min(actor.Output["TotalDotInstance"]*speed*actor.Output["Duration"]*utils.OrDefault(skillData.DpsMultiplier, 1)*quantityMultiplier, data.DotDpsCap)
		actor.Output["TotalDotCalcSection"] = actor.Output["TotalDot"]
	} else if skillModList.Flag(nil, "dotIsBurningGround") ||
		skillModList.Flag(nil, "dotIsCausticGround") ||
		skillModList.Flag(nil, "dotIsCorruptingBlood") {
		actor.Output["TotalDot"] = 0
		actor.Output["TotalDotCalcSection"] = actor.Output["TotalDotInstance"]
		//pob:block:src/Modules/CalcOffence.lua:5763-5782
		switch {
		case skillModList.Flag(nil, "dotIsBurningGround"):
			if actor.Output["BurningGroundDPS"] < actor.Output["TotalDotInstance"] {
				actor.Output["BurningGroundDPS"] = max(actor.Output["BurningGroundDPS"], actor.Output["TotalDotInstance"])
			}
		case skillModList.Flag(nil, "dotIsCausticGround"):
			if actor.Output["CausticGroundDPS"] < actor.Output["TotalDotInstance"] {
				actor.Output["CausticGroundDPS"] = max(actor.Output["CausticGroundDPS"], actor.Output["TotalDotInstance"])
			}
		case skillModList.Flag(nil, "dotIsCorruptingBlood"):
			if actor.Output["CorruptingBloodDPS"] < actor.Output["TotalDotInstance"] {
				actor.Output["CorruptingBloodDPS"] = max(actor.Output["CorruptingBloodDPS"], actor.Output["TotalDotInstance"])
			}
		}
	} else {
		if skillModList.Flag(nil, "DotCanStackAsTotems") && skillFlags[SkillFlagTotem] {
			skillFlags[SkillFlagDotCanStack] = true
		}
		//pob:block:src/Modules/CalcOffence.lua:5787-5789
		attachedBrandCount := float64(1)
		if activeSkill.SkillTypes[data.SkillTypeBrand] && !skillData.CountsAttachedBrandsInDamage {
			attachedBrandCount = actor.Output["AttachedBrandCount"]
		}
		if attachedBrandCount > 1 {
			actor.Output["TotalDot"] = math.Min(actor.Output["TotalDotInstance"]*attachedBrandCount, data.DotDpsCap)
		} else {
			actor.Output["TotalDot"] = actor.Output["TotalDotInstance"]
		}
		actor.Output["TotalDotCalcSection"] = actor.Output["TotalDot"]
	}

	//pob:block:src/Modules/CalcOffence.lua:4448-4488
	eb := env.ModDB.Flag(nil, "EnergyShieldProtectsMana")
	for resource, val := range costs {
		//	not (output[resource.."PerSecondHasCost"] and not (EB and Sum("BASE", skillCfg, "ManaCostAsEnergyShieldCost")))
		if !val.Upfront || !hasCost[resource] || actor.Output[resource+"Cost"] <= 0 ||
			(hasCost[resource+"PerSecond"] && !eb) ||
			!(actor.Output["Speed"] > 0 || actor.Output["Cooldown"] != 0) {
			continue
		}

		usedResource := resource
		if eb && resource == "Mana" {
			usedResource = "ES"
		}

		repeats := actor.Output["Repeats"]
		if repeats == 0 {
			repeats = 1
		}
		var useSpeed float64
		switch {
		case utils.HasTrue(skillFlags, SkillFlagTrap) || utils.HasTrue(skillFlags, SkillFlagMine):
			preSpeed := utils.OrF(actor.Output["TrapThrowingSpeed"], actor.Output["MineLayingSpeed"])
			cooldown := utils.OrF(actor.Output["TrapCooldown"], actor.Output["Cooldown"])
			useSpeed = utils.Ternary(cooldown > 0, 1/cooldown, preSpeed)
		case utils.HasTrue(skillFlags, SkillFlagTotem):
			placement := actor.Output["TotemPlacementSpeed"]
			if cd := actor.Output["Cooldown"]; cd > 0 {
				useSpeed = utils.Ternary(placement > 0, placement, 1/cd) / repeats
			} else {
				useSpeed = placement / repeats
			}
		case skillModList.Flag(nil, "HasSeals") && skillModList.Flag(nil, "UseMaxUnleash") &&
			env.Player.MainSkill.SkillData.HitTimeOverride != 0:
			useSpeed = env.Player.MainSkill.SkillData.HitTimeOverride / repeats
		default:
			speed := actor.Output["Speed"]
			if cd := actor.Output["Cooldown"]; cd > 0 {
				useSpeed = utils.Ternary(speed > 0, speed, 1/cd) / repeats
			} else {
				useSpeed = speed / repeats
			}
		}

		hasCost[usedResource+"PerSecond"] = true
		actor.Output[usedResource+"PerSecondHasCost"] = 1
		actor.Output[usedResource+"PerSecondCost"] += actor.Output[resource+"Cost"] * useSpeed
	}
	//pob:block:src/Modules/CalcOffence.lua:5835-5980
	{
		selfHitSum := float64(0)
		selfHitAny := false
		add := func(dmgType string, dmg float64, mult float64) {
			_, taken := ApplyDmgTakenConversion(skillModList, actor.Output, breakdown,
				data.DamageType(selfHitDamageTypeName(dmgType)), dmg)
			selfHitSum += taken * mult
			selfHitAny = true
		}
		sumAll := func(name string) (string, float64, bool) {
			dmgType, dmgVal, found := "", float64(0), false
			for _, v := range skillModList.List(nil, name) {
				sd, ok := v.(mod.SelfDamage)
				if !ok {
					continue
				}
				dmgVal += sd.BaseDamage
				dmgType = sd.DamageType
				found = true
			}
			return dmgType, dmgVal, found
		}
		first := func(name string) (mod.SelfDamage, bool) {
			for _, v := range skillModList.List(nil, name) {
				if sd, ok := v.(mod.SelfDamage); ok {
					return sd, true
				}
			}
			return mod.SelfDamage{}, false
		}
		grantedEffectName := ""
		if activeSkill.ActiveEffect != nil && activeSkill.ActiveEffect.GrantedEffect != nil {
			grantedEffectName = activeSkill.ActiveEffect.GrantedEffect.Name()
		}

		if grantedEffectName == "Summon Skeletons" {
			if dmgType, dmgVal, found := sumAll("HeartboundLoopSelfDamage"); found {
				add(dmgType, dmgVal, actor.Output["SummonedMinionsPerCast"])
			}
		}
		if grantedEffectName == "Herald of Thunder" {
			if dmgType, dmgVal, found := sumAll("StormSecretSelfDamage"); found {
				add(dmgType, dmgVal, 1)
			}
		}
		if sd, found := first("EyeOfInnocenceSelfDamage"); found && skillFlags[SkillFlagIgnite] {
			add(sd.DamageType, sd.BaseDamage, 1)
		}
		if sd, found := first("EchoesOfCreationSelfDamage"); found {
			averageWarcryCount := actor.Output["GlobalWarcryUptimeRatio"] / 100
			if skillModList.Flag(nil, "Condition:WarcryMaxHit") {
				averageWarcryCount = env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:ExertingWarcryCount")
			}
			add(sd.DamageType, actor.Output["Life"]*sd.DmgMult/100*averageWarcryCount, 1)
		}
		if sd, found := first("ScoldsBridleSelfDamage"); found && hasCost["Mana"] {
			add(sd.DamageType, actor.Output["ManaCost"]*sd.DmgMult/100, 1)
		}
		if activeSkill.BaseSkillModList != nil && activeSkill.BaseSkillModList.Flag(nil, "HasTrauma") {
			currentTraumaStacks := math.Max(skillModList.Sum(mod.TypeBase, nil, "Multiplier:TraumaStacks"), 1)
			add("Physical", skillModList.Sum(mod.TypeBase, nil, "TraumaSelfDamageTakenLife")*currentTraumaStacks, 1)
		}
		if dmgType, dmgVal, found := sumAll("EnmitysEmbraceSelfDamage"); found {
			add(dmgType, dmgVal, 1)
		}

		//	for selfHitSkill, displayName in ipairs({["FRDamageTaken"] = "Forbidden Rite"}) do
		if selfHitAny {
			actor.Output["SelfHitDamage"] = selfHitSum
		}
	}

	// Calculate combined DPS estimate, including DoTs
	baseDPS := actor.Output[utils.Ternary(skillData.ShowAverage, "AverageDamage", "TotalDPS")]
	actor.Output["CombinedDPS"] = baseDPS
	actor.Output["CombinedAvg"] = baseDPS
	if skillFlags[SkillFlagDot] {
		actor.Output["WithDotDPS"] = baseDPS + (actor.Output["TotalDot"])
	}
	if quantityMultiplier > 1 && actor.Output["TotalPoisonDPS"] > 0 {
		actor.Output["TotalPoisonDPS"] = min(actor.Output["TotalPoisonDPS"]*quantityMultiplier, data.DotDpsCap)
	}
	if skillData.ShowAverage {
		actor.Output["CombinedAvg"] = actor.Output["CombinedAvg"] + (actor.Output["TotalPoisonAverageDamage"])
		actor.Output["WithPoisonDPS"] = baseDPS + (actor.Output["TotalPoisonAverageDamage"])
	} else {
		actor.Output["WithPoisonDPS"] = baseDPS + (actor.Output["TotalPoisonDPS"])
	}
	if skillFlags[SkillFlagIgnite] {
		if skillFlags[SkillFlagIgniteCanStack] {
			if skillData.ShowAverage {
				actor.Output["CombinedAvg"] = actor.Output["CombinedDPS"] + actor.Output["IgniteDamage"]
			} else {
				actor.Output["WithIgniteDPS"] = baseDPS + actor.Output["TotalIgniteDPS"]
			}
		} else if skillData.ShowAverage {
			actor.Output["WithIgniteDPS"] = baseDPS + actor.Output["IgniteDamage"]
			actor.Output["CombinedAvg"] = actor.Output["CombinedAvg"] + actor.Output["IgniteDamage"]
		} else {
			actor.Output["WithIgniteDPS"] = baseDPS + actor.Output["IgniteDPS"]
		}
	} else {
		actor.Output["WithIgniteDPS"] = baseDPS
	}
	//pob:block:src/Modules/CalcOffence.lua:6014-6016
	if skillFlags[SkillFlagMonsterExplode] {
		actor.Output["CombinedAvgToMonsterLife"] = actor.Output["CombinedAvg"] / monsterLife * 100
	}
	if skillFlags[SkillFlagBleed] {
		if skillData.ShowAverage {
			actor.Output["WithBleedDPS"] = baseDPS + actor.Output["BleedDamage"]
			actor.Output["CombinedAvg"] = actor.Output["CombinedAvg"] + actor.Output["BleedDamage"]
		} else {
			actor.Output["WithBleedDPS"] = baseDPS + actor.Output["BleedDPS"]
		}
	} else {
		actor.Output["WithBleedDPS"] = baseDPS
	}
	bestCull := float64(1)

	//pob:block:src/Modules/CalcOffence.lua:4662-4696
	if mirage := activeSkill.Mirage; mirage != nil && mirage.Output != nil && hasMirageTotalDPS(mirage.Output) {
		mirageCount := mirage.Count
		if mirageCount == 0 {
			mirageCount = 1
		}
		actor.Output["MirageDPS"] = mirage.Output["TotalDPS"] * mirageCount
		actor.Output["CombinedDPS"] += mirage.Output["TotalDPS"] * mirageCount

		if mirage.Output["IgniteDPS"] > actor.Output["IgniteDPS"] {
			actor.Output["MirageDPS"] += mirage.Output["IgniteDPS"]
			actor.Output["IgniteDPS"] = 0
		}
		if mirage.Output["BleedDPS"] > actor.Output["BleedDPS"] {
			actor.Output["MirageDPS"] += mirage.Output["BleedDPS"]
			actor.Output["BleedDPS"] = 0
		}

		if mirage.Output["PoisonDPS"] != 0 {
			actor.Output["MirageDPS"] += mirage.Output["PoisonDPS"] * mirageCount
			actor.Output["CombinedDPS"] += mirage.Output["PoisonDPS"] * mirageCount
		}
		if mirage.Output["ImpaleDPS"] != 0 {
			actor.Output["MirageDPS"] += mirage.Output["ImpaleDPS"] * mirageCount
			actor.Output["CombinedDPS"] += mirage.Output["ImpaleDPS"] * mirageCount
		}
		if mirage.Output["DecayDPS"] != 0 {
			actor.Output["MirageDPS"] += mirage.Output["DecayDPS"]
			actor.Output["CombinedDPS"] += mirage.Output["DecayDPS"]
		}
		if mirage.Output["TotalDot"] != 0 && (skillFlags[SkillFlagDotCanStack] || actor.Output["TotalDot"] == 0) {
			dotCount := float64(1)
			if skillFlags[SkillFlagDotCanStack] {
				dotCount = mirageCount
			}
			actor.Output["MirageDPS"] += mirage.Output["TotalDot"] * dotCount
			actor.Output["CombinedDPS"] += mirage.Output["TotalDot"] * dotCount
		}
		if mirage.Output["CullMultiplier"] > 1 {
			bestCull = mirage.Output["CullMultiplier"]
		}
	}

	//pob:block:src/Modules/CalcOffence.lua:6126-6127
	totalDotDPS := (actor.Output["TotalDot"]) + (actor.Output["TotalPoisonDPS"]) +
		(actor.Output["CausticGroundDPS"]) +
		utils.OrDefault(actor.Output["TotalIgniteDPS"], actor.Output["IgniteDPS"]) +
		(actor.Output["BurningGroundDPS"]) +
		(actor.Output["BleedDPS"]) + (actor.Output["CorruptingBloodDPS"]) + (actor.Output["DecayDPS"])
	actor.Output["TotalDotDPS"] = min(totalDotDPS, data.DotDpsCap)
	//pob:block:src/Modules/CalcOffence.lua:6128-6130
	if actor.Output["TotalDotDPS"] != totalDotDPS {
		actor.Output["showTotalDotDPS"] = 1
	}
	//pob:block:src/Modules/CalcOffence.lua:6131-6133
	if !skillData.ShowAverage {
		actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + actor.Output["TotalDotDPS"]
	}
	//pob:block:src/Modules/CalcOffence.lua:6027-6039
	if skillFlags[SkillFlagImpale] {
		dpsMultiplier := utils.OrDefault(skillData.DpsMultiplier, 1)
		var mainHandImpaleDPS, offHandImpaleDPS float64
		splitHands := skillFlags[SkillFlagAttack] && skillData.DoubleHitsWhenDualWielding && skillFlags[SkillFlagBothWeaponAttack]
		if splitHands {
			mh, oh := actor.OutputTable[OutTableMainHand], actor.OutputTable[OutTableOffHand]
			mainHandImpaleDPS = mh["impaleStoredHitAvg"] * (utils.OrDefault(mh["ImpaleModifier"], 1) - 1) * mh["HitChance"] / 100 * dpsMultiplier
			offHandImpaleDPS = oh["impaleStoredHitAvg"] * (utils.OrDefault(oh["ImpaleModifier"], 1) - 1) * oh["HitChance"] / 100 * dpsMultiplier
			actor.Output["ImpaleDPS"] = mainHandImpaleDPS + offHandImpaleDPS
		} else {
			actor.Output["ImpaleDPS"] = actor.Output["impaleStoredHitAvg"] * (utils.OrDefault(actor.Output["ImpaleModifier"], 1) - 1) * actor.Output["HitChance"] / 100 * dpsMultiplier
		}
		if actor.Output["ImpaleDuration"] <= 0 {
			actor.Output["ImpaleDPS"] = 0
		}
		if skillData.ShowAverage {
			actor.Output["WithImpaleDPS"] = actor.Output["AverageDamage"] + actor.Output["ImpaleDPS"]
			actor.Output["CombinedAvg"] = actor.Output["CombinedAvg"] + actor.Output["ImpaleDPS"]
		} else {
			skillFlags[SkillFlagNotAverage] = true
			actor.Output["ImpaleDPS"] = actor.Output["ImpaleDPS"] * utils.OrDefault(actor.Output["HitSpeed"], actor.Output["Speed"])
			actor.Output["WithImpaleDPS"] = actor.Output["TotalDPS"] + actor.Output["ImpaleDPS"]
		}
		if quantityMultiplier > 1 {
			actor.Output["ImpaleDPS"] = actor.Output["ImpaleDPS"] * quantityMultiplier
		}
		actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + actor.Output["ImpaleDPS"]
		//pob:block:src/Modules/CalcOffence.lua:6052-6084
		if breakdown != nil {
			if splitHands {
				mh, oh := actor.OutputTable[OutTableMainHand], actor.OutputTable[OutTableOffHand]
				breakdown.AddLine("ImpaleDPS", "Main Hand:")
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("%.2f ^8(MH average physical hit before mitigation)", mh["impaleStoredHitAvg"]))
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %.2f ^8(MH chance to hit)", mh["HitChance"]/100))
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %.2f ^8(MH impale damage multiplier)\n", utils.OrDefault(mh["ImpaleModifier"], 1)-1))
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("= %.2f", mainHandImpaleDPS))
				breakdown.AddLine("ImpaleDPS", "Off Hand:")
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("%.2f ^8(OH average physical hit before mitigation)", oh["impaleStoredHitAvg"]))
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %.2f ^8(OH chance to hit)", oh["HitChance"]/100))
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %.2f ^8(OH impale damage multiplier)", utils.OrDefault(oh["ImpaleModifier"], 1)-1))
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("= %.2f", offHandImpaleDPS))
				breakdown.AddLine("ImpaleDPS", "Combined total:")
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("%.2f + %.2f", mainHandImpaleDPS, offHandImpaleDPS))
			} else {
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("%.2f ^8(average physical hit before mitigation)", actor.Output["impaleStoredHitAvg"]))
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %.2f ^8(chance to hit)", actor.Output["HitChance"]/100))
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %.2f ^8(impale damage multiplier)", utils.OrDefault(actor.Output["ImpaleModifier"], 1)-1))
			}
			if skillFlags[SkillFlagNotAverage] {
				breakdown.AddLine("ImpaleDPS", utils.Ternary(actor.Output["HitSpeed"] != 0, fmt.Sprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"]), fmt.Sprintf("x %.2f ^8(%s rate)", actor.Output["Speed"], utils.Ternary(skillFlags[SkillFlagAttack], "attack", "cast"))))
			}
			if skillData.DpsMultiplier != 0 {
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %g ^8(dps multiplier for this skill)", skillData.DpsMultiplier))
			}
			if quantityMultiplier > 1 {
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %g ^8(quantity multiplier for this skill)", quantityMultiplier))
			}
			if actor.Output["ImpaleDuration"] <= 0 {
				breakdown.AddLine("ImpaleDPS", "x 0 ^8(no Impale Duration)")
			}
			breakdown.AddLine("ImpaleDPS", fmt.Sprintf("= %.1f", actor.Output["ImpaleDPS"]))
		}
	}

	bestCull = max(bestCull, actor.Output["CullMultiplier"])
	actor.Output["CullingDPS"] = actor.Output["CombinedDPS"] * (bestCull - 1)
	//pob:block:src/Modules/CalcOffence.lua:6137-6138
	actor.Output["ReservationDPS"] = actor.Output["CombinedDPS"] * (actor.Output["ReservationDpsMultiplier"] - 1)
	actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] * bestCull * actor.Output["ReservationDpsMultiplier"]
}

func hasMirageTotalDPS(output map[string]float64) bool {
	_, ok := output["TotalDPS"]
	return ok
}

//pob:block:src/Modules/CalcOffence.lua:3802
func ailmentSkillCond(skillCfg *moddb.ListCfg, cfg *moddb.ListCfg) map[string]bool {
	out := make(map[string]bool, len(skillCfg.SkillCond)+len(cfg.SkillCond)+1)
	for k, v := range cfg.SkillCond {
		out[k] = v
	}
	for k, v := range skillCfg.SkillCond {
		out[k] = v || out[k]
	}
	out["CriticalStrike"] = true
	return out
}

//pob:block:src/Modules/CalcOffence.lua:301-302
func durationModClamped(list moddb.ModStoreFuncs, cfg *moddb.ListCfg, names ...string) float64 {
	return math.Max(CalcMod(list, cfg, names...), 0)
}

func CalcTotemLife(activeSkill *ActiveSkill) (float64, float64) {
	lifeMod := calclib.Mod(activeSkill.SkillModList, activeSkill.SkillCfg, "TotemLife")
	totemMult, ok := totemLifeMult(activeSkill.SkillTotemId)
	if !ok {
		totemMult = 1
	}
	level := activeSkill.SkillData.TotemLevel
	if level < 0 || level >= len(data.MonsterAllyLifeTable) {
		return 0, lifeMod
	}
	return utils.RoundTo(math.Floor(data.MonsterAllyLifeTable[level]*totemMult)*lifeMod, 0), lifeMod
}

func selfHitDamageTypeName(s string) string {
	r := []rune(" " + s)
	for i := 0; i+1 < len(r); i++ {
		if isLuaWordRune(r[i]) || r[i+1] < 'a' || r[i+1] > 'z' {
			continue
		}
		r[i+1] = r[i+1] - 'a' + 'A'
		i++
	}
	return string(r[1:])
}

func isLuaWordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
