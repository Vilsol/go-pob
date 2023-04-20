package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type ShieldType struct {
	raw2.ShieldType
}

var ShieldTypes []*ShieldType

func InitializeShieldTypes(version string) error {
	return InitHelper(version, "ShieldTypes", &ShieldTypes, nil)
}
