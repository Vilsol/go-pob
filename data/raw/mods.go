package raw

type Mod struct {
	AchievementItemsKey              []interface{} `json:"AchievementItemsKey"`
	ArchnemesisMinionMod             *int          `json:"ArchnemesisMinionMod"`
	BuffTemplate                     *int          `json:"BuffTemplate"`
	ChestModType                     []int         `json:"ChestModType"`
	CorrectGroup                     string        `json:"CorrectGroup"`
	CraftingItemClassRestrictions    []int         `json:"CraftingItemClassRestrictions"`
	Domain                           int           `json:"Domain"`
	FullAreaClearAchievementItemsKey []interface{} `json:"FullAreaClear_AchievementItemsKey"`
	GenerationType                   int           `json:"GenerationType"`
	GenerationWeightTagsKeys         []int         `json:"GenerationWeight_TagsKeys"`
	GenerationWeightValues           []int         `json:"GenerationWeight_Values"`
	GrantedEffectsPerLevelKeys       []int         `json:"GrantedEffectsPerLevelKeys"`
	Hash16                           int           `json:"HASH16"`
	Hash32                           int           `json:"HASH32"`
	HeistAddStatValue1               int           `json:"Heist_AddStatValue1"`
	HeistAddStatValue2               int           `json:"Heist_AddStatValue2"`
	HeistStatsKey0                   *int          `json:"Heist_StatsKey0"`
	HeistStatsKey1                   *int          `json:"Heist_StatsKey1"`
	HeistSubStatValue1               int           `json:"Heist_SubStatValue1"`
	HeistSubStatValue2               int           `json:"Heist_SubStatValue2"`
	ID                               string        `json:"Id"`
	ImplicitTagsKeys                 []int         `json:"ImplicitTagsKeys"`
	InfluenceTypes                   int           `json:"InfluenceTypes"`
	IsEssenceOnlyModifier            bool          `json:"IsEssenceOnlyModifier"`
	Level                            int           `json:"Level"`
	MaxLevel                         int           `json:"MaxLevel"`
	ModTypeKey                       int           `json:"ModTypeKey"`
	ModifyMapsAchievements           []int         `json:"ModifyMapsAchievements"`
	MonsterKillAchievements          []int         `json:"MonsterKillAchievements"`
	MonsterMetadata                  string        `json:"MonsterMetadata"`
	MonsterOnDeath                   string        `json:"MonsterOnDeath"`
	Name                             string        `json:"Name"`
	SpawnWeightTagsKeys              []int         `json:"SpawnWeight_TagsKeys"`
	SpawnWeightValues                []int         `json:"SpawnWeight_Values"`
	Stat1Max                         int           `json:"Stat1Max"`
	Stat1Min                         int           `json:"Stat1Min"`
	Stat2Max                         int           `json:"Stat2Max"`
	Stat2Min                         int           `json:"Stat2Min"`
	Stat3Max                         int           `json:"Stat3Max"`
	Stat3Min                         int           `json:"Stat3Min"`
	Stat4Max                         int           `json:"Stat4Max"`
	Stat4Min                         int           `json:"Stat4Min"`
	Stat5Max                         int           `json:"Stat5Max"`
	Stat5Min                         int           `json:"Stat5Min"`
	Stat6Max                         int           `json:"Stat6Max"`
	Stat6Min                         int           `json:"Stat6Min"`
	StatsKey1                        *int          `json:"StatsKey1"`
	StatsKey2                        *int          `json:"StatsKey2"`
	StatsKey3                        *int          `json:"StatsKey3"`
	StatsKey4                        *int          `json:"StatsKey4"`
	StatsKey5                        *int          `json:"StatsKey5"`
	StatsKey6                        *int          `json:"StatsKey6"`
	TagsKeys                         []int         `json:"TagsKeys"`
	Key                              int           `json:"_key"`
}

var Mods []*Mod

func InitializeMods(version string) error {
	return InitHelper(version, "Mods", &Mods)
}
