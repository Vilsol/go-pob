package raw

type ItemExperiencePerLevel struct {
	BaseItemTypesKey int `json:"BaseItemTypesKey"`
	Experience       int `json:"Experience"`
	ItemCurrentLevel int `json:"ItemCurrentLevel"`
	Key              int `json:"_key"`
}

var ItemExperiencePerLevels []*ItemExperiencePerLevel

func InitializeItemExperiencePerLevels(version string) error {
	return InitHelper(version, "ItemExperiencePerLevel", &ItemExperiencePerLevels)
}
