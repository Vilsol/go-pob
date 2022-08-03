package raw

type ShieldType struct {
	BaseItemTypesKey int `json:"BaseItemTypesKey"`
	Block            int `json:"Block"`
	Key              int `json:"_key"`
}

var ShieldTypes []*ShieldType
var ShieldTypesMap map[int]*ShieldType

func InitializeShieldTypes(version string) error {
	if err := InitHelper(version, "ShieldTypes", &ShieldTypes); err != nil {
		return err
	}

	ShieldTypesMap = make(map[int]*ShieldType)
	for _, i := range ShieldTypes {
		ShieldTypesMap[i.Key] = i
	}

	return nil
}
