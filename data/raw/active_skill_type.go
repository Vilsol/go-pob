package raw

type ActiveSkillType struct {
	FlagStat *int   `json:"FlagStat"`
	ID       string `json:"Id"`
	Key      int    `json:"_key"`
}

var ActiveSkillTypes []*ActiveSkillType

func InitializeActiveSkillTypes(version string) error {
	return InitHelper(version, "ActiveSkillType", &ActiveSkillTypes)
}
