package raw

type ItemExperiencePerLevel struct {
	BaseItemTypesKey int `json:"BaseItemTypesKey"`
	Experience       int `json:"Experience"`
	ItemCurrentLevel int `json:"ItemCurrentLevel"`
	Key              int `json:"_key"`
}

var ItemExperiencePerLevels []*ItemExperiencePerLevel
var ItemExperiencePerLevelsMap map[int]*ItemExperiencePerLevel

func InitializeItemExperiencePerLevels(version string) error {
	if err := InitHelper(version, "ItemExperiencePerLevel", &ItemExperiencePerLevels); err != nil {
		return err
	}

	ItemExperiencePerLevelsMap = make(map[int]*ItemExperiencePerLevel)
	for _, i := range ItemExperiencePerLevels {
		ItemExperiencePerLevelsMap[i.Key] = i
	}

	return nil
}
