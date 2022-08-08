package raw

type ShieldType struct {
	BaseItemTypesKey int `json:"BaseItemTypesKey"`
	Block            int `json:"Block"`
	Key              int `json:"_key"`
}

var ShieldTypes []*ShieldType

func InitializeShieldTypes(version string) error {
	return InitHelper(version, "ShieldTypes", &ShieldTypes)
}
