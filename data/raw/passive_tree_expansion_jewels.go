package raw

type PassiveTreeExpansionJewel struct {
	Art                               string `json:"Art"`
	BaseItemTypesKey                  int    `json:"BaseItemTypesKey"`
	MaxNodes                          int    `json:"MaxNodes"`
	MinNodes                          int    `json:"MinNodes"`
	NotableIndices                    []int  `json:"NotableIndices"`
	PassiveTreeExpansionJewelSizesKey int    `json:"PassiveTreeExpansionJewelSizesKey"`
	SmallIndices                      []int  `json:"SmallIndices"`
	SocketIndices                     []int  `json:"SocketIndices"`
	SoundEffectsKey                   int    `json:"SoundEffectsKey"`
	TotalIndices                      int    `json:"TotalIndices"`
	Key                               int    `json:"_key"`
}

var PassiveTreeExpansionJewels []*PassiveTreeExpansionJewel
var PassiveTreeExpansionJewelsMap map[int]*PassiveTreeExpansionJewel

func InitializePassiveTreeExpansionJewels(version string) error {
	if err := InitHelper(version, "PassiveTreeExpansionJewels", &PassiveTreeExpansionJewels); err != nil {
		return err
	}

	PassiveTreeExpansionJewelsMap = make(map[int]*PassiveTreeExpansionJewel)
	for _, i := range PassiveTreeExpansionJewels {
		PassiveTreeExpansionJewelsMap[i.Key] = i
	}

	return nil
}
