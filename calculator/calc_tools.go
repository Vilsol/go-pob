package calculator

import (
	"log/slog"
	"math"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/moddb"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

//pob:src/Modules/CalcTools.lua:15-18
func CalcMod(list moddb.ModStoreFuncs, cfg *moddb.ListCfg, names ...string) float64 {
	return (1 + moddb.SumOf(list, mod.TypeIncrease, cfg, names...)/100) * moddb.MoreOf(list, cfg, names...)
}

//pob:src/Modules/CalcTools.lua:31-39
func CalcVal(modStore moddb.ModStoreFuncs, name string, cfg *moddb.ListCfg) float64 {
	baseVal := moddb.SumOf(modStore, mod.TypeBase, cfg, name)
	if baseVal != 0 {
		return baseVal * CalcMod(modStore, cfg, name)
	}
	return 0
}

func init() {
	moddb.GemIsType = CalcGemIsType
}

func gemIsSupport(gem *poe.SkillGem) bool {
	pobGem, ok := pobGemFor(gem)
	if !ok {
		return false
	}
	return pobGem.Tags["support"]
}

//pob:block:src/Classes/SkillsTab.lua:439-451
func pobGemForVariant(gem *poe.SkillGem, grantedEffectID string) (raw2.PoBGem, bool) {
	if grantedEffectID != "" {
		if row, ok := raw2.GemVariants[grantedEffectID]; ok {
			return row, true
		}
	}
	return pobGemFor(gem)
}

func pobGemFor(gem *poe.SkillGem) (raw2.PoBGem, bool) {
	if gem == nil {
		return raw2.PoBGem{}, false
	}
	base := gem.GetBaseItemType()
	if base == nil {
		return raw2.PoBGem{}, false
	}
	if pobGem, ok := raw2.Gems[base.ID]; ok {
		return pobGem, true
	}
	pobGem, ok := raw2.Gems[strings.Replace(base.ID, "/SupportGem", "/SkillGemSupport", 1)]
	return pobGem, ok
}

func primaryGrantedEffectOf(gemData *poe.SkillGem) *poe.GrantedEffect {
	effects := raw2.GrantedEffectsForGem(gemData)
	if len(effects) == 0 {
		return nil
	}
	return effects[0]
}

var vaalTransfiguredAlts = [...]string{"AltX", "AltY"}

//pob:block:src/Modules/Data.lua:1098-1119
func secondaryGrantedEffectOf(gemData *poe.SkillGem, grantedEffectID string, variantID *string) *poe.GrantedEffect {
	pobGem, ok := pobGemForVariant(gemData, grantedEffectID)
	if !ok || pobGem.SecondaryGrantedEffectID == "" {
		return nil
	}
	if gemData != nil && gemData.IsVaalGem && variantID != nil {
		for _, alt := range vaalTransfiguredAlts {
			if !strings.HasSuffix(*variantID, alt) {
				continue
			}
			if effect := poe.GrantedEffectByID(pobGem.SecondaryGrantedEffectID + alt); effect != nil {
				return effect
			}
		}
	}
	return poe.GrantedEffectByID(pobGem.SecondaryGrantedEffectID)
}

//pob:src/Modules/CalcTools.lua:154-167
func CalcGemIsTypeVariant(gem *poe.SkillGem, grantedEffectID string, t string) bool {
	t = strings.ToLower(t)
	if t == "all" {
		return true
	}
	if grantedEffectID != "" {
		if row, ok := vaalTransfiguredGemRow(gem, grantedEffectID); ok {
			return gemRowIsType(row, t, true)
		}
		if row, ok := raw2.GemVariants[grantedEffectID]; ok {
			return gemRowIsType(row, t, true)
		}
	}
	pobGem, ok := pobGemFor(gem)
	if !ok {
		return false
	}
	return gemRowIsType(pobGem, t, true)
}

//pob:block:src/Modules/Data.lua:1098-1117
func vaalTransfiguredGemRow(gem *poe.SkillGem, secondaryID string) (raw2.PoBGem, bool) {
	if gem == nil || !gem.IsVaalGem {
		return raw2.PoBGem{}, false
	}
	row, ok := pobGemFor(gem)
	if !ok || row.SecondaryGrantedEffectID == "" {
		return raw2.PoBGem{}, false
	}
	for _, alt := range vaalTransfiguredAlts {
		if row.SecondaryGrantedEffectID+alt != secondaryID {
			continue
		}
		if variant, ok := raw2.GemVariants[secondaryID]; ok {
			row.Name = "Vaal " + variant.Name
		}
		row.SecondaryGrantedEffectID = secondaryID
		return row, true
	}
	return raw2.PoBGem{}, false
}

func gemIDsAreSame(row raw2.PoBGem, typeName string) bool {
	if row.GrantedEffectID == "" {
		return false
	}
	mine := dropVaalGameID(row, raw2.GemGameIDForGrantedEffectID[row.GrantedEffectID])
	typeID := raw2.GemGameIDForName(typeName)
	if typeRow, ok := raw2.Gems[typeID]; ok {
		typeID = dropVaalGameID(typeRow, typeID)
	}
	return mine != "" && mine == typeID
}

//pob:block:src/Modules/CalcTools.lua:277-281
func dropVaalGameID(row raw2.PoBGem, gameID string) string {
	if row.Tags["vaal"] && row.SecondaryGrantedEffectID != "" {
		return raw2.GemGameIDForGrantedEffectID[row.SecondaryGrantedEffectID]
	}
	return gameID
}

//pob:src/Modules/CalcTools.lua:154-167
func CalcGemIsType(gem *poe.SkillGem, t string) bool {
	t = strings.ToLower(t)

	if t == "all" {
		return true
	}

	pobGem, ok := pobGemFor(gem)
	if !ok {
		return false
	}
	return gemRowIsType(pobGem, t, false)
}

func gemRowIsType(pobGem raw2.PoBGem, t string, includeTransfigured bool) bool {
	tags := pobGem.Tags

	switch t {
	case "elemental":
		return tags["fire"] || tags["cold"] || tags["lightning"]
	case "aoe":
		return tags["area"]
	case "trap or mine":
		return tags["trap"] || tags["mine"]
	case "active skill", "active_skill", "grants_active_skill", "skill":
		return tags["grants_active_skill"] && !tags["support"]
	case "non-vaal":
		return !tags["vaal"]
	case "non-exceptional":
		return !tags["exceptional"]
	}

	if strings.EqualFold(t, pobGem.Name) {
		return true
	}
	if strings.EqualFold(t, strings.TrimPrefix(strings.ToLower(pobGem.Name), "vaal ")) {
		return true
	}
	if includeTransfigured && gemIDsAreSame(pobGem, t) {
		return true
	}

	return tags[t]
}

//pob:block:src/Export/Scripts/skills.lua:181-184
func TypesToSkillTypes(in []*poe.ActiveSkillType) map[data.SkillType]bool {
	types := make(map[data.SkillType]bool)
	for _, skillTypeRaw := range in {
		types[data.SkillType(skillTypeRaw.ID)] = true
	}
	return types
}

func grantedEffectSkillTypes(grantedEffect *poe.GrantedEffect) map[data.SkillType]bool {
	if grantedEffect == nil {
		return make(map[data.SkillType]bool)
	}
	if names, ok := raw2.SkillTypes[grantedEffect.ID]; ok {
		out := make(map[data.SkillType]bool, len(names))
		for _, n := range names {
			out[data.SkillType(n)] = true
		}
		return out
	}
	if as := grantedEffect.GetActiveSkill(); as != nil {
		return TypesToSkillTypes(as.GetActiveSkillTypes())
	}
	return make(map[data.SkillType]bool)
}

func supportTypeExpression(pob map[string][]string, grantedEffect *poe.GrantedEffect, game []*poe.ActiveSkillType) []data.SkillType {
	if names, ok := pob[grantedEffect.ID]; ok {
		out := make([]data.SkillType, len(names))
		for i, n := range names {
			out[i] = data.SkillType(n)
		}
		return out
	}
	return data.RawToSkillTypes(game)
}

// local statType = 0.7
// local req = round( ( 20 + ( level - 3 ) * 3 ) * ( multi / 100 ) ^ 0.9 * statType )
//
//pob:src/Modules/CalcTools.lua:170-180
func getGemStatRequirement(level float64, isSupport bool, multi float64) float64 {
	if multi == 0 {
		return 0
	}
	statType := 0.7
	if isSupport {
		statType = 0.5
	}
	req := math.Floor(float64((20+float64((level-3)*3))*math.Pow(multi/100, 0.9)*statType) + 0.5)
	if req < 14 {
		return 0
	}
	return req
}

func naturalMaxLevelOf(gemData *poe.SkillGem) int {
	pobGem, ok := pobGemFor(gemData)
	if !ok {
		return 0
	}
	return pobGem.NaturalMaxLevel
}

func luaSequenceLength[V any](m map[int]V) int {
	n := 0
	for {
		if _, ok := m[n+1]; !ok {
			return n
		}
		n++
	}
}

//pob:src/Modules/CalcTools.lua:41-58
func CalcValidateGemLevel(gemInstance *GemEffect) {
	// local grantedEffect = gemInstance.grantedEffect or gemInstance.gemData.grantedEffect
	grantedEffect := gemInstance.GrantedEffect

	levels := raw2.GetCalculatedGrantedEffect(grantedEffect.Raw).GetCalculatedLevels()
	if _, ok := levels[gemInstance.Level]; !ok {
		// Try limiting to the level range of the skill
		gemInstance.Level = max(1, gemInstance.Level)
		if n := luaSequenceLength(levels); n > 0 {
			gemInstance.Level = min(n, gemInstance.Level)
		}
	}

	if _, ok := levels[gemInstance.Level]; !ok {
		if n := naturalMaxLevelOf(gemInstance.GemData); n > 0 {
			gemInstance.Level = n
		}
	}

	if _, ok := levels[gemInstance.Level]; !ok {
		// That failed, so just grab any level
		for l := range levels {
			slog.Warn(
				"failed to find level",
				slog.String("grantedEffect", grantedEffect.Raw.ID),
				slog.Int("gemLevel", gemInstance.Level),
				slog.Int("default", l),
			)
			gemInstance.Level = l
			break
		}
	}
}

//pob:src/Modules/CalcTools.lua:147-206
func CalcBuildSkillInstanceStats(skillInstance *GemEffect, grantedEffect *GrantedEffect) map[string]float64 {
	stats := make(map[string]float64)
	allQualities := grantedEffect.Raw.GetEffectQualityStats()

	if skillInstance.Quality > 0 && allQualities != nil {
		qualityID := 0
		switch skillInstance.QualityID {
		case "Alternate1":
			qualityID = 1
		case "Alternate2":
			qualityID = 2
		case "Alternate3":
			qualityID = 3
		}

		if qualityStats := allQualities[qualityID]; qualityStats != nil {
			for i, stat := range qualityStats.GetStats() {
				baseVal := float64(qualityStats.StatsValuesPermille[i]) / 1000

				stats[stat.ID] = stats[stat.ID] + utils.ModF(baseVal*float64(skillInstance.Quality))
			}
		}
	}

	calculatedGrantedEffect := raw2.GetCalculatedGrantedEffect(grantedEffect.Raw)
	levels := calculatedGrantedEffect.GetCalculatedLevels()
	level := levels[skillInstance.Level]
	if level == nil {
		level = &raw2.CalculatedLevel{}
	}

	var availableEffectiveness *float64

	//pob:block:src/Modules/CalcTools.lua:193
	actorLevel := float64(level.LevelRequirement)
	if skillInstance.ActorLevel != nil {
		actorLevel = float64(*skillInstance.ActorLevel)
	}

	for index, stat := range calculatedGrantedEffect.GetCalculatedStats() {
		// Static value used as default (assumes statInterpolation == 1)

		statValue := float64(1)

		if len(level.Values) > index {
			statValue = level.Values[index]
		}

		if level.StatInterpolation != nil {
			statInterpolation := 0
			if len(level.StatInterpolation) > index {
				statInterpolation = level.StatInterpolation[index]
			}

			if statInterpolation == 3 {
				// Effectiveness interpolation
				if availableEffectiveness == nil {
					baseEffectiveness, incrementalEffectiveness := float64(1), float64(0)
					if statSet := calculatedGrantedEffect.GetStatSet(); statSet != nil {
						incrementalEffectiveness = float64(float32(statSet.IncrementalEffectiveness))
						if statSet.BaseEffectiveness != 0 {
							baseEffectiveness = float64(float32(statSet.BaseEffectiveness))
						}
					}

					availableEffectiveness = utils.Ptr((3.885209 + float64(0.360246*(actorLevel-1))) * (baseEffectiveness) * math.Pow(1+(incrementalEffectiveness), actorLevel-1))
				}
				statValue = math.Round(*availableEffectiveness * level.Values[index])
			} else if statInterpolation == 2 {
				//pob:src/Modules/CalcTools.lua:209-215
				ordered := make([]int, 0, len(levels))
				for lvl := range levels {
					ordered = append(ordered, lvl)
				}
				sort.Ints(ordered)

				currentLevelIndex := -1
				for idx, lvl := range ordered {
					if lvl == skillInstance.Level {
						currentLevelIndex = idx
					}
				}

				switch {
				case len(ordered) == 0 || currentLevelIndex < 0:
				case len(ordered) > 1:
					nextIdx := min(currentLevelIndex+1, len(ordered)-1)
					next, prev := levels[ordered[nextIdx]], levels[ordered[max(nextIdx-1, 0)]]
					nextReq, prevReq := float64(next.LevelRequirement), float64(prev.LevelRequirement)
					nextStat, prevStat := next.Values[index], prev.Values[index]
					if nextReq == prevReq {
						statValue = math.Round(prevStat)
					} else {
						statValue = math.Round(prevStat + (nextStat-prevStat)*(actorLevel-prevReq)/(nextReq-prevReq))
					}
				default:
					statValue = math.Round(levels[ordered[currentLevelIndex]].Values[index])
				}
			}
		}

		stats[stat] = stats[stat] + statValue
	}

	for id, stat := range calculatedGrantedEffect.GetCalculatedConstantStats() {
		stats[id] = stats[id] + stat
	}

	return stats
}

// active skill
//
//pob:src/Modules/CalcTools.lua:84-152
func CalcCanGrantedEffectSupportActiveSkill(grantedEffect *GrantedEffect, activeSkill *ActiveSkill) bool {
	if activeSkill.ActiveEffect.GrantedEffect.Raw.CannotBeSupported {
		return false
	}

	if grantedEffect.Raw.SupportsGemsOnly && activeSkill.ActiveEffect.GemData == nil {
		return false
	}

	//pob:block:src/Modules/CalcTools.lua:93-96
	if grantedEffect.FromItem() && grantedEffect.IsSupport() &&
		(activeSkill.ActiveEffect.GrantedEffect.FromItem() || srcInstanceFromItem(activeSkill)) {
		return false
	}

	if grantedEffectIsTrigger(grantedEffect) && actorPlayer(activeSkill.Actor) != activeSkill.Actor {
		return false
	}

	//pob:block:src/Modules/CalcTools.lua:116-150
	if supportTypes := grantedEffect.WeaponTypes(); len(supportTypes) > 0 {
		activeTypeLookup := make(map[data.ItemClassName]bool)
		if activeSkill.ActiveEffect != nil && activeSkill.ActiveEffect.GrantedEffect != nil {
			for _, activeType := range activeSkill.ActiveEffect.GrantedEffect.WeaponTypes() {
				activeTypeLookup[activeType] = true
			}
		}

		if len(activeTypeLookup) == 0 {
			return false
		}

		actor := activeSkill.Actor
		if actor != nil && ((actor.WeaponData1 != nil && actor.WeaponData1.CountsAsAll1H) ||
			(actor.WeaponData2 != nil && actor.WeaponData2.CountsAsAll1H)) {
			for _, name := range []data.ItemClassName{data.Claw, data.Dagger, data.OneHandAxe,
				data.OneHandMace, data.OneHandSword} {
				activeTypeLookup[name] = true
			}
		}

		typeMatch := false
		for _, grantedType := range supportTypes {
			if activeTypeLookup[grantedType] {
				typeMatch = true
				break
			}
		}
		if !typeMatch {
			return false
		}
	}

	if activeSkill.SummonSkill != nil {
		return CalcCanGrantedEffectSupportActiveSkill(grantedEffect, activeSkill.SummonSkill)
	}

	exclude := supportTypeExpression(raw2.SkillExcludeSkillTypes, grantedEffect.Raw, grantedEffect.Raw.GetExcludeTypes())
	if len(exclude) > 0 && CalcDoesTypeExpressionMatch(exclude, activeSkill.SkillTypes, nil) {
		return false
	}

	require := supportTypeExpression(raw2.SkillRequireSkillTypes, grantedEffect.Raw, grantedEffect.Raw.GetSupportTypes())
	if len(require) == 0 {
		return true
	}

	var minionTypes map[data.SkillType]bool
	if !grantedEffect.Raw.IgnoreMinionTypes {
		minionTypes = activeSkill.MinionSkillTypes
	}

	return CalcDoesTypeExpressionMatch(require, activeSkill.SkillTypes, minionTypes)
}

func actorPlayer(actor *Actor) *Actor {
	if actor == nil || actor.Enemy == nil {
		return actor
	}
	return actor.Enemy.Player
}

//pob:src/Modules/CalcTools.lua:60-82
func CalcDoesTypeExpressionMatch(checkTypes []data.SkillType, skillTypes map[data.SkillType]bool, minionTypes map[data.SkillType]bool) bool {
	stack := make([]bool, 0)
	for _, skillType := range checkTypes {
		if skillType == data.SkillTypeOR {
			other := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			stack[len(stack)-1] = stack[len(stack)-1] || other
		} else if skillType == data.SkillTypeAND {
			other := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			stack[len(stack)-1] = stack[len(stack)-1] && other
		} else if skillType == data.SkillTypeNOT {
			stack[len(stack)-1] = !stack[len(stack)-1]
		} else {
			stack = append(stack, skillTypes[skillType] || (minionTypes != nil && minionTypes[skillType]))
		}
	}

	for _, val := range stack {
		if val {
			return true
		}
	}

	return false
}
