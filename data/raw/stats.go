package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type Stat struct {
	raw2.Stat
}

var Stats []*Stat

func InitializeStats(version string) error {
	return InitHelper(version, "Stats", &Stats, nil)
}
