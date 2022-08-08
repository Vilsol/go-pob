package raw

type PassiveTreeExpansionSpecialSkill struct {
	PassiveSkillsKey int `json:"PassiveSkillsKey"`
	StatsKey         int `json:"StatsKey"`
	Key              int `json:"_key"`
}

var PassiveTreeExpansionSpecialSkills []*PassiveTreeExpansionSpecialSkill

func InitializePassiveTreeExpansionSpecialSkills(version string) error {
	return InitHelper(version, "PassiveTreeExpansionSpecialSkills", &PassiveTreeExpansionSpecialSkills)
}
