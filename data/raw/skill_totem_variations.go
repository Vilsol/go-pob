package raw

type SkillTotemVariation struct {
	MonsterVarietiesKey int `json:"MonsterVarietiesKey"`
	SkillTotemsKey      int `json:"SkillTotemsKey"`
	TotemSkinID         int `json:"TotemSkinId"`
	Key                 int `json:"_key"`
}

var SkillTotemVariations []*SkillTotemVariation

func InitializeSkillTotemVariations(version string) error {
	return InitHelper(version, "SkillTotemVariations", &SkillTotemVariations)
}
