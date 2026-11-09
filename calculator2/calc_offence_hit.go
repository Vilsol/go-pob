package calculator2

import (
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// conversionEntry is one activeSkill.conversionTable[type] row: the share converted to each type
// plus mult, the unconverted share.
type conversionEntry struct {
	to   map[string]float64
	mult float64
}

// passSource is a damage pass's source: a copy of the weapon data for an attack pass, skillData for
// the skill pass.
type passSource struct {
	weapon    map[string]float64
	skillData map[string]any
	typ       string
}

func (s *passSource) get(key string) (float64, bool) {
	if s.weapon != nil {
		v, ok := s.weapon[key]
		return v, ok
	}
	return sdNum(s.skillData, key)
}

// num is `source[key] or 0`.
func (s *passSource) num(key string) float64 {
	v, _ := s.get(key)
	return v
}

func (s *passSource) set(key string, v float64) {
	if s.weapon != nil {
		s.weapon[key] = v
		return
	}
	s.skillData[key] = v
}

type offencePass struct {
	label     string
	source    *passSource
	cfg       *moddb.ListCfg
	output    map[string]float64
	breakdown *Breakdown
}

// offenceHitState is what the hit half leaves for the ailment half: the passes, the main/off hand
// outputs and the conversion and gain tables activeSkill carries.
type offenceHitState struct {
	passList        []*offencePass
	mainHand        map[string]float64
	offHand         map[string]float64
	conversionTable map[string]*conversionEntry
	gainTable       map[string]map[string]float64
	canDeal         map[string]bool
	isAttack        bool
	// quantityMultiplier and monsterLife are calcs.offence's locals of those names.
	quantityMultiplier float64
	monsterLife        float64
}

//pob2:src/Modules/CalcOffence.lua:70-103
func processDamageConversion(skillModList *moddb.ModList, skillCfg *moddb.ListCfg, fromType string, skill bool) (map[string]float64, float64) {
	total := 0.0
	totalConv := map[string]float64{}

	// Calculate conversion for this damage type
	for _, toType := range dmgTypeList {
		var conv float64
		if skill {
			conv = math.Max(skillModList.Sum(mod.TypeBase, skillCfg,
				"SkillDamageConvertTo"+toType,
				"Skill"+fromType+"DamageConvertTo"+toType), 0)
		} else {
			names := []string{"DamageConvertTo" + toType, fromType + "DamageConvertTo" + toType}
			if isElemental[fromType] {
				names = append(names, "ElementalDamageConvertTo"+toType)
			}
			if fromType != "Chaos" {
				names = append(names, "NonChaosDamageConvertTo"+toType)
			}
			conv = math.Max(skillModList.Sum(mod.TypeBase, skillCfg, names...), 0)
		}

		totalConv[toType] = conv / 100
		total += conv
	}

	// Scale if over 100%
	if total > 100 {
		factor := 100 / total
		for typ, val := range totalConv {
			totalConv[typ] = val * factor
		}
		total = 100
	}

	return totalConv, total
}

//pob2:src/Modules/CalcOffence.lua:141-156
func calcConvertedDamage(state *offenceHitState, output map[string]float64, damageType string) (float64, float64) {
	// Calculate conversions
	convertedMin, convertedMax := 0.0, 0.0
	for _, otherType := range dmgTypeList {
		convMult := state.conversionTable[otherType].to[damageType]
		if convMult > 0 {
			// Damage is being converted/gained from the other damage type
			convertedMin += float64(output[otherType+"MinBase"] * convMult)
			convertedMax += float64(output[otherType+"MaxBase"] * convMult)
		}
	}
	return convertedMin, convertedMax
}

//pob2:src/Modules/CalcOffence.lua:158-175
func calcGainedDamage(state *offenceHitState, output map[string]float64, damageType string) (float64, float64) {
	gainedMin, gainedMax := 0.0, 0.0
	for _, otherType := range dmgTypeList {
		baseMin := float64(output[otherType+"MinBase"] * state.conversionTable[otherType].mult)
		baseMax := float64(output[otherType+"MaxBase"] * state.conversionTable[otherType].mult)
		gainMult := state.gainTable[otherType][damageType]
		if gainMult > 0 {
			// Damage is being converted/gained from the other damage type
			convertedMin, convertedMax := calcConvertedDamage(state, output, otherType)
			gainedMin += float64((baseMin + convertedMin) * gainMult)
			gainedMax += float64((baseMax + convertedMax) * gainMult)
		}
	}
	return gainedMin, gainedMax
}

var dmgTypeFlagOrder = []string{"Physical", "Lightning", "Cold", "Fire", "Elemental", "Chaos"}

var dmgTypeFlags = map[string]int{
	"Physical":  0x01,
	"Lightning": 0x02,
	"Cold":      0x04,
	"Fire":      0x08,
	"Elemental": 0x0E,
	"Chaos":     0x10,
}

//pob2:src/Modules/CalcOffence.lua:54-64
func damageStatsForTypes(typeFlags int) []string {
	modNames := []string{"Damage"}
	for _, typ := range dmgTypeFlagOrder {
		if typeFlags&dmgTypeFlags[typ] != 0 {
			modNames = append(modNames, typ+"Damage")
		}
	}
	return modNames
}

// Calculate min/max damage for the given damage type
//
//pob2:src/Modules/CalcOffence.lua:178-223
func calcDamage(activeSkill *ActiveSkill, output map[string]float64, cfg *moddb.ListCfg, damageType string, typeFlags int) (float64, float64) {
	skillModList := activeSkill.SkillModList

	typeFlags |= dmgTypeFlags[damageType]

	addMin, addMax := 0.0, 0.0
	summedMin := output[damageType+"SummedMinBase"]
	summedMax := output[damageType+"SummedMaxBase"]

	if summedMin == 0 && summedMax == 0 {
		// No base damage for this type, don't need to calculate modifiers
		return addMin, addMax
	}

	// Combine modifiers
	modNames := damageStatsForTypes(typeFlags)
	inc := 1 + skillModList.Sum(mod.TypeIncrease, cfg, modNames...)/100
	more := skillModList.More(cfg, modNames...)
	moreMinDamage := skillModList.More(cfg, "Min"+damageType+"Damage")
	moreMaxDamage := skillModList.More(cfg, "Max"+damageType+"Damage")

	return round(float64(summedMin*inc*more*moreMinDamage) + addMin),
		round(float64(summedMax*inc*more*moreMaxDamage) + addMax)
}

//pob2:src/Modules/CalcOffence.lua:466-468
func calcPenResist(resist, minPen, pen float64) float64 {
	if resist > minPen {
		return math.Max(resist-pen, minPen)
	}
	return resist
}

//pob2:src/Modules/CalcOffence.lua:470-475
func calcHitResist(resist, minPen, pen float64, cannotElePenIgnore, ignoreNonNegativeEleRes bool) float64 {
	if !cannotElePenIgnore && ignoreNonNegativeEleRes && resist >= 0 {
		return 0
	}
	if cannotElePenIgnore {
		return resist
	}
	return calcPenResist(resist, minPen, pen)
}

// Determine base leech value according to resource (using function to avoid repetition)
//
//pob2:src/Modules/CalcOffence.lua:481-522
func getBaseLeech(resource, dmgType string, skillModList *moddb.ModList, cfg *moddb.ListCfg, enemyDB *moddb.ModDB) float64 {
	leech := 0.0
	if !skillModList.Flag(cfg, "Condition:No"+resource+"LeechFrom"+dmgType+"Damage") && !(isElemental[dmgType] && skillModList.Flag(cfg, "No"+resource+"LeechFromElementalDamage")) {
		// Check if converted physical leech (most PoE2 leech is physical only by default)
		var convertModName, convertFlag string
		if isElemental[dmgType] && skillModList.Flag(cfg, resource+"LeechBasedOnElementalDamage") {
			convertFlag = resource + "LeechBasedOnElementalDamage"
			convertModName = "ElementalDamage" + resource + "Leech"
		} else if skillModList.Flag(cfg, resource+"LeechBasedOn"+dmgType+"Damage") {
			convertFlag = resource + "LeechBasedOn" + dmgType + "Damage"
			convertModName = dmgType + "Damage" + resource + "Leech"
		}
		if convertModName != "" && convertFlag != "" {
			tempCfg := *cfg
			tempCfg.OverrideCond = map[string]bool{"No" + resource + "LeechFromPhysicalDamage": false} // Need to force Condition to `false`, to calculate original phys leech values
			physLeechMods := skillModList.Tabulate(mod.TypeBase, &tempCfg, "PhysicalDamage"+resource+"Leech")
			for _, entry := range physLeechMods {
				// Add new leech mods for that damage type with the same conditions, source, etc.
				var tags []mod.Tag
				for _, tag := range entry.Mod.Tags() {
					// Tags that specifically disable Physical Damage leech need to be removed
					if c, ok := tag.(*mod.ConditionTag); ok && c.Negative && len(c.VarList) == 1 && c.VarList[0] == "No"+resource+"LeechFromPhysicalDamage" {
						continue
					}
					tags = append(tags, tag)
				}
				newMod := copyMod(convertModName, entry.Mod.Type(), entry.Mod.Value().Float(), entry.Mod.GetSource(), entry.Mod.Flags(), entry.Mod.KeywordFlags(), tags)
				skillModList.ReplaceMod(newMod)
			}
		}
		names := []string{"Damage" + resource + "Leech", dmgType + "Damage" + resource + "Leech"}
		if isElemental[dmgType] {
			names = append(names, "ElementalDamage"+resource+"Leech")
		}
		leech = skillModList.Sum(mod.TypeBase, cfg, names...) + enemyDB.Sum(mod.TypeBase, cfg, "SelfDamage"+resource+"Leech")/100
	}
	// The Condition:No<resource>LeechFrom<type>Damage branch only tags mods for a later conversion,
	// which the mod kernel's Tabulate results do not expose for in-place edits; it is not ported.
	return leech
}

// Calculate hit chance
//
//pob2:src/Modules/CalcDefence.lua:33-39
func hitChance(evasion, accuracy float64, uncapped bool) float64 {
	if accuracy < 0 {
		return 5
	}
	rawChance := (accuracy * 1.25) / (accuracy + float64(evasion*0.3)) * 100
	if uncapped {
		return math.Max(round(rawChance), 5)
	}
	return math.Max(math.Min(round(rawChance), 100), 5)
}

// calcResistForType is calcs.offence's local of that name. Party members are not ported.
//
//pob2:src/Modules/CalcOffence.lua:686-699
func calcResistForType(env *Environment, enemyDB *moddb.ModDB, damageType string, cfg *moddb.ListCfg) float64 {
	resistOverride := enemyDB.Override(cfg, damageType+"Resist")
	var maxResist float64
	if enemyDB.Flag(nil, "DoNotChangeMaxResFromConfig") {
		maxResist = miscConstant("EnemyMaxResist")
	} else {
		configResist, ok := configInputNumber(env, "enemy"+damageType+"Resist")
		if !ok {
			configResist = miscConstant("EnemyMaxResist")
		}
		maxResist = math.Min(math.Max(configResist, miscConstant("EnemyMaxResist")), miscConstant("MaxResistCap"))
	}
	var resist float64
	if resistOverride != nil {
		resist = resistOverride.Float()
	} else if env.ModDB.Flag(nil, "Enemy"+damageType+"ResistEqualToYours") {
		resist = env.Player.Output[damageType+"Resist"]
	} else {
		names := []string{damageType + "Resist"}
		if isElemental[damageType] {
			names = append(names, "ElementalResist")
		}
		resist = enemyDB.Sum(mod.TypeBase, cfg, names...) * math.Max(calcLibMod(enemyDB, cfg, names...), 0)
	}
	return math.Max(math.Min(resist, maxResist), miscConstant("ResistFloor"))
}

// offenceConversionTables builds activeSkill.conversionTable and the empty gainTable.
//
//pob2:block:src/Modules/CalcOffence.lua:2425-2512
func offenceConversionTables(state *offenceHitState, skillModList *moddb.ModList, skillCfg *moddb.ListCfg) {
	// Cache global damage disabling flags
	state.canDeal = map[string]bool{}
	for _, damageType := range dmgTypeList {
		state.canDeal[damageType] = !skillModList.Flag(skillCfg, "DealNo"+damageType, "DealNoDamage")
	}

	// Calculate damage conversion percentages
	// Initialize conversion tables
	state.conversionTable = map[string]*conversionEntry{}
	state.gainTable = map[string]map[string]float64{}
	for _, typ := range dmgTypeList {
		state.conversionTable[typ] = &conversionEntry{to: map[string]float64{}}
		state.gainTable[typ] = map[string]float64{}
		for _, otherType := range dmgTypeList {
			state.conversionTable[typ].to[otherType] = 0
		}
	}

	// First step: Process skill conversion
	for _, damageType := range dmgTypeList {
		skillConv, skillTotal := processDamageConversion(skillModList, skillCfg, damageType, true)
		for toType, amount := range skillConv {
			state.conversionTable[damageType].to[toType] = amount
		}
		state.conversionTable[damageType].mult = 1 - math.Min(skillTotal/100, 1)
	}

	type tempConversion struct {
		mult, base float64
		conv       map[string]float64
	}
	// Second step: Process global conversion and gains
	for _, damageType := range dmgTypeList {
		entry := state.conversionTable[damageType]
		tempConversions := map[string]*tempConversion{}

		// Handle global conversion of unconverted damage first
		if entry.mult > 0 {
			globalConv, globalTotal := processDamageConversion(skillModList, skillCfg, damageType, false)
			if globalTotal > 0 {
				unconvertedMult := entry.mult
				tempConversions[damageType] = &tempConversion{
					mult: unconvertedMult * (1 - globalTotal/100),
					conv: map[string]float64{},
				}
				for globalToType, globalAmount := range globalConv {
					tempConversions[damageType].conv[globalToType] = unconvertedMult * globalAmount
				}
			}
		}

		// Process global conversion on skill-converted damage
		for _, toType := range dmgTypeList {
			amount := entry.to[toType]
			if amount > 0 {
				globalConv, globalTotal := processDamageConversion(skillModList, skillCfg, toType, false)
				if globalTotal > 0 {
					tempConversions[toType] = &tempConversion{
						base: amount * (1 - globalTotal/100),
						conv: map[string]float64{},
					}
					for globalToType, globalAmount := range globalConv {
						tempConversions[toType].conv[globalToType] = amount * globalAmount
					}
				}
			}
		}

		// Apply all conversions simultaneously
		// The Lua's single pairs() loop lets a later base write overwrite an earlier row's added
		// conversion, so its result depends on table order; every base is written before any addition.
		for _, fromType := range dmgTypeList {
			data, ok := tempConversions[fromType]
			if !ok {
				continue
			}
			if fromType == damageType {
				entry.mult = data.mult
			} else {
				entry.to[fromType] = data.base
			}
		}
		for _, fromType := range dmgTypeList {
			data, ok := tempConversions[fromType]
			if !ok {
				continue
			}
			for _, toType := range dmgTypeList {
				if amount, ok := data.conv[toType]; ok {
					entry.to[toType] += amount
				}
			}
		}

		// The game stores each final conversion destination as a whole percent.
		convertedTotal := 0.0
		for _, toType := range dmgTypeList {
			conversion := round(entry.to[toType] * 100)
			entry.to[toType] = conversion / 100
			convertedTotal += conversion
		}
		entry.mult = 1 - math.Min(convertedTotal/100, 1)
	}
}

// offencePasses configures the damage passes.
//
//pob2:block:src/Modules/CalcOffence.lua:2514-2598
func offencePasses(env *Environment, actor *Actor, activeSkill *ActiveSkill, skillFlags map[string]bool, state *offenceHitState) {
	output := actor.Output
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	if state.isAttack {
		state.mainHand = map[string]float64{}
		state.offHand = map[string]float64{}
		actor.OutputMainHand, actor.OutputOffHand = state.mainHand, state.offHand
		// output.PreciseTechnique is a boolean, which the output map does not hold.
		critOverride := skillModList.Override(skillCfg, "WeaponBaseCritChance")
		critChance, hasCritChance := sdNum(skillData, "CritChance")
		breakdown := actor.Breakdown
		if skillFlags["weapon1Attack"] {
			//pob2:block:src/Modules/CalcOffence.lua:2522-2524
			if breakdown != nil {
				breakdown.MainHand = newBreakdown(skillModList, state.mainHand, nil)
			}
			activeSkill.Weapon1Cfg.SkillStats = state.mainHand
			source := &passSource{weapon: maps.Clone(weaponData(env, actor, 1)), typ: actor.WeaponData1Type}
			// Unarmed override for Concoction skills
			if skillFlags["unarmed"] {
				source = &passSource{weapon: unarmedWeaponDataFor(env), typ: "None"}
			}
			if actor.WeaponData1FacebreakerItemDamage && !skillFlags["unarmed"] && activeSkill.ActiveEffect.GrantedEffect.WeaponTypes != nil && activeSkill.ActiveEffect.GrantedEffect.WeaponTypes["One Hand Mace"] {
				for _, damageType := range dmgTypeList {
					source.set(damageType+"Min", source.weapon["Facebreaker"+damageType+"Min"])
					source.set(damageType+"Max", source.weapon["Facebreaker"+damageType+"Max"])
				}
			}
			if critOverride != nil && source.typ != "" && source.typ != "None" {
				source.set("CritChance", critOverride.Float())
			}
			if hasCritChance {
				source.set("CritChance", critChance)
			}
			state.passList = append(state.passList, &offencePass{
				label:  "Main Hand",
				source: source,
				cfg:    activeSkill.Weapon1Cfg,
				output: state.mainHand,
			})
			if breakdown != nil {
				state.passList[len(state.passList)-1].breakdown = breakdown.MainHand
			}
		}
		if skillFlags["weapon2Attack"] {
			//pob2:block:src/Modules/CalcOffence.lua:2552-2554
			if breakdown != nil {
				breakdown.OffHand = newBreakdown(skillModList, state.offHand, nil)
			}
			activeSkill.Weapon2Cfg.SkillStats = state.offHand
			source := &passSource{weapon: maps.Clone(weaponData(env, actor, 2)), typ: actor.WeaponData2Type}
			// Unarmed override for Concoction skills
			if skillFlags["unarmed"] {
				source = &passSource{weapon: unarmedWeaponDataFor(env), typ: "None"}
			}
			if critOverride != nil && source.typ != "" && source.typ != "None" {
				source.set("CritChance", critOverride.Float())
			}
			if hasCritChance {
				source.set("CritChance", critChance)
			}
			for _, damageType := range []string{"Physical", "Fire", "Cold"} {
				setMin, okMin := sdNum(skillData, "setOffHand"+damageType+"Min")
				setMax, okMax := sdNum(skillData, "setOffHand"+damageType+"Max")
				if okMin && okMax {
					source.set(damageType+"Min", setMin)
					source.set(damageType+"Max", setMax)
				}
			}
			if attackTime, ok := sdNum(skillData, "attackTime"); ok {
				source.set("AttackRate", 1000/attackTime)
			}
			state.passList = append(state.passList, &offencePass{
				label:  "Off Hand",
				source: source,
				cfg:    activeSkill.Weapon2Cfg,
				output: state.offHand,
			})
			if breakdown != nil {
				state.passList[len(state.passList)-1].breakdown = breakdown.OffHand
			}
		}
	} else {
		state.passList = append(state.passList, &offencePass{
			label:     "Skill",
			source:    &passSource{skillData: skillData},
			cfg:       skillCfg,
			output:    output,
			breakdown: actor.Breakdown,
		})
	}
}

// combineStat combines stats from Main Hand and Off Hand according to the mode. extra is the
// varargs key the CHANCE modes weight by; the breakdowns are not ported.
//
//pob2:block:src/Modules/CalcOffence.lua:2600-2687
func (state *offenceHitState) combineStat(output map[string]float64, skillFlags map[string]bool, skillData map[string]any, stat, mode string, extra ...string) {
	mh, mhOK := state.mainHand[stat]
	oh, ohOK := state.offHand[stat]
	switch {
	case mode == "OR" || !skillFlags["bothWeaponAttack"]:
		switch {
		case mhOK:
			output[stat] = mh
		case ohOK:
			output[stat] = oh
		default:
			delete(output, stat)
		}
	case mode == "ADD":
		output[stat] = mh + oh
	case mode == "AVERAGE":
		output[stat] = (mh + oh) / 2
	case mode == "CRIT":
		if skillFlags["bothWeaponAttack"] && truthy(skillData["combinesHitsWhenDualWielding"]) {
			output[stat] = mh + oh - (mh * oh / 100)
		} else {
			output[stat] = (mh + oh) / 2
		}
	case mode == "HARMONICMEAN":
		if (mhOK && mh == 0) || (ohOK && oh == 0) {
			output[stat] = 0
		} else {
			output[stat] = 2 / ((1 / mh) + (1 / oh))
		}
	case mode == "CHANCE" || mode == "CHANCE_AILMENT":
		if mhOK && ohOK {
			mainChance := float64(state.mainHand[extra[0]] * state.mainHand["HitChance"])
			offChance := float64(state.offHand[extra[0]] * state.offHand["HitChance"])
			mainPortion := mainChance / (mainChance + offChance)
			offPortion := offChance / (mainChance + offChance)
			if mode == "CHANCE" {
				output[stat] = float64(mh*mainPortion) + float64(oh*offPortion)
			} else {
				maxInstance := math.Max(mh, oh)
				minInstance := math.Min(mh, oh)
				stackName := strings.Replace(stat, "DPS", "", -1) + "Stacks"
				stacks, ok := output[stackName]
				if !ok {
					stacks = 1
				}
				stacksMax, ok := output[stackName+"Max"]
				if !ok {
					stacksMax = 1
				}
				maxInstanceStacks := math.Min(1, stacks/stacksMax)
				output[stat] = float64(maxInstance*maxInstanceStacks) + float64(minInstance*(1-maxInstanceStacks))
			}
		} else if mhOK {
			output[stat] = mh
		} else if ohOK {
			output[stat] = oh
		} else {
			delete(output, stat)
		}
	case mode == "DPS":
		output[stat] = mh + oh
		if !truthy(skillData["combinesHitsWhenDualWielding"]) {
			output[stat] /= 2
		}
	}
}

// offenceHitSpeed calculates how often you hit (speed, accuracy, block, etc). The breakdowns are
// not ported.
//
//pob2:block:src/Modules/CalcOffence.lua:2689-3159
func offenceHitSpeed(env *Environment, actor *Actor, activeSkill *ActiveSkill, skillFlags map[string]bool, state *offenceHitState) {
	globalOutput := actor.Output
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	isAttack := state.isAttack
	tick := miscConstant("ServerTickRate")

	var storedMainHandAccuracy, storedMainHandAccuracyVsEnemy float64
	hasStoredMainHandAccuracy := false
	for _, pass := range state.passList {
		source := pass.source
		output := pass.output
		cfg := pass.cfg

		if v, ok := sdNum(skillData, "averageBurstHits"); ok {
			output["AverageBurstHits"] = v
		} else if repeats, ok := output["Repeats"]; ok && repeats > 1 {
			output["AverageBurstHits"] = repeats
		}

		// Calculate hit chance
		base := skillModList.Sum(mod.TypeBase, cfg, "Accuracy")
		baseVsEnemy := skillModList.Sum(mod.TypeBase, cfg, "Accuracy", "AccuracyVsEnemy")
		inc := skillModList.Sum(mod.TypeIncrease, cfg, "Accuracy")
		incVsEnemy := skillModList.Sum(mod.TypeIncrease, cfg, "Accuracy", "AccuracyVsEnemy")
		more := skillModList.More(cfg, "Accuracy")
		moreVsEnemy := skillModList.More(cfg, "Accuracy", "AccuracyVsEnemy")

		falloffStart, falloffEnd := miscConstant("AccuracyFalloffStart"), miscConstant("AccuracyFalloffEnd")
		enemyDistance := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:enemyDistance") / 10
		enemyDistanceCapped := math.Max(math.Min(enemyDistance*10, falloffEnd), falloffStart)
		modValue := math.Floor(miscConstant("MaxAccuracyRangePenalty") * calcLibMod(skillModList, cfg, "AccuracyPenalty"))
		accuracyPenalty := 1 - ((enemyDistanceCapped-falloffStart)/(falloffEnd-falloffStart))*modValue/100

		output["Accuracy"] = math.Max(0, math.Floor(base*(1+inc/100)*more))
		accuracyVsEnemy := math.Max(0, math.Floor(baseVsEnemy*(1+incVsEnemy/100)*moreVsEnemy))
		accuracyVsEnemyBase := accuracyVsEnemy
		noAccuracyDistancePenalty := modDB.Flag(cfg, "NoAccuracyDistancePenalty") // saving this in local variable as it gets used again later
		if !noAccuracyDistancePenalty {
			accuracyVsEnemy = math.Floor(accuracyVsEnemy * accuracyPenalty)
		}
		breakdown := pass.breakdown
		//pob2:block:src/Modules/CalcOffence.lua:2732-2751
		if breakdown != nil {
			baseLink := link("%g ^8(base)", base)
			lines, _ := breakdown.multiChain(nil, multiChainSpec{
				Base:  &baseLink,
				Links: []chainLink{link("%.2f ^8(increased/reduced)", 1+inc/100), link("%.2f ^8(more/less)", more)},
				Total: sFormat("= %g", output["Accuracy"]),
			})
			if output["Accuracy"] != accuracyVsEnemy {
				lines = append(lines, "")
				vsBase := link("%g ^8(base)", baseVsEnemy)
				lines, _ = breakdown.multiChain(lines, multiChainSpec{
					Label: "Effective Accuracy vs Enemy",
					Base:  &vsBase,
					Links: []chainLink{link("%.2f ^8(distance penalty)", accuracyPenalty), link("%.2f ^8(increased/reduced)", 1+incVsEnemy/100), link("%.2f ^8(more/less)", moreVsEnemy)},
					Total: sFormat("= %g", accuracyVsEnemy),
				})
			}
			breakdown.set("Accuracy", lines...)
		}
		if skillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.label == "Main Hand" {
			storedMainHandAccuracy = output["Accuracy"]
			storedMainHandAccuracyVsEnemy = accuracyVsEnemy
			hasStoredMainHandAccuracy = true
		} else if skillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.label == "Off Hand" && hasStoredMainHandAccuracy {
			output["Accuracy"] = storedMainHandAccuracy
			accuracyVsEnemy = storedMainHandAccuracyVsEnemy
			if breakdown != nil {
				breakdown.set("Accuracy", "Using Main Hand Accuracy due to Mastery: "+luaToString(output["Accuracy"]))
			}
		}
		if !isAttack {
			output["AccuracyHitChance"] = 100
		} else {
			enemyEvasion := math.Max(round(calcLibVal(enemyDB, "Evasion", nil)), 0)
			hitChanceMod := calcLibMod(skillModList, cfg, "HitChance")
			cannotBeEvaded := skillModList.Flag(cfg, "CannotBeEvaded") || truthy(skillData["cannotBeEvaded"]) || (env.ModeEffective && enemyDB.Flag(nil, "CannotEvade"))
			if cannotBeEvaded {
				output["AccuracyHitChance"] = 100
			} else {
				output["AccuracyHitChance"] = hitChance(enemyEvasion, accuracyVsEnemy, false) * hitChanceMod
			}
			// Accounting for mods that enable "Chance to hit with Attacks can exceed 100%"
			// `and calcs.hitChance(...)` is never below 5, so it never falsifies the flag.
			exceedsHitChance := skillModList.Flag(cfg, "Condition:HitChanceCanExceed100")
			if exceedsHitChance {
				output["AccuracyHitChanceUncapped"] = math.Max(hitChance(enemyEvasion, accuracyVsEnemy, true)*calcLibMod(skillModList, cfg, "HitChance"), output["AccuracyHitChance"]) // keep higher chance in case of "CannotBeEvaded"
			} else {
				delete(output, "AccuracyHitChanceUncapped")
			}
			handCondition := "MainHandAttack"
			if pass.label == "Off Hand" {
				handCondition = "OffHandAttack"
			}
			if exceedsHitChance && output["AccuracyHitChanceUncapped"]-100 > 0 {
				skillModList.NewMod("Multiplier:ExcessHitChance", mod.TypeBase, round(output["AccuracyHitChanceUncapped"]-100, 2), "HitChanceCanExceed100", mod.Condition(handCondition))
			}
			if breakdown != nil {
				accuracyHitChanceBreakdown(env, breakdown, output, enemyEvasion, enemyDistance, modValue, accuracyVsEnemy, accuracyVsEnemyBase, hitChanceMod, noAccuracyDistancePenalty, cannotBeEvaded, exceedsHitChance)
			}
		}
		//enemy block chance
		output["enemyBlockChance"] = math.Max(math.Min(enemyDB.Sum(mod.TypeBase, cfg, "BlockChance"), 100)-skillModList.Sum(mod.TypeBase, cfg, "reduceEnemyBlock"), 0)
		if enemyDB.Flag(nil, "CannotBlockAttacks") && isAttack {
			output["enemyBlockChance"] = 0
		}

		output["HitChance"] = output["AccuracyHitChance"] * (1 - output["enemyBlockChance"]/100)
		if output["enemyBlockChance"] > 0 && !isAttack {
			actor.setOutputFlag("enemyHasSpellBlock", true)
		}
		//pob2:block:src/Modules/CalcOffence.lua:2828-2841
		if breakdown != nil && output["enemyBlockChance"] > 0 {
			var lines []string
			if output["AccuracyHitChance"] < 100 {
				lines = append(lines, "Accuracy Hit Chance: "+luaToString(output["AccuracyHitChance"])+"%")
			}
			breakdown.set("HitChance", append(lines,
				"Enemy Block Chance: "+luaToString(output["enemyBlockChance"])+"%",
				"Approximate hit chance: "+luaToString(output["HitChance"])+"%",
			)...)
		}

		// Check Precise Technique Keystone condition per pass as MH/OH might have different values
		condName := strings.ReplaceAll(pass.label, " ", "") + "AccRatingHigherThanMaxLife"
		moddb.SetCondition(skillModList.ModStore, condName, output["Accuracy"] > env.Player.Output["Life"])

		// Calculate attack/cast speed
		castTime := activeSkill.ActiveEffect.GrantedEffect.CastTime
		_, hasCastTimeOverride := sdNum(skillData, "castTimeOverride")
		timeOverride, hasTimeOverride := sdNum(skillData, "timeOverride")
		triggerTime, hasTriggerTime := sdNum(skillData, "triggerTime")
		triggerRate, hasTriggerRate := sdNum(skillData, "triggerRate")
		switch {
		case activeSkill.ActiveEffect.GrantedEffect.HasCastTime && castTime == 0 && !hasCastTimeOverride && !truthy(skillData["triggered"]):
			output["Time"] = 0
			output["Speed"] = 0
		case hasTimeOverride:
			output["Time"] = timeOverride
			output["Speed"] = 1 / output["Time"]
		case truthy(skillData["fixedCastTime"]):
			output["Time"] = castTime
			output["Speed"] = 1 / output["Time"]
		case hasTriggerTime && truthy(skillData["triggered"]):
			activeSkillsLinked := skillModList.Sum(mod.TypeBase, cfg, "ActiveSkillsLinkedToTrigger")
			if activeSkillsLinked > 0 {
				output["Time"] = triggerTime / (1 + skillModList.Sum(mod.TypeIncrease, cfg, "CooldownRecovery")/100) * activeSkillsLinked
			} else {
				output["Time"] = triggerTime / (1 + skillModList.Sum(mod.TypeIncrease, cfg, "CooldownRecovery")/100)
			}
			output["TriggerTime"] = output["Time"]
			output["Speed"] = 1 / output["Time"]
		case hasTriggerRate && truthy(skillData["triggered"]):
			output["Time"] = 1 / triggerRate
			output["TriggerTime"] = output["Time"]
			output["Speed"] = triggerRate
			skillData["showAverage"] = false
		default:
			var baseTime float64
			if isAttack {
				if attackSpeedMultiplier, ok := sdNum(skillData, "attackSpeedMultiplier"); ok {
					if attackRate, ok := source.get("AttackRate"); ok {
						source.set("AttackRate", attackRate*(1+attackSpeedMultiplier/100))
					}
				}
				attackRate, ok := source.get("AttackRate")
				if !ok {
					attackRate = 1
				}
				switch {
				case truthy(skillData["castTimeOverridesAttackTime"]):
					// Skill is overriding weapon attack speed
					baseTime = castTime / (1 + source.num("AttackSpeedInc")/100)
				case calcLibMod(skillModList, skillCfg, "SkillAttackTime") > 0:
					baseTime = (1/attackRate + skillModList.Sum(mod.TypeBase, cfg, "Speed")) * calcLibMod(skillModList, skillCfg, "SkillAttackTime")
				default:
					baseTime = 1/attackRate + skillModList.Sum(mod.TypeBase, cfg, "Speed")
				}
			} else {
				ct, ok := sdNum(skillData, "castTimeOverride")
				if !ok {
					ct = castTime
					if !activeSkill.ActiveEffect.GrantedEffect.HasCastTime {
						ct = 1
					}
				}
				baseTime = ct + skillModList.Sum(mod.TypeBase, cfg, "Speed")
			}
			more := skillModList.More(cfg, "Speed")
			if repeats, ok := globalOutput["Repeats"]; ok {
				output["Repeats"] = repeats
			} else {
				output["Repeats"] = 1
			}

			//Calculates the max number of trauma stacks you can sustain
			if skillModList.Flag(nil, "HasTrauma") {
				effectiveAttackRateCap := float64(tick * output["Repeats"])
				duration := skillModList.Sum(mod.TypeBase, cfg, "TraumaDuration") * calcLibMod(skillModList, skillCfg, "Duration", "DamagingAilmentDuration")
				traumaPerAttack := 1 + math.Min(skillModList.Sum(mod.TypeBase, cfg, "ExtraTrauma"), 100)/100
				incAttackSpeedPerTrauma := skillModList.Sum(mod.TypeIncrease, skillCfg, "SpeedPerTrauma")
				// compute trauma using an exact form.
				configTrauma := skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:TraumaStacks")
				inc := skillModList.Sum(mod.TypeIncrease, cfg, "Speed") - float64(incAttackSpeedPerTrauma*configTrauma) // remove trauma attack speed added by config.
				attackSpeedBeforeInc := 1 / baseTime * globalOutput["ActionSpeedMod"] * more
				incAttackSpeedPerTraumaCap := (effectiveAttackRateCap - float64(attackSpeedBeforeInc*(1+inc/100))) / attackSpeedBeforeInc * 100
				traumaRateBeforeInc := traumaPerAttack * (output["HitChance"] / 100) * attackSpeedBeforeInc / output["Repeats"]
				trauma := traumaRateBeforeInc * (1 + inc/100) / (1/duration - traumaRateBeforeInc*incAttackSpeedPerTrauma/100)
				if trauma < 0 || incAttackSpeedPerTrauma*trauma > incAttackSpeedPerTraumaCap { // invalid long term trauma generation as maximum attack rate is once per tick.
					trauma = traumaPerAttack * (output["HitChance"] / 100) * effectiveAttackRateCap / output["Repeats"] * duration
				}
				if skillFlags["bothWeaponAttack"] { // halve trauma rate when dual wielding so pass 2 doesn't double your trauma rate
					trauma /= 2
				}
				skillModList.NewMod("Multiplier:SustainableTraumaStacks", mod.TypeBase, trauma, "Maximum Sustainable Trauma Stacks")
			}
			if skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:TraumaStacks") == 0 {
				skillModList.NewMod("Multiplier:TraumaStacks", mod.TypeBase, skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:SustainableTraumaStacks"), "Maximum Sustainable Trauma Stacks")
			}
			inc := skillModList.Sum(mod.TypeIncrease, cfg, "Speed")

			if skillFlags["warcry"] {
				output["Speed"] = 1 / output["WarcryCastTime"]
			} else {
				output["Speed"] = 1 / (baseTime/round((1+inc/100)*more, 2) + skillModList.Sum(mod.TypeBase, cfg, "TotalAttackTime") + skillModList.Sum(mod.TypeBase, cfg, "TotalCastTime"))
			}
			output["CastRate"] = output["Speed"]
			if skillFlags["selfCast"] {
				// Self-cast skill; apply action speed
				output["Speed"] *= globalOutput["ActionSpeedMod"]
				output["CastRate"] = output["Speed"]
			}
			if channelTimeMultiplier, ok := sdNum(skillData, "channelTimeMultiplier"); ok {
				minTime, _ := sdNum(skillData, "minChannelTime")
				channelTime := output["Speed"]
				if channelTimeOverride, ok := sdNum(skillData, "channelTimeOverride"); ok {
					channelTime = 1 / channelTimeOverride
				}
				output["ChannelTime"] = math.Max(channelTimeMultiplier/channelTime, minTime)
				output["ChannelSpeed"] = output["Speed"]
			}
			if skillFlags["totem"] {
				// Totem skill. Apply action speed
				totemActionSpeed := 1 + (modDB.Sum(mod.TypeIncrease, nil, "TotemActionSpeed") / 100)
				output["TotemActionSpeed"] = totemActionSpeed
				output["Speed"] *= totemActionSpeed
				output["CastRate"] = output["Speed"]
			}
			if cooldown, ok := globalOutput["Cooldown"]; ok {
				output["Cooldown"] = cooldown
				if !skillModList.Flag(skillCfg, "CooldownDoesNotLimitSkillSpeed") {
					output["Speed"] = math.Min(output["Speed"], 1/output["Cooldown"]*output["Repeats"])
				}
			}
			_, hasCooldown := output["Cooldown"]
			_, hasMaxHitRate := sdNum(skillData, "maxHitRatePerEnemy")
			_, hasHitTimeOverride := sdNum(skillData, "hitTimeOverride")
			if hasCooldown && skillFlags["selfCast"] || hasMaxHitRate || hasHitTimeOverride {
				skillFlags["notAverage"] = true
				skillFlags["showAverage"] = false
				skillData["showAverage"] = false
			}
			if !activeSkill.SkillTypes["Channel"] {
				output["Speed"] = math.Min(output["Speed"], tick*output["Repeats"])
			}
			// Crossbows: Adjust attack speed values for Crossbow skills that need to reload
			if reloadTime, ok := sdNum(skillData, "reloadTime"); ok && reloadTime > 0 {
				output["FiringRate"] = output["Speed"]
				output["BoltCount"], _ = sdNum(skillData, "boltCount")
				output["EffectiveBoltCount"] = output["BoltCount"]
				output["ChanceToNotConsumeAmmo"] = activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "ChanceToNotConsumeAmmo")
				output["ChanceToReloadInstantly"] = math.Min(activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "InstantReloadChance"), 100)
				if output["ChanceToNotConsumeAmmo"] > 0 {
					if output["ChanceToNotConsumeAmmo"] < 100 {
						output["EffectiveBoltCount"] = output["BoltCount"] / (1 - (output["ChanceToNotConsumeAmmo"] / 100))
					} else {
						output["EffectiveBoltCount"] = math.Inf(1) // apparently division by zero is handled as "infinite" just fine
					}
				}
				if output["ChanceToNotConsumeAmmo"] >= 100 {
					output["TotalFiringTime"] = 1 / output["FiringRate"] * 0
				} else {
					output["TotalFiringTime"] = 1 / output["FiringRate"] * output["EffectiveBoltCount"]
				}
				output["ReloadRate"] = 1 / reloadTime
				output["ReloadTime"] = reloadTime
				output["EffectiveReloadTime"] = output["ReloadTime"] * (1 - output["ChanceToReloadInstantly"]/100)
				if output["ChanceToNotConsumeAmmo"] >= 100 {
					output["Speed"] = output["FiringRate"]
				} else {
					output["Speed"] = 1 / ((output["TotalFiringTime"] + output["EffectiveReloadTime"]) / (output["EffectiveBoltCount"]))
				}

				// Average bolts reloaded past six second for purposes of calculating Fresh Clip support damage bonus
				configSource := mod.Source("Config")
				boltsReloaded := func(name string, seconds float64) float64 {
					if o := skillModList.Override(&moddb.ListCfg{Source: &configSource}, name); o != nil {
						return o.Float()
					}
					if output["ChanceToNotConsumeAmmo"] > 100 {
						return 0
					}
					return output["BoltCount"] * seconds / (output["TotalFiringTime"] + output["EffectiveReloadTime"]) // assume 0 bolts reloaded when none are consumed
				}
				boltsReloadedPastSixSeconds := boltsReloaded("Multiplier:BoltsReloadedPastSixSeconds", 6)
				boltsReloadedPastEightSeconds := boltsReloaded("Multiplier:BoltsReloadedPastEightSeconds", 8)
				if boltsReloadedPastSixSeconds > 0 {
					skillModList.ReplaceMod(newMod("Multiplier:BoltsReloadedPastSixSeconds", mod.TypeBase, boltsReloadedPastSixSeconds, mod.Source(activeSkill.ActiveEffect.GrantedEffect.Name)))
				}
				if boltsReloadedPastEightSeconds > 0 {
					skillModList.ReplaceMod(newMod("Multiplier:BoltsReloadedPastEightSeconds", mod.TypeBase, boltsReloadedPastEightSeconds, mod.Source(activeSkill.ActiveEffect.GrantedEffect.Name)))
				}
			}
			if output["Speed"] == 0 {
				output["Time"] = 0
			} else {
				output["Time"] = 1 / output["Speed"]
			}
			//pob2:block:src/Modules/CalcOffence.lua:3054-3122
			if breakdown != nil {
				actionSpeed := 1.0
				if v, ok := output["TotemActionSpeed"]; ok && skillFlags["totem"] {
					actionSpeed = v
				} else if v, ok := globalOutput["ActionSpeedMod"]; ok && skillFlags["selfCast"] {
					actionSpeed = v
				}
				speedLinks := []chainLink{link("%.2f ^8(increased/reduced)", 1+inc/100), link("%.2f ^8(more/less)", more), link("%.2f ^8(action speed modifier)", actionSpeed)}
				speedBase := link("%.2f ^8(base)", 1/baseTime)
				lines, _ := breakdown.multiChain(nil, multiChainSpec{Base: &speedBase, Links: speedLinks, Total: sFormat("= %.2f ^8casts per second", output["CastRate"])})
				sourceReload, hasSourceReload := source.get("ReloadTime")
				if _, ok := output["ReloadTime"]; ok && hasSourceReload {
					globalBreakdown := actor.Breakdown
					firing, _ := breakdown.multiChain(nil, multiChainSpec{Base: &speedBase, Links: speedLinks, Total: sFormat("= %.2f ^8bolts per second", output["FiringRate"])})
					globalBreakdown.set("FiringRate", firing...)
					globalBreakdown.set("TotalFiringTime",
						sFormat("  1.00s / %.2f ^8(firing rate)", output["FiringRate"]),
						sFormat("= %.2fs ^8(time per bolt)", 1/output["FiringRate"]),
						"\n",
						sFormat("  %.2fs ^8(time per bolt)", 1/output["FiringRate"]),
						sFormat("x %.2f ^8(eff. bolt count)", output["EffectiveBoltCount"]),
						sFormat("= %.2fs ^8(total firing time)", output["TotalFiringTime"]),
					)
					incReloadSpeed := skillModList.Sum(mod.TypeIncrease, skillCfg, "ReloadSpeed")
					moreReloadSpeed := skillModList.More(skillCfg, "ReloadSpeed")
					globalBreakdown.set("ReloadTime",
						sFormat("  1.00s / %.2f ^8(base reload time)", sourceReload),
						sFormat("= %.2f ^8(base reload rate)", 1/sourceReload),
						"\n",
						"^8Note: modifiers to attack speed also affect reload speed",
						sFormat("x %.2f ^8(increased/reduced)", 1+(incReloadSpeed/100)+(inc/100)),
						sFormat("x %.2f ^8(more/less)", 1*moreReloadSpeed*more),
						sFormat("= %.2f ^8(reload rate)", output["ReloadRate"]),
						"\n",
						sFormat("   1.00s / %.2f ^8(reload rate)", output["ReloadRate"]),
						sFormat("= %.2fs ^8(reload time)", output["ReloadTime"]),
					)
					lines = []string{
						sFormat("  %.2fs ^8(total firing time)", output["TotalFiringTime"]),
						sFormat("+ %.2fs ^8(eff. reload time)", output["EffectiveReloadTime"]),
						sFormat("= %.2fs ^8(total attack time)", output["TotalFiringTime"]+output["EffectiveReloadTime"]),
						"\n",
						sFormat("  %.2fs ^8(total attack time)", output["TotalFiringTime"]+output["EffectiveReloadTime"]),
						sFormat("/ %.2f ^8(eff. bolt count)", output["EffectiveBoltCount"]),
						sFormat("= %.2fs ^8(eff. attack time)", 1/output["Speed"]),
						"\n",
						sFormat(" 1 / %.2fs ^8(eff. attack time)", 1/output["Speed"]),
						sFormat("= %.2f ^8(eff. attack rate)", output["Speed"]),
					}
				}
				if cooldown, ok := output["Cooldown"]; ok && 1/cooldown < output["CastRate"] && !skillModList.Flag(skillCfg, "CooldownDoesNotLimitSkillSpeed") {
					lines = append(lines, "\n", sFormat("1 / %.2f ^8(skill cooldown)", cooldown))
					if output["Repeats"] > 1 {
						lines = append(lines, sFormat("x %d ^8(repeat count)", output["Repeats"]))
					}
					lines = append(lines, sFormat("= %.2f ^8(casts per second)", output["Repeats"]/cooldown), "\n", sFormat("= %.2f ^8(lower of cast rates)", output["Speed"]))
				}
				breakdown.set("Speed", lines...)
			}
			//pob2:block:src/Modules/CalcOffence.lua:3123-3130
			if attackTimeMod := calcLibMod(skillModList, skillCfg, "SkillAttackTime"); breakdown != nil && attackTimeMod > 0 {
				timeBase := link("%.2f ^8(base)", 1/(output["Speed"]*attackTimeMod))
				lines, _ := breakdown.multiChain(nil, multiChainSpec{Base: &timeBase, Links: []chainLink{link("%.2f ^8(total modifier)", attackTimeMod)}, Total: sFormat("= %.2f ^8seconds per attack", output["Time"])})
				breakdown.set("Time", lines...)
			}
		}
		hitTimeOverride, hasHitTimeOverride := sdNum(skillData, "hitTimeOverride")
		maxHitRatePerEnemy, hasMaxHitRate := sdNum(skillData, "maxHitRatePerEnemy")
		hitTimeMultiplier, hasHitTimeMultiplier := sdNum(skillData, "hitTimeMultiplier")
		_, hasTime := output["Time"]
		if hasHitTimeOverride && !truthy(skillData["triggeredOnDeath"]) || hasMaxHitRate {
			//checks if skill has a max hit rate per enemy and adjust hitTimeOverride
			if hasMaxHitRate && !hasHitTimeOverride {
				hitTimeOverride = maxHitRatePerEnemy
				skillData["hitTimeOverride"] = hitTimeOverride
			} else if hasMaxHitRate && hitTimeOverride < maxHitRatePerEnemy {
				hitTimeOverride = maxHitRatePerEnemy
				skillData["hitTimeOverride"] = hitTimeOverride
			}
			output["HitTime"] = hitTimeOverride
			output["HitSpeed"] = 1 / output["HitTime"]
			//Brands always have hitTimeOverride
			if skillFlags["brand"] && !skillModList.Flag(nil, "UnlimitedBrandDuration") {
				output["BrandTicks"] = math.Floor(output["Duration"] * output["HitSpeed"])
			}
		} else if hasHitTimeMultiplier && hasTime && !truthy(skillData["triggeredOnDeath"]) {
			output["HitTime"] = output["Time"] * hitTimeMultiplier
			if minChannelTime, ok := sdNum(skillData, "minChannelTime"); skillFlags["channelRelease"] && ok {
				output["HitTime"] = math.Max(output["HitTime"], minChannelTime)
			}
			cooldown, hasCooldown := output["Cooldown"]
			if hasCooldown && truthy(skillData["triggered"]) {
				output["HitSpeed"] = 1 / (math.Max(output["HitTime"], cooldown))
			} else if hasCooldown && !skillFlags["channelRelease"] {
				output["HitSpeed"] = 1 / (output["HitTime"] + cooldown)
			} else {
				output["HitSpeed"] = 1 / output["HitTime"]
			}
		}
	}
}

