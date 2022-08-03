package raw

type ComponentCharge struct {
	BaseItemTypesKey string `json:"BaseItemTypesKey"`
	MaxCharges       int    `json:"MaxCharges"`
	MaxCharges2      int    `json:"MaxCharges2"`
	PerCharge        int    `json:"PerCharge"`
	PerCharge2       int    `json:"PerCharge2"`
	Key              int    `json:"_key"`
}

var ComponentCharges []*ComponentCharge
var ComponentChargesMap map[int]*ComponentCharge

func InitializeComponentCharges(version string) error {
	if err := InitHelper(version, "ComponentCharges", &ComponentCharges); err != nil {
		return err
	}

	ComponentChargesMap = make(map[int]*ComponentCharge)
	for _, i := range ComponentCharges {
		ComponentChargesMap[i.Key] = i
	}

	return nil
}
