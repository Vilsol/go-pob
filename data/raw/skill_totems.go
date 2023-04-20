package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type SkillTotem struct {
	raw2.SkillTotem
}

var SkillTotems []*SkillTotem

func InitializeSkillTotems(version string) error {
	return InitHelper(version, "SkillTotems", &SkillTotems, nil)
}
