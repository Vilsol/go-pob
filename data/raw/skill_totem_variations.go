package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type SkillTotemVariation struct {
	raw2.SkillTotemVariation
}

var SkillTotemVariations []*SkillTotemVariation

func InitializeSkillTotemVariations(version string) error {
	return InitHelper(version, "SkillTotemVariations", &SkillTotemVariations, nil)
}
