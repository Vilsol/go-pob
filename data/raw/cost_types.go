package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type CostType struct {
	raw2.CostType
}

var CostTypes []*CostType

func InitializeCostTypes(version string) error {
	return InitHelper(version, "CostTypes", &CostTypes, nil)
}
