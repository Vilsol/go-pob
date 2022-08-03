package raw

type SkillTotemVariation struct {
	MonsterVarietiesKey int `json:"MonsterVarietiesKey"`
	SkillTotemsKey      int `json:"SkillTotemsKey"`
	TotemSkinID         int `json:"TotemSkinId"`
	Key                 int `json:"_key"`
}

var SkillTotemVariations []*SkillTotemVariation
var SkillTotemVariationsMap map[int]*SkillTotemVariation

func InitializeSkillTotemVariations(version string) error {
	if err := InitHelper(version, "SkillTotemVariations", &SkillTotemVariations); err != nil {
		return err
	}

	SkillTotemVariationsMap = make(map[int]*SkillTotemVariation)
	for _, i := range SkillTotemVariations {
		SkillTotemVariationsMap[i.Key] = i
	}

	return nil
}
