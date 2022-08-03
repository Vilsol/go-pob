package raw

type AlternatePassiveSkill struct {
	AchievementItemsKeys     []interface{} `json:"AchievementItemsKeys"`
	AlternateTreeVersionsKey int           `json:"AlternateTreeVersionsKey"`
	DDSIcon                  string        `json:"DDSIcon"`
	FlavourText              string        `json:"FlavourText"`
	ID                       string        `json:"Id"`
	Name                     string        `json:"Name"`
	PassiveType              []int         `json:"PassiveType"`
	RandomMax                int           `json:"RandomMax"`
	RandomMin                int           `json:"RandomMin"`
	SpawnWeight              int           `json:"SpawnWeight"`
	Stat1Max                 int           `json:"Stat1Max"`
	Stat1Min                 int           `json:"Stat1Min"`
	Stat2Max                 int           `json:"Stat2Max"`
	Stat2Min                 int           `json:"Stat2Min"`
	StatsKeys                []int         `json:"StatsKeys"`
	Key                      int           `json:"_key"`
}

var AlternatePassiveSkills []*AlternatePassiveSkill
var AlternatePassiveSkillsMap map[int]*AlternatePassiveSkill

func InitializeAlternatePassiveSkills(version string) error {
	if err := InitHelper(version, "AlternatePassiveSkills", &AlternatePassiveSkills); err != nil {
		return err
	}

	AlternatePassiveSkillsMap = make(map[int]*AlternatePassiveSkill)
	for _, i := range AlternatePassiveSkills {
		AlternatePassiveSkillsMap[i.Key] = i
	}

	return nil
}
