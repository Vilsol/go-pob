package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type ComponentCharge struct {
	raw2.ComponentCharge
}

var ComponentCharges []*ComponentCharge

func InitializeComponentCharges(version string) error {
	return InitHelper(version, "ComponentCharges", &ComponentCharges, nil)
}
