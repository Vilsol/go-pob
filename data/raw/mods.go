package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type Mod struct {
	raw2.Mod
}

var Mods []*Mod

func InitializeMods(version string) error {
	return InitHelper(version, "Mods", &Mods, nil)
}
