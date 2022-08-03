package calculator

import (
	"math"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

func CalcMod(list ModStoreFuncs, cfg *ListCfg, names ...string) float64 {
	return (1 + list.Sum(mod.TypeIncrease, cfg, names...)/100) * list.More(cfg, names...)
}

func CalcVal(modStore ModStoreFuncs, name string, cfg *ListCfg) float64 {
	baseVal := modStore.Sum(mod.TypeBase, cfg, name)
	if baseVal != 0 {
		return baseVal * CalcMod(modStore, cfg, name)
	}
	return 0
}

func CalcGemIsType(gem *raw.SkillGem, t string) bool {
	if t == "all" {
		return true
	}

	tags := gem.GetTags()
	if t == "elemental" && (utils.Has(tags, raw.TagFire) || utils.Has(tags, raw.TagCold) || utils.Has(tags, raw.TagLightning)) {
		return true
	}

	// TODO AOE
	//if t == "aoe" && utils.Has(tags, raw.TagArea) {
	//	return true
	//}

	// TODO Trap and Mine
	//if t == "trap or mine" && (utils.Has(tags, raw.TagTrap) || utils.Has(tags, raw.TagMine)) {
	//	return true
	//}

	// TODO Active Skill
	//if t == "active skill" && utils.Has(tags, raw.TagActiveSkill) {
	//	return true
	//}

	// TODO Name
	//if t == strings.ToLower(gem.Name) {
	//	return true
	//}

	_, ok := tags[raw.TagName(t)]
	return ok
}

func TypesToFlagsAndTypes(in []*raw.ActiveSkillType) (map[SkillFlag]bool, map[data.SkillType]bool) {
	flags := make(map[SkillFlag]bool)
	types := make(map[data.SkillType]bool)
	for _, skillTypeRaw := range in {
		skillType := data.SkillType(skillTypeRaw.ID)
		types[skillType] = true

		switch skillType {
		case data.SkillTypeBrand:
			flags[SkillFlagBrand] = true
		case data.SkillTypeHex:
			flags[SkillFlagHex] = true
			flags[SkillFlagCurse] = true
		case data.SkillTypeAppliesCurse:
			flags[SkillFlagCurse] = true
		case data.SkillTypeAttack:
			flags[SkillFlagAttack] = true
			flags[SkillFlagHit] = true
		case data.SkillTypeProjectile:
			flags[SkillFlagProjectile] = true
			flags[SkillFlagHit] = true
		case data.SkillTypeTrapped:
			flags[SkillFlagTrap] = true
		case data.SkillTypeTrappable:
			flags[SkillFlagTrap] = true
		case data.SkillTypeRemoteMined:
			flags[SkillFlagMine] = true
		case data.SkillTypeSummonsTotem:
			flags[SkillFlagTotem] = true
		case data.SkillTypeSpell:
			flags[SkillFlagSpell] = true
		case data.SkillTypeAreaSpell:
			flags[SkillFlagSpell] = true
			flags[SkillFlagArea] = true
		case data.SkillTypeMelee:
			flags[SkillFlagMelee] = true
		case data.SkillTypeMeleeSingleTarget:
			flags[SkillFlagMelee] = true
		case data.SkillTypeChains:
			flags[SkillFlagChaining] = true
		case data.SkillTypeArea:
			flags[SkillFlagArea] = true
		case data.SkillTypeDamage:
			flags[SkillFlagHit] = true
			// TODO SkillFlagCast
			//case data.SkillType...:
			//	flags[SkillFlagCast] = true
		}
	}
	return flags, types
}

// CalcValidateGemLevel Validates the level of the given gem
func CalcValidateGemLevel(gemInstance *ActiveEffect) {
	// TODO
	// local grantedEffect = gemInstance.grantedEffect or gemInstance.gemData.grantedEffect
	grantedEffect := gemInstance.GrantedEffect

	levels := grantedEffect.Raw.Levels()
	if _, ok := levels[gemInstance.Level]; !ok {
		// Try limiting to the level range of the skill
		gemInstance.Level = utils.Max(1, gemInstance.Level)
		if len(levels) > 0 {
			gemInstance.Level = utils.Min(len(levels)-1, gemInstance.Level)
		}
	}

	if _, ok := levels[gemInstance.Level]; !ok {
		if gemInstance.GemData != nil && gemInstance.GemData.DefaultLevel() > 0 {
			gemInstance.Level = gemInstance.GemData.DefaultLevel()
		}
	}

	if _, ok := levels[gemInstance.Level]; !ok {
		// That failed, so just grab any level
		for l := range levels {
			gemInstance.Level = l
			break
		}
	}
}

func CalcBuildSkillInstanceStats(skillInstance *ActiveEffect, grantedEffect *GrantedEffect) map[string]float64 {
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

		qualityStats := allQualities[qualityID]
		for i, stat := range qualityStats.GetStats() {
			baseVal := float64(qualityStats.StatsValuesPermille[i])
			if baseVal > 0 {
				baseVal = baseVal / 1000
			}

			stats[stat.ID] = stats[stat.ID] + utils.ModF(baseVal*float64(skillInstance.Quality))
		}
	}

	levels := grantedEffect.Raw.GetCalculatedLevels()
	level := levels[skillInstance.Level]

	var availableEffectiveness *float64 = nil

	// TODO local actorLevel = skillInstance.actorLevel or level.levelRequirement
	actorLevel := float64(level.LevelRequirement)

	for index, stat := range grantedEffect.Raw.GetCalculatedStats() {
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
					baseEffectiveness := grantedEffect.Raw.GetGrantedEffectStatSet().BaseEffectiveness
					if baseEffectiveness == 0 {
						baseEffectiveness = 1
					}

					incrementalEffectiveness := grantedEffect.Raw.GetGrantedEffectStatSet().IncrementalEffectiveness

					availableEffectiveness = utils.Ptr((3.885209 + 0.360246*(actorLevel-1)) * (baseEffectiveness) * math.Pow(1+(incrementalEffectiveness), actorLevel-1))
				}
				statValue = math.Round(*availableEffectiveness * level.Values[index])
			} else if statInterpolation == 2 {
				// Linear interpolation; I'm actually just guessing how this works
				nextLevel := utils.Min(skillInstance.Level+1, len(levels)-1)
				nextReq := float64(levels[nextLevel].LevelRequirement)
				prevReq := float64(levels[nextLevel-1].LevelRequirement)
				nextStat := levels[nextLevel].Values[index]
				prevStat := levels[nextLevel-1].Values[index]
				statValue = math.Round(prevStat + (nextStat-prevStat)*(actorLevel-prevReq)/(nextReq-prevReq))
			}
		}

		stats[stat] = stats[stat] + statValue
	}

	for id, stat := range grantedEffect.Raw.GetCalculatedConstantStats() {
		stats[id] = stats[id] + stat
	}

	return stats
}
