package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type MonsterMapDifficulty struct {
	raw2.MonsterMapDifficulty
}

var MonsterMapDifficulties []*MonsterMapDifficulty

func InitializeMonsterMapDifficulties(version string) error {
	return InitHelper(version, "MonsterMapDifficulty", &MonsterMapDifficulties, nil)
}
