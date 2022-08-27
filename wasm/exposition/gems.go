package exposition

import (
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
	MaxLevel int
	ID       string
	GemType  GemType
	Base     GemPart
	Vaal     GemPart
	Support  bool
}

func (g SkillGem) CalculateStuff() {
	println("HERE")
}

var skillGemCache []SkillGem

func GetSkillGems() []SkillGem {
	if skillGemCache == nil {
		skillGemCache = make([]SkillGem, 0)
		for _, gem := range raw.SkillGems {
			baseType := gem.GetBaseItemType()
			if baseType.SiteVisibility < 1 {
				continue
			}

			description := gem.Description
			grantedEffect := gem.GetGrantedEffect()
			if description == "" {
				description = grantedEffect.GetActiveSkill().Description
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

			if gem.Str > gem.Dex && gem.Str > gem.Int {
				outGem.GemType = GemTypeStrength
			} else if gem.Dex > gem.Int && gem.Dex > gem.Str {
				outGem.GemType = GemTypeDexterity
			} else if gem.Int > gem.Dex && gem.Int > gem.Str {
				outGem.GemType = GemTypeIntelligence
			}

			if gem.IsVaalGem {
				nonVaalGem := gem.GetNonVaal()
				if nonVaalGem == nil && gem.SecondaryGrantedEffect != nil {
					nonVaalGem = gem.GetSecondaryGrantedEffect().GetSkillGem()
				}

				if nonVaalGem != nil {
					outGem.Vaal = GemPart{
						Name:        nonVaalGem.GetBaseItemType().Name,
						Description: nonVaalGem.GetGrantedEffect().GetActiveSkill().Description,
					}
				}
			}

			skillGemCache = append(skillGemCache, outGem)
		}
	}

	return skillGemCache
}
