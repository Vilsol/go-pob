package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type Essence struct {
	raw2.Essence
}

var Essences []*Essence

func InitializeEssences(version string) error {
	return InitHelper(version, "Essences", &Essences, nil)
}
