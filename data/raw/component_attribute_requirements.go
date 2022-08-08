package raw

type ComponentAttributeRequirement struct {
	BaseItemTypesKey string `json:"BaseItemTypesKey"`
	ReqDex           int    `json:"ReqDex"`
	ReqInt           int    `json:"ReqInt"`
	ReqStr           int    `json:"ReqStr"`
	Key              int    `json:"_key"`
}

var ComponentAttributeRequirements []*ComponentAttributeRequirement

func InitializeComponentAttributeRequirements(version string) error {
	return InitHelper(version, "ComponentAttributeRequirements", &ComponentAttributeRequirements)
}
