package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type ComponentAttributeRequirement struct {
	raw2.ComponentAttributeRequirement
}

var ComponentAttributeRequirements []*ComponentAttributeRequirement

func InitializeComponentAttributeRequirements(version string) error {
	return InitHelper(version, "ComponentAttributeRequirements", &ComponentAttributeRequirements, nil)
}
