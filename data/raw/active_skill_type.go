package raw

type ActiveSkillType struct {
	FlagStat *int   `json:"FlagStat"`
	ID       string `json:"Id"`
	Key      int    `json:"_key"`
}

var ActiveSkillTypes []*ActiveSkillType
var ActiveSkillTypesMap map[int]*ActiveSkillType

func InitializeActiveSkillTypes(version string) error {
	if err := InitHelper(version, "ActiveSkillType", &ActiveSkillTypes); err != nil {
		return err
	}

	ActiveSkillTypesMap = make(map[int]*ActiveSkillType)
	for _, i := range ActiveSkillTypes {
		ActiveSkillTypesMap[i.Key] = i
	}

	return nil
}
