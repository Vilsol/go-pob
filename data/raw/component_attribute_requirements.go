package raw

type ComponentAttributeRequirement struct {
	BaseItemTypesKey string `json:"BaseItemTypesKey"`
	ReqDex           int    `json:"ReqDex"`
	ReqInt           int    `json:"ReqInt"`
	ReqStr           int    `json:"ReqStr"`
	Key              int    `json:"_key"`
}

var ComponentAttributeRequirements []*ComponentAttributeRequirement
var ComponentAttributeRequirementsMap map[int]*ComponentAttributeRequirement

func InitializeComponentAttributeRequirements(version string) error {
	if err := InitHelper(version, "ComponentAttributeRequirements", &ComponentAttributeRequirements); err != nil {
		return err
	}

	ComponentAttributeRequirementsMap = make(map[int]*ComponentAttributeRequirement)
	for _, i := range ComponentAttributeRequirements {
		ComponentAttributeRequirementsMap[i.Key] = i
	}

	return nil
}
