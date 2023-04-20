package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type AlternatePassiveAddition struct {
	raw2.AlternatePassiveAddition
}

var AlternatePassiveAdditions []*AlternatePassiveAddition

func InitializeAlternatePassiveAdditions(version string) error {
	return InitHelper(version, "AlternatePassiveAdditions", &AlternatePassiveAdditions, nil)
}
