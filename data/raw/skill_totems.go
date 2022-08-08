package raw

type SkillTotem struct {
	Key int `json:"_key"`
}

var SkillTotems []*SkillTotem

func InitializeSkillTotems(version string) error {
	return InitHelper(version, "SkillTotems", &SkillTotems)
}
