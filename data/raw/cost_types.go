package raw

type CostType struct {
	FormatText string `json:"FormatText"`
	ID         string `json:"Id"`
	StatsKey   int    `json:"StatsKey"`
	Key        int    `json:"_key"`
}

var CostTypes []*CostType

func InitializeCostTypes(version string) error {
	return InitHelper(version, "CostTypes", &CostTypes)
}
