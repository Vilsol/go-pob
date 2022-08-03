package raw

type PassiveTreeExpansionSkill struct {
	MasteryPassiveSkillsKey           *int `json:"Mastery_PassiveSkillsKey"`
	PassiveSkillsKey                  int  `json:"PassiveSkillsKey"`
	PassiveTreeExpansionJewelSizesKey int  `json:"PassiveTreeExpansionJewelSizesKey"`
	TagsKey                           int  `json:"TagsKey"`
	Key                               int  `json:"_key"`
}

var PassiveTreeExpansionSkills []*PassiveTreeExpansionSkill
var PassiveTreeExpansionSkillsMap map[int]*PassiveTreeExpansionSkill

func InitializePassiveTreeExpansionSkills(version string) error {
	if err := InitHelper(version, "PassiveTreeExpansionSkills", &PassiveTreeExpansionSkills); err != nil {
		return err
	}

	PassiveTreeExpansionSkillsMap = make(map[int]*PassiveTreeExpansionSkill)
	for _, i := range PassiveTreeExpansionSkills {
		PassiveTreeExpansionSkillsMap[i.Key] = i
	}

	return nil
}
