package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type DefaultMonsterStat struct {
	raw2.DefaultMonsterStat
}

var DefaultMonsterStats []*DefaultMonsterStat

func InitializeDefaultMonsterStats(version string) error {
	return InitHelper(version, "DefaultMonsterStats", &DefaultMonsterStats, nil)
}
