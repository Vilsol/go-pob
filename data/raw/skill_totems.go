package raw

type SkillTotem struct {
	Key int `json:"_key"`
}

var SkillTotems []*SkillTotem
var SkillTotemsMap map[int]*SkillTotem

func InitializeSkillTotems(version string) error {
	if err := InitHelper(version, "SkillTotems", &SkillTotems); err != nil {
		return err
	}

	SkillTotemsMap = make(map[int]*SkillTotem)
	for _, i := range SkillTotems {
		SkillTotemsMap[i.Key] = i
	}

	return nil
}
