package calculator

import (
	"fmt"
	"maps"
	"math"
	"slices"
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

//pob:block:src/Classes/ConfigTab.lua:1293-1294
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

//pob:src/Modules/CalcOffence.lua:69-141
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
			addMin += float64(minDamage * convMult)
			addMax += float64(maxDamage * convMult)
		}
	}

	baseMin := output[string(damageType)+"MinBase"]
	baseMax := output[string(damageType)+"MaxBase"]

	if baseMin == 0 && baseMax == 0 {
		// No base damage for this type, don't need to calculate modifiers
		if breakdown != nil && (addMin != 0 || addMax != 0) {
			row := BDamageType{
				Source:  string(damageType),
				ConvSrc: utils.Ternary(addMin != 0 || addMax != 0, utils.Ptr(fmt.Sprint(math.Round(addMin))+" to "+fmt.Sprint(math.Round(addMax))), nil),
				Total:   fmt.Sprint(math.Round(addMin)) + " to " + fmt.Sprint(math.Round(addMax)),
			}
			if convDst != nil {
				table := activeSkill.conversionTableFor(cfg)[damageType]
				row.ConvDst = utils.Ptr(luaSprintf("%d%% to %s", luaInt(table.Conversion[*convDst]*100), *convDst))
				row.GainDst = utils.Ptr(luaSprintf("%d%% gained as %s", luaInt(table.Gain[*convDst]*100), *convDst))
			}
			breakdown.DamageType(breakdownKey, row)
		}
		if convDst != nil {
			return addMin, addMax
		}
		return math.Round(addMin), math.Round(addMax)
	}

	// Combine modifiers
	modNames := damageStatsForTypes[typeFlags]
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
			ConvSrc: utils.Ternary(addMin != 0 || addMax != 0, utils.Ptr(fmt.Sprint(math.Round(addMin))+" to "+fmt.Sprint(math.Round(addMax))), nil),
			Total:   fmt.Sprint(math.Round(float64(baseMin*inc*more)+addMin)) + " to " + fmt.Sprint(math.Round(float64(baseMax*inc*more)+addMax)),
		}
		//pob:block:src/Modules/CalcOffence.lua:129-130
		if convDst != nil {
			table := activeSkill.conversionTableFor(cfg)[damageType]
			if table.Conversion[*convDst] > 0 {
				row.ConvDst = utils.Ptr(luaSprintf("%d%% to %s", luaInt(table.Conversion[*convDst]*100), *convDst))
			}
			if table.Gain[*convDst] > 0 {
				row.GainDst = utils.Ptr(luaSprintf("%d%% gained as %s", luaInt(table.Gain[*convDst]*100), *convDst))
			}
		}
		breakdown.DamageType(breakdownKey, row)
	}

	minDamage := (float64((baseMin*inc*more)*genericMoreMinDamage) + addMin) * moreMinDamage * incMinDamage
	maxDamage := (float64((baseMax*inc*more)*genericMoreMaxDamage) + addMax) * moreMaxDamage * incMaxDamage
	if convDst != nil {
		// Converted paths stay fractional until they reach their final damage type.
		return minDamage, maxDamage
	}
	return math.Round(minDamage), math.Round(maxDamage)
}

//pob:src/Modules/CalcOffence.lua:143-154
func calcAilmentSourceDamage(activeSkill *ActiveSkill, output map[string]float64, cfg *moddb.ListCfg, breakdown *Breakdown, breakdownKey string, damageType data.DamageType, typeFlags int) (float64, float64) {
	minDamage, maxDamage := calcDamage(activeSkill, output, cfg, breakdown, breakdownKey, damageType, typeFlags, nil)
	convMult := activeSkill.conversionTableFor(cfg)[damageType].Mult
	if breakdown != nil && convMult != 1 {
		breakdown.AddLine(breakdownKey, "Source damage:")
		breakdown.AddLine(breakdownKey, luaSprintf("%d to %d ^8(total damage)", luaInt(minDamage), luaInt(maxDamage)))
		breakdown.AddLine(breakdownKey, luaSprintf("x %.6g ^8(%.6g%% converted to other damage types)", convMult, (1-convMult)*100))
		breakdown.AddLine(breakdownKey, luaSprintf("= %d to %d", luaInt(minDamage*convMult), luaInt(maxDamage*convMult)))
	}
	return minDamage * convMult, maxDamage * convMult
}

var conversionNames = func() [5][5]struct{ global, skill, add []string } {
	var out [5][5]struct{ global, skill, add []string }
	types := data.DamageType("").Values()
	for i, damageType := range types {
		for j := i + 1; j < len(types); j++ {
			otherType := types[j]
			n := &out[i][j]
			n.global = []string{string(damageType) + "DamageConvertTo" + string(otherType)}
			n.add = []string{string(damageType) + "DamageGainAs" + string(otherType)}
			if damageType.IsElemental() {
				n.global = append(n.global, "ElementalDamageConvertTo"+string(otherType))
				n.add = append(n.add, "ElementalDamageGainAs"+string(otherType))
			}
			if damageType != data.DamageTypeChaos {
				n.global = append(n.global, "NonChaosDamageConvertTo"+string(otherType))
				n.add = append(n.add, "NonChaosDamageGainAs"+string(otherType))
			}
			n.skill = []string{"Skill" + string(damageType) + "DamageConvertTo" + string(otherType)}
		}
	}
	return out
}()

//pob:src/Modules/CalcOffence.lua:1938-1987
func buildConversionTable(skillModList *moddb.ModList, cfg *moddb.ListCfg) map[data.DamageType]ConversionTable {
	table := make(map[data.DamageType]ConversionTable)
	types := data.DamageType("").Values()
	totalDamageTypes := len(types)
	for damageTypeIndex := 0; damageTypeIndex < totalDamageTypes; damageTypeIndex++ {
		damageType := types[damageTypeIndex]
		var globalConv, skillConv, add [5]float64
		globalTotal := float64(0)
		skillTotal := float64(0)

		for otherTypeIndex := damageTypeIndex + 1; otherTypeIndex < totalDamageTypes; otherTypeIndex++ {
			// For all possible destination types, check for global and skill conversions
			names := &conversionNames[damageTypeIndex][otherTypeIndex]
			globalConv[otherTypeIndex] = math.Max(skillModList.Sum(mod.TypeBase, cfg, names.global...), 0)
			globalTotal += globalConv[otherTypeIndex]

			skillConv[otherTypeIndex] = math.Max(skillModList.Sum(mod.TypeBase, cfg, names.skill...), 0)
			skillTotal += skillConv[otherTypeIndex]

			add[otherTypeIndex] = math.Max(skillModList.Sum(mod.TypeBase, cfg, names.add...), 0)
		}

		if skillTotal > 100 {
			// Skill conversion exceeds 100%, scale it down and remove non-skill conversions
			factor := 100 / skillTotal
			for i := damageTypeIndex + 1; i < totalDamageTypes; i++ {
				skillConv[i] = skillConv[i] * factor
				globalConv[i] = 0
			}
		} else if globalTotal+skillTotal > 100 {
			// Conversion exceeds 100%, scale down non-skill conversions
			factor := (100 - skillTotal) / globalTotal
			for i := damageTypeIndex + 1; i < totalDamageTypes; i++ {
				globalConv[i] = globalConv[i] * factor
			}
			globalTotal = globalTotal * factor //nolint:staticcheck
		}

		n := totalDamageTypes - damageTypeIndex - 1
		dmgTable := ConversionTable{
			Targets:    make(map[data.DamageType]float64, n),
			Conversion: make(map[data.DamageType]float64, n),
			Gain:       make(map[data.DamageType]float64, n),
		}
		convertedTotal := float64(0)
		for i := damageTypeIndex + 1; i < totalDamageTypes; i++ {
			convType := types[i]
			// The game stores each scaled conversion destination as a whole percent.
			conversion := math.Round(globalConv[i] + skillConv[i])
			dmgTable.Conversion[convType] = conversion / 100
			dmgTable.Gain[convType] = add[i] / 100
			dmgTable.Targets[convType] = (conversion + add[i]) / 100
			convertedTotal = convertedTotal + conversion
		}

		dmgTable.Mult = 1 - math.Min(convertedTotal/100, 1)
		table[damageType] = dmgTable
	}

	table[data.DamageTypeChaos] = ConversionTable{
		Mult: 1,
	}
	return table
}

func (a *ActiveSkill) conversionTableFor(cfg *moddb.ListCfg) map[data.DamageType]ConversionTable {
	//pob:src/Modules/CalcOffence.lua:77
	if cfg != nil {
		if t, ok := a.CfgConversionTable[cfg]; ok {
			return t
		}
	}
	return a.ConversionTable
}

//pob:src/Modules/CalcOffence.lua:160-166
func calcAreaMod(incArea float64, moreArea float64) float64 {
	baseAreaPercent := math.Round(incArea * 100)
	finalAreaPercent := math.Round(max(0, moreArea) * 100)
	areaPercent := baseAreaPercent * finalAreaPercent / 100
	if areaPercent >= 0 {
		areaPercent = math.Floor(areaPercent)
	} else {
		areaPercent = math.Ceil(areaPercent)
	}
	return max(0, areaPercent) / 100
}

//pob:src/Modules/CalcOffence.lua:173-178
func calcRadius(baseRadius float64, areaMod float64, radiusMod float64) float64 {
	radiusPercent := math.Floor(100 * math.Sqrt(areaMod))
	// Final radius modifiers apply after Area of Effect has been converted to radius.
	radiusPercent = math.Floor(radiusPercent * radiusMod)
	return math.Floor(baseRadius * radiusPercent / 100)
}

type radiusBreakpoints struct{ inc, more, red, less int }

//pob:src/Modules/CalcOffence.lua:198-224
func calcRadiusBreakpoints(baseRadius float64, incArea float64, moreArea float64, radiusMod float64) *radiusBreakpoints {
	radius := calcRadius(baseRadius, calcAreaMod(incArea, moreArea), radiusMod)
	if radius <= 0 {
		return nil
	}
	bp := &radiusBreakpoints{}
	for {
		bp.inc++
		if calcRadius(baseRadius, calcAreaMod(incArea+float64(bp.inc)/100, moreArea), radiusMod) > radius {
			break
		}
	}
	for {
		bp.red++
		if calcRadius(baseRadius, calcAreaMod(incArea-float64(bp.red)/100, moreArea), radiusMod) < radius {
			break
		}
	}
	for {
		bp.more++
		if calcRadius(baseRadius, calcAreaMod(incArea, moreArea*(1+float64(bp.more)/100)), radiusMod) > radius {
			break
		}
	}
	for {
		bp.less++
		if calcRadius(baseRadius, calcAreaMod(incArea, moreArea*(1-float64(bp.less)/100)), radiusMod) < radius {
			break
		}
	}
	return bp
}

//pob:src/Modules/CalcOffence.lua:185-190
func calcMoltenStrikeTertiaryRadius(baseRadius float64, deadzoneRadius float64, areaMod float64, speedMod float64) float64 {
	maxDistIgnoringSpeed := math.Sqrt(float64(baseRadius*baseRadius*areaMod) - float64(deadzoneRadius*deadzoneRadius*(areaMod-1)))
	return math.Floor(float64((maxDistIgnoringSpeed-deadzoneRadius)*speedMod) + deadzoneRadius)
}

//pob:src/Modules/CalcOffence.lua:235-294
func setMoltenStrikeTertiaryRadiusBreakdown(breakdown *Breakdown, deadzoneRadius, baseRadius float64, label string, incArea, moreArea, incSpd, moreSpd float64) {
	calc := func(extraIncAoePct, extraMoreAoePct, extraIncSpdPct, extraMoreSpdPct float64) (float64, float64, float64) {
		areaMod := utils.RoundTo(utils.RoundTo((incArea+extraIncAoePct/100)*moreArea*(1+extraMoreAoePct/100), 10), 2)
		speedMod := utils.RoundTo(utils.RoundTo((incSpd+extraIncSpdPct/100)*moreSpd*(1+extraMoreSpdPct/100), 10), 2)
		return calcMoltenStrikeTertiaryRadius(baseRadius, deadzoneRadius, areaMod, speedMod), areaMod, speedMod
	}
	currentDist, currentAreaMod, currentSpeedMod := calc(0, 0, 0, 0)
	entry := breakdown.Data["AreaOfEffectRadiusTertiary"]
	if entry == nil {
		entry = &Entry{}
		breakdown.Data["AreaOfEffectRadiusTertiary"] = entry
	}
	entry.Lines = append(entry.Lines, label, " = (sqrt(R*R*a - r*r*(a-1)) - r) * s + r", luaSprintf(" = %d", int(currentDist)))
	if currentDist > 0 {
		findBreakpoint := func(sign float64, argIdx int) int {
			args := [4]float64{}
			for {
				args[argIdx] += sign
				newDist, _, _ := calc(args[0], args[1], args[2], args[3])
				if newDist != currentDist || newDist == 0 {
					break
				}
			}
			return int(args[argIdx] * sign)
		}
		entry.Lines = append(entry.Lines,
			luaSprintf("^8Next AoE breakpoint: %d%% increased or %d%% more", findBreakpoint(1, 0), findBreakpoint(1, 1)),
			luaSprintf("^8Next Proj. Speed breakpoint: %d%% increased or %d%% more", findBreakpoint(1, 2), findBreakpoint(1, 3)),
			luaSprintf("^8Previous AoE breakpoint: %d%% increased or %d%% more", findBreakpoint(-1, 0), findBreakpoint(-1, 1)),
			luaSprintf("^8Previous Proj. Speed breakpoint: %d%% increased or %d%% more", findBreakpoint(-1, 2), findBreakpoint(-1, 3)))
	}
	entry.Label = "Inputs"
	entry.Rows = nil
	entry.Columns = []BCol{{"Variable", "name"}, {"Value", "value"}, {"Description", "description"}}
	entry.Rows = append(entry.Rows,
		map[string]string{"name": "r", "value": luaSprintf("%d", int(deadzoneRadius)), "description": "fixed deadzone radius"},
		map[string]string{"name": "R", "value": luaSprintf("%d", int(baseRadius)), "description": "base outer radius"},
		map[string]string{"name": "a", "value": luaSprintf("%.2f", currentAreaMod), "description": "net AoE multiplier (scales area)"},
		map[string]string{"name": "s", "value": luaSprintf("%.2f", currentSpeedMod), "description": "net projectile speed multiplier (scales range)"})
	entry.Radius = currentDist
}

//pob:src/Modules/CalcOffence.lua:1083
var pointBlankDistanceRamp = [][]float64{{10, 1}, {35, 0}, {120, -1}}

//pob:src/Modules/CalcOffence.lua:1086
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

//pob:block:src/Modules/CalcOffence.lua:5813-5818
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

//pob:block:src/Modules/CalcOffence.lua:1477-1478
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

//pob:src/Modules/CalcOffence.lua:296-309
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

//pob:src/Modules/CalcOffence.lua:291-300
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

//pob:src/Modules/CalcOffence.lua:302-314
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

//pob:block:src/Export/Scripts/miscdata.lua:96-103
func totemLifeMult(skillTotemID int) (float64, bool) {
	for _, variation := range poe.SkillTotemVariations {
		if variation.SkillTotemsKey != skillTotemID {
			continue
		}
		return float64(raw.MonsterVarieties()[variation.MonsterVarietiesKey].LifeMultiplier) / 100, true
	}
	return 0, false
}

//pob:block:src/Modules/CalcMirages.lua:125
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

