package raw

type BaseItemType struct {
	DropLevel                     int           `json:"DropLevel"`
	EquipAchievementItemsKey      *int          `json:"Equip_AchievementItemsKey"`
	FlavourTextKey                *int          `json:"FlavourTextKey"`
	FragmentBaseItemTypesKey      *int          `json:"FragmentBaseItemTypesKey"`
	Hash                          int           `json:"HASH"`
	Height                        int           `json:"Height"`
	ID                            string        `json:"Id"`
	IdentifyMagicAchievementItems []interface{} `json:"IdentifyMagic_AchievementItems"`
	IdentifyAchievementItems      []interface{} `json:"Identify_AchievementItems"`
	ImplicitModsKeys              []int         `json:"Implicit_ModsKeys"`
	Inflection                    string        `json:"Inflection"`
	InheritsFrom                  string        `json:"InheritsFrom"`
	IsCorrupted                   bool          `json:"IsCorrupted"`
	ItemClassesKey                int           `json:"ItemClassesKey"`
	ItemVisualIdentity            int           `json:"ItemVisualIdentity"`
	ModDomain                     int           `json:"ModDomain"`
	Name                          string        `json:"Name"`
	SiteVisibility                int           `json:"SiteVisibility"`
	SizeOnGround                  int           `json:"SizeOnGround"`
	SoundEffect                   *int          `json:"SoundEffect"`
	TagsKeys                      []int         `json:"TagsKeys"`
	VendorRecipeAchievementItems  []int         `json:"VendorRecipe_AchievementItems"`
	Width                         int           `json:"Width"`
	Key                           int           `json:"_key"`
}

var BaseItemTypes []*BaseItemType
var BaseItemTypesMap map[int]*BaseItemType

var BaseItemTypeByIDMap map[string]*BaseItemType

func InitializeBaseItemTypes(version string) error {
	if err := InitHelper(version, "BaseItemTypes", &BaseItemTypes); err != nil {
		return err
	}

	BaseItemTypesMap = make(map[int]*BaseItemType)
	for _, i := range BaseItemTypes {
		BaseItemTypesMap[i.Key] = i
	}

	BaseItemTypeByIDMap = make(map[string]*BaseItemType)
	for _, i := range BaseItemTypes {
		BaseItemTypeByIDMap[i.ID] = i
	}

	return nil
}

func (b *BaseItemType) SkillGem() *SkillGem {
	return skillGemsByBaseItemTypeMap[b.Key]
}
