package raw

type PassiveTreeExpansionSpecialSkill struct {
	PassiveSkillsKey int `json:"PassiveSkillsKey"`
	StatsKey         int `json:"StatsKey"`
	Key              int `json:"_key"`
}

var PassiveTreeExpansionSpecialSkills []*PassiveTreeExpansionSpecialSkill
var PassiveTreeExpansionSpecialSkillsMap map[int]*PassiveTreeExpansionSpecialSkill

func InitializePassiveTreeExpansionSpecialSkills(version string) error {
	if err := InitHelper(version, "PassiveTreeExpansionSpecialSkills", &PassiveTreeExpansionSpecialSkills); err != nil {
		return err
	}

	PassiveTreeExpansionSpecialSkillsMap = make(map[int]*PassiveTreeExpansionSpecialSkill)
	for _, i := range PassiveTreeExpansionSpecialSkills {
		PassiveTreeExpansionSpecialSkillsMap[i.Key] = i
	}

	return nil
}
