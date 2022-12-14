package raw

type Essence struct {
	AmuletModsKey                    *int  `json:"Amulet_ModsKey"`
	BaseItemTypesKey                 int   `json:"BaseItemTypesKey"`
	BeltModsKey                      *int  `json:"Belt_ModsKey"`
	BodyArmourModsKey                *int  `json:"BodyArmour_ModsKey"`
	BootsModsKey                     *int  `json:"Boots_ModsKey"`
	BowModsKey                       *int  `json:"Bow_ModsKey"`
	ClawModsKey                      *int  `json:"Claw_ModsKey"`
	DaggerModsKey                    *int  `json:"Dagger_ModsKey"`
	DisplayAmuletModsKey             *int  `json:"Display_Amulet_ModsKey"`
	DisplayArmourModsKey             *int  `json:"Display_Armour_ModsKey"`
	DisplayBeltModsKey               *int  `json:"Display_Belt_ModsKey"`
	DisplayBodyArmourModsKey         *int  `json:"Display_BodyArmour_ModsKey"`
	DisplayBootsModsKey              *int  `json:"Display_Boots_ModsKey"`
	DisplayBowModsKey                *int  `json:"Display_Bow_ModsKey"`
	DisplayGlovesModsKey             *int  `json:"Display_Gloves_ModsKey"`
	DisplayHelmetModsKey             *int  `json:"Display_Helmet_ModsKey"`
	DisplayItemsModsKey              *int  `json:"Display_Items_ModsKey"`
	DisplayJewelleryModsKey          *int  `json:"Display_Jewellery_ModsKey"`
	DisplayMeleeWeaponModsKey        *int  `json:"Display_MeleeWeapon_ModsKey"`
	DisplayMonsterModsKey            int   `json:"Display_Monster_ModsKey"`
	DisplayOneHandWeaponModsKey      *int  `json:"Display_OneHandWeapon_ModsKey"`
	DisplayQuiverModsKey             *int  `json:"Display_Quiver_ModsKey"`
	DisplayRangedWeaponModsKey       *int  `json:"Display_RangedWeapon_ModsKey"`
	DisplayRingModsKey               *int  `json:"Display_Ring_ModsKey"`
	DisplayShieldModsKey             *int  `json:"Display_Shield_ModsKey"`
	DisplayTwoHandMeleeWeaponModsKey *int  `json:"Display_TwoHandMeleeWeapon_ModsKey"`
	DisplayTwoHandWeaponModsKey      *int  `json:"Display_TwoHandWeapon_ModsKey"`
	DisplayWandModsKey               *int  `json:"Display_Wand_ModsKey"`
	DisplayWeaponModsKey             *int  `json:"Display_Weapon_ModsKey"`
	DropLevelMaximum                 int   `json:"DropLevelMaximum"`
	DropLevelMinimum                 int   `json:"DropLevelMinimum"`
	EssenceTypeKey                   int   `json:"EssenceTypeKey"`
	GlovesModsKey                    *int  `json:"Gloves_ModsKey"`
	HelmetModsKey                    *int  `json:"Helmet_ModsKey"`
	IsScreamingEssence               bool  `json:"IsScreamingEssence"`
	ItemLevelRestriction             int   `json:"ItemLevelRestriction"`
	Level                            int   `json:"Level"`
	MonsterModsKeys                  []int `json:"Monster_ModsKeys"`
	OneHandAxeModsKey                *int  `json:"OneHandAxe_ModsKey"`
	OneHandMaceModsKey               *int  `json:"OneHandMace_ModsKey"`
	OneHandSwordModsKey              *int  `json:"OneHandSword_ModsKey"`
	OneHandThrustingSwordModsKey     *int  `json:"OneHandThrustingSword_ModsKey"`
	RingModsKey                      *int  `json:"Ring_ModsKey"`
	SceptreModsKey                   *int  `json:"Sceptre_ModsKey"`
	ShieldModsKey                    *int  `json:"Shield_ModsKey"`
	StaffModsKey                     *int  `json:"Staff_ModsKey"`
	TwoHandAxeModsKey                *int  `json:"TwoHandAxe_ModsKey"`
	TwoHandMaceModsKey               *int  `json:"TwoHandMace_ModsKey"`
	TwoHandSwordModsKey              *int  `json:"TwoHandSword_ModsKey"`
	WandModsKey                      *int  `json:"Wand_ModsKey"`
	Key                              int   `json:"_key"`
}

var Essences []*Essence

func InitializeEssences(version string) error {
	return InitHelper(version, "Essences", &Essences)
}
