package raw

type AlternatePassiveAddition struct {
	AlternateTreeVersionsKey int    `json:"AlternateTreeVersionsKey"`
	ID                       string `json:"Id"`
	PassiveType              []int  `json:"PassiveType"`
	SpawnWeight              int    `json:"SpawnWeight"`
	Stat1Max                 int    `json:"Stat1Max"`
	Stat1Min                 int    `json:"Stat1Min"`
	StatsKeys                []int  `json:"StatsKeys"`
	Key                      int    `json:"_key"`
}

var AlternatePassiveAdditions []*AlternatePassiveAddition
var AlternatePassiveAdditionsMap map[int]*AlternatePassiveAddition

func InitializeAlternatePassiveAdditions(version string) error {
	if err := InitHelper(version, "AlternatePassiveAdditions", &AlternatePassiveAdditions); err != nil {
		return err
	}

	AlternatePassiveAdditionsMap = make(map[int]*AlternatePassiveAddition)
	for _, i := range AlternatePassiveAdditions {
		AlternatePassiveAdditionsMap[i.Key] = i
	}

	return nil
}
