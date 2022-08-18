package raw

type SkillGem struct {
	BaseItemType           int    `json:"BaseItemTypesKey"`
	GrantedEffect          int    `json:"GrantedEffectsKey"`
	Str                    int    `json:"Str"`
	Dex                    int    `json:"Dex"`
	Int                    int    `json:"Int"`
	Tags                   []int  `json:"GemTagsKeys"`
	VaalGem                *int   `json:"VaalVariant_BaseItemTypesKey"`
	IsVaalGem              bool   `json:"IsVaalVariant"`
	Description            string `json:"Description"`
	HungryLoopMod          *int   `json:"Consumed_ModsKey"`
	SecondaryGrantedEffect *int   `json:"GrantedEffectsKey2"`
	GlobalGemLevelStat     *int   `json:"MinionGlobalSkillLevelStat"`
	SecondarySupportName   string `json:"SupportSkillName"`
	AwakenedVariant        *int   `json:"AwakenedVariant"`
	RegularVariant         *int   `json:"RegularVariant"`
	Key                    int    `json:"_key"`
}

var SkillGems []*SkillGem

var skillGemsByBaseItemTypeMap map[int]*SkillGem
var skillGemVaalBase map[int]*SkillGem
var skillGemsByGrantedEffect map[int]*SkillGem

func InitializeSkillGems(version string) error {
	if err := InitHelper(version, "SkillGems", &SkillGems); err != nil {
		return err
	}

	skillGemsByBaseItemTypeMap = make(map[int]*SkillGem)
	skillGemVaalBase = make(map[int]*SkillGem)
	skillGemsByGrantedEffect = make(map[int]*SkillGem)
	for _, gem := range SkillGems {
		skillGemsByBaseItemTypeMap[gem.BaseItemType] = gem
		skillGemsByGrantedEffect[gem.GrantedEffect] = gem
		if gem.VaalGem != nil {
			skillGemVaalBase[*gem.VaalGem] = gem
		}
	}

	return nil
}

func (s *SkillGem) GetGrantedEffect() *GrantedEffect {
	return GrantedEffects[s.GrantedEffect]
}

func (s *SkillGem) GetSecondaryGrantedEffect() *GrantedEffect {
	if s.SecondaryGrantedEffect == nil {
		return nil
	}

	return GrantedEffects[*s.SecondaryGrantedEffect]
}

func (s *SkillGem) GetGrantedEffects() []*GrantedEffect {
	out := make([]*GrantedEffect, 1)
	out[0] = s.GetGrantedEffect()

	secondary := s.GetSecondaryGrantedEffect()
	if secondary != nil {
		out = append(out, secondary)
	}

	return out
}

func (s *SkillGem) GetTags() map[TagName]*Tag {
	out := make(map[TagName]*Tag, len(s.Tags))
	for _, tag := range s.Tags {
		t := Tags[tag]
		out[t.Name] = t
	}
	return out
}

func (s *SkillGem) DefaultLevel() int {
	levels := s.GetGrantedEffect().Levels()
	if len(levels) > 20 {
		return len(levels) - 21
	}
	// TODO Awakened gem default level?
	return 1
}

func (s *SkillGem) GetBaseItemType() *BaseItemType {
	return BaseItemTypes[s.BaseItemType]
}

func (s *SkillGem) GetNonVaal() *SkillGem {
	return skillGemVaalBase[s.BaseItemType]
}