//pob:src/Modules/CalcOffence.lua:316-4216
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

	//pob:src/Modules/CalcOffence.lua:494-507
	dotSourceRes := func(damageType data.DamageType) data.DamageType {
		if env.ModDB.Flag(nil, "Enemy"+string(damageType)+"ResistEqualToYours") {
			return data.DamageType("Your " + string(damageType) + " Resistance")
		}
		return damageType
	}

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
		//pob:src/Modules/CalcOffence.lua:375-377
		incArea, moreArea := calclib.Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectPrimary")
		output["AreaOfEffectMod"] = calcAreaMod(incArea, moreArea)
		radiusMod := calclib.Mod(skillModList, skillCfg, "AreaOfEffectRadius")
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
			output["AreaOfEffectRadius"] = calcRadius(baseRadius, output["AreaOfEffectMod"], radiusMod)
			output["AreaOfEffectRadiusMetres"] = output["AreaOfEffectRadius"] / 10
			if breakdown != nil {
				breakdown.area("AreaOfEffectRadius", baseRadius, output["AreaOfEffectMod"], output["AreaOfEffectRadius"], calcRadiusBreakpoints(baseRadius, incArea, moreArea, radiusMod), skillData.RadiusLabel, radiusMod)
			}
			if skillData.RadiusSecondary != 0 {
				incAreaSecondary, moreAreaSecondary := calclib.Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectSecondary")
				output["AreaOfEffectModSecondary"] = calcAreaMod(incAreaSecondary, moreAreaSecondary)
				baseRadius = skillData.RadiusSecondary + skillData.RadiusExtra
				output["AreaOfEffectRadiusSecondary"] = calcRadius(baseRadius, output["AreaOfEffectModSecondary"], radiusMod)
				output["AreaOfEffectRadiusSecondaryMetres"] = output["AreaOfEffectRadiusSecondary"] / 10
				if breakdown != nil {
					var bp *radiusBreakpoints
					if !skillData.ProjectileSpeedAppliesToMSAreaOfEffect {
						bp = calcRadiusBreakpoints(baseRadius, incAreaSecondary, moreAreaSecondary, radiusMod)
					}
					breakdown.area("AreaOfEffectRadiusSecondary", baseRadius, output["AreaOfEffectModSecondary"], output["AreaOfEffectRadiusSecondary"], bp, skillData.RadiusSecondaryLabel, radiusMod)
				}
			}
			if skillData.RadiusTertiary != 0 {
				incAreaTertiary, moreAreaTertiary := calclib.Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectTertiary")
				output["AreaOfEffectModTertiary"] = calcAreaMod(incAreaTertiary, moreAreaTertiary)
				baseRadius = skillData.RadiusTertiary + skillData.RadiusExtra
				if skillData.ProjectileSpeedAppliesToMSAreaOfEffect {
					incSpeedTertiary, moreSpeedTertiary := calclib.Mods(skillModList, skillCfg, "ProjectileSpeed")
					output["SpeedModTertiary"] = utils.RoundTo(utils.RoundTo(incSpeedTertiary*moreSpeedTertiary, 10), 2)
					output["AreaOfEffectRadiusTertiary"] = calcMoltenStrikeTertiaryRadius(baseRadius, skillData.RadiusSecondary, output["AreaOfEffectModTertiary"], output["SpeedModTertiary"])
					//pob:block:src/Modules/CalcOffence.lua:419-424
					if breakdown != nil {
						setMoltenStrikeTertiaryRadiusBreakdown(breakdown, skillData.RadiusSecondary, baseRadius, skillData.RadiusTertiaryLabel,
							incAreaTertiary, moreAreaTertiary, incSpeedTertiary, moreSpeedTertiary)
					}
				} else if skillData.RadiusTertiaryBaseMargin != 0 {
					//pob:block:src/Modules/CalcOffence.lua:425-445
					margin := skillData.RadiusTertiaryBaseMargin / 100
					marginWidth := skillData.RadiusTertiaryBaseMargin*2 + 1
					baseRadiiOccurrences := make(map[float64]float64)
					for deviation := 1 - margin; deviation <= 1+margin+0.01; deviation += 0.01 {
						radiusForDeviation := math.Floor(baseRadius * deviation)
						baseRadiiOccurrences[radiusForDeviation]++
					}
					sumOfRandomRadii := float64(0)
					radiusForBaseRadius := make(map[float64]float64)
					radiiOccurrences := make(map[float64]float64)
					for adjustedBaseRadius, occurrenceCount := range baseRadiiOccurrences {
						radiusForDeviation := calcRadius(adjustedBaseRadius, output["AreaOfEffectModTertiary"], radiusMod)
						radiusForBaseRadius[adjustedBaseRadius] = radiusForDeviation
						sumOfRandomRadii += float64(radiusForDeviation * occurrenceCount)
						radiiOccurrences[radiusForDeviation] += occurrenceCount
					}
					output["AreaOfEffectRadiusTertiary"] = sumOfRandomRadii / marginWidth
					activeSkill.areaOfEffectRadiusTertiaryOccurrences = radiiOccurrences
					//pob:block:src/Modules/CalcOffence.lua:446-471
					if breakdown != nil {
						var out []string
						bp := radiusBreakpoints{math.MaxInt, math.MaxInt, math.MaxInt, math.MaxInt}
						if skillData.RadiusTertiaryLabel != "" {
							out = append(out, skillData.RadiusTertiaryLabel)
						}
						out = append(out, luaSprintf("R ^8(base radius)^7 x %.2f ^8(square root of area of effect modifier)", math.Floor(100*math.Sqrt(output["AreaOfEffectModTertiary"]))/100))
						if radiusMod != 1 {
							out = append(out, luaSprintf("x %.2f ^8(final radius modifier)", radiusMod))
						}
						baseRadii := slices.Sorted(maps.Keys(baseRadiiOccurrences))
						for _, adjustedBaseRadius := range baseRadii {
							out = append(out, luaSprintf("%.1f%% ^8chance of^7 %.1fm ^8base radius resulting in^7 %.1fm ^8final radius", baseRadiiOccurrences[adjustedBaseRadius]/marginWidth*100, adjustedBaseRadius/10, radiusForBaseRadius[adjustedBaseRadius]/10))
							if intermediate := calcRadiusBreakpoints(adjustedBaseRadius, incAreaTertiary, moreAreaTertiary, radiusMod); intermediate != nil {
								bp.inc = min(bp.inc, intermediate.inc)
								bp.more = min(bp.more, intermediate.more)
								bp.red = min(bp.red, intermediate.red)
								bp.less = min(bp.less, intermediate.less)
							}
						}
						out = append(out,
							luaSprintf("^8Next closest 0.1m breakpoint: %d%% increased AoE / a %d%% more AoE multiplier", bp.inc, bp.more),
							luaSprintf("^8Previous closest 0.1m breakpoint: %d%% reduced AoE / a %d%% less AoE multiplier", bp.red, bp.less),
							luaSprintf("On average, the radius is %.2fm", output["AreaOfEffectRadiusTertiary"]/10),
						)
						breakdown.Data["AreaOfEffectRadiusTertiary"] = &Entry{Lines: out}
					}
				} else {
					output["AreaOfEffectRadiusTertiary"] = calcRadius(baseRadius, output["AreaOfEffectModTertiary"], radiusMod)
					if breakdown != nil {
						breakdown.area("AreaOfEffectRadiusTertiary", baseRadius, output["AreaOfEffectModTertiary"], output["AreaOfEffectRadiusTertiary"], calcRadiusBreakpoints(baseRadius, incAreaTertiary, moreAreaTertiary, radiusMod), skillData.RadiusTertiaryLabel, radiusMod)
					}
				}
				output["AreaOfEffectRadiusTertiaryMetres"] = output["AreaOfEffectRadiusTertiary"] / 10
			}
		}
		//pob:block:src/Modules/CalcOffence.lua:482-491
		if breakdown != nil {
			breakdown.Data["AreaOfEffectMod"] = &Entry{}
			if output["AreaOfEffectMod"] != 1 {
				breakdown.MultiChain("AreaOfEffectMod", BMultiChain{
					Total: luaSprintf("= %.2f", output["AreaOfEffectMod"]),
					Items: []BMultiChainItem{
						{"%.2f ^8(increased/reduced)", 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "AreaOfEffect")/100},
						{"%.2f ^8(more/less)", skillModList.More(skillCfg, "AreaOfEffect")},
					},
				})
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
	//pob:block:src/Modules/CalcOffence.lua:518-521
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
			utils.Set(skillData, cVal.Key, cVal.SetValue())
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
	// set other limits
	actor.Output["ActiveTrapLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveTrapLimit")
	actor.Output["ActiveMineLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveMineLimit")

	// set flask scaling
	//pob:block:src/Modules/CalcOffence.lua:565-566
	actor.Output["LifeFlaskRecovery"] = env.ItemModDB.Multipliers["LifeFlaskRecovery"]
	actor.Output["LifeFlaskCharges"] = env.ItemModDB.Multipliers["LifeFlaskCharges"]

	//pob:block:src/Modules/CalcOffence.lua:568-583
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
	//pob:block:src/Modules/CalcOffence.lua:595-612
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
	//pob:block:src/Modules/CalcOffence.lua:613
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
	//pob:block:src/Modules/CalcOffence.lua:644-650
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
					//pob:block:src/Modules/CalcOffence.lua:659-662
					skillModList.ReplaceMod(mod.NewFloat("PhysicalDamage", mod.TypeIncrease, 0).Source("Strength").Flag(mod.MFlagMelee))
					skillModList.ReplaceMod(mod.NewFloat("PhysicalDamage", mod.TypeIncrease, 0).Source("Strength").Flag(mod.MFlagAttack | mod.MFlagProjectile))
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
	//pob:block:src/Modules/CalcOffence.lua:731-758
	for _, critFrom := range []struct{ flag, stat string }{
		{"CritChanceIncreasedByUncappedLightningRes", "LightningResistTotal"},
		{"CritChanceIncreasedByLightningRes", "LightningResist"},
		{"CritChanceIncreasedByOvercappedLightningRes", "LightningResistOverCap"},
		{"CritChanceIncreasedBySpellSuppressChance", "SpellSuppressionChance"},
	} {
		if skillModList.Flag(nil, critFrom.flag) {
			if tab := modDB.Tabulate(mod.TypeFlag, nil, critFrom.flag); len(tab) > 0 {
				skillModList.AddMod(mod.NewFloat("CritChance", mod.TypeIncrease, actor.Output[critFrom.stat]).
					Source(tab[0].Mod.GetSource()))
			}
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:759-765
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
	//pob:src/Modules/CalcOffence.lua:803-806
	if skillModList.Flag(nil, "SequentialProjectiles") && !skillModList.Flag(nil, "OneShotProj") && !skillModList.Flag(nil, "NoAdditionalProjectiles") && !skillModList.Flag(nil, "SingleProjectile") && !skillModList.Flag(nil, "TriggeredBySnipe") {
		// Applies DPS multiplier based on projectile count
		skillData.DpsMultiplier = skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount")
	}

	//pob:block:src/Modules/CalcOffence.lua:808-819
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
	//pob:block:src/Modules/CalcOffence.lua:820-903
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
			modValue := float64(raw * (repeats - 1))
			if repeatMode == "AVERAGE" {
				if damageFinalMoreValueTotal != 1 {
					modValue = (float64((100+raw*(repeats-2)/2)*(repeats-1))+
						float64((100+float64(raw*(repeats-1)))*damageFinalMoreValueTotal))/
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
					(float64(100*repeats)+damageFinalMoreValueTotal)/(1+damageFinalMoreValueTotal/100)-100))
			}
		}
		if skillFlags[SkillFlagTrap] || skillFlags[SkillFlagMine] {
			skillModList.AddMod(mod.NewFloat("DPS", mod.TypeMore, (repeats-1)*100).Source("Repeat Count"))
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:905-912
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
	//pob:block:src/Modules/CalcOffence.lua:913-931
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
		//pob:block:src/Modules/CalcOffence.lua:948
		actor.Output["AverageBurstHits"] = actor.Output["SealMax"]
		actor.Output["TimeMaxSeals"] = actor.Output["SealCooldown"] * actor.Output["SealMax"]

		if skillData.HitTimeOverride == 0 {
			if skillModList.Flag(nil, "UseMaxUnleash") {
				for _, value := range skillModList.Tabulate("INC", nil, "MaxSealCrit") {
					skillModList.AddMod(mod.NewFloat("CritChance", "INC", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
				}
				//pob:block:src/Modules/CalcOffence.lua:957-960
				for _, value := range skillModList.Tabulate("MORE", nil, "MaxSealDamage") {
					skillModList.AddMod(mod.NewFloat("Damage", "MORE", value.Mod.Value().Float()*actor.Output["SealMax"]).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
				}
				env.Player.MainSkill.SkillData.DpsMultiplier = 1 + float64(actor.Output["SealMax"]*calclib.Mod(skillModList, skillCfg, "SealRepeatPenalty"))
				env.Player.MainSkill.SkillData.HitTimeOverride = max(actor.Output["TimeMaxSeals"], 1/activeSkill.ActiveEffect.GrantedEffect.CastTime()*1.1*calclib.Mod(skillModList, skillCfg, "Speed")*actor.Output["ActionSpeedMod"])
			} else {
				env.Player.MainSkill.SkillData.DpsMultiplier = 1 + float64(1/actor.Output["SealCooldown"]/(1/activeSkill.ActiveEffect.GrantedEffect.CastTime()*1.1*calclib.Mod(skillModList, skillCfg, "Speed")*actor.Output["ActionSpeedMod"])*calclib.Mod(skillModList, skillCfg, "SealRepeatPenalty"))
			}
		}

		if breakdown != nil {
			breakdown.MultiChain("SealGainTime", BMultiChain{
				Label: "Gain frequency:",
				Base:  &BMultiChainItem{"%.2fs ^8(base gain frequency)", skillModList.Sum(mod.TypeBase, skillCfg, "SealGainFrequency")},
				Total: luaSprintf("= %.2fs ^8per Seal", actor.Output["SealCooldown"]),
				Items: []BMultiChainItem{
					{"%.2f ^8(increased/reduced gain frequency)", 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "SealGainFrequency")/100},
					{"%.2f ^8(action speed modifier)", actor.Output["ActionSpeedMod"]},
				},
			})
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:979-1034
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

	//pob:block:src/Modules/CalcOffence.lua:1037-1050
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
		//pob:block:src/Modules/CalcOffence.lua:1061-1064
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
					float64(10*skillModList.Sum(mod.TypeBase, activeSkill.Weapon1Cfg, "MeleeWeaponRangeMetre"))
			} else {
				actor.WeaponRange1 = 6 + skillModList.Sum(mod.TypeBase, skillCfg, "UnarmedRange") +
					float64(10*skillModList.Sum(mod.TypeBase, skillCfg, "UnarmedRangeMetre"))
			}
		}
		if skillFlags[SkillFlagWeapon2Attack] {
			if actor.WeaponData2.Range != 0 {
				actor.WeaponRange2 = actor.WeaponData2.Range +
					skillModList.Sum(mod.TypeBase, activeSkill.Weapon2Cfg, "MeleeWeaponRange") +
					float64(10*skillModList.Sum(mod.TypeBase, activeSkill.Weapon2Cfg, "MeleeWeaponRangeMetre"))
			} else {
				actor.WeaponRange2 = 6 + skillModList.Sum(mod.TypeBase, skillCfg, "UnarmedRange") +
					float64(10*skillModList.Sum(mod.TypeBase, skillCfg, "UnarmedRangeMetre"))
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
			//pob:block:src/Modules/CalcOffence.lua:1172-1181
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
	//pob:src/Modules/CalcOffence.lua:1193-1212
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
	//pob:block:src/Modules/CalcOffence.lua:1219-1225
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
			breakdown.AddLine("WarcryEffectMod", luaSprintf("%.2f ^8(effect modifiers)", unscaledEffect))
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
		//pob:block:src/Modules/CalcOffence.lua:1264-1271
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
		areaMod := calcAreaMod(incArea, moreArea)
		actor.Output["TrapTriggerRadius"] = calcRadius(data.TrapTriggerRadiusBase, areaMod, 1)
		actor.Output["TrapTriggerRadiusMetre"] = actor.Output["TrapTriggerRadius"] / 10
		if actor.Breakdown != nil {
			actor.Breakdown.area("TrapTriggerRadius", data.TrapTriggerRadiusBase, areaMod, actor.Output["TrapTriggerRadius"], calcRadiusBreakpoints(data.TrapTriggerRadiusBase, incArea, moreArea, 1), "", 1)
		}
	} else if skillData.Cooldown != 0 || skillModList.Sum(mod.TypeBase, skillCfg, "CooldownRecovery") != 0 {
		actor.Output["Cooldown"] = calcSkillCooldown(skillModList, skillCfg, skillData)
		//pob:block:src/Modules/CalcOffence.lua:1322-1331
		if breakdown != nil {
			addedCooldown := skillModList.Sum(mod.TypeBase, skillCfg, "CooldownRecovery")
			rounded := !((skillData.StoredUses != nil && *skillData.StoredUses > 1) || skillModList.Sum(mod.TypeBase, skillCfg, "AdditionalCooldownUses") > 0)
			breakdown.Data["Cooldown"] = &Entry{Lines: []string{
				luaSprintf("%.2fs ^8(base)", utils.OrF(skillData.Cooldown, 0+addedCooldown)),
				luaSprintf("/ %.2f ^8(increased/reduced cooldown recovery)", 1+skillModList.Sum(mod.TypeIncrease, skillCfg, "CooldownRecovery")/100),
			}}
			if rounded {
				breakdown.AddLine("Cooldown", "rounded up to nearest server tick")
			}
			breakdown.AddLine("Cooldown", luaSprintf("= %.3fs", actor.Output["Cooldown"]))
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:986-997
	if skillData.StoredUses != nil {
		baseUses := *skillData.StoredUses
		additionalUses := skillModList.Sum(mod.TypeBase, skillCfg, "AdditionalCooldownUses", "AdditionalUses")
		actor.Output["StoredUses"] = baseUses + additionalUses
		//pob:block:src/Modules/CalcOffence.lua:1337-1343
		if breakdown != nil {
			breakdown.Data["StoredUses"] = &Entry{Lines: []string{luaSprintf("%d ^8(skill use%s)", luaInt(baseUses), utils.Ternary(baseUses == 1, "", "s"))}}
			if additionalUses != 0 {
				breakdown.AddLine("StoredUses",
					luaSprintf("+ %d ^8(additional use%s)", luaInt(additionalUses), utils.Ternary(additionalUses == 1, "", "s")),
					luaSprintf("= %d ^8(total use%s)", luaInt(actor.Output["StoredUses"]), utils.Ternary(actor.Output["StoredUses"] == 1, "", "s")),
				)
			}
		}
	}
	if skillFlags[SkillFlagMine] {
		baseSpeed := 1 / skillModList.Sum(mod.TypeBase, skillCfg, "MineLayingTime")
		timeMod := calclib.Mod(skillModList, skillCfg, "SkillMineThrowingTime")
		if timeMod > 0 {
			baseSpeed = baseSpeed * (1 / timeMod)
		}
		actor.Output["MineLayingSpeed"] = baseSpeed * calclib.Mod(skillModList, skillCfg, "MineLayingSpeed") * actor.Output["ActionSpeedMod"]
		//pob:block:src/Modules/CalcOffence.lua:1353-1370
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
			actor.Output["MineLayingSpeed"] = actor.Output["MineLayingSpeed"] / (1 + float64((actor.Output["MineThrowCount"]-1)*0.1))
		}
		actor.Output["MineLayingSpeed"] = min(actor.Output["MineLayingSpeed"], data.ServerTickRate)
		actor.Output["MineLayingTime"] = 1 / actor.Output["MineLayingSpeed"]
		if skillFlags[SkillFlagTrap] {
			skillData.TimeOverride = actor.Output["MineLayingTime"] / actor.Output["MineThrowCount"] / actor.Output["TrapThrowCount"]
		} else {
			skillData.TimeOverride = actor.Output["MineLayingTime"] / actor.Output["MineThrowCount"]
		}

		incArea, moreArea := calclib.Mods(skillModList, skillCfg, "MineDetonationAreaOfEffect")
		areaMod := calcAreaMod(incArea, moreArea)
		actor.Output["MineDetonationRadius"] = calcRadius(data.MineDetonationRadiusBase, areaMod, 1)
		actor.Output["MineDetonationRadiusMetre"] = actor.Output["MineDetonationRadius"] / 10
		if actor.Breakdown != nil {
			actor.Breakdown.area("MineDetonationRadius", data.MineDetonationRadiusBase, areaMod, actor.Output["MineDetonationRadius"], calcRadiusBreakpoints(data.MineDetonationRadiusBase, incArea, moreArea, 1), "", 1)
		}

		if activeSkill.SkillTypes[data.SkillTypeAura] {
			actor.Output["MineAuraRadius"] = calcRadius(data.MineAuraRadiusBase, actor.Output["AreaOfEffectMod"], 1)
			actor.Output["MineAuraRadiusMetre"] = actor.Output["MineAuraRadius"] / 10
			if actor.Breakdown != nil {
				incArea, moreArea := calclib.Mods(skillModList, skillCfg, "AreaOfEffect")
				actor.Breakdown.area("MineAuraRadius", data.MineAuraRadiusBase, actor.Output["AreaOfEffectMod"], actor.Output["MineAuraRadius"], calcRadiusBreakpoints(data.MineAuraRadiusBase, incArea, moreArea, 1), "", 1)
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
				Total: luaSprintf("= %.2f ^8per second", actor.Output["TotemPlacementSpeed"]),
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
		//pob:block:src/Modules/CalcOffence.lua:1441-1443
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
	//pob:src/Modules/CalcOffence.lua:1098-1100
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
					luaSprintf("%d ^8(base life of a level %d monster)", env.data.monsterLifeTable[actor.Output["CorpseLevel"] or 1], actor.Output["CorpseLevel"] or "n/a"),
					luaSprintf("x %.2f ^8(%s variety multiplier)", env.data.monsterVarietyLifeMult[skillData.corpseMonsterVariety] or 1, skillData.corpseMonsterVariety),
					luaSprintf("x %.2f ^8(map level %d monster life multiplier from config)", env.data.mapLevelLifeMult[env.enemyLevel] or 1, env.enemyLevel),
					luaSprintf(" = %d ^8(base corpse life)", actor.Output["BaseCorpseLife"]),
					luaSprintf(""),
					luaSprintf("x %.2f ^8(corpse maximum life increases)", actor.Output["CorpseLifeInc"]),
					luaSprintf(" = %d", actor.Output["CorpseLife"]),
				}
			}
		}
	*/
	//pob:block:src/Modules/CalcOffence.lua:1085-1137
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

			//pob:src/Modules/CalcMirages.lua:384-388
			if !activeSkill.SkillTypes[data.SkillTypeChannel] {
				skillData.HitTimeOverride = 1
			}

			// Supported Attacks Count as Exerted.
			//pob:src/Modules/CalcMirages.lua:401-421
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
			//pob:block:src/Modules/CalcMirages.lua:382-391,src/Modules/CalcMirages.lua:423-424
			mirageSpawnTime := 0.3 + float64(0.2*maxMirageWarriors)
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
		//pob:src/Modules/CalcOffence.lua:1497
		actor.Output["DurationMod"] = max(actor.Output["DurationMod"], 0)
		if breakdown != nil {
			breakdown.Mod("DurationMod", skillModList, skillCfg, "Duration", "PrimaryDuration", utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
			//pob:block:src/Modules/CalcOffence.lua:1500-1502
			if breakdown.Has("DurationMod") && skillData.DurationSecondary != 0 {
				breakdown.Data["DurationMod"].Lines = append([]string{"Primary duration:"}, breakdown.Data["DurationMod"].Lines...)
			}
		}
		durationBase := (skillData.Duration) + skillModList.Sum(mod.TypeBase, skillCfg, "Duration", "PrimaryDuration")
		if durationBase > 0 {
			actor.Output["Duration"] = durationBase * actor.Output["DurationMod"]
			if skillData.Debuff {
				actor.Output["Duration"] = actor.Output["Duration"] * debuffDurationMult
			}
			actor.Output["Duration"] = math.Ceil(actor.Output["Duration"]*data.ServerTickRate) / data.ServerTickRate
			//pob:block:src/Modules/CalcOffence.lua:1511-1514
			if breakdown != nil && actor.Output["Duration"] != durationBase {
				breakdown.Data["Duration"] = &Entry{Lines: []string{
					luaSprintf("%.2fs ^8(base)", durationBase),
				}}
				if actor.Output["DurationMod"] != 1 {
					breakdown.AddLine("Duration", luaSprintf("x %.4f ^8(duration modifier)", actor.Output["DurationMod"]))
				}
				if skillData.Debuff && debuffDurationMult != 1 {
					breakdown.AddLine("Duration", luaSprintf("/ %.3f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
				}
				breakdown.AddLine("Duration", "rounded up to nearest server tick")
				breakdown.AddLine("Duration", luaSprintf("= %.3fs", actor.Output["Duration"]))
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
				//pob:block:src/Modules/CalcOffence.lua:1536-1541
				if breakdown.Has("SecondaryDurationMod") {
					breakdown.Data["SecondaryDurationMod"].Lines = append([]string{"Secondary duration:"}, breakdown.Data["SecondaryDurationMod"].Lines...)
				}
				breakdown.Data["DurationSecondary"] = &Entry{Lines: []string{
					luaSprintf("%.2fs ^8(base)", durationBase),
				}}
				if actor.Output["DurationMod"] != 1 {
					breakdown.AddLine("DurationSecondary", luaSprintf("x %.4f ^8(duration modifier)", durationMod))
				}
				if skillData.DebuffSecondary && debuffDurationMult != 1 {
					breakdown.AddLine("DurationSecondary", luaSprintf("/ %.3f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
				}
				breakdown.AddLine("DurationSecondary", "rounded up to nearest server tick")
				breakdown.AddLine("DurationSecondary", luaSprintf("= %.3fs", actor.Output["DurationSecondary"]))
			}
		}
		//pob:block:src/Modules/CalcOffence.lua:1552-1560
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
					luaSprintf("%.2fs ^8(base)", durationBase),
					luaSprintf("x %.4f ^8(duration modifier)", durationMod),
					"rounded up to nearest server tick",
					luaSprintf("= %.3fs", actor.Output["AuraDuration"]),
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
					luaSprintf("%.2fs ^8(base)", durationBase),
					luaSprintf("x %.4f ^8(duration modifier)", durationMod),
					"rounded up to nearest server tick",
					luaSprintf("= %.3fs", actor.Output["ReserveDuration"]),
				)
			}
		}

		//pob:block:src/Modules/CalcOffence.lua:1609-1623
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
					luaSprintf("%.2fs ^8(base)", durationBase),
					luaSprintf("x %.4f ^8(duration modifier)", durationMod),
					"rounded up to nearest server tick",
					luaSprintf("= %.3fs", actor.Output["SoulGainPreventionDuration"]),
				)
			}
		}
		//pob:block:src/Modules/CalcOffence.lua:1624-1627
		actor.Output["TotemDurationMod"] = math.Max(calclib.Mod(skillModList, skillCfg, "TotemDuration"), 0)
		totemDurationBase := skillModList.Sum(mod.TypeBase, skillCfg, "TotemDuration")
		actor.Output["TotemDuration"] = math.Ceil(totemDurationBase*actor.Output["TotemDurationMod"]*data.ServerTickRate) / data.ServerTickRate
		if breakdown != nil {
			breakdown.Mod("TotemDurationMod", skillModList, skillCfg, "TotemDuration")
			breakdown.AddLine("TotemDuration", luaSprintf("%.2fs ^8(base)", totemDurationBase))
			if actor.Output["TotemDurationMod"] != 1 {
				breakdown.AddLine("TotemDuration", luaSprintf("x %.4f ^8(duration modifier)", actor.Output["TotemDurationMod"]))
			}
			breakdown.AddLine("TotemDuration", "rounded up to nearest server tick")
			breakdown.AddLine("TotemDuration", luaSprintf("= %.3fs", actor.Output["TotemDuration"]))
		}
		//pob:block:src/Modules/CalcOffence.lua:1639-1642
		actor.Output["ImpaleDurationMod"] = math.Max(calclib.Mod(skillModList, skillCfg, "ImpaleDuration"), 0)
		impaleDurationBase := float64(data.ImpaledDebuffBaseDurationMs) / 1000
		actor.Output["ImpaleDuration"] = math.Ceil(impaleDurationBase*actor.Output["ImpaleDurationMod"]*data.ServerTickRate*debuffDurationMult) / data.ServerTickRate
		//pob:block:src/Modules/CalcOffence.lua:1643-1656
		if breakdown != nil {
			breakdown.Mod("ImpaleDurationMod", skillModList, skillCfg, "ImpaleDuration")
			breakdown.Data["ImpaleDuration"] = &Entry{Lines: []string{luaSprintf("%.2fs ^8(base)", impaleDurationBase)}}
			if debuffDurationMult != 1 {
				breakdown.AddLine("ImpaleDuration", luaSprintf("/ %.3f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
			}
			if actor.Output["ImpaleDurationMod"] != 1 {
				breakdown.AddLine("ImpaleDuration", luaSprintf("x %.4f ^8(duration modifier)", actor.Output["ImpaleDurationMod"]))
			}
			breakdown.AddLine("ImpaleDuration", "rounded up to nearest server tick", luaSprintf("= %.3fs", actor.Output["ImpaleDuration"]))
		}
	}

	// Skill uptime.
	//pob:block:src/Modules/CalcOffence.lua:1659-1689
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
						luaSprintf("%.2fs / (%.2fs + %.2fs)", duration, cooldown, duration),
						luaSprintf("= %d%%", luaInt(actor.Output[durationType+"Uptime"])),
					)
				} else {
					breakdown.AddLine(durationType+"Uptime",
						luaSprintf("%.2fs / %.2fs", duration, cooldown),
						luaSprintf("= %d%%", luaInt(actor.Output[durationType+"Uptime"])),
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

	costOrder := offenceCostOrder

	hasCost := map[string]bool{}
	//pob:block:src/Modules/CalcOffence.lua:1711-1717
	if !skillModList.Flag(skillCfg, "HasNoCost") {
		mult := math.Floor(float64(skillModList.More(skillCfg, "SupportManaMultiplier")*10000)+0.0001) / 10000
		additionalLifeCost := skillModList.Sum(mod.TypeBase, skillCfg, "ManaCostAsLifeCost") / 100
		additionalESCost := skillModList.Sum(mod.TypeBase, skillCfg, "ManaCostAsEnergyShieldCost") / 100
		hybridLifeCost := skillModList.Sum(mod.TypeBase, skillCfg, "HybridManaAndLifeCost_Life") / 100

		// First pass to calculate base costs. Used for cost conversion (e.g. Petrified Blood)
		//pob:block:src/Modules/CalcOffence.lua:1718-1800
		for ri, resource := range costOrder {
			names := &offenceCostNames[ri]
			val := costs[resource]
			skillCost := float64(activeSkill.ActiveEffect.GrantedEffectLevel.Cost[resource])
			if override := skillModList.Override(skillCfg, names.baseOverride); override != nil {
				skillCost = override.ValueFloat
			}
			baseCost := utils.RoundTo(utils.Ternary[float64](skillCost > 0, skillCost/costDivisors[resource], 0), 2)
			baseCostNoMult := skillModList.Sum(mod.TypeBase, skillCfg, names.noMult)
			divineBlessingCorrection := float64(0)
			if val.Upfront {
				baseCost = baseCost + skillModList.Sum(mod.TypeBase, skillCfg, names.base)
				if resource == "Mana" && skillData.BaseManaCostIsAtLeastPercentUnreservedMana > 0 {
					baseCost = max(baseCost, math.Floor((actor.Output["ManaUnreserved"])*skillData.BaseManaCostIsAtLeastPercentUnreservedMana/100))
				}
				val.TotalCost = skillModList.Sum(mod.TypeBase, skillCfg, names.cost, "Cost")
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
				raw := float64(val.BaseCost*mult) + val.BaseCostNoMult + divineBlessingCorrection
				val.BaseCostRaw = &raw
			}

			switch val.Type {
			case "Life":
				manaType := strings.ReplaceAll(resource, "Life", "Mana")
				if skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") { // Blood Magic / Lifetap
					if val.Upfront {
						manaTotalCost := skillModList.Sum(mod.TypeBase, skillCfg, manaType+"Cost")
						val.TotalCost = val.TotalCost + max(0, manaTotalCost)
						costs[manaType].TotalCost = costs[manaType].TotalCost - manaTotalCost
					}
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
					if val.Upfront && hybridLifeCost > 0 {
						// Only positive flat mana cost is converted, and it is rounded separately from base cost.
						val.TotalCost = val.TotalCost + utils.RoundTo(max(0, skillModList.Sum(mod.TypeBase, skillCfg, manaType+"Cost"))*hybridLifeCost, 0)
					}
				}
			case "ES":
				manaType := strings.ReplaceAll(resource, "ES", "Mana")
				lifeType := strings.ReplaceAll(resource, "ES", "Life")
				if skillModList.Flag(skillCfg, "CostESInsteadOfManaOrLife") { // Whispers of Infinity
					if val.Upfront {
						manaTotalCost := skillModList.Sum(mod.TypeBase, skillCfg, manaType+"Cost")
						lifeTotalCost := skillModList.Sum(mod.TypeBase, skillCfg, lifeType+"Cost")
						val.TotalCost = val.TotalCost + max(0, manaTotalCost) + max(0, lifeTotalCost)
						costs[manaType].TotalCost = costs[manaType].TotalCost - manaTotalCost
						costs[lifeType].TotalCost = costs[lifeType].TotalCost - lifeTotalCost
					}
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

		//pob:block:src/Modules/CalcOffence.lua:1801-1858
		for ri, resource := range costOrder {
			val := costs[resource]
			keyNames := &offenceCostNames[ri].perSecond
			if val.Upfront {
				keyNames = &offenceCostNames[ri].upfront
			}
			costKey := keyNames.key
			costName := keyNames.cost
			costNameRaw := keyNames.costRaw
			hasCost[costKey] = val.BaseCost > 0 || val.TotalCost > 0 || val.BaseCostNoMult > 0 || val.FinalBaseCost > 0
			actor.Output[keyNames.hasCost] = utils.Ternary(hasCost[costKey], 1.0, 0.0)

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
				if val.Upfront && !val.Percent {
					// The game stores upfront resource costs as whole numbers after efficiency.
					actor.Output[costName] = utils.RoundTo(actor.Output[costName], 0)
				}
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
				if val.Upfront && !val.Percent {
					// The game stores upfront resource costs as whole numbers after efficiency.
					actor.Output[costName] = utils.RoundTo(actor.Output[costName], 0)
				}
				actor.Output[costName] = max(0, actor.Output[costName]+val.TotalCost)
				if val.BaseCostRaw != nil {
					actor.Output[costNameRaw] = max(0, max(0, (1+inc/100)*(*val.BaseCostRaw+val.BaseCostNoMult)*moreType/costEfficiency)+val.TotalCost)
				}
			}
			//pob:block:src/Modules/CalcOffence.lua:1859-1894
			if breakdown != nil && hasCost[costKey] {
				breakdown.Data[costName] = &Entry{Lines: []string{
					luaSprintf("%.2f"+utils.Ternary(val.Percent, "%%", "")+" ^8(base "+val.Text+" cost)", val.BaseCost),
				}}
				if mult != 1 {
					breakdown.AddLine(costName, luaSprintf("x %.4f ^8(cost multiplier)", mult))
				}
				if val.BaseCostNoMult != 0 {
					breakdown.AddLine(costName, luaSprintf("+ %d ^8(additional "+val.Text+" cost)", luaInt(val.BaseCostNoMult)))
				}
				if val.Type == "Life" && (hybridLifeCost+additionalLifeCost) != 0 && !skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") {
					breakdown.AddLine(costName, luaSprintf("x %.2f ^8(mana cost conversion)", hybridLifeCost+additionalLifeCost))
				}
				if val.Type == "ES" && additionalESCost != 0 && !skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") && !skillModList.Flag(skillCfg, "CostESInsteadOfManaOrLife") {
					breakdown.AddLine(costName, luaSprintf("x %.2f ^8(mana cost conversion)", additionalESCost))
				}
				if inc != 0 {
					breakdown.AddLine(costName, luaSprintf("x %.2f ^8(increased/reduced "+val.Text+" cost)", 1+inc/100))
				}
				if moreCost != 1 {
					breakdown.AddLine(costName, luaSprintf("x %.2f ^8(more/less cost)", moreCost))
				}
				if moreType != 1 {
					breakdown.AddLine(costName, luaSprintf("x %.2f ^8(more/less "+val.Text+" cost)", moreType))
				}
				if costEfficiency != 1 {
					breakdown.AddLine(costName, luaSprintf("/ %.2f ^8("+val.Text+" cost efficiency)", costEfficiency))
				}
				if val.TotalCost != 0 {
					breakdown.AddLine(costName, luaSprintf("%+d ^8(total "+val.Text+" cost)", int(val.TotalCost)))
				}
				if val.Type == "Mana" && hybridLifeCost > 0 {
					breakdown.AddLine(costName, luaSprintf("x %.2f ^8(%d%% paid for with life)", 1-hybridLifeCost, luaInt(hybridLifeCost*100)))
				}
				if val.Upfront {
					breakdown.AddLine(costName, luaSprintf("= %d"+utils.Ternary(val.Percent, "%%", ""), luaInt(actor.Output[costName])))
				} else {
					breakdown.AddLine(costName, luaSprintf("= %.2f"+utils.Ternary(val.Percent, "%%", ""), actor.Output[costName]))
				}
			}
		}
	}
	// Eldritch Battery adds maximum Energy Shield to the Mana available for payable-cost damage.
	//pob:block:src/Modules/CalcOffence.lua:1897-1898
	actor.Output["ManaCostPayablePool"] = actor.Output["ManaUnreserved"]
	if modDB.Flag(nil, "EnergyShieldProtectsMana") {
		actor.Output["ManaCostPayablePool"] += actor.Output["EnergyShield"]
	}

	// account for Sacrificial Zeal
	// Note: Sacrificial Zeal grants Added Spell Physical Damage equal to 25% of the Skill's Mana Cost, and causes you to take Physical Damage over Time, for 4 seconds
	//pob:block:src/Modules/CalcOffence.lua:1902-1906
	if skillModList.Flag(nil, "Condition:SacrificialZeal") && actor.Output["ManaHasCost"] != 0 {
		multiplier := 0.25
		skillModList.AddMod(mod.NewFloat("PhysicalMin", mod.TypeBase, math.Floor(actor.Output["ManaCost"]*multiplier)).Source("Sacrificial Zeal").Flag(mod.MFlagSpell))
		skillModList.AddMod(mod.NewFloat("PhysicalMax", mod.TypeBase, math.Floor(actor.Output["ManaCost"]*multiplier)).Source("Sacrificial Zeal").Flag(mod.MFlagSpell))
	}
	runSkillFunc("preDamageFunc")

	//pob:block:src/Modules/CalcOffence.lua:1910-1913
	if activeSkill.SkillTypes[data.SkillTypeBrand] {
		damageLabel := utils.Ternary(skillData.CountsAttachedBrandsInDamage, "Average Damage", "DPS")
		count := int(actor.Output["AttachedBrandCount"])
		activeSkill.infoMessage = luaSprintf("%s for %d attached Brand%s", damageLabel, count, utils.Ternary(count == 1, "", "s"))
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

	//pob:block:src/Modules/CalcOffence.lua:1916
	monsterLife := skillData.CorpseLife
	if monsterLife == 0 {
		monsterLife = 100
		if env.EnemyLevel >= 1 && env.EnemyLevel < len(data.MonsterLifeTable) {
			monsterLife = data.MonsterLifeTable[env.EnemyLevel]
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:1922-1929
	if skillFlags[SkillFlagMonsterExplode] {
		for _, damageType := range data.DamageType("").Values() {
			percentage := utils.GetOr(skillData, string(damageType)+"EffectiveExplodePercentage", float64(0))
			base := percentage * monsterLife / 100
			utils.Set(skillData, string(damageType)+"Min", base)
			utils.Set(skillData, string(damageType)+"Max", base)
		}
	}

	// Cache global damage disabling flags
	//pob:src/Modules/CalcOffence.lua:1931-1935
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
		//pob:block:src/Modules/CalcOffence.lua:2002
		_, preciseTechnique := env.KeystonesAdded["Precise Technique"]
		actor.Output["PreciseTechnique"] = utils.Ternary(preciseTechnique, 1.0, 0.0)
		critOverride := skillModList.Override(skillCfg, "WeaponBaseCritChance")
		if skillFlags[SkillFlagWeapon1Attack] {
			var handBreakdown *Breakdown
			if breakdown != nil {
				breakdown.mainHand = newHandBreakdown(skillModList, outputTable[OutTableMainHand])
				handBreakdown = breakdown.mainHand
			}
			activeSkill.Weapon1Cfg.SkillStats = outputTable[OutTableMainHand]
			//pob:src/Modules/CalcOffence.lua:2009
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
				Breakdown: handBreakdown,
			})
		}

		if skillFlags[SkillFlagWeapon2Attack] {
			var handBreakdown *Breakdown
			if breakdown != nil {
				breakdown.offHand = newHandBreakdown(skillModList, outputTable[OutTableOffHand])
				handBreakdown = breakdown.offHand
			}
			activeSkill.Weapon2Cfg.SkillStats = outputTable[OutTableOffHand]
			//pob:src/Modules/CalcOffence.lua:2026
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
			//pob:block:src/Modules/CalcOffence.lua:2037-2044
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
				Breakdown: handBreakdown,
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
				mainChance := float64(outputTable[OutTableMainHand][idk[0]] * outputTable[OutTableMainHand]["HitChance"])
				offChance := float64(outputTable[OutTableOffHand][idk[0]] * outputTable[OutTableOffHand]["HitChance"])

				if skillData.DoubleHitsWhenDualWielding {
					mainChance = mainChance / 10000
					offChance = offChance / 10000
					actor.Output[stat] = float64(outputTable[OutTableMainHand][stat]*mainChance) + float64(outputTable[OutTableOffHand][stat]*offChance)
					if breakdown != nil {
						breakdown.AddLine(stat, "Contribution from Main Hand:")
						breakdown.AddLine(stat, luaSprintf("%.1f", actor.OutputTable[OutTableMainHand][stat]))
						breakdown.AddLine(stat, luaSprintf("x %.3f ^8(chance of main hand)", mainChance))
						breakdown.AddLine(stat, luaSprintf("= %.1f", actor.OutputTable[OutTableMainHand][stat]*mainChance))
						breakdown.AddLine(stat, "Contribution from Off Hand:")
						breakdown.AddLine(stat, luaSprintf("%.1f", actor.OutputTable[OutTableOffHand][stat]))
						breakdown.AddLine(stat, luaSprintf("x %.3f ^8(chance of  off hand)", offChance))
						breakdown.AddLine(stat, luaSprintf("= %.1f", actor.OutputTable[OutTableOffHand][stat]*offChance))
						breakdown.AddLine(stat, "Total:")
						breakdown.AddLine(stat, luaSprintf("%.1f + %.1f", actor.OutputTable[OutTableMainHand][stat]*mainChance, actor.OutputTable[OutTableOffHand][stat]*offChance))
						breakdown.AddLine(stat, luaSprintf("= %.1f", actor.Output[stat]))
					}
					return
				}

				mainPortion := mainChance / (mainChance + offChance)
				offPortion := offChance / (mainChance + offChance)
				actor.Output[stat] = float64(outputTable[OutTableMainHand][stat]*mainPortion) + float64(outputTable[OutTableOffHand][stat]*offPortion)
				if breakdown != nil {
					breakdown.AddLine(stat, "Contribution from Main Hand:")
					breakdown.AddLine(stat, luaSprintf("%.1f", actor.OutputTable[OutTableMainHand][stat]))
					breakdown.AddLine(stat, luaSprintf("x %.3f ^8(portion of instances created by main hand)", mainPortion))
					breakdown.AddLine(stat, luaSprintf("= %.1f", actor.OutputTable[OutTableMainHand][stat]*mainPortion))
					breakdown.AddLine(stat, "Contribution from Off Hand:")
					breakdown.AddLine(stat, luaSprintf("%.1f", actor.OutputTable[OutTableOffHand][stat]))
					breakdown.AddLine(stat, luaSprintf("x %.3f ^8(portion of instances created by off hand)", offPortion))
					breakdown.AddLine(stat, luaSprintf("= %.1f", actor.OutputTable[OutTableOffHand][stat]*offPortion))
					breakdown.AddLine(stat, "Total:")
					breakdown.AddLine(stat, luaSprintf("%.1f + %.1f", actor.OutputTable[OutTableMainHand][stat]*mainPortion, actor.OutputTable[OutTableOffHand][stat]*offPortion))
					breakdown.AddLine(stat, luaSprintf("= %.1f", actor.Output[stat]))
				}
			} else {
				//pob:src/Modules/CalcOffence.lua:2126
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

				actor.Output[stat] = float64(maxInstance*maxInstanceStacks) + float64(minInstance*(1-maxInstanceStacks))
				if breakdown != nil {
					breakdown.AddLine(stat, "")
					breakdown.AddLine(stat, luaSprintf("%.2f%% of ailment stacks use maximum damage", maxInstanceStacks*100))
					breakdown.AddLine(stat, luaSprintf("Max Damage comes from %s", utils.Ternary(actor.OutputTable[OutTableMainHand][stat] >= actor.OutputTable[OutTableOffHand][stat], "Main Hand", "Off Hand")))
					breakdown.AddLine(stat, luaSprintf("= %.1f", maxInstance*maxInstanceStacks))
					if maxInstanceStacks < 1 {
						breakdown.AddLine(stat, luaSprintf("%.2f%% of ailment stacks use non-maximum damage", (1-maxInstanceStacks)*100))
						breakdown.AddLine(stat, luaSprintf("= %.1f", minInstance*(1-maxInstanceStacks)))
					}
					breakdown.AddLine(stat, "")
					breakdown.AddLine(stat, "Total:")
					if maxInstanceStacks < 1 {
						breakdown.AddLine(stat, luaSprintf("%.1f + %.1f", maxInstance*maxInstanceStacks, minInstance*(1-maxInstanceStacks)))
					}
					breakdown.AddLine(stat, luaSprintf("= %.1f", actor.Output[stat]))
				}
			} else {
				//pob:src/Modules/CalcOffence.lua:2157
				mainOrOff(stat)
				if breakdown != nil {
					breakdown.AddLine(stat, luaSprintf("All ailment stacks comes from %s", utils.Ternary(actor.OutputTable[OutTableMainHand][stat] != 0, "Main Hand", "Off Hand")))
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
		//pob:block:src/Modules/CalcOffence.lua:2181
		breakdown := pass.Breakdown
		//pob:block:src/Modules/CalcOffence.lua:2183-2187
		if pass.Output["Repeats"] > 1 {
			pass.Output["AverageBurstHits"] = pass.Output["Repeats"]
		}

		// Calculate hit chance
		//pob:block:src/Modules/CalcOffence.lua:2189-2198
		accuracyMore := skillModList.More(pass.Config, "Accuracy")
		base := skillModList.Sum(mod.TypeBase, pass.Config, "Accuracy")
		inc := skillModList.Sum(mod.TypeIncrease, pass.Config, "Accuracy")
		baseVsEnemy := skillModList.Sum(mod.TypeBase, pass.Config, "Accuracy", "AccuracyVsEnemy")
		incVsEnemy := skillModList.Sum(mod.TypeIncrease, pass.Config, "Accuracy", "AccuracyVsEnemy")
		moreVsEnemy := skillModList.More(pass.Config, "Accuracy", "AccuracyVsEnemy")

		pass.Output["Accuracy"] = math.Max(0, math.Floor(base*(1+inc/100)*accuracyMore))
		accuracyVsEnemy := math.Max(0, math.Floor(baseVsEnemy*(1+incVsEnemy/100)*moreVsEnemy))
		//pob:block:src/Modules/CalcOffence.lua:2199-2217
		if breakdown != nil {
			breakdown.Data["Accuracy"] = &Entry{}
			breakdown.MultiChain("Accuracy", BMultiChain{
				Base:  &BMultiChainItem{"%.6g ^8(base)", base},
				Total: luaSprintf("= %.6g", pass.Output["Accuracy"]),
				Items: []BMultiChainItem{
					{"%.2f ^8(increased/reduced)", 1 + inc/100},
					{"%.2f ^8(more/less)", accuracyMore},
				},
			})
			if pass.Output["Accuracy"] != accuracyVsEnemy {
				breakdown.AddLine("Accuracy", "")
				breakdown.MultiChain("Accuracy", BMultiChain{
					Label: "Effective Accuracy vs Enemy",
					Base:  &BMultiChainItem{"%.6g ^8(base)", baseVsEnemy},
					Total: luaSprintf("= %.6g", accuracyVsEnemy),
					Items: []BMultiChainItem{
						{"%.2f ^8(increased/reduced)", 1 + incVsEnemy/100},
						{"%.2f ^8(more/less)", moreVsEnemy},
					},
				})
			}
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
				breakdown.Data["Accuracy"] = &Entry{Lines: []string{
					luaSprintf("Using Main Hand Accuracy due to Mastery: %.14g", pass.Output["Accuracy"]),
				}}
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
			//pob:block:src/Modules/CalcOffence.lua:2235-2241
			if breakdown != nil {
				breakdown.Data["AccuracyHitChance"] = &Entry{Lines: []string{
					"Enemy level: " + fmt.Sprint(env.EnemyLevel) + utils.Ternary(env.Build.GetNumberOption("enemyLevel") != 0, " ^8(overridden from the Configuration tab", " ^8(can be overridden in the Configuration tab)"),
					luaSprintf("Enemy evasion: %.14g", enemyEvasion),
					luaSprintf("Approximate hit chance: %.14g%%", pass.Output["AccuracyHitChance"]),
				}}
			}
		}

		// enemy block chance
		//pob:block:src/Modules/CalcOffence.lua:1666-1668
		pass.Output["enemyBlockChance"] = min(max(enemyDB.Sum(mod.TypeBase, pass.Config, "BlockChance"), 0), 100)
		pass.Output["HitChance"] = pass.Output["AccuracyHitChance"] * (1 - pass.Output["enemyBlockChance"]/100)
		//pob:block:src/Modules/CalcOffence.lua:2250-2252
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
		//pob:src/Modules/CalcOffence.lua:2268-2270
		condName := strings.ReplaceAll(pass.Label, " ", "") + "AccRatingHigherThanMaxLife"
		moddb.SetCondition(skillModList.ModStore, condName, pass.Output["Accuracy"] > env.Player.Output["Life"])

		// Calculate attack/cast speed
		//pob:src/Modules/CalcOffence.lua:2272-2275
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
								luaSprintf("%.2f ^8(trigger rate)", skillData.triggerRate),
								luaSprintf("* %.2f ^8(multiplier from Unleash)", cachedSourceSkill.ActiveSkill.skillData.dpsMultiplier or 1),
								luaSprintf("= %.2f", skillData.unleashTriggerRate),
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
				//pob:src/Modules/CalcOffence.lua:2299-2300
				if skillData.AttackSpeedMultiplier != 0 && pass.Source != nil && pass.Source.AttackRate != 0 {
					pass.Source.AttackRate *= 1 + skillData.AttackSpeedMultiplier/100
				}
				//pob:src/Modules/CalcOffence.lua:2302-2304
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

			//pob:block:src/Modules/CalcOffence.lua:2317-2338
			if skillModList.Flag(nil, "HasTrauma") {
				effectiveAttackRateCap := float64(data.ServerTickRate * pass.Output["Repeats"])
				duration := skillModList.Sum(mod.TypeBase, pass.Config, "TraumaDuration") *
					CalcMod(skillModList, skillCfg, "Duration")
				traumaPerAttack := 1 + min(skillModList.Sum(mod.TypeBase, pass.Config, "ExtraTrauma"), 100)/100
				incAttackSpeedPerTrauma := skillModList.Sum(mod.TypeIncrease, skillCfg, "SpeedPerTrauma")
				configTrauma := skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:TraumaStacks")
				traumaInc := skillModList.Sum(mod.TypeIncrease, pass.Config, "Speed") - float64(incAttackSpeedPerTrauma*configTrauma)
				attackSpeedBeforeInc := 1 / baseTime * actor.Output["ActionSpeedMod"] * more
				incAttackSpeedPerTraumaCap := (effectiveAttackRateCap - float64(attackSpeedBeforeInc*(1+traumaInc/100))) / attackSpeedBeforeInc * 100
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

			//pob:block:src/Modules/CalcOffence.lua:2397-2399
			if skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:TraumaStacks") == 0 {
				skillModList.AddMod(mod.NewFloat("Multiplier:TraumaStacks", mod.TypeBase,
					skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:SustainableTraumaStacks")).
					Source("Maximum Sustainable Trauma Stacks"))
			}

			inc := skillModList.Sum(mod.TypeIncrease, pass.Config, "Speed")

			if skillFlags[SkillFlagWarcry] {
				pass.Output["Speed"] = 1 / pass.Output["WarcryCastTime"]
			} else {
				//pob:block:src/Modules/CalcOffence.lua:2405
				pass.Output["Speed"] = 1 / (baseTime/utils.RoundTo((1+inc/100)*more, 2) + skillModList.Sum(mod.TypeBase, pass.Config, "TotalAttackTime") + skillModList.Sum(mod.TypeBase, pass.Config, "TotalCastTime"))
			}
			pass.Output["CastRate"] = pass.Output["Speed"]

			if skillFlags[SkillFlagSelfCast] {
				// Self-cast skill; apply action speed
				pass.Output["Speed"] = pass.Output["Speed"] * actor.Output["ActionSpeedMod"]
				pass.Output["CastRate"] = pass.Output["Speed"]
			}

			//pob:block:src/Modules/CalcOffence.lua:1878-1884
			if skillFlags[SkillFlagTotem] {
				// Totem skill. Apply action speed
				totemActionSpeed := 1 + (modDB.Sum(mod.TypeIncrease, nil, "TotemActionSpeed") / 100)
				pass.Output["TotemActionSpeed"] = totemActionSpeed
				pass.Output["Speed"] = pass.Output["Speed"] * totemActionSpeed
				pass.Output["CastRate"] = pass.Output["Speed"]
				//pob:src/Modules/CalcOffence.lua:2420-2423
				if skillData.TotemFireOnce {
					pass.Output["HitTime"] = 1/pass.Output["Speed"] + actor.Output["TotemPlacementTime"]
					pass.Output["HitSpeed"] = 1 / pass.Output["HitTime"]
				}
			}

			//pob:src/Modules/CalcOffence.lua:2425-2428
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

			//pob:block:src/Modules/CalcOffence.lua:2442-2469
			if breakdown != nil {
				actionSpeed := float64(1)
				if totemActionSpeed, ok := pass.Output["TotemActionSpeed"]; skillFlags[SkillFlagTotem] && ok {
					actionSpeed = totemActionSpeed
				} else if skillFlags[SkillFlagSelfCast] {
					actionSpeed = actor.Output["ActionSpeedMod"]
				}
				breakdown.Data["Speed"] = &Entry{}
				breakdown.MultiChain("Speed", BMultiChain{
					Base:  &BMultiChainItem{"%.2f ^8(base)", 1 / baseTime},
					Total: luaSprintf("= %.2f ^8casts per second", pass.Output["CastRate"]),
					Items: []BMultiChainItem{
						{"%.2f ^8(increased/reduced)", 1 + inc/100},
						{"%.2f ^8(more/less)", more},
						{"%.2f ^8(action speed modifier)", actionSpeed},
					},
				})
				if utils.Has(pass.Output, "Cooldown") && (1/pass.Output["Cooldown"]) < pass.Output["CastRate"] {
					breakdown.AddLine("Speed", "\n")
					breakdown.AddLine("Speed", luaSprintf("1 / %.2f ^8(skill cooldown)", pass.Output["Cooldown"]))
					if pass.Output["Repeats"] > 1 {
						breakdown.AddLine("Speed", luaSprintf("x %d ^8(repeat count)", luaInt(pass.Output["Repeats"])))
					}
					breakdown.AddLine("Speed", luaSprintf("= %.2f ^8(casts per second)", pass.Output["Repeats"]/pass.Output["Cooldown"]))
					breakdown.AddLine("Speed", "\n")
					breakdown.AddLine("Speed", luaSprintf("= %.2f ^8(lower of cast rates)", pass.Output["Speed"]))
				}
			}
			if breakdown != nil && calclib.Mod(skillModList, skillCfg, "SkillAttackTime") > 0 {
				breakdown.Data["Time"] = &Entry{}
				breakdown.MultiChain("Time", BMultiChain{
					Base:  &BMultiChainItem{"%.2f ^8(base)", 1 / (pass.Output["Speed"] * calclib.Mod(skillModList, skillCfg, "SkillAttackTime"))},
					Total: luaSprintf("= %.2f ^8seconds per attack", pass.Output["Time"]),
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

	//pob:block:src/Modules/CalcOffence.lua:2504
	if skillModList.Flag(nil, "HasTrauma") {
		actor.Output["SustainableTrauma"] = skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:SustainableTraumaStacks")
	}

	//pob:block:src/Modules/CalcOffence.lua:2505-2507
	modDB.Multipliers["BuffOnSelf"] += skillModList.Sum(mod.TypeBase, nil, "Multiplier:TraumaStacks")
	modDB.Multipliers["BuffOnSelf"] += skillModList.Sum(mod.TypeBase, nil, "Multiplier:VoltaxicWaitingStages")

	if utils.HasTrue(skillFlags, SkillFlagAttack) {
		// Combine hit chance and attack speed
		//pob:block:src/Modules/CalcOffence.lua:1953-1957
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
			//pob:block:src/Modules/CalcOffence.lua:2527-2540
			if breakdown != nil {
				breakdown.Data["Speed"] = &Entry{Lines: []string{
					"Use Offhand Weapon Attack Speed:",
					luaSprintf("= %.2f", actor.Output["Speed"]),
				}}
			}
		} else if utils.HasTrue(skillFlags, SkillFlagBothWeaponAttack) {
			if breakdown != nil {
				breakdown.Data["Speed"] = &Entry{Lines: []string{
					"Both weapons:",
					luaSprintf("2 / (1 / %.2f + 1 / %.2f)", actor.OutputTable[OutTableMainHand]["Speed"], actor.OutputTable[OutTableOffHand]["Speed"]),
					luaSprintf("= %.2f", actor.Output["Speed"]),
				}}
			}
		}
		//pob:block:src/Modules/CalcOffence.lua:2542-2555
		if breakdown != nil {
			if skillFlags[SkillFlagBothWeaponAttack] {
				breakdown.Data["HitChance"] = &Entry{Lines: []string{
					"Both weapons:",
					luaSprintf("%.2f%% ^8(main hand)", actor.OutputTable[OutTableMainHand]["HitChance"]),
					luaSprintf("%.2f%% ^8(off hand)", actor.OutputTable[OutTableOffHand]["HitChance"]),
					luaSprintf("= %.2f%% ^8(average)", actor.Output["HitChance"]),
				}}
			} else if handBreakdown := utils.Ternary(skillFlags[SkillFlagWeapon1Attack], breakdown.mainHand, breakdown.offHand); handBreakdown != nil {
				setBreakdownEntry(breakdown, "HitChance", handBreakdown.Data["HitChance"], handBreakdown.Data["AccuracyHitChance"])
				setBreakdownEntry(breakdown, "Speed", breakdown.Data["Speed"], handBreakdown.Data["Speed"])
			}
		}
		//pob:block:src/Modules/CalcOffence.lua:2556-2568
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
	//pob:block:src/Modules/CalcMirages.lua:393-396
	if generalsCryCooldown != 0 {
		skillTime := actor.Output["Time"]
		if hitTime, ok := actor.Output["HitTime"]; ok && activeSkill.SkillTypes[data.SkillTypeChannel] {
			skillTime = hitTime
		}
		cooldown := max(generalsCryCooldown, skillTime)
		skillModList.AddMod(mod.NewFloat("DPS", mod.TypeMore, (1/cooldown-1)*100).Source("General's Cry Cooldown"))
	}

	//pob:block:src/Modules/CalcOffence.lua:2489-2490
	dpsMultiplier := skillData.DpsMultiplier
	if dpsMultiplier == 0 {
		dpsMultiplier = 1
	}
	skillData.DpsMultiplier = dpsMultiplier *
		(1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "DPS")/100) *
		skillModList.More(skillCfg, "DPS")

	//pob:block:src/Modules/CalcOffence.lua:2491-2494
	if activeSkill.SkillTypes[data.SkillTypeBrand] && !skillData.CountsAttachedBrandsInDamage {
		skillData.DpsMultiplier *= actor.Output["AttachedBrandCount"]
		skillDPSMultiplier := actor.Output["SkillDPSMultiplier"]
		if skillDPSMultiplier == 0 {
			skillDPSMultiplier = 1
		}
		actor.Output["SkillDPSMultiplier"] = skillDPSMultiplier * actor.Output["AttachedBrandCount"]
	}

	//pob:block:src/Modules/CalcOffence.lua:2495-2497
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

	//pob:src/Modules/CalcOffence.lua:1725-2030
	for _, pass := range passList {
		globalOutput := actor.Output
		output, cfg := pass.Output, pass.Config
		globalBreakdown, breakdown := breakdown, pass.Breakdown

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
			//pob:block:src/Modules/CalcOffence.lua:2628-2629
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
				//pob:block:src/Modules/CalcOffence.lua:2847-2856
				if globalBreakdown != nil {
					globalBreakdown.Data["ExertedAttackUptimeRatio"] = &Entry{Lines: []string{"Maximum of:"}}
					for _, cryTimeRatio := range []string{"AncestralUpTimeRatio", "InfernalUpTimeRatio", "IntimidatingUpTimeRatio", "RallyingUpTimeRatio", "SeismicUpTimeRatio", "BattlemageUpTimeRatio"} {
						if utils.Has(globalOutput, cryTimeRatio) {
							globalBreakdown.AddLine("ExertedAttackUptimeRatio", luaSprintf("%d%% ^8(%s Cry Uptime)", luaInt(globalOutput[cryTimeRatio]), strings.TrimSuffix(cryTimeRatio, "UpTimeRatio")))
						}
					}
					globalBreakdown.AddLine("ExertedAttackUptimeRatio", luaSprintf("= %d%%", luaInt(globalOutput["ExertedAttackUptimeRatio"])))
				}
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
						luaSprintf("(%.2f ^8(number of Empowered)", count),
						luaSprintf("/ %.2f) ^8(casts per second)", globalOutput["Speed"]),
						luaSprintf("/ (%.2f ^8(pact cooldown)", cooldown),
						luaSprintf("+ %.2f) ^8(pact cast time)", castTime),
						luaSprintf("* %.2f ^8(stored uses)", storedUses),
						luaSprintf("= %.2f%%", uptime),
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
			//pob:block:src/Modules/CalcOffence.lua:2980-2992
			ruthlessEffect := utils.OrS(env.Build.GetStringOption("ruthlessSupportMode"), "AVERAGE")
			// Calculate Ruthless Blow chance/multipliers + Fist of War multipliers
			pass.Output["RuthlessBlowMaxCount"] = skillModList.Sum(mod.TypeBase, pass.Config, "RuthlessBlowMaxCount")
			if pass.Output["RuthlessBlowMaxCount"] > 0 {
				switch ruthlessEffect {
				case "AVERAGE":
					pass.Output["RuthlessBlowChance"] = utils.RoundTo(100/pass.Output["RuthlessBlowMaxCount"], 0)
				case "MAX":
					pass.Output["RuthlessBlowChance"] = 100
					skillData.DpsMultiplier /= utils.OrDefault(pass.Output["RuthlessBlowMaxCount"], 1)
				}
			} else {
				pass.Output["RuthlessBlowChance"] = 0
			}
			pass.Output["RuthlessBlowHitMultiplier"] = 1 + skillModList.Sum(mod.TypeBase, pass.Config, "RuthlessBlowHitMultiplier")/100
			pass.Output["RuthlessBlowAilmentMultiplier"] = 1 + skillModList.Sum(mod.TypeBase, pass.Config, "RuthlessBlowAilmentMultiplier")/100
			pass.Output["RuthlessBlowHitEffect"] = 1 - pass.Output["RuthlessBlowChance"]/100 + float64(pass.Output["RuthlessBlowChance"]/100*pass.Output["RuthlessBlowHitMultiplier"])
			pass.Output["RuthlessBlowAilmentEffect"] = 1 - pass.Output["RuthlessBlowChance"]/100 + float64(pass.Output["RuthlessBlowChance"]/100*pass.Output["RuthlessBlowAilmentMultiplier"])

			actor.Output["FistOfWarCooldown"] = skillModList.Sum(mod.TypeBase, pass.Config, "FistOfWarCooldown")
			// If Fist of War & Active Skill is a Slam Skill & NOT a Vaal Skill & NOT used by mirage or other
			if actor.Output["FistOfWarCooldown"] != 0 && activeSkill.SkillTypes[data.SkillTypeSlam] && !activeSkill.SkillTypes[data.SkillTypeVaal] && !activeSkill.SkillTypes[data.SkillTypeOtherThingUsesSkill] {
				actor.Output["FistOfWarDamageMultiplier"] = skillModList.Sum(mod.TypeBase, nil, "FistOfWarDamageMultiplier") / 100
				actor.Output["FistOfWarUptimeRatio"] = min((1/pass.Output["Speed"])/actor.Output["FistOfWarCooldown"], 1) * 100
				if actor.Breakdown != nil {
					actor.Breakdown.AddLine("FistOfWarUptimeRatio",
						luaSprintf("min( (1 / %.2f) ^8(second per attack)", pass.Output["Speed"]),
						luaSprintf("/ %.2f, 1) ^8(fist of war cooldown)", actor.Output["FistOfWarCooldown"]),
						luaSprintf("= %.2f%%", actor.Output["FistOfWarUptimeRatio"]),
					)
				}
				actor.Output["AvgFistOfWarDamage"] = actor.Output["FistOfWarDamageMultiplier"]
				actor.Output["AvgFistOfWarDamageEffect"] = 1 + float64(actor.Output["FistOfWarDamageMultiplier"]*(actor.Output["FistOfWarUptimeRatio"]/100))
				if actor.Breakdown != nil {
					actor.Breakdown.AddLine("AvgFistOfWarDamageEffect",
						luaSprintf("1 + (%.2f ^8(fist of war damage multiplier)", actor.Output["FistOfWarDamageMultiplier"]),
						luaSprintf("x %.2f) ^8(fist of war uptime ratio)", actor.Output["FistOfWarUptimeRatio"]/100),
						luaSprintf("= %.2f", actor.Output["AvgFistOfWarDamageEffect"]),
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
		//pob:block:src/Modules/CalcOffence.lua:3035-3039
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

			//pob:block:src/Modules/CalcOffence.lua:3078-3079
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

			//pob:block:src/Modules/CalcOffence.lua:3085-3091
			if skillModList.Flag(pass.Config, "BaseCritFromMainHand") && actor.WeaponData1 != nil {
				baseCrit = actor.WeaponData1.CritChance
			} else if skillModList.Flag(pass.Config, "AttackCritIsEqualToParentMainHand") &&
				actor.Parent != nil && actor.Parent.WeaponData1 != nil {
				baseCrit = actor.Parent.WeaponData1.CritChance
			}

			//pob:block:src/Modules/CalcOffence.lua:3093-3096
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

					//pob:block:src/Modules/CalcOffence.lua:3102-3104
					if env.ModeEffective {
						base += enemyDB.Sum(mod.TypeBase, nil, "SelfCritChance")
						inc += enemyDB.Sum(mod.TypeIncrease, nil, "SelfCritChance")
						more *= enemyDB.More(nil, "SelfCritChance")
					}
				}

				//pob:block:src/Modules/CalcOffence.lua:3106-3107
				// The game uses an integer permyriad chance for each critical strike roll.
				pass.Output["CritChance"] = utils.RoundTo((baseCrit+base)*(1+inc/100)*more*100, 0) / 100

				preCapCritChance := pass.Output["CritChance"]
				//pob:block:src/Modules/CalcOffence.lua:3109
				if critCap := skillModList.Override(nil, "CritChanceCap"); critCap != nil {
					pass.Output["CritChance"] = math.Min(pass.Output["CritChance"], critCap.Float())
				} else {
					pass.Output["CritChance"] = math.Min(pass.Output["CritChance"], skillModList.Sum(mod.TypeBase, pass.Config, "CritChanceCap"))
				}

				if baseCrit+base > 0 {
					pass.Output["CritChance"] = math.Max(pass.Output["CritChance"], 0)
				}

				pass.Output["PreEffectiveCritChance"] = pass.Output["CritChance"]
				preLuckyCritChance := pass.Output["CritChance"]

				//pob:block:src/Modules/CalcOffence.lua:3115-3130
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
						pass.Output["CritChance"] = (float64(3*p*p) - float64(2*p*p*p)) * 100
					} else {
						pass.Output["CritChance"] = (1 - math.Pow(1-p, critRolls+1)) * 100
					}
				}

				//pob:block:src/Modules/CalcOffence.lua:3131-3135
				pass.Output["PreBifurcateCritChance"] = pass.Output["CritChance"]
				if env.ModeEffective && skillModList.Flag(pass.Config, "BifurcateCrit") {
					pass.Output["CritChance"] = (1 - math.Pow(1-pass.Output["CritChance"]/100, 2)) * 100
				}

				preHitCheckCritChance := pass.Output["CritChance"]
				preSkillUseCritChance := pass.Output["CritChance"]
				if env.ModeEffective {
					//pob:block:src/Modules/CalcOffence.lua:3139-3144
					if skillModList.Flag(skillCfg, "Every3UseCrit") {
						pass.Output["CritChance"] = (2*pass.Output["CritChance"] + 100) / 3
					}
					if skillModList.Flag(skillCfg, "Every5UseCrit") {
						pass.Output["CritChance"] = (float64(4*pass.Output["CritChance"]) + 100) / 5
					}
					preHitCheckCritChance = pass.Output["CritChance"]
					//pob:block:src/Modules/CalcOffence.lua:2386-2388
					pass.Output["CritChance"] = pass.Output["CritChance"] * pass.Output["AccuracyHitChance"] / 100
				}

				//pob:block:src/Modules/CalcOffence.lua:3148-3200
				if breakdown != nil && pass.Output["CritChance"] != baseCrit {
					breakdown.Data["CritChance"] = &Entry{}
					baseCritFromMainHandStr := ""
					if skillModList.Flag(pass.Config, "BaseCritFromMainHand") {
						baseCritFromMainHandStr = " from main weapon"
					} else if skillModList.Flag(pass.Config, "AttackCritIsEqualToParentMainHand") {
						baseCritFromMainHandStr = " from parent main weapon"
					}
					if base != 0 {
						breakdown.AddLine("CritChance", luaSprintf("(%.6g + %.6g) ^8(base%s)", baseCrit, base, baseCritFromMainHandStr))
					} else {
						breakdown.AddLine("CritChance", luaSprintf("%.6g ^8(base%s)", baseCrit+base, baseCritFromMainHandStr))
					}
					if inc != 0 {
						breakdown.AddLine("CritChance", luaSprintf("x %.2f", 1+inc/100)+" ^8(increased/reduced)")
					}
					if more != 1 {
						breakdown.AddLine("CritChance", luaSprintf("x %.2f", more)+" ^8(more/less)")
					}
					breakdown.AddLine("CritChance", luaSprintf("= %.2f%% ^8(crit chance)", pass.Output["PreEffectiveCritChance"]))
					if preCapCritChance > 100 {
						overCap := preCapCritChance - 100
						breakdown.AddLine("CritChance", luaSprintf("Crit is overcapped by %.2f%% (%d%% increased Critical Strike Chance)", overCap, luaInt(overCap/more/(baseCrit+base)*100)))
					}
					//pob:src/Modules/CalcOffence.lua:3167
					breakdown.Data["PreEffectiveCritChance"] = &Entry{Lines: slices.Clone(breakdown.Data["CritChance"].Lines)}
					if env.ModeEffective {
						if critRolls != 0 {
							if skillModList.Flag(skillCfg, "Unexciting") {
								breakdown.AddLine("CritChance", "Crit Chance is Unexciting:")
								breakdown.AddLine("CritChance", luaSprintf("(3 x %.4f^ 2) - (2 x %.4f^ 3)", preLuckyCritChance/100, preLuckyCritChance/100))
							} else {
								breakdown.AddLine("CritChance", "Crit Chance is Lucky:")
								breakdown.AddLine("CritChance", luaSprintf("1 - (1 - %.4f)^ %d", preLuckyCritChance/100, luaInt(critRolls)+1))
							}
							breakdown.AddLine("CritChance", luaSprintf("= %.2f%%", pass.Output["PreBifurcateCritChance"]))
						}
						if skillModList.Flag(pass.Config, "BifurcateCrit") {
							breakdown.AddLine("CritChance", "Critical Strike Bifurcates:")
							breakdown.AddLine("CritChance", luaSprintf("1 - (1 - %.4f) x (1 - %.4f)", pass.Output["PreBifurcateCritChance"]/100, pass.Output["PreBifurcateCritChance"]/100))
							breakdown.AddLine("CritChance", luaSprintf("= %.2f%%", preSkillUseCritChance))
						}
					}
					if env.ModeEffective {
						every3 := skillModList.Flag(skillCfg, "Every3UseCrit")
						every5 := skillModList.Flag(skillCfg, "Every5UseCrit")
						if every3 {
							breakdown.AddLine("CritChance", luaSprintf("+ %.2f%% ^8(crit every 3rd use)", (2*preSkillUseCritChance+100)/3-preSkillUseCritChance))
						}
						if every5 {
							breakdown.AddLine("CritChance", luaSprintf("+ %.2f%% ^8(crit every 5th use)", (float64(4*preSkillUseCritChance)+100)/5-preSkillUseCritChance))
						}
						if every3 || every5 {
							breakdown.AddLine("CritChance", luaSprintf("= %.2f%%", preHitCheckCritChance))
						}
					}
					if env.ModeEffective && pass.Output["AccuracyHitChance"] < 100 {
						breakdown.AddLine("CritChance", "Crit confirmation roll:")
						breakdown.AddLine("CritChance", luaSprintf("%.2f%%", preHitCheckCritChance))
						breakdown.AddLine("CritChance", luaSprintf("x %.2f ^8(chance to hit)", pass.Output["AccuracyHitChance"]/100))
						breakdown.AddLine("CritChance", luaSprintf("= %.2f%%", pass.Output["CritChance"]))
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
							luaSprintf("%.2f%% ^8(additional extra damage)", (enemyDB.Sum(mod.TypeBase, nil, "SelfCritMultiplier")+skillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplier"))/100),
							luaSprintf("x %.2f ^8(increased/reduced extra crit damage taken by enemy)", enemyInc),
							luaSprintf("= %.2f%% ^8(extra crit damage)", extraDamage*100),
						)
					}
				}

				//pob:block:src/Modules/CalcOffence.lua:3225-3259
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
					extraDamage += float64(conditionalBifurcateChance * extraDamage)
					// mod doesn't affect output and is purely descriptive
					//pob:block:src/Modules/CalcOffence.lua:3257-3258
					skillModList.AddMod(mod.NewFloat("CritMultiplier", mod.TypeMore, math.Floor(float64(conditionalBifurcateChance*100*100)+0.0001)/100).
						Source("Bifurcated Crit Damage Bonus").Flag(mod.MFlagHit))
				}

				pass.Output["CritMultiplier"] = 1 + math.Max(0, extraDamage)
			}

			critChancePercentage := pass.Output["CritChance"] / 100
			pass.Output["CritEffect"] = 1 - critChancePercentage + float64(critChancePercentage*pass.Output["CritMultiplier"])
			pass.Output["BonusCritDotMultiplier"] = (skillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplier") - 50) * skillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplierAppliesToDegen") / 10000

			//pob:block:src/Modules/CalcOffence.lua:3265-3271
			if breakdown != nil && pass.Output["CritEffect"] != 1 {
				breakdown.Data["CritEffect"] = &Entry{Lines: []string{
					luaSprintf("(1 - %.4f) ^8(portion of damage from non-crits)", critChancePercentage),
					luaSprintf("+ [ (%.4f x %.6g) ^8(portion of damage from crits)", critChancePercentage, pass.Output["CritMultiplier"]),
					luaSprintf("= %.3f", pass.Output["CritEffect"]),
				}}
			}
		}

		//pob:src/Modules/CalcOffence.lua:3273-3275
		if pass.Output["CritChance"] != 0 {
			moddb.SetCondition(skillModList.ModStore, "CritInPast8Sec", true)
		}

		//pob:block:src/Modules/CalcOffence.lua:3277-3298
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
		//pob:src/Modules/CalcOffence.lua:3300
		selectedSpeed, gotSpeed := actor.Output["HitSpeed"]
		if !gotSpeed || selectedSpeed == 0 {
			selectedSpeed = actor.Output["Speed"]
		}
		DpsMultiplier := utils.OrDefault(skillData.DpsMultiplier, 1)
		hitRate := pass.Output["HitChance"] / 100 * selectedSpeed * DpsMultiplier

		criticalCull := skillModList.Max(pass.Config, "CriticalCullPercent")
		if criticalCull > 0 {
			//pob:src/Modules/CalcOffence.lua:3305
			criticalCull = math.Min(criticalCull,
				criticalCull*(1-math.Pow(1-pass.Output["CritChance"]/100, hitRate)))
		}
		regularCull := skillModList.Max(pass.Config, "CullPercent")
		maxCullPercent := max(criticalCull, regularCull)
		actor.Output["CullPercent"] = maxCullPercent
		actor.Output["CullMultiplier"] = 100 / (100 - actor.Output["CullPercent"])

		// Calculate reservation DPS
		//pob:block:src/Modules/CalcOffence.lua:3313
		actor.Output["ReservationDpsMultiplier"] = 100 / (100 - enemyDB.Sum(mod.TypeBase, nil, "LifeReservationPercent"))
		runSkillFunc("postCritFunc")

		//pob:block:src/Modules/CalcOffence.lua:3317-3344
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
			dtNames := damageTypeNameSets[damageType]
			damageTypeMin := dtNames.min
			damageTypeMax := dtNames.max

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

			addedMin := skillModList.Sum(mod.TypeBase, pass.Config, damageTypeMin) + enemyDB.Sum(mod.TypeBase, pass.Config, dtNames.selfMin)
			addedMax := skillModList.Sum(mod.TypeBase, pass.Config, damageTypeMax) + enemyDB.Sum(mod.TypeBase, pass.Config, dtNames.selfMax)
			addedMult := CalcMod(skillModList, pass.Config, dtNames.added, "AddedDamage")

			baseMin := float64((utils.GetOr(pass.Source, damageTypeMin, utils.Interface(float64(0))).(float64)+
				utils.GetOr(pass.Source, dtNames.bonusMin, utils.Interface(float64(0))).(float64))*
				baseMultiplier) + float64(addedMin*damageEffectiveness*addedMult)

			baseMax := float64((utils.GetOr(pass.Source, damageTypeMax, utils.Interface(float64(0))).(float64)+
				utils.GetOr(pass.Source, dtNames.bonusMax, utils.Interface(float64(0))).(float64))*
				baseMultiplier) + float64(addedMax*damageEffectiveness*addedMult)

			pass.Output[dtNames.minBase] = baseMin
			pass.Output[dtNames.maxBase] = baseMax

			//pob:block:src/Modules/CalcOffence.lua:3359-3360
			if breakdown != nil {
				breakdown.Data[string(damageType)] = &Entry{}
				if baseMin != 0 && baseMax != 0 {
					breakdown.AddLine(string(damageType), "Base damage:")
					plus := ""
					sourceMin := utils.GetOr(pass.Source, damageTypeMin, float64(0))
					sourceMax := utils.GetOr(pass.Source, damageTypeMax, float64(0))
					if sourceMin != 0 || sourceMax != 0 {
						breakdown.AddLine(string(damageType), luaSprintf("%d to %d ^8(base damage from %s)", luaInt(sourceMin), luaInt(sourceMax), utils.Ternary(pass.Source.Type != "", "weapon", "skill")))
						if baseMultiplier != 1 {
							breakdown.AddLine(string(damageType), luaSprintf("x %.2f ^8(base damage multiplier)", baseMultiplier))
						}
						plus = "+ "
					}
					if addedMin != 0 || addedMax != 0 {
						breakdown.AddLine(string(damageType), luaSprintf("%s%d to %d ^8(added damage)", plus, luaInt(addedMin), luaInt(addedMax)))
						if damageEffectiveness != 1 {
							breakdown.AddLine(string(damageType), luaSprintf("x %.2f ^8(damage effectiveness)", damageEffectiveness))
						}
						if addedMult != 1 {
							breakdown.AddLine(string(damageType), luaSprintf("x %.2f ^8(added damage multiplier)", addedMult))
						}
					}
					breakdown.AddLine(string(damageType), luaSprintf("= %.1f to %.1f", baseMin, baseMax))
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
						breakdown.AddLine(string(damageType), luaSprintf("%d to %d ^8(total damage)", luaInt(damageTypeHitMin), luaInt(damageTypeHitMax)))
						if convMult != 1 {
							breakdown.AddLine(string(damageType), luaSprintf("x %.6g ^8(%.6g%% converted to other damage types)", convMult, (1-convMult)*100))
						}
						if pass.Output["DoubleDamageEffect"] != 0 {
							if pass.Output["TripleDamageEffect"] != 0 {
								breakdown.AddLine(string(damageType), luaSprintf("x %.2f ^8(1 + %.2f + %.2f multiplier from %.1f%% chance to deal double damage and %d%% chance to deal triple damage)", 1+pass.Output["DoubleDamageEffect"]+pass.Output["TripleDamageEffect"], pass.Output["DoubleDamageEffect"], pass.Output["TripleDamageEffect"], pass.Output["DoubleDamageChance"], luaInt(pass.Output["TripleDamageChance"])))
							} else {
								breakdown.AddLine(string(damageType), luaSprintf("x %.2f ^8(multiplier from %d%% chance to deal double damage)", 1+pass.Output["DoubleDamageEffect"], luaInt(pass.Output["DoubleDamageChance"])))
							}
						} else if pass.Output["TripleDamageEffect"] != 0 {
							breakdown.AddLine(string(damageType), luaSprintf("x %.2f ^8(multiplier from %d%% chance to deal triple damage)", 1+pass.Output["TripleDamageEffect"], luaInt(pass.Output["TripleDamageChance"])))
						}
						if pass.Output["RuthlessBlowHitEffect"] != 1 {
							breakdown.AddLine(string(damageType), luaSprintf("x %.2f ^8(ruthless blow effect modifier)", pass.Output["RuthlessBlowHitEffect"]))
						}
						if pass.Output["FistOfWarDamageEffect"] != 1 {
							breakdown.AddLine(string(damageType), luaSprintf("x %.2f ^8(fist of war effect modifier)", pass.Output["FistOfWarDamageEffect"]))
						}
						if actor.Output["OffensiveWarcryEffect"] != 1 && !activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
							breakdown.AddLine(string(damageType), luaSprintf("x %.2f ^8(aggregated warcry exerted effect modifier)", actor.Output["OffensiveWarcryEffect"]))
						}
						if actor.Output["MaxOffensiveWarcryEffect"] != 1 && activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
							breakdown.AddLine(string(damageType), luaSprintf("x %.2f ^8(aggregated max warcry exerted effect modifier)", actor.Output["MaxOffensiveWarcryEffect"]))
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
						damageTypeHitAvg = float64(damageTypeHitAvgNotLucky*(1-damageTypeLuckyChance)) + float64(damageTypeHitAvgLucky*damageTypeLuckyChance)
					} else {
						damageTypeHitAvg = float64(damageTypeHitAvgNotLucky*(1-math.Abs(damageTypeLuckyChance))) + float64(damageTypeHitAvgUnlucky*math.Abs(damageTypeLuckyChance))
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
								pass.Output["impaleStoredHitAvg"] = pass.Output["impaleStoredHitAvg"] + float64(damageTypeHitAvg*(pass.Output["CritChance"]/100))
							} else {
								pass.Output["impaleStoredHitAvg"] = pass.Output["impaleStoredHitAvg"] + float64(damageTypeHitAvg*(1-pass.Output["CritChance"]/100))
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
								effectiveResist = resist - float64(2*invertChance*resist)
							}
							effMult *= 1 - effectiveResist/100
						} else if useRes {
							if invertChance > 0 {
								effectiveResist = (float64((resist-pen)*(1-invertChance)) + float64((-resist-pen)*invertChance)) * useResChance
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
							breakdown.AddLine(string(damageType), luaSprintf("x %.3f ^8(effective DPS modifier)", effMult))
							breakdown.EffMult(string(damageType+"EffMult"), damageType, resist, 0, takenInc, effMult, takenMore, sourceRes, useRes, invertChance, &effectiveResist)
						} else if p == 2 && breakdown != nil && (effMult != 1 || sourceRes != "" || invertChance > 0) {
							breakdown.AddLine(string(damageType), luaSprintf("x %.3f ^8(effective DPS modifier)", effMult))
							breakdown.EffMult(string(damageType+"EffMult"), damageType, resist, pen, takenInc, effMult, takenMore, sourceRes, useRes, invertChance, &effectiveResist)
						}
					}
					if p == 2 && breakdown != nil {
						breakdown.AddLine(string(damageType), luaSprintf("= %d to %d", luaInt(damageTypeHitMin), luaInt(damageTypeHitMax)))
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
								names := damageTypeNameSets[damageType].lifeLeech
								lifeLeech = skillModList.Sum(mod.TypeBase, pass.Config, names...) + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDamageLifeLeech")/100
							}
							if lifeLeech > 0 {
								lifeLeechTotal = lifeLeechTotal + damageTypeHitAvg*lifeLeech/100
							}
						}

						if !noEnergyShieldLeech {
							names := damageTypeNameSets[damageType].energyShieldLeech
							energyShieldLeech := skillModList.Sum(mod.TypeBase, pass.Config, names...) + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDamageEnergyShieldLeech")/100
							if energyShieldLeech > 0 {
								energyShieldLeechTotal = energyShieldLeechTotal + damageTypeHitAvg*energyShieldLeech/100
							}
						}

						if !noManaLeech {
							names := damageTypeNameSets[damageType].manaLeech
							manaLeech := skillModList.Sum(mod.TypeBase, pass.Config, names...) + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDamageManaLeech")/100
							if manaLeech > 0 {
								manaLeechTotal = manaLeechTotal + damageTypeHitAvg*manaLeech/100
							}
						}
					}
				} else {
					//pob:block:src/Modules/CalcOffence.lua:3641-3645
					if breakdown != nil {
						breakdown.Data[string(damageType)] = &Entry{Lines: []string{"You can't deal " + string(damageType) + " damage"}}
					}
				}

				if p == 1 {
					pass.Output[damageTypeNameSets[damageType].critAverage] = damageTypeHitAvg
					totalCritAvg = totalCritAvg + damageTypeHitAvg
					totalCritMin = totalCritMin + damageTypeHitMin
					totalCritMax = totalCritMax + damageTypeHitMax
				} else {
					//pob:src/Modules/CalcOffence.lua:3653-3656
					if env.Mode == OutputModeCalcs {
						pass.Output[string(damageType)+"Min"] = damageTypeHitMin
						pass.Output[string(damageType)+"Max"] = damageTypeHitMax
					}
					pass.Output[damageTypeNameSets[damageType].hitAverage] = damageTypeHitAvg
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

			//pob:block:src/Modules/CalcOffence.lua:3670-3680
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
				ghostReaverLifeLeech += float64(lifeLeechTotal * portion)
			} else {
				pass.Output["LifeLeech"] += float64(lifeLeechTotal * portion)
			}

			pass.Output["EnergyShieldLeech"] += float64(energyShieldLeechTotal * portion)
			pass.Output["ManaLeech"] += float64(manaLeechTotal * portion)
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
					breakdown.AddLine(damageType, luaSprintf("Portion of total damage: %d%%", luaInt(portion)))
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
		//pob:block:src/Modules/CalcOffence.lua:3758-3770
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

		//pob:block:src/Modules/CalcOffence.lua:3772-3781
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

		//pob:block:src/Modules/CalcOffence.lua:3783-3786
		pass.Output["EnemyLifeRegen"] = enemyDB.Sum(mod.TypeIncrease, pass.Config, "LifeRegen")
		pass.Output["EnemyManaRegen"] = enemyDB.Sum(mod.TypeIncrease, pass.Config, "ManaRegen")
		pass.Output["EnemyEnergyShieldRegen"] = enemyDB.Sum(mod.TypeIncrease, pass.Config, "EnergyShieldRegen")

		// Calculate average damage and final DPS
		pass.Output["AverageHit"] = float64(totalHitAvg*(1-pass.Output["CritChance"]/100)) + totalCritAvg*pass.Output["CritChance"]/100
		//pob:block:src/Modules/CalcOffence.lua:3790-3795
		if skillFlags[SkillFlagMonsterExplode] {
			pass.Output["AverageHitToMonsterLifePercentage"] = pass.Output["AverageHit"] / monsterLife * 100
			if skillData.HitChanceIsExplodeChance {
				pass.Output["HitChance"] = pass.Output["ExplodeChance"]
			}
		}
		pass.Output["AverageDamage"] = pass.Output["AverageHit"] * pass.Output["HitChance"] / 100

		//pob:block:src/Modules/CalcOffence.lua:3797
		actor.Output["AverageBurstHits"] = utils.OrDefault(pass.Output["AverageBurstHits"], 1)
		//pob:src/Modules/CalcOffence.lua:3798-3799
		repeatPenalty := float64(1)
		if skillModList.Flag(nil, "HasSeals") && activeSkill.SkillTypes[data.SkillTypeCanRapidFire] &&
			!skillModList.Flag(nil, "NoRepeatBonuses") {
			repeatPenalty = calclib.Mod(skillModList, skillCfg, "SealRepeatPenalty")
		}
		actor.Output["AverageBurstDamage"] = pass.Output["AverageDamage"] +
			float64(pass.Output["AverageDamage"]*(actor.Output["AverageBurstHits"]-1)*repeatPenalty)

		//pob:src/Modules/CalcOffence.lua:3801
		pass.Output["TotalDPS"] = pass.Output["AverageDamage"] * selectedSpeed * utils.OrDefault(skillData.DpsMultiplier, 1) * quantityMultiplier
		//pob:block:src/Modules/CalcOffence.lua:3802-3834
		if breakdown != nil {
			if pass.Output["CritEffect"] != 1 {
				breakdown.Data["AverageHit"] = &Entry{}
				damageRolls := pass.Output["DamageRolls"]
				critRolls := pass.Output["CritDamageRolls"]
				rolls := int(math.Abs(damageRolls)) + 2
				rollsCrit := int(math.Abs(critRolls)) + 2
				if damageRolls > 0 {
					breakdown.AddLine("AverageHit", luaSprintf("(1/%d) x %d + (%d/%d) x %d = %.1f ^8(average from non-crits)", rolls, luaInt(totalHitMin), rolls-1, rolls, luaInt(totalHitMax), totalHitAvg))
				}
				if damageRolls < 0 {
					breakdown.AddLine("AverageHit", luaSprintf("(%d/%d) x %d + (1/%d) x %d = %.1f ^8(average from non-crits)", rolls-1, rolls, luaInt(totalHitMin), rolls, luaInt(totalHitMax), totalHitAvg))
				}
				if damageRolls > 0 || critRolls > 0 || skillModList.Flag(skillCfg, "CritLucky") {
					breakdown.AddLine("AverageHit", luaSprintf("(1/%d) x %d + (%d/%d) x %d = %.1f ^8(average from crits)", rollsCrit, luaInt(totalCritMin), rollsCrit-1, rollsCrit, luaInt(totalCritMax), totalCritAvg))
					breakdown.AddLine("AverageHit", "")
				}
				if damageRolls < 0 && !skillModList.Flag(skillCfg, "CritLucky") {
					breakdown.AddLine("AverageHit", luaSprintf("(%d/%d) x %d + (1/%d) x %d = %.1f ^8(average from crits)", rolls-1, rolls, luaInt(totalCritMin), rolls, luaInt(totalCritMax), totalCritAvg))
					breakdown.AddLine("AverageHit", "")
				}
				breakdown.AddLine("AverageHit", luaSprintf("%.1f x (1 - %.4f) ^8(damage from non-crits)", totalHitAvg, pass.Output["CritChance"]/100))
				breakdown.AddLine("AverageHit", luaSprintf("+ %.1f x %.4f ^8(damage from crits)", totalCritAvg, pass.Output["CritChance"]/100))
				breakdown.AddLine("AverageHit", luaSprintf("= %.1f", pass.Output["AverageHit"]))
			}
			if pass.Output["HitChance"] < 100 {
				breakdown.Data["AverageDamage"] = &Entry{Lines: []string{
					luaSprintf("%s:", pass.Label),
					luaSprintf("%.1f ^8(average hit)", pass.Output["AverageHit"]),
					luaSprintf("x %.2f ^8(chance to hit)", pass.Output["HitChance"]/100),
					luaSprintf("= %.1f", pass.Output["AverageDamage"]),
				}}
			}
		}

		//pob:block:src/Modules/CalcOffence.lua:3835-3854
		if globalBreakdown != nil && actor.Output["AverageBurstDamage"] > 0 {
			key := "AverageBurstDamage"
			globalBreakdown.Data[key] = &Entry{Lines: []string{luaSprintf("%.1f ^8(average hit)", pass.Output["AverageHit"])}}
			if pass.Output["HitChance"] < 100 {
				globalBreakdown.AddLine(key, luaSprintf("x %.2f ^8(chance to hit)", pass.Output["HitChance"]/100))
			}
			if repeatPenalty < 1 {
				globalBreakdown.AddLine(key,
					luaSprintf("x %.2f ^8(number of repeats)", actor.Output["AverageBurstHits"]-1),
					luaSprintf("x %.2f ^8(repeat penalty)", repeatPenalty),
					luaSprintf("= %.1f ^8(repeat damage total)", actor.Output["AverageBurstDamage"]-pass.Output["AverageDamage"]),
					"",
					luaSprintf("+ %.1f ^8(first hit)", pass.Output["AverageHit"]),
				)
				if pass.Output["HitChance"] < 100 {
					globalBreakdown.AddLine(key, luaSprintf("x %.2f ^8(chance to hit)", pass.Output["HitChance"]/100))
				}
			} else {
				globalBreakdown.AddLine(key, luaSprintf("x %.2f ^8(number of hits)", actor.Output["AverageBurstHits"]))
			}
			globalBreakdown.AddLine(key, luaSprintf("= %.1f ^8(total burst damage)", actor.Output["AverageBurstDamage"]))
		}

		// Calculate PvP values
		//pob:block:src/Modules/CalcOffence.lua:3867-3954
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
			portionNonElemental := float64(math.Pow(output["AverageHit"]/PvpTvalue/PvpNonElemental2, PvpNonElemental1) * PvpTvalue * PvpNonElemental2 * percentageNonElemental)
			portionElemental := float64(math.Pow(output["AverageHit"]/PvpTvalue/PvpElemental2, PvpElemental1) * PvpTvalue * PvpElemental2 * percentageElemental)
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
				breakdown.AddLine("PvpAverageHit", luaSprintf("Pvp Formula is (D/(T*M))^E*T*%s, where D is the damage, T is the time taken,", utils.Ternary(percentBoth, "M*P", "M")))
				breakdown.AddLine("PvpAverageHit", luaSprintf(" M is the multiplier%s", utils.Ternary(percentBoth, ", E is the exponent and P is the percentage of that type (ele or non ele)", " and E is the exponent")))
				if percentBoth {
					breakdown.AddLine("PvpAverageHit", luaSprintf("(M= %.1f for ele and %.1f for non-ele)(E= %.2f for ele and %.2f for non-ele)", PvpElemental2, PvpNonElemental2, PvpElemental1, PvpNonElemental1))
					breakdown.AddLine("PvpAverageHit", luaSprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f * %.2f = %.1f", output["AverageHit"], PvpTvalue, PvpNonElemental2, PvpNonElemental1, PvpTvalue, PvpNonElemental2, percentageNonElemental, portionNonElemental))
					breakdown.AddLine("PvpAverageHit", luaSprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f * %.2f = %.1f", output["AverageHit"], PvpTvalue, PvpElemental2, PvpElemental1, PvpTvalue, PvpElemental2, percentageElemental, portionElemental))
					breakdown.AddLine("PvpAverageHit", luaSprintf("(portionNonElemental + portionElemental)%s", utils.Ternary(PvpMultiplier != 1, " * PvP multiplier", " ")))
					if PvpMultiplier != 1 {
						breakdown.AddLine("PvpAverageHit", luaSprintf("(%.1f + %.1f) * %g", portionNonElemental, portionElemental, PvpMultiplier))
					} else {
						breakdown.AddLine("PvpAverageHit", luaSprintf("%.1f + %.1f", portionNonElemental, portionElemental))
					}
				} else if percentageElemental <= 0 {
					breakdown.AddLine("PvpAverageHit", luaSprintf("(M= %.1f for non-ele)(E= %.2f for non-ele)", PvpNonElemental2, PvpNonElemental1))
					breakdown.AddLine("PvpAverageHit", luaSprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f = %.1f", output["AverageHit"], PvpTvalue, PvpNonElemental2, PvpNonElemental1, PvpTvalue, PvpNonElemental2, portionNonElemental))
					if PvpMultiplier != 1 {
						breakdown.AddLine("PvpAverageHit", luaSprintf("%.1f * %g ^8(portionNonElemental * PvP multiplier)", portionNonElemental, PvpMultiplier))
					}
				} else if percentageNonElemental <= 0 {
					breakdown.AddLine("PvpAverageHit", luaSprintf("(M= %.1f for ele)(E= %.2f for ele)", PvpElemental2, PvpElemental1))
					breakdown.AddLine("PvpAverageHit", luaSprintf("(%.1f / (%.2f * %.1f)) ^ %.2f * %.2f * %.1f = %.1f", output["AverageHit"], PvpTvalue, PvpElemental2, PvpElemental1, PvpTvalue, PvpElemental2, portionElemental))
					if PvpMultiplier != 1 {
						breakdown.AddLine("PvpAverageHit", luaSprintf("%.1f * %g ^8(portionElemental * PvP multiplier)", portionElemental, PvpMultiplier))
					}
				}
				breakdown.AddLine("PvpAverageHit", luaSprintf("= %.1f", output["PvpAverageHit"]))
				if isAttack {
					breakdown.AddLine("PvpAverageDamage", luaSprintf("%s:", pass.Label))
					breakdown.AddLine("PvpAverageDamage", luaSprintf("%.1f ^8(average pvp hit)", output["PvpAverageHit"]))
					breakdown.AddLine("PvpAverageDamage", luaSprintf("x %.2f ^8(chance to hit)", output["HitChance"]/100))
					breakdown.AddLine("PvpAverageDamage", luaSprintf("= %.1f", output["PvpAverageDamage"]))
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
		//pob:src/Modules/CalcOffence.lua:3963
		combineStat("PvpAverageDamage", "DPS")
		combineStat("TotalDPS", "DPS")
		//pob:src/Modules/CalcOffence.lua:3965
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
		//pob:block:src/Modules/CalcOffence.lua:3991-4020
		if breakdown != nil {
			if skillFlags[SkillFlagBothWeaponAttack] {
				for _, critStat := range []string{"PreEffectiveCritChance", "CritChance"} {
					combined := []string{"Both weapons:"}
					for _, pass := range passList {
						combined = append(combined, pass.Label+":")
						if pass.Breakdown != nil && pass.Breakdown.Data[critStat] != nil {
							combined = append(combined, pass.Breakdown.Data[critStat].Lines...)
						} else {
							combined = append(combined, luaSprintf("%.2f%%", pass.Output[critStat]))
						}
					}
					combined = append(combined, luaSprintf("= %.2f%% ^8(average)", actor.Output[critStat]))
					breakdown.Data[critStat] = &Entry{Lines: combined}
				}
			} else if handBreakdown := utils.Ternary(skillFlags[SkillFlagWeapon1Attack], breakdown.mainHand, breakdown.offHand); handBreakdown != nil {
				setBreakdownEntry(breakdown, "PreEffectiveCritChance", handBreakdown.Data["PreEffectiveCritChance"])
				setBreakdownEntry(breakdown, "CritChance", handBreakdown.Data["CritChance"])
			}
			breakdownSource := utils.Ternary(skillFlags[SkillFlagWeapon1Attack], "MainHand.CritChance", "OffHand.CritChance")
			for _, critStat := range []string{"PreEffectiveCritChance", "CritChance"} {
				if entry := breakdown.Data[critStat]; entry != nil {
					entry.BreakdownSource = breakdownSource
				}
			}
		}

		//pob:block:src/Modules/CalcOffence.lua:4021-4030
		if skillFlags[SkillFlagBothWeaponAttack] {
			if breakdown != nil {
				breakdown.Data["AverageDamage"] = &Entry{Lines: []string{"Both weapons:"}}
				if skillData.DoubleHitsWhenDualWielding {
					breakdown.AddLine("AverageDamage", luaSprintf("%.1f + %.1f ^8(skill hits with both weapons at once)", actor.OutputTable[OutTableMainHand]["AverageDamage"], actor.OutputTable[OutTableOffHand]["AverageDamage"]))
				} else {
					breakdown.AddLine("AverageDamage", luaSprintf("(%.1f + %.1f) / 2 ^8(skill alternates weapons)", actor.OutputTable[OutTableMainHand]["AverageDamage"], actor.OutputTable[OutTableOffHand]["AverageDamage"]))
				}
				breakdown.AddLine("AverageDamage", luaSprintf("= %.1f", actor.Output["AverageDamage"]))
			}
		}
	}

	/*
		TODO CALCS
		if env.mode == "CALCS" {
			if skillData.ShowAverage {
				actor.Output["DisplayDamage"] = formatNumSep(luaSprintf("%.1f", actor.Output["AverageDamage"])) + " average damage"
			} else {
				actor.Output["DisplayDamage"] = formatNumSep(luaSprintf("%.1f", actor.Output["TotalDPS"])) + " DPS"
			}
		}
	*/
	//pob:block:src/Modules/CalcOffence.lua:4051-4074
	if breakdown != nil {
		rate := luaSprintf("x %.2f ^8(cast rate)", actor.Output["Speed"])
		if isAttack {
			rate = luaSprintf("x %.2f ^8(attack rate)", actor.Output["Speed"])
		} else if skillData.Triggered {
			rate = luaSprintf("x %.2f ^8(trigger rate)", actor.Output["Speed"])
		}
		if utils.Has(actor.Output, "HitSpeed") {
			rate = luaSprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"])
		}
		breakdown.Data["TotalDPS"] = &Entry{Lines: []string{
			luaSprintf("%.1f ^8(%s)", actor.Output["AverageDamage"], utils.Ternary(isAttack || skillData.Triggered, "average damage", "average hit")),
			rate,
		}}
		if dpsMultiplier := utils.OrDefault(skillData.DpsMultiplier, 1); dpsMultiplier != 1 {
			breakdown.AddLine("TotalDPS", luaSprintf("x %.6g ^8(DPS multiplier for this skill)", dpsMultiplier))
		}
		if quantityMultiplier > 1 {
			breakdown.AddLine("TotalDPS", luaSprintf("x %.6g ^8(quantity multiplier for this skill)", quantityMultiplier))
		}
		breakdown.AddLine("TotalDPS", luaSprintf("= %.1f", actor.Output["TotalDPS"]))
		//pob:block:src/Modules/CalcOffence.lua:4075-4093
		if env.Build.GetBooleanOption("PvpScaling") {
			rateType := "cast"
			if isAttack {
				rateType = "attack"
			} else if skillData.Triggered {
				rateType = "trigger"
			}
			breakdown.AddLine("PvpTotalDPS",
				luaSprintf("%.1f ^8(average pvp hit)", actor.Output["PvpAverageDamage"]),
				utils.Ternary(actor.Output["HitSpeed"] != 0, luaSprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"]), luaSprintf("x %.2f ^8(%s rate)", actor.Output["Speed"], rateType)),
			)
			if skillData.DpsMultiplier != 1 {
				breakdown.AddLine("PvpTotalDPS", luaSprintf("x %g ^8(DPS multiplier for this skill)", skillData.DpsMultiplier))
			}
			if quantityMultiplier > 1 {
				breakdown.AddLine("PvpTotalDPS", luaSprintf("x %g ^8(quantity multiplier for this skill)", quantityMultiplier))
			}
			breakdown.AddLine("PvpTotalDPS", luaSprintf("= %.1f", actor.Output["PvpTotalDPS"]))
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:4096-4098
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

	actor.Output["LifeLeechRate"] = actor.Output["LifeLeechInstantRate"] + float64(min(actor.Output["LifeLeechRate"], actor.Output["MaxLifeLeechRate"])*actor.Output["LifeRecoveryRateMod"])
	actor.Output["LifeLeechPerHit"] = actor.Output["LifeLeechInstant"] + float64(min(actor.Output["LifeLeechPerHit"], actor.Output["MaxLifeLeechRate"])*actor.Output["LifeLeechDuration"]*actor.Output["LifeRecoveryRateMod"])
	actor.Output["EnergyShieldLeechRate"] = actor.Output["EnergyShieldLeechInstantRate"] + float64(min(actor.Output["EnergyShieldLeechRate"], actor.Output["MaxEnergyShieldLeechRate"])*actor.Output["EnergyShieldRecoveryRateMod"])
	actor.Output["EnergyShieldLeechPerHit"] = actor.Output["EnergyShieldLeechInstant"] + float64(min(actor.Output["EnergyShieldLeechPerHit"], actor.Output["MaxEnergyShieldLeechRate"])*actor.Output["EnergyShieldLeechDuration"]*actor.Output["EnergyShieldRecoveryRateMod"])
	actor.Output["ManaLeechRate"] = actor.Output["ManaLeechInstantRate"] + float64(min(actor.Output["ManaLeechRate"], actor.Output["MaxManaLeechRate"])*actor.Output["ManaRecoveryRateMod"])
	actor.Output["ManaLeechPerHit"] = actor.Output["ManaLeechInstant"] + float64(min(actor.Output["ManaLeechPerHit"], actor.Output["MaxManaLeechRate"])*actor.Output["ManaLeechDuration"]*actor.Output["ManaRecoveryRateMod"])

	//pob:block:src/Modules/CalcOffence.lua:4130-4132
	skillFlags["leechLife"] = actor.Output["LifeLeechRate"] > 0
	skillFlags["leechES"] = actor.Output["EnergyShieldLeechRate"] > 0
	skillFlags["leechMana"] = actor.Output["ManaLeechRate"] > 0
	if skillData.ShowAverage {
		actor.Output["LifeLeechGainPerHit"] = actor.Output["LifeLeechPerHit"] + actor.Output["LifeOnHit"]
		actor.Output["EnergyShieldLeechGainPerHit"] = actor.Output["EnergyShieldLeechPerHit"] + actor.Output["EnergyShieldOnHit"]
		actor.Output["ManaLeechGainPerHit"] = actor.Output["ManaLeechPerHit"] + actor.Output["ManaOnHit"]
	} else {
		actor.Output["LifeLeechGainRate"] = actor.Output["LifeLeechRate"] + actor.Output["LifeOnHitRate"]
		actor.Output["EnergyShieldLeechGainRate"] = actor.Output["EnergyShieldLeechRate"] + actor.Output["EnergyShieldOnHitRate"]
		actor.Output["ManaLeechGainRate"] = actor.Output["ManaLeechRate"] + actor.Output["ManaOnHitRate"]
	}
	//pob:block:src/Modules/CalcOffence.lua:4142-4153
	if breakdown != nil {
		hitRate := actor.Output["HitChance"] / 100 * utils.OrDefault(actor.Output["HitSpeed"], actor.Output["Speed"]) * utils.OrDefault(skillData.DpsMultiplier, 1)
		if skillFlags["leechLife"] {
			breakdown.Data["LifeLeech"] = &Entry{Lines: breakdown.leech(actor.Output["LifeLeechInstant"], actor.Output["LifeLeechInstantRate"], actor.Output["LifeLeechInstances"], actor.Output["Life"], "LifeLeechRate", actor.Output["MaxLifeLeechRate"], actor.Output["LifeLeechDuration"], actor.Output["LifeLeechInstantProportion"], hitRate)}
		}
		if skillFlags["leechES"] {
			breakdown.Data["EnergyShieldLeech"] = &Entry{Lines: breakdown.leech(actor.Output["EnergyShieldLeechInstant"], actor.Output["EnergyShieldLeechInstantRate"], actor.Output["EnergyShieldLeechInstances"], actor.Output["EnergyShield"], "EnergyShieldLeechRate", actor.Output["MaxEnergyShieldLeechRate"], actor.Output["EnergyShieldLeechDuration"], actor.Output["EnergyShieldLeechInstantProportion"], hitRate)}
		}
		if skillFlags["leechMana"] {
			breakdown.Data["ManaLeech"] = &Entry{Lines: breakdown.leech(actor.Output["ManaLeechInstant"], actor.Output["ManaLeechInstantRate"], actor.Output["ManaLeechInstances"], actor.Output["Mana"], "ManaLeechRate", actor.Output["MaxManaLeechRate"], actor.Output["ManaLeechDuration"], actor.Output["ManaLeechInstantProportion"], hitRate)}
		}
	}
	skillFlags[SkillFlagIgniteCanStack] = skillModList.Flag(skillCfg, "IgniteCanStack")
	skillFlags[SkillFlagIgniteToChaos] = skillModList.Flag(skillCfg, "IgniteToChaos")

	for _, pass := range passList {
		globalOutput := actor.Output
		output, cfg := pass.Output, pass.Config
		globalBreakdown, breakdown := breakdown, pass.Breakdown

		//pob:block:src/Modules/CalcOffence.lua:4185-4192
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

		//pob:block:src/Modules/CalcOffence.lua:4231-4262
		for _, ailment := range data.ElementalAilment("").Values() {
			aNames := ailmentNameSets[ailment]
			chance := skillModList.Sum(mod.TypeBase, cfg, aNames.enemyChance) + enemyDB.Sum(mod.TypeBase, nil, aNames.selfChance)
			avoidPercent := enemyDB.Sum(mod.TypeBase, nil, aNames.avoid, "AvoidAilments", "AvoidElementalAilments")
			if enemyDB.Flag(nil, aNames.immune, "ElementalAilmentImmune") {
				avoidPercent = 100
			}
			avoid := 1 - math.Min(avoidPercent, 100)/100
			if ailment == data.AilmentChill {
				chance = 100
			}
			chance *= avoid

			if ailment == data.AilmentIgnite && modDB.Flag(nil, "IgniteCanScorch") {
				output["ScorchChance"] = min(100, chance)
				skillModList.AddMod(mod.NewFloat("EnemyScorchChance", mod.TypeBase, chance).Source("Ignite Chance"))
			}

			if skillFlags[SkillFlagHit] && !skillModList.Flag(cfg, aNames.cannot) && avoid != 0 {
				output[aNames.chanceOnHit] = min(100, chance)

				if skillModList.Flag(cfg, aNames.critsDontAlways) || // e.g. Painseeker
					(ailment.IsNonDamaging() && data.NonDamagingAilments[ailment].Alt && !skillModList.Flag(cfg, "CritAlwaysAltAilments")) { // e.g. Secrets of Suffering
					output[aNames.chanceOnCrit] = output[aNames.chanceOnHit]
				} else {
					output[aNames.chanceOnCrit] = 100
				}
			} else {
				output[aNames.chanceOnHit] = 0
				output[aNames.chanceOnCrit] = 0
			}

			if ailment == data.AilmentScorch && modDB.Flag(nil, "IgniteCanScorch") && avoid != 0 {
				output["ScorchChanceOnCrit"] = 100
			}

			ailmentChance := output[aNames.chanceOnHit]
			if !skillModList.Flag(cfg, "NeverCrit") {
				ailmentChance += output[aNames.chanceOnCrit]
			}

			if ailmentChance > 0 {
				skillFlags[aNames.inflict] = true
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

		//pob:block:src/Modules/CalcOffence.lua:4278-4287
		if env.ModeEffective {
			for _, ailment := range data.Ailment("").Values() {
				aNames := ailmentNameSets[ailment]
				mult := 1 - enemyDB.Sum(mod.TypeBase, nil, aNames.avoid)/100
				if enemyDB.Flag(nil, aNames.immune) {
					mult = 0
				}
				output[aNames.chanceOnHit] = output[aNames.chanceOnHit] * mult
				output[aNames.chanceOnCrit] = output[aNames.chanceOnCrit] * mult
				if ailment == data.AilmentPoison {
					output["ChaosPoisonChance"] = output["ChaosPoisonChance"] * mult
				}
			}
		}

		//pob:block:src/Modules/CalcOffence.lua:4289-4294
		ailmentMode := utils.OrS(env.Build.GetStringOption("ailmentMode"), "AVERAGE")
		if ailmentMode == "CRIT" || modDB.Flag(nil, "AilmentsOnlyFromCrit") {
			for _, ailment := range data.Ailment("").Values() {
				output[string(ailment)+"ChanceOnHit"] = 0
			}
		}

		//pob:block:src/Modules/CalcOffence.lua:4299-4303
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

		//pob:block:src/Modules/CalcOffence.lua:4333-4410
		calcAilmentDamage := func(typee string, sourceCritChance float64, sourceHitDmg float64, sourceCritDmg float64, hideFromBreakdown bool) float64 {
			// Calculate the inflict chance and base damage of a secondary effect (bleed/poison/ignite/shock/freeze)
			chanceOnHit, chanceOnCrit := output[typee+"ChanceOnHit"], output[typee+"ChanceOnCrit"]
			chanceFromHit := float64(chanceOnHit * (1 - sourceCritChance/100))
			chanceFromCrit := chanceOnCrit * sourceCritChance / 100
			chance := chanceFromHit + chanceFromCrit
			output[typee+"Chance"] = chance
			baseFromHit := sourceHitDmg * chanceFromHit / (chanceFromHit + chanceFromCrit)
			baseFromCrit := sourceCritDmg * chanceFromCrit / (chanceFromHit + chanceFromCrit)
			baseVal := baseFromHit + baseFromCrit
			sourceMult := skillModList.More(cfg, typee+"AsThoughDealing")
			//pob:block:src/Modules/CalcOffence.lua:4344-4362
			if breakdown != nil && chance != 0 && !hideFromBreakdown {
				key := typee + "Chance"
				if breakdown.Data[key] == nil {
					breakdown.Data[key] = &Entry{}
				}
				if len(breakdown.Data[key].Lines) > 0 {
					breakdown.AddLine(key, "")
				}
				if isAttack {
					breakdown.AddLine(key, pass.Label+":")
				}
				breakdown.AddLine(key,
					luaSprintf("Chance on Non-crit: %d%%", luaInt(chanceOnHit)),
					luaSprintf("Chance on Crit: %d%%", luaInt(chanceOnCrit)),
				)
				if chanceOnHit != chanceOnCrit {
					breakdown.AddLine(key,
						"Combined chance:",
						luaSprintf("%d x (1 - %.4f) ^8(chance from non-crits)", luaInt(chanceOnHit), sourceCritChance/100),
						luaSprintf("+ %d x %.4f ^8(chance from crits)", luaInt(chanceOnCrit), sourceCritChance/100),
						luaSprintf("= %.2f", float64(chanceOnHit*(1-sourceCritChance/100))+float64(chanceOnCrit*sourceCritChance/100)),
					)
				}
			}
			//pob:block:src/Modules/CalcOffence.lua:4363-4408
			if breakdown != nil && baseVal > 0 && !hideFromBreakdown {
				key := typee + "DPS"
				if breakdown.Data[key] != nil && len(breakdown.Data[key].Lines) > 0 {
					breakdown.AddLine(key, "")
				}
				if isAttack {
					breakdown.AddLine(key, pass.Label+":")
				}
				if sourceHitDmg == sourceCritDmg || output["CritChance"] == 0 {
					breakdown.AddLine(key, "Total damage:", luaSprintf("%.1f ^8(source damage)", sourceHitDmg))
					if sourceMult > 1 {
						breakdown.AddLine(key,
							luaSprintf("x %.2f ^8(inflicting as though dealing more damage)", sourceMult),
							luaSprintf("= %.1f", baseVal*sourceMult),
						)
					}
				} else {
					if baseFromHit > 0 {
						breakdown.AddLine(key,
							"Damage from Non-crits:",
							luaSprintf("%.1f ^8(source damage from non-crits)", sourceHitDmg),
							luaSprintf("x %.3f ^8(portion of instances created by non-crits)", chanceFromHit/(chanceFromHit+chanceFromCrit)),
						)
						if sourceMult == 1 || baseFromCrit != 0 {
							breakdown.AddLine(key, luaSprintf("= %.1f", baseFromHit))
						}
					}
					if baseFromCrit > 0 {
						breakdown.AddLine(key,
							"Damage from Crits:",
							luaSprintf("%.1f ^8(source damage from crits)", sourceCritDmg),
							luaSprintf("x %.3f ^8(portion of instances created by crits)", chanceFromCrit/(chanceFromHit+chanceFromCrit)),
						)
						if sourceMult == 1 || baseFromHit != 0 {
							breakdown.AddLine(key, luaSprintf("= %.1f", baseFromCrit))
						}
					}
					if baseFromHit > 0 && baseFromCrit > 0 {
						breakdown.AddLine(key, "Total damage:", luaSprintf("%.1f + %.1f", baseFromHit, baseFromCrit))
						if sourceMult == 1 {
							breakdown.AddLine(key, luaSprintf("= %.1f", baseVal))
						}
					}
					if sourceMult > 1 {
						breakdown.AddLine(key,
							luaSprintf("x %.2f ^8(inflicting as though dealing more damage)", sourceMult),
							luaSprintf("= %.1f", baseVal*sourceMult),
						)
					}
				}
			}
			return baseVal
		}

		// Calculate bleeding chance and damage
		if canDeal[data.DamageTypePhysical] && (output["BleedChanceOnHit"]+output["BleedChanceOnCrit"]) > 0 {
			badIdea := ailmentSkillCond(skillCfg, cfg)

			//pob:src/Modules/CalcOffence.lua:3360-3369
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
			//pob:block:src/Modules/CalcOffence.lua:4428-4430
			if breakdown != nil {
				breakdown.Data["BleedPhysical"] = &Entry{}
			}

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
			//pob:src/Modules/CalcOffence.lua:4438-4439
			durationMod := math.Max(CalcMod(skillModList, dotCfg, names...)*CalcMod(enemyDB, nil, "SelfBleedDuration", "SelfAilmentDuration")/CalcMod(enemyDB, dotCfg, "BleedExpireRate"), 0)
			rateMod := CalcMod(skillModList, cfg, "BleedFaster") + enemyDB.Sum(mod.TypeIncrease, nil, "SelfBleedFaster")/100
			globalOutput["BleedDuration"] = durationBase * durationMod / rateMod * debuffDurationMult
			//pob:block:src/Modules/CalcOffence.lua:4443-4464
			bleedChance := float64(output["BleedChanceOnHit"]/100*(1-output["CritChance"]/100)) +
				output["BleedChanceOnCrit"]/100*output["CritChance"]/100
			bleedStacks := float64(output["HitChance"] / 100 * bleedChance * utils.OrDefault(skillData.DpsMultiplier, 1))
			hitRate := utils.OrF(globalOutput["HitSpeed"], globalOutput["Speed"])
			if hitRate > 0 {
				// assume skills with no cast, attack, or cooldown time are single cast
				bleedStacks = float64(bleedStacks * globalOutput["BleedDuration"] * hitRate)
			}
			activeTotems := utils.Or(env.ModDB.Override(nil, "TotemsSummoned"),
				skillModList.Sum(mod.TypeBase, skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit"))
			if skillFlags[SkillFlagTotem] {
				bleedStacks = float64(bleedStacks * activeTotems)
			}
			if configStacks > 0 {
				bleedStacks = configStacks
			}

			bleedStackPotential := bleedStacks / maxStacks
			o := skillModList.Override(nil, "BleedStackPotentialOverride")
			if o != nil {
				bleedStackPotential = o.Float() / maxStacks
			}
			globalOutput["BleedStackPotential"] = bleedStackPotential

			//pob:block:src/Modules/CalcOffence.lua:4466-4485
			if globalBreakdown != nil {
				key := "BleedStackPotential"
				globalBreakdown.Data[key] = &Entry{Lines: []string{"Ratio of bleed stacks to maximum active bleeds:"}}
				if o != nil {
					globalBreakdown.AddLine(key, luaSprintf("= %g ^8(stack potential override)", bleedStackPotential))
				} else {
					globalBreakdown.AddLine(key,
						luaSprintf("%.2f ^8(chance to hit)", output["HitChance"]/100),
						luaSprintf("* %.2f ^8(chance to bleed)", bleedChance),
					)
					if hitRate > 0 {
						globalBreakdown.AddLine(key, luaSprintf("* (%.2f / %.2f) ^8(Bleed duration / Attack Time)", globalOutput["BleedDuration"], utils.OrDefault(output["HitTime"], output["Time"])))
					}
					if dpsMultiplier := utils.OrDefault(skillData.DpsMultiplier, 1); dpsMultiplier != 1 {
						globalBreakdown.AddLine(key, luaSprintf("* %g ^8(DPS multiplier for this skill)", dpsMultiplier))
					}
					if skillFlags[SkillFlagTotem] {
						globalBreakdown.AddLine(key, luaSprintf("* %d ^8(active number of Totems)", luaInt(activeTotems)))
					}
					globalBreakdown.AddLine(key,
						luaSprintf("/ %d ^8(max number of stacks)", luaInt(maxStacks)),
						luaSprintf("= %.2f", bleedStackPotential),
					)
				}
			}

			//pob:block:src/Modules/CalcOffence.lua:4487-4496
			bleedRollAverage := float64(50)
			if bleedStackPotential > 1 {
				bleedRollAverage = float64((bleedStacks - (maxStacks-1)/2) / (bleedStacks + 1) * 100)
			}
			globalOutput["BleedRollAverage"] = bleedRollAverage
			//pob:block:src/Modules/CalcOffence.lua:4497-4511
			if globalBreakdown != nil {
				key := "BleedRollAverage"
				globalBreakdown.Data[key] = &Entry{Lines: []string{
					colorCustom + "This is the average roll of a Bleed affecting the enemy if you are constantly attacking",
					luaSprintf(colorCustom+"If attacking constantly, your average strongest bleed currently achieves ^7%.2f%%"+colorCustom+" of its max damage", bleedRollAverage),
					"",
					"Average Bleed Roll:",
				}}
				if bleedStacks >= maxStacks {
					globalBreakdown.AddLine(key,
						luaSprintf("%.2f - (%.2f - 1)/2", bleedStacks, maxStacks),
						luaSprintf("/ (%.2f + 1)", bleedStacks),
						luaSprintf("= %.2f%%", bleedRollAverage),
					)
				} else {
					globalBreakdown.AddLine(key, luaSprintf("50%% (averaging <= %d bleeds)", luaInt(maxStacks)))
				}
			}

			var sourceMinHitDmg, sourceMaxHitDmg, sourceMaxCritDmg float64

			for _, sub_pass := range []int{1, 2} {
				dotCfg.SkillCond["CriticalStrike"] = sub_pass != 1

				//pob:block:src/Modules/CalcOffence.lua:4516
				output["BleedPhysicalMin"], output["BleedPhysicalMax"] = calcAilmentSourceDamage(activeSkill, output, dotCfg, utils.Ternary(sub_pass == 1, breakdown, nil), "BleedPhysical", "Physical", 0)

				minDmg, maxDmg := output["BleedPhysicalMin"], output["BleedPhysicalMax"]
				//pob:block:src/Modules/CalcOffence.lua:4519-4535
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
					sourceMinCritDmg := float64(minDmg * output["CritBleedDotMulti"])
					sourceMaxCritDmg = float64(maxDmg * output["CritBleedDotMulti"])
					sourceCritDmg = sourceMinCritDmg + (sourceMaxCritDmg-sourceMinCritDmg)*bleedRollAverage/100
				} else {
					output["BleedDotMulti"] = bleedDotMulti()
					sourceMinHitDmg = float64(minDmg * output["BleedDotMulti"])
					sourceMaxHitDmg = float64(maxDmg * output["BleedDotMulti"])
					sourceHitDmg = sourceMinHitDmg + (sourceMaxHitDmg-sourceMinHitDmg)*bleedRollAverage/100
				}
			}

			//pob:block:src/Modules/CalcOffence.lua:4538-4567
			if breakdown != nil {
				breakdown.Data["BleedDPS"] = &Entry{Lines: []string{colorCustom + "NOTE: Calculation uses new Weighted Avg formula for critical hits"}}
			}
			if globalBreakdown != nil {
				derivation := func(multiLabel string, multi float64, format string, avg float64) []string {
					return []string{
						luaSprintf("%.0f + (%.0f - %.0f) * %.2f%%", output["BleedPhysicalMin"], output["BleedPhysicalMax"], output["BleedPhysicalMin"], bleedRollAverage),
						"^8min source phys + (max source phys - min source phys) * average bleed roll",
						luaSprintf("* %.2f ^8(%s)", multi, multiLabel),
						luaSprintf(format, avg),
					}
				}
				if sourceHitDmg == sourceCritDmg || output["CritChance"] == 0 {
					globalBreakdown.Data["BleedDPS"] = &Entry{Lines: append([]string{"Dmg Derivation:"}, derivation("Bleed DoT Multi", output["BleedDotMulti"], "= %.2f", sourceHitDmg)...)}
				} else {
					lines := append([]string{"Non-Crit Dmg Derivation:"}, derivation("Bleed DoT Multi for Non-Crit", output["BleedDotMulti"], "= %.0f", sourceHitDmg)...)
					lines = append(lines, "", "Crit Dmg Derivation:")
					lines = append(lines, derivation("Bleed DoT Multi for Crit", output["CritBleedDotMulti"], "= %.2f", sourceCritDmg)...)
					globalBreakdown.Data["BleedDPS"] = &Entry{Lines: lines}
				}
			}

			basePercent := utils.Ternary(skillData.BleedBasePercent != 0, skillData.BleedBasePercent, data.BleedPercentBase)
			// over-stacking bleed stacks increases the chance a critical bleed is present
			//pob:block:src/Modules/CalcOffence.lua:3452-3454
			//pob:block:src/Modules/CalcOffence.lua:4570
			ailmentCritChance := 100 * (1 - math.Pow(1-output["CritChance"]/100, max(bleedStackPotential, 1)))
			//pob:block:src/Modules/CalcOffence.lua:4572-4573
			baseMinVal := calcAilmentDamage("Bleed", ailmentCritChance, sourceMinHitDmg, 0, true) * basePercent / 100
			baseMaxVal := calcAilmentDamage("Bleed", 100, sourceMaxHitDmg, sourceMaxCritDmg, true) * basePercent / 100 * output["RuthlessBlowAilmentEffect"] * output["FistOfWarDamageEffect"] * globalOutput["AilmentWarcryEffect"]
			baseVal := calcAilmentDamage("Bleed", ailmentCritChance, sourceHitDmg, sourceCritDmg, false) * basePercent / 100 * output["RuthlessBlowAilmentEffect"] * output["FistOfWarDamageEffect"] * globalOutput["AilmentWarcryEffect"]
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
					//pob:block:src/Modules/CalcOffence.lua:4586-4588
					if breakdown != nil && effMult != 1 {
						globalBreakdown.EffMult("BleedEffMult", data.DamageTypePhysical, resist, 0, takenInc, effMult, takenMore, "", true, 0, nil)
					}
				}

				effectMod := CalcMod(skillModList, dotCfg, "AilmentEffect")
				//pob:block:src/Modules/CalcOffence.lua:4590-4601
				activeBleeds := min(bleedStacks, maxStacks)
				output["BaseBleedDPS"] = baseVal * effectMod * rateMod * activeBleeds * effMult
				bleedDPSUncapped := output["BaseBleedDPS"]
				bleedDPSCapped := min(bleedDPSUncapped, data.DotDpsCap)
				minBleedDPSUncapped := float64(baseMinVal * effectMod * rateMod * activeBleeds * effMult)
				minBleedDPSCapped := min(minBleedDPSUncapped, data.DotDpsCap)
				maxBleedDPSUncapped := float64(baseMaxVal * effectMod * rateMod * activeBleeds * effMult)
				maxBleedDPSCapped := min(maxBleedDPSUncapped, data.DotDpsCap)
				output["BleedDPS"] = bleedDPSCapped
				globalOutput["BleedStacks"] = bleedStacks
				globalOutput["BleedDamage"] = output["BaseBleedDPS"] * globalOutput["BleedDuration"]
				//pob:block:src/Modules/CalcOffence.lua:4602-4680
				if breakdown != nil {
					if critMulti, ok := output["CritBleedDotMulti"]; ok && critMulti != output["BleedDotMulti"] {
						chanceFromHit := float64(output["BleedChanceOnHit"] / 100 * (1 - globalOutput["CritChance"]/100))
						chanceFromCrit := float64(output["BleedChanceOnCrit"] / 100 * ailmentCritChance / 100)
						totalFromHit := chanceFromHit / (chanceFromHit + chanceFromCrit)
						totalFromCrit := chanceFromCrit / (chanceFromHit + chanceFromCrit)
						breakdown.Data["BleedDotMulti"] = &Entry{Lines: breakdownCritDot(output["BleedDotMulti"], critMulti, totalFromHit, totalFromCrit)}
						output["BleedDotMulti"] = float64(output["BleedDotMulti"]*totalFromHit) + float64(critMulti*totalFromCrit)
					}
					key := "BleedDPS"
					breakdown.AddLine(key, luaSprintf("x %.2f ^8(bleed deals %d%% per second)", basePercent/100, luaInt(basePercent)))
					if effectMod != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(ailment effect modifier)", effectMod))
					}
					if output["RuthlessBlowAilmentEffect"] != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(ruthless blow effect modifier)", output["RuthlessBlowAilmentEffect"]))
					}
					if output["FistOfWarDamageEffect"] != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(fist of war effect modifier)", output["FistOfWarDamageEffect"]))
					}
					if globalOutput["AilmentWarcryEffect"] > 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(combined ailment warcry effect modifier)", globalOutput["AilmentWarcryEffect"]))
					}
					breakdown.AddLine(key, luaSprintf("= %.1f", baseVal), "", "Total Bleed DPS:")
					if baseVal != bleedDPSUncapped {
						breakdown.AddLine(key, luaSprintf("%.1f ^8(base damage per second)", baseVal))
					}
					if effectMod != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(ailment effect modifier)", effectMod))
					}
					if rateMod != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(damage rate modifier)", rateMod))
					}
					if activeBleeds != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(avg bleed stacks)", activeBleeds))
					}
					if effMult != 1 {
						breakdown.AddLine(key, luaSprintf("x %.3f ^8(effective DPS modifier from enemy debuffs)", effMult))
					}
					if output["BleedDPS"] != bleedDPSUncapped {
						breakdown.AddLine(key,
							luaSprintf("= %.1f ^8(Uncapped raw Bleed DPS)", bleedDPSUncapped),
							luaSprintf("^8(Raw Bleed DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", bleedDPSUncapped-bleedDPSCapped, (bleedDPSUncapped-bleedDPSCapped)/bleedDPSCapped*100),
							luaSprintf("= %d ^8(Capped Bleed DPS)", luaInt(bleedDPSCapped)),
						)
					} else {
						breakdown.AddLine(key, luaSprintf("= %.1f ^8per second", output["BleedDPS"]))
					}
					breakdown.AddLine(key, luaSprintf("%.2f%% of Maximum Bleed DPS", output["BleedDPS"]/maxBleedDPSCapped*100), "", "DPS Range:")
					if maxBleedDPSCapped == maxBleedDPSUncapped && minBleedDPSCapped == minBleedDPSUncapped {
						breakdown.AddLine(key, luaSprintf("%.0f to %.0f ^8(Bleed DPS Range)", minBleedDPSUncapped, maxBleedDPSUncapped))
					} else {
						breakdown.AddLine(key, luaSprintf("%.0f to %.0f ^8(Uncapped Bleed DPS Range)", minBleedDPSUncapped, maxBleedDPSUncapped))
					}
					if minBleedDPSCapped != minBleedDPSUncapped {
						breakdown.AddLine(key, luaSprintf("^8(Raw Min Bleed DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", minBleedDPSUncapped-minBleedDPSCapped, (minBleedDPSUncapped-minBleedDPSCapped)/minBleedDPSCapped*100))
					}
					if maxBleedDPSCapped != maxBleedDPSUncapped {
						breakdown.AddLine(key, luaSprintf("^8(Raw Max Bleed DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", maxBleedDPSUncapped-maxBleedDPSCapped, (maxBleedDPSUncapped-maxBleedDPSCapped)/maxBleedDPSCapped*100))
					}
					if maxBleedDPSCapped != maxBleedDPSUncapped || minBleedDPSCapped != minBleedDPSUncapped {
						breakdown.AddLine(key, luaSprintf("%.0f to %.0f ^8(Capped Bleed DPS Range)", minBleedDPSCapped, maxBleedDPSCapped))
					}
					if globalOutput["BleedDuration"] != durationBase {
						key := "BleedDuration"
						globalBreakdown.Data[key] = &Entry{Lines: []string{luaSprintf("%.2fs ^8(base duration)", durationBase)}}
						if durationMod != 1 {
							globalBreakdown.AddLine(key, luaSprintf("x %.2f ^8(duration modifier)", durationMod))
						}
						if rateMod != 1 {
							globalBreakdown.AddLine(key, luaSprintf("/ %.2f ^8(damage rate modifier)", rateMod))
						}
						if debuffDurationMult != 1 {
							globalBreakdown.AddLine(key, luaSprintf("/ %.2f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
						}
						globalBreakdown.AddLine(key, luaSprintf("= %.2fs", globalOutput["BleedDuration"]))
					}
				}
			}
		}
		// Calculate poison chance and damage
		if canDeal[data.DamageTypeChaos] && (output["PoisonChanceOnHit"]+output["PoisonChanceOnCrit"]+output["ChaosPoisonChance"]) > 0 {
			badIdea := ailmentSkillCond(skillCfg, cfg)

			//pob:src/Modules/CalcOffence.lua:3556-3565
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

			//pob:block:src/Modules/CalcOffence.lua:4713-4746
			chaosPoisonChance := utils.Ternary(output["ChaosHitAverage"] > 0, output["ChaosPoisonChance"], 0)
			poisonChanceOnHit := min(100, output["PoisonChanceOnHit"]+chaosPoisonChance)
			poisonChanceOnCrit := min(100, output["PoisonChanceOnCrit"]+chaosPoisonChance)
			poisonChance := float64(poisonChanceOnHit/100*(1-output["CritChance"]/100)) + poisonChanceOnCrit/100*output["CritChance"]/100

			additionalPoisonStacks := 1.0
			if !skillModList.Flag(nil, "CannotMultiplePoison") {
				additionalPoisonStacks = 1 +
					min(skillModList.Sum(mod.TypeBase, cfg, "AdditionalPoisonChance")/100, 1) +
					skillModList.Sum(mod.TypeBase, cfg, "AdditionalPoisonStacks")
			}

			//pob:block:src/Modules/CalcOffence.lua:4726-4743
			poisonStackLimit, hasPoisonStackLimit := skillModList.Min(cfg, "PoisonStackLimit")
			poisonStacks := output["HitChance"] / 100 * poisonChance * additionalPoisonStacks *
				utils.OrDefault(skillData.DpsMultiplier, 1) *
				utils.OrDefault(skillData.StackMultiplier, 1) * quantityMultiplier
			var uncappedPoisonStacks *float64
			speed := utils.OrDefault(globalOutput["HitSpeed"], globalOutput["Speed"])
			if speed > 0 {
				poisonStacks = poisonStacks * globalOutput["PoisonDuration"] * speed

				if hasPoisonStackLimit && poisonStackLimit > 0 && poisonStacks > poisonStackLimit {
					numPoisoningHits := math.Ceil(poisonStackLimit / additionalPoisonStacks)
					maxPoisonStacks := numPoisoningHits * additionalPoisonStacks

					uncappedPoisonStacks = utils.Ptr(max(poisonStacks, maxPoisonStacks))
					poisonStacks = min(poisonStacks, maxPoisonStacks)
				}
			}
			poisonOnEnemy, _ := env.Build.GetNumberOptionOK("multiplierPoisonOnEnemy")
			if poisonStacks < additionalPoisonStacks && poisonOnEnemy == 0 {
				skillModList.AddMod(mod.NewFlag("Condition:NonPoisonedOnly", true).Source("Calculation"))
			}
			//pob:block:src/Modules/CalcOffence.lua:4747-4773
			if globalBreakdown != nil {
				key := "PoisonStacks"
				globalBreakdown.Data[key] = &Entry{}
				globalBreakdown.MultiChain(key, BMultiChain{
					Base: &BMultiChainItem{Format: "%.2fs ^8(poison duration)", Value: globalOutput["PoisonDuration"]},
					Items: []BMultiChainItem{
						{Format: "%.2f ^8(poison chance)", Value: poisonChance},
						{Format: "%.2f ^8(hit chance)", Value: output["HitChance"] / 100},
						{Format: "%.2f ^8(avg. # of poisons inflicted)", Value: additionalPoisonStacks},
						{Format: "%.2f ^8(hits per second)", Value: speed},
						{Format: "%g ^8(dps multiplier for this skill)", Value: utils.OrDefault(skillData.DpsMultiplier, 1)},
						{Format: "%g ^8(stack multiplier for this skill)", Value: utils.OrDefault(skillData.StackMultiplier, 1)},
						{Format: "%g ^8(quantity multiplier for this skill)", Value: quantityMultiplier},
					},
					Total: luaSprintf("= %.2f", poisonStacks),
				})
				if skillModList.Flag(nil, "Condition:NonPoisonedOnly") {
					globalBreakdown.AddLine(key, "Assuming 'non-Poisoned' Enemy")
					if poisonStacks < additionalPoisonStacks {
						globalBreakdown.AddLine(key, "^8(time between hits is longer than poison duration)")
					} else {
						globalBreakdown.AddLine(key, "^8(affected by poison stack limit of: "+luaNumber(additionalPoisonStacks)+")")
					}
				} else if hasPoisonStackLimit && poisonStacks >= poisonStackLimit {
					globalBreakdown.AddLine(key, "^8(affected by poison stack limit of: "+luaNumber(poisonStackLimit)+")")
					if uncappedPoisonStacks != nil {
						globalBreakdown.AddLine(key, "^8(uncapped poison stacks: "+luaSprintf("%.2f", *uncappedPoisonStacks)+")")
					}
				}
			}

			sourceHitDmg := float64(0)
			sourceCritDmg := float64(0)
			var sourceMinHitDmg, sourceMaxHitDmg, sourceMaxCritDmg float64

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
					totalMin += float64(minDmg * nonChaosMult)
					totalMax += float64(maxDmg * nonChaosMult)
				}

				if canDeal[data.DamageTypeCold] && skillModList.Flag(cfg, "ColdCanPoison") {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "PoisonCold", data.DamageTypeCold, data.DamageTypeFlags[data.DamageTypeChaos])
					output["PoisonColdMin"] = minDmg
					output["PoisonColdMax"] = maxDmg
					totalMin += float64(minDmg * nonChaosMult)
					totalMax += float64(maxDmg * nonChaosMult)
				}

				if canDeal[data.DamageTypeFire] && skillModList.Flag(cfg, "FireCanPoison") {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "PoisonFire", data.DamageTypeFire, data.DamageTypeFlags[data.DamageTypeChaos])
					output["PoisonFireMin"] = minDmg
					output["PoisonFireMax"] = maxDmg
					totalMin += float64(minDmg * nonChaosMult)
					totalMax += float64(maxDmg * nonChaosMult)
				}

				if canDeal[data.DamageTypePhysical] {
					minDmg, maxDmg := calcAilmentSourceDamage(activeSkill, output, dotCfg, subBreakdown, "PoisonPhysical", data.DamageTypePhysical, data.DamageTypeFlags[data.DamageTypeChaos])
					output["PoisonPhysicalMin"] = minDmg
					output["PoisonPhysicalMax"] = maxDmg
					totalMin += float64(minDmg * nonChaosMult)
					totalMax += float64(maxDmg * nonChaosMult)
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
					sourceMaxCritDmg = totalMax * output["CritPoisonDotMulti"]
				} else {
					output["PoisonDotMulti"] = poisonDotMulti()
					sourceHitDmg = (totalMin + totalMax) / 2 * output["PoisonDotMulti"]
					sourceMinHitDmg = totalMin * output["PoisonDotMulti"]
					sourceMaxHitDmg = totalMax * output["PoisonDotMulti"]
				}
			}
			//pob:block:src/Modules/CalcOffence.lua:4838-4842
			if globalBreakdown != nil {
				globalBreakdown.Data["PoisonDPS"] = &Entry{Lines: []string{
					luaSprintf("Ailment mode: %s ^8(can be changed in the Configuration tab)", utils.Ternary(ailmentMode == "CRIT", "Crits Only", "Average Damage")),
				}}
			}

			//pob:block:src/Modules/CalcOffence.lua:4843-4844
			baseMinVal := calcAilmentDamage("Poison", output["CritChance"], sourceMinHitDmg, 0, true) * data.PoisonPercentBase
			baseMaxVal := calcAilmentDamage("Poison", 100, sourceMaxHitDmg, sourceMaxCritDmg, true) * data.PoisonPercentBase * output["RuthlessBlowAilmentEffect"] * output["FistOfWarDamageEffect"] * globalOutput["AilmentWarcryEffect"]
			//pob:src/Modules/CalcOffence.lua:4845
			baseVal := calcAilmentDamage("Poison", output["CritChance"], sourceHitDmg, sourceCritDmg, false) * data.PoisonPercentBase * output["RuthlessBlowAilmentEffect"] * output["FistOfWarDamageEffect"] * globalOutput["AilmentWarcryEffect"]
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
					//pob:block:src/Modules/CalcOffence.lua:4856-4859
					if breakdown != nil && effMult != 1 {
						globalBreakdown.EffMult("PoisonEffMult", data.DamageTypeChaos, resist, 0, takenInc, effMult, takenMore, dotSourceRes(data.DamageTypeChaos), true, 0, nil)
					}
				}

				//pob:block:src/Modules/CalcOffence.lua:4861-4864
				if skillModList.Flag(nil, "Condition:NonPoisonedOnly") {
					poisonStacks = min(additionalPoisonStacks, poisonStacks)
				}
				globalOutput["PoisonStacks"] = poisonStacks

				effectMod := CalcMod(skillModList, dotCfg, "AilmentEffect")
				singlePoisonDPSUncapped := min(baseVal*effectMod*rateMod*effMult, data.DotDpsCap)
				singlePoisonDPSCapped := min(singlePoisonDPSUncapped, data.DotDpsCap)
				output["PoisonDPS"] = singlePoisonDPSCapped
				output["PoisonDamage"] = output["PoisonDPS"] * globalOutput["PoisonDuration"]
				minPoisonDPSUncapped := float64(baseMinVal * effectMod * rateMod * effMult)
				minPoisonDPSCapped := min(minPoisonDPSUncapped, data.DotDpsCap)
				maxPoisonDPSUncapped := float64(baseMaxVal * effectMod * rateMod * effMult)
				maxPoisonDPSCapped := min(maxPoisonDPSUncapped, data.DotDpsCap)

				//pob:block:src/Modules/CalcOffence.lua:4874-4879
				groundMult := max(
					skillModList.Max(nil, "PoisonDpsAsCausticGround"),
					enemyDB.Max(nil, "PoisonDpsAsCausticGround"),
				)
				if groundMult > 0 {
					output["CausticGroundDPS"] = min(baseVal*effectMod*rateMod*effMult*groundMult/100, data.DotDpsCap)
					//pob:block:src/Modules/CalcOffence.lua:4880-4887
					if breakdown != nil {
						breakdown.Data["CausticGroundDPS"] = &Entry{Lines: []string{
							luaSprintf("%.1f ^8(single poison damage per second)", baseVal*effectMod*rateMod),
							luaSprintf("* %.1f%% ^8(percent as Caustic ground)", groundMult),
							luaSprintf("* %.3f ^8(effect mult)", effMult),
							luaSprintf("= %.1f ^8per second", output["CausticGroundDPS"]),
						}}
					}
				}

				var poisonDPSUncapped, poisonDPSCapped float64
				if skillData.ShowAverage {
					//pob:src/Modules/CalcOffence.lua:4892
					output["TotalPoisonAverageDamage"] = output["PoisonDamage"]
					output["TotalPoisonDPS"] = output["PoisonDPS"]
				} else {
					poisonDPSUncapped = singlePoisonDPSCapped * poisonStacks
					poisonDPSCapped = min(poisonDPSUncapped, data.DotDpsCap)
					output["TotalPoisonDPS"] = poisonDPSCapped
				}
				//pob:block:src/Modules/CalcOffence.lua:4899-4986
				if breakdown != nil {
					if critMulti, ok := output["CritPoisonDotMulti"]; ok && critMulti != output["PoisonDotMulti"] {
						chanceFromHit := float64(output["PoisonChanceOnHit"] / 100 * (1 - globalOutput["CritChance"]/100))
						chanceFromCrit := float64(output["PoisonChanceOnCrit"] / 100 * globalOutput["CritChance"] / 100)
						totalFromHit := chanceFromHit / (chanceFromHit + chanceFromCrit)
						totalFromCrit := chanceFromCrit / (chanceFromHit + chanceFromCrit)
						breakdown.Data["PoisonDotMulti"] = &Entry{Lines: breakdownCritDot(output["PoisonDotMulti"], critMulti, totalFromHit, totalFromCrit)}
						output["PoisonDotMulti"] = float64(output["PoisonDotMulti"]*totalFromHit) + float64(critMulti*totalFromCrit)
					}
					key := "PoisonDPS"
					if output["RuthlessBlowAilmentEffect"] != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(ruthless blow effect modifier)", output["RuthlessBlowAilmentEffect"]))
					}
					breakdown.AddLine(key, "x 0.30 ^8(poison deals 30% per second)", luaSprintf("= %.1f", baseVal), "", "Poison DPS:")
					if baseVal != singlePoisonDPSUncapped {
						breakdown.AddLine(key, luaSprintf("%.1f ^8(base damage per second)", baseVal))
					}
					if effectMod != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(ailment effect modifier)", effectMod))
					}
					if rateMod != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(damage rate modifier)", rateMod))
					}
					if effMult != 1 {
						breakdown.AddLine(key, luaSprintf("x %.3f ^8(effective DPS modifier from enemy debuffs)", effMult))
					}
					if output["PoisonDPS"] != singlePoisonDPSUncapped {
						breakdown.AddLine(key,
							luaSprintf("= %.1f ^8(Uncapped raw Poison DPS)", singlePoisonDPSUncapped),
							luaSprintf("^8(Raw Poison DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", singlePoisonDPSUncapped-singlePoisonDPSCapped, (singlePoisonDPSUncapped-singlePoisonDPSCapped)/singlePoisonDPSCapped*100),
							luaSprintf("= %d ^8(Capped Poison DPS)", luaInt(singlePoisonDPSCapped)),
						)
					} else {
						breakdown.AddLine(key, luaSprintf("= %.1f ^8per second", output["PoisonDPS"]))
					}
					breakdown.AddLine(key, luaSprintf("%.2f%% of Maximum Poison DPS", output["PoisonDPS"]/maxPoisonDPSCapped*100), "", "DPS Range:")
					if maxPoisonDPSCapped == maxPoisonDPSUncapped && minPoisonDPSCapped == minPoisonDPSUncapped {
						breakdown.AddLine(key, luaSprintf("%.0f to %.0f ^8(Poison DPS Range)", minPoisonDPSUncapped, maxPoisonDPSUncapped))
					} else {
						breakdown.AddLine(key, luaSprintf("%.0f to %.0f ^8(Uncapped Poison DPS Range)", minPoisonDPSUncapped, maxPoisonDPSUncapped))
					}
					if minPoisonDPSCapped != minPoisonDPSUncapped {
						breakdown.AddLine(key, luaSprintf("^8(Raw Min Poison DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", minPoisonDPSUncapped-minPoisonDPSCapped, (minPoisonDPSUncapped-minPoisonDPSCapped)/minPoisonDPSCapped*100))
					}
					if maxPoisonDPSCapped != maxPoisonDPSUncapped {
						breakdown.AddLine(key, luaSprintf("^8(Raw Max Poison DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", maxPoisonDPSUncapped-maxPoisonDPSCapped, (maxPoisonDPSUncapped-maxPoisonDPSCapped)/maxPoisonDPSCapped*100))
					}
					if maxPoisonDPSCapped != maxPoisonDPSUncapped || minPoisonDPSCapped != minPoisonDPSUncapped {
						breakdown.AddLine(key, luaSprintf("%.0f to %.0f ^8(Capped Poison DPS Range)", minPoisonDPSCapped, maxPoisonDPSCapped))
					}
					if !skillData.ShowAverage && poisonStacks != 1 {
						key := "TotalPoisonDPS"
						breakdown.Data[key] = &Entry{Lines: []string{
							luaSprintf("%.1f ^8single poison damage per second", singlePoisonDPSCapped),
							luaSprintf("x %.2f ^8(poison stacks)", poisonStacks),
						}}
						if output["TotalPoisonDPS"] != poisonDPSUncapped {
							breakdown.AddLine(key,
								luaSprintf("= %.1f ^8(Uncapped raw Poison DPS)", poisonDPSUncapped),
								luaSprintf("^8(Raw Poison DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", poisonDPSUncapped-poisonDPSCapped, (poisonDPSUncapped-poisonDPSCapped)/poisonDPSCapped*100),
								luaSprintf("= %d ^8(Capped Poison DPS)", luaInt(poisonDPSCapped)),
							)
						} else {
							breakdown.AddLine(key, luaSprintf("= %.1f ^8per second", poisonDPSCapped))
						}
					}
					if globalOutput["PoisonDuration"] != 2 {
						key := "PoisonDuration"
						globalBreakdown.Data[key] = &Entry{Lines: []string{luaSprintf("%.2fs ^8(base duration)", durationBase)}}
						if durationMod != 1 {
							globalBreakdown.AddLine(key, luaSprintf("x %.2f ^8(duration modifier)", durationMod))
						}
						if rateMod != 1 {
							globalBreakdown.AddLine(key, luaSprintf("/ %.2f ^8(damage rate modifier)", rateMod))
						}
						if debuffDurationMult != 1 {
							globalBreakdown.AddLine(key, luaSprintf("/ %.2f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
						}
						globalBreakdown.AddLine(key, luaSprintf("= %.2fs", globalOutput["PoisonDuration"]))
					}
					key = "PoisonDamage"
					breakdown.Data[key] = &Entry{}
					if isAttack {
						breakdown.AddLine(key, pass.Label+":")
					}
					breakdown.AddLine(key,
						luaSprintf("%.1f ^8(damage per second)", singlePoisonDPSCapped),
						luaSprintf("x %.2fs ^8(poison duration)", globalOutput["PoisonDuration"]),
						luaSprintf("= %.1f ^8damage per poison stack", output["PoisonDamage"]),
					)
				}
			}
		}
		// Calculate ignite chance and damage
		if canDeal[data.DamageTypeFire] && (output["IgniteChanceOnHit"]+output["IgniteChanceOnCrit"]) > 0 {
			badIdea := ailmentSkillCond(skillCfg, cfg)

			//pob:src/Modules/CalcOffence.lua:4992-5003
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
			var sourceMaxCritDmg, sourceMinHitDmg, sourceMaxHitDmg float64
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

			//pob:block:src/Modules/CalcOffence.lua:5014
			globalOutput["IgniteChancePerHit"] = float64(output["IgniteChanceOnHit"]*(1-output["CritChance"]/100)) + output["IgniteChanceOnCrit"]*output["CritChance"]/100

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

			//pob:block:src/Modules/CalcOffence.lua:5029-5042
			igniteChance := float64(output["IgniteChanceOnHit"]/100*(1-output["CritChance"]/100)) +
				output["IgniteChanceOnCrit"]/100*output["CritChance"]/100
			igniteStacks := float64(output["HitChance"] / 100 * igniteChance * utils.OrDefault(skillData.DpsMultiplier, 1))
			if !skillData.TriggeredOnDeath {
				_, hasCooldown := output["Cooldown"]
				hitSpeed := utils.OrDefault(globalOutput["HitSpeed"], globalOutput["Speed"])
				switch {
				case hasCooldown:
					igniteStacks = igniteStacks * globalOutput["IgniteDuration"] /
						max(output["Cooldown"], utils.OrDefault(output["HitTime"], output["Time"]))
				case hitSpeed > 0:
					igniteStacks = float64(igniteStacks * globalOutput["IgniteDuration"] * hitSpeed)
				}
			}
			if overrideStackPotential != nil {
				globalOutput["IgniteStackPotential"] = *overrideStackPotential
			} else {
				globalOutput["IgniteStackPotential"] = igniteStacks / maxStacks
			}
			//pob:block:src/Modules/CalcOffence.lua:5043-5071
			if globalBreakdown != nil {
				key := "IgniteStackPotential"
				globalBreakdown.Data[key] = &Entry{Lines: []string{colorCustom + "NOTE: Calculation uses a Weighted Avg formula", ""}}
				if override := skillModList.Override(nil, "IgniteStackPotentialOverride"); override != nil {
					if maxStacks != 1 {
						globalBreakdown.AddLine(key,
							luaSprintf("= %d / %d ^8(stack potential override / max ignite stacks)", luaInt(override.Float()), luaInt(maxStacks)),
							luaSprintf("= %g ^8(stack potential)", *overrideStackPotential),
						)
					} else {
						globalBreakdown.AddLine(key, luaSprintf("= %g ^8(stack potential override)", *overrideStackPotential))
					}
				} else {
					globalBreakdown.AddLine(key,
						luaSprintf("%.2f ^8(chance to hit)", output["HitChance"]/100),
						luaSprintf("* %.2f ^8(chance to ignite)", igniteChance),
					)
					_, hasCooldown := output["Cooldown"]
					switch {
					case skillData.TriggeredOnDeath:
						globalBreakdown.AddLine(key, "* 1 ^8Cast on Death override")
					case hasCooldown:
						globalBreakdown.AddLine(key, luaSprintf("* (%.2f / max(%.2f, %.2f) ^8(Ignite Duration / max(Cooldown, Cast Time))", globalOutput["IgniteDuration"], output["Cooldown"], utils.OrDefault(output["HitTime"], output["Time"])))
					case utils.OrDefault(globalOutput["HitSpeed"], globalOutput["Speed"]) > 0:
						globalBreakdown.AddLine(key, luaSprintf("* (%.2f / %.2f) ^8(Ignite Duration / Attack Time)", globalOutput["IgniteDuration"], utils.OrDefault(globalOutput["HitTime"], output["Time"])))
					}
					if dpsMultiplier := utils.OrDefault(skillData.DpsMultiplier, 1); dpsMultiplier != 1 {
						globalBreakdown.AddLine(key, luaSprintf("* %g ^8(DPS multiplier for this skill)", dpsMultiplier))
					}
					globalBreakdown.AddLine(key,
						luaSprintf("/ %d ^8(max number of stacks)", luaInt(maxStacks)),
						luaSprintf("= %.2f", globalOutput["IgniteStackPotential"]),
					)
				}
			}

			//pob:block:src/Modules/CalcOffence.lua:5073-5082
			igniteRollAverage := float64(50)
			if globalOutput["IgniteStackPotential"] > 1 {
				igniteRollAverage = float64((igniteStacks - (maxStacks-1)/2) / (igniteStacks + 1) * 100)
			}
			globalOutput["IgniteRollAverage"] = igniteRollAverage
			//pob:block:src/Modules/CalcOffence.lua:5083-5097
			if globalBreakdown != nil {
				key := "IgniteRollAverage"
				globalBreakdown.Data[key] = &Entry{Lines: []string{
					colorCustom + "This is the average roll of an ignite affecting the enemy if you are constantly attacking",
					luaSprintf(colorCustom+"If attacking constantly, your average strongest Ignite currently achieves ^7%.2f%%"+colorCustom+" of its max damage", igniteRollAverage),
					"",
					"Average Ignite Roll:",
				}}
				if igniteStacks >= maxStacks {
					globalBreakdown.AddLine(key,
						luaSprintf("%.2f - (%.2f - 1)/2", igniteStacks, maxStacks),
						luaSprintf("/ (%.2f + 1)", igniteStacks),
						luaSprintf("= %.2f%%", igniteRollAverage),
					)
				} else {
					globalBreakdown.AddLine(key, luaSprintf("50%% (averaging <= %d ignites)", luaInt(maxStacks)))
				}
			}
			/*
				TODO Breakdown
				if globalBreakdown {
					globalBreakdown.IgniteStackPotential = {
						luaSprintf(colorCodes.CUSTOM+"NOTE: Calculation uses new Weighted Avg Ailment formula"),
						luaSprintf(""),
						luaSprintf("(%.2f / %.2f) ^8(IgniteDuration / Cast Time)", globalOutput.IgniteDuration, output.Time),
						luaSprintf("/ %d ^8(max number of stacks)", maxStacks),
						luaSprintf("= %.2f", globalOutput.IgniteStackPotential),
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
					sourceMaxCritDmg = totalMax * output["CritIgniteDotMulti"]
				} else {
					output["IgniteDotMulti"] = dotMulti()
					sourceHitDmg = (totalMin + (totalMax-totalMin)*igniteRollAverage/100) * output["IgniteDotMulti"]
					sourceMinHitDmg = totalMin * output["IgniteDotMulti"]
					sourceMaxHitDmg = totalMax * output["IgniteDotMulti"]
				}

				output["IgniteTotalMin"] = totalMin
				output["IgniteTotalMax"] = totalMax
			}

			//pob:block:src/Modules/CalcOffence.lua:5157-5181
			if globalBreakdown != nil {
				if sourceHitDmg == sourceCritDmg || output["CritChance"] == 0 {
					globalBreakdown.Data["IgniteDPS"] = &Entry{Lines: []string{
						"Dmg Derivation:",
						luaSprintf("%.0f + (%.0f - %.0f) * %.2f%%", output["IgniteTotalMin"], output["IgniteTotalMax"], output["IgniteTotalMin"], igniteRollAverage),
						"^8min combined sources + (max combined sources - min combined sources) * average ignite roll",
						luaSprintf("* %.2f ^8(Ignite DoT Multi)", output["IgniteDotMulti"]),
						luaSprintf("= %.2f", sourceHitDmg),
					}}
				} else {
					globalBreakdown.Data["IgniteDPS"] = &Entry{Lines: []string{
						"Non-Crit Dmg Derivation:",
						luaSprintf("%.0f + (%.0f - %.0f) * %.2f%%", output["IgniteTotalMin"], output["IgniteTotalMax"], output["IgniteTotalMin"], igniteRollAverage),
						"^8min combined sources + (max combined sources - min combined sources) * average ignite roll",
						luaSprintf("* %.2f ^8(Ignite DoT Multi for Non-Crit)", output["IgniteDotMulti"]),
						luaSprintf("= %.2f", sourceHitDmg),
						"",
						"Crit Dmg Derivation:",
						luaSprintf("%.0f + (%.0f - %.0f) * %.2f%%", output["IgniteTotalMin"], output["IgniteTotalMax"], output["IgniteTotalMin"], igniteRollAverage),
						"^8min combined sources + (max combined sources - min combined sources) * average ignite roll",
						luaSprintf("* %.2f ^8(Ignite DoT Multi for Crit)", output["CritIgniteDotMulti"]),
						luaSprintf("= %.2f", sourceCritDmg),
					}}
				}
			}
			// over-stacking ignite stacks increases the chance a critical ignite is present
			//pob:block:src/Modules/CalcOffence.lua:5183-5186
			ailmentCritChance := 100 * (1 - math.Pow(1-output["CritChance"]/100, max(1, igniteStacks)))
			baseMinVal := calcAilmentDamage("Ignite", ailmentCritChance, sourceMinHitDmg, 0, true) * data.IgnitePercentBase
			baseMaxVal := calcAilmentDamage("Ignite", 100, sourceMaxHitDmg, sourceMaxCritDmg, true) * data.IgnitePercentBase * output["RuthlessBlowAilmentEffect"] * output["FistOfWarDamageEffect"] * globalOutput["AilmentWarcryEffect"]
			baseVal := calcAilmentDamage("Ignite", ailmentCritChance, sourceHitDmg, sourceCritDmg, false) * data.IgnitePercentBase * output["RuthlessBlowAilmentEffect"] * output["FistOfWarDamageEffect"] * globalOutput["AilmentWarcryEffect"]
			if baseVal > 0 {
				skillFlags[SkillFlagIgnite] = true

				effMult := float64(1)
				if env.ModeEffective {
					if skillModList.Flag(cfg, "IgniteToChaos") {
						resist := calcResistForType(data.DamageTypeChaos, dotCfg)
						takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
						takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
						effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
						//pob:block:src/Modules/CalcOffence.lua:5197-5200
						if breakdown != nil && effMult != 1 {
							globalBreakdown.EffMult("IgniteEffMult", data.DamageTypeChaos, resist, 0, takenInc, effMult, takenMore, dotSourceRes(data.DamageTypeChaos), true, 0, nil)
						}
					} else {
						resist := calcResistForType(data.DamageTypeFire, dotCfg)
						takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
						takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
						effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
						//pob:block:src/Modules/CalcOffence.lua:5207-5210
						if breakdown != nil && effMult != 1 {
							globalBreakdown.EffMult("IgniteEffMult", data.DamageTypeFire, resist, 0, takenInc, effMult, takenMore, dotSourceRes(data.DamageTypeFire), true, 0, nil)
						}
					}
					globalOutput["IgniteEffMult"] = effMult
				}

				effectMod := CalcMod(skillModList, dotCfg, "AilmentEffect")
				//pob:block:src/Modules/CalcOffence.lua:5214-5215
				activeIgnites := min(igniteStacks, maxStacks)

				igniteDPSUncapped := float64(baseVal * effectMod * rateMod * activeIgnites * effMult)
				igniteDPSCapped := min(igniteDPSUncapped, data.DotDpsCap)
				minIgniteDPSUncapped := float64(baseMinVal * effectMod * rateMod * activeIgnites * effMult)
				minIgniteDPSCapped := min(minIgniteDPSUncapped, data.DotDpsCap)
				maxIgniteDPSUncapped := float64(baseMaxVal * effectMod * rateMod * activeIgnites * effMult)
				maxIgniteDPSCapped := min(maxIgniteDPSUncapped, data.DotDpsCap)
				output["IgniteDPS"] = igniteDPSCapped
				globalOutput["IgniteDamage"] = output["IgniteDPS"] * globalOutput["IgniteDuration"]

				//pob:block:src/Modules/CalcOffence.lua:5222-5231
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
					//pob:block:src/Modules/CalcOffence.lua:5233-5240
					if globalBreakdown != nil {
						globalBreakdown.Data["BurningGroundDPS"] = &Entry{Lines: []string{
							luaSprintf("%.1f ^8(ignite damage per second)", baseVal*effectMod*rateMod),
							luaSprintf("* %.1f%% ^8(percent as burning ground)", groundMult),
							luaSprintf("* %.3f ^8(effective DPS modifier)", fireEffMult),
							luaSprintf("= %.1f ^8per second", globalOutput["BurningGroundDPS"]),
						}}
					}
				}

				if skillFlags[SkillFlagIgniteCanStack] {
					output["IgniteDamage"] = output["IgniteDPS"] * globalOutput["IgniteDuration"]
					output["IgniteStacksMax"] = maxStacks
					output["TotalIgniteDPS"] = output["IgniteDPS"]
				}

				//pob:block:src/Modules/CalcOffence.lua:5249-5328
				if breakdown != nil {
					key := "IgniteDPS"
					if output["RuthlessBlowAilmentEffect"] != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(ruthless blow effect modifier)", output["RuthlessBlowAilmentEffect"]))
					}
					breakdown.AddLine(key, "x 0.9 ^8(ignite deals 90% per second)", luaSprintf("= %.1f", baseVal), "", "Average Ignite DPS:")
					if baseVal != igniteDPSUncapped {
						breakdown.AddLine(key, luaSprintf("%.1f ^8(base damage per second)", baseVal))
					}
					if effectMod != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(ailment effect modifier)", effectMod))
					}
					if rateMod != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(burn rate modifier)", rateMod))
					}
					if activeIgnites != 1 {
						breakdown.AddLine(key, luaSprintf("x %.2f ^8(avg ignite stacks)", activeIgnites))
					}
					if effMult != 1 {
						breakdown.AddLine(key, luaSprintf("x %.3f ^8(effective DPS modifier from enemy debuffs)", effMult))
					}
					if output["IgniteDPS"] != igniteDPSUncapped {
						breakdown.AddLine(key,
							luaSprintf("= %.1f ^8(Uncapped raw Ignite DPS)", igniteDPSUncapped),
							luaSprintf("^8(Raw Ignite DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", igniteDPSUncapped-igniteDPSCapped, (igniteDPSUncapped-igniteDPSCapped)/igniteDPSCapped*100),
							luaSprintf("= %d ^8(Capped Ignite DPS)", luaInt(igniteDPSCapped)),
						)
					} else {
						breakdown.AddLine(key, luaSprintf("= %.1f ^8per second", output["IgniteDPS"]))
					}
					breakdown.AddLine(key, luaSprintf("%.2f%% of Maximum Ignite DPS", output["IgniteDPS"]/maxIgniteDPSCapped*100), "", "DPS Range:")
					if maxIgniteDPSCapped == maxIgniteDPSUncapped && minIgniteDPSCapped == minIgniteDPSUncapped {
						breakdown.AddLine(key, luaSprintf("%.0f to %.0f ^8(Ignite DPS Range)", minIgniteDPSUncapped, maxIgniteDPSUncapped))
					} else {
						breakdown.AddLine(key, luaSprintf("%.0f to %.0f ^8(Uncapped Ignite DPS Range)", minIgniteDPSUncapped, maxIgniteDPSUncapped))
					}
					if minIgniteDPSCapped != minIgniteDPSUncapped {
						breakdown.AddLine(key, luaSprintf("^8(Raw Min Ignite DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", minIgniteDPSUncapped-minIgniteDPSCapped, (minIgniteDPSUncapped-minIgniteDPSCapped)/minIgniteDPSCapped*100))
					}
					if maxIgniteDPSCapped != maxIgniteDPSUncapped {
						breakdown.AddLine(key, luaSprintf("^8(Raw Max Ignite DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", maxIgniteDPSUncapped-maxIgniteDPSCapped, (maxIgniteDPSUncapped-maxIgniteDPSCapped)/maxIgniteDPSCapped*100))
					}
					if maxIgniteDPSCapped != maxIgniteDPSUncapped || minIgniteDPSCapped != minIgniteDPSUncapped {
						breakdown.AddLine(key, luaSprintf("%.0f to %.0f ^8(Capped Ignite DPS Range)", minIgniteDPSCapped, maxIgniteDPSCapped))
					}
					if critMulti, ok := output["CritIgniteDotMulti"]; ok && critMulti != output["IgniteDotMulti"] {
						chanceFromHit := float64(output["IgniteChanceOnHit"] / 100 * (1 - globalOutput["CritChance"]/100))
						chanceFromCrit := float64(output["IgniteChanceOnCrit"] / 100 * ailmentCritChance / 100)
						totalFromHit := chanceFromHit / (chanceFromHit + chanceFromCrit)
						totalFromCrit := chanceFromCrit / (chanceFromHit + chanceFromCrit)
						breakdown.Data["IgniteDotMulti"] = &Entry{Lines: breakdownCritDot(output["IgniteDotMulti"], critMulti, totalFromHit, totalFromCrit)}
						output["IgniteDotMulti"] = float64(output["IgniteDotMulti"]*totalFromHit) + float64(critMulti*totalFromCrit)
					}
					if skillFlags[SkillFlagIgniteCanStack] {
						breakdown.Data["IgniteDamage"] = &Entry{}
						if isAttack {
							breakdown.AddLine("IgniteDamage", pass.Label+":")
						}
						breakdown.AddLine("IgniteDamage",
							luaSprintf("%.1f ^8(damage per second)", output["IgniteDPS"]),
							luaSprintf("x %.2fs ^8(ignite duration)", globalOutput["IgniteDuration"]),
							luaSprintf("= %.1f ^8damage per ignite stack", output["IgniteDamage"]),
						)
					}
					if globalOutput["IgniteDuration"] != data.IgniteDurationBase {
						key := "IgniteDuration"
						globalBreakdown.Data[key] = &Entry{Lines: []string{luaSprintf("%.2fs ^8(base duration)", durationBase)}}
						if durationMod != 1 {
							globalBreakdown.AddLine(key, luaSprintf("x %.2f ^8(duration modifier)", durationMod))
						}
						if rateMod != 1 {
							globalBreakdown.AddLine(key, luaSprintf("/ %.2f ^8(burn rate modifier)", rateMod))
						}
						if debuffDurationMult != 1 {
							globalBreakdown.AddLine(key, luaSprintf("/ %.2f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
						}
						globalBreakdown.AddLine(key, luaSprintf("= %.2fs", globalOutput["IgniteDuration"]))
					}
				}
			}
		}
		// Calculate non-damaging ailments effect and duration modifiers
		if activeSkill.SkillTypes[data.SkillTypeChillingArea] || activeSkill.SkillTypes[data.SkillTypeNonHitChill] {
			skillFlags["chill"] = true
			//pob:block:src/Modules/CalcOffence.lua:5380-5385
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
			//pob:block:src/Modules/CalcOffence.lua:5387-5396
			if breakdown != nil {
				breakdown.Data["DotChill"] = &Entry{}
				breakdown.MultiChain("DotChill", BMultiChain{
					Label: luaSprintf("Effect of Chill: ^8(capped at %d%%)", luaInt(chillMax)),
					Items: []BMultiChainItem{
						{Format: "%.2f ^8(increased/reduced effect of chill)", Value: 1 + incChill/100},
						{Format: "%.2f ^8(more/less effect of chill)", Value: moreChill},
					},
					Total: luaSprintf("= %.0f%%", output["ChillSourceEffect"]),
				})
			}
		}

		if (output["FreezeChanceOnHit"] + output["FreezeChanceOnCrit"]) > 0 {
			//pob:block:src/Modules/CalcOffence.lua:5400-5404
			if breakdown != nil {
				globalBreakdown.Data["FreezeDurationMod"] = &Entry{Lines: []string{
					luaSprintf("Ailment mode: %s ^8(can be changed in the Configuration tab)", utils.Ternary(ailmentMode == "CRIT", "Crits Only", "Average Damage")),
				}}
			}
			freezeHitDmg, freezeCritDmg := calcAverageSourceDamage(data.AilmentFreeze)
			baseVal := calcAilmentDamage("Freeze", output["CritChance"], freezeHitDmg, freezeCritDmg, false) * skillModList.More(cfg, "FreezeAsThoughDealing")
			if baseVal > 0 {
				skillFlags["freeze"] = true
				output["FreezeDurationMod"] = 1 + skillModList.Sum(mod.TypeIncrease, cfg, "EnemyFreezeDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration")/100 + enemyDB.Sum(mod.TypeIncrease, nil, "SelfFreezeDuration", "SelfElementalAilmentDuration", "SelfAilmentDuration", "HoarfrostFreezeDuration")/100
				//pob:block:src/Modules/CalcOffence.lua:5409-5412
				if breakdown != nil {
					breakdown.AddLine("FreezeDPS",
						luaSprintf("For freeze to apply for the minimum of 0.3 seconds, target must have no more than %.0f Ailment Threshold.", baseVal*20*output["FreezeDurationMod"]),
						"^8(Ailment Threshold is about equal to Life except on bosses where it is about half of their life)",
					)
				}
			}
		}

		for _, ailment := range nonDamagingAilmentList {
			name := string(ailment)
			if (output[name+"ChanceOnHit"] + output[name+"ChanceOnCrit"]) <= 0 {
				continue
			}
			//pob:block:src/Modules/CalcOffence.lua:5418-5422
			if breakdown != nil {
				globalBreakdown.Data[name+"EffectMod"] = &Entry{Lines: []string{
					luaSprintf("Ailment mode: %s ^8(can be changed in the Configuration tab)", utils.Ternary(ailmentMode == "CRIT", "Crits Only", "Average Damage")),
				}}
			}

			cfg.SkillCond["CriticalStrike"] = ailmentMode == "CRIT" && !modDB.Flag(nil, "AilmentsAreNeverFromCrit")

			hitDmg, critDmg := calcAverageSourceDamage(ailment)
			damage := calcAilmentDamage(name, output["CritChance"], hitDmg, critDmg, false) * skillModList.More(cfg, name+"AsThoughDealing")
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
			//pob:block:src/Modules/CalcOffence.lua:5494-5510
			if baseDuration := *data.NonDamagingAilments[ailment].Duration; breakdown != nil && output[name+"Duration"] != baseDuration {
				key := name + "Duration"
				breakdown.Data[key] = &Entry{}
				if isAttack {
					breakdown.AddLine(key, pass.Label+":")
				}
				breakdown.AddLine(key, luaSprintf("%.2fs ^8(base duration)", baseDuration))
				if incDur != 0 {
					breakdown.AddLine(key, luaSprintf("x %.2f ^8(increased/reduced duration)", 1+incDur/100))
				}
				if moreDur != 1 {
					breakdown.AddLine(key, luaSprintf("x %.2f ^8(more/less duration)", moreDur))
				}
				if debuffDurationMult != 1 {
					breakdown.AddLine(key, luaSprintf("/ %.2f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
				}
				breakdown.AddLine(key, luaSprintf("= %.2fs", output[name+"Duration"]))
			}
		}
		// Calculate knockback chance/distance
		//pob:src/Modules/CalcOffence.lua:5515-5524
		output["KnockbackChance"] = min(100, float64(output["KnockbackChanceOnHit"]*(1-output["CritChance"]/100))+output["KnockbackChanceOnCrit"]*output["CritChance"]/100+enemyDB.Sum(mod.TypeBase, nil, "SelfKnockbackChance"))
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
		//pob:block:src/Modules/CalcOffence.lua:5533-5563
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
		minimumStunDuration := stunBase * moreStunDur / (1 + incStunRecov/100)
		maximumStunDuration := minimumStunDuration
		output["EnemyStunDuration"] = minimumStunDuration
		if incStunDurCrit != 0 && output["CritChance"] != 0 {
			if output["CritChance"] == 100 {
				minimumStunDuration *= 1 + (incStunDur+incStunDurCrit)/100
				maximumStunDuration = minimumStunDuration
				output["EnemyStunDuration"] = minimumStunDuration
			} else {
				minimumStunDuration *= 1 + incStunDur/100
				maximumStunDuration *= 1 + (incStunDur+incStunDurCrit)/100
				output["EnemyStunDuration"] *= 1 + (incStunDur+incStunDurCrit*output["CritChance"]/100)/100
			}
		} else {
			minimumStunDuration *= 1 + incStunDur/100
			maximumStunDuration = minimumStunDuration
			output["EnemyStunDuration"] = minimumStunDuration
		}
		if chanceToDouble != 0 {
			if chanceToDouble == 100 {
				minimumStunDuration *= 2
			}
			maximumStunDuration *= 2
			output["EnemyStunDuration"] *= 1 + chanceToDouble/100
		}
		//pob:block:src/Modules/CalcOffence.lua:5564-5594
		if breakdown != nil && output["EnemyStunDuration"] != stunBase {
			key := "EnemyStunDuration"
			breakdown.Data[key] = &Entry{Lines: []string{luaSprintf("%.2fs ^8(base duration)", stunBase)}}
			if incStunDur != 0 || (incStunDurCrit != 0 && output["CritChance"] != 0) {
				breakdown.AddLine(key, luaSprintf("x %.2f ^8(increased/reduced stun duration)", 1+(incStunDur+incStunDurCrit*output["CritChance"]/100)/100))
			}
			if moreStunDur != 1 {
				breakdown.AddLine(key, luaSprintf("x %.2f ^8(more/less stun duration)", moreStunDur))
			}
			if chanceToDouble != 0 {
				breakdown.AddLine(key, luaSprintf("x %.2f ^8(chance to double stun duration)", 1+chanceToDouble/100))
			}
			if incStunRecov != 0 {
				breakdown.AddLine(key, luaSprintf("/ %.2f ^8(increased/reduced enemy stun recovery)", 1+incStunRecov/100))
			}
			breakdown.AddLine(key, luaSprintf("= %.2fs", output["EnemyStunDuration"]))
			if minimumStunDuration != maximumStunDuration {
				breakdown.AddLine(key, luaSprintf("(minimum: %.2fs, maximum: %.2fs)", minimumStunDuration, maximumStunDuration))
			}
			if enemyActionSpeed := CalcActionSpeedMod(actor.Enemy); enemyActionSpeed != 1 {
				breakdown.AddLine(key,
					luaSprintf("/ %.2f ^8(enemy action speed)", enemyActionSpeed),
					luaSprintf("= %.2fs (note that for effects that care about duration this is ignored)", output["EnemyStunDuration"]/enemyActionSpeed),
				)
				if minimumStunDuration != maximumStunDuration {
					breakdown.AddLine(key, luaSprintf("(minimum: %.2fs, maximum: %.2fs)", minimumStunDuration/enemyActionSpeed, maximumStunDuration/enemyActionSpeed))
				}
			}
		}
		if canDeal[data.DamageTypePhysical] && (output["ImpaleChance"]+output["ImpaleChanceOnCrit"]) > 0 {
			skillFlags[SkillFlagImpale] = true
			impaleCritChance := output["CritChance"] / 100
			impaleChance := float64(min(output["ImpaleChance"]/100, 1)*(1-impaleCritChance)) + float64(min(output["ImpaleChanceOnCrit"]/100, 1)*impaleCritChance)
			maxStacks := skillModList.Sum(mod.TypeBase, cfg, "ImpaleStacksMax") *
				(1 + skillModList.Sum(mod.TypeBase, cfg, "ImpaleAdditionalDurationChance")/100)
			configStacks := enemyDB.Sum(mod.TypeBase, cfg, "Multiplier:ImpaleStacks")
			impaleStacks := min(maxStacks, configStacks)

			baseStoredDamage := float64(data.ImpaleStoredDamageBase)
			//pob:src/Modules/CalcOffence.lua:5606
			storedExpectedDamageIncOnBleed := float64(skillModList.Sum(mod.TypeIncrease, cfg, "ImpaleEffectOnBleed") * min(skillModList.Sum(mod.TypeBase, cfg, "BleedChance")/100, 1))
			storedExpectedDamageInc := (skillModList.Sum(mod.TypeIncrease, cfg, "ImpaleEffect") + storedExpectedDamageIncOnBleed) / 100
			storedExpectedDamageMore := utils.RoundTo(skillModList.More(cfg, "ImpaleEffect"), 2)
			storedExpectedDamageModifier := (1 + storedExpectedDamageInc) * storedExpectedDamageMore
			impaleStoredDamage := baseStoredDamage * storedExpectedDamageModifier
			impaleHitDamageMod := impaleStoredDamage * impaleStacks // Source: https://www.reddit.com/r/pathofexile/comments/chgqqt/impale_and_armor_interaction/

			enemyArmour := max(calclib.Val(enemyDB, "Armour"), 0)
			impaleArmourReduction := CalcArmourReductionF(enemyArmour, impaleHitDamageMod*output["impaleStoredHitAvg"])
			impaleResist := min(max(0, enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction")+skillModList.Sum(mod.TypeBase, cfg, "EnemyImpalePhysicalDamageReduction")+impaleArmourReduction), data.EnemyPhysicalDamageReductionCap)
			//pob:block:src/Modules/CalcOffence.lua:5616-5618
			if skillModList.Flag(cfg, "IgnoreEnemyImpalePhysicalDamageReduction") {
				impaleResist = 0
			}

			impaleTakenCfg := &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagHit)}
			impaleTaken := (1 + enemyDB.Sum(mod.TypeIncrease, impaleTakenCfg, "DamageTaken", "PhysicalDamageTaken", "ReflectedDamageTaken")/100) *
				enemyDB.More(impaleTakenCfg, "DamageTaken", "PhysicalDamageTaken", "ReflectedDamageTaken")

			impaleDMGModifier := float64(impaleHitDamageMod * (1 - impaleResist/100) * impaleChance * impaleTaken)

			globalOutput["ImpaleStacksMax"] = maxStacks
			globalOutput["ImpaleStacks"] = impaleStacks
			// ImpaleStoredDamage should be named ImpaleEffect or similar
			// Using the variable name ImpaleEffect breaks the calculations sidebar (?!)
			output["ImpaleStoredDamage"] = impaleStoredDamage * 100
			output["ImpaleModifier"] = 1 + impaleDMGModifier

			//pob:block:src/Modules/CalcOffence.lua:5631-5638
			if breakdown != nil {
				breakdown.Data["ImpaleStoredDamage"] = &Entry{}
				breakdown.Data["ImpaleModifier"] = &Entry{}
				breakdown.AddLine("ImpaleStoredDamage", "10% ^8(base value)")
				breakdown.AddLine("ImpaleStoredDamage", luaSprintf("x %.2f ^8(increased effectiveness)", storedExpectedDamageModifier))
				breakdown.AddLine("ImpaleStoredDamage", luaSprintf("= %.1f%%", output["ImpaleStoredDamage"]))

				breakdown.AddLine("ImpaleModifier", luaSprintf("%d ^8(number of stacks, can be overridden in the Configuration tab)", luaInt(impaleStacks)))
				breakdown.AddLine("ImpaleModifier", luaSprintf("x %.3f ^8(stored damage)", impaleStoredDamage))
				breakdown.AddLine("ImpaleModifier", luaSprintf("x %.2f ^8(impale chance)", impaleChance))
				breakdown.AddLine("ImpaleModifier", luaSprintf("x %.2f ^8(impale enemy physical damage reduction)", 1-impaleResist/100))
				if impaleTaken != 1 {
					breakdown.AddLine("ImpaleModifier", luaSprintf("x %.2f ^8(impale enemy damage taken)", impaleTaken))
				}
				breakdown.AddLine("ImpaleModifier", luaSprintf("= %.3f ^8(impale damage multiplier)", impaleDMGModifier))
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
	//pob:src/Modules/CalcOffence.lua:3803-3830
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
			//pob:block:src/Modules/CalcOffence.lua:5718-5721
			if breakdown != nil && effMult != 1 {
				breakdown.EffMult("DecayEffMult", data.DamageTypeChaos, resist, 0, takenInc, effMult, takenMore, dotSourceRes(data.DamageTypeChaos), true, 0, nil)
			}
		}
		inc := skillModList.Sum(mod.TypeIncrease, dotCfg, "Damage", "ChaosDamage")
		more := skillModList.More(dotCfg, "Damage", "ChaosDamage")
		mult := skillModList.Sum(mod.TypeBase, dotCfg, "DotMultiplier") + skillModList.Sum(mod.TypeBase, dotCfg, "ChaosDotMultiplier")
		if override := skillModList.Override(dotCfg, "DotMultiplier"); override != nil {
			mult = override.Float()
		}
		actor.Output["DecayDPS"] = skillData.Decay * (1 + inc/100) * more * (1 + mult/100) * effMult
		actor.Output["DecayDuration"] = 8 * debuffDurationMult
		//pob:block:src/Modules/CalcOffence.lua:5728-5740
		if breakdown != nil {
			breakdown.Dot("DecayDPS", skillData.Decay, inc, more, mult, 1, 1, effMult, actor.Output["DecayDPS"])
			if actor.Output["DecayDuration"] != 8 {
				breakdown.Data["DecayDuration"] = &Entry{Lines: []string{luaSprintf("%.2fs ^8(base duration)", 8.0)}}
				if debuffDurationMult != 1 {
					breakdown.AddLine("DecayDuration", luaSprintf("/ %.2f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
				}
				breakdown.AddLine("DecayDuration", luaSprintf("= %.2fs", actor.Output["DecayDuration"]))
			}
		}
	}

	//pob:block:src/Modules/CalcOffence.lua:5743-5769
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
	//pob:block:src/Modules/CalcOffence.lua:5772-5812
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
			//pob:block:src/Modules/CalcOffence.lua:5834-5837
			if breakdown != nil && effMult != 1 {
				breakdown.EffMult(string(damageType)+"DotEffMult", damageType, resist, 0, takenInc, effMult, takenMore, dotSourceRes(damageType), true, 0, nil)
			}
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

		total := float64(baseVal * (1 + inc/100) * more * (1 + mult/100) * auraMult * effMult)
		if actor.Output[string(damageType)+"Dot"] == 0 {
			actor.Output[string(damageType)+"Dot"] = total
			actor.Output["TotalDotInstance"] = min(actor.Output["TotalDotInstance"]+total, data.DotDpsCap)
		} else {
			actor.Output["TotalDotInstance"] = min(actor.Output["TotalDotInstance"]+total+actor.Output[string(damageType)+"Dot"], data.DotDpsCap)
		}
		//pob:block:src/Modules/CalcOffence.lua:5854-5857
		if breakdown != nil {
			breakdown.Dot(string(damageType)+"Dot", baseVal, inc, more, mult, 1, auraMult, effMult, total)
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
		//pob:block:src/Modules/CalcOffence.lua:5887-5906
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
		//pob:block:src/Modules/CalcOffence.lua:5911-5913
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

	//pob:block:src/Modules/CalcOffence.lua:4572-4612
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
		var timeType string
		switch {
		case utils.HasTrue(skillFlags, SkillFlagTrap) || utils.HasTrue(skillFlags, SkillFlagMine):
			preSpeed := utils.OrF(actor.Output["TrapThrowingSpeed"], actor.Output["MineLayingSpeed"])
			cooldown := utils.OrF(actor.Output["TrapCooldown"], actor.Output["Cooldown"])
			useSpeed = utils.Ternary(cooldown > 0, 1/cooldown, preSpeed)
			timeType = utils.Ternary(utils.HasTrue(skillFlags, SkillFlagTrap), "trap throwing", "mine laying")
		case utils.HasTrue(skillFlags, SkillFlagTotem):
			placement := actor.Output["TotemPlacementSpeed"]
			if cd := actor.Output["Cooldown"]; cd > 0 {
				useSpeed = utils.Ternary(placement > 0, placement, 1/cd) / repeats
			} else {
				useSpeed = placement / repeats
			}
			timeType = "totem placement"
		case skillModList.Flag(nil, "HasSeals") && skillModList.Flag(nil, "UseMaxUnleash") &&
			env.Player.MainSkill.SkillData.HitTimeOverride != 0:
			useSpeed = env.Player.MainSkill.SkillData.HitTimeOverride / repeats
			timeType = "full unleash"
		default:
			speed := actor.Output["Speed"]
			if cd := actor.Output["Cooldown"]; cd > 0 {
				useSpeed = utils.Ternary(speed > 0, speed, 1/cd) / repeats
			} else {
				useSpeed = speed / repeats
			}
			switch {
			case skillData.Triggered:
				timeType = "trigger"
			case utils.HasTrue(skillFlags, SkillFlagAttack):
				timeType = "attack"
			default:
				timeType = "cast"
			}
		}

		hasCost[usedResource+"PerSecond"] = true
		actor.Output[usedResource+"PerSecondHasCost"] = 1
		actor.Output[usedResource+"PerSecondCost"] += float64(actor.Output[resource+"Cost"] * useSpeed)

		//pob:block:src/Modules/CalcOffence.lua:5950-5955
		if breakdown != nil {
			var lines []string
			if e := breakdown.Data[resource+"Cost"]; e != nil && len(e.Lines) > 0 {
				lines = append(lines, e.Lines[:len(e.Lines)-1]...)
			}
			lines = append(lines,
				luaSprintf("x %.2f ^8("+timeType+" speed)", useSpeed),
				luaSprintf("= %.2f per second", actor.Output[usedResource+"PerSecondCost"]),
			)
			breakdown.Data[usedResource+"PerSecondCost"] = &Entry{Lines: lines}
		}
	}
	//pob:block:src/Modules/CalcOffence.lua:5959-6104
	{
		selfHitSum := float64(0)
		selfHitAny := false
		add := func(dmgType string, dmg float64, mult float64) {
			_, taken := ApplyDmgTakenConversion(skillModList, actor.Output, breakdown,
				data.DamageType(selfHitDamageTypeName(dmgType)), dmg)
			selfHitSum += float64(taken * mult)
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
	//pob:block:src/Modules/CalcOffence.lua:6138-6140
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

	//pob:block:src/Modules/CalcOffence.lua:4786-4820
	if mirage := activeSkill.Mirage; mirage != nil && mirage.Output != nil && hasMirageTotalDPS(mirage.Output) {
		mirageCount := mirage.Count
		if mirageCount == 0 {
			mirageCount = 1
		}
		actor.Output["MirageDPS"] = mirage.Output["TotalDPS"] * mirageCount
		actor.Output["CombinedDPS"] += float64(mirage.Output["TotalDPS"] * mirageCount)

		if mirage.Output["IgniteDPS"] > actor.Output["IgniteDPS"] {
			actor.Output["MirageDPS"] += mirage.Output["IgniteDPS"]
			actor.Output["IgniteDPS"] = 0
		}
		if mirage.Output["BleedDPS"] > actor.Output["BleedDPS"] {
			actor.Output["MirageDPS"] += mirage.Output["BleedDPS"]
			actor.Output["BleedDPS"] = 0
		}

		if mirage.Output["PoisonDPS"] != 0 {
			actor.Output["MirageDPS"] += float64(mirage.Output["PoisonDPS"] * mirageCount)
			actor.Output["CombinedDPS"] += float64(mirage.Output["PoisonDPS"] * mirageCount)
		}
		if mirage.Output["ImpaleDPS"] != 0 {
			actor.Output["MirageDPS"] += float64(mirage.Output["ImpaleDPS"] * mirageCount)
			actor.Output["CombinedDPS"] += float64(mirage.Output["ImpaleDPS"] * mirageCount)
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
			actor.Output["MirageDPS"] += float64(mirage.Output["TotalDot"] * dotCount)
			actor.Output["CombinedDPS"] += float64(mirage.Output["TotalDot"] * dotCount)
		}
		if mirage.Output["CullMultiplier"] > 1 {
			bestCull = mirage.Output["CullMultiplier"]
		}
	}

	//pob:block:src/Modules/CalcOffence.lua:6250-6251
	totalDotDPS := (actor.Output["TotalDot"]) + (actor.Output["TotalPoisonDPS"]) +
		(actor.Output["CausticGroundDPS"]) +
		utils.OrDefault(actor.Output["TotalIgniteDPS"], actor.Output["IgniteDPS"]) +
		(actor.Output["BurningGroundDPS"]) +
		(actor.Output["BleedDPS"]) + (actor.Output["CorruptingBloodDPS"]) + (actor.Output["DecayDPS"])
	actor.Output["TotalDotDPS"] = min(totalDotDPS, data.DotDpsCap)
	//pob:block:src/Modules/CalcOffence.lua:6252-6254
	if actor.Output["TotalDotDPS"] != totalDotDPS {
		actor.Output["showTotalDotDPS"] = 1
	}
	//pob:block:src/Modules/CalcOffence.lua:6255-6257
	if !skillData.ShowAverage {
		actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + actor.Output["TotalDotDPS"]
	}
	//pob:block:src/Modules/CalcOffence.lua:6151-6163
	if skillFlags[SkillFlagImpale] {
		dpsMultiplier := utils.OrDefault(skillData.DpsMultiplier, 1)
		var mainHandImpaleDPS, offHandImpaleDPS float64
		splitHands := skillFlags[SkillFlagAttack] && skillData.DoubleHitsWhenDualWielding && skillFlags[SkillFlagBothWeaponAttack]
		if splitHands {
			mh, oh := actor.OutputTable[OutTableMainHand], actor.OutputTable[OutTableOffHand]
			mainHandImpaleDPS = float64(mh["impaleStoredHitAvg"] * (utils.OrDefault(mh["ImpaleModifier"], 1) - 1) * mh["HitChance"] / 100 * dpsMultiplier)
			offHandImpaleDPS = float64(oh["impaleStoredHitAvg"] * (utils.OrDefault(oh["ImpaleModifier"], 1) - 1) * oh["HitChance"] / 100 * dpsMultiplier)
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
		//pob:block:src/Modules/CalcOffence.lua:6176-6208
		if breakdown != nil {
			if splitHands {
				mh, oh := actor.OutputTable[OutTableMainHand], actor.OutputTable[OutTableOffHand]
				breakdown.AddLine("ImpaleDPS", "Main Hand:")
				breakdown.AddLine("ImpaleDPS", luaSprintf("%.2f ^8(MH average physical hit before mitigation)", mh["impaleStoredHitAvg"]))
				breakdown.AddLine("ImpaleDPS", luaSprintf("x %.2f ^8(MH chance to hit)", mh["HitChance"]/100))
				breakdown.AddLine("ImpaleDPS", luaSprintf("x %.2f ^8(MH impale damage multiplier)\n", utils.OrDefault(mh["ImpaleModifier"], 1)-1))
				breakdown.AddLine("ImpaleDPS", luaSprintf("= %.2f", mainHandImpaleDPS))
				breakdown.AddLine("ImpaleDPS", "Off Hand:")
				breakdown.AddLine("ImpaleDPS", luaSprintf("%.2f ^8(OH average physical hit before mitigation)", oh["impaleStoredHitAvg"]))
				breakdown.AddLine("ImpaleDPS", luaSprintf("x %.2f ^8(OH chance to hit)", oh["HitChance"]/100))
				breakdown.AddLine("ImpaleDPS", luaSprintf("x %.2f ^8(OH impale damage multiplier)", utils.OrDefault(oh["ImpaleModifier"], 1)-1))
				breakdown.AddLine("ImpaleDPS", luaSprintf("= %.2f", offHandImpaleDPS))
				breakdown.AddLine("ImpaleDPS", "Combined total:")
				breakdown.AddLine("ImpaleDPS", luaSprintf("%.2f + %.2f", mainHandImpaleDPS, offHandImpaleDPS))
			} else {
				breakdown.AddLine("ImpaleDPS", luaSprintf("%.2f ^8(average physical hit before mitigation)", actor.Output["impaleStoredHitAvg"]))
				breakdown.AddLine("ImpaleDPS", luaSprintf("x %.2f ^8(chance to hit)", actor.Output["HitChance"]/100))
				breakdown.AddLine("ImpaleDPS", luaSprintf("x %.2f ^8(impale damage multiplier)", utils.OrDefault(actor.Output["ImpaleModifier"], 1)-1))
			}
			if skillFlags[SkillFlagNotAverage] {
				breakdown.AddLine("ImpaleDPS", utils.Ternary(actor.Output["HitSpeed"] != 0, luaSprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"]), luaSprintf("x %.2f ^8(%s rate)", actor.Output["Speed"], utils.Ternary(skillFlags[SkillFlagAttack], "attack", "cast"))))
			}
			if skillData.DpsMultiplier != 0 {
				breakdown.AddLine("ImpaleDPS", luaSprintf("x %g ^8(dps multiplier for this skill)", skillData.DpsMultiplier))
			}
			if quantityMultiplier > 1 {
				breakdown.AddLine("ImpaleDPS", luaSprintf("x %g ^8(quantity multiplier for this skill)", quantityMultiplier))
			}
			if actor.Output["ImpaleDuration"] <= 0 {
				breakdown.AddLine("ImpaleDPS", "x 0 ^8(no Impale Duration)")
			}
			breakdown.AddLine("ImpaleDPS", luaSprintf("= %.1f", actor.Output["ImpaleDPS"]))
		}
	}

	bestCull = max(bestCull, actor.Output["CullMultiplier"])
	actor.Output["CullingDPS"] = actor.Output["CombinedDPS"] * (bestCull - 1)
	//pob:block:src/Modules/CalcOffence.lua:6261-6262
	actor.Output["ReservationDPS"] = actor.Output["CombinedDPS"] * (actor.Output["ReservationDpsMultiplier"] - 1)
	actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] * bestCull * actor.Output["ReservationDpsMultiplier"]
}

func hasMirageTotalDPS(output map[string]float64) bool {
	_, ok := output["TotalDPS"]
	return ok
}

//pob:block:src/Modules/CalcOffence.lua:3891
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

//pob:block:src/Modules/CalcOffence.lua:323-324
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

var offenceCostOrder = []string{
	"Mana", "Life", "ES", "Soul", "Rage", "ManaPercent", "LifePercent",
	"ManaPerMinute", "LifePerMinute", "ManaPercentPerMinute", "LifePercentPerMinute",
	"ESPerMinute", "ESPercentPerMinute",
}

type offenceCostKeyNames struct{ key, cost, costRaw, hasCost string }

var offenceCostNames = func() []struct {
	baseOverride, noMult, base, cost string
	upfront, perSecond               offenceCostKeyNames
} {
	out := make([]struct {
		baseOverride, noMult, base, cost string
		upfront, perSecond               offenceCostKeyNames
	}, len(offenceCostOrder))
	keyNames := func(key string) offenceCostKeyNames {
		return offenceCostKeyNames{key: key, cost: key + "Cost", costRaw: key + "Cost" + "Raw", hasCost: key + "HasCost"}
	}
	for i, resource := range offenceCostOrder {
		n := &out[i]
		n.baseOverride, n.noMult, n.base, n.cost = "Base"+resource+"CostOverride", resource+"CostNoMult", resource+"CostBase", resource+"Cost"
		n.upfront, n.perSecond = keyNames(resource), keyNames(strings.ReplaceAll(resource, "Minute", "Second"))
	}
	return out
}()

var damageTypeNameSets = func() map[data.DamageType]*struct {
	min, max, minBase, maxBase, selfMin, selfMax, added, bonusMin, bonusMax, critAverage, hitAverage string
	lifeLeech, energyShieldLeech, manaLeech                                                          []string
} {
	out := map[data.DamageType]*struct {
		min, max, minBase, maxBase, selfMin, selfMax, added, bonusMin, bonusMax, critAverage, hitAverage string
		lifeLeech, energyShieldLeech, manaLeech                                                          []string
	}{}
	for _, damageType := range data.DamageType("").Values() {
		t := string(damageType)
		out[damageType] = &struct {
			min, max, minBase, maxBase, selfMin, selfMax, added, bonusMin, bonusMax, critAverage, hitAverage string
			lifeLeech, energyShieldLeech, manaLeech                                                          []string
		}{t + "Min", t + "Max", t + "Min" + "Base", t + "Max" + "Base", "Self" + t + "Min", "Self" + t + "Max", "Added" + t + "Damage",
			t + "BonusMin", t + "BonusMax", t + "CritAverage", t + "HitAverage",
			[]string{"DamageLeech", "DamageLifeLeech", t + "DamageLifeLeech"}, []string{"DamageEnergyShieldLeech", t + "DamageEnergyShieldLeech"},
			[]string{"DamageLeech", "DamageManaLeech", t + "DamageManaLeech"}}
		if damageType.IsElemental() {
			n := out[damageType]
			n.lifeLeech = append(n.lifeLeech, "ElementalDamageLifeLeech")
			n.energyShieldLeech = append(n.energyShieldLeech, "ElementalDamageEnergyShieldLeech")
			n.manaLeech = append(n.manaLeech, "ElementalDamageManaLeech")
		}
	}
	return out
}()

var ailmentNameSets = func() map[data.Ailment]*struct {
	enemyChance, selfChance, avoid, immune, cannot, chanceOnHit, chanceOnCrit, critsDontAlways string
	inflict                                                                                    SkillFlag
} {
	out := map[data.Ailment]*struct {
		enemyChance, selfChance, avoid, immune, cannot, chanceOnHit, chanceOnCrit, critsDontAlways string
		inflict                                                                                    SkillFlag
	}{}
	for _, ailment := range data.Ailment("").Values() {
		a := string(ailment)
		out[ailment] = &struct {
			enemyChance, selfChance, avoid, immune, cannot, chanceOnHit, chanceOnCrit, critsDontAlways string
			inflict                                                                                    SkillFlag
		}{"Enemy" + a + "Chance", "Self" + a + "Chance", "Avoid" + a, a + "Immune", "Cannot" + a, a + "ChanceOnHit", a + "ChanceOnCrit",
			"CritsDontAlways" + a, SkillFlag("inflict" + a)}
	}
	return out
}()

//pob:block:src/Modules/CalcOffence.lua:35-63
var damageStatsForTypes = func() [0x20][]string {
	var out [0x20][]string
	for k := range out {
		out[k] = []string{"Damage"}
		for _, t := range []data.DamageType{data.DamageTypePhysical, data.DamageTypeLightning, data.DamageTypeCold, data.DamageTypeFire, data.DamageTypeElemental, data.DamageTypeChaos} {
			if k&data.DamageTypeFlags[t] != 0 {
				out[k] = append(out[k], string(t)+"Damage")
			}
		}
	}
	return out
}()
