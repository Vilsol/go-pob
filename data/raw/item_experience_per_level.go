package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type ItemExperiencePerLevel struct {
	raw2.ItemExperiencePerLevel
}

var ItemExperiencePerLevels []*ItemExperiencePerLevel

func InitializeItemExperiencePerLevels(version string) error {
	return InitHelper(version, "ItemExperiencePerLevel", &ItemExperiencePerLevels, nil)
}