// offenceCombineSpeed combines the passes' hit chance and speed. The breakdowns are not ported.
//
//pob2:block:src/Modules/CalcOffence.lua:3160-3307
func offenceCombineSpeed(env *Environment, actor *Actor, activeSkill *ActiveSkill, skillFlags map[string]bool, state *offenceHitState) {
	output := actor.Output
	modDB := actor.ModDB
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData

	// Other Misc DPS multipliers (like custom source)
	repeatMode, _ := configListValue(env, "repeatMode")
	if repeatMode == "FINAL" || skillModList.Flag(nil, "OnlyFinalRepeat") {
		repeats, ok := output["Repeats"]
		if !ok {
			repeats = 1
		}
		skillModList.NewMod("DPS", mod.TypeMore, -100/repeats, "Only Final Repeat")
	}
	if skillModList.Flag(nil, "TriggeredBySnipe") {
		skillFlags["channelRelease"] = true
	}
	if skillModList.Flag(nil, "HasTrauma") {
		output["SustainableTrauma"] = skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:SustainableTraumaStacks")
	} else {
		delete(output, "SustainableTrauma")
	}
	//Mantra of Flames buff count
	// cfg is out of scope here, so the Lua reads the nil global.
	modDB.Multipliers["BuffOnSelf"] += skillModList.Sum(mod.TypeBase, nil, "Multiplier:TraumaStacks")
	modDB.Multipliers["BuffOnSelf"] += skillModList.Sum(mod.TypeBase, nil, "Multiplier:VoltaxicWaitingStages")
	if !state.isAttack {
		return
	}
	// Combine hit chance and attack speed
	state.combineStat(output, skillFlags, skillData, "AccuracyHitChance", "AVERAGE")
	state.combineStat(output, skillFlags, skillData, "HitChance", "AVERAGE")
	state.combineStat(output, skillFlags, skillData, "AccuracyHitChanceUncapped", "AVERAGE")
	state.combineStat(output, skillFlags, skillData, "Speed", "HARMONICMEAN")
	state.combineStat(output, skillFlags, skillData, "HitSpeed", "OR")
	state.combineStat(output, skillFlags, skillData, "HitTime", "OR")
	if output["Speed"] == 0 {
		output["Time"] = 0
	} else {
		output["Time"] = 1 / output["Speed"]
	}

	if output["Time"] > 1 {
		modDB.AddMod(newFlagMod("Condition:OneSecondAttackTime", ""))
	}
	breakdown := actor.Breakdown
	if skillModList.Flag(nil, "UseOffhandAttackSpeed") {
		output["Speed"] = state.offHand["Speed"]
		output["Time"] = state.offHand["Time"]
		//pob2:block:src/Modules/CalcOffence.lua:3194-3199
		if breakdown != nil {
			breakdown.set("Speed", "Use Offhand Weapon Attack Speed:", sFormat("= %.2f", output["Speed"]))
		}
	} else if skillFlags["bothWeaponAttack"] && breakdown != nil {
		//pob2:block:src/Modules/CalcOffence.lua:3201-3222
		mh, oh := state.mainHand["Speed"], state.offHand["Speed"]
		switch {
		case truthy(skillData["combinesHitsWhenDualWielding"]):
			breakdown.set("Speed", "Combined hit from both weapons:", sFormat("1 / (1 / %.2f + 1 / %.2f)", mh, oh), sFormat("= %.2f", output["Speed"]))
		case truthy(skillData["doubleHitsWhenDualWielding"]):
			breakdown.set("Speed", "Simultaneous hits from each weapon:", sFormat("2 / (1 / %.2f + 1 / %.2f)", mh, oh), sFormat("%.2f ^8(hits twice per attack)", output["Speed"]), sFormat("= %.2f", output["Speed"]))
		default:
			breakdown.set("Speed", "Alternating both weapons:", sFormat("2 / (1 / %.2f + 1 / %.2f)", mh, oh), sFormat("= %.2f", output["Speed"]))
		}
	}
	if channelTimeMultiplier, ok := sdNum(skillData, "channelTimeMultiplier"); ok {
		minTime, _ := sdNum(skillData, "minChannelTime")
		channelTime := output["Speed"]
		if channelTimeOverride, ok := sdNum(skillData, "channelTimeOverride"); ok {
			channelTime = 1 / channelTimeOverride
		}
		output["ChannelTime"] = math.Max(channelTimeMultiplier/channelTime, minTime)
		output["ChannelSpeed"] = output["Speed"]
	}
	//pob2:block:src/Modules/CalcOffence.lua:3233-3246
	if breakdown != nil {
		if skillFlags["bothWeaponAttack"] {
			breakdown.set("HitChance", "Both weapons:",
				sFormat("%.2f%% ^8(main hand)", state.mainHand["HitChance"]),
				sFormat("%.2f%% ^8(off hand)", state.offHand["HitChance"]),
				sFormat("= %.2f%% ^8(average)", output["HitChance"]),
			)
		} else {
			hand := breakdown.OffHand
			if skillFlags["weapon1Attack"] {
				hand = breakdown.MainHand
			}
			if hand == nil {
				hand = newBreakdown(skillModList, nil, nil)
			}
			if e := hand.Entries["HitChance"]; e != nil {
				breakdown.setEntry("HitChance", e)
			} else {
				breakdown.setEntry("HitChance", hand.Entries["AccuracyHitChance"])
			}
			if breakdown.Entries["Speed"] == nil {
				breakdown.setEntry("Speed", hand.Entries["Speed"])
			}
		}
	}
	hitTimeOverride, hasHitTimeOverride := sdNum(skillData, "hitTimeOverride")
	timeOverride, hasTimeOverride := sdNum(skillData, "timeOverride")
	hitTimeMultiplier, hasHitTimeMultiplier := sdNum(skillData, "hitTimeMultiplier")
	_, hasTime := output["Time"]
	triggeredOnDeath := truthy(skillData["triggeredOnDeath"])
	switch {
	case hasHitTimeOverride && !triggeredOnDeath:
		output["HitTime"] = hitTimeOverride
		output["HitSpeed"] = 1 / output["HitTime"]
	case hasTimeOverride && !triggeredOnDeath:
		output["Time"] = timeOverride
	case hasHitTimeMultiplier && hasTime && !triggeredOnDeath:
		output["HitTime"] = output["Time"] * hitTimeMultiplier
		if minChannelTime, ok := sdNum(skillData, "minChannelTime"); skillFlags["channelRelease"] && ok {
			output["HitTime"] = math.Max(output["HitTime"], minChannelTime)
		}
		cooldown, hasCooldown := output["Cooldown"]
		switch {
		case hasCooldown && truthy(skillData["triggered"]):
			output["HitSpeed"] = 1 / (math.Max(output["HitTime"], cooldown))
		case hasCooldown:
			output["HitSpeed"] = 1 / (output["HitTime"] + cooldown)
		default:
			output["HitSpeed"] = math.Min(1/output["HitTime"], miscConstant("ServerTickRate"))
		}
	}
}

