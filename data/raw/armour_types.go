package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type ArmourType struct {
	raw2.ArmourType
}

var ArmourTypes []*ArmourType

func InitializeArmourTypes(version string) error {
	return InitHelper(version, "ArmourTypes", &ArmourTypes, nil)
}
