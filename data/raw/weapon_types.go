package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type WeaponType struct {
	raw2.WeaponType
}

var WeaponTypes []*WeaponType

func InitializeWeaponTypes(version string) error {
	return InitHelper(version, "WeaponTypes", &WeaponTypes, nil)
}
