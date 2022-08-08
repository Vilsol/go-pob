package raw

type PassiveTreeExpansionSkill struct {
	MasteryPassiveSkillsKey           *int `json:"Mastery_PassiveSkillsKey"`
	PassiveSkillsKey                  int  `json:"PassiveSkillsKey"`
	PassiveTreeExpansionJewelSizesKey int  `json:"PassiveTreeExpansionJewelSizesKey"`
	TagsKey                           int  `json:"TagsKey"`
	Key                               int  `json:"_key"`
}

var PassiveTreeExpansionSkills []*PassiveTreeExpansionSkill

func InitializePassiveTreeExpansionSkills(version string) error {
	return InitHelper(version, "PassiveTreeExpansionSkills", &PassiveTreeExpansionSkills)
}
