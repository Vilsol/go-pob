package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type MonsterMapBossDifficulty struct {
	raw2.MonsterMapBossDifficulty
}

var MonsterMapBossDifficulties []*MonsterMapBossDifficulty

func InitializeMonsterMapBossDifficulties(version string) error {
	return InitHelper(version, "MonsterMapBossDifficulty", &MonsterMapBossDifficulties, nil)
}