// offenceMovementPenalty grabs the quantity multiplier and the movement speed while using the skill.
//
//pob2:block:src/Modules/CalcOffence.lua:3309-3338
func offenceMovementPenalty(actor *Actor, activeSkill *ActiveSkill) float64 {
	output := actor.Output
	skillModList, skillCfg := activeSkill.SkillModList, activeSkill.SkillCfg
	// Grab quantity multiplier
	quantityMultiplier := math.Max(activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "QuantityMultiplier"), 1)
	if quantityMultiplier > 1 {
		output["QuantityMultiplier"] = quantityMultiplier
	}

	if activeSkill.SkillTypes["UsableWhileMoving"] {
		inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "MovementSpeedPenalty")
		more := skillModList.More(skillCfg, "MovementSpeedPenalty")
		penaltyMod := (1 + inc/100) * more
		if o := skillModList.Override(skillCfg, "MovementSpeedPenalty"); o != nil {
			penaltyMod = o.Float()
		}
		penaltyMod = math.Max(0, penaltyMod)
		base := calcLibMod(skillModList, skillCfg, "SkillMovementSpeed")
		total := float64((1 - base) * penaltyMod)
		output["MovementSpeedWhileUsingSkill"] = (1 - total) * output["MovementSpeedMod"]
		output["MovementSpeedWhileUsingSkillPercent"] = (1 - total) * 100
	}
	return quantityMultiplier
}

