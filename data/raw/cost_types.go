package raw

type CostType struct {
	FormatText string `json:"FormatText"`
	ID         string `json:"Id"`
	StatsKey   int    `json:"StatsKey"`
	Key        int    `json:"_key"`
}

var CostTypes []*CostType
var CostTypesMap map[int]*CostType

func InitializeCostTypes(version string) error {
	if err := InitHelper(version, "CostTypes", &CostTypes); err != nil {
		return err
	}

	CostTypesMap = make(map[int]*CostType)
	for _, i := range CostTypes {
		CostTypesMap[i.Key] = i
	}

	return nil
}
