package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type MonsterVariety struct {
	raw2.MonsterVariety
}

var MonsterVarieties []*MonsterVariety

func InitializeMonsterVarieties(version string) error {
	return InitHelper(version, "MonsterVarieties", &MonsterVarieties, nil)
}