// accuracyHitChanceBreakdown is the attack pass's breakdown.AccuracyHitChance: the hit chance at 2, 5 and 9 metres
// and at the configured distance.
//
//pob2:block:src/Modules/CalcOffence.lua:2719-2723,src/Modules/CalcOffence.lua:2778-2816
func accuracyHitChanceBreakdown(env *Environment, breakdown *Breakdown, output map[string]float64, enemyEvasion, enemyDistance, modValue, accuracyVsEnemy, accuracyVsEnemyBase, hitChanceMod float64, noAccuracyDistancePenalty, cannotBeEvaded, exceedsHitChance bool) {
	falloffStart, falloffEnd := miscConstant("AccuracyFalloffStart"), miscConstant("AccuracyFalloffEnd")
	distances := []float64{2, 5, 9}
	accuracyPenalty := func(distance float64) float64 {
		return 1 - ((float64(distance*10)-falloffStart)/(falloffEnd-falloffStart))*modValue/100
	}
	from := " ^8(can be overridden in the Configuration tab)"
	if _, ok := env.ConfigInput["enemyLevel"]; ok {
		from = " ^8(overridden from the Configuration tab"
	}
	lines := []string{
		"Enemy level: " + strconv.Itoa(env.EnemyLevel) + from,
		"Enemy evasion: " + luaToString(enemyEvasion),
		"",
		"Approximate hit chance at:",
	}
	hitChances := []float64{enemyDistance}
	for _, distance := range distances {
		last := hitChances[len(hitChances)-1]
		if distance < last {
			hitChances = slices.Insert(hitChances, len(hitChances)-1, distance)
		} else if distance > last {
			hitChances = append(hitChances, distance)
		}
	}
	distBuffer := []string{"     ", "   ", "  ", ""}
	chanceBuffer := []string{"    ", "  ", ""}
	buffer := func(buffers []string, s string) string {
		if n := len(s); n >= 1 && n <= len(buffers) {
			return buffers[n-1]
		}
		return ""
	}
	for _, distance := range hitChances {
		adjustedAccuracy := accuracyVsEnemy
		if distance != enemyDistance {
			penalty := 1.0
			if !noAccuracyDistancePenalty {
				penalty = accuracyPenalty(distance)
			}
			adjustedAccuracy = math.Floor(accuracyVsEnemyBase * penalty)
		}
		capped := 0.0
		switch {
		case distance == enemyDistance:
			capped = output["AccuracyHitChance"]
		case cannotBeEvaded:
			capped = 100
		default:
			capped = hitChance(enemyEvasion, adjustedAccuracy, false) * hitChanceMod
		}
		excessText := ""
		if exceedsHitChance {
			uncapped := math.Max(hitChance(enemyEvasion, adjustedAccuracy, true)*hitChanceMod, capped)
			if uncapped > 100 {
				excess := luaToString(uncapped - capped)
				excessText = " ^8(+" + buffer(chanceBuffer, excess+"x") + excess + "%)"
			}
		}
		config := ""
		if distance == enemyDistance {
			config = " ^8(current config)"
		}
		dist, chance := luaToString(distance), luaToString(capped)
		lines = append(lines, buffer(distBuffer, dist)+dist+"m: "+buffer(chanceBuffer, chance)+chance+"%"+excessText+config)
	}
	if exceedsHitChance {
		lines = append(lines, "", "^8Note: Your hit chance can exceed 100%.\nExcess values are shown as (+x%)", "")
	}
	breakdown.set("AccuracyHitChance", lines...)
}
