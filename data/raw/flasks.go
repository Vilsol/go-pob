package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type Flask struct {
	raw2.Flask
}

var Flasks []*Flask

func InitializeFlasks(version string) error {
	return InitHelper(version, "Flasks", &Flasks, nil)
}
