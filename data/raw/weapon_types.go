package raw

type WeaponType struct {
	BaseItemTypesKey int `json:"BaseItemTypesKey"`
	Critical         int `json:"Critical"`
	DamageMax        int `json:"DamageMax"`
	DamageMin        int `json:"DamageMin"`
	Null6            int `json:"Null6"`
	RangeMax         int `json:"RangeMax"`
	Speed            int `json:"Speed"`
	Key              int `json:"_key"`
}

var WeaponTypes []*WeaponType
var WeaponTypesMap map[int]*WeaponType

func InitializeWeaponTypes(version string) error {
	if err := InitHelper(version, "WeaponTypes", &WeaponTypes); err != nil {
		return err
	}

	WeaponTypesMap = make(map[int]*WeaponType)
	for _, i := range WeaponTypes {
		WeaponTypesMap[i.Key] = i
	}

	return nil
}
