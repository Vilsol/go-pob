package exposition

import (
	"slices"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
)

type GemPart struct {
	Name        string
	Description string
}

type GemType string

const (
	GemTypeStrength     = GemType("STR")
	GemTypeDexterity    = GemType("DEX")
	GemTypeIntelligence = GemType("INT")
	GemTypeNone         = GemType("NONE")
)

type SkillGem struct {
	MaxLevel         int
	ID               string
	GemType          GemType
	Base             GemPart
	Vaal             GemPart
	Support          bool
	Name             string
	Tags             []string
	LevelRequirement int
	GlobalEffects    []string
	Legacy           bool
	HideFromGemList  bool
	LegacyAwakened   bool
}

var skillGemCache []SkillGem

func GetSkillGems() []SkillGem {
	if skillGemCache == nil {
		skillGemCache = make([]SkillGem, 0)
		for _, gem := range poe.SkillGems {
			baseType := gem.GetBaseItemType()
			if baseType == nil {
				continue
			}

			pobGem, shipped := raw.Gems[baseType.ID]
			if !shipped {
				continue
			}

			grantedEffect := poe.GrantedEffectByID(pobGem.GrantedEffectID)
			if grantedEffect == nil {
				continue
			}

			description := gem.Description
			if description == "" {
				if activeSkill := grantedEffect.GetActiveSkill(); activeSkill != nil {
					description = activeSkill.Description
				}
			}

			outGem := SkillGem{
				MaxLevel: len(grantedEffect.Levels()),
				ID:       baseType.ID,
				GemType:  GemTypeNone,
				Support:  grantedEffect.IsSupport,
				Base: GemPart{
					Name:        baseType.Name,
					Description: description,
				},
			}
			outGem.Name = pobGem.Name
			pobSkill := raw.PoBSkills[pobGem.GrantedEffectID]
			outGem.Legacy = pobSkill.Legacy
			outGem.HideFromGemList = pobSkill.HideFromGemList
			outGem.LegacyAwakened = pobSkill.Legacy && pobSkill.PlusVersionOf != ""
			for tag, on := range pobGem.Tags {
				if on {
					outGem.Tags = append(outGem.Tags, tag)
				}
			}
			slices.Sort(outGem.Tags)
			//pob:block:src/Classes/GemSelectControl.lua:126
			outGem.LevelRequirement = 1
			if calc := raw.GetCalculatedGrantedEffect(grantedEffect); calc != nil {
				if first, ok := calc.GetCalculatedLevels()[1]; ok && first.LevelRequirement != 0 {
					outGem.LevelRequirement = first.LevelRequirement
				}
			}

			switch {
			case pobGem.ReqStr > pobGem.ReqDex && pobGem.ReqStr > pobGem.ReqInt:
				outGem.GemType = GemTypeStrength
			case pobGem.ReqDex > pobGem.ReqInt && pobGem.ReqDex > pobGem.ReqStr:
				outGem.GemType = GemTypeDexterity
			case pobGem.ReqInt > pobGem.ReqDex && pobGem.ReqInt > pobGem.ReqStr:
				outGem.GemType = GemTypeIntelligence
			}

			//pob:block:src/Classes/SkillsTab.lua:1048-1075
			if gem.IsVaalGem {
				outGem.GlobalEffects = []string{"", ""}
				for i, id := range []string{pobGem.GrantedEffectID, pobGem.SecondaryGrantedEffectID} {
					if effect := poe.GrantedEffectByID(id); id != "" && effect != nil && !effect.IsSupport {
						outGem.GlobalEffects[i] = grantedEffectName(effect)
					}
				}
			}
			if gem.IsVaalGem {
				nonVaalGem := gem.GetNonVaal()
				if nonVaalGem == nil && pobGem.SecondaryGrantedEffectID != "" {
					nonVaalGem = raw.SkillGemForGrantedEffectID(pobGem.SecondaryGrantedEffectID)
				}

				if nonVaalBase := baseItemTypeOf(nonVaalGem); nonVaalBase != nil {
					outGem.Vaal = GemPart{
						Name:        nonVaalBase.Name,
						Description: activeSkillDescriptionOf(nonVaalGem),
					}
				}
			}

			skillGemCache = append(skillGemCache, outGem)
		}
	}

	return skillGemCache
}

func baseItemTypeOf(gem *poe.SkillGem) *poe.BaseItemType {
	if gem == nil {
		return nil
	}
	return gem.GetBaseItemType()
}

func activeSkillDescriptionOf(gem *poe.SkillGem) string {
	for _, effect := range raw.GrantedEffectsForGem(gem) {
		if activeSkill := effect.GetActiveSkill(); activeSkill != nil {
			return activeSkill.Description
		}
	}
	return ""
}

//pob:block:src/Classes/SkillsTab.lua:1169-1204
func GemError(gemID, skillID, nameSpec string) string {
	if gemID != "" || skillID != "" || strings.TrimSpace(nameSpec) == "" {
		return ""
	}
	return calculator.FindSkillGemError(nameSpec)
}

func grantedEffectName(effect *poe.GrantedEffect) string {
	if skill, ok := raw.PoBSkills[effect.ID]; ok && skill.Name != "" {
		return skill.Name
	}
	if as := effect.GetActiveSkill(); as != nil && as.DisplayedName != "" {
		return as.DisplayedName
	}
	return effect.ID
}
